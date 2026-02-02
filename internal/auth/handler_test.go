package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HerbHall/netvantage/internal/store"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

// setupHandlerEnv creates an in-memory database with auth tables,
// returns a Handler with routes registered on a fresh mux.
func setupHandlerEnv(t *testing.T) (*Handler, *http.ServeMux) {
	t.Helper()

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	userStore, err := NewUserStore(ctx, db)
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}

	tokens := NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	svc := NewService(userStore, tokens, testLogger())
	handler := NewHandler(svc, testLogger())

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return handler, mux
}

func doRequest(mux *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func doAuthRequest(mux *http.ServeMux, method, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		// Inject claims directly into context to simulate middleware.
		claims := &Claims{UserID: "test", Username: "admin", Role: "admin"}
		ctx := context.WithValue(req.Context(), authUserKey{}, claims)
		req = req.WithContext(ctx)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHandleSetup_Success(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "POST", "/api/v1/auth/setup", map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": "securepassword",
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var user User
	if err := json.NewDecoder(w.Body).Decode(&user); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if user.Username != "admin" {
		t.Errorf("username = %q, want admin", user.Username)
	}
}

func TestHandleSetup_MissingFields(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "POST", "/api/v1/auth/setup", map[string]string{
		"username": "admin",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSetup_SecondTime(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	doRequest(mux, "POST", "/api/v1/auth/setup", map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": "securepassword",
	})

	w := doRequest(mux, "POST", "/api/v1/auth/setup", map[string]string{
		"username": "admin2",
		"email":    "admin2@example.com",
		"password": "securepassword",
	})
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleLogin_Success(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	doRequest(mux, "POST", "/api/v1/auth/setup", map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": "securepassword",
	})

	w := doRequest(mux, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "securepassword",
	})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var pair TokenPair
	if err := json.NewDecoder(w.Body).Decode(&pair); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if pair.RefreshToken == "" {
		t.Error("expected non-empty refresh_token")
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	doRequest(mux, "POST", "/api/v1/auth/setup", map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": "securepassword",
	})

	w := doRequest(mux, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "wrongpassword",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleLogin_MissingFields(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleRefresh_Success(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	doRequest(mux, "POST", "/api/v1/auth/setup", map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": "securepassword",
	})

	loginResp := doRequest(mux, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "securepassword",
	})
	var pair TokenPair
	_ = json.NewDecoder(loginResp.Body).Decode(&pair)

	w := doRequest(mux, "POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": pair.RefreshToken,
	})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var newPair TokenPair
	_ = json.NewDecoder(w.Body).Decode(&newPair)
	if newPair.AccessToken == "" {
		t.Error("expected new access token")
	}
}

func TestHandleRefresh_InvalidToken(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": "invalid-token",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleLogout_Success(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	doRequest(mux, "POST", "/api/v1/auth/setup", map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": "securepassword",
	})
	loginResp := doRequest(mux, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "securepassword",
	})
	var pair TokenPair
	_ = json.NewDecoder(loginResp.Body).Decode(&pair)

	w := doRequest(mux, "POST", "/api/v1/auth/logout", map[string]string{
		"refresh_token": pair.RefreshToken,
	})
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestHandleListUsers_RequiresAdmin(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Without auth context -- should return 401.
	w := doRequest(mux, "GET", "/api/v1/users", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleListUsers_WithAdmin(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	doRequest(mux, "POST", "/api/v1/auth/setup", map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": "securepassword",
	})

	w := doAuthRequest(mux, "GET", "/api/v1/users", "admin", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var users []User
	_ = json.NewDecoder(w.Body).Decode(&users)
	if len(users) != 1 {
		t.Errorf("users count = %d, want 1", len(users))
	}
}

func TestHandleListUsers_NonAdminForbidden(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Inject viewer claims.
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	claims := &Claims{UserID: "test", Username: "viewer", Role: "viewer"}
	ctx := context.WithValue(req.Context(), authUserKey{}, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestWriteAuthError_Format(t *testing.T) {
	w := httptest.NewRecorder()
	writeAuthError(w, http.StatusBadRequest, "something went wrong")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["detail"] != "something went wrong" {
		t.Errorf("detail = %q, want 'something went wrong'", resp["detail"])
	}
	if resp["status"] != float64(http.StatusBadRequest) {
		t.Errorf("status field = %v, want %d", resp["status"], http.StatusBadRequest)
	}
}
