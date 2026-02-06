package models

import "net"

// NetworkInterface represents a network interface on the server or agent host.
type NetworkInterface struct {
	Name       string   `json:"name"`
	Index      int      `json:"index"`
	MTU        int      `json:"mtu"`
	MACAddress string   `json:"mac_address"`
	Addresses  []string `json:"addresses"`
	IsUp       bool     `json:"is_up"`
	IsLoopback bool     `json:"is_loopback"`
}

// Subnet represents an IP subnet for scanning.
type Subnet struct {
	CIDR    string `json:"cidr"`
	Network net.IP `json:"-"`
	Mask    net.IP `json:"-"`
}

// ScanResult holds the result of a network scan.
type ScanResult struct {
	ID        string   `json:"id" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	Subnet    string   `json:"subnet" example:"192.168.1.0/24"`
	StartedAt string   `json:"started_at" example:"2026-01-15T10:30:00Z"`
	EndedAt   string   `json:"ended_at,omitempty" example:"2026-01-15T10:32:15Z"`
	Status    string   `json:"status" example:"completed"`
	Devices   []Device `json:"devices,omitempty"`
	Total     int      `json:"total" example:"12"`
	Online    int      `json:"online" example:"8"`
}

// AgentInfo represents the state of a connected Scout agent.
type AgentInfo struct {
	ID          string `json:"id" example:"agent-550e8400"`
	DeviceID    string `json:"device_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Version     string `json:"version" example:"0.1.0"`
	Status      string `json:"status" example:"connected"`
	LastCheckIn string `json:"last_check_in" example:"2026-01-15T10:30:00Z"`
	EnrolledAt  string `json:"enrolled_at" example:"2026-01-10T08:00:00Z"`
	Platform    string `json:"platform" example:"linux/amd64"`
}
