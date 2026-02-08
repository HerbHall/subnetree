package vault

import (
	"strings"
	"testing"
)

func TestValidateCredentialType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"ssh_password", CredTypeSSHPassword, false},
		{"ssh_key", CredTypeSSHKey, false},
		{"snmp_v2c", CredTypeSNMPv2c, false},
		{"snmp_v3", CredTypeSNMPv3, false},
		{"api_key", CredTypeAPIKey, false},
		{"http_basic", CredTypeHTTPBasic, false},
		{"custom", CredTypeCustom, false},
		{"unknown", "ftp_password", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentialType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentialName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid", "Router SSH Creds", false},
		{"short", "x", false},
		{"empty", "", true},
		{"whitespace_only", "   ", true},
		{"too_long", strings.Repeat("a", 256), true},
		{"max_length", strings.Repeat("a", 255), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentialName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentialData_SSHPassword(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{"valid", map[string]any{"username": "admin", "password": "secret"}, false},
		{"missing_username", map[string]any{"password": "secret"}, true},
		{"missing_password", map[string]any{"username": "admin"}, true},
		{"empty_username", map[string]any{"username": "", "password": "secret"}, true},
		{"empty_password", map[string]any{"username": "admin", "password": ""}, true},
		{"non_string", map[string]any{"username": 123, "password": "secret"}, true},
		{"nil_data", nil, true},
		{"extra_fields_ok", map[string]any{"username": "admin", "password": "secret", "port": 22}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialData(CredTypeSSHPassword, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentialData_SSHKey(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{"valid", map[string]any{"username": "admin", "private_key": "-----BEGIN..."}, false},
		{"missing_key", map[string]any{"username": "admin"}, true},
		{"missing_username", map[string]any{"private_key": "-----BEGIN..."}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialData(CredTypeSSHKey, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentialData_SNMPv2c(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{"valid", map[string]any{"community": "public"}, false},
		{"missing", map[string]any{}, true},
		{"empty", map[string]any{"community": ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialData(CredTypeSNMPv2c, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentialData_SNMPv3(t *testing.T) {
	valid := map[string]any{
		"username":       "snmpuser",
		"auth_protocol":  "SHA",
		"auth_key":       "authpass123",
		"security_level": "authPriv",
	}

	if err := ValidateCredentialData(CredTypeSNMPv3, valid); err != nil {
		t.Errorf("valid SNMPv3 data returned error: %v", err)
	}

	for _, missing := range []string{"username", "auth_protocol", "auth_key", "security_level"} {
		data := make(map[string]any)
		for k, v := range valid {
			data[k] = v
		}
		delete(data, missing)

		if err := ValidateCredentialData(CredTypeSNMPv3, data); err == nil {
			t.Errorf("SNMPv3 missing %q should return error", missing)
		}
	}
}

func TestValidateCredentialData_APIKey(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{"valid", map[string]any{"key": "abc-123-xyz"}, false},
		{"missing", map[string]any{}, true},
		{"empty", map[string]any{"key": ""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialData(CredTypeAPIKey, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentialData_HTTPBasic(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{"valid", map[string]any{"username": "admin", "password": "pass"}, false},
		{"missing_password", map[string]any{"username": "admin"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialData(CredTypeHTTPBasic, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentialData_Custom(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{"valid", map[string]any{"fields": map[string]any{"host": "example.com"}}, false},
		{"missing_fields", map[string]any{}, true},
		{"fields_not_map", map[string]any{"fields": "not a map"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentialData(CredTypeCustom, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentialData_UnknownType(t *testing.T) {
	err := ValidateCredentialData("ftp", map[string]any{"foo": "bar"})
	if err == nil {
		t.Error("unknown type should return error")
	}
}
