package services

import (
	"net"
	"strings"
)

// NetworkInterface represents a network interface with its properties.
type NetworkInterface struct {
	Name      string `json:"name" example:"eth0"`
	IPAddress string `json:"ip_address" example:"192.168.1.100"`
	Subnet    string `json:"subnet" example:"192.168.1.0/24"`
	MAC       string `json:"mac" example:"00:1a:2b:3c:4d:5e"`
	Status    string `json:"status" example:"up"` // "up" or "down"
}

// InterfaceService provides methods for network interface discovery.
type InterfaceService struct{}

// NewInterfaceService creates an InterfaceService instance.
func NewInterfaceService() *InterfaceService {
	return &InterfaceService{}
}

// ListNetworkInterfaces returns all available network interfaces
// that have at least one IPv4 address (excluding loopback).
func (s *InterfaceService) ListNetworkInterfaces() ([]NetworkInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	result := make([]NetworkInterface, 0, len(ifaces))
	for i := range ifaces {
		iface := &ifaces[i]

		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Get addresses for this interface
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// Find the first IPv4 address
		var ipAddr, subnet string
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue // Skip IPv6
			}
			ipAddr = ip.String()
			// Calculate subnet in CIDR notation
			ones, _ := ipNet.Mask.Size()
			subnet = ip.Mask(ipNet.Mask).String() + "/" + itoa(ones)
			break
		}

		// Skip interfaces without IPv4 addresses
		if ipAddr == "" {
			continue
		}

		// Determine status
		status := "down"
		if iface.Flags&net.FlagUp != 0 {
			status = "up"
		}

		// Format MAC address
		mac := formatMAC(iface.HardwareAddr)

		result = append(result, NetworkInterface{
			Name:      iface.Name,
			IPAddress: ipAddr,
			Subnet:    subnet,
			MAC:       mac,
			Status:    status,
		})
	}

	return result, nil
}

// itoa converts an integer to a string (simple implementation to avoid strconv import).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// formatMAC formats a hardware address as a colon-separated string.
func formatMAC(addr net.HardwareAddr) string {
	if len(addr) == 0 {
		return ""
	}
	parts := make([]string, len(addr))
	for i, b := range addr {
		parts[i] = byteToHex(b)
	}
	return strings.Join(parts, ":")
}

// byteToHex converts a byte to a two-character lowercase hex string.
func byteToHex(b byte) string {
	const hex = "0123456789abcdef"
	return string([]byte{hex[b>>4], hex[b&0x0f]})
}
