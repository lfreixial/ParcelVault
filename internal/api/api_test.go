package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	for _, tc := range []struct {
		method, path, body string
		ready              bool
		code               int
		want               string
	}{
		{"GET", "/healthz", "", false, 200, `{"status":"ok"}`},
		{"GET", "/readyz", "", true, 200, `{"status":"ready"}`},
		{"GET", "/readyz", "", false, 503, `{"status":"unavailable"}`},
		{"GET", "/api/v1/version?private-marker", "", true, 200, `{"version":"0.0.2"}`},
		{"GET", "/private-marker", "", true, 404, `"status":404`},
		{"POST", "/readyz", "", true, 405, `"status":405`},
		{"GET", "/readyz", "private-marker", true, 413, `"status":413`},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("Authorization", "private-marker")
		w := httptest.NewRecorder()
		New(func(context.Context) bool { return tc.ready }).ServeHTTP(w, req)
		if w.Code != tc.code || !strings.Contains(w.Body.String(), tc.want) || strings.Contains(w.Body.String(), "private-marker") {
			t.Errorf("%s %s: %d %s", tc.method, tc.path, w.Code, w.Body)
		}
		contentType := "application/json"
		if tc.code >= 400 && tc.code != 503 {
			contentType = "application/problem+json"
		}
		if w.Header().Get("Content-Type") != contentType || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("unexpected headers: %v", w.Header())
		}
		if tc.code == http.StatusMethodNotAllowed && w.Header().Get("Allow") != "GET, HEAD" {
			t.Error("missing Allow")
		}
	}
}
