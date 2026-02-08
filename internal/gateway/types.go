package gateway

import (
	"sync/atomic"
	"time"
)

// SessionType represents the kind of remote access session.
type SessionType string

// Supported session types.
const (
	SessionTypeProxy SessionType = "http_proxy"
	SessionTypeSSH   SessionType = "ssh"
)

// Session represents an active remote access session.
type Session struct {
	ID          string      `json:"id"`
	DeviceID    string      `json:"device_id"`
	UserID      string      `json:"user_id"`
	SessionType SessionType `json:"session_type"`
	Target      ProxyTarget `json:"target"`
	SourceIP    string      `json:"source_ip"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   time.Time   `json:"expires_at"`

	// Thread-safe byte counters updated by proxy goroutines.
	BytesIn  atomic.Int64 `json:"-"`
	BytesOut atomic.Int64 `json:"-"`
}

// BytesInCount returns the current inbound byte count.
func (s *Session) BytesInCount() int64 {
	return s.BytesIn.Load()
}

// BytesOutCount returns the current outbound byte count.
func (s *Session) BytesOutCount() int64 {
	return s.BytesOut.Load()
}

// ProxyTarget describes the target host and port for a session.
type ProxyTarget struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// AuditEntry records a gateway access event.
type AuditEntry struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	DeviceID    string    `json:"device_id"`
	UserID      string    `json:"user_id"`
	SessionType string    `json:"session_type"`
	Target      string    `json:"target"`
	Action      string    `json:"action"`
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	SourceIP    string    `json:"source_ip"`
	Timestamp   time.Time `json:"timestamp"`
}

// sessionView is the JSON-serializable representation of a Session.
// It includes byte counters that are otherwise hidden via json:"-" on atomics.
type sessionView struct {
	ID          string      `json:"id"`
	DeviceID    string      `json:"device_id"`
	UserID      string      `json:"user_id"`
	SessionType SessionType `json:"session_type"`
	Target      ProxyTarget `json:"target"`
	SourceIP    string      `json:"source_ip"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
	BytesIn     int64       `json:"bytes_in"`
	BytesOut    int64       `json:"bytes_out"`
}

// toView converts a Session to its JSON-serializable form.
func (s *Session) toView() sessionView {
	return sessionView{
		ID:          s.ID,
		DeviceID:    s.DeviceID,
		UserID:      s.UserID,
		SessionType: s.SessionType,
		Target:      s.Target,
		SourceIP:    s.SourceIP,
		CreatedAt:   s.CreatedAt,
		ExpiresAt:   s.ExpiresAt,
		BytesIn:     s.BytesInCount(),
		BytesOut:    s.BytesOutCount(),
	}
}
