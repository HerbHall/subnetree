package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDemoAuthMiddleware_APIPathInjectsClaims(t *testing.T) {
	var captured *Claims

	backend := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = UserFromContext(r.Context())
	})

	handler := DemoAuthMiddleware()(backend)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recon/devices", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured == nil {
		t.Fatal("expected demo claims in context, got nil")
	}
	if captured.UserID != "demo-user" {
		t.Errorf("UserID = %q, want %q", captured.UserID, "demo-user")
	}
	if captured.Username != "demo" {
		t.Errorf("Username = %q, want %q", captured.Username, "demo")
	}
	if captured.Role != "viewer" {
		t.Errorf("Role = %q, want %q", captured.Role, "viewer")
	}
}

func TestDemoAuthMiddleware_NonAPIPathPassesThrough(t *testing.T) {
	var captured *Claims

	backend := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = UserFromContext(r.Context())
	})

	handler := DemoAuthMiddleware()(backend)

	req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if captured != nil {
		t.Errorf("expected nil claims for non-API path, got %+v", captured)
	}
}

func TestDemoAuthMiddleware_ClaimsAccessibleViaUserFromContext(t *testing.T) {
	var extractedClaims *Claims

	backend := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		extractedClaims = UserFromContext(r.Context())
	})

	handler := DemoAuthMiddleware()(backend)

	// Test multiple API paths.
	paths := []string{
		"/api/v1/recon/devices",
		"/api/v1/pulse/alerts",
		"/api/v1/health",
	}

	for _, path := range paths {
		extractedClaims = nil
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if extractedClaims == nil {
			t.Errorf("path %s: expected claims, got nil", path)
			continue
		}
		if extractedClaims.UserID != "demo-user" {
			t.Errorf("path %s: UserID = %q, want demo-user", path, extractedClaims.UserID)
		}
	}
}
