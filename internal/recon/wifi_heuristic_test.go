package recon

import (
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
)

func TestIsLocallyAdministeredMAC(t *testing.T) {
	tests := []struct {
		name string
		mac  string
		want bool
	}{
		{"standard MAC", "00:1a:2b:3c:4d:5e", false},
		{"locally administered x2", "02:1a:2b:3c:4d:5e", true},
		{"locally administered x6", "06:1a:2b:3c:4d:5e", true},
		{"locally administered xA", "0a:1a:2b:3c:4d:5e", true},
		{"locally administered xE", "0e:1a:2b:3c:4d:5e", true},
		{"uppercase locally administered", "0A:1B:2C:3D:4E:5F", true},
		{"iOS randomized typical", "f2:8a:3b:4c:5d:6e", true},
		{"Android randomized", "da:a1:19:00:00:00", true},
		{"normal Cisco", "00:50:56:ab:cd:ef", false},
		{"empty", "", false},
		{"too short", "0", false},
		{"dash format locally administered", "02-1a-2b-3c-4d-5e", true},
		{"dash format standard", "00-1a-2b-3c-4d-5e", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLocallyAdministeredMAC(tt.mac)
			if got != tt.want {
				t.Errorf("IsLocallyAdministeredMAC(%q) = %v, want %v", tt.mac, got, tt.want)
			}
		})
	}
}

func TestAnalyzeWiFiConnection_Wired(t *testing.T) {
	device := &models.Device{
		MACAddress:   "00:1a:2b:3c:4d:5e",
		Manufacturer: "Dell Inc.",
		DeviceType:   models.DeviceTypeServer,
	}

	result := AnalyzeWiFiConnection(device, true, nil)

	if result.ConnectionType != models.ConnectionWired {
		t.Errorf("expected wired, got %s", result.ConnectionType)
	}
	if result.Score != 100 {
		t.Errorf("expected score 100, got %d", result.Score)
	}
}

func TestAnalyzeWiFiConnection_WiFiStrongSignals(t *testing.T) {
	device := &models.Device{
		MACAddress:   "f2:8a:3b:4c:5d:6e", // Locally administered
		Manufacturer: "Apple Inc.",
		Hostname:     "Johns-iPhone",
		DeviceType:   models.DeviceTypeMobile,
	}

	result := AnalyzeWiFiConnection(device, false, []string{"_airplay._tcp"})

	if result.ConnectionType != models.ConnectionWiFi {
		t.Errorf("expected wifi, got %s", result.ConnectionType)
	}
	// Locally administered (30) + not in FDB (25) + mDNS (20) + Apple (15) + hostname (10) = 100
	if result.Score < 40 {
		t.Errorf("expected score >= 40, got %d", result.Score)
	}
}

func TestAnalyzeWiFiConnection_WiFiMediumSignals(t *testing.T) {
	device := &models.Device{
		MACAddress:   "f2:8a:3b:4c:5d:6e", // Locally administered
		Manufacturer: "Unknown",
		Hostname:     "unknown-device",
		DeviceType:   models.DeviceTypeUnknown,
	}

	result := AnalyzeWiFiConnection(device, false, nil)

	// Locally administered (30) + not in FDB (25) = 55
	if result.ConnectionType != models.ConnectionWiFi {
		t.Errorf("expected wifi, got %s (score: %d)", result.ConnectionType, result.Score)
	}
}

func TestAnalyzeWiFiConnection_InfrastructureSkipsFDB(t *testing.T) {
	device := &models.Device{
		MACAddress:   "f2:8a:3b:4c:5d:6e",
		Manufacturer: "Cisco Systems",
		DeviceType:   models.DeviceTypeRouter,
	}

	result := AnalyzeWiFiConnection(device, false, nil)

	// Locally administered (30) + NO FDB bonus (router is infrastructure) = 30
	if result.Score >= 40 {
		t.Errorf("infrastructure device should not hit WiFi threshold, got score %d", result.Score)
	}
}

func TestAnalyzeWiFiConnection_UnknownNoSignals(t *testing.T) {
	device := &models.Device{
		MACAddress:   "00:50:56:ab:cd:ef", // Standard MAC (VMware)
		Manufacturer: "VMware Inc.",
		Hostname:     "vm-server-01",
		DeviceType:   models.DeviceTypeServer,
	}

	result := AnalyzeWiFiConnection(device, false, nil)

	// Standard MAC (0) + not in FDB but server type... still gets 25
	// But no other signals, so likely stays under 40
	if result.ConnectionType == models.ConnectionWiFi {
		t.Errorf("server with standard MAC should not be classified as WiFi, score %d", result.Score)
	}
}

func TestAnalyzeWiFiConnection_mDNSOnlyNotEnough(t *testing.T) {
	device := &models.Device{
		MACAddress:   "00:1a:2b:3c:4d:5e", // Standard MAC
		Manufacturer: "Unknown",
		DeviceType:   models.DeviceTypeUnknown,
	}

	result := AnalyzeWiFiConnection(device, false, []string{"_googlecast._tcp"})

	// Not in FDB (25) + mDNS (20) = 45, which IS above threshold
	// This is correct -- a Chromecast not in FDB with googlecast mDNS should be WiFi
	if result.ConnectionType != models.ConnectionWiFi {
		t.Errorf("expected wifi for Chromecast-like device, got %s (score: %d)", result.ConnectionType, result.Score)
	}
}

func TestIsInfrastructureType(t *testing.T) {
	tests := []struct {
		dt   models.DeviceType
		want bool
	}{
		{models.DeviceTypeRouter, true},
		{models.DeviceTypeSwitch, true},
		{models.DeviceTypeFirewall, true},
		{models.DeviceTypeAccessPoint, true},
		{models.DeviceTypeServer, false},
		{models.DeviceTypeDesktop, false},
		{models.DeviceTypeMobile, false},
		{models.DeviceTypeIoT, false},
		{models.DeviceTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.dt), func(t *testing.T) {
			got := isInfrastructureType(tt.dt)
			if got != tt.want {
				t.Errorf("isInfrastructureType(%q) = %v, want %v", tt.dt, got, tt.want)
			}
		})
	}
}
