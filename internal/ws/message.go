package ws

import (
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

// MessageType discriminates WebSocket messages.
type MessageType string

const (
	MessageScanStarted     MessageType = "scan.started"
	MessageScanProgress    MessageType = "scan.progress"
	MessageScanDeviceFound MessageType = "scan.device_found"
	MessageScanCompleted   MessageType = "scan.completed"
	MessageScanError       MessageType = "scan.error"
)

// Message is the envelope for all WebSocket messages.
type Message struct {
	Type      MessageType `json:"type"`
	ScanID    string      `json:"scan_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      any         `json:"data"`
}

// ScanStartedData is the payload for scan.started messages.
type ScanStartedData struct {
	TargetCIDR string `json:"target_cidr"`
	Status     string `json:"status"`
}

// ScanProgressData is the payload for scan.progress messages.
type ScanProgressData struct {
	HostsAlive int `json:"hosts_alive"`
	SubnetSize int `json:"subnet_size"`
}

// ScanDeviceFoundData is the payload for scan.device_found messages.
type ScanDeviceFoundData struct {
	Device *models.Device `json:"device"`
}

// ScanCompletedData is the payload for scan.completed messages.
type ScanCompletedData struct {
	Total   int    `json:"total"`
	Online  int    `json:"online"`
	EndedAt string `json:"ended_at"`
}

// ScanErrorData is the payload for scan.error messages.
type ScanErrorData struct {
	Error string `json:"error"`
}
