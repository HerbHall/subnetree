package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
)

// TOTPService handles TOTP secret management, encryption, and MFA token issuance.
type TOTPService struct {
	key    []byte // 32-byte AES-GCM key derived from JWT secret
	secret []byte // raw JWT secret for MFA token signing
}

// NewTOTPService creates a TOTPService using the JWT secret for key derivation.
func NewTOTPService(jwtSecret []byte) *TOTPService {
	h := sha256.Sum256(jwtSecret)
	return &TOTPService{key: h[:], secret: jwtSecret}
}

// GenerateSecret creates a new TOTP secret and returns the raw secret and otpauth URL.
func (t *TOTPService) GenerateSecret(accountName, issuer string) (secret, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate TOTP key: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// Validate checks a TOTP code against a secret.
func (t *TOTPService) Validate(code, secret string) bool {
	return totp.Validate(code, secret)
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded ciphertext.
func (t *TOTPService) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(t.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded AES-256-GCM ciphertext.
func (t *TOTPService) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(t.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, encrypted := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GenerateRecoveryCodes creates n random 8-character alphanumeric recovery codes.
// Returns the plaintext codes (to show to the user) and their SHA-256 hashes (to store).
func (t *TOTPService) GenerateRecoveryCodes(n int) (plain, hashed []string, err error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const codeLen = 8

	plain = make([]string, n)
	hashed = make([]string, n)

	for i := 0; i < n; i++ {
		b := make([]byte, codeLen)
		if _, err := io.ReadFull(rand.Reader, b); err != nil {
			return nil, nil, fmt.Errorf("generate recovery code: %w", err)
		}
		for j := range b {
			b[j] = charset[b[j]%byte(len(charset))]
		}
		code := string(b)
		plain[i] = code
		h := sha256.Sum256([]byte(code))
		hashed[i] = hex.EncodeToString(h[:])
	}

	return plain, hashed, nil
}

// mfaClaims are JWT claims for short-lived MFA tokens.
type mfaClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
	MFA    bool   `json:"mfa"`
}

// IssueMFAToken creates a short-lived JWT indicating MFA is required.
func (t *TOTPService) IssueMFAToken(userID string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := mfaClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "subnetree-mfa",
		},
		UserID: userID,
		MFA:    true,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	if err != nil {
		return "", fmt.Errorf("sign MFA token: %w", err)
	}
	return signed, nil
}

// ValidateMFAToken parses and validates an MFA token, returning the user ID.
func (t *TOTPService) ValidateMFAToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &mfaClaims{}, func(_ *jwt.Token) (interface{}, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return "", fmt.Errorf("parse MFA token: %w", err)
	}

	claims, ok := token.Claims.(*mfaClaims)
	if !ok || !token.Valid || !claims.MFA {
		return "", errors.New("invalid MFA token claims")
	}

	return claims.UserID, nil
}
