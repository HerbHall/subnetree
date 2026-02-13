package pulse

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCPChecker_InterfaceCompliance(t *testing.T) {
	var _ Checker = (*TCPChecker)(nil)

	checker := NewTCPChecker(5 * time.Second)
	var _ Checker = checker
}

func TestNewTCPChecker(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"default", 5 * time.Second},
		{"short", 1 * time.Second},
		{"zero", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewTCPChecker(tt.timeout)
			if checker == nil {
				t.Fatal("NewTCPChecker() returned nil")
			}
			if checker.timeout != tt.timeout {
				t.Errorf("timeout = %v, want %v", checker.timeout, tt.timeout)
			}
		})
	}
}

func TestTCPChecker_Success(t *testing.T) {
	// Start a local TCP listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer ln.Close()

	// Accept connections in background.
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	checker := NewTCPChecker(5 * time.Second)
	result, err := checker.Check(context.Background(), ln.Addr().String())

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

func TestTCPChecker_ConnectionRefused(t *testing.T) {
	// Find a port that's definitely not listening by binding then closing.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() // Close immediately so port is not listening.

	checker := NewTCPChecker(2 * time.Second)
	result, err := checker.Check(context.Background(), addr)

	if err == nil {
		t.Error("Check() error = nil, want error for refused connection")
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

func TestTCPChecker_InvalidTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{"no port", "192.168.1.1"},
		{"empty", ""},
		{"just port", ":"},
		{"invalid format", "not-a-host-port"},
	}

	checker := NewTCPChecker(2 * time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := checker.Check(context.Background(), tt.target)

			if err == nil {
				t.Error("Check() error = nil, want error for invalid target")
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
		})
	}
}

func TestTCPChecker_ContextCancelled(t *testing.T) {
	// Use a non-routable IP that will cause the dial to hang.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	checker := NewTCPChecker(30 * time.Second) // Long timeout, context should cancel first.
	start := time.Now()
	result, err := checker.Check(ctx, "10.255.255.1:80")
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
	// Should return within a reasonable time after context cancellation.
	if elapsed > 2*time.Second {
		t.Errorf("Check() took %v, want < 2s (should respect context cancellation)", elapsed)
	}
}
