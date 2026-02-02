package auth

import (
	"testing"
)

func TestHashPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery", 4) // low cost for speed
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if !CheckPassword(hash, "correct-horse-battery") {
		t.Error("CheckPassword should return true for correct password")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Error("CheckPassword should return false for incorrect password")
	}
}

func TestHashPassword_DefaultCost(t *testing.T) {
	hash, err := HashPassword("testpassword", 0) // 0 = use default
	if err != nil {
		t.Fatalf("HashPassword with default cost: %v", err)
	}
	if !CheckPassword(hash, "testpassword") {
		t.Error("CheckPassword should succeed with default cost hash")
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid 8 chars", "12345678", false},
		{"valid long", "a-very-secure-password", false},
		{"too short", "1234567", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestValidRoles(t *testing.T) {
	if !ValidRoles[RoleAdmin] {
		t.Error("admin should be a valid role")
	}
	if !ValidRoles[RoleOperator] {
		t.Error("operator should be a valid role")
	}
	if !ValidRoles[RoleViewer] {
		t.Error("viewer should be a valid role")
	}
	if ValidRoles[Role("superuser")] {
		t.Error("superuser should not be a valid role")
	}
}
