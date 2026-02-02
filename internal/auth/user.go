package auth

import (
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role represents user authorization levels.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

// ValidRoles contains all valid role values.
var ValidRoles = map[Role]bool{
	RoleAdmin:    true,
	RoleOperator: true,
	RoleViewer:   true,
}

// User represents a NetVantage user account.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialized
	Role         Role      `json:"role"`
	AuthProvider string    `json:"auth_provider"`
	OIDCSubject  string    `json:"oidc_subject,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
	Disabled     bool      `json:"disabled"`
}

// HashPassword creates a bcrypt hash of the given password.
func HashPassword(password string, cost int) (string, error) {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword verifies a password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ValidatePassword checks that a password meets minimum requirements.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}
