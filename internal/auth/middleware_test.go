package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthMiddleware_SkipsNonAPIPath(t *testing.T) {
	ts := NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	mw := AuthMiddleware(ts)

	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should have been called for non-API path")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAuthMiddleware_SkipsPublicPaths(t *testing.T) {
	ts := NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	mw := AuthMiddleware(ts)

	for _, path := range []string{
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout",
		"/api/v1/auth/setup",
	} {
		t.Run(path, func(t *testing.T) {
			called := false
			handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))

			req := httptest.NewRequest("POST", path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if !called {
				t.Errorf("handler should have been called for public path %s", path)
			}
		})
	}
}

func TestAuthMiddleware_RejectsNoHeader(t *testing.T) {
	ts := NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	mw := AuthMiddleware(ts)

	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/api/v1/plugins", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should NOT have been called without auth header")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_RejectsBadToken(t *testing.T) {
	ts := NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	mw := AuthMiddleware(ts)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/api/v1/plugins", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_AcceptsValidToken(t *testing.T) {
	ts := NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	mw := AuthMiddleware(ts)

	user := &User{ID: "user-1", Username: "alice", Role: RoleAdmin}
	token, err := ts.IssueAccessToken(user)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	var gotClaims *Claims
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/plugins", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if gotClaims == nil {
		t.Fatal("expected claims in context")
	}
	if gotClaims.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", gotClaims.UserID)
	}
	if gotClaims.Username != "alice" {
		t.Errorf("Username = %q, want alice", gotClaims.Username)
	}
}

func TestAuthMiddleware_RejectsNonBearerScheme(t *testing.T) {
	ts := NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	mw := AuthMiddleware(ts)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/api/v1/plugins", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestUserFromContext_Nil(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	claims := UserFromContext(req.Context())
	if claims != nil {
		t.Error("expected nil claims for empty context")
	}
}
