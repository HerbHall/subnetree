package vault

import (
	"context"
	"testing"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"go.uber.org/zap"
)

// Compile-time interface guards (repeated in tests to catch regressions).
var (
	_ plugin.Plugin       = (*Module)(nil)
	_ plugin.HTTPProvider = (*Module)(nil)
)

func TestPluginContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func TestInfo(t *testing.T) {
	m := New()
	info := m.Info()

	if info.Name != "vault" {
		t.Errorf("Name = %q, want %q", info.Name, "vault")
	}
	if info.Version != "0.1.0" {
		t.Errorf("Version = %q, want %q", info.Version, "0.1.0")
	}
	if info.Description == "" {
		t.Error("Description must not be empty")
	}
	if info.Required {
		t.Error("Required should be false for vault stub")
	}
	if info.APIVersion != plugin.APIVersionCurrent {
		t.Errorf("APIVersion = %d, want %d", info.APIVersion, plugin.APIVersionCurrent)
	}

	wantRoles := []string{"credential_store"}
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

	// Init
	if err := m.Init(context.Background(), deps); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Start
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop
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
	if len(routes) == 0 {
		t.Fatal("Routes() returned no routes")
	}

	// Verify expected routes exist.
	want := map[string]string{
		"GET /credentials":  "",
		"POST /credentials": "",
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
