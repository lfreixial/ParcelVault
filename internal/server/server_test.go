package server

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestShutdown(t *testing.T) {
	for _, force := range []bool{false, true} {
		name := "drain"
		if force {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			ctx, stop := context.WithCancel(context.Background())
			defer stop()
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
			defer close(release)
			complete := make(chan struct{})
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(started)
				defer close(finished)
				select {
				case <-r.Context().Done():
				case <-release:
				case <-complete:
					_, _ = io.WriteString(w, "finished")
				}
			})
			result := make(chan error, 1)
			timeout := 3 * time.Second
			if force {
				timeout = 40 * time.Millisecond
			}
			go func() { result <- Run(ctx, ln, handler, timeout, slog.New(slog.DiscardHandler)) }()
			response := make(chan string, 1)
			go func() {
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Get("http://" + ln.Addr().String())
				if err != nil {
					response <- ""
					return
				}
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				response <- string(b)
			}()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("handler did not start")
			}
			stop()
			if !force {
				// Wait until Shutdown closes the listener while the request is still active.
				until := time.Now().Add(2 * time.Second)
				for {
					conn, err := net.DialTimeout("tcp", ln.Addr().String(), 100*time.Millisecond)
					if err != nil {
						break
					}
					conn.Close()
					if time.Now().After(until) {
						t.Fatal("listener still accepting requests")
					}
					time.Sleep(time.Millisecond)
				}
				close(complete)
			}
			select {
			case err := <-result:
				if (err != nil) != force {
					t.Fatalf("shutdown returned %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("server failed to stop")
			}
			select {
			case <-finished:
			case <-time.After(time.Second):
				t.Fatal("request context still active")
			}
			if body := <-response; !force && body != "finished" {
				t.Fatalf("request interrupted: %q", body)
			}
		})
	}
}

func TestHTTPDiagnosticRedaction(t *testing.T) {
	var output bytes.Buffer
	writer := httpErrors{slog.New(slog.NewJSONHandler(&output, nil))}
	_, _ = writer.Write([]byte("private-marker"))
	if strings.Contains(output.String(), "private-marker") || !strings.Contains(output.String(), "HTTP connection error") {
		t.Fatal("unsafe or missing diagnostic")
	}
}
