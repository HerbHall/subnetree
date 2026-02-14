package pulse

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// Compile-time interface guard.
var _ Checker = (*HTTPChecker)(nil)

// HTTPChecker tests HTTP/HTTPS endpoint reachability by sending GET requests.
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker creates a new HTTP checker with the given timeout.
// Self-signed TLS certificates are accepted (InsecureSkipVerify).
func NewHTTPChecker(timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}, //nolint:gosec // G402: monitoring must work with self-signed certs
				DisableKeepAlives: true,
			},
		},
	}
}

// Check sends a GET request to the target URL and checks for a 2xx response.
func (c *HTTPChecker) Check(ctx context.Context, target string) (*CheckResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return &CheckResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("invalid URL %q: %v", target, err),
			CheckedAt:    time.Now().UTC(),
		}, fmt.Errorf("invalid URL %q: %w", target, err)
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return &CheckResult{
			Success:      false,
			LatencyMs:    float64(elapsed) / float64(time.Millisecond),
			ErrorMessage: err.Error(),
			CheckedAt:    time.Now().UTC(),
		}, fmt.Errorf("http get %s: %w", target, err)
	}
	resp.Body.Close()

	result := &CheckResult{
		LatencyMs: float64(elapsed) / float64(time.Millisecond),
		CheckedAt: time.Now().UTC(),
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
	} else {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		return result, fmt.Errorf("http %s: status %d", target, resp.StatusCode)
	}

	return result, nil
}
