package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func newTestTOTPService() *TOTPService {
	return NewTOTPService([]byte("test-secret-key-32bytes-long!!"))
}

func TestGenerateSecret(t *testing.T) {
	svc := newTestTOTPService()

	tests := []struct {
		name        string
		account     string
		issuer      string
		wantSecret  bool
		wantURL     bool
	}{
		{
			name:       "basic generation",
			account:    "admin",
			issuer:     "SubNetree",
			wantSecret: true,
			wantURL:    true,
		},
		{
			name:       "with email account",
			account:    "user@example.com",
			issuer:     "SubNetree",
			wantSecret: true,
			wantURL:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			secret, url, err := svc.GenerateSecret(tc.account, tc.issuer)
			if err != nil {
				t.Fatalf("GenerateSecret: %v", err)
			}
			if tc.wantSecret && secret == "" {
				t.Error("expected non-empty secret")
			}
			if tc.wantURL && url == "" {
				t.Error("expected non-empty URL")
			}
			if tc.wantURL && len(url) > 10 {
				if url[:10] != "otpauth://" {
					t.Errorf("URL should start with otpauth://, got %q", url[:10])
				}
			}
		})
	}
}

func TestValidateTOTP(t *testing.T) {
	svc := newTestTOTPService()

	secret, _, err := svc.GenerateSecret("admin", "SubNetree")
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}

	// Generate a valid code for the secret.
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	tests := []struct {
		name   string
		code   string
		secret string
		want   bool
	}{
		{
			name:   "valid code",
			code:   code,
			secret: secret,
			want:   true,
		},
		{
			name:   "wrong code",
			code:   "000000",
			secret: secret,
			want:   false,
		},
		{
			name:   "empty code",
			code:   "",
			secret: secret,
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.Validate(tc.code, tc.secret)
			if got != tc.want {
				t.Errorf("Validate(%q) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	svc := newTestTOTPService()

	tests := []struct {
		name      string
		plaintext string
	}{
		{name: "simple string", plaintext: "hello world"},
		{name: "TOTP secret", plaintext: "JBSWY3DPEHPK3PXP"},
		{name: "empty string", plaintext: ""},
		{name: "special chars", plaintext: "p@$$w0rd!#%^&*()"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := svc.Encrypt(tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}

			if encrypted == tc.plaintext {
				t.Error("encrypted should differ from plaintext")
			}

			decrypted, err := svc.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("Decrypt = %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestEncryptDecrypt_DifferentCiphertexts(t *testing.T) {
	svc := newTestTOTPService()

	// Same plaintext should produce different ciphertexts (random nonce).
	c1, err := svc.Encrypt("same input")
	if err != nil {
		t.Fatalf("first Encrypt: %v", err)
	}
	c2, err := svc.Encrypt("same input")
	if err != nil {
		t.Fatalf("second Encrypt: %v", err)
	}

	if c1 == c2 {
		t.Error("expected different ciphertexts for same plaintext (random nonce)")
	}
}

func TestDecrypt_InvalidInput(t *testing.T) {
	svc := newTestTOTPService()

	tests := []struct {
		name       string
		ciphertext string
	}{
		{name: "not base64", ciphertext: "not-valid-base64!@#"},
		{name: "too short", ciphertext: "aGVsbG8="},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Decrypt(tc.ciphertext)
			if err == nil {
				t.Error("expected error for invalid ciphertext")
			}
		})
	}
}

func TestGenerateRecoveryCodes(t *testing.T) {
	svc := newTestTOTPService()

	tests := []struct {
		name string
		n    int
	}{
		{name: "10 codes", n: 10},
		{name: "5 codes", n: 5},
		{name: "1 code", n: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plain, hashed, err := svc.GenerateRecoveryCodes(tc.n)
			if err != nil {
				t.Fatalf("GenerateRecoveryCodes: %v", err)
			}

			if len(plain) != tc.n {
				t.Errorf("plain count = %d, want %d", len(plain), tc.n)
			}
			if len(hashed) != tc.n {
				t.Errorf("hashed count = %d, want %d", len(hashed), tc.n)
			}

			// Check uniqueness of plain codes.
			seen := make(map[string]bool)
			for _, code := range plain {
				if len(code) != 8 {
					t.Errorf("code length = %d, want 8", len(code))
				}
				if seen[code] {
					t.Errorf("duplicate code: %s", code)
				}
				seen[code] = true
			}

			// Check hashed codes are valid SHA-256 hex strings (64 chars).
			for _, h := range hashed {
				if len(h) != 64 {
					t.Errorf("hash length = %d, want 64", len(h))
				}
			}
		})
	}
}

func TestMFAToken(t *testing.T) {
	svc := newTestTOTPService()

	token, err := svc.IssueMFAToken("user-123", 5*time.Minute)
	if err != nil {
		t.Fatalf("IssueMFAToken: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	userID, err := svc.ValidateMFAToken(token)
	if err != nil {
		t.Fatalf("ValidateMFAToken: %v", err)
	}

	if userID != "user-123" {
		t.Errorf("userID = %q, want %q", userID, "user-123")
	}
}

func TestMFATokenExpired(t *testing.T) {
	svc := newTestTOTPService()

	// Issue a token that has already expired.
	token, err := svc.IssueMFAToken("user-123", -1*time.Second)
	if err != nil {
		t.Fatalf("IssueMFAToken: %v", err)
	}

	_, err = svc.ValidateMFAToken(token)
	if err == nil {
		t.Error("expected error for expired MFA token")
	}
}

func TestMFAToken_InvalidString(t *testing.T) {
	svc := newTestTOTPService()

	_, err := svc.ValidateMFAToken("totally-invalid-token")
	if err == nil {
		t.Error("expected error for invalid MFA token")
	}
}
