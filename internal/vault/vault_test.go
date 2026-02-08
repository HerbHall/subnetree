package vault

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// Compile-time interface guards (repeated in tests to catch regressions).
var (
	_ plugin.Plugin            = (*Module)(nil)
	_ plugin.HTTPProvider      = (*Module)(nil)
	_ plugin.HealthChecker     = (*Module)(nil)
	_ roles.CredentialProvider = (*Module)(nil)
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
		t.Error("Required should be false for vault")
	}
	if info.APIVersion != plugin.APIVersionCurrent {
		t.Errorf("APIVersion = %d, want %d", info.APIVersion, plugin.APIVersionCurrent)
	}

	wantRoles := []string{roles.RoleCredentialStore}
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
	// Prevent stdin blocking in tests.
	m.readPassphrase = func() (string, error) { return "", fmt.Errorf("no terminal") }

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
	m.readPassphrase = func() (string, error) { return "test-passphrase", nil }

	deps := plugin.Dependencies{
		Logger: zap.NewNop(),
		Store:  db,
	}

	if err := m.Init(context.Background(), deps); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// After Start with passphrase, vault should be unsealed.
	if m.km.IsSealed() {
		t.Error("vault should be unsealed after Start with passphrase")
	}

	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// After Stop, vault should be sealed (KEK zeroed).
	if !m.km.IsSealed() {
		t.Error("vault should be sealed after Stop")
	}
}

func TestRoutes(t *testing.T) {
	m := New()
	m.readPassphrase = func() (string, error) { return "", fmt.Errorf("no terminal") }

	deps := plugin.Dependencies{
		Logger: zap.NewNop(),
	}
	if err := m.Init(context.Background(), deps); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	routes := m.Routes()

	want := map[string]string{
		"GET /credentials":                    "",
		"POST /credentials":                   "",
		"GET /credentials/{id}":               "",
		"PUT /credentials/{id}":               "",
		"DELETE /credentials/{id}":            "",
		"GET /credentials/{id}/data":          "",
		"GET /credentials/device/{device_id}": "",
		"POST /rotate-keys":                   "",
		"POST /seal":                          "",
		"POST /unseal":                        "",
		"GET /status":                         "",
		"GET /audit":                          "",
		"GET /audit/{credential_id}":          "",
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

func TestHealth_WithStore_Unsealed(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	m.readPassphrase = func() (string, error) { return "pass", nil }

	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(), Store: db,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = m.Stop(context.Background()) })

	status := m.Health(context.Background())
	if status.Status != "healthy" {
		t.Errorf("Health().Status = %q, want %q", status.Status, "healthy")
	}
	if status.Details["store"] != "connected" {
		t.Errorf("Details[store] = %q, want %q", status.Details["store"], "connected")
	}
	if status.Details["vault"] != "unsealed" {
		t.Errorf("Details[vault] = %q, want %q", status.Details["vault"], "unsealed")
	}
}

func TestHealth_Sealed(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	m.readPassphrase = func() (string, error) { return "", fmt.Errorf("no terminal") }

	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(), Store: db,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	status := m.Health(context.Background())
	if status.Status != "degraded" {
		t.Errorf("Health().Status = %q, want %q", status.Status, "degraded")
	}
	if status.Details["vault"] != "sealed" {
		t.Errorf("Details[vault] = %q, want %q", status.Details["vault"], "sealed")
	}
}

func TestHealth_NilStore(t *testing.T) {
	m := New()
	m.readPassphrase = func() (string, error) { return "", fmt.Errorf("no terminal") }

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

// --- CredentialProvider Tests ---

func TestCredentialProvider_Credential_Found(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	m.readPassphrase = func() (string, error) { return "", fmt.Errorf("no terminal") }

	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(), Store: db,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Insert test credential directly via store.
	now := time.Now().UTC()
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "cred-1", Name: "Test SSH", Type: CredTypeSSHPassword,
		DeviceID: "dev-1", EncryptedData: []byte("enc"),
		CreatedAt: now, UpdatedAt: now,
	})

	cred, err := m.Credential(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("Credential() error = %v", err)
	}
	if cred == nil {
		t.Fatal("expected non-nil credential")
	}
	if cred.ID != "cred-1" {
		t.Errorf("ID = %q, want %q", cred.ID, "cred-1")
	}
	if cred.Name != "Test SSH" {
		t.Errorf("Name = %q, want %q", cred.Name, "Test SSH")
	}
	if cred.Type != CredTypeSSHPassword {
		t.Errorf("Type = %q, want %q", cred.Type, CredTypeSSHPassword)
	}
	if cred.DeviceID != "dev-1" {
		t.Errorf("DeviceID = %q, want %q", cred.DeviceID, "dev-1")
	}
}

func TestCredentialProvider_Credential_NotFound(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	m.readPassphrase = func() (string, error) { return "", fmt.Errorf("no terminal") }

	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(), Store: db,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	cred, err := m.Credential(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Credential() error = %v", err)
	}
	if cred != nil {
		t.Error("expected nil for nonexistent credential")
	}
}

func TestCredentialProvider_CredentialsForDevice(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	m.readPassphrase = func() (string, error) { return "", fmt.Errorf("no terminal") }

	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(), Store: db,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	now := time.Now().UTC()
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "cred-1", Name: "SSH", Type: CredTypeSSHPassword, DeviceID: "dev-1",
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "cred-2", Name: "SNMP", Type: CredTypeSNMPv2c, DeviceID: "dev-1",
		EncryptedData: []byte("b"), CreatedAt: now, UpdatedAt: now,
	})
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "cred-3", Name: "Other", Type: CredTypeAPIKey, DeviceID: "dev-2",
		EncryptedData: []byte("c"), CreatedAt: now, UpdatedAt: now,
	})

	creds, err := m.CredentialsForDevice(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("CredentialsForDevice() error = %v", err)
	}
	if len(creds) != 2 {
		t.Fatalf("len = %d, want 2", len(creds))
	}
}

func TestCredentialProvider_NilStore(t *testing.T) {
	m := New()
	m.readPassphrase = func() (string, error) { return "", fmt.Errorf("no terminal") }

	if err := m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	_, err := m.Credential(context.Background(), "cred-1")
	if err == nil {
		t.Error("Credential() with nil store should return error")
	}

	_, err = m.CredentialsForDevice(context.Background(), "dev-1")
	if err == nil {
		t.Error("CredentialsForDevice() with nil store should return error")
	}
}

