package netbox

import (
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
)

func TestPluginContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func TestMapDeviceRole(t *testing.T) {
	tests := []struct {
		dt   models.DeviceType
		want string
	}{
		{models.DeviceTypeRouter, "Router"},
		{models.DeviceTypeSwitch, "Switch"},
		{models.DeviceTypeFirewall, "Firewall"},
		{models.DeviceTypeAccessPoint, "Access Point"},
		{models.DeviceTypeServer, "Server"},
		{models.DeviceTypeDesktop, "Desktop"},
		{models.DeviceTypeLaptop, "Laptop"},
		{models.DeviceTypeMobile, "Mobile Device"},
		{models.DeviceTypePrinter, "Printer"},
		{models.DeviceTypeIoT, "IoT Device"},
		{models.DeviceTypeNAS, "NAS"},
		{models.DeviceTypePhone, "Phone"},
		{models.DeviceTypeTablet, "Tablet"},
		{models.DeviceTypeCamera, "Camera"},
		{models.DeviceTypeVM, "Virtual Machine"},
		{models.DeviceTypeContainer, "Container"},
		{models.DeviceTypeUnknown, "Unknown"},
	}

	for _, tc := range tests {
		t.Run(string(tc.dt), func(t *testing.T) {
			got := MapDeviceRole(tc.dt)
			if got != tc.want {
				t.Errorf("MapDeviceRole(%q) = %q, want %q", tc.dt, got, tc.want)
			}
		})
	}
}

func TestMapDeviceStatus(t *testing.T) {
	tests := []struct {
		status models.DeviceStatus
		want   string
	}{
		{models.DeviceStatusOnline, "active"},
		{models.DeviceStatusOffline, "offline"},
		{models.DeviceStatusDegraded, "planned"},
		{models.DeviceStatusUnknown, "inventory"},
		{models.DeviceStatus("other"), "inventory"},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			got := MapDeviceStatus(tc.status)
			if got != tc.want {
				t.Errorf("MapDeviceStatus(%q) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestModuleRoutes(t *testing.T) {
	m := New()
	routes := m.Routes()
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}

	expected := []struct {
		method string
		path   string
	}{
		{"POST", "/sync"},
		{"POST", "/sync/{id}"},
		{"GET", "/status"},
	}

	for i, e := range expected {
		if routes[i].Method != e.method || routes[i].Path != e.path {
			t.Errorf("route[%d] = %s %s, want %s %s",
				i, routes[i].Method, routes[i].Path, e.method, e.path)
		}
	}
}
