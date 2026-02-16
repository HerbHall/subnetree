package recon

import (
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
)

func TestDetectUnmanagedSwitches(t *testing.T) {
	tests := []struct {
		name       string
		devices    []UnmanagedDeviceInfo
		wantCount  int
		wantIDs    []string // expected device IDs in order
		wantMinCon int      // minimum confidence expected for first candidate
		wantMaxCon int      // maximum confidence expected for first candidate
	}{
		{
			name:      "empty input returns empty",
			devices:   nil,
			wantCount: 0,
		},
		{
			name: "non-infrastructure OUI is not flagged",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:     "dev-1",
					IP:           "192.168.1.10",
					MAC:          "AA:BB:CC:00:00:01",
					Manufacturer: "Samsung",
					DeviceType:   models.DeviceTypeUnknown,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
			},
			wantCount: 0,
		},
		{
			name: "infrastructure OUI with SNMP is not flagged (managed)",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:     "dev-2",
					IP:           "192.168.1.1",
					MAC:          "AA:BB:CC:00:00:02",
					Manufacturer: "Netgear",
					DeviceType:   models.DeviceTypeUnknown,
					HasSNMP:      true,
					HasOpenPorts: false,
				},
			},
			wantCount: 0,
		},
		{
			name: "infrastructure OUI with open ports is not flagged",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:      "dev-3",
					IP:            "192.168.1.2",
					MAC:           "AA:BB:CC:00:00:03",
					Manufacturer:  "TP-Link",
					DeviceType:    models.DeviceTypeUnknown,
					HasSNMP:       false,
					HasOpenPorts:  true,
					OpenPortCount: 2,
				},
			},
			wantCount: 0,
		},
		{
			name: "infrastructure OUI no SNMP no ports type Unknown is flagged",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:     "dev-4",
					IP:           "192.168.1.5",
					MAC:          "AA:BB:CC:00:00:04",
					Manufacturer: "Netgear",
					DeviceType:   models.DeviceTypeUnknown,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
			},
			wantCount:  1,
			wantIDs:    []string{"dev-4"},
			wantMinCon: 15,
			wantMaxCon: 30,
		},
		{
			name: "already classified device is not flagged",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:     "dev-5",
					IP:           "192.168.1.6",
					MAC:          "AA:BB:CC:00:00:05",
					Manufacturer: "Cisco",
					DeviceType:   models.DeviceTypeSwitch,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
			},
			wantCount: 0,
		},
		{
			name: "already classified as Router is not flagged",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:     "dev-6",
					IP:           "192.168.1.7",
					MAC:          "AA:BB:CC:00:00:06",
					Manufacturer: "Netgear",
					DeviceType:   models.DeviceTypeRouter,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
			},
			wantCount: 0,
		},
		{
			name: "multiple candidates sorted by confidence descending",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:     "dev-low",
					IP:           "192.168.1.20",
					MAC:          "AA:BB:CC:00:00:20",
					Manufacturer: "Tenda", // infrastructure but not in OUI classifier rules
					DeviceType:   models.DeviceTypeUnknown,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
				{
					DeviceID:     "dev-high",
					IP:           "192.168.1.10",
					MAC:          "AA:BB:CC:00:00:10",
					Manufacturer: "Cisco", // in OUI classifier as Router => IsInfrastructureOUI = true
					DeviceType:   models.DeviceTypeUnknown,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
			},
			wantCount: 2,
			wantIDs:   []string{"dev-high", "dev-low"},
		},
		{
			name: "device with OpenPortCount > 0 but HasOpenPorts false is not flagged",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:      "dev-ports",
					IP:            "192.168.1.30",
					MAC:           "AA:BB:CC:00:00:30",
					Manufacturer:  "Netgear",
					DeviceType:    models.DeviceTypeUnknown,
					HasSNMP:       false,
					HasOpenPorts:  false,
					OpenPortCount: 3,
				},
			},
			wantCount: 0,
		},
		{
			name: "MAC clustering bonus for same manufacturer",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:     "dev-a",
					IP:           "192.168.1.40",
					MAC:          "AA:BB:CC:00:00:40",
					Manufacturer: "Netgear",
					DeviceType:   models.DeviceTypeUnknown,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
				{
					DeviceID:     "dev-b",
					IP:           "192.168.1.41",
					MAC:          "AA:BB:CC:00:00:41",
					Manufacturer: "Netgear",
					DeviceType:   models.DeviceTypeUnknown,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
			},
			wantCount:  2,
			wantMinCon: 25, // base 20 + OUI infra 5 + cluster 5 = 30 (capped)
			wantMaxCon: 30,
		},
		{
			name: "ghost MAC with no manufacturer",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:     "dev-ghost",
					IP:           "192.168.1.50",
					MAC:          "AA:BB:CC:00:00:50",
					Manufacturer: "",
					DeviceType:   models.DeviceTypeUnknown,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
			},
			// No manufacturer means neither isInfrastructureManufacturer nor
			// IsInfrastructureOUI returns true, so not flagged.
			wantCount: 0,
		},
		{
			name: "mixed: some flagged some not",
			devices: []UnmanagedDeviceInfo{
				{
					DeviceID:     "managed-switch",
					IP:           "192.168.1.1",
					MAC:          "AA:BB:CC:00:00:01",
					Manufacturer: "Cisco",
					DeviceType:   models.DeviceTypeSwitch,
					HasSNMP:      true,
					HasOpenPorts: true,
				},
				{
					DeviceID:     "desktop",
					IP:           "192.168.1.100",
					MAC:          "DD:EE:FF:00:00:01",
					Manufacturer: "Dell",
					DeviceType:   models.DeviceTypeDesktop,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
				{
					DeviceID:     "suspect-switch",
					IP:           "192.168.1.200",
					MAC:          "AA:BB:CC:00:00:C8",
					Manufacturer: "TP-Link",
					DeviceType:   models.DeviceTypeUnknown,
					HasSNMP:      false,
					HasOpenPorts: false,
				},
			},
			wantCount: 1,
			wantIDs:   []string{"suspect-switch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectUnmanagedSwitches(tt.devices)

			if len(got) != tt.wantCount {
				t.Fatalf("DetectUnmanagedSwitches() returned %d candidates, want %d", len(got), tt.wantCount)
			}

			if tt.wantIDs != nil {
				for i, wantID := range tt.wantIDs {
					if i >= len(got) {
						t.Errorf("missing candidate at index %d, want ID %q", i, wantID)
						continue
					}
					if got[i].DeviceID != wantID {
						t.Errorf("candidate[%d].DeviceID = %q, want %q", i, got[i].DeviceID, wantID)
					}
				}
			}

			if tt.wantCount > 0 && tt.wantMinCon > 0 {
				if got[0].Confidence < tt.wantMinCon {
					t.Errorf("candidate[0].Confidence = %d, want >= %d", got[0].Confidence, tt.wantMinCon)
				}
			}
			if tt.wantCount > 0 && tt.wantMaxCon > 0 {
				if got[0].Confidence > tt.wantMaxCon {
					t.Errorf("candidate[0].Confidence = %d, want <= %d", got[0].Confidence, tt.wantMaxCon)
				}
			}

			// Verify all candidates have non-empty reasons.
			for i, c := range got {
				if c.Reason == "" {
					t.Errorf("candidate[%d].Reason is empty", i)
				}
			}
		})
	}
}

func TestDetectUnmanagedSwitches_SortOrder(t *testing.T) {
	// Verify descending confidence, then ascending IP for ties.
	devices := []UnmanagedDeviceInfo{
		{
			DeviceID:     "dev-z",
			IP:           "192.168.1.99",
			MAC:          "AA:BB:CC:00:00:99",
			Manufacturer: "Zyxel", // infrastructure manufacturer, not in OUI classifier
			DeviceType:   models.DeviceTypeUnknown,
		},
		{
			DeviceID:     "dev-c",
			IP:           "192.168.1.10",
			MAC:          "AA:BB:CC:00:00:0A",
			Manufacturer: "Cisco", // infrastructure manufacturer + OUI classifier match
			DeviceType:   models.DeviceTypeUnknown,
		},
		{
			DeviceID:     "dev-n",
			IP:           "192.168.1.50",
			MAC:          "AA:BB:CC:00:00:32",
			Manufacturer: "Cisco", // same as dev-c: cluster bonus applies
			DeviceType:   models.DeviceTypeUnknown,
		},
	}

	got := DetectUnmanagedSwitches(devices)
	if len(got) != 3 {
		t.Fatalf("expected 3 candidates, got %d", len(got))
	}

	// Cisco devices get OUI infra bonus (25 base) + cluster bonus (30 capped).
	// Zyxel gets base 20 only (no OUI classifier match, no cluster).
	// So Cisco devices should come first (higher confidence), then Zyxel.
	for i := 1; i < len(got); i++ {
		if got[i].Confidence > got[i-1].Confidence {
			t.Errorf("candidates not sorted by confidence desc: [%d]=%d > [%d]=%d",
				i, got[i].Confidence, i-1, got[i-1].Confidence)
		}
		// For same confidence, check IP ordering.
		if got[i].Confidence == got[i-1].Confidence && got[i].IP < got[i-1].IP {
			t.Errorf("same-confidence candidates not sorted by IP asc: [%d]=%s < [%d]=%s",
				i, got[i].IP, i-1, got[i-1].IP)
		}
	}
}

func TestIsInfrastructureManufacturer(t *testing.T) {
	tests := []struct {
		manufacturer string
		want         bool
	}{
		{"", false},
		{"Samsung", false},
		{"Dell Inc.", false},
		{"Apple", false},
		{"Netgear", true},
		{"NETGEAR", true},
		{"TP-Link Technologies", true},
		{"Cisco Systems", true},
		{"Ubiquiti Networks", true},
		{"D-Link Corporation", true},
		{"Zyxel Communications", true},
		{"Tenda Technology", true},
		{"Buffalo Inc.", true},
		{"Aruba Networks", true},
		{"Juniper Networks", true},
		{"MikroTik", true},
	}

	for _, tt := range tests {
		t.Run(tt.manufacturer, func(t *testing.T) {
			got := isInfrastructureManufacturer(tt.manufacturer)
			if got != tt.want {
				t.Errorf("isInfrastructureManufacturer(%q) = %v, want %v", tt.manufacturer, got, tt.want)
			}
		})
	}
}

func TestCountInfraManufacturers(t *testing.T) {
	devices := []UnmanagedDeviceInfo{
		{Manufacturer: "Netgear"},
		{Manufacturer: "Netgear"},
		{Manufacturer: "Cisco"},
		{Manufacturer: "Dell"},
		{Manufacturer: ""},
	}

	counts := countInfraManufacturers(devices)

	if counts["netgear"] != 2 {
		t.Errorf("netgear count = %d, want 2", counts["netgear"])
	}
	if counts["cisco"] != 1 {
		t.Errorf("cisco count = %d, want 1", counts["cisco"])
	}
	if _, ok := counts["dell"]; ok {
		t.Error("dell should not be in infrastructure counts")
	}
	if _, ok := counts[""]; ok {
		t.Error("empty manufacturer should not be in counts")
	}
}
