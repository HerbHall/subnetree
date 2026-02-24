package auth

// LoginRequest is the request body for POST /auth/login.
type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"securepassword123"`
}

// RefreshRequest is the request body for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJl..."`
}

// LogoutRequest is the request body for POST /auth/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJl..."`
}

// SetupRequest is the request body for POST /auth/setup.
type SetupRequest struct {
	Username string `json:"username" example:"admin"`
	Email    string `json:"email" example:"admin@example.com"`
	Password string `json:"password" example:"securepassword123"`
}

// UpdateUserRequest is the request body for PUT /users/{id}.
type UpdateUserRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Role     string `json:"role" example:"operator"`
	Disabled bool   `json:"disabled" example:"false"`
}

// SetupStatusResponse is the response for GET /auth/setup/status.
type SetupStatusResponse struct {
	SetupRequired bool   `json:"setup_required" example:"false"`
	Version       string `json:"version" example:"0.4.0"`
}

// MFAChallengeResponse is returned when MFA verification is required after password auth.
type MFAChallengeResponse struct {
	MFARequired bool   `json:"mfa_required" example:"true"`
	MFAToken    string `json:"mfa_token" example:"eyJhbG..."`
}

// MFAVerifyRequest is the request body for POST /auth/mfa/verify.
type MFAVerifyRequest struct {
	MFAToken string `json:"mfa_token" example:"eyJhbG..."`
	TOTPCode string `json:"totp_code" example:"123456"`
}

// MFARecoveryRequest is the request body for POST /auth/mfa/verify-recovery.
type MFARecoveryRequest struct {
	MFAToken     string `json:"mfa_token" example:"eyJhbG..."`
	RecoveryCode string `json:"recovery_code" example:"a1b2c3d4"`
}

// MFASetupResponse is the response from POST /auth/mfa/setup.
type MFASetupResponse struct {
	OTPAuthURL    string   `json:"otpauth_url" example:"otpauth://totp/SubNetree:admin?secret=..."`
	RecoveryCodes []string `json:"recovery_codes"`
}

// MFAVerifySetupRequest is the request body for POST /auth/mfa/verify-setup.
type MFAVerifySetupRequest struct {
	TOTPCode string `json:"totp_code" example:"123456"`
}

// MFADisableRequest is the request body for POST /auth/mfa/disable.
type MFADisableRequest struct {
	TOTPCode string `json:"totp_code" example:"123456"`
}
