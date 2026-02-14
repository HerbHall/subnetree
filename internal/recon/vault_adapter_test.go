package recon

import (
	"context"
	"fmt"
	"testing"
)

// mockDecrypter implements CredentialDecrypter for testing.
type mockDecrypter struct {
	data map[string]any
	err  error
}

func (m *mockDecrypter) DecryptCredential(_ context.Context, _ string) (map[string]any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func TestVaultCredentialAdapter_SNMPv2c(t *testing.T) {
	dec := &mockDecrypter{
		data: map[string]any{
			"type":      "snmp_v2c",
			"community": "public",
		},
	}

	adapter := NewVaultCredentialAdapter(dec)
	cred, err := adapter.GetCredential(context.Background(), "cred-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cred.Type != "snmp_v2c" {
		t.Errorf("type = %q, want %q", cred.Type, "snmp_v2c")
	}
	if cred.Community != "public" {
		t.Errorf("community = %q, want %q", cred.Community, "public")
	}
}

func TestVaultCredentialAdapter_SNMPv3(t *testing.T) {
	dec := &mockDecrypter{
		data: map[string]any{
			"type":                     "snmp_v3",
			"username":                 "admin",
			"auth_protocol":            "SHA-256",
			"auth_passphrase":          "secret123",
			"privacy_protocol":         "AES-256",
			"privacy_passphrase":       "privpass",
			"security_level":           "authPriv",
			"context_name":             "ctx1",
			"authoritative_engine_id":  "engine01",
		},
	}

	adapter := NewVaultCredentialAdapter(dec)
	cred, err := adapter.GetCredential(context.Background(), "cred-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cred.Type != "snmp_v3" {
		t.Errorf("type = %q, want %q", cred.Type, "snmp_v3")
	}
	if cred.Username != "admin" {
		t.Errorf("username = %q, want %q", cred.Username, "admin")
	}
	if cred.AuthProtocol != "SHA-256" {
		t.Errorf("auth_protocol = %q, want %q", cred.AuthProtocol, "SHA-256")
	}
	if cred.AuthPassphrase != "secret123" {
		t.Errorf("auth_passphrase = %q, want %q", cred.AuthPassphrase, "secret123")
	}
	if cred.PrivacyProtocol != "AES-256" {
		t.Errorf("privacy_protocol = %q, want %q", cred.PrivacyProtocol, "AES-256")
	}
	if cred.PrivacyPassphrase != "privpass" {
		t.Errorf("privacy_passphrase = %q, want %q", cred.PrivacyPassphrase, "privpass")
	}
	if cred.SecurityLevel != "authPriv" {
		t.Errorf("security_level = %q, want %q", cred.SecurityLevel, "authPriv")
	}
	if cred.ContextName != "ctx1" {
		t.Errorf("context_name = %q, want %q", cred.ContextName, "ctx1")
	}
	if cred.AuthoritativeEngineID != "engine01" {
		t.Errorf("authoritative_engine_id = %q, want %q", cred.AuthoritativeEngineID, "engine01")
	}
}

func TestVaultCredentialAdapter_UnsupportedType(t *testing.T) {
	dec := &mockDecrypter{
		data: map[string]any{
			"type": "ssh_key",
		},
	}

	adapter := NewVaultCredentialAdapter(dec)
	_, err := adapter.GetCredential(context.Background(), "cred-3")
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}

func TestVaultCredentialAdapter_DecryptError(t *testing.T) {
	dec := &mockDecrypter{
		err: fmt.Errorf("vault is sealed"),
	}

	adapter := NewVaultCredentialAdapter(dec)
	_, err := adapter.GetCredential(context.Background(), "cred-4")
	if err == nil {
		t.Fatal("expected error from decrypter, got nil")
	}
}

func TestVaultCredentialAdapter_InterfaceGuard(t *testing.T) {
	// Verify compile-time interface guard works.
	var _ CredentialAccessor = (*VaultCredentialAdapter)(nil)
}
