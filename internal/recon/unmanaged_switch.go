package recon

import (
	"sort"
	"strings"

	"github.com/HerbHall/subnetree/pkg/models"
)

// UnmanagedSwitchCandidate represents a device suspected of being an unmanaged switch.
type UnmanagedSwitchCandidate struct {
	DeviceID     string
	IP           string
	MAC          string
	Manufacturer string
	Reason       string // human-readable reason for suspicion
	Confidence   int    // 0-100 (should be low, 15-30 range)
}

// UnmanagedDeviceInfo is the input data needed for each device.
type UnmanagedDeviceInfo struct {
	DeviceID      string
	IP            string
	MAC           string
	Manufacturer  string
	DeviceType    models.DeviceType // current classification
	HasSNMP       bool
	HasOpenPorts  bool
	OpenPortCount int
}

// infrastructureOUIPatterns lists manufacturer name substrings that indicate
// networking infrastructure vendors. Matched case-insensitively.
var infrastructureOUIPatterns = []string{
	"netgear", "tp-link", "d-link", "linksys", "cisco", "meraki",
	"mikrotik", "ubiquiti", "aruba", "ruckus", "juniper", "asus",
	"hewlett packard enterprise", "hp networking", "zyxel", "tenda",
	"buffalo", "eero",
}

// isInfrastructureManufacturer checks whether the manufacturer string matches
// a known networking infrastructure vendor. This is a broader check than
// IsInfrastructureOUI which checks device type; here we match on the raw
// manufacturer name to catch devices whose OUI classification returned a
// non-infrastructure type (e.g., "router" or "access_point" are infrastructure,
// but the OUI classifier may return "unknown" for ambiguous vendors).
func isInfrastructureManufacturer(manufacturer string) bool {
	if manufacturer == "" {
		return false
	}
	lower := strings.ToLower(manufacturer)
	for _, pattern := range infrastructureOUIPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// DetectUnmanagedSwitches analyzes post-scan data to find potential unmanaged switches.
// It looks for devices with infrastructure vendor OUIs that have no SNMP, no open ports,
// and were not classified by any other method.
//
// Heuristics applied:
//  1. OUI-only infrastructure: infrastructure vendor OUI + no SNMP + no open ports + type Unknown.
//  2. Ghost MAC: device has a MAC (appeared in ARP) but no open ports and no SNMP.
//
// Confidence is intentionally low (15-30) since these are heuristic guesses.
func DetectUnmanagedSwitches(devices []UnmanagedDeviceInfo) []UnmanagedSwitchCandidate {
	if len(devices) == 0 {
		return nil
	}

	candidates := make([]UnmanagedSwitchCandidate, 0, len(devices))

	// Track infrastructure OUI count per manufacturer for clustering bonus.
	infraCount := countInfraManufacturers(devices)

	for i := range devices {
		d := &devices[i]

		// Skip devices that are already classified as something specific.
		if d.DeviceType != models.DeviceTypeUnknown {
			continue
		}

		// Skip devices with SNMP -- they are managed.
		if d.HasSNMP {
			continue
		}

		// Skip devices with open ports -- they are visible on the network.
		if d.HasOpenPorts || d.OpenPortCount > 0 {
			continue
		}

		// Check if the OUI type from classification suggests infrastructure.
		ouiType := ClassifyByManufacturer(d.Manufacturer)
		ouiIsInfra := IsInfrastructureOUI(ouiType)

		// Also check if the raw manufacturer name matches infrastructure vendors.
		mfrIsInfra := isInfrastructureManufacturer(d.Manufacturer)

		if !ouiIsInfra && !mfrIsInfra {
			continue
		}

		// Base confidence: infrastructure OUI with no ports, no SNMP, type unknown.
		confidence := 20
		reason := "infrastructure vendor OUI with no open ports and no SNMP"

		// Boost confidence if OUI classifier also recognizes it as infrastructure.
		if ouiIsInfra {
			confidence += 5
			reason = "OUI classified as " + string(ouiType) + " but no management ports detected"
		}

		// MAC clustering bonus: multiple devices from the same infrastructure
		// manufacturer on the segment increases likelihood of unmanaged switch.
		if d.Manufacturer != "" && infraCount[strings.ToLower(d.Manufacturer)] > 1 {
			confidence += 5
			reason += "; multiple devices from same infrastructure vendor on segment"
		}

		// Ghost MAC signal: has MAC but nothing else.
		if d.MAC != "" && d.Manufacturer == "" {
			confidence = 15
			reason = "MAC present in ARP table but no manufacturer, ports, or SNMP"
		}

		// Cap confidence at 30 for heuristic detection.
		if confidence > 30 {
			confidence = 30
		}

		candidates = append(candidates, UnmanagedSwitchCandidate{
			DeviceID:     d.DeviceID,
			IP:           d.IP,
			MAC:          d.MAC,
			Manufacturer: d.Manufacturer,
			Reason:       reason,
			Confidence:   confidence,
		})
	}

	// Sort by confidence descending, then by IP for deterministic ordering.
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Confidence != candidates[j].Confidence {
			return candidates[i].Confidence > candidates[j].Confidence
		}
		return candidates[i].IP < candidates[j].IP
	})

	return candidates
}

// countInfraManufacturers counts how many devices share each infrastructure
// manufacturer name (case-insensitive). Used for the MAC clustering heuristic.
func countInfraManufacturers(devices []UnmanagedDeviceInfo) map[string]int {
	counts := make(map[string]int)
	for i := range devices {
		mfr := strings.ToLower(devices[i].Manufacturer)
		if mfr != "" && isInfrastructureManufacturer(devices[i].Manufacturer) {
			counts[mfr]++
		}
	}
	return counts
}
