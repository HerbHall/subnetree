package recon

import (
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

// Event topics published by the Recon module.
const (
	TopicDeviceDiscovered = "recon.device.discovered"
	TopicDeviceUpdated    = "recon.device.updated"
	TopicDeviceLost       = "recon.device.lost"
	TopicScanStarted      = "recon.scan.started"
	TopicScanCompleted    = "recon.scan.completed"
	TopicScanProgress     = "recon.scan.progress"
)

// DeviceLostEvent is the payload for TopicDeviceLost events.
type DeviceLostEvent struct {
	DeviceID string    `json:"device_id"`
	IP       string    `json:"ip"`
	LastSeen time.Time `json:"last_seen"`
}

// DeviceEvent wraps a device with its scan ID for event payloads.
type DeviceEvent struct {
	ScanID string         `json:"scan_id"`
	Device *models.Device `json:"device"`
}

// ScanProgressEvent reports scan progress after the ping phase completes.
type ScanProgressEvent struct {
	ScanID     string `json:"scan_id"`
	HostsAlive int    `json:"hosts_alive"`
	SubnetSize int    `json:"subnet_size"`
}
