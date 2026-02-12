package docs

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockCollector is a test double that implements the Collector interface.
type mockCollector struct {
	name      string
	available bool
	apps      []Application
	discErr   error
	configs   map[string]*CollectedConfig
	collectErr error
}

func (m *mockCollector) Name() string        { return m.name }
func (m *mockCollector) Available() bool      { return m.available }

func (m *mockCollector) Discover(_ context.Context) ([]Application, error) {
	if m.discErr != nil {
		return nil, m.discErr
	}
	return m.apps, nil
}

func (m *mockCollector) Collect(_ context.Context, appID string) (*CollectedConfig, error) {
	if m.collectErr != nil {
		return nil, m.collectErr
	}
	cfg, ok := m.configs[appID]
	if !ok {
		return nil, fmt.Errorf("app %s not found", appID)
	}
	return cfg, nil
}

func TestRunCollection_NoCollectors(t *testing.T) {
	m := newTestModule(t)
	result := m.RunCollection(context.Background())

	if result.AppsDiscovered != 0 {
		t.Errorf("AppsDiscovered = %d, want 0", result.AppsDiscovered)
	}
	if result.SnapshotsCreated != 0 {
		t.Errorf("SnapshotsCreated = %d, want 0", result.SnapshotsCreated)
	}
	if len(result.Errors) != 0 {
		t.Errorf("len(Errors) = %d, want 0", len(result.Errors))
	}
}

func TestRunCollection_CollectorNotAvailable(t *testing.T) {
	m := newTestModule(t)
	m.collectors = []Collector{
		&mockCollector{name: "test", available: false},
	}

	result := m.RunCollection(context.Background())

	if result.AppsDiscovered != 0 {
		t.Errorf("AppsDiscovered = %d, want 0", result.AppsDiscovered)
	}
	if result.SnapshotsCreated != 0 {
		t.Errorf("SnapshotsCreated = %d, want 0", result.SnapshotsCreated)
	}
}

func TestRunCollection_NewAppsAndSnapshots(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	m.collectors = []Collector{
		&mockCollector{
			name:      "test",
			available: true,
			apps: []Application{
				{
					ID: "app-001", Name: "nginx", AppType: "docker-container",
					Collector: "test", Status: "active", Metadata: "{}",
					DiscoveredAt: now, UpdatedAt: now,
				},
				{
					ID: "app-002", Name: "postgres", AppType: "docker-container",
					Collector: "test", Status: "active", Metadata: "{}",
					DiscoveredAt: now, UpdatedAt: now,
				},
			},
			configs: map[string]*CollectedConfig{
				"app-001": {Content: `{"image":"nginx:latest"}`, Format: "json"},
				"app-002": {Content: `{"image":"postgres:16"}`, Format: "json"},
			},
		},
	}

	result := m.RunCollection(context.Background())

	if result.AppsDiscovered != 2 {
		t.Errorf("AppsDiscovered = %d, want 2", result.AppsDiscovered)
	}
	if result.SnapshotsCreated != 2 {
		t.Errorf("SnapshotsCreated = %d, want 2", result.SnapshotsCreated)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", result.Errors)
	}

	// Verify the apps were inserted in the store.
	app, err := m.store.GetApplication(context.Background(), "app-001")
	if err != nil {
		t.Fatalf("GetApplication error: %v", err)
	}
	if app == nil {
		t.Fatal("GetApplication returned nil, want app")
	}
	if app.Name != "nginx" {
		t.Errorf("app.Name = %q, want %q", app.Name, "nginx")
	}

	// Verify snapshots exist.
	snap, err := m.store.GetLatestSnapshot(context.Background(), "app-001")
	if err != nil {
		t.Fatalf("GetLatestSnapshot error: %v", err)
	}
	if snap == nil {
		t.Fatal("GetLatestSnapshot returned nil, want snapshot")
	}
	if snap.Content != `{"image":"nginx:latest"}` {
		t.Errorf("snapshot.Content = %q, want %q", snap.Content, `{"image":"nginx:latest"}`)
	}
}

func TestRunCollection_DeduplicateByHash(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	mc := &mockCollector{
		name:      "test",
		available: true,
		apps: []Application{
			{
				ID: "app-001", Name: "nginx", AppType: "docker-container",
				Collector: "test", Status: "active", Metadata: "{}",
				DiscoveredAt: now, UpdatedAt: now,
			},
		},
		configs: map[string]*CollectedConfig{
			"app-001": {Content: `{"same":"content"}`, Format: "json"},
		},
	}
	m.collectors = []Collector{mc}

	// First run: should create snapshot.
	r1 := m.RunCollection(context.Background())
	if r1.SnapshotsCreated != 1 {
		t.Fatalf("first run SnapshotsCreated = %d, want 1", r1.SnapshotsCreated)
	}

	// Second run with identical content: no new snapshot.
	r2 := m.RunCollection(context.Background())
	if r2.SnapshotsCreated != 0 {
		t.Errorf("second run SnapshotsCreated = %d, want 0 (dedup)", r2.SnapshotsCreated)
	}
	if r2.AppsDiscovered != 1 {
		t.Errorf("second run AppsDiscovered = %d, want 1", r2.AppsDiscovered)
	}
}

func TestRunCollection_MultipleCollectors(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	m.collectors = []Collector{
		&mockCollector{
			name:      "docker",
			available: true,
			apps: []Application{
				{
					ID: "docker-001", Name: "nginx", AppType: "docker-container",
					Collector: "docker", Status: "active", Metadata: "{}",
					DiscoveredAt: now, UpdatedAt: now,
				},
			},
			configs: map[string]*CollectedConfig{
				"docker-001": {Content: `{"from":"docker"}`, Format: "json"},
			},
		},
		&mockCollector{
			name:      "systemd",
			available: true,
			apps: []Application{
				{
					ID: "systemd-001", Name: "sshd", AppType: "systemd-service",
					Collector: "systemd", Status: "active", Metadata: "{}",
					DiscoveredAt: now, UpdatedAt: now,
				},
			},
			configs: map[string]*CollectedConfig{
				"systemd-001": {Content: `{"from":"systemd"}`, Format: "json"},
			},
		},
	}

	result := m.RunCollection(context.Background())

	if result.AppsDiscovered != 2 {
		t.Errorf("AppsDiscovered = %d, want 2", result.AppsDiscovered)
	}
	if result.SnapshotsCreated != 2 {
		t.Errorf("SnapshotsCreated = %d, want 2", result.SnapshotsCreated)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", result.Errors)
	}
}

func TestRunCollection_DiscoverError(t *testing.T) {
	m := newTestModule(t)
	m.collectors = []Collector{
		&mockCollector{
			name:      "failing",
			available: true,
			discErr:   fmt.Errorf("connection refused"),
		},
	}

	result := m.RunCollection(context.Background())

	if result.AppsDiscovered != 0 {
		t.Errorf("AppsDiscovered = %d, want 0", result.AppsDiscovered)
	}
	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(result.Errors))
	}
}

func TestRunCollection_CollectError(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	m.collectors = []Collector{
		&mockCollector{
			name:      "failing",
			available: true,
			apps: []Application{
				{
					ID: "app-001", Name: "nginx", AppType: "docker-container",
					Collector: "failing", Status: "active", Metadata: "{}",
					DiscoveredAt: now, UpdatedAt: now,
				},
			},
			collectErr: fmt.Errorf("inspect failed"),
		},
	}

	result := m.RunCollection(context.Background())

	if result.AppsDiscovered != 1 {
		t.Errorf("AppsDiscovered = %d, want 1", result.AppsDiscovered)
	}
	if result.SnapshotsCreated != 0 {
		t.Errorf("SnapshotsCreated = %d, want 0", result.SnapshotsCreated)
	}
	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(result.Errors))
	}
}
