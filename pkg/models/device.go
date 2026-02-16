package models

import "time"

// DeviceType categorizes a network device.
type DeviceType string

const (
	DeviceTypeServer      DeviceType = "server"
	DeviceTypeDesktop     DeviceType = "desktop"
	DeviceTypeLaptop      DeviceType = "laptop"
	DeviceTypeMobile      DeviceType = "mobile"
	DeviceTypeRouter      DeviceType = "router"
	DeviceTypeSwitch      DeviceType = "switch"
	DeviceTypePrinter     DeviceType = "printer"
	DeviceTypeIoT         DeviceType = "iot"
	DeviceTypeAccessPoint DeviceType = "access_point"
	DeviceTypeFirewall    DeviceType = "firewall"
	DeviceTypeNAS         DeviceType = "nas"
	DeviceTypePhone       DeviceType = "phone"
	DeviceTypeTablet      DeviceType = "tablet"
	DeviceTypeCamera      DeviceType = "camera"
	DeviceTypeUnknown     DeviceType = "unknown"
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
	Location        string            `json:"location,omitempty" example:"Rack A3, U12"`
	Category        string            `json:"category,omitempty" example:"production"`
	PrimaryRole     string            `json:"primary_role,omitempty" example:"web-server"`
	Owner           string            `json:"owner,omitempty" example:"platform-team"`

	// Classification metadata from the composite classifier.
	ClassificationConfidence int    `json:"classification_confidence,omitempty" example:"75"`
	ClassificationSource     string `json:"classification_source,omitempty" example:"snmp_bridge_mib"`
	ClassificationSignals    string `json:"classification_signals,omitempty"` // JSON-encoded signal breakdown

	// Network hierarchy metadata from hierarchy inference.
	ParentDeviceID string `json:"parent_device_id,omitempty"`
	NetworkLayer   int    `json:"network_layer,omitempty" example:"4"` // 0=unknown, 1=gateway, 2=distribution, 3=access, 4=endpoint
}

// Network layer constants for hierarchy inference.
const (
	NetworkLayerUnknown      = 0 // Unclassified
	NetworkLayerGateway      = 1 // Routers, firewalls
	NetworkLayerDistribution = 2 // L3 switches, core switches
	NetworkLayerAccess       = 3 // L2 switches, APs
	NetworkLayerEndpoint     = 4 // Servers, desktops, IoT, etc.
)
