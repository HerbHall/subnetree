package recon

import "github.com/HerbHall/subnetree/pkg/models"

// portFingerprint maps a set of required and optional ports to a device type.
type portFingerprint struct {
	deviceType    models.DeviceType
	requiredPorts []int // ALL must be open
	optionalPorts []int // At least one should be open (if specified)
	description   string
}

// portFingerprints defines known port combination patterns for infrastructure devices.
// Order matters: more specific patterns (more required ports) should come first.
var portFingerprints = []portFingerprint{
	// Ubiquiti UniFi switch/AP (very specific).
	{
		deviceType:    models.DeviceTypeSwitch,
		requiredPorts: []int{22, 80, 8443},
		description:   "Ubiquiti UniFi device",
	},
	// MikroTik router (Winbox port is distinctive).
	{
		deviceType:    models.DeviceTypeRouter,
		requiredPorts: []int{80, 8291},
		description:   "MikroTik RouterOS",
	},
	// Cisco-like managed switch/router (SSH + Telnet + HTTP + SNMP).
	{
		deviceType:    models.DeviceTypeSwitch,
		requiredPorts: []int{22, 23, 80},
		description:   "Managed switch (Cisco-like)",
	},
	// Managed switch with SNMP or HTTPS alongside SSH + HTTP.
	{
		deviceType:    models.DeviceTypeSwitch,
		requiredPorts: []int{22, 80},
		optionalPorts: []int{161, 443},
		description:   "Managed switch with management services",
	},
	// SSH + HTTP management (infrastructure OUI required by caller).
	{
		deviceType:    models.DeviceTypeSwitch,
		requiredPorts: []int{22, 80},
		description:   "Managed switch (generic)",
	},
	// Device with only web management (infra OUI required).
	{
		deviceType:    models.DeviceTypeRouter,
		requiredPorts: []int{80, 443},
		description:   "Consumer router/AP with web management",
	},
	// SSH-only device with SNMP (likely embedded/infrastructure).
	{
		deviceType:    models.DeviceTypeSwitch,
		requiredPorts: []int{22},
		optionalPorts: []int{161},
		description:   "SSH-managed infrastructure device with SNMP",
	},
}

// ClassifyByPorts attempts to classify a device based on its open ports.
// Only call this for devices with infrastructure OUI vendors.
// Returns DeviceTypeUnknown if no fingerprint matches.
func ClassifyByPorts(openPorts []int) models.DeviceType {
	if len(openPorts) == 0 {
		return models.DeviceTypeUnknown
	}

	portSet := make(map[int]bool, len(openPorts))
	for _, p := range openPorts {
		portSet[p] = true
	}

	for i := range portFingerprints {
		if matchesFingerprint(portSet, &portFingerprints[i]) {
			return portFingerprints[i].deviceType
		}
	}

	return models.DeviceTypeUnknown
}

// matchesFingerprint checks if the open ports match a fingerprint pattern.
func matchesFingerprint(portSet map[int]bool, fp *portFingerprint) bool {
	// All required ports must be open.
	for _, p := range fp.requiredPorts {
		if !portSet[p] {
			return false
		}
	}

	// If optional ports specified, at least one must be open.
	if len(fp.optionalPorts) > 0 {
		anyOptional := false
		for _, p := range fp.optionalPorts {
			if portSet[p] {
				anyOptional = true
				break
			}
		}
		if !anyOptional {
			return false
		}
	}

	return true
}

// IsInfrastructureOUI checks whether a device type from OUI classification
// suggests the device is network infrastructure worth port scanning.
func IsInfrastructureOUI(deviceType models.DeviceType) bool {
	switch deviceType {
	case models.DeviceTypeRouter,
		models.DeviceTypeSwitch,
		models.DeviceTypeAccessPoint,
		models.DeviceTypeFirewall:
		return true
	default:
		return false
	}
}
