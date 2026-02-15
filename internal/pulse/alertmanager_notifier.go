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
var _ Notifier = (*AlertmanagerNotifier)(nil)

// alertmanagerPayload matches the Prometheus Alertmanager webhook receiver format.
type alertmanagerPayload struct {
	Version string              `json:"version"`
	Status  string              `json:"status"`
	Alerts  []alertmanagerAlert `json:"alerts"`
}

// alertmanagerAlert represents a single alert in the Alertmanager payload.
type alertmanagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

// AlertmanagerNotifier delivers notifications in Prometheus Alertmanager webhook format.
type AlertmanagerNotifier struct {
	client *http.Client
	cfg    AlertmanagerConfig
}

// NewAlertmanagerNotifier creates a new Alertmanager-format notifier with the given config.
func NewAlertmanagerNotifier(cfg AlertmanagerConfig) *AlertmanagerNotifier {
	return &AlertmanagerNotifier{
		client: &http.Client{Timeout: 10 * time.Second},
		cfg:    cfg,
	}
}

// Notify sends an alert in Alertmanager webhook format to the configured URL.
func (n *AlertmanagerNotifier) Notify(ctx context.Context, alert *Alert, eventType string) error {
	status := "firing"
	if eventType == "resolved" {
		status = "resolved"
	}

	labels := map[string]string{
		"alertname": "SubNetreeAlert",
		"device_id": alert.DeviceID,
		"check_id":  alert.CheckID,
		"severity":  alert.Severity,
		"source":    "subnetree",
	}

	annotations := map[string]string{
		"summary": alert.Message,
	}

	amAlert := alertmanagerAlert{
		Status:      status,
		Labels:      labels,
		Annotations: annotations,
		StartsAt:    alert.TriggeredAt,
		EndsAt:      time.Time{}, // zero value for firing alerts
	}
	if alert.ResolvedAt != nil {
		amAlert.EndsAt = *alert.ResolvedAt
	}

	payload := alertmanagerPayload{
		Version: "4",
		Status:  status,
		Alerts:  []alertmanagerAlert{amAlert},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal alertmanager payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create alertmanager request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SubNetree-Alertmanager/0.1")

	// Add HMAC-SHA256 signature if secret is configured.
	if n.cfg.Secret != "" {
		mac := hmac.New(sha256.New, []byte(n.cfg.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Signature", sig)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("alertmanager POST %s: %w", n.cfg.URL, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck // drain body for connection reuse

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("alertmanager POST %s: status %d", n.cfg.URL, resp.StatusCode)
	}

	return nil
}

// Type returns the notifier type identifier.
func (n *AlertmanagerNotifier) Type() string {
	return "alertmanager"
}
