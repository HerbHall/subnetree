package auth

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/netvantage/internal/store"
)

// testEnv sets up an in-memory database with auth migrations and returns
// the UserStore, TokenService, and Service for testing.
func testEnv(t *testing.T) (*UserStore, *TokenService, *Service) {
	t.Helper()

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	userStore, err := NewUserStore(ctx, db)
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}

	tokens := NewTokenService([]byte("test-secret-key-32bytes-long!!"), 15*time.Minute, 7*24*time.Hour)
	svc := NewService(userStore, tokens, testLogger())
	return userStore, tokens, svc
}

func TestSetup_CreatesAdmin(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	needs, err := svc.NeedsSetup(ctx)
	if err != nil {
		t.Fatalf("NeedsSetup: %v", err)
	}
	if !needs {
		t.Fatal("expected NeedsSetup=true before any users created")
	}

	user, err := svc.Setup(ctx, "admin", "admin@example.com", "securepassword")
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if user.Role != RoleAdmin {
		t.Errorf("user.Role = %q, want admin", user.Role)
	}
	if user.Username != "admin" {
		t.Errorf("user.Username = %q, want admin", user.Username)
	}

	needs, err = svc.NeedsSetup(ctx)
	if err != nil {
		t.Fatalf("NeedsSetup after setup: %v", err)
	}
	if needs {
		t.Error("expected NeedsSetup=false after setup")
	}
}

func TestSetup_OnlyOnce(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	_, err := svc.Setup(ctx, "admin", "admin@example.com", "securepassword")
	if err != nil {
		t.Fatalf("first Setup: %v", err)
	}

	_, err = svc.Setup(ctx, "admin2", "admin2@example.com", "securepassword")
	if err != ErrSetupComplete {
		t.Errorf("second Setup err = %v, want ErrSetupComplete", err)
	}
}

func TestSetup_WeakPassword(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	_, err := svc.Setup(ctx, "admin", "admin@example.com", "short")
	if err == nil {
		t.Error("expected error for short password")
	}
}

func TestLogin_Success(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	_, err := svc.Setup(ctx, "admin", "admin@example.com", "securepassword")
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	pair, err := svc.Login(ctx, "admin", "securepassword")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if pair.ExpiresIn <= 0 {
		t.Error("expected positive ExpiresIn")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	_, _ = svc.Setup(ctx, "admin", "admin@example.com", "securepassword")

	_, err := svc.Login(ctx, "admin", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Login err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	_, err := svc.Login(ctx, "nobody", "password")
	if err != ErrInvalidCredentials {
		t.Errorf("Login err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_DisabledUser(t *testing.T) {
	us, _, svc := testEnv(t)
	ctx := context.Background()

	user, _ := svc.Setup(ctx, "admin", "admin@example.com", "securepassword")
	user.Disabled = true
	_ = us.UpdateUser(ctx, user)

	_, err := svc.Login(ctx, "admin", "securepassword")
	if err != ErrUserDisabled {
		t.Errorf("Login err = %v, want ErrUserDisabled", err)
	}
}

func TestRefresh_Rotation(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	_, _ = svc.Setup(ctx, "admin", "admin@example.com", "securepassword")
	pair1, _ := svc.Login(ctx, "admin", "securepassword")

	// Refresh should return a new pair.
	pair2, err := svc.Refresh(ctx, pair1.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if pair2.RefreshToken == pair1.RefreshToken {
		t.Error("refresh should issue a new refresh token (rotation)")
	}

	// Old refresh token should be revoked.
	_, err = svc.Refresh(ctx, pair1.RefreshToken)
	if err != ErrInvalidToken {
		t.Errorf("reuse of old refresh token: err = %v, want ErrInvalidToken", err)
	}

	// New refresh token should still work.
	pair3, err := svc.Refresh(ctx, pair2.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh with new token: %v", err)
	}
	if pair3.AccessToken == "" {
		t.Error("expected non-empty access token from second refresh")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	_, err := svc.Refresh(ctx, "totally-fake-token")
	if err != ErrInvalidToken {
		t.Errorf("Refresh err = %v, want ErrInvalidToken", err)
	}
}

func TestLogout_RevokesToken(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	_, _ = svc.Setup(ctx, "admin", "admin@example.com", "securepassword")
	pair, _ := svc.Login(ctx, "admin", "securepassword")

	if err := svc.Logout(ctx, pair.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// Refresh with the revoked token should fail.
	_, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != ErrInvalidToken {
		t.Errorf("Refresh after logout: err = %v, want ErrInvalidToken", err)
	}
}

func TestLogout_Idempotent(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	// Logging out a non-existent token should not error.
	if err := svc.Logout(ctx, "nonexistent-token"); err != nil {
		t.Errorf("Logout of nonexistent token: %v", err)
	}
}

func TestUserCRUD(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	admin, _ := svc.Setup(ctx, "admin", "admin@example.com", "securepassword")

	// ListUsers
	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("ListUsers len = %d, want 1", len(users))
	}

	// GetUser
	got, err := svc.GetUser(ctx, admin.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Username != "admin" {
		t.Errorf("GetUser.Username = %q, want admin", got.Username)
	}

	// UpdateUser
	updated, err := svc.UpdateUser(ctx, admin.ID, "new@example.com", RoleViewer, false)
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if updated.Email != "new@example.com" {
		t.Errorf("UpdateUser.Email = %q, want new@example.com", updated.Email)
	}
	if updated.Role != RoleViewer {
		t.Errorf("UpdateUser.Role = %q, want viewer", updated.Role)
	}

	// DeleteUser
	if err := svc.DeleteUser(ctx, admin.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	// GetUser after delete
	_, err = svc.GetUser(ctx, admin.ID)
	if err != ErrUserNotFound {
		t.Errorf("GetUser after delete: err = %v, want ErrUserNotFound", err)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	_, _, svc := testEnv(t)
	ctx := context.Background()

	err := svc.DeleteUser(ctx, "nonexistent-id")
	if err != ErrUserNotFound {
		t.Errorf("DeleteUser err = %v, want ErrUserNotFound", err)
	}
}
