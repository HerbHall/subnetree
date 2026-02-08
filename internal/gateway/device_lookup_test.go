package gateway

import (
	"context"
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
)

// mockPluginResolver implements plugin.PluginResolver for testing.
type mockPluginResolver struct {
	byName map[string]plugin.Plugin
	byRole map[string][]plugin.Plugin
}

func (r *mockPluginResolver) Resolve(name string) (plugin.Plugin, bool) {
	p, ok := r.byName[name]
	return p, ok
}

func (r *mockPluginResolver) ResolveByRole(role string) []plugin.Plugin {
	return r.byRole[role]
}

// mockDiscoveryPlugin implements plugin.Plugin and DeviceLookup.
type mockDiscoveryPlugin struct {
	plugin.Plugin
	devices map[string]*models.Device
}

func (p *mockDiscoveryPlugin) DeviceByID(_ context.Context, id string) (*models.Device, error) {
	d, ok := p.devices[id]
	if !ok {
		return nil, nil
	}
	return d, nil
}

// mockNonLookupPlugin implements plugin.Plugin but NOT DeviceLookup.
type mockNonLookupPlugin struct {
	plugin.Plugin
}

func TestResolveDeviceLookup_NilResolver(t *testing.T) {
	_, err := resolveDeviceLookup(nil)
	if err == nil {
		t.Error("expected error for nil resolver")
	}
}

func TestResolveDeviceLookup_NoDiscoveryPlugins(t *testing.T) {
	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{},
	}

	_, err := resolveDeviceLookup(resolver)
	if err == nil {
		t.Error("expected error when no discovery plugins exist")
	}
}

func TestResolveDeviceLookup_Success(t *testing.T) {
	discovery := &mockDiscoveryPlugin{
		devices: map[string]*models.Device{
			"dev-1": {ID: "dev-1", Hostname: "server-01", IPAddresses: []string{"192.168.1.10"}},
		},
	}

	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{
			"discovery": {discovery},
		},
	}

	dl, err := resolveDeviceLookup(resolver)
	if err != nil {
		t.Fatalf("resolveDeviceLookup() error = %v", err)
	}

	device, err := dl.DeviceByID(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("DeviceByID() error = %v", err)
	}
	if device == nil {
		t.Fatal("DeviceByID() returned nil device")
	}
	if device.IPAddresses[0] != "192.168.1.10" {
		t.Errorf("IP = %q, want %q", device.IPAddresses[0], "192.168.1.10")
	}
}

func TestResolveDeviceLookup_WrongType(t *testing.T) {
	nonLookup := &mockNonLookupPlugin{}

	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{
			"discovery": {nonLookup},
		},
	}

	_, err := resolveDeviceLookup(resolver)
	if err == nil {
		t.Error("expected error when discovery plugin doesn't implement DeviceLookup")
	}
}

func TestResolveDeviceLookup_FirstMatchWins(t *testing.T) {
	nonLookup := &mockNonLookupPlugin{}
	discovery := &mockDiscoveryPlugin{
		devices: map[string]*models.Device{
			"dev-1": {ID: "dev-1"},
		},
	}

	resolver := &mockPluginResolver{
		byRole: map[string][]plugin.Plugin{
			"discovery": {nonLookup, discovery},
		},
	}

	dl, err := resolveDeviceLookup(resolver)
	if err != nil {
		t.Fatalf("resolveDeviceLookup() error = %v", err)
	}
	if dl == nil {
		t.Fatal("expected non-nil DeviceLookup")
	}
}
