package auth

import (
	"testing"
	"time"
)

func newTestTokenService() *TokenService {
	return NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
}

func newTestUser() *User {
	return &User{
		ID:       "user-123",
		Username: "alice",
		Email:    "alice@example.com",
		Role:     RoleAdmin,
	}
}

func TestIssueAndValidateAccessToken(t *testing.T) {
	ts := newTestTokenService()
	user := newTestUser()

	token, err := ts.IssueAccessToken(user)
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := ts.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID = %q, want %q", claims.UserID, user.ID)
	}
	if claims.Username != user.Username {
		t.Errorf("Username = %q, want %q", claims.Username, user.Username)
	}
	if claims.Role != string(user.Role) {
		t.Errorf("Role = %q, want %q", claims.Role, string(user.Role))
	}
	if claims.Issuer != "netvantage" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "netvantage")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	ts1 := NewTokenService([]byte("secret-one-is-32-bytes-long!!!!"), 15*time.Minute, 7*24*time.Hour)
	ts2 := NewTokenService([]byte("secret-two-is-32-bytes-long!!!!"), 15*time.Minute, 7*24*time.Hour)

	token, err := ts1.IssueAccessToken(newTestUser())
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	_, err = ts2.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error validating token with wrong secret")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	ts := NewTokenService([]byte("test-secret-key-32bytes-long!!"), -1*time.Second, 7*24*time.Hour)
	token, err := ts.IssueAccessToken(newTestUser())
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	_, err = ts.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestValidateAccessToken_Garbage(t *testing.T) {
	ts := newTestTokenService()
	_, err := ts.ValidateAccessToken("not.a.jwt")
	if err == nil {
		t.Error("expected error for garbage token")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	ts := newTestTokenService()
	raw, hash, expiresAt, err := ts.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if raw == "" {
		t.Error("expected non-empty raw token")
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if expiresAt.Before(time.Now()) {
		t.Error("expiresAt should be in the future")
	}

	// Hash should be deterministic for the same input.
	if HashToken(raw) != hash {
		t.Error("HashToken(raw) should match the returned hash")
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	ts := newTestTokenService()
	raw1, _, _, _ := ts.GenerateRefreshToken()
	raw2, _, _, _ := ts.GenerateRefreshToken()
	if raw1 == raw2 {
		t.Error("two generated refresh tokens should be different")
	}
}

func TestTokenServiceTTLs(t *testing.T) {
	ts := newTestTokenService()
	if ts.AccessTokenTTL() != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 15m", ts.AccessTokenTTL())
	}
	if ts.RefreshTokenTTL() != 7*24*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want 168h", ts.RefreshTokenTTL())
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	input := "some-token-value"
	h1 := HashToken(input)
	h2 := HashToken(input)
	if h1 != h2 {
		t.Error("HashToken should be deterministic")
	}
	if h1 == input {
		t.Error("HashToken should not return the original input")
	}
}
