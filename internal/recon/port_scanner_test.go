package recon

import (
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

func TestClassifyByPorts(t *testing.T) {
	tests := []struct {
		name      string
		openPorts []int
		want      models.DeviceType
	}{
		// Specific fingerprints.
		{
			name:      "UniFi device (SSH + HTTP + HTTPS alt)",
			openPorts: []int{22, 80, 8443},
			want:      models.DeviceTypeSwitch,
		},
		{
			name:      "MikroTik router (HTTP + Winbox)",
			openPorts: []int{80, 8291},
			want:      models.DeviceTypeRouter,
		},
		{
			name:      "Cisco-like switch (SSH + Telnet + HTTP)",
			openPorts: []int{22, 23, 80},
			want:      models.DeviceTypeSwitch,
		},
		// Managed switch with optional ports.
		{
			name:      "managed switch with SNMP (SSH + HTTP + SNMP)",
			openPorts: []int{22, 80, 161},
			want:      models.DeviceTypeSwitch,
		},
		{
			name:      "managed switch with HTTPS (SSH + HTTP + HTTPS)",
			openPorts: []int{22, 80, 443},
			want:      models.DeviceTypeSwitch,
		},
		// Generic managed switch.
		{
			name:      "generic managed switch (SSH + HTTP only)",
			openPorts: []int{22, 80},
			want:      models.DeviceTypeSwitch,
		},
		// Consumer router.
		{
			name:      "consumer router (HTTP + HTTPS)",
			openPorts: []int{80, 443},
			want:      models.DeviceTypeRouter,
		},
		// SSH-managed with SNMP.
		{
			name:      "SSH-managed device with SNMP",
			openPorts: []int{22, 161},
			want:      models.DeviceTypeSwitch,
		},
		// No match cases.
		{
			name:      "empty ports returns unknown",
			openPorts: []int{},
			want:      models.DeviceTypeUnknown,
		},
		{
			name:      "nil ports returns unknown",
			openPorts: nil,
			want:      models.DeviceTypeUnknown,
		},
		{
			name:      "single HTTP port too generic",
			openPorts: []int{80},
			want:      models.DeviceTypeUnknown,
		},
		{
			name:      "RDP and VNC are not infrastructure",
			openPorts: []int{3389, 5900},
			want:      models.DeviceTypeUnknown,
		},
		{
			name:      "SSH alone without SNMP returns unknown",
			openPorts: []int{22},
			want:      models.DeviceTypeUnknown,
		},
		// Full infrastructure device with many ports.
		{
			name:      "fully featured device matches most specific first",
			openPorts: []int{22, 23, 80, 161, 443, 8080, 8443},
			want:      models.DeviceTypeSwitch, // matches UniFi first (22+80+8443)
		},
		// MikroTik with extra ports still matches MikroTik.
		{
			name:      "MikroTik with SSH still matches MikroTik",
			openPorts: []int{22, 80, 8291},
			want:      models.DeviceTypeRouter, // UniFi needs 8443 (missing), MikroTik [80,8291] matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyByPorts(tt.openPorts)
			if got != tt.want {
				t.Errorf("ClassifyByPorts(%v) = %q, want %q", tt.openPorts, got, tt.want)
			}
		})
	}
}

func TestIsInfrastructureOUI(t *testing.T) {
	tests := []struct {
		name       string
		deviceType models.DeviceType
		want       bool
	}{
		{"router is infrastructure", models.DeviceTypeRouter, true},
		{"switch is infrastructure", models.DeviceTypeSwitch, true},
		{"access point is infrastructure", models.DeviceTypeAccessPoint, true},
		{"firewall is infrastructure", models.DeviceTypeFirewall, true},
		{"desktop is not infrastructure", models.DeviceTypeDesktop, false},
		{"server is not infrastructure", models.DeviceTypeServer, false},
		{"unknown is not infrastructure", models.DeviceTypeUnknown, false},
		{"printer is not infrastructure", models.DeviceTypePrinter, false},
		{"nas is not infrastructure", models.DeviceTypeNAS, false},
		{"mobile is not infrastructure", models.DeviceTypeMobile, false},
		{"iot is not infrastructure", models.DeviceTypeIoT, false},
		{"camera is not infrastructure", models.DeviceTypeCamera, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInfrastructureOUI(tt.deviceType)
			if got != tt.want {
				t.Errorf("IsInfrastructureOUI(%q) = %v, want %v", tt.deviceType, got, tt.want)
			}
		})
	}
}

func TestPortScannerCreation(t *testing.T) {
	logger := zap.NewNop()

	t.Run("defaults when zero values provided", func(t *testing.T) {
		ps := NewPortScanner(0, 0, logger)
		if ps.timeout != 2*time.Second {
			t.Errorf("expected default timeout 2s, got %v", ps.timeout)
		}
		if ps.concurrency != 10 {
			t.Errorf("expected default concurrency 10, got %d", ps.concurrency)
		}
	})

	t.Run("defaults when negative values provided", func(t *testing.T) {
		ps := NewPortScanner(-1*time.Second, -5, logger)
		if ps.timeout != 2*time.Second {
			t.Errorf("expected default timeout 2s, got %v", ps.timeout)
		}
		if ps.concurrency != 10 {
			t.Errorf("expected default concurrency 10, got %d", ps.concurrency)
		}
	})

	t.Run("custom values preserved", func(t *testing.T) {
		ps := NewPortScanner(5*time.Second, 20, logger)
		if ps.timeout != 5*time.Second {
			t.Errorf("expected timeout 5s, got %v", ps.timeout)
		}
		if ps.concurrency != 20 {
			t.Errorf("expected concurrency 20, got %d", ps.concurrency)
		}
	})
}

func TestMatchesFingerprint(t *testing.T) {
	tests := []struct {
		name  string
		ports map[int]bool
		fp    portFingerprint
		want  bool
	}{
		{
			name:  "all required ports present",
			ports: map[int]bool{22: true, 80: true},
			fp:    portFingerprint{requiredPorts: []int{22, 80}},
			want:  true,
		},
		{
			name:  "missing required port",
			ports: map[int]bool{22: true},
			fp:    portFingerprint{requiredPorts: []int{22, 80}},
			want:  false,
		},
		{
			name:  "required met with optional present",
			ports: map[int]bool{22: true, 80: true, 161: true},
			fp:    portFingerprint{requiredPorts: []int{22, 80}, optionalPorts: []int{161, 443}},
			want:  true,
		},
		{
			name:  "required met but no optional present",
			ports: map[int]bool{22: true, 80: true},
			fp:    portFingerprint{requiredPorts: []int{22, 80}, optionalPorts: []int{161, 443}},
			want:  false,
		},
		{
			name:  "no required or optional ports",
			ports: map[int]bool{22: true},
			fp:    portFingerprint{},
			want:  true,
		},
		{
			name:  "extra ports do not prevent match",
			ports: map[int]bool{22: true, 80: true, 443: true, 8080: true},
			fp:    portFingerprint{requiredPorts: []int{22, 80}},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesFingerprint(tt.ports, &tt.fp)
			if got != tt.want {
				t.Errorf("matchesFingerprint() = %v, want %v", got, tt.want)
			}
		})
	}
}
