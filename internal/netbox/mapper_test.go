package netbox

import (
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
)

func TestSlugFromName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Hello World", "hello-world"},
		{"Dell Inc.", "dell-inc"},
		{"  Spaces  ", "spaces"},
		{"Under_Score", "under-score"},
		{"UPPER CASE", "upper-case"},
		{"special!@#chars", "specialchars"},
		{"already-slug", "already-slug"},
		{"  ", "unknown"},
		{"", "unknown"},
		{"access_point", "access-point"},
		{"My NAS Device", "my-nas-device"},
		{"---leading-trailing---", "leading-trailing"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SlugFromName(tc.name)
			if got != tc.want {
				t.Errorf("SlugFromName(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestDeviceToNetBoxRequest(t *testing.T) {
	device := &models.Device{
		ID:              "550e8400-e29b-41d4-a716-446655440000",
		Hostname:        "web-server-01",
		IPAddresses:     []string{"192.168.1.10"},
		MACAddress:      "00:1a:2b:3c:4d:5e",
		Status:          models.DeviceStatusOnline,
		DeviceType:      models.DeviceTypeServer,
		OS:              "Ubuntu 22.04",
		Notes:           "Production server",
		DiscoveryMethod: models.DiscoveryICMP,
	}

	req := DeviceToNetBoxRequest(device, 1, 2, 3, 4)

	if req.Name != "web-server-01" {
		t.Errorf("Name = %q, want %q", req.Name, "web-server-01")
	}
	if req.Role != 1 {
		t.Errorf("Role = %d, want 1", req.Role)
	}
	if req.DeviceType != 2 {
		t.Errorf("DeviceType = %d, want 2", req.DeviceType)
	}
	if req.Site != 3 {
		t.Errorf("Site = %d, want 3", req.Site)
	}
	if req.Status != "active" {
		t.Errorf("Status = %q, want %q", req.Status, "active")
	}
	if len(req.Tags) != 1 || req.Tags[0] != 4 {
		t.Errorf("Tags = %v, want [4]", req.Tags)
	}
	if req.CustomFields["subnetree_id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Error("expected subnetree_id custom field")
	}
	if req.CustomFields["subnetree_mac"] != "00:1a:2b:3c:4d:5e" {
		t.Error("expected subnetree_mac custom field")
	}
	if req.CustomFields["subnetree_discovery"] != "icmp" {
		t.Error("expected subnetree_discovery custom field")
	}
}

func TestDeviceToNetBoxRequest_FallbackNames(t *testing.T) {
	t.Run("uses IP when hostname empty", func(t *testing.T) {
		device := &models.Device{
			ID:          "abcdef01-0000-0000-0000-000000000000",
			IPAddresses: []string{"10.0.0.5"},
			Status:      models.DeviceStatusOnline,
		}
		req := DeviceToNetBoxRequest(device, 1, 2, 3, 0)
		if req.Name != "10.0.0.5" {
			t.Errorf("Name = %q, want %q", req.Name, "10.0.0.5")
		}
	})

	t.Run("uses device ID prefix when no hostname or IP", func(t *testing.T) {
		device := &models.Device{
			ID:     "abcdef01-0000-0000-0000-000000000000",
			Status: models.DeviceStatusOffline,
		}
		req := DeviceToNetBoxRequest(device, 1, 2, 3, 0)
		if req.Name != "device-abcdef01" {
			t.Errorf("Name = %q, want %q", req.Name, "device-abcdef01")
		}
	})

	t.Run("no tag when tagID is 0", func(t *testing.T) {
		device := &models.Device{
			ID:       "abcdef01-0000-0000-0000-000000000000",
			Hostname: "test",
			Status:   models.DeviceStatusOnline,
		}
		req := DeviceToNetBoxRequest(device, 1, 2, 3, 0)
		if req.Tags != nil {
			t.Errorf("Tags = %v, want nil", req.Tags)
		}
	})
}
