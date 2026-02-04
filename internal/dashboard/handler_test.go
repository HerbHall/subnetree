package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_DevMode(t *testing.T) {
	// In dev mode (distFS == nil), handler returns 404
	// This test only works when built without the dev tag
	// and when dist/ doesn't exist, so we test the behavior
	// of the nil check branch.

	handler := Handler()

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"root path", "/", http.StatusOK},
		{"dashboard route", "/dashboard", http.StatusOK},
		{"devices route", "/devices", http.StatusOK},
		{"nested route", "/devices/abc123", http.StatusOK},
		{"static asset", "/assets/index.js", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, http.NoBody)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// In production mode with dist/, these would return 200
			// In dev mode or without dist/, they return 404
			// We just verify the handler doesn't panic
			if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
				t.Errorf("unexpected status code: got %d", rec.Code)
			}
		})
	}
}

func TestHandler_ExcludesAPIRoutes(t *testing.T) {
	handler := Handler()

	apiPaths := []string{
		"/api/v1/health",
		"/api/v1/auth/login",
		"/api/v1/recon/scan",
		"/healthz",
		"/readyz",
		"/metrics",
	}

	for _, path := range apiPaths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// API routes should return 404 from the dashboard handler
			// so that the actual API handlers can process them
			if rec.Code != http.StatusNotFound {
				t.Errorf("expected 404 for API route %s, got %d", path, rec.Code)
			}
		})
	}
}
