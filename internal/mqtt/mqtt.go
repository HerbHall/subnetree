package mqtt

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/HerbHall/subnetree/internal/pulse"
	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/pkg/models"
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
	logger    *zap.Logger
	cfg       Config
	client    pahomqtt.Client
	mu        sync.RWMutex
	haEnabled bool
	haPrefix  string
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
		if deps.Config.IsSet("ha_discovery") {
			m.cfg.HADiscovery = deps.Config.GetBool("ha_discovery")
		}
		if p := deps.Config.GetString("ha_discovery_prefix"); p != "" {
			m.cfg.HADiscoveryPrefix = p
		}
	}

	m.haEnabled = m.cfg.HADiscovery
	m.haPrefix = m.cfg.HADiscoveryPrefix

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
		zap.Bool("ha_discovery", m.haEnabled),
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

	// Publish HA discovery configs if enabled.
	if m.haEnabled {
		m.publishHAForEvent(event)
	}
}

// publishHAForEvent handles HA MQTT auto-discovery for device and alert events.
func (m *Module) publishHAForEvent(event plugin.Event) {
	switch event.Topic {
	case recon.TopicDeviceDiscovered, recon.TopicDeviceUpdated:
		device := extractDevice(event.Payload)
		if device == nil {
			return
		}
		configs := BuildDeviceDiscoveryConfigs(device, m.cfg.TopicPrefix, m.haPrefix)
		m.publishHADiscovery(configs)
		m.publishDeviceState(device)

	case recon.TopicDeviceLost:
		deviceID := extractDeviceID(event.Payload)
		if deviceID == "" {
			return
		}
		// Publish offline state before removing discovery configs.
		m.publishState(m.cfg.TopicPrefix+"/device/"+deviceID+"/online", "OFF")
		configs := BuildDeviceRemovalConfigs(deviceID, m.haPrefix)
		m.publishHADiscovery(configs)

	case "pulse.alert.triggered":
		alert := extractAlert(event.Payload)
		if alert == nil {
			return
		}
		cfg := BuildAlertDiscoveryConfig(alert.DeviceID, alert.DeviceName, alert.ID, alert.Severity, m.cfg.TopicPrefix, m.haPrefix)
		if len(cfg.Payload) > 0 {
			m.publishHADiscovery([]DiscoveryConfig{cfg})
		}
		m.publishState(m.cfg.TopicPrefix+"/alert/"+alert.ID+"/state", "triggered")

	case "pulse.alert.resolved":
		alert := extractAlert(event.Payload)
		if alert == nil {
			return
		}
		m.publishState(m.cfg.TopicPrefix+"/alert/"+alert.ID+"/state", "resolved")
	}
}

// publishHADiscovery publishes a batch of HA discovery config payloads.
func (m *Module) publishHADiscovery(configs []DiscoveryConfig) {
	for i := range configs {
		var payload []byte
		if configs[i].Payload != nil {
			payload = configs[i].Payload
		}
		// Discovery configs are always retained so HA picks them up on restart.
		token := m.client.Publish(configs[i].Topic, m.cfg.QoS, true, payload)
		if !token.WaitTimeout(m.cfg.Timeout) {
			m.logger.Warn("ha discovery publish timed out",
				zap.String("topic", configs[i].Topic),
			)
			continue
		}
		if token.Error() != nil {
			m.logger.Warn("ha discovery publish failed",
				zap.String("topic", configs[i].Topic),
				zap.Error(token.Error()),
			)
			continue
		}
		m.logger.Debug("ha discovery published",
			zap.String("topic", configs[i].Topic),
			zap.Bool("removal", len(configs[i].Payload) == 0),
		)
	}
}

// publishState publishes a retained state value to an MQTT topic.
func (m *Module) publishState(topic, value string) {
	token := m.client.Publish(topic, m.cfg.QoS, true, []byte(value))
	if !token.WaitTimeout(m.cfg.Timeout) {
		m.logger.Warn("state publish timed out", zap.String("topic", topic))
		return
	}
	if token.Error() != nil {
		m.logger.Warn("state publish failed",
			zap.String("topic", topic),
			zap.Error(token.Error()),
		)
		return
	}
	m.logger.Debug("state published", zap.String("topic", topic), zap.String("value", value))
}

// publishDeviceState publishes current state values for a device's HA entities.
func (m *Module) publishDeviceState(device *models.Device) {
	prefix := m.cfg.TopicPrefix + "/device/" + device.ID

	// Online status.
	onlineState := "OFF"
	if device.Status == models.DeviceStatusOnline || device.Status == models.DeviceStatusDegraded {
		onlineState = "ON"
	}
	m.publishState(prefix+"/online", onlineState)

	// Device type.
	m.publishState(prefix+"/type", string(device.DeviceType))

	// IP address.
	if len(device.IPAddresses) > 0 {
		m.publishState(prefix+"/ip", device.IPAddresses[0])
	}
}

// extractDevice attempts to extract a *models.Device from an event payload.
func extractDevice(payload interface{}) *models.Device {
	switch v := payload.(type) {
	case *recon.DeviceEvent:
		return v.Device
	case recon.DeviceEvent:
		return v.Device
	default:
		// Try JSON round-trip for payloads that were serialized.
		data, err := json.Marshal(payload)
		if err != nil {
			return nil
		}
		var de recon.DeviceEvent
		if err := json.Unmarshal(data, &de); err != nil {
			return nil
		}
		return de.Device
	}
}

// extractDeviceID attempts to extract a device ID from a device-lost event payload.
func extractDeviceID(payload interface{}) string {
	switch v := payload.(type) {
	case recon.DeviceLostEvent:
		return v.DeviceID
	case *recon.DeviceLostEvent:
		return v.DeviceID
	default:
		data, err := json.Marshal(payload)
		if err != nil {
			return ""
		}
		var dle recon.DeviceLostEvent
		if err := json.Unmarshal(data, &dle); err != nil {
			return ""
		}
		return dle.DeviceID
	}
}

// extractAlert attempts to extract a *pulse.Alert from an event payload.
func extractAlert(payload interface{}) *pulse.Alert {
	switch v := payload.(type) {
	case *pulse.Alert:
		return v
	case pulse.Alert:
		return &v
	default:
		data, err := json.Marshal(payload)
		if err != nil {
			return nil
		}
		var a pulse.Alert
		if err := json.Unmarshal(data, &a); err != nil {
			return nil
		}
		if a.ID == "" {
			return nil
		}
		return &a
	}
}
