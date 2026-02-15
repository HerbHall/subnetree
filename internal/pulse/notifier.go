package pulse

import (
	"context"
	"time"
)

// Notifier delivers alert notifications through a specific channel type.
type Notifier interface {
	// Notify sends an alert notification. eventType is "triggered" or "resolved".
	Notify(ctx context.Context, alert *Alert, eventType string) error
	// Type returns the notifier type identifier (e.g., "webhook", "alertmanager", "email").
	Type() string
}

// NotificationChannel represents a configured notification delivery channel.
type NotificationChannel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`   // "webhook", "alertmanager", "email"
	Config    string    `json:"config"` // JSON blob
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WebhookConfig holds configuration for webhook notification delivery.
type WebhookConfig struct {
	URL     string            `json:"url"`
	Secret  string            `json:"secret,omitempty"` //nolint:gosec // G101: config field name, not a credential
	Headers map[string]string `json:"headers,omitempty"`
}

// AlertmanagerConfig holds configuration for Alertmanager-compatible webhook delivery.
type AlertmanagerConfig struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"` //nolint:gosec // G101: config field name, not a credential
}

// EmailConfig holds configuration for email notification delivery (stub).
type EmailConfig struct {
	SMTPHost string   `json:"smtp_host"`
	SMTPPort int      `json:"smtp_port"`
	Username string   `json:"username"`
	Password string   `json:"password,omitempty"` //nolint:gosec // G101: config field name, not a credential
	From     string   `json:"from"`
	To       []string `json:"to"`
	UseTLS   bool     `json:"use_tls"`
}
