package recon

import (
	"strings"

	"github.com/HerbHall/subnetree/pkg/models"
)

// classificationRule maps a set of manufacturer name patterns to a device type.
type classificationRule struct {
	deviceType models.DeviceType
	patterns   []string
}

// ouiClassificationRules defines manufacturer-to-device-type mappings.
// Patterns are matched case-insensitively via strings.Contains against the
// manufacturer name returned by OUI lookup.
//
// Order matters: more specific patterns (e.g., "hp networking") must appear
// before broader ones (e.g., "hewlett packard") so the first match wins.
var ouiClassificationRules = []classificationRule{
	// Networking infrastructure -- specific patterns first.
	{models.DeviceTypeRouter, []string{
		"cisco", "meraki", "mikrotik", "netgear", "tp-link",
		"d-link", "linksys", "asus",
	}},
	{models.DeviceTypeAccessPoint, []string{
		"ubiquiti", "eero", "nest wifi",
	}},
	{models.DeviceTypeSwitch, []string{
		"aruba", "ruckus", "juniper", "hp networking",
		"hewlett packard enterprise",
	}},

	// Cameras.
	{models.DeviceTypeCamera, []string{
		"ring", "wyze", "hikvision", "dahua", "reolink", "amcrest",
	}},

	// Printers.
	{models.DeviceTypePrinter, []string{
		"brother", "canon", "epson", "lexmark", "xerox", "ricoh",
	}},

	// NAS.
	{models.DeviceTypeNAS, []string{
		"synology", "qnap", "western digital", "readynas",
	}},

	// Mobile devices.
	{models.DeviceTypeMobile, []string{
		"samsung", "oneplus", "xiaomi", "huawei", "oppo", "vivo",
		"motorola", "lg electronics",
	}},

	// IoT.
	{models.DeviceTypeIoT, []string{
		"sonos", "roku", "amazon", "chromecast",
		"raspberry pi", "espressif",
		"philips", "ikea", "shelly",
	}},

	// Desktops/workstations -- broad patterns last to avoid false matches.
	{models.DeviceTypeDesktop, []string{
		"apple", "dell", "lenovo", "hp inc", "hewlett packard",
		"microsoft",
	}},
}

// ClassifyByManufacturer returns a device type hint based on the manufacturer
// name from OUI lookup. Returns DeviceTypeUnknown if no match is found.
func ClassifyByManufacturer(manufacturer string) models.DeviceType {
	if manufacturer == "" {
		return models.DeviceTypeUnknown
	}

	lower := strings.ToLower(manufacturer)

	for i := range ouiClassificationRules {
		for _, pattern := range ouiClassificationRules[i].patterns {
			if strings.Contains(lower, pattern) {
				return ouiClassificationRules[i].deviceType
			}
		}
	}
	return models.DeviceTypeUnknown
}
