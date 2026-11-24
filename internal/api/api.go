// Package api serves the unauthenticated operational endpoints.
package api

import (
	"encoding/json"
	"net/http"

	"github.com/lfreixial/ParcelVault/internal/version"
)

func New(ready func() bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", readOnly(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	mux.HandleFunc("/readyz", readOnly(func(w http.ResponseWriter, _ *http.Request) {
		if !ready() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}))
	mux.HandleFunc("/api/v1/version", readOnly(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": version.Version})
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		problem(w, http.StatusNotFound)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}

func readOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			problem(w, http.StatusMethodNotAllowed)
			return
		}
		if r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
			problem(w, http.StatusRequestEntityTooLarge)
			return
		}
		next(w, r)
	}
}

func problem(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/problem+json")
	writeJSON(w, status, struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
	}{http.StatusText(status), status})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
