package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/HerbHall/netvantage/internal/recon"
	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin          = (*Module)(nil)
	_ plugin.EventSubscriber = (*Module)(nil)
)

// Config holds the webhook plugin configuration.
type Config struct {
	URL     string
	Timeout time.Duration
	Enabled bool
}

// Module implements the Webhook notifier plugin.
type Module struct {
	logger *zap.Logger
	cfg    Config
	client *http.Client
}

// New creates a new Webhook plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "webhook",
		Version:     "0.1.0",
		Description: "Sends HTTP POST notifications to a configurable webhook URL on device events",
		Roles:       []string{"notification"},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger

	// Defaults.
	m.cfg = Config{
		Timeout: 10 * time.Second,
		Enabled: true,
	}

	if deps.Config != nil {
		if u := deps.Config.GetString("url"); u != "" {
			m.cfg.URL = u
		}
		if d := deps.Config.GetDuration("timeout"); d > 0 {
			m.cfg.Timeout = d
		}
		if deps.Config.IsSet("enabled") {
			m.cfg.Enabled = deps.Config.GetBool("enabled")
		}
	}

	m.client = &http.Client{Timeout: m.cfg.Timeout}

	if m.cfg.URL == "" {
		m.logger.Warn("webhook URL not configured; notifications will be dropped",
			zap.String("component", "webhook"),
		)
	}

	m.logger.Info("webhook module initialized",
		zap.String("url", m.cfg.URL),
		zap.Duration("timeout", m.cfg.Timeout),
		zap.Bool("enabled", m.cfg.Enabled),
	)
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.logger.Info("webhook module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("webhook module stopped")
	return nil
}

// Subscriptions implements plugin.EventSubscriber.
func (m *Module) Subscriptions() []plugin.Subscription {
	return []plugin.Subscription{
		{Topic: recon.TopicDeviceDiscovered, Handler: m.handleEvent},
		{Topic: recon.TopicDeviceUpdated, Handler: m.handleEvent},
		{Topic: recon.TopicDeviceLost, Handler: m.handleEvent},
	}
}

// WebhookPayload is the JSON body sent to the webhook URL.
type WebhookPayload struct {
	Event     string `json:"event"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

func (m *Module) handleEvent(ctx context.Context, event plugin.Event) {
	if !m.cfg.Enabled || m.cfg.URL == "" {
		return
	}

	payload := WebhookPayload{
		Event:     event.Topic,
		Source:    event.Source,
		Timestamp: event.Timestamp.UTC().Format(time.RFC3339),
		Data:      event.Payload,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		m.logger.Error("failed to marshal webhook payload",
			zap.String("topic", event.Topic),
			zap.Error(err),
		)
		return
	}

	m.send(ctx, body, event.Topic)
}

func (m *Module) send(ctx context.Context, body []byte, topic string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.URL, bytes.NewReader(body))
	if err != nil {
		m.logger.Error("failed to create webhook request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NetVantage-Webhook/0.1")

	resp, err := m.client.Do(req)
	if err != nil {
		m.logger.Warn("webhook delivery failed",
			zap.String("url", m.cfg.URL),
			zap.String("topic", topic),
			zap.Error(err),
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		m.logger.Warn("webhook endpoint returned error",
			zap.String("url", m.cfg.URL),
			zap.String("topic", topic),
			zap.Int("status_code", resp.StatusCode),
		)
		return
	}

	m.logger.Debug("webhook delivered",
		zap.String("topic", topic),
		zap.Int("status_code", resp.StatusCode),
	)
}
