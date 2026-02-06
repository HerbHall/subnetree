package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service errors.
var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserDisabled       = errors.New("user account is disabled")
	ErrAccountLocked      = errors.New("account is locked due to too many failed login attempts")
	ErrUserExists         = errors.New("username or email already exists")
	ErrSetupComplete      = errors.New("setup already completed")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUserNotFound       = errors.New("user not found")
)

const (
	// DefaultMaxFailedAttempts is the number of failed logins before lockout.
	DefaultMaxFailedAttempts = 5
	// DefaultLockoutDuration is how long an account stays locked.
	DefaultLockoutDuration = 15 * time.Minute
)

// TokenPair contains an access token and refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	RefreshToken string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJl..."`
	ExpiresIn    int    `json:"expires_in" example:"900"` // Access token TTL in seconds
}

// Service provides authentication business logic.
type Service struct {
	store  *UserStore
	tokens *TokenService
	logger *zap.Logger
}

// NewService creates an auth Service.
func NewService(store *UserStore, tokens *TokenService, logger *zap.Logger) *Service {
	return &Service{
		store:  store,
		tokens: tokens,
		logger: logger,
	}
}

// Tokens returns the token service for middleware use.
func (s *Service) Tokens() *TokenService {
	return s.tokens
}

// Login authenticates a user and returns a token pair.
func (s *Service) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	user, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("lookup user: %w", err)
	}

	if user.Disabled {
		return nil, ErrUserDisabled
	}

	// Check if account is locked.
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		s.logger.Warn("login attempt on locked account",
			zap.String("username", username),
			zap.Time("locked_until", *user.LockedUntil),
		)
		return nil, ErrAccountLocked
	}

	if !CheckPassword(user.PasswordHash, password) {
		s.handleFailedLogin(ctx, user)
		return nil, ErrInvalidCredentials
	}

	// Successful login -- clear any failed attempts.
	if user.FailedLoginAttempts > 0 {
		_ = s.store.ClearFailedLogins(ctx, user.ID)
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	_ = s.store.UpdateLastLogin(ctx, user.ID)
	s.logger.Info("user logged in", zap.String("username", username), zap.String("user_id", user.ID))
	return pair, nil
}

func (s *Service) handleFailedLogin(ctx context.Context, user *User) {
	attempts, err := s.store.RecordFailedLogin(ctx, user.ID)
	if err != nil {
		s.logger.Error("failed to record failed login", zap.Error(err))
		return
	}

	if attempts >= DefaultMaxFailedAttempts {
		lockedUntil := time.Now().Add(DefaultLockoutDuration)
		if err := s.store.LockAccount(ctx, user.ID, lockedUntil); err != nil {
			s.logger.Error("failed to lock account", zap.Error(err))
			return
		}
		s.logger.Warn("account locked due to failed login attempts",
			zap.String("username", user.Username),
			zap.String("user_id", user.ID),
			zap.Int("attempts", attempts),
			zap.Time("locked_until", lockedUntil),
		)
	}
}

// Setup creates the initial admin account. Only works when no users exist.
func (s *Service) Setup(ctx context.Context, username, email, password string) (*User, error) {
	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil, ErrSetupComplete
	}

	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	hash, err := HashPassword(password, 0)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           uuid.New().String(),
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         RoleAdmin,
		AuthProvider: "local",
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.store.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create admin: %w", err)
	}

	s.logger.Info("initial admin account created", zap.String("username", username))
	return user, nil
}

// Refresh validates a refresh token and returns a new token pair (rotation).
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tokenHash := HashToken(refreshToken)
	rt, err := s.store.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("lookup refresh token: %w", err)
	}

	if rt.Revoked || rt.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidToken
	}

	// Revoke the old token (rotation).
	_ = s.store.RevokeRefreshToken(ctx, rt.ID)

	user, err := s.store.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, fmt.Errorf("lookup user for refresh: %w", err)
	}
	if user.Disabled {
		return nil, ErrUserDisabled
	}

	return s.issueTokenPair(ctx, user)
}

// Logout revokes a refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := HashToken(refreshToken)
	rt, err := s.store.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // Already revoked or doesn't exist -- idempotent.
		}
		return fmt.Errorf("lookup refresh token: %w", err)
	}
	return s.store.RevokeRefreshToken(ctx, rt.ID)
}

// NeedsSetup returns true if no users exist (first-run state).
func (s *Service) NeedsSetup(ctx context.Context) (bool, error) {
	count, err := s.store.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// ListUsers returns all users (for admin endpoints).
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.store.ListUsers(ctx)
}

// GetUser returns a user by ID.
func (s *Service) GetUser(ctx context.Context, id string) (*User, error) {
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// UpdateUser updates a user's email, role, and disabled status.
func (s *Service) UpdateUser(ctx context.Context, id, email string, role Role, disabled bool) (*User, error) {
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	user.Email = email
	user.Role = role
	user.Disabled = disabled

	if err := s.store.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	// If the user was disabled, revoke all their refresh tokens.
	if disabled {
		_ = s.store.RevokeUserRefreshTokens(ctx, id)
	}

	return user, nil
}

// DeleteUser removes a user by ID.
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	if err := s.store.DeleteUser(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *Service) issueTokenPair(ctx context.Context, user *User) (*TokenPair, error) {
	accessToken, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return nil, err
	}

	rawRefresh, hashRefresh, expiresAt, err := s.tokens.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	tokenID := uuid.New().String()
	if err := s.store.SaveRefreshToken(ctx, tokenID, user.ID, hashRefresh, expiresAt); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(s.tokens.AccessTokenTTL().Seconds()),
	}, nil
}
