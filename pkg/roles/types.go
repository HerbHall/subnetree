package roles

import "time"

// MonitorStatus represents the health/monitoring state of a device.
type MonitorStatus struct {
	DeviceID  string    `json:"device_id"`
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// Credential represents a stored credential (opaque to callers).
type Credential struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // "ssh", "snmp_v2", "snmp_v3", "api_key", etc.
	DeviceID string `json:"device_id,omitempty"`
}

// Notification represents a message to be delivered by a Notifier.
type Notification struct {
	Topic   string         `json:"topic"`
	Summary string         `json:"summary"`
	Body    string         `json:"body,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
}
