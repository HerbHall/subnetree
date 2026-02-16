package recon

import (
	"sort"

	"github.com/HerbHall/subnetree/pkg/models"
)

// ClassificationSignal represents a single piece of evidence for device type classification.
type ClassificationSignal struct {
	Source     string            `json:"source"`      // e.g., "snmp_bridge_mib", "oui", "lldp_caps"
	DeviceType models.DeviceType `json:"device_type"` // The device type this signal suggests
	Weight     int               `json:"weight"`      // Confidence weight (0-50)
	Detail     string            `json:"detail"`      // Human-readable explanation
}

// ClassificationResult holds the output of the composite classifier.
type ClassificationResult struct {
	DeviceType models.DeviceType      `json:"device_type"`
	Confidence int                    `json:"confidence"` // 0-100
	Source     string                 `json:"source"`     // Primary signal source
	Signals    []ClassificationSignal `json:"signals"`    // All contributing signals
}

// ConfidenceLevel represents the confidence tier.
type ConfidenceLevel string

const (
	ConfidenceIdentified ConfidenceLevel = "identified" // Score >= 50
	ConfidenceProbable   ConfidenceLevel = "probable"   // Score 25-49
	ConfidenceUnknown    ConfidenceLevel = "unknown"    // Score < 25
)

// ConfidenceLevelFor returns the confidence tier for a given score.
func ConfidenceLevelFor(score int) ConfidenceLevel {
	switch {
	case score >= 50:
		return ConfidenceIdentified
	case score >= 25:
		return ConfidenceProbable
	default:
		return ConfidenceUnknown
	}
}

// Signal weight constants.
const (
	WeightSNMPBridgeMIB   = 35 // BRIDGE-MIB responds -> definitive switch
	WeightSNMPSysServices = 30 // sysServices layer match
	WeightLLDPCaps        = 40 // LLDP capability bitmap
	WeightUPnPDeviceType  = 25 // UPnP device type URN
	WeightMDNSService     = 20 // mDNS vendor-specific service
	WeightPortProfile     = 15 // Port combination fingerprint
	WeightOUIVendor       = 25 // OUI manufacturer hint
	WeightTTLNetwork      = 10 // TTL=255 (network equipment)
	WeightSNMPSysDescr    = 10 // sysDescr keyword match
)

// DeviceSignals collects all available classification data for a single device.
type DeviceSignals struct {
	// OUI-based classification (from oui_classifier.go).
	OUIDeviceType models.DeviceType
	Manufacturer  string

	// SNMP-based classification (from snmp_collector.go).
	SNMPDeviceType models.DeviceType
	SNMPInfo       *SNMPSystemInfo // nil if SNMP not available

	// LLDP capability-based classification.
	LLDPDeviceType models.DeviceType
	LLDPCaps       uint16

	// Port fingerprinting classification (from port_fingerprint.go).
	PortDeviceType models.DeviceType
	OpenPorts      []int

	// TTL-based OS hint (from icmp.go).
	TTL    int
	OSHint string

	// Manual override (user set).
	ManualType models.DeviceType
}

// Classify runs the composite classification engine on the collected signals.
// It returns the best device type with confidence scoring.
func Classify(signals *DeviceSignals) *ClassificationResult {
	if signals == nil {
		return &ClassificationResult{
			DeviceType: models.DeviceTypeUnknown,
			Confidence: 0,
			Source:     "none",
		}
	}

	// Rule 1: Never override manual classification.
	if signals.ManualType != "" && signals.ManualType != models.DeviceTypeUnknown {
		return &ClassificationResult{
			DeviceType: signals.ManualType,
			Confidence: 100,
			Source:     "manual",
			Signals: []ClassificationSignal{{
				Source:     "manual",
				DeviceType: signals.ManualType,
				Weight:     100,
				Detail:     "Manually set by user",
			}},
		}
	}

	// Collect all signals.
	var allSignals []ClassificationSignal

	// SNMP BRIDGE-MIB signal.
	if signals.SNMPInfo != nil && (signals.SNMPInfo.BridgeAddress != "" || signals.SNMPInfo.BridgeNumPorts > 1) {
		dt := models.DeviceTypeSwitch
		if signals.SNMPInfo.Services&0x04 != 0 {
			dt = models.DeviceTypeRouter // L3 switch
		}
		allSignals = append(allSignals, ClassificationSignal{
			Source:     "snmp_bridge_mib",
			DeviceType: dt,
			Weight:     WeightSNMPBridgeMIB,
			Detail:     "BRIDGE-MIB responded with bridge data",
		})
	}

	// SNMP sysServices signal.
	if signals.SNMPInfo != nil && signals.SNMPInfo.Services != 0 {
		var dt models.DeviceType
		switch {
		case signals.SNMPInfo.Services&0x04 != 0 && signals.SNMPInfo.Services&0x02 == 0:
			dt = models.DeviceTypeRouter
		case signals.SNMPInfo.Services&0x02 != 0:
			dt = models.DeviceTypeSwitch
		}
		if dt != "" {
			allSignals = append(allSignals, ClassificationSignal{
				Source:     "snmp_sys_services",
				DeviceType: dt,
				Weight:     WeightSNMPSysServices,
				Detail:     "sysServices OSI layer bitmask",
			})
		}
	}

	// SNMP sysDescr signal.
	if signals.SNMPDeviceType != "" && signals.SNMPDeviceType != models.DeviceTypeUnknown {
		allSignals = append(allSignals, ClassificationSignal{
			Source:     "snmp_sys_descr",
			DeviceType: signals.SNMPDeviceType,
			Weight:     WeightSNMPSysDescr,
			Detail:     "sysDescr keyword match",
		})
	}

	// LLDP capabilities signal.
	if signals.LLDPDeviceType != "" && signals.LLDPDeviceType != models.DeviceTypeUnknown {
		allSignals = append(allSignals, ClassificationSignal{
			Source:     "lldp_caps",
			DeviceType: signals.LLDPDeviceType,
			Weight:     WeightLLDPCaps,
			Detail:     "LLDP capability bitmap",
		})
	}

	// Port fingerprint signal.
	if signals.PortDeviceType != "" && signals.PortDeviceType != models.DeviceTypeUnknown {
		allSignals = append(allSignals, ClassificationSignal{
			Source:     "port_fingerprint",
			DeviceType: signals.PortDeviceType,
			Weight:     WeightPortProfile,
			Detail:     "Infrastructure port combination match",
		})
	}

	// OUI vendor signal.
	if signals.OUIDeviceType != "" && signals.OUIDeviceType != models.DeviceTypeUnknown {
		allSignals = append(allSignals, ClassificationSignal{
			Source:     "oui_vendor",
			DeviceType: signals.OUIDeviceType,
			Weight:     WeightOUIVendor,
			Detail:     "Manufacturer OUI classification for " + signals.Manufacturer,
		})
	}

	// TTL signal (only for network equipment hint).
	if signals.TTL == 255 {
		// TTL=255 strongly suggests network equipment but doesn't tell us which type.
		// Apply as a boost to any infrastructure classification.
		allSignals = append(allSignals, ClassificationSignal{
			Source:     "ttl_hint",
			DeviceType: models.DeviceTypeRouter, // Default to router for TTL=255
			Weight:     WeightTTLNetwork,
			Detail:     "TTL=255 indicates network equipment",
		})
	}

	if len(allSignals) == 0 {
		return &ClassificationResult{
			DeviceType: models.DeviceTypeUnknown,
			Confidence: 0,
			Source:     "none",
		}
	}

	// Aggregate scores by device type.
	typeScores := make(map[models.DeviceType]int)
	for i := range allSignals {
		typeScores[allSignals[i].DeviceType] += allSignals[i].Weight
	}

	// Find the highest-scoring device type.
	var bestType models.DeviceType
	var bestScore int
	for dt, score := range typeScores {
		if score > bestScore {
			bestScore = score
			bestType = dt
		}
	}

	// Cap confidence at 100.
	if bestScore > 100 {
		bestScore = 100
	}

	// Find the primary source (highest-weight signal for the winning type).
	var primarySource string
	var highestWeight int
	for i := range allSignals {
		if allSignals[i].DeviceType == bestType && allSignals[i].Weight > highestWeight {
			highestWeight = allSignals[i].Weight
			primarySource = allSignals[i].Source
		}
	}

	// Sort signals by weight descending for display.
	sort.Slice(allSignals, func(i, j int) bool {
		return allSignals[i].Weight > allSignals[j].Weight
	})

	return &ClassificationResult{
		DeviceType: bestType,
		Confidence: bestScore,
		Source:     primarySource,
		Signals:    allSignals,
	}
}
