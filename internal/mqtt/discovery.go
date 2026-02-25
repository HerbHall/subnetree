package mqtt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/HerbHall/subnetree/pkg/models"
)

// nonAlphanumeric matches any character that is not alphanumeric or underscore.
var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// DiscoveryConfig holds a single HA MQTT discovery payload.
type DiscoveryConfig struct {
	Topic   string // Full MQTT topic (homeassistant/...)
	Payload []byte // JSON-encoded config (empty = remove)
	Retain  bool   // Discovery configs should always be retained
}

// HADevice is the "device" block in HA discovery payloads.
type HADevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Model        string   `json:"model,omitempty"`
	Manufacturer string   `json:"manufacturer,omitempty"`
	SWVersion    string   `json:"sw_version,omitempty"`
	ViaDevice    string   `json:"via_device,omitempty"`
}

// BinarySensorConfig is the HA discovery payload for binary_sensor.
type BinarySensorConfig struct {
	Name              string   `json:"name"`
	ObjectID          string   `json:"object_id"`
	UniqueID          string   `json:"unique_id"`
	StateTopic        string   `json:"state_topic"`
	DeviceClass       string   `json:"device_class,omitempty"`
	PayloadOn         string   `json:"payload_on"`
	PayloadOff        string   `json:"payload_off"`
	Device            HADevice `json:"device"`
	AvailabilityTopic string   `json:"availability_topic,omitempty"`
	Icon              string   `json:"icon,omitempty"`
}

// SensorConfig is the HA discovery payload for sensor.
type SensorConfig struct {
	Name       string   `json:"name"`
	ObjectID   string   `json:"object_id"`
	UniqueID   string   `json:"unique_id"`
	StateTopic string   `json:"state_topic"`
	Icon       string   `json:"icon,omitempty"`
	Device     HADevice `json:"device"`
}

// SafeObjectID sanitizes a string for use as an HA object_id.
// Replaces any non-alphanumeric character (except underscore) with underscore,
// lowercases, and trims leading/trailing underscores.
func SafeObjectID(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "unknown"
	}
	return s
}

// buildHADevice creates the HA device block from a SubNetree device.
func buildHADevice(device *models.Device) HADevice {
	name := device.Hostname
	if name == "" {
		if len(device.IPAddresses) > 0 {
			name = device.IPAddresses[0]
		} else {
			name = device.ID
		}
	}
	return HADevice{
		Identifiers:  []string{"subnetree_" + device.ID},
		Name:         name,
		Model:        string(device.DeviceType),
		Manufacturer: device.Manufacturer,
		SWVersion:    device.OS,
		ViaDevice:    "subnetree",
	}
}

// BuildDeviceDiscoveryConfigs creates HA discovery config payloads for a device.
// Returns configs for: online/offline binary_sensor, device type sensor, IP sensor.
func BuildDeviceDiscoveryConfigs(device *models.Device, topicPrefix, haPrefix string) []DiscoveryConfig {
	if device == nil {
		return nil
	}

	safeID := SafeObjectID(device.ID)
	haDevice := buildHADevice(device)

	configs := make([]DiscoveryConfig, 0, 3)

	// 1. Binary sensor for online/offline status.
	onlineCfg := BinarySensorConfig{
		Name:        haDevice.Name + " Online",
		ObjectID:    "subnetree_" + safeID + "_online",
		UniqueID:    "subnetree_" + safeID + "_online",
		StateTopic:  topicPrefix + "/device/" + device.ID + "/online",
		DeviceClass: "connectivity",
		PayloadOn:   "ON",
		PayloadOff:  "OFF",
		Device:      haDevice,
	}
	onlinePayload, err := json.Marshal(onlineCfg)
	if err == nil {
		configs = append(configs, DiscoveryConfig{
			Topic:   fmt.Sprintf("%s/binary_sensor/subnetree_%s/online/config", haPrefix, safeID),
			Payload: onlinePayload,
			Retain:  true,
		})
	}

	// 2. Sensor for device type classification.
	typeCfg := SensorConfig{
		Name:       haDevice.Name + " Type",
		ObjectID:   "subnetree_" + safeID + "_type",
		UniqueID:   "subnetree_" + safeID + "_type",
		StateTopic: topicPrefix + "/device/" + device.ID + "/type",
		Icon:       DeviceTypeIcon(device.DeviceType),
		Device:     haDevice,
	}
	typePayload, err := json.Marshal(typeCfg)
	if err == nil {
		configs = append(configs, DiscoveryConfig{
			Topic:   fmt.Sprintf("%s/sensor/subnetree_%s/type/config", haPrefix, safeID),
			Payload: typePayload,
			Retain:  true,
		})
	}

	// 3. Sensor for IP address.
	if len(device.IPAddresses) > 0 {
		ipCfg := SensorConfig{
			Name:       haDevice.Name + " IP",
			ObjectID:   "subnetree_" + safeID + "_ip",
			UniqueID:   "subnetree_" + safeID + "_ip",
			StateTopic: topicPrefix + "/device/" + device.ID + "/ip",
			Icon:       "mdi:ip-network",
			Device:     haDevice,
		}
		ipPayload, err := json.Marshal(ipCfg)
		if err == nil {
			configs = append(configs, DiscoveryConfig{
				Topic:   fmt.Sprintf("%s/sensor/subnetree_%s/ip/config", haPrefix, safeID),
				Payload: ipPayload,
				Retain:  true,
			})
		}
	}

	return configs
}

// BuildAlertDiscoveryConfig creates an HA discovery config for an alert binary_sensor.
func BuildAlertDiscoveryConfig(deviceID, deviceName, alertID, severity, topicPrefix, haPrefix string) DiscoveryConfig {
	safeAlertID := SafeObjectID(alertID)
	safeDeviceID := SafeObjectID(deviceID)

	name := deviceName
	if name == "" {
		name = deviceID
	}

	cfg := BinarySensorConfig{
		Name:        name + " Alert",
		ObjectID:    "subnetree_" + safeAlertID + "_alert",
		UniqueID:    "subnetree_" + safeAlertID + "_alert",
		StateTopic:  topicPrefix + "/alert/" + alertID + "/state",
		DeviceClass: "problem",
		PayloadOn:   "triggered",
		PayloadOff:  "resolved",
		Icon:        alertSeverityIcon(severity),
		Device: HADevice{
			Identifiers: []string{"subnetree_" + deviceID},
			Name:        name,
			ViaDevice:   "subnetree",
		},
	}

	payload, err := json.Marshal(cfg)
	if err != nil {
		return DiscoveryConfig{}
	}
	return DiscoveryConfig{
		Topic:   fmt.Sprintf("%s/binary_sensor/subnetree_%s/alert_%s/config", haPrefix, safeDeviceID, safeAlertID),
		Payload: payload,
		Retain:  true,
	}
}

// BuildDeviceRemovalConfigs returns discovery configs with empty payloads to
// remove a device from HA. Publishing an empty payload to a discovery topic
// tells HA to remove the entity.
func BuildDeviceRemovalConfigs(deviceID, haPrefix string) []DiscoveryConfig {
	safeID := SafeObjectID(deviceID)
	return []DiscoveryConfig{
		{
			Topic:   fmt.Sprintf("%s/binary_sensor/subnetree_%s/online/config", haPrefix, safeID),
			Payload: nil,
			Retain:  true,
		},
		{
			Topic:   fmt.Sprintf("%s/sensor/subnetree_%s/type/config", haPrefix, safeID),
			Payload: nil,
			Retain:  true,
		},
		{
			Topic:   fmt.Sprintf("%s/sensor/subnetree_%s/ip/config", haPrefix, safeID),
			Payload: nil,
			Retain:  true,
		},
	}
}

// DeviceTypeIcon maps a SubNetree DeviceType to a Material Design Icon string
// for use in Home Assistant.
func DeviceTypeIcon(dt models.DeviceType) string {
	switch dt {
	case models.DeviceTypeServer:
		return "mdi:server"
	case models.DeviceTypeDesktop:
		return "mdi:desktop-tower"
	case models.DeviceTypeLaptop:
		return "mdi:laptop"
	case models.DeviceTypeMobile:
		return "mdi:cellphone"
	case models.DeviceTypeRouter:
		return "mdi:router-wireless"
	case models.DeviceTypeSwitch:
		return "mdi:switch"
	case models.DeviceTypePrinter:
		return "mdi:printer"
	case models.DeviceTypeIoT:
		return "mdi:devices"
	case models.DeviceTypeAccessPoint:
		return "mdi:access-point"
	case models.DeviceTypeFirewall:
		return "mdi:shield-lock"
	case models.DeviceTypeNAS:
		return "mdi:nas"
	case models.DeviceTypePhone:
		return "mdi:phone"
	case models.DeviceTypeTablet:
		return "mdi:tablet"
	case models.DeviceTypeCamera:
		return "mdi:cctv"
	case models.DeviceTypeVM:
		return "mdi:monitor-dashboard"
	case models.DeviceTypeContainer:
		return "mdi:docker"
	case models.DeviceTypeUnknown:
		return "mdi:help-network"
	}
	return "mdi:help-network"
}

// alertSeverityIcon maps an alert severity to an MDI icon.
func alertSeverityIcon(severity string) string {
	switch severity {
	case "critical":
		return "mdi:alert-circle"
	case "warning":
		return "mdi:alert"
	default:
		return "mdi:information"
	}
}
