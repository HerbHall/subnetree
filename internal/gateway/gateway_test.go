package gateway

import (
	"context"
	"testing"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// Compile-time interface guards (repeated in tests to catch regressions).
var (
	_ plugin.Plugin              = (*Module)(nil)
	_ plugin.HTTPProvider        = (*Module)(nil)
	_ plugin.HealthChecker       = (*Module)(nil)
	_ roles.RemoteAccessProvider = (*Module)(nil)
)

func TestPluginContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func TestInfo(t *testing.T) {
	m := New()
	info := m.Info()

	if info.Name != "gateway" {
		t.Errorf("Name = %q, want %q", info.Name, "gateway")
	}
	if info.Version != "0.1.0" {
		t.Errorf("Version = %q, want %q", info.Version, "0.1.0")
	}
	if info.Description == "" {
		t.Error("Description must not be empty")
	}
	if info.Required {
		t.Error("Required should be false for gateway")
	}
	if info.APIVersion != plugin.APIVersionCurrent {
		t.Errorf("APIVersion = %d, want %d", info.APIVersion, plugin.APIVersionCurrent)
	}

	wantRoles := []string{roles.RoleRemoteAccess}
	if len(info.Roles) != len(wantRoles) {
		t.Fatalf("Roles = %v, want %v", info.Roles, wantRoles)
	}
	for i, r := range info.Roles {
		if r != wantRoles[i] {
			t.Errorf("Roles[%d] = %q, want %q", i, r, wantRoles[i])
		}
	}
}

func TestLifecycle(t *testing.T) {
	m := New()

	deps := plugin.Dependencies{
		Logger: zap.NewNop(),
	}

	if err := m.Init(context.Background(), deps); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestLifecycle_WithStore(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()

	deps := plugin.Dependencies{
		Logger: zap.NewNop(),
		Store:  db,
	}

	if err := m.Init(context.Background(), deps); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if m.store == nil {
		t.Error("store should be initialized after Init with Store dependency")
	}
	if m.sessions == nil {
		t.Error("sessions manager should be initialized after Init")
	}

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestRoutes(t *testing.T) {
	m := New()

	deps := plugin.Dependencies{
		Logger: zap.NewNop(),
	}
	if err := m.Init(context.Background(), deps); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	routes := m.Routes()

	want := map[string]string{
		"GET /sessions":                           "",
		"GET /sessions/{id}":                      "",
		"DELETE /sessions/{id}":                   "",
		"GET /status":                             "",
		"GET /audit":                              "",
		"POST /proxy/{device_id}":                 "",
		"GET /proxy/s/{session_id}/{path...}":     "",
		"POST /proxy/s/{session_id}/{path...}":    "",
		"PUT /proxy/s/{session_id}/{path...}":     "",
		"DELETE /proxy/s/{session_id}/{path...}":  "",
		"PATCH /proxy/s/{session_id}/{path...}":   "",
	}

	if len(routes) != len(want) {
		t.Fatalf("Routes() returned %d routes, want %d", len(routes), len(want))
	}
	for _, r := range routes {
		key := r.Method + " " + r.Path
		if _, ok := want[key]; !ok {
			t.Errorf("unexpected route: %s", key)
		}
		delete(want, key)
		if r.Handler == nil {
			t.Errorf("route %s has nil handler", key)
		}
	}
	for key := range want {
		t.Errorf("missing expected route: %s", key)
	}
}

// --- Health Tests ---

func TestHealth_WithStore(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(), Store: db,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	status := m.Health(context.Background())
	if status.Status != "healthy" {
		t.Errorf("Health().Status = %q, want %q", status.Status, "healthy")
	}
	if status.Details["store"] != "connected" {
		t.Errorf("Details[store] = %q, want %q", status.Details["store"], "connected")
	}
	if status.Details["active_sessions"] != "0" {
		t.Errorf("Details[active_sessions] = %q, want %q", status.Details["active_sessions"], "0")
	}
}

func TestHealth_NilStore(t *testing.T) {
	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	status := m.Health(context.Background())
	if status.Status != "degraded" {
		t.Errorf("Health().Status = %q, want %q", status.Status, "degraded")
	}
	if status.Details["store"] != "unavailable" {
		t.Errorf("Details[store] = %q, want %q", status.Details["store"], "unavailable")
	}
}

// --- RemoteAccessProvider Tests ---

func TestAvailable_WithCapacity(t *testing.T) {
	m := New()
	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	available, err := m.Available(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("Available() error = %v", err)
	}
	if !available {
		t.Error("Available() should return true when sessions are below max")
	}
}

func TestAvailable_NilSessions(t *testing.T) {
	m := &Module{
		logger: zap.NewNop(),
		cfg:    DefaultConfig(),
	}

	available, err := m.Available(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("Available() error = %v", err)
	}
	if available {
		t.Error("Available() should return false when sessions manager is nil")
	}
}
