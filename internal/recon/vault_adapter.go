package recon

import (
	"context"
	"fmt"
)

// CredentialDecrypter retrieves decrypted credential data from the vault.
type CredentialDecrypter interface {
	DecryptCredential(ctx context.Context, id string) (map[string]any, error)
}

// VaultCredentialAdapter implements CredentialAccessor by using the vault
// to retrieve and decrypt SNMP credentials.
type VaultCredentialAdapter struct {
	decrypter CredentialDecrypter
}

// NewVaultCredentialAdapter creates a new adapter.
func NewVaultCredentialAdapter(dec CredentialDecrypter) *VaultCredentialAdapter {
	return &VaultCredentialAdapter{decrypter: dec}
}

// Compile-time interface guard.
var _ CredentialAccessor = (*VaultCredentialAdapter)(nil)

// GetCredential retrieves and parses an SNMP credential from the vault.
func (a *VaultCredentialAdapter) GetCredential(ctx context.Context, id string) (*SNMPCredential, error) {
	data, err := a.decrypter.DecryptCredential(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential %s: %w", id, err)
	}

	cred := &SNMPCredential{}

	// Parse type field.
	if t, ok := data["type"].(string); ok {
		cred.Type = t
	}

	switch cred.Type {
	case "snmp_v2c":
		if v, ok := data["community"].(string); ok {
			cred.Community = v
		}
	case "snmp_v3":
		if v, ok := data["username"].(string); ok {
			cred.Username = v
		}
		if v, ok := data["auth_protocol"].(string); ok {
			cred.AuthProtocol = v
		}
		if v, ok := data["auth_passphrase"].(string); ok {
			cred.AuthPassphrase = v
		}
		if v, ok := data["privacy_protocol"].(string); ok {
			cred.PrivacyProtocol = v
		}
		if v, ok := data["privacy_passphrase"].(string); ok {
			cred.PrivacyPassphrase = v
		}
		if v, ok := data["security_level"].(string); ok {
			cred.SecurityLevel = v
		}
		if v, ok := data["context_name"].(string); ok {
			cred.ContextName = v
		}
		if v, ok := data["authoritative_engine_id"].(string); ok {
			cred.AuthoritativeEngineID = v
		}
	default:
		return nil, fmt.Errorf("unsupported credential type: %s", cred.Type)
	}

	return cred, nil
}
