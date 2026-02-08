package vault

import "time"

// Supported credential types.
const (
	CredTypeSSHPassword = "ssh_password"
	CredTypeSSHKey      = "ssh_key"
	CredTypeSNMPv2c     = "snmp_v2c"
	CredTypeSNMPv3      = "snmp_v3"
	CredTypeAPIKey      = "api_key"
	CredTypeHTTPBasic   = "http_basic"
	CredTypeCustom      = "custom"
)

// ValidCredentialTypes is the set of recognized credential type strings.
var ValidCredentialTypes = map[string]bool{
	CredTypeSSHPassword: true,
	CredTypeSSHKey:      true,
	CredTypeSNMPv2c:     true,
	CredTypeSNMPv3:      true,
	CredTypeAPIKey:      true,
	CredTypeHTTPBasic:   true,
	CredTypeCustom:      true,
}

// CredentialRecord is the full database representation of a stored credential.
type CredentialRecord struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	DeviceID      string    `json:"device_id,omitempty"`
	Description   string    `json:"description,omitempty"`
	EncryptedData []byte    `json:"-"` // AES-256-GCM encrypted credential data
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CredentialMeta is the public-facing metadata (never contains secrets).
type CredentialMeta struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	DeviceID    string    `json:"device_id,omitempty"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CredentialData holds decrypted secret data alongside metadata.
type CredentialData struct {
	CredentialMeta
	Data map[string]any `json:"data"`
}

// CredentialKey stores the wrapped DEK for a credential.
type CredentialKey struct {
	CredentialID string `json:"credential_id"`
	WrappedKey   []byte `json:"wrapped_key"`
}

// MasterKeyRecord stores the salt and verification blob for the master key.
type MasterKeyRecord struct {
	ID               int       `json:"id"`
	Salt             []byte    `json:"salt"`
	VerificationBlob []byte    `json:"verification_blob"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// AuditEntry records a credential access event.
type AuditEntry struct {
	ID           int64     `json:"id"`
	CredentialID string    `json:"credential_id"`
	UserID       string    `json:"user_id"`
	Action       string    `json:"action"` // "create", "read", "update", "delete", "rotate_keys"
	Purpose      string    `json:"purpose,omitempty"`
	SourceIP     string    `json:"source_ip,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}
