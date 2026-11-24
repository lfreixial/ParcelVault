// Package server owns the listener and the lifetime of HTTP requests.
package server

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Run serves an already bound listener and drains in-flight requests on cancellation.
func Run(ctx context.Context, ln net.Listener, handler http.Handler, timeout time.Duration, logger *slog.Logger) error {
	requests, cancelRequests := context.WithCancel(context.Background())
	defer cancelRequests()
	srv := &http.Server{
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return requests },
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 16 << 10,
		ErrorLog:       log.New(httpErrors{logger}, "", 0),
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	logger.Info("server listening", "address", ln.Addr().String())
	select {
	case <-ctx.Done():
	case <-done:
		return errors.New("HTTP server stopped unexpectedly")
	}

	logger.Info("shutdown started")
	drain, cancelDrain := context.WithTimeout(context.Background(), timeout)
	defer cancelDrain()
	if err := srv.Shutdown(drain); err != nil {
		cancelRequests()
		_ = srv.Close()
		return errors.New("HTTP shutdown deadline exceeded")
	}
	if err := <-done; !errors.Is(err, http.ErrServerClosed) {
		return errors.New("HTTP server stopped unexpectedly")
	}
	logger.Info("shutdown complete")
	return nil
}

// net/http diagnostics can contain peer addresses or panic values from handlers.
type httpErrors struct{ logger *slog.Logger }

func (h httpErrors) Write(p []byte) (int, error) {
	h.logger.Warn("HTTP connection error")
	return len(p), nil
}
