package recon

import (
	"time"

	"github.com/google/uuid"
)

// ServiceMovement records when a network service (port+protocol) moves
// from one device to another between scans.
type ServiceMovement struct {
	ID          string    `json:"id"`
	Port        int       `json:"port"`
	Protocol    string    `json:"protocol"`
	ServiceName string    `json:"service_name"`
	FromDevice  string    `json:"from_device_id"`
	ToDevice    string    `json:"to_device_id"`
	DetectedAt  time.Time `json:"detected_at"`
}

// wellKnownServices maps common ports to human-readable service names.
var wellKnownServices = map[int]string{
	20:   "ftp-data",
	21:   "ftp",
	22:   "ssh",
	25:   "smtp",
	53:   "dns",
	67:   "dhcp",
	68:   "dhcp-client",
	80:   "http",
	110:  "pop3",
	123:  "ntp",
	143:  "imap",
	161:  "snmp",
	162:  "snmp-trap",
	443:  "https",
	445:  "smb",
	993:  "imaps",
	995:  "pop3s",
	3306: "mysql",
	3389: "rdp",
	5432: "postgres",
	5900: "vnc",
	6379: "redis",
	8080: "http-alt",
	8443: "https-alt",
	9090: "prometheus",
}

// lookupServiceName returns a human-readable name for a well-known port,
// or an empty string if the port is not recognized.
func lookupServiceName(port int) string {
	return wellKnownServices[port]
}

// detectServiceMovements compares the previous and current scan's service
// maps and returns movements where a port disappeared from exactly one
// device and appeared on exactly one other device.
//
// Each map entry is device_id -> list of open ports. Only clear 1:1
// movements are flagged; ports that appear on multiple new devices
// (replication) or simply disappear are not considered movements.
func detectServiceMovements(previous, current map[string][]int) []ServiceMovement {
	// Build port -> set-of-devices for previous and current scans.
	prevDevices := portToDeviceSet(previous)
	currDevices := portToDeviceSet(current)

	var movements []ServiceMovement

	for port, prevSet := range prevDevices {
		currSet := currDevices[port]

		// Find devices that had the port before but not now.
		var removed []string
		for deviceID := range prevSet {
			if !currSet[deviceID] {
				removed = append(removed, deviceID)
			}
		}

		// Find devices that have the port now but didn't before.
		var added []string
		for deviceID := range currSet {
			if !prevSet[deviceID] {
				added = append(added, deviceID)
			}
		}

		// Only flag as movement: exactly one removed AND exactly one added.
		if len(removed) == 1 && len(added) == 1 {
			movements = append(movements, ServiceMovement{
				ID:          uuid.New().String(),
				Port:        port,
				Protocol:    "tcp",
				ServiceName: lookupServiceName(port),
				FromDevice:  removed[0],
				ToDevice:    added[0],
				DetectedAt:  time.Now().UTC(),
			})
		}
	}

	return movements
}

// portToDeviceSet builds a map of port -> set-of-device-IDs.
func portToDeviceSet(serviceMap map[string][]int) map[int]map[string]bool {
	result := make(map[int]map[string]bool)
	for deviceID, ports := range serviceMap {
		for _, port := range ports {
			if result[port] == nil {
				result[port] = make(map[string]bool)
			}
			result[port][deviceID] = true
		}
	}
	return result
}
