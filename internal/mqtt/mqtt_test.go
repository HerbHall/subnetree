package mqtt

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"go.uber.org/zap"
)

func TestContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func TestInfo_ReturnsCorrectMetadata(t *testing.T) {
	m := New()
	info := m.Info()

	if info.Name != "mqtt" {
		t.Errorf("Name = %q, want mqtt", info.Name)
	}
	if info.Version != "0.1.0" {
		t.Errorf("Version = %q, want 0.1.0", info.Version)
	}
	if len(info.Roles) != 2 {
		t.Fatalf("Roles length = %d, want 2", len(info.Roles))
	}
	if info.Roles[0] != "notification" || info.Roles[1] != "integration" {
		t.Errorf("Roles = %v, want [notification integration]", info.Roles)
	}
	if info.APIVersion != plugin.APIVersionCurrent {
		t.Errorf("APIVersion = %d, want %d", info.APIVersion, plugin.APIVersionCurrent)
	}
}

func TestSubscriptions_ReturnsExpectedTopics(t *testing.T) {
	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{Logger: zap.NewNop()}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	subs := m.Subscriptions()
	if len(subs) != 5 {
		t.Fatalf("Subscriptions() returned %d, want 5", len(subs))
	}

	topics := make(map[string]bool)
	for _, s := range subs {
		topics[s.Topic] = true
	}

	expected := []string{
		recon.TopicDeviceDiscovered,
		recon.TopicDeviceUpdated,
		recon.TopicDeviceLost,
		"pulse.alert.triggered",
		"pulse.alert.resolved",
	}
	for _, topic := range expected {
		if !topics[topic] {
			t.Errorf("missing subscription for topic %q", topic)
		}
	}
}

func TestMqttTopicFromEvent_MapsCorrectly(t *testing.T) {
	m := &Module{cfg: Config{TopicPrefix: "subnetree"}}

	tests := []struct {
		eventTopic string
		want       string
	}{
		{recon.TopicDeviceDiscovered, "subnetree/device/discovered"},
		{recon.TopicDeviceUpdated, "subnetree/device/updated"},
		{recon.TopicDeviceLost, "subnetree/device/lost"},
		{"pulse.alert.triggered", "subnetree/alert/triggered"},
		{"pulse.alert.resolved", "subnetree/alert/resolved"},
		{"unknown.topic", "subnetree/unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.eventTopic, func(t *testing.T) {
			got := m.mqttTopicFromEvent(tt.eventTopic)
			if got != tt.want {
				t.Errorf("mqttTopicFromEvent(%q) = %q, want %q", tt.eventTopic, got, tt.want)
			}
		})
	}
}

func TestPublishEvent_NoOpWhenClientNil(t *testing.T) {
	m := &Module{
		logger: zap.NewNop(),
		cfg:    DefaultConfig(),
	}

	// client is nil -- should not panic.
	m.publishEvent(context.Background(), plugin.Event{
		Topic:     recon.TopicDeviceDiscovered,
		Source:    "recon",
		Timestamp: time.Now(),
		Payload:   map[string]string{"ip": "192.168.1.1"},
	})
}

func TestStart_NoOpWithEmptyBrokerURL(t *testing.T) {
	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{Logger: zap.NewNop()}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// BrokerURL is empty by default -- Start should return nil without attempting connection.
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}

	// Client should remain nil when no broker is configured.
	if m.client != nil {
		t.Error("client should be nil when no broker URL is configured")
	}
}

func TestHealth_NoBrokerConfigured(t *testing.T) {
	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{Logger: zap.NewNop()}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	status := m.Health(context.Background())
	if status.Status != "healthy" {
		t.Errorf("Health().Status = %q, want healthy", status.Status)
	}
	if status.Message != "no broker configured (no-op mode)" {
		t.Errorf("Health().Message = %q, want 'no broker configured (no-op mode)'", status.Message)
	}
}

func TestHealth_DegradedWhenNotConnected(t *testing.T) {
	m := &Module{
		logger: zap.NewNop(),
		cfg:    Config{BrokerURL: "tcp://localhost:1883"},
		// client is nil -- simulates "configured but not connected"
	}

	status := m.Health(context.Background())
	if status.Status != "degraded" {
		t.Errorf("Health().Status = %q, want degraded", status.Status)
	}
}

func TestMqttTopicFromEvent_CustomPrefix(t *testing.T) {
	m := &Module{cfg: Config{TopicPrefix: "homelab/net"}}

	got := m.mqttTopicFromEvent(recon.TopicDeviceDiscovered)
	want := "homelab/net/device/discovered"
	if got != want {
		t.Errorf("mqttTopicFromEvent with custom prefix = %q, want %q", got, want)
	}
}
