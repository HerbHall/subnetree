package pulse

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Compile-time interface guard.
var _ Notifier = (*WebhookNotifier)(nil)

// webhookPayload is the JSON body sent to webhook endpoints.
type webhookPayload struct {
	EventType string    `json:"event_type"`
	Alert     *Alert    `json:"alert"`
	Timestamp time.Time `json:"timestamp"`
}

// WebhookNotifier delivers notifications via HTTP POST to a configured URL.
type WebhookNotifier struct {
	client *http.Client
	cfg    WebhookConfig
}

// NewWebhookNotifier creates a new webhook notifier with the given config.
func NewWebhookNotifier(cfg WebhookConfig) *WebhookNotifier {
	return &WebhookNotifier{
		client: &http.Client{Timeout: 10 * time.Second},
		cfg:    cfg,
	}
}

// Notify sends an alert to the configured webhook URL.
func (w *WebhookNotifier) Notify(ctx context.Context, alert *Alert, eventType string) error {
	payload := webhookPayload{
		EventType: eventType,
		Alert:     alert,
		Timestamp: time.Now().UTC(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SubNetree-Webhook/0.1")

	// Add HMAC-SHA256 signature if secret is configured.
	if w.cfg.Secret != "" {
		mac := hmac.New(sha256.New, []byte(w.cfg.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Signature", sig)
	}

	// Add custom headers.
	for k, v := range w.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook POST %s: %w", w.cfg.URL, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck // drain body for connection reuse

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook POST %s: status %d", w.cfg.URL, resp.StatusCode)
	}

	return nil
}

// Type returns the notifier type identifier.
func (w *WebhookNotifier) Type() string {
	return "webhook"
}
