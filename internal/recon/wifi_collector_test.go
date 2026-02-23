package recon

import (
	"context"
	"testing"
)

func TestFreqToChannel(t *testing.T) {
	tests := []struct {
		name    string
		freqMHz int
		want    int
	}{
		// 2.4 GHz band
		{"2.4GHz channel 1", 2412, 1},
		{"2.4GHz channel 6", 2437, 6},
		{"2.4GHz channel 11", 2462, 11},
		{"2.4GHz channel 13", 2472, 13},
		{"2.4GHz channel 14 (Japan)", 2484, 14},

		// 5 GHz band
		{"5GHz channel 36", 5180, 36},
		{"5GHz channel 40", 5200, 40},
		{"5GHz channel 44", 5220, 44},
		{"5GHz channel 48", 5240, 48},
		{"5GHz channel 149", 5745, 149},
		{"5GHz channel 165", 5825, 165},

		// 6 GHz band (WiFi 6E)
		{"6GHz channel 1", 5955, 1},
		{"6GHz channel 5", 5975, 5},
		{"6GHz channel 233", 7115, 233},

		// Edge / invalid cases
		{"below 2.4GHz", 2400, 0},
		{"between bands", 3000, 0},
		{"zero frequency", 0, 0},
		{"negative frequency", -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := freqToChannel(tt.freqMHz)
			if got != tt.want {
				t.Errorf("freqToChannel(%d) = %d, want %d", tt.freqMHz, got, tt.want)
			}
		})
	}
}

func TestQualityToDBm(t *testing.T) {
	tests := []struct {
		name    string
		quality int
		want    int
	}{
		{"0 percent", 0, -100},
		{"50 percent", 50, -75},
		{"100 percent", 100, -50},
		{"below zero", -10, -100},
		{"above 100", 150, -50},
		{"25 percent", 25, -88},  // integer division: half of 25 minus 100
		{"75 percent", 75, -63},  // integer division: half of 75 minus 100
		{"1 percent", 1, -100},   // integer division: half of 1 minus 100
		{"99 percent", 99, -51},  // integer division: half of 99 minus 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := qualityToDBm(tt.quality)
			if got != tt.want {
				t.Errorf("qualityToDBm(%d) = %d, want %d", tt.quality, got, tt.want)
			}
		})
	}
}

func TestAuthAlgoToSecurity(t *testing.T) {
	tests := []struct {
		name string
		algo int
		want string
	}{
		{"Open", 1, "Open"},
		{"SharedKey (WEP)", 2, "WEP"},
		{"WPA", 3, "WPA"},
		{"WPA-PSK", 4, "WPA"},
		{"WPA-NONE", 5, "Open"},
		{"RSNA (WPA2-Enterprise)", 6, "WPA2"},
		{"RSNA-PSK (WPA2-Personal)", 7, "WPA2"},
		{"WPA3 Enterprise 192-bit", 8, "WPA3"},
		{"WPA3-SAE (Personal)", 9, "WPA3"},
		{"OWE", 10, "OWE"},
		{"WPA3 Enterprise", 11, "WPA3"},
		{"unknown algorithm", 99, "Unknown"},
		{"zero", 0, "Unknown"},
		{"negative", -1, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authAlgoToSecurity(tt.algo)
			if got != tt.want {
				t.Errorf("authAlgoToSecurity(%d) = %q, want %q", tt.algo, got, tt.want)
			}
		})
	}
}

func TestAccessPointInfoConstruction(t *testing.T) {
	ap := AccessPointInfo{
		BSSID:     "aa:bb:cc:dd:ee:ff",
		SSID:      "MyNetwork",
		Channel:   6,
		Frequency: 2437,
		Signal:    -65,
		Security:  "WPA2",
	}

	if ap.BSSID != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("BSSID = %q, want %q", ap.BSSID, "aa:bb:cc:dd:ee:ff")
	}
	if ap.SSID != "MyNetwork" {
		t.Errorf("SSID = %q, want %q", ap.SSID, "MyNetwork")
	}
	if ap.Channel != 6 {
		t.Errorf("Channel = %d, want %d", ap.Channel, 6)
	}
	if ap.Frequency != 2437 {
		t.Errorf("Frequency = %d, want %d", ap.Frequency, 2437)
	}
	if ap.Signal != -65 {
		t.Errorf("Signal = %d, want %d", ap.Signal, -65)
	}
	if ap.Security != "WPA2" {
		t.Errorf("Security = %q, want %q", ap.Security, "WPA2")
	}
}

// noopWifiScanner is a test-only stub used in place of the build-tagged
// stubWifiScanner so the test compiles on all platforms.
type noopWifiScanner struct{}

func (s *noopWifiScanner) Available() bool                              { return false }
func (s *noopWifiScanner) Scan(_ context.Context) ([]AccessPointInfo, error) { return nil, nil }

func TestWifiScannerInterface(t *testing.T) {
	// Verify a no-op implementation satisfies the interface contract.
	var scanner WifiScanner = &noopWifiScanner{}

	if scanner.Available() {
		t.Error("noop scanner should not be available")
	}

	aps, err := scanner.Scan(context.Background())
	if err != nil {
		t.Errorf("noop scanner Scan returned error: %v", err)
	}
	if aps != nil {
		t.Errorf("noop scanner Scan returned non-nil: %v", aps)
	}
}
