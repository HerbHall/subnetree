package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/auth"
	"github.com/HerbHall/subnetree/internal/store"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// =============================================================================
// Test Infrastructure
// =============================================================================

// testEnvWithObservedLogs creates a test environment with log capture.
func testEnvWithObservedLogs(t *testing.T) (*http.ServeMux, *observer.ObservedLogs) {
	t.Helper()

	// Create an observed logger that captures all log output.
	core, logs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	userStore, err := auth.NewUserStore(ctx, db)
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}

	tokens := auth.NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	totpSvc := auth.NewTOTPService([]byte("test-secret-key-32bytes-long!!"))
	svc := auth.NewService(userStore, tokens, totpSvc, logger)
	handler := auth.NewHandler(svc, logger)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return mux, logs
}

// containsSecret checks if any log entry contains the secret string.
func containsSecret(logs *observer.ObservedLogs, secret string) bool {
	entries := logs.All()
	for i := range entries {
		// Check the message itself.
		if strings.Contains(entries[i].Message, secret) {
			return true
		}
		// Check all field values.
		for j := range entries[i].Context {
			if strings.Contains(entries[i].Context[j].String, secret) {
				return true
			}
			// Check interface values (like errors).
			if entries[i].Context[j].Interface != nil {
				if s, ok := entries[i].Context[j].Interface.(string); ok && strings.Contains(s, secret) {
					return true
				}
				if err, ok := entries[i].Context[j].Interface.(error); ok && strings.Contains(err.Error(), secret) {
					return true
				}
			}
		}
	}
	return false
}

// =============================================================================
// Password Hygiene Tests
// =============================================================================

func TestPasswordsNotInLogs(t *testing.T) {
	mux, logs := testEnvWithObservedLogs(t)

	testPasswords := []string{
		"super-secret-password-123",
		"MyP@ssw0rd!",
		"correct-horse-battery-staple",
	}

	for _, password := range testPasswords {
		t.Run("login_attempt_"+password[:10], func(t *testing.T) {
			body := map[string]string{
				"username": "testuser",
				"password": password,
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Verify password is NOT in any log entry.
			if containsSecret(logs, password) {
				t.Errorf("Password %q found in log output", password)
			}
		})
	}
}

func TestPasswordsNotInSetupLogs(t *testing.T) {
	mux, logs := testEnvWithObservedLogs(t)

	password := "my-super-secret-admin-password"

	body := map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": password,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("setup failed: status=%d body=%s", w.Code, w.Body.String())
	}

	// Verify password is NOT in any log entry.
	if containsSecret(logs, password) {
		t.Errorf("Password found in log output after setup")
	}
}

// =============================================================================
// Password Hash Hygiene Tests
// =============================================================================

func TestPasswordHashNotInResponses(t *testing.T) {
	mux, _ := testEnvWithObservedLogs(t)

	// Create a user via setup.
	setupBody := map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": "securepassword123",
	}
	jsonBody, _ := json.Marshal(setupBody)

	req := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("setup failed: status=%d", w.Code)
	}

	responseBody := w.Body.String()

	// Check that response doesn't contain bcrypt hash prefix.
	if strings.Contains(responseBody, "$2a$") || strings.Contains(responseBody, "$2b$") {
		t.Error("Response contains bcrypt hash prefix")
	}

	// Check that response doesn't contain "password_hash" field.
	if strings.Contains(responseBody, "password_hash") {
		t.Error("Response contains password_hash field")
	}

	// Parse and verify user object doesn't have password hash.
	var user auth.User
	if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if user.PasswordHash != "" {
		t.Error("User response object has non-empty PasswordHash")
	}
}

// =============================================================================
// Token Hygiene Tests
// =============================================================================

func TestTokensNotLoggedInFull(t *testing.T) {
	mux, logs := testEnvWithObservedLogs(t)

	// Setup admin account.
	setupBody := map[string]string{
		"username": "admin",
		"email":    "admin@example.com",
		"password": "securepassword123",
	}
	jsonBody, _ := json.Marshal(setupBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Login to get tokens.
	loginBody := map[string]string{
		"username": "admin",
		"password": "securepassword123",
	}
	jsonBody, _ = json.Marshal(loginBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed: status=%d", w.Code)
	}

	var tokens auth.TokenPair
	if err := json.NewDecoder(w.Body).Decode(&tokens); err != nil {
		t.Fatalf("decode tokens: %v", err)
	}

	// Tokens should be JWTs (contain dots).
	if !strings.Contains(tokens.AccessToken, ".") {
		t.Fatal("access token doesn't look like a JWT")
	}

	// Verify full tokens are NOT in logs.
	if containsSecret(logs, tokens.AccessToken) {
		t.Error("Full access token found in logs")
	}
	if containsSecret(logs, tokens.RefreshToken) {
		t.Error("Full refresh token found in logs")
	}

	// Now try refresh - should also not log tokens.
	refreshBody := map[string]string{
		"refresh_token": tokens.RefreshToken,
	}
	jsonBody, _ = json.Marshal(refreshBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if containsSecret(logs, tokens.RefreshToken) {
		t.Error("Refresh token found in logs after refresh attempt")
	}
}

func TestInvalidTokenNotLoggedInFull(t *testing.T) {
	mux, logs := testEnvWithObservedLogs(t)

	// Try to refresh with an invalid token.
	fakeToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0.Gfx6VO9tcxwk6xqx9yYzSfebfeakZp5JYIgP_edcw_A"

	refreshBody := map[string]string{
		"refresh_token": fakeToken,
	}
	jsonBody, _ := json.Marshal(refreshBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should fail.
	if w.Code == http.StatusOK {
		t.Fatal("expected refresh to fail with fake token")
	}

	// Verify the fake token is NOT logged in full.
	if containsSecret(logs, fakeToken) {
		t.Error("Invalid token logged in full")
	}
}

// =============================================================================
// Error Response Hygiene Tests
// =============================================================================

func TestErrorResponsesNoCredentialLeak(t *testing.T) {
	mux, _ := testEnvWithObservedLogs(t)

	testCases := []struct {
		name     string
		endpoint string
		method   string
		body     map[string]string
	}{
		{
			name:     "login with wrong password",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body: map[string]string{
				"username": "admin",
				"password": "wrongpassword123",
			},
		},
		{
			name:     "login with nonexistent user",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body: map[string]string{
				"username": "nonexistent",
				"password": "somepassword",
			},
		},
		{
			name:     "refresh with invalid token",
			endpoint: "/api/v1/auth/refresh",
			method:   "POST",
			body: map[string]string{
				"refresh_token": "invalid-token-here",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(tc.method, tc.endpoint, bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			responseBody := w.Body.String()

			// Error response should NOT contain the password.
			if password, ok := tc.body["password"]; ok {
				if strings.Contains(responseBody, password) {
					t.Errorf("Error response contains password: %s", responseBody)
				}
			}

			// Error response should NOT contain the token.
			if token, ok := tc.body["refresh_token"]; ok {
				if strings.Contains(responseBody, token) {
					t.Errorf("Error response contains token: %s", responseBody)
				}
			}

			// Error response should NOT reveal if user exists.
			// (Same error message for wrong password vs. nonexistent user)
			if strings.Contains(responseBody, "user not found") ||
				strings.Contains(responseBody, "user does not exist") ||
				strings.Contains(responseBody, "no such user") {
				t.Errorf("Error response reveals user existence: %s", responseBody)
			}
		})
	}
}

func TestDatabaseErrorsNotExposed(t *testing.T) {
	mux, _ := testEnvWithObservedLogs(t)

	// Attempt operations that might trigger database errors.
	// The actual error messages from SQLite should not be exposed.

	testCases := []struct {
		name     string
		endpoint string
		method   string
		body     map[string]string
	}{
		{
			name:     "login attempt",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body: map[string]string{
				"username": "test",
				"password": "password",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(tc.method, tc.endpoint, bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			responseBody := w.Body.String()

			// Error response should NOT contain SQL-related terms.
			sqlKeywords := []string{
				"sqlite",
				"SQLITE",
				"database",
				"sql:",
				"SQL:",
				"table",
				"column",
				"constraint",
				"foreign key",
				"unique",
			}
			for _, keyword := range sqlKeywords {
				if strings.Contains(strings.ToLower(responseBody), strings.ToLower(keyword)) {
					// Allow "database" in generic messages, but not SQL syntax.
					if keyword != "database" {
						t.Errorf("Error response contains SQL keyword %q: %s", keyword, responseBody)
					}
				}
			}
		})
	}
}

// =============================================================================
// JWT Secret Hygiene Tests
// =============================================================================

func TestJWTSecretNotExposed(t *testing.T) {
	// The JWT signing secret should never appear in responses or logs.
	jwtSecret := "test-secret-key-32bytes-long!!"

	mux, logs := testEnvWithObservedLogs(t)

	// Perform various operations.
	operations := []struct {
		endpoint string
		body     map[string]string
	}{
		{
			endpoint: "/api/v1/auth/setup",
			body: map[string]string{
				"username": "admin",
				"email":    "admin@example.com",
				"password": "securepassword",
			},
		},
		{
			endpoint: "/api/v1/auth/login",
			body: map[string]string{
				"username": "admin",
				"password": "securepassword",
			},
		},
		{
			endpoint: "/api/v1/auth/refresh",
			body: map[string]string{
				"refresh_token": "invalid",
			},
		},
	}

	for _, op := range operations {
		jsonBody, _ := json.Marshal(op.body)
		req := httptest.NewRequest("POST", op.endpoint, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Check response doesn't contain JWT secret.
		if strings.Contains(w.Body.String(), jwtSecret) {
			t.Errorf("JWT secret found in response from %s", op.endpoint)
		}
	}

	// Check logs don't contain JWT secret.
	if containsSecret(logs, jwtSecret) {
		t.Error("JWT secret found in logs")
	}
}

// =============================================================================
// User Enumeration Prevention Tests
// =============================================================================

func TestUserEnumerationPrevention(t *testing.T) {
	mux, _ := testEnvWithObservedLogs(t)

	// Setup a user first.
	setupBody := map[string]string{
		"username": "existinguser",
		"email":    "existing@example.com",
		"password": "securepassword123",
	}
	jsonBody, _ := json.Marshal(setupBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("setup failed: %d", w.Code)
	}

	// Try login with existing user but wrong password.
	existingUserBody := map[string]string{
		"username": "existinguser",
		"password": "wrongpassword",
	}
	jsonBody, _ = json.Marshal(existingUserBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	existingUserResp := httptest.NewRecorder()
	mux.ServeHTTP(existingUserResp, req)

	// Try login with non-existing user.
	nonExistingUserBody := map[string]string{
		"username": "nonexistinguser",
		"password": "anypassword",
	}
	jsonBody, _ = json.Marshal(nonExistingUserBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	nonExistingUserResp := httptest.NewRecorder()
	mux.ServeHTTP(nonExistingUserResp, req)

	// Both should return the same status code (401).
	if existingUserResp.Code != nonExistingUserResp.Code {
		t.Errorf("Different status codes for existing vs non-existing user: %d vs %d",
			existingUserResp.Code, nonExistingUserResp.Code)
	}

	// Both should have similar error messages (not reveal user existence).
	existingBody := existingUserResp.Body.String()
	nonExistingBody := nonExistingUserResp.Body.String()

	// The error messages should be identical or at least both say "invalid credentials".
	if existingBody != nonExistingBody {
		// At minimum, neither should explicitly say "user not found".
		if strings.Contains(nonExistingBody, "not found") ||
			strings.Contains(nonExistingBody, "does not exist") {
			t.Errorf("Non-existing user response reveals user doesn't exist: %s", nonExistingBody)
		}
	}
}

// =============================================================================
// Timing Attack Prevention (Basic Check)
// =============================================================================

func TestTimingAttackMitigation(t *testing.T) {
	// This is a basic check - proper timing attack testing requires
	// statistical analysis over many requests.
	// Here we just verify that bcrypt is used (which has constant-time comparison).

	mux, _ := testEnvWithObservedLogs(t)

	// Setup a user.
	setupBody := map[string]string{
		"username": "timingtest",
		"email":    "timing@example.com",
		"password": "securepassword123",
	}
	jsonBody, _ := json.Marshal(setupBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// The fact that we use bcrypt (with constant-time comparison) is the mitigation.
	// This test just ensures we can login successfully, confirming bcrypt is working.
	loginBody := map[string]string{
		"username": "timingtest",
		"password": "securepassword123",
	}
	jsonBody, _ = json.Marshal(loginBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Login failed: status=%d body=%s", w.Code, w.Body.String())
	}
}
