package pulse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPChecker_InterfaceCompliance(t *testing.T) {
	var _ Checker = (*HTTPChecker)(nil)

	checker := NewHTTPChecker(5 * time.Second)
	var _ Checker = checker
}

func TestNewHTTPChecker(t *testing.T) {
	checker := NewHTTPChecker(10 * time.Second)
	if checker == nil {
		t.Fatal("NewHTTPChecker() returned nil")
	}
	if checker.client == nil {
		t.Fatal("NewHTTPChecker() client is nil")
	}
	if checker.client.Timeout != 10*time.Second {
		t.Errorf("client.Timeout = %v, want 10s", checker.client.Timeout)
	}
}

func TestHTTPChecker_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker(5 * time.Second)
	result, err := checker.Check(context.Background(), server.URL)

	if err != nil {
		t.Errorf("Check() error = %v, want nil", err)
	}
	if result == nil {
		t.Fatal("Check() returned nil result")
	}
	if !result.Success {
		t.Errorf("Check() Success = false, want true")
	}
	if result.LatencyMs < 0 {
		t.Errorf("Check() LatencyMs = %v, want >= 0", result.LatencyMs)
	}
	if result.ErrorMessage != "" {
		t.Errorf("Check() ErrorMessage = %q, want empty", result.ErrorMessage)
	}
	if result.CheckedAt.IsZero() {
		t.Error("Check() CheckedAt is zero")
	}
}

func TestHTTPChecker_Non2xx(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"server error", http.StatusInternalServerError},
		{"service unavailable", http.StatusServiceUnavailable},
		{"not found", http.StatusNotFound},
		{"forbidden", http.StatusForbidden},
		{"redirect", http.StatusMovedPermanently},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			checker := NewHTTPChecker(5 * time.Second)
			result, err := checker.Check(context.Background(), server.URL)

			if err == nil {
				t.Error("Check() error = nil, want error for non-2xx response")
			}
			if result == nil {
				t.Fatal("Check() returned nil result")
			}
			if result.Success {
				t.Error("Check() Success = true, want false")
			}
			if result.ErrorMessage == "" {
				t.Error("Check() ErrorMessage is empty, want non-empty")
			}
			if result.LatencyMs < 0 {
				t.Errorf("Check() LatencyMs = %v, want >= 0", result.LatencyMs)
			}
		})
	}
}

func TestHTTPChecker_ConnectionRefused(t *testing.T) {
	// Use a URL that will refuse the connection.
	checker := NewHTTPChecker(2 * time.Second)
	result, err := checker.Check(context.Background(), "http://127.0.0.1:1")

	if err == nil {
		t.Error("Check() error = nil, want error for connection refused")
	}
	if result == nil {
		t.Fatal("Check() returned nil result")
	}
	if result.Success {
		t.Error("Check() Success = true, want false")
	}
	if result.ErrorMessage == "" {
		t.Error("Check() ErrorMessage is empty, want non-empty")
	}
}

func TestHTTPChecker_InvalidURL(t *testing.T) {
	checker := NewHTTPChecker(2 * time.Second)
	result, err := checker.Check(context.Background(), "://invalid")

	if err == nil {
		t.Error("Check() error = nil, want error for invalid URL")
	}
	if result == nil {
		t.Fatal("Check() returned nil result")
	}
	if result.Success {
		t.Error("Check() Success = true, want false")
	}
}

func TestHTTPChecker_HTTPS_SelfSigned(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker(5 * time.Second)
	result, err := checker.Check(context.Background(), server.URL)

	if err != nil {
		t.Errorf("Check() error = %v, want nil (self-signed certs should be accepted)", err)
	}
	if result == nil {
		t.Fatal("Check() returned nil result")
	}
	if !result.Success {
		t.Errorf("Check() Success = false, want true (self-signed cert should be accepted)")
	}
}

func TestHTTPChecker_ContextCancelled(t *testing.T) {
	// Server that delays response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	checker := NewHTTPChecker(30 * time.Second)
	start := time.Now()
	result, err := checker.Check(ctx, server.URL)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Check() error = nil, want error for cancelled context")
	}
	if result == nil {
		t.Fatal("Check() returned nil result")
	}
	if result.Success {
		t.Error("Check() Success = true, want false")
	}
	if elapsed > 2*time.Second {
		t.Errorf("Check() took %v, want < 2s (should respect context cancellation)", elapsed)
	}
}

func TestHTTPChecker_2xxRange(t *testing.T) {
	// Test that all 2xx codes are considered success.
	codes := []int{200, 201, 202, 204}
	for _, code := range codes {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
		}))

		checker := NewHTTPChecker(5 * time.Second)
		result, err := checker.Check(context.Background(), server.URL)
		server.Close()

		if err != nil {
			t.Errorf("status %d: Check() error = %v, want nil", code, err)
		}
		if result == nil {
			t.Fatalf("status %d: Check() returned nil result", code)
		}
		if !result.Success {
			t.Errorf("status %d: Check() Success = false, want true", code)
		}
	}
}
