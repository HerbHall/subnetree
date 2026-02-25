package mqtt

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
)

func TestSafeObjectID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple hostname", "web-server-01", "web_server_01"},
		{"UUID", "550e8400-e29b-41d4-a716-446655440000", "550e8400_e29b_41d4_a716_446655440000"},
		{"dots and colons", "00:1a:2b:3c:4d:5e", "00_1a_2b_3c_4d_5e"},
		{"IP address", "192.168.1.1", "192_168_1_1"},
		{"already clean", "mydevice", "mydevice"},
		{"uppercase", "MyDevice", "mydevice"},
		{"leading special chars", "---test", "test"},
		{"trailing special chars", "test---", "test"},
		{"empty string", "", "unknown"},
		{"only special chars", "---", "unknown"},
		{"mixed special", "device@home#1", "device_home_1"},
		{"underscores preserved", "my_device_01", "my_device_01"},
		{"spaces", "my device", "my_device"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeObjectID(tt.input)
			if got != tt.want {
				t.Errorf("SafeObjectID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeviceTypeIcon(t *testing.T) {
	tests := []struct {
		dt   models.DeviceType
		want string
	}{
		{models.DeviceTypeServer, "mdi:server"},
		{models.DeviceTypeDesktop, "mdi:desktop-tower"},
		{models.DeviceTypeLaptop, "mdi:laptop"},
		{models.DeviceTypeMobile, "mdi:cellphone"},
		{models.DeviceTypeRouter, "mdi:router-wireless"},
		{models.DeviceTypeSwitch, "mdi:switch"},
		{models.DeviceTypePrinter, "mdi:printer"},
		{models.DeviceTypeIoT, "mdi:devices"},
		{models.DeviceTypeAccessPoint, "mdi:access-point"},
		{models.DeviceTypeFirewall, "mdi:shield-lock"},
		{models.DeviceTypeNAS, "mdi:nas"},
		{models.DeviceTypePhone, "mdi:phone"},
		{models.DeviceTypeTablet, "mdi:tablet"},
		{models.DeviceTypeCamera, "mdi:cctv"},
		{models.DeviceTypeVM, "mdi:monitor-dashboard"},
		{models.DeviceTypeContainer, "mdi:docker"},
		{models.DeviceTypeUnknown, "mdi:help-network"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dt), func(t *testing.T) {
			got := DeviceTypeIcon(tt.dt)
			if got != tt.want {
				t.Errorf("DeviceTypeIcon(%q) = %q, want %q", tt.dt, got, tt.want)
			}
		})
	}
}

func TestDeviceTypeIcon_UnrecognizedType(t *testing.T) {
	got := DeviceTypeIcon(models.DeviceType("alien_ship"))
	if got != "mdi:help-network" {
		t.Errorf("DeviceTypeIcon(alien_ship) = %q, want mdi:help-network", got)
	}
}

func TestBuildDeviceDiscoveryConfigs(t *testing.T) {
	device := &models.Device{
		ID:           "dev-001",
		Hostname:     "web-server",
		IPAddresses:  []string{"192.168.1.10"},
		MACAddress:   "00:1a:2b:3c:4d:5e",
		Manufacturer: "Dell Inc.",
		DeviceType:   models.DeviceTypeServer,
		OS:           "Ubuntu 22.04",
	}

	configs := BuildDeviceDiscoveryConfigs(device, "subnetree", "homeassistant")
	if len(configs) != 3 {
		t.Fatalf("BuildDeviceDiscoveryConfigs() returned %d configs, want 3", len(configs))
	}

	// Verify all configs are retained.
	for i, cfg := range configs {
		if !cfg.Retain {
			t.Errorf("configs[%d].Retain = false, want true", i)
		}
		if len(cfg.Payload) == 0 {
			t.Errorf("configs[%d].Payload is empty", i)
		}
	}

	// Check online binary_sensor config.
	if !strings.Contains(configs[0].Topic, "binary_sensor") {
		t.Errorf("configs[0].Topic = %q, want binary_sensor in path", configs[0].Topic)
	}
	if !strings.Contains(configs[0].Topic, "online") {
		t.Errorf("configs[0].Topic = %q, want online in path", configs[0].Topic)
	}

	var onlineCfg BinarySensorConfig
	if err := json.Unmarshal(configs[0].Payload, &onlineCfg); err != nil {
		t.Fatalf("unmarshal online config: %v", err)
	}
	if onlineCfg.DeviceClass != "connectivity" {
		t.Errorf("online.DeviceClass = %q, want connectivity", onlineCfg.DeviceClass)
	}
	if onlineCfg.PayloadOn != "ON" {
		t.Errorf("online.PayloadOn = %q, want ON", onlineCfg.PayloadOn)
	}
	if onlineCfg.PayloadOff != "OFF" {
		t.Errorf("online.PayloadOff = %q, want OFF", onlineCfg.PayloadOff)
	}
	if onlineCfg.StateTopic != "subnetree/device/dev-001/online" {
		t.Errorf("online.StateTopic = %q, want subnetree/device/dev-001/online", onlineCfg.StateTopic)
	}
	if onlineCfg.Device.Name != "web-server" {
		t.Errorf("online.Device.Name = %q, want web-server", onlineCfg.Device.Name)
	}
	if onlineCfg.Device.Manufacturer != "Dell Inc." {
		t.Errorf("online.Device.Manufacturer = %q, want Dell Inc.", onlineCfg.Device.Manufacturer)
	}
	if onlineCfg.Device.ViaDevice != "subnetree" {
		t.Errorf("online.Device.ViaDevice = %q, want subnetree", onlineCfg.Device.ViaDevice)
	}

	// Check type sensor config.
	var typeCfg SensorConfig
	if err := json.Unmarshal(configs[1].Payload, &typeCfg); err != nil {
		t.Fatalf("unmarshal type config: %v", err)
	}
	if typeCfg.Icon != "mdi:server" {
		t.Errorf("type.Icon = %q, want mdi:server", typeCfg.Icon)
	}
	if typeCfg.StateTopic != "subnetree/device/dev-001/type" {
		t.Errorf("type.StateTopic = %q, want subnetree/device/dev-001/type", typeCfg.StateTopic)
	}

	// Check IP sensor config.
	var ipCfg SensorConfig
	if err := json.Unmarshal(configs[2].Payload, &ipCfg); err != nil {
		t.Fatalf("unmarshal ip config: %v", err)
	}
	if ipCfg.Icon != "mdi:ip-network" {
		t.Errorf("ip.Icon = %q, want mdi:ip-network", ipCfg.Icon)
	}
	if ipCfg.StateTopic != "subnetree/device/dev-001/ip" {
		t.Errorf("ip.StateTopic = %q, want subnetree/device/dev-001/ip", ipCfg.StateTopic)
	}
}

func TestBuildDeviceDiscoveryConfigs_MinimalDevice(t *testing.T) {
	device := &models.Device{
		ID:          "minimal-dev",
		IPAddresses: []string{"10.0.0.5"},
		DeviceType:  models.DeviceTypeUnknown,
	}

	configs := BuildDeviceDiscoveryConfigs(device, "net", "homeassistant")
	if len(configs) != 3 {
		t.Fatalf("got %d configs, want 3 (online + type + ip)", len(configs))
	}

	// Device name should fall back to IP since hostname is empty.
	var onlineCfg BinarySensorConfig
	if err := json.Unmarshal(configs[0].Payload, &onlineCfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if onlineCfg.Device.Name != "10.0.0.5" {
		t.Errorf("Device.Name = %q, want 10.0.0.5 (fallback to IP)", onlineCfg.Device.Name)
	}
	if onlineCfg.Device.Manufacturer != "" {
		t.Errorf("Device.Manufacturer = %q, want empty", onlineCfg.Device.Manufacturer)
	}
}

func TestBuildDeviceDiscoveryConfigs_NoIPAddress(t *testing.T) {
	device := &models.Device{
		ID:         "no-ip-dev",
		Hostname:   "orphan",
		DeviceType: models.DeviceTypeUnknown,
	}

	configs := BuildDeviceDiscoveryConfigs(device, "subnetree", "homeassistant")
	// Should return 2 configs (online + type) but not IP since no addresses.
	if len(configs) != 2 {
		t.Fatalf("got %d configs, want 2 (no IP sensor when no addresses)", len(configs))
	}
}

func TestBuildDeviceDiscoveryConfigs_NilDevice(t *testing.T) {
	configs := BuildDeviceDiscoveryConfigs(nil, "subnetree", "homeassistant")
	if configs != nil {
		t.Errorf("BuildDeviceDiscoveryConfigs(nil) = %v, want nil", configs)
	}
}

func TestBuildDeviceDiscoveryConfigs_CustomPrefixes(t *testing.T) {
	device := &models.Device{
		ID:          "dev-99",
		Hostname:    "myhost",
		IPAddresses: []string{"10.0.0.1"},
		DeviceType:  models.DeviceTypeRouter,
	}

	configs := BuildDeviceDiscoveryConfigs(device, "mynet/devices", "ha_custom")
	if len(configs) != 3 {
		t.Fatalf("got %d configs, want 3", len(configs))
	}

	// Verify the HA prefix is used in discovery topics.
	if !strings.HasPrefix(configs[0].Topic, "ha_custom/") {
		t.Errorf("discovery topic = %q, want ha_custom/ prefix", configs[0].Topic)
	}

	// Verify the topic prefix is used in state topics.
	var onlineCfg BinarySensorConfig
	if err := json.Unmarshal(configs[0].Payload, &onlineCfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.HasPrefix(onlineCfg.StateTopic, "mynet/devices/") {
		t.Errorf("state topic = %q, want mynet/devices/ prefix", onlineCfg.StateTopic)
	}
}

func TestBuildAlertDiscoveryConfig(t *testing.T) {
	cfg := BuildAlertDiscoveryConfig("dev-001", "web-server", "alert-123", "critical", "subnetree", "homeassistant")

	if cfg.Topic == "" {
		t.Fatal("topic is empty")
	}
	if !cfg.Retain {
		t.Error("Retain = false, want true")
	}
	if !strings.Contains(cfg.Topic, "binary_sensor") {
		t.Errorf("topic = %q, want binary_sensor in path", cfg.Topic)
	}
	if !strings.Contains(cfg.Topic, "alert") {
		t.Errorf("topic = %q, want alert in path", cfg.Topic)
	}

	var bsCfg BinarySensorConfig
	if err := json.Unmarshal(cfg.Payload, &bsCfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if bsCfg.DeviceClass != "problem" {
		t.Errorf("DeviceClass = %q, want problem", bsCfg.DeviceClass)
	}
	if bsCfg.PayloadOn != "triggered" {
		t.Errorf("PayloadOn = %q, want triggered", bsCfg.PayloadOn)
	}
	if bsCfg.PayloadOff != "resolved" {
		t.Errorf("PayloadOff = %q, want resolved", bsCfg.PayloadOff)
	}
	if bsCfg.StateTopic != "subnetree/alert/alert-123/state" {
		t.Errorf("StateTopic = %q, want subnetree/alert/alert-123/state", bsCfg.StateTopic)
	}
	if bsCfg.Icon != "mdi:alert-circle" {
		t.Errorf("Icon = %q, want mdi:alert-circle for critical", bsCfg.Icon)
	}
	if bsCfg.Device.Name != "web-server" {
		t.Errorf("Device.Name = %q, want web-server", bsCfg.Device.Name)
	}
}

func TestBuildAlertDiscoveryConfig_EmptyDeviceName(t *testing.T) {
	cfg := BuildAlertDiscoveryConfig("dev-002", "", "alert-456", "warning", "subnetree", "homeassistant")

	var bsCfg BinarySensorConfig
	if err := json.Unmarshal(cfg.Payload, &bsCfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Should fall back to device ID.
	if bsCfg.Device.Name != "dev-002" {
		t.Errorf("Device.Name = %q, want dev-002 (fallback)", bsCfg.Device.Name)
	}
}

func TestBuildDeviceRemovalConfigs(t *testing.T) {
	configs := BuildDeviceRemovalConfigs("dev-001", "homeassistant")

	if len(configs) != 3 {
		t.Fatalf("BuildDeviceRemovalConfigs() returned %d configs, want 3", len(configs))
	}

	for i, cfg := range configs {
		if !cfg.Retain {
			t.Errorf("configs[%d].Retain = false, want true", i)
		}
		if cfg.Payload != nil {
			t.Errorf("configs[%d].Payload = %v, want nil (empty payload removes entity)", i, cfg.Payload)
		}
		if cfg.Topic == "" {
			t.Errorf("configs[%d].Topic is empty", i)
		}
	}

	// Verify topics match the same pattern as creation.
	if !strings.Contains(configs[0].Topic, "binary_sensor") || !strings.Contains(configs[0].Topic, "online") {
		t.Errorf("configs[0].Topic = %q, want binary_sensor/.../online/config", configs[0].Topic)
	}
	if !strings.Contains(configs[1].Topic, "sensor") || !strings.Contains(configs[1].Topic, "type") {
		t.Errorf("configs[1].Topic = %q, want sensor/.../type/config", configs[1].Topic)
	}
	if !strings.Contains(configs[2].Topic, "sensor") || !strings.Contains(configs[2].Topic, "ip") {
		t.Errorf("configs[2].Topic = %q, want sensor/.../ip/config", configs[2].Topic)
	}
}

func TestAlertSeverityIcon(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"critical", "mdi:alert-circle"},
		{"warning", "mdi:alert"},
		{"info", "mdi:information"},
		{"unknown", "mdi:information"},
		{"", "mdi:information"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := alertSeverityIcon(tt.severity)
			if got != tt.want {
				t.Errorf("alertSeverityIcon(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func TestBuildDeviceDiscoveryConfigs_UniqueIDsAreUnique(t *testing.T) {
	device := &models.Device{
		ID:          "dev-unique",
		Hostname:    "unique-host",
		IPAddresses: []string{"10.0.0.1"},
		DeviceType:  models.DeviceTypeServer,
	}

	configs := BuildDeviceDiscoveryConfigs(device, "subnetree", "homeassistant")

	uniqueIDs := make(map[string]bool)
	for _, cfg := range configs {
		var raw map[string]interface{}
		if err := json.Unmarshal(cfg.Payload, &raw); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		uid, ok := raw["unique_id"].(string)
		if !ok {
			t.Fatal("unique_id missing or not string")
		}
		if uniqueIDs[uid] {
			t.Errorf("duplicate unique_id: %q", uid)
		}
		uniqueIDs[uid] = true
	}
}

func TestBuildDeviceDiscoveryConfigs_TopicFormat(t *testing.T) {
	device := &models.Device{
		ID:          "abc-123",
		Hostname:    "myhost",
		IPAddresses: []string{"10.0.0.1"},
		DeviceType:  models.DeviceTypeRouter,
	}

	configs := BuildDeviceDiscoveryConfigs(device, "subnetree", "homeassistant")

	expectedTopics := []string{
		"homeassistant/binary_sensor/subnetree_abc_123/online/config",
		"homeassistant/sensor/subnetree_abc_123/type/config",
		"homeassistant/sensor/subnetree_abc_123/ip/config",
	}

	for i, want := range expectedTopics {
		if configs[i].Topic != want {
			t.Errorf("configs[%d].Topic = %q, want %q", i, configs[i].Topic, want)
		}
	}
}
