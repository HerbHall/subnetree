package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/auth"
	"github.com/HerbHall/subnetree/internal/store"
	"go.uber.org/zap"
)

// testAuthEnv creates a test environment with auth handler registered.
func testAuthEnv(t *testing.T) *http.ServeMux {
	t.Helper()

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

	logger, _ := zap.NewDevelopment()
	tokens := auth.NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	totpSvc := auth.NewTOTPService([]byte("test-secret-key-32bytes-long!!"))
	svc := auth.NewService(userStore, tokens, totpSvc, logger)
	handler := auth.NewHandler(svc, logger)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return mux
}

// =============================================================================
// Malformed JSON Tests
// =============================================================================

func TestMalformedJSON(t *testing.T) {
	mux := testAuthEnv(t)

	tests := []struct {
		name     string
		endpoint string
		method   string
		body     string
		wantCode int
	}{
		{
			name:     "truncated JSON",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": "admin", "password":`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid JSON syntax",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{username: admin}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "array instead of object",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `["admin", "password"]`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "string instead of object",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `"just a string"`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "null body",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `null`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty body",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     ``,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "setup - truncated JSON",
			endpoint: "/api/v1/auth/setup",
			method:   "POST",
			body:     `{"username": "admin"`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "refresh - invalid JSON",
			endpoint: "/api/v1/auth/refresh",
			method:   "POST",
			body:     `not json at all`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "logout - malformed",
			endpoint: "/api/v1/auth/logout",
			method:   "POST",
			body:     `{refresh_token: missing_quotes}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantCode, w.Body.String())
			}
		})
	}
}

// =============================================================================
// Empty and Null Input Tests
// =============================================================================

func TestEmptyAndNullInputs(t *testing.T) {
	mux := testAuthEnv(t)

	tests := []struct {
		name     string
		endpoint string
		method   string
		body     string
		wantCode int
	}{
		{
			name:     "login - empty username",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": "", "password": "secret"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "login - empty password",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": "admin", "password": ""}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "login - null username",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": null, "password": "secret"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "login - missing username key",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"password": "secret"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "login - missing password key",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": "admin"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "login - whitespace only username",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": "   ", "password": "secret"}`,
			wantCode: http.StatusUnauthorized, // Whitespace is treated as a valid (but non-existent) username.
		},
		{
			name:     "setup - empty object",
			endpoint: "/api/v1/auth/setup",
			method:   "POST",
			body:     `{}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "setup - null email",
			endpoint: "/api/v1/auth/setup",
			method:   "POST",
			body:     `{"username": "admin", "email": null, "password": "securepassword"}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "refresh - empty token",
			endpoint: "/api/v1/auth/refresh",
			method:   "POST",
			body:     `{"refresh_token": ""}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "refresh - null token",
			endpoint: "/api/v1/auth/refresh",
			method:   "POST",
			body:     `{"refresh_token": null}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantCode, w.Body.String())
			}
		})
	}
}

// =============================================================================
// SQL Injection Tests
// =============================================================================

func TestSQLInjectionPatterns(t *testing.T) {
	mux := testAuthEnv(t)

	// SQL injection payloads that should be safely handled.
	sqlPayloads := []string{
		`' OR '1'='1`,
		`'; DROP TABLE users; --`,
		`" OR "1"="1`,
		`1; DELETE FROM users`,
		`admin'--`,
		`' UNION SELECT * FROM users --`,
		`'; EXEC xp_cmdshell('dir'); --`,
		`' AND 1=0 UNION SELECT password FROM users --`,
		`Robert'); DROP TABLE students;--`,
		`1' AND '1'='1`,
	}

	for _, payload := range sqlPayloads {
		t.Run("login_username_"+payload[:minInt(len(payload), 20)], func(t *testing.T) {
			body := map[string]string{
				"username": payload,
				"password": "testpassword",
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Should get 401 (invalid credentials) or 400 (bad input), not 500 (SQL error).
			if w.Code == http.StatusInternalServerError {
				t.Errorf("SQL injection payload caused server error; status = %d, body: %s", w.Code, w.Body.String())
			}
		})

		t.Run("login_password_"+payload[:minInt(len(payload), 20)], func(t *testing.T) {
			body := map[string]string{
				"username": "admin",
				"password": payload,
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				t.Errorf("SQL injection payload caused server error; status = %d, body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// =============================================================================
// XSS Payload Tests
// =============================================================================

func TestXSSPayloads(t *testing.T) {
	mux := testAuthEnv(t)

	xssPayloads := []string{
		`<script>alert('xss')</script>`,
		`<img src=x onerror=alert('xss')>`,
		`<svg onload=alert('xss')>`,
		`javascript:alert('xss')`,
		`<body onload=alert('xss')>`,
		`<iframe src="javascript:alert('xss')">`,
		`<input onfocus=alert('xss') autofocus>`,
		`"><script>alert('xss')</script>`,
		`'><script>alert('xss')</script>`,
		`<a href="javascript:alert('xss')">click</a>`,
		`<div style="background:url(javascript:alert('xss'))">`,
		`\u003cscript\u003ealert('xss')\u003c/script\u003e`,
	}

	for _, payload := range xssPayloads {
		t.Run("setup_username_xss", func(t *testing.T) {
			body := map[string]string{
				"username": payload,
				"email":    "test@example.com",
				"password": "securepassword123",
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// XSS payload in username should be rejected or safely handled.
			// Should not cause 500 error, and response should not reflect raw HTML.
			if w.Code == http.StatusInternalServerError {
				t.Errorf("XSS payload caused server error; status = %d", w.Code)
			}

			// Check that response doesn't contain unescaped XSS payload.
			responseBody := w.Body.String()
			if strings.Contains(responseBody, "<script>") {
				t.Errorf("Response contains unescaped script tag: %s", responseBody)
			}
		})

		t.Run("setup_email_xss", func(t *testing.T) {
			body := map[string]string{
				"username": "testuser",
				"email":    payload + "@example.com",
				"password": "securepassword123",
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/api/v1/auth/setup", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code == http.StatusInternalServerError {
				t.Errorf("XSS payload caused server error; status = %d", w.Code)
			}
		})
	}
}

// =============================================================================
// Path Traversal Tests
// =============================================================================

func TestPathTraversalPatterns(t *testing.T) {
	mux := testAuthEnv(t)

	traversalPayloads := []string{
		`../../../etc/passwd`,
		`..\..\..\..\windows\system32\config\sam`,
		`....//....//....//etc/passwd`,
		`..%2f..%2f..%2fetc/passwd`,
		`..%252f..%252f..%252fetc/passwd`,
		`%2e%2e%2f%2e%2e%2f%2e%2e%2fetc/passwd`,
		`..%c0%af..%c0%af..%c0%afetc/passwd`,
		`/etc/passwd`,
		`C:\Windows\System32\config\SAM`,
		`file:///etc/passwd`,
	}

	for _, payload := range traversalPayloads {
		t.Run("user_id_"+payload[:minInt(len(payload), 15)], func(t *testing.T) {
			// Test path traversal in user ID parameter.
			req := httptest.NewRequest("GET", "/api/v1/users/"+payload, http.NoBody)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Should get 401 (unauthorized) or 404 (not found), not 500.
			if w.Code == http.StatusInternalServerError {
				t.Errorf("Path traversal payload caused server error; status = %d, body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// =============================================================================
// Integer Overflow and Boundary Tests
// =============================================================================

func TestIntegerBoundaries(t *testing.T) {
	mux := testAuthEnv(t)

	tests := []struct {
		name     string
		endpoint string
		method   string
		body     string
	}{
		{
			name:     "extremely large number in JSON",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": "admin", "password": "test", "attempt": 99999999999999999999999999999999}`,
		},
		{
			name:     "negative number where positive expected",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": "admin", "password": "test", "limit": -1}`,
		},
		{
			name:     "float where integer expected",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": "admin", "password": "test", "count": 1.5}`,
		},
		{
			name:     "scientific notation",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			body:     `{"username": "admin", "password": "test", "value": 1e308}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Extra fields should be ignored, not cause crashes.
			// Should get auth error (401) or bad request (400), not 500.
			if w.Code == http.StatusInternalServerError {
				t.Errorf("Integer boundary test caused server error; status = %d, body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// =============================================================================
// Content-Type Enforcement Tests
// =============================================================================

// TestContentTypeEnforcement verifies how the API handles various Content-Type headers.
// NOTE: The current implementation is lenient and parses JSON regardless of Content-Type.
// This documents current behavior. Strict Content-Type enforcement could be added as
// middleware in the future if required.
func TestContentTypeEnforcement(t *testing.T) {
	mux := testAuthEnv(t)

	tests := []struct {
		name        string
		contentType string
		body        string
		wantCode    int
	}{
		// Current behavior: API is lenient and attempts JSON parsing regardless of Content-Type.
		// Valid JSON with wrong Content-Type is parsed and processed (returns 401 for bad creds).
		{
			name:        "missing Content-Type - lenient parsing",
			contentType: "",
			body:        `{"username": "admin", "password": "test"}`,
			wantCode:    http.StatusUnauthorized, // Lenient: JSON parsed despite missing Content-Type.
		},
		{
			name:        "text/plain Content-Type - lenient parsing",
			contentType: "text/plain",
			body:        `{"username": "admin", "password": "test"}`,
			wantCode:    http.StatusUnauthorized, // Lenient: JSON parsed despite text/plain.
		},
		{
			name:        "text/html Content-Type - lenient parsing",
			contentType: "text/html",
			body:        `{"username": "admin", "password": "test"}`,
			wantCode:    http.StatusUnauthorized, // Lenient: JSON parsed despite text/html.
		},
		{
			name:        "application/xml Content-Type - invalid body",
			contentType: "application/xml",
			body:        `<root><username>admin</username><password>test</password></root>`,
			wantCode:    http.StatusBadRequest, // XML body fails JSON parsing.
		},
		{
			name:        "multipart/form-data Content-Type - lenient parsing",
			contentType: "multipart/form-data",
			body:        `{"username": "admin", "password": "test"}`,
			wantCode:    http.StatusUnauthorized, // Lenient: JSON parsed despite multipart.
		},
		{
			name:        "valid application/json",
			contentType: "application/json",
			body:        `{"username": "admin", "password": "test"}`,
			wantCode:    http.StatusUnauthorized, // Valid format, but invalid credentials.
		},
		{
			name:        "application/json with charset",
			contentType: "application/json; charset=utf-8",
			body:        `{"username": "admin", "password": "test"}`,
			wantCode:    http.StatusUnauthorized, // Should also be accepted.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantCode, w.Body.String())
			}
		})
	}
}

// =============================================================================
// Oversized Payload Tests
// =============================================================================

func TestOversizedPayloads(t *testing.T) {
	mux := testAuthEnv(t)

	tests := []struct {
		name     string
		endpoint string
		method   string
		size     int // Size in bytes.
	}{
		{
			name:     "1MB payload",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			size:     1 * 1024 * 1024,
		},
		{
			name:     "10MB payload",
			endpoint: "/api/v1/auth/login",
			method:   "POST",
			size:     10 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a large payload with valid JSON structure.
			largeValue := strings.Repeat("a", tt.size)
			body := `{"username": "` + largeValue + `", "password": "test"}`

			req := httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Oversized payloads should either:
			// - Return 413 Request Entity Too Large (if size limit enforced).
			// - Return 400 Bad Request (if parsed but rejected).
			// - Return 401 Unauthorized (if parsed and processed as invalid creds).
			// Should NOT return 500 Internal Server Error.
			if w.Code == http.StatusInternalServerError {
				t.Errorf("Oversized payload caused server error; status = %d", w.Code)
			}
		})
	}
}

// TestDeeplyNestedJSON tests handling of deeply nested JSON structures.
func TestDeeplyNestedJSON(t *testing.T) {
	mux := testAuthEnv(t)

	// Create deeply nested JSON.
	var nested strings.Builder
	depth := 1000
	for i := 0; i < depth; i++ {
		nested.WriteString(`{"nested":`)
	}
	nested.WriteString(`"value"`)
	for i := 0; i < depth; i++ {
		nested.WriteString(`}`)
	}

	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(nested.String()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should handle gracefully, not crash.
	if w.Code == http.StatusInternalServerError {
		t.Errorf("Deeply nested JSON caused server error; status = %d", w.Code)
	}
}

// =============================================================================
// Unicode and Encoding Edge Cases
// =============================================================================

func TestUnicodeAndEncodingEdgeCases(t *testing.T) {
	mux := testAuthEnv(t)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "null byte in string",
			body: `{"username": "admin\u0000injected", "password": "test"}`,
		},
		{
			name: "unicode escape sequences",
			body: `{"username": "\u0061\u0064\u006d\u0069\u006e", "password": "test"}`,
		},
		{
			name: "BOM at start",
			body: "\xef\xbb\xbf" + `{"username": "admin", "password": "test"}`,
		},
		{
			name: "emoji in username",
			body: `{"username": "admin🔐", "password": "test"}`,
		},
		{
			name: "RTL override character",
			body: `{"username": "admin\u202efdp", "password": "test"}`,
		},
		{
			name: "zero-width characters",
			body: `{"username": "a\u200bd\u200bm\u200bi\u200bn", "password": "test"}`,
		},
		{
			name: "invalid UTF-8 sequence",
			body: `{"username": "admin` + string([]byte{0xff, 0xfe}) + `", "password": "test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Should handle gracefully (400 bad request or 401 unauthorized), not 500.
			if w.Code == http.StatusInternalServerError {
				t.Errorf("Unicode edge case caused server error; status = %d, body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// =============================================================================
// Type Coercion Tests
// =============================================================================

func TestTypeCoercion(t *testing.T) {
	mux := testAuthEnv(t)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "number where string expected",
			body: `{"username": 12345, "password": "test"}`,
		},
		{
			name: "boolean where string expected",
			body: `{"username": true, "password": "test"}`,
		},
		{
			name: "array where string expected",
			body: `{"username": ["admin"], "password": "test"}`,
		},
		{
			name: "object where string expected",
			body: `{"username": {"name": "admin"}, "password": "test"}`,
		},
		{
			name: "string number",
			body: `{"username": "12345", "password": "test"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Type mismatches should be handled gracefully (400 or type coercion).
			// Should NOT cause 500.
			if w.Code == http.StatusInternalServerError {
				t.Errorf("Type coercion test caused server error; status = %d, body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// =============================================================================
// Response Format Validation
// =============================================================================

func TestErrorResponseFormat(t *testing.T) {
	mux := testAuthEnv(t)

	// Send a request that should result in an error.
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	// Verify error response is valid JSON.
	body, _ := io.ReadAll(w.Body)
	if !json.Valid(body) {
		t.Errorf("error response is not valid JSON: %s", body)
	}

	// Check Content-Type header.
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" && ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/json or application/problem+json", ct)
	}
}

// =============================================================================
// Helpers
// =============================================================================

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
