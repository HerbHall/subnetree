package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT payload for access tokens.
type Claims struct {
	jwt.RegisteredClaims
	UserID   string `json:"uid"`
	Username string `json:"usr"`
	Role     string `json:"role"`
}

// TokenService handles JWT access tokens and refresh token generation.
type TokenService struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewTokenService creates a TokenService with the given signing secret and TTLs.
func NewTokenService(secret []byte, accessTTL, refreshTTL time.Duration) *TokenService {
	return &TokenService{
		secret:          secret,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// IssueAccessToken generates a signed JWT access token for the given user.
func (s *TokenService) IssueAccessToken(user *User) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
			Issuer:    "netvantage",
		},
		UserID:   user.ID,
		Username: user.Username,
		Role:     string(user.Role),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// ValidateAccessToken parses and validates a JWT access token, returning the claims.
func (s *TokenService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (interface{}, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// GenerateRefreshToken creates a cryptographically random refresh token
// and returns both the raw token (sent to client) and its SHA-256 hash (stored in DB).
func (s *TokenService) GenerateRefreshToken() (raw string, hash string, expiresAt time.Time, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}
	raw = hex.EncodeToString(b)
	hash = HashToken(raw)
	expiresAt = time.Now().Add(s.refreshTokenTTL)
	return raw, hash, expiresAt, nil
}

// RefreshTokenTTL returns the configured refresh token lifetime.
func (s *TokenService) RefreshTokenTTL() time.Duration {
	return s.refreshTokenTTL
}

// AccessTokenTTL returns the configured access token lifetime.
func (s *TokenService) AccessTokenTTL() time.Duration {
	return s.accessTokenTTL
}

// HashToken returns the SHA-256 hex hash of a token string.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
