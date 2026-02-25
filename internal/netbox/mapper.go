package netbox

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/HerbHall/subnetree/pkg/models"
)

// slugRegexp matches characters that are not alphanumeric or hyphens.
var slugRegexp = regexp.MustCompile(`[^a-z0-9-]+`)

// SlugFromName converts a human-readable name to a NetBox-compatible slug.
// Lowercases, replaces spaces/special chars with hyphens, and trims edges.
func SlugFromName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = slugRegexp.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	if s == "" {
		return "unknown"
	}
	return s
}

// MapDeviceRole maps a SubNetree DeviceType to a NetBox device role name.
func MapDeviceRole(dt models.DeviceType) string {
	switch dt {
	case models.DeviceTypeRouter:
		return "Router"
	case models.DeviceTypeSwitch:
		return "Switch"
	case models.DeviceTypeFirewall:
		return "Firewall"
	case models.DeviceTypeAccessPoint:
		return "Access Point"
	case models.DeviceTypeServer:
		return "Server"
	case models.DeviceTypeDesktop:
		return "Desktop"
	case models.DeviceTypeLaptop:
		return "Laptop"
	case models.DeviceTypeMobile:
		return "Mobile Device"
	case models.DeviceTypePrinter:
		return "Printer"
	case models.DeviceTypeIoT:
		return "IoT Device"
	case models.DeviceTypeNAS:
		return "NAS"
	case models.DeviceTypePhone:
		return "Phone"
	case models.DeviceTypeTablet:
		return "Tablet"
	case models.DeviceTypeCamera:
		return "Camera"
	case models.DeviceTypeVM:
		return "Virtual Machine"
	case models.DeviceTypeContainer:
		return "Container"
	case models.DeviceTypeUnknown:
		return "Unknown"
	}
	return "Unknown"
}

// MapDeviceStatus maps a SubNetree DeviceStatus to a NetBox device status value.
func MapDeviceStatus(status models.DeviceStatus) string {
	switch status {
	case models.DeviceStatusOnline:
		return "active"
	case models.DeviceStatusOffline:
		return "offline"
	case models.DeviceStatusDegraded:
		return "planned"
	case models.DeviceStatusUnknown:
		return "inventory"
	}
	return "inventory"
}

// DeviceToNetBoxRequest maps a SubNetree Device to a NetBox create/update request.
func DeviceToNetBoxRequest(device *models.Device, roleID, typeID, siteID, tagID int) NBDeviceCreateRequest {
	name := device.Hostname
	if name == "" && len(device.IPAddresses) > 0 {
		name = device.IPAddresses[0]
	}
	if name == "" {
		name = fmt.Sprintf("device-%s", device.ID[:8])
	}

	comments := device.Notes
	if device.OS != "" {
		if comments != "" {
			comments += "\n"
		}
		comments += fmt.Sprintf("OS: %s", device.OS)
	}

	req := NBDeviceCreateRequest{
		Name:       name,
		DeviceType: typeID,
		Role:       roleID,
		Site:       siteID,
		Status:     MapDeviceStatus(device.Status),
		Comments:   comments,
	}

	if tagID > 0 {
		req.Tags = []int{tagID}
	}

	cf := make(map[string]interface{})
	cf["subnetree_id"] = device.ID
	if device.MACAddress != "" {
		cf["subnetree_mac"] = device.MACAddress
	}
	if device.DiscoveryMethod != "" {
		cf["subnetree_discovery"] = string(device.DiscoveryMethod)
	}
	req.CustomFields = cf

	return req
}
