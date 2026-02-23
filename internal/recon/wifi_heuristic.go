package recon

import (
	"context"
	"strings"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// WiFi-associated mDNS service types that strongly imply wireless connectivity.
var wifiMDNSServices = map[string]bool{
	"_airplay._tcp":    true,
	"_raop._tcp":       true,
	"_googlecast._tcp": true,
	"_homekit._tcp":    true,
	"_hap._tcp":        true,
}

// Manufacturers whose devices are overwhelmingly WiFi-connected.
var wifiTypicalManufacturers = []string{
	"apple", "samsung", "google", "ring", "nest", "sonos", "roku",
	"amazon", "espressif", "shelly", "xiaomi", "tp-link smart",
	"philips hue", "ikea", "wyze", "ecobee", "wemo",
}

// Hostname patterns that strongly suggest WiFi devices.
var wifiHostnamePatterns = []string{
	"iphone", "ipad", "galaxy", "pixel", "android",
	"esp_", "esp32", "esp8266", "tasmota", "shelly",
	"chromecast", "google-home", "echo", "alexa",
}

// WiFiAnalysis holds the result of WiFi heuristic analysis for a single device.
type WiFiAnalysis struct {
	ConnectionType string   // "wired", "wifi", or "unknown"
	Score          int      // Cumulative heuristic score
	Signals        []string // Human-readable signal descriptions
}

// IsLocallyAdministeredMAC checks if a MAC address has the locally-administered
// bit set (second nibble of first octet is 2, 3, 6, 7, A, B, E, or F). This
// indicates MAC randomization, used by iOS 14+ and Android 10+ for WiFi connections.
func IsLocallyAdministeredMAC(mac string) bool {
	if len(mac) < 2 {
		return false
	}
	// MAC format: "AA:BB:CC:DD:EE:FF" or "aa:bb:cc:dd:ee:ff"
	// The locally-administered bit is bit 1 of the first octet.
	// In hex, if the second character of the first octet is 2, 3, 6, 7, A, B, E, F
	// then the bit is set.
	cleaned := strings.ReplaceAll(strings.ReplaceAll(mac, ":", ""), "-", "")
	if len(cleaned) < 2 {
		return false
	}
	secondNibble := cleaned[1]
	switch secondNibble {
	case '2', '3', '6', '7', 'a', 'b', 'e', 'f', 'A', 'B', 'E', 'F':
		return true
	}
	return false
}

// AnalyzeWiFiConnection determines the likely connection type for a device
// using multiple heuristic signals.
func AnalyzeWiFiConnection(device *models.Device, inFDB bool, discoveredMDNSServices []string) WiFiAnalysis {
	// If the device is in a switch FDB table, it's definitively wired.
	if inFDB {
		return WiFiAnalysis{
			ConnectionType: models.ConnectionWired,
			Score:          100,
			Signals:        []string{"Found in switch FDB table"},
		}
	}

	var score int
	var signals []string

	// Signal 1: Locally-administered MAC (randomized by phones/tablets)
	if device.MACAddress != "" && IsLocallyAdministeredMAC(device.MACAddress) {
		score += 30
		signals = append(signals, "Locally-administered MAC (randomized, likely phone/tablet on WiFi)")
	}

	// Signal 2: Not in any switch FDB table (only for endpoint-class devices)
	if device.MACAddress != "" && !isInfrastructureType(device.DeviceType) {
		score += 25
		signals = append(signals, "Not found in any switch FDB table")
	}

	// Signal 3: WiFi-associated mDNS services
	for _, svc := range discoveredMDNSServices {
		if wifiMDNSServices[svc] {
			score += 20
			signals = append(signals, "mDNS service "+svc+" associated with WiFi devices")
			break // Only count once
		}
	}

	// Signal 4: WiFi-typical manufacturer
	if device.Manufacturer != "" {
		mfgLower := strings.ToLower(device.Manufacturer)
		for _, pattern := range wifiTypicalManufacturers {
			if strings.Contains(mfgLower, pattern) {
				score += 15
				signals = append(signals, "Manufacturer "+device.Manufacturer+" typically WiFi")
				break
			}
		}
	}

	// Signal 5: WiFi-typical hostname patterns
	if device.Hostname != "" {
		hostLower := strings.ToLower(device.Hostname)
		for _, pattern := range wifiHostnamePatterns {
			if strings.Contains(hostLower, pattern) {
				score += 10
				signals = append(signals, "Hostname matches WiFi device pattern: "+pattern)
				break
			}
		}
	}

	connType := models.ConnectionUnknown
	if score >= 40 {
		connType = models.ConnectionWiFi
	}

	return WiFiAnalysis{
		ConnectionType: connType,
		Score:          score,
		Signals:        signals,
	}
}

// isInfrastructureType returns true for device types that are never WiFi clients.
func isInfrastructureType(dt models.DeviceType) bool {
	switch dt {
	case models.DeviceTypeRouter, models.DeviceTypeSwitch,
		models.DeviceTypeFirewall, models.DeviceTypeAccessPoint:
		return true
	}
	return false
}

// WiFiHeuristicAnalyzer runs WiFi connection heuristics across all devices.
type WiFiHeuristicAnalyzer struct {
	store  *ReconStore
	logger *zap.Logger
}

// NewWiFiHeuristicAnalyzer creates a new analyzer.
func NewWiFiHeuristicAnalyzer(store *ReconStore, logger *zap.Logger) *WiFiHeuristicAnalyzer {
	return &WiFiHeuristicAnalyzer{store: store, logger: logger}
}

// AnalyzeAll runs WiFi heuristics on all devices and updates their connection types.
func (a *WiFiHeuristicAnalyzer) AnalyzeAll(ctx context.Context) {
	fdbMACs, err := a.store.GetAllFDBMACs(ctx)
	if err != nil {
		a.logger.Error("failed to get FDB MACs for WiFi analysis", zap.Error(err))
		fdbMACs = make(map[string]bool)
	}

	devices, err := a.store.ListAllDevices(ctx)
	if err != nil {
		a.logger.Error("failed to list devices for WiFi analysis", zap.Error(err))
		return
	}

	var wiredCount, wifiCount int
	for i := range devices {
		if ctx.Err() != nil {
			return
		}

		d := &devices[i]
		inFDB := fdbMACs[strings.ToLower(d.MACAddress)]

		// TODO: In the future, query mDNS cache for discovered services per device.
		// For now, pass empty -- the other heuristics are sufficient.
		var mdnsServices []string

		analysis := AnalyzeWiFiConnection(d, inFDB, mdnsServices)

		if analysis.ConnectionType != models.ConnectionUnknown && analysis.ConnectionType != d.ConnectionType {
			if updateErr := a.store.UpdateDeviceConnectionType(ctx, d.ID, analysis.ConnectionType); updateErr != nil {
				a.logger.Error("failed to update connection type",
					zap.String("device_id", d.ID),
					zap.Error(updateErr))
				continue
			}
			switch analysis.ConnectionType {
			case models.ConnectionWired:
				wiredCount++
			case models.ConnectionWiFi:
				wifiCount++
			}
		}
	}

	if wiredCount > 0 || wifiCount > 0 {
		a.logger.Info("WiFi heuristic analysis complete",
			zap.Int("wired", wiredCount),
			zap.Int("wifi", wifiCount))
	}
}
