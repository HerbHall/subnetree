package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// TailscaleDevice represents a device returned by the Tailscale API.
type TailscaleDevice struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`      // Full MagicDNS name (e.g., "host.tail12345.ts.net")
	Addresses []string `json:"addresses"` // Tailscale IPs (100.x.y.z, fd7a:...)
	OS        string   `json:"os"`
	Hostname  string   `json:"hostname"` // Short hostname
	Tags      []string `json:"tags"`
	LastSeen  string   `json:"lastSeen"` // RFC3339
	Online    bool     `json:"online"`
	NodeKey   string   `json:"nodeKey"`
}

// TailscaleClient is a thin HTTP wrapper for the Tailscale API v2.
type TailscaleClient struct {
	apiKey  string
	baseURL string
	tailnet string
	http    *http.Client
}

// NewClient creates a TailscaleClient for the given tailnet.
func NewClient(apiKey, baseURL, tailnet string) *TailscaleClient {
	return &TailscaleClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		tailnet: tailnet,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// devicesResponse wraps the Tailscale API list-devices response.
type devicesResponse struct {
	Devices []TailscaleDevice `json:"devices"`
}

// ListDevices fetches all devices in the tailnet.
func (c *TailscaleClient) ListDevices(ctx context.Context) (devices []TailscaleDevice, err error) {
	url := fmt.Sprintf("%s/api/v2/tailnet/%s/devices", c.baseURL, c.tailnet)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tailscale API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")
		secs, _ := strconv.Atoi(retryAfter)
		if secs == 0 {
			secs = 60
		}
		return nil, fmt.Errorf("rate limited by Tailscale API, retry after %ds", secs)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("tailscale API returned %d: %s", resp.StatusCode, string(body))
	}

	var result devicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Devices, nil
}
