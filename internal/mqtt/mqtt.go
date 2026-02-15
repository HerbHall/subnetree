package mqtt

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/pkg/plugin"
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin          = (*Module)(nil)
	_ plugin.EventSubscriber = (*Module)(nil)
	_ plugin.HealthChecker   = (*Module)(nil)
)

// Module implements the MQTT publisher plugin. It subscribes to device and
// alert events via the event bus and publishes them to an MQTT broker,
// enabling Home Assistant auto-discovery and other integrations.
type Module struct {
	logger *zap.Logger
	cfg    Config
	client pahomqtt.Client
	mu     sync.RWMutex
}

// New creates a new MQTT publisher plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "mqtt",
		Version:     "0.1.0",
		Description: "Publishes device and alert events to an MQTT broker",
		Roles:       []string{"notification", "integration"},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger
	m.cfg = DefaultConfig()

	if deps.Config != nil {
		if u := deps.Config.GetString("broker_url"); u != "" {
			m.cfg.BrokerURL = u
		}
		if u := deps.Config.GetString("username"); u != "" {
			m.cfg.Username = u
		}
		if p := deps.Config.GetString("password"); p != "" {
			m.cfg.Password = p
		}
		if c := deps.Config.GetString("client_id"); c != "" {
			m.cfg.ClientID = c
		}
		if t := deps.Config.GetString("topic_prefix"); t != "" {
			m.cfg.TopicPrefix = t
		}
		if deps.Config.IsSet("qos") {
			m.cfg.QoS = byte(deps.Config.GetInt("qos"))
		}
		if deps.Config.IsSet("retain") {
			m.cfg.Retain = deps.Config.GetBool("retain")
		}
		if deps.Config.IsSet("use_tls") {
			m.cfg.UseTLS = deps.Config.GetBool("use_tls")
		}
		if d := deps.Config.GetDuration("timeout"); d > 0 {
			m.cfg.Timeout = d
		}
	}

	if m.cfg.BrokerURL == "" {
		m.logger.Warn("MQTT broker URL not configured; events will be dropped",
			zap.String("component", "mqtt"),
		)
	}

	m.logger.Info("mqtt module initialized",
		zap.String("broker_url", m.cfg.BrokerURL),
		zap.String("client_id", m.cfg.ClientID),
		zap.String("topic_prefix", m.cfg.TopicPrefix),
		zap.Uint8("qos", m.cfg.QoS),
	)
	return nil
}

func (m *Module) Start(_ context.Context) error {
	if m.cfg.BrokerURL == "" {
		m.logger.Info("mqtt module started (no-op: no broker configured)")
		return nil
	}

	opts := pahomqtt.NewClientOptions().
		AddBroker(m.cfg.BrokerURL).
		SetClientID(m.cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectTimeout(m.cfg.Timeout)

	if m.cfg.Username != "" {
		opts.SetUsername(m.cfg.Username)
		opts.SetPassword(m.cfg.Password) //nolint:gosec // G101: config field
	}

	m.client = pahomqtt.NewClient(opts)
	token := m.client.Connect()

	switch {
	case !token.WaitTimeout(m.cfg.Timeout):
		m.logger.Warn("mqtt connection timed out; will reconnect in background")
	case token.Error() != nil:
		m.logger.Warn("mqtt connection failed; will reconnect in background",
			zap.Error(token.Error()),
		)
	default:
		m.logger.Info("mqtt connected to broker",
			zap.String("broker_url", m.cfg.BrokerURL),
		)
	}
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.client != nil && m.client.IsConnected() {
		m.client.Disconnect(250)
		m.logger.Info("mqtt disconnected")
	}
	return nil
}

// Subscriptions implements plugin.EventSubscriber.
func (m *Module) Subscriptions() []plugin.Subscription {
	return []plugin.Subscription{
		{Topic: recon.TopicDeviceDiscovered, Handler: m.publishEvent},
		{Topic: recon.TopicDeviceUpdated, Handler: m.publishEvent},
		{Topic: recon.TopicDeviceLost, Handler: m.publishEvent},
		{Topic: "pulse.alert.triggered", Handler: m.publishEvent},
		{Topic: "pulse.alert.resolved", Handler: m.publishEvent},
	}
}

// Health implements plugin.HealthChecker.
func (m *Module) Health(_ context.Context) plugin.HealthStatus {
	if m.cfg.BrokerURL == "" {
		return plugin.HealthStatus{
			Status:  "healthy",
			Message: "no broker configured (no-op mode)",
		}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.client == nil || !m.client.IsConnected() {
		return plugin.HealthStatus{
			Status:  "degraded",
			Message: "not connected to MQTT broker",
		}
	}
	return plugin.HealthStatus{
		Status:  "healthy",
		Message: "connected to " + m.cfg.BrokerURL,
	}
}

// mqttTopicFromEvent maps an event bus topic to an MQTT topic path.
func (m *Module) mqttTopicFromEvent(eventTopic string) string {
	switch eventTopic {
	case recon.TopicDeviceDiscovered:
		return m.cfg.TopicPrefix + "/device/discovered"
	case recon.TopicDeviceUpdated:
		return m.cfg.TopicPrefix + "/device/updated"
	case recon.TopicDeviceLost:
		return m.cfg.TopicPrefix + "/device/lost"
	case "pulse.alert.triggered":
		return m.cfg.TopicPrefix + "/alert/triggered"
	case "pulse.alert.resolved":
		return m.cfg.TopicPrefix + "/alert/resolved"
	default:
		return m.cfg.TopicPrefix + "/unknown"
	}
}

func (m *Module) publishEvent(_ context.Context, event plugin.Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.client == nil || !m.client.IsConnected() {
		return
	}

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		m.logger.Warn("failed to marshal MQTT payload",
			zap.String("topic", event.Topic),
			zap.Error(err),
		)
		return
	}

	mqttTopic := m.mqttTopicFromEvent(event.Topic)
	token := m.client.Publish(mqttTopic, m.cfg.QoS, m.cfg.Retain, payload)
	if !token.WaitTimeout(m.cfg.Timeout) {
		m.logger.Warn("mqtt publish timed out",
			zap.String("mqtt_topic", mqttTopic),
		)
		return
	}
	if token.Error() != nil {
		m.logger.Warn("mqtt publish failed",
			zap.String("mqtt_topic", mqttTopic),
			zap.Error(token.Error()),
		)
		return
	}

	m.logger.Debug("mqtt event published",
		zap.String("mqtt_topic", mqttTopic),
		zap.String("event_topic", event.Topic),
	)
}
