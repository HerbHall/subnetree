package pulse

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Compile-time interface guard.
var _ Checker = (*TCPChecker)(nil)

// TCPChecker tests TCP connectivity to host:port targets.
type TCPChecker struct {
	timeout time.Duration
}

// NewTCPChecker creates a new TCP checker with the given connection timeout.
func NewTCPChecker(timeout time.Duration) *TCPChecker {
	return &TCPChecker{timeout: timeout}
}

// Check connects to the target (host:port) and measures connection time.
func (c *TCPChecker) Check(ctx context.Context, target string) (*CheckResult, error) {
	// Validate target format: must include a port.
	_, _, err := net.SplitHostPort(target)
	if err != nil {
		return &CheckResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("invalid target %q: %v", target, err),
			CheckedAt:    time.Now().UTC(),
		}, fmt.Errorf("invalid target %q: %w", target, err)
	}

	start := time.Now()

	// Use a dialer with context for clean cancellation.
	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", target)
	elapsed := time.Since(start)

	if err != nil {
		return &CheckResult{
			Success:      false,
			LatencyMs:    float64(elapsed) / float64(time.Millisecond),
			ErrorMessage: err.Error(),
			CheckedAt:    time.Now().UTC(),
		}, fmt.Errorf("tcp connect %s: %w", target, err)
	}
	conn.Close()

	return &CheckResult{
		Success:   true,
		LatencyMs: float64(elapsed) / float64(time.Millisecond),
		CheckedAt: time.Now().UTC(),
	}, nil
}
