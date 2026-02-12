package docs

import (
	"context"
	"testing"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"go.uber.org/zap"
)

func TestPluginContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func TestNew(t *testing.T) {
	m := New()
	if m == nil {
		t.Fatal("New() returned nil")
	}
}

func TestInfo(t *testing.T) {
	m := New()
	info := m.Info()

	if info.Name != "docs" {
		t.Errorf("Info().Name = %q, want %q", info.Name, "docs")
	}
	if info.Version != "0.1.0" {
		t.Errorf("Info().Version = %q, want %q", info.Version, "0.1.0")
	}
	if info.Description == "" {
		t.Error("Info().Description is empty, want non-empty")
	}
	if info.Required {
		t.Error("Info().Required = true, want false")
	}
	if info.APIVersion < plugin.APIVersionMin {
		t.Errorf("Info().APIVersion = %d, below minimum %d", info.APIVersion, plugin.APIVersionMin)
	}
}

func TestInit_NilStore(t *testing.T) {
	m := New()
	err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("Init() with nil store error = %v", err)
	}

	if m.store != nil {
		t.Error("store should be nil when deps.Store is nil")
	}

	defaults := DefaultConfig()
	if m.cfg.RetentionPeriod != defaults.RetentionPeriod {
		t.Errorf("cfg.RetentionPeriod = %v, want default %v", m.cfg.RetentionPeriod, defaults.RetentionPeriod)
	}
	if m.cfg.MaxSnapshotsPerApp != defaults.MaxSnapshotsPerApp {
		t.Errorf("cfg.MaxSnapshotsPerApp = %d, want default %d", m.cfg.MaxSnapshotsPerApp, defaults.MaxSnapshotsPerApp)
	}
	if m.cfg.CollectInterval != defaults.CollectInterval {
		t.Errorf("cfg.CollectInterval = %v, want default %v", m.cfg.CollectInterval, defaults.CollectInterval)
	}
}

func TestInit_WithStore(t *testing.T) {
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

	if m.store == nil {
		t.Error("store should be non-nil when deps.Store is provided")
	}
}

func TestHealth_NoStore(t *testing.T) {
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
	if v, ok := status.Details["store"]; !ok || v != "unavailable" {
		t.Errorf("Health().Details[\"store\"] = %q, want %q", v, "unavailable")
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

	status := m.Health(context.Background())

	if status.Status != "healthy" {
		t.Errorf("Health().Status = %q, want %q", status.Status, "healthy")
	}
	if v, ok := status.Details["store"]; !ok || v != "connected" {
		t.Errorf("Health().Details[\"store\"] = %q, want %q", v, "connected")
	}
}

func TestRoutes(t *testing.T) {
	m := New()
	routes := m.Routes()

	if len(routes) != 5 {
		t.Fatalf("Routes() returned %d routes, want 5", len(routes))
	}

	expected := []struct {
		Method string
		Path   string
	}{
		{"GET", "/applications"},
		{"GET", "/applications/{id}"},
		{"GET", "/snapshots"},
		{"GET", "/snapshots/{id}"},
		{"POST", "/snapshots"},
	}

	for i, want := range expected {
		if routes[i].Method != want.Method {
			t.Errorf("routes[%d].Method = %q, want %q", i, routes[i].Method, want.Method)
		}
		if routes[i].Path != want.Path {
			t.Errorf("routes[%d].Path = %q, want %q", i, routes[i].Path, want.Path)
		}
		if routes[i].Handler == nil {
			t.Errorf("routes[%d].Handler is nil, want non-nil", i)
		}
	}
}

func TestStartStop(t *testing.T) {
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

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}
