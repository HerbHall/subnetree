package models

import "time"

// DeviceType categorizes a network device.
type DeviceType string

const (
	DeviceTypeServer  DeviceType = "server"
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeLaptop  DeviceType = "laptop"
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypeRouter  DeviceType = "router"
	DeviceTypeSwitch  DeviceType = "switch"
	DeviceTypePrinter DeviceType = "printer"
	DeviceTypeIoT     DeviceType = "iot"
	DeviceTypeUnknown DeviceType = "unknown"
)

// DeviceStatus represents the current state of a device.
type DeviceStatus string

const (
	DeviceStatusOnline   DeviceStatus = "online"
	DeviceStatusOffline  DeviceStatus = "offline"
	DeviceStatusDegraded DeviceStatus = "degraded"
	DeviceStatusUnknown  DeviceStatus = "unknown"
)

// DiscoveryMethod indicates how a device was discovered.
type DiscoveryMethod string

const (
	DiscoveryAgent  DiscoveryMethod = "agent"
	DiscoveryICMP   DiscoveryMethod = "icmp"
	DiscoveryARP    DiscoveryMethod = "arp"
	DiscoverySNMP   DiscoveryMethod = "snmp"
	DiscoverymDNS   DiscoveryMethod = "mdns"
	DiscoveryUPnP   DiscoveryMethod = "upnp"
	DiscoveryMQTT   DiscoveryMethod = "mqtt"
	DiscoveryManual DiscoveryMethod = "manual"
)

// Device represents a network device tracked by SubNetree.
type Device struct {
	ID              string            `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Hostname        string            `json:"hostname" example:"web-server-01"`
	IPAddresses     []string          `json:"ip_addresses"`
	MACAddress      string            `json:"mac_address,omitempty" example:"00:1a:2b:3c:4d:5e"`
	Manufacturer    string            `json:"manufacturer,omitempty" example:"Dell Inc."`
	DeviceType      DeviceType        `json:"device_type" example:"server"`
	OS              string            `json:"os,omitempty" example:"Ubuntu 22.04"`
	Status          DeviceStatus      `json:"status" example:"online"`
	DiscoveryMethod DiscoveryMethod   `json:"discovery_method" example:"icmp"`
	AgentID         string            `json:"agent_id,omitempty" example:"agent-01"`
	LastSeen        time.Time         `json:"last_seen" example:"2026-01-15T10:30:00Z"`
	FirstSeen       time.Time         `json:"first_seen" example:"2026-01-10T08:00:00Z"`
	Notes           string            `json:"notes,omitempty" example:"Production web server"`
	Tags            []string          `json:"tags,omitempty"`
	CustomFields    map[string]string `json:"custom_fields,omitempty"`
}
