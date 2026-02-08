package pulse

import (
	"context"
	"testing"

	"github.com/HerbHall/subnetree/internal/config"
	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"github.com/HerbHall/subnetree/pkg/roles"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func TestPluginContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func TestInit_WithConfig(t *testing.T) {
	v := viper.New()
	v.Set("check_interval", "10s")
	v.Set("ping_count", 5)

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	err = m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Config: config.New(v),
		Store:  db,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if m.cfg.CheckInterval.Seconds() != 10 {
		t.Errorf("cfg.CheckInterval = %v, want 10s", m.cfg.CheckInterval)
	}
	if m.cfg.PingCount != 5 {
		t.Errorf("cfg.PingCount = %d, want 5", m.cfg.PingCount)
	}
}

func TestInit_NilConfig(t *testing.T) {
	m := New()
	err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("Init() with nil config error = %v", err)
	}

	defaults := DefaultConfig()
	if m.cfg.CheckInterval != defaults.CheckInterval {
		t.Errorf("cfg.CheckInterval = %v, want default %v", m.cfg.CheckInterval, defaults.CheckInterval)
	}
	if m.cfg.PingCount != defaults.PingCount {
		t.Errorf("cfg.PingCount = %d, want default %d", m.cfg.PingCount, defaults.PingCount)
	}
	if m.cfg.MaxWorkers != defaults.MaxWorkers {
		t.Errorf("cfg.MaxWorkers = %d, want default %d", m.cfg.MaxWorkers, defaults.MaxWorkers)
	}
}

func TestInfo_HasCorrectRoles(t *testing.T) {
	m := New()
	info := m.Info()

	if info.Name != "pulse" {
		t.Errorf("Info().Name = %q, want %q", info.Name, "pulse")
	}
	if info.Required {
		t.Error("Info().Required = true, want false")
	}

	found := false
	for _, r := range info.Roles {
		if r == roles.RoleMonitoring {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Info().Roles = %v, want to contain %q", info.Roles, roles.RoleMonitoring)
	}
}

func TestHealth_ReportsStatus(t *testing.T) {
	m := New()
	err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	status := m.Health(context.Background())

	if status.Status != "degraded" {
		t.Errorf("Health().Status = %q, want %q (store is nil)", status.Status, "degraded")
	}
}

func TestHealth_WithStore(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	err = m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Store:  db,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	err = m.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = m.Stop(context.Background()) })

	status := m.Health(context.Background())

	if status.Status != "healthy" {
		t.Errorf("Health().Status = %q, want %q", status.Status, "healthy")
	}
	if v, ok := status.Details["scheduler"]; !ok || v != "running" {
		t.Errorf("Health().Details[\"scheduler\"] = %q, want %q", v, "running")
	}
	if v, ok := status.Details["store"]; !ok || v != "connected" {
		t.Errorf("Health().Details[\"store\"] = %q, want %q", v, "connected")
	}
}

func TestSubscriptions_ReturnsTopics(t *testing.T) {
	m := New()

	subs := m.Subscriptions()
	if len(subs) != 1 {
		t.Fatalf("Subscriptions() returned %d, want 1", len(subs))
	}

	if subs[0].Topic != TopicDeviceDiscovered {
		t.Errorf("Subscriptions()[0].Topic = %q, want %q", subs[0].Topic, TopicDeviceDiscovered)
	}
	if subs[0].Handler == nil {
		t.Error("Subscriptions()[0].Handler is nil, want non-nil")
	}
}
