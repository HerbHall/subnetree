package vault

import (
	"fmt"
	"strings"
)

const maxCredentialNameLen = 255

// ValidateCredentialType checks that the type string is recognized.
func ValidateCredentialType(credType string) error {
	if !ValidCredentialTypes[credType] {
		return fmt.Errorf("unsupported credential type: %q", credType)
	}
	return nil
}

// ValidateCredentialName checks that the name is non-empty and within limits.
func ValidateCredentialName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("credential name must not be empty")
	}
	if len(trimmed) > maxCredentialNameLen {
		return fmt.Errorf("credential name exceeds %d characters", maxCredentialNameLen)
	}
	return nil
}

// ValidateCredentialData validates that the data map has the required
// fields for the given credential type.
func ValidateCredentialData(credType string, data map[string]any) error {
	if data == nil {
		return fmt.Errorf("credential data must not be nil")
	}

	switch credType {
	case CredTypeSSHPassword:
		return requireStringFields(data, "username", "password")
	case CredTypeSSHKey:
		return requireStringFields(data, "username", "private_key")
	case CredTypeSNMPv2c:
		return requireStringFields(data, "community")
	case CredTypeSNMPv3:
		return requireStringFields(data, "username", "auth_protocol", "auth_key", "security_level")
	case CredTypeAPIKey:
		return requireStringFields(data, "key")
	case CredTypeHTTPBasic:
		return requireStringFields(data, "username", "password")
	case CredTypeCustom:
		if _, ok := data["fields"]; !ok {
			return fmt.Errorf("custom credential requires %q field", "fields")
		}
		if _, ok := data["fields"].(map[string]any); !ok {
			return fmt.Errorf("custom credential %q must be a map", "fields")
		}
		return nil
	default:
		return fmt.Errorf("unsupported credential type: %q", credType)
	}
}

// requireStringFields checks that all named keys exist in data and are
// non-empty strings.
func requireStringFields(data map[string]any, fields ...string) error {
	for _, f := range fields {
		v, ok := data[f]
		if !ok {
			return fmt.Errorf("missing required field %q", f)
		}
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("field %q must be a string", f)
		}
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("field %q must not be empty", f)
		}
	}
	return nil
}
