package recon

import (
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAnsibleInventory_Basic(t *testing.T) {
	devices := []models.Device{
		{
			ID:          "dev-1",
			Hostname:    "gateway",
			IPAddresses: []string{"192.168.1.1"},
			DeviceType:  models.DeviceTypeRouter,
			Status:      models.DeviceStatusOnline,
		},
		{
			ID:          "dev-2",
			Hostname:    "webserver",
			IPAddresses: []string{"192.168.1.10"},
			DeviceType:  models.DeviceTypeServer,
			Status:      models.DeviceStatusOnline,
			Category:    "production",
			Tags:        []string{"web", "linux"},
		},
		{
			ID:          "dev-3",
			Hostname:    "nas",
			IPAddresses: []string{"192.168.1.20"},
			DeviceType:  models.DeviceTypeNAS,
			Status:      models.DeviceStatusOnline,
			Category:    "production",
		},
	}

	inv := buildAnsibleInventory(devices)

	// All hosts present
	require.Len(t, inv.All.Hosts, 3)
	assert.Contains(t, inv.All.Hosts, "gateway")
	assert.Contains(t, inv.All.Hosts, "webserver")
	assert.Contains(t, inv.All.Hosts, "nas")

	// Host vars correct
	gw := inv.All.Hosts["gateway"]
	assert.Equal(t, "192.168.1.1", gw["ansible_host"])
	assert.Equal(t, "router", gw["subnetree_device_type"])

	ws := inv.All.Hosts["webserver"]
	assert.Equal(t, "192.168.1.10", ws["ansible_host"])
	assert.Equal(t, []string{"web", "linux"}, ws["subnetree_tags"])

	// Type groups exist
	require.Contains(t, inv.All.Children, "by_type")
	typeGroup := inv.All.Children["by_type"]
	assert.Contains(t, typeGroup.Children, "router")
	assert.Contains(t, typeGroup.Children, "server")
	assert.Contains(t, typeGroup.Children, "nas")

	// Subnet group exists (all in same /24)
	require.Contains(t, inv.All.Children, "by_subnet")
	subnetGroup := inv.All.Children["by_subnet"]
	assert.Len(t, subnetGroup.Children, 1)
	assert.Contains(t, subnetGroup.Children, "192_168_1_0_24")

	// Category group exists
	require.Contains(t, inv.All.Children, "by_category")
	catGroup := inv.All.Children["by_category"]
	assert.Contains(t, catGroup.Children, "production")
	assert.Len(t, catGroup.Children["production"].Hosts, 2) // webserver + nas
}

func TestBuildAnsibleInventory_SkipsNoIPOrHostname(t *testing.T) {
	devices := []models.Device{
		{ID: "no-ip", Hostname: "orphan", DeviceType: models.DeviceTypeServer, Status: models.DeviceStatusOnline},
		{ID: "no-host", IPAddresses: []string{"10.0.0.1"}, DeviceType: models.DeviceTypeServer, Status: models.DeviceStatusOnline},
		{ID: "ok", Hostname: "valid", IPAddresses: []string{"10.0.0.2"}, DeviceType: models.DeviceTypeServer, Status: models.DeviceStatusOnline},
	}

	inv := buildAnsibleInventory(devices)
	assert.Len(t, inv.All.Hosts, 1)
	assert.Contains(t, inv.All.Hosts, "valid")
}

func TestBuildAnsibleInventory_MultipleSubnets(t *testing.T) {
	devices := []models.Device{
		{ID: "d1", Hostname: "host-a", IPAddresses: []string{"10.0.1.10"}, DeviceType: models.DeviceTypeServer, Status: models.DeviceStatusOnline},
		{ID: "d2", Hostname: "host-b", IPAddresses: []string{"10.0.2.10"}, DeviceType: models.DeviceTypeServer, Status: models.DeviceStatusOnline},
	}

	inv := buildAnsibleInventory(devices)
	subnetGroup := inv.All.Children["by_subnet"]
	assert.Len(t, subnetGroup.Children, 2)
	assert.Contains(t, subnetGroup.Children, "10_0_1_0_24")
	assert.Contains(t, subnetGroup.Children, "10_0_2_0_24")
}

func TestSanitizeGroupName(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"server", "server"},
		{"access-point", "access_point"},
		{"my group", "my_group"},
		{"10.0.1.0/24", "10_0_1_0_24"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, sanitizeGroupName(tt.input), "input: %s", tt.input)
	}
}

func TestSubnetKey(t *testing.T) {
	assert.Equal(t, "192_168_1_0_24", subnetKey("192.168.1.100"))
	assert.Equal(t, "10_0_0_0_24", subnetKey("10.0.0.1"))
	assert.Equal(t, "", subnetKey("invalid"))
	assert.Equal(t, "", subnetKey("::1")) // IPv6 not supported
}
