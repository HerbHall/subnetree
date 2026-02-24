package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

// UserStore provides persistence for user accounts.
type UserStore struct {
	db *sql.DB
}

// NewUserStore creates a UserStore and runs auth migrations.
func NewUserStore(ctx context.Context, store plugin.Store) (*UserStore, error) {
	if err := store.Migrate(ctx, "auth", migrations); err != nil {
		return nil, fmt.Errorf("auth migrations: %w", err)
	}
	return &UserStore{db: store.DB()}, nil
}

// CreateUser inserts a new user.
func (s *UserStore) CreateUser(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_users (id, username, email, password_hash, role, auth_provider, oidc_subject, created_at, disabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.Email, u.PasswordHash, string(u.Role),
		u.AuthProvider, u.OIDCSubject, u.CreatedAt, u.Disabled,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetUserByID returns a user by ID.
func (s *UserStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM auth_users WHERE id = ?`, id))
}

// GetUserByUsername returns a user by username.
func (s *UserStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM auth_users WHERE username = ?`, username))
}

// ListUsers returns all users.
func (s *UserStore) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+userColumns+` FROM auth_users ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := s.scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// UpdateUser updates a user's mutable fields.
func (s *UserStore) UpdateUser(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE auth_users SET email = ?, role = ?, disabled = ? WHERE id = ?`,
		u.Email, string(u.Role), u.Disabled, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// UpdateLastLogin sets the last_login timestamp.
func (s *UserStore) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_users SET last_login = ? WHERE id = ?`,
		time.Now().UTC(), userID,
	)
	return err
}

// DeleteUser removes a user by ID.
func (s *UserStore) DeleteUser(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM auth_users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CountUsers returns the total number of users.
func (s *UserStore) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM auth_users`).Scan(&count)
	return count, err
}

// SaveRefreshToken stores a hashed refresh token.
func (s *UserStore) SaveRefreshToken(ctx context.Context, id, userID, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		id, userID, tokenHash, expiresAt, time.Now().UTC(),
	)
	return err
}

// GetRefreshToken looks up a refresh token by its hash.
func (s *UserStore) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	var rt RefreshToken
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked
		FROM auth_refresh_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt, &rt.Revoked)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

// RevokeRefreshToken marks a refresh token as revoked.
func (s *UserStore) RevokeRefreshToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_refresh_tokens SET revoked = 1 WHERE id = ?`, id)
	return err
}

// RevokeUserRefreshTokens revokes all refresh tokens for a user.
func (s *UserStore) RevokeUserRefreshTokens(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_refresh_tokens SET revoked = 1 WHERE user_id = ?`, userID)
	return err
}

// CleanExpiredTokens removes expired refresh tokens.
func (s *UserStore) CleanExpiredTokens(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM auth_refresh_tokens WHERE expires_at < ? OR revoked = 1`,
		time.Now().UTC(),
	)
	return err
}

// RefreshToken represents a stored refresh token.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	Revoked   bool
}

// RecordFailedLogin increments the failed attempt counter and returns the new count.
func (s *UserStore) RecordFailedLogin(ctx context.Context, userID string) (attempts int, err error) {
	_, err = s.db.ExecContext(ctx,
		`UPDATE auth_users SET failed_login_attempts = failed_login_attempts + 1 WHERE id = ?`,
		userID)
	if err != nil {
		return 0, fmt.Errorf("record failed login: %w", err)
	}
	err = s.db.QueryRowContext(ctx,
		`SELECT failed_login_attempts FROM auth_users WHERE id = ?`, userID).Scan(&attempts)
	return attempts, err
}

// LockAccount sets the locked_until timestamp for a user.
func (s *UserStore) LockAccount(ctx context.Context, userID string, lockedUntil time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_users SET locked_until = ? WHERE id = ?`,
		lockedUntil, userID)
	return err
}

// ClearFailedLogins resets the failed attempt counter and unlocks the account.
func (s *UserStore) ClearFailedLogins(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_users SET failed_login_attempts = 0, locked_until = NULL WHERE id = ?`,
		userID)
	return err
}

// userColumns is the shared SELECT column list for user queries.
const userColumns = `id, username, email, password_hash, role, auth_provider, oidc_subject,
	created_at, last_login, disabled, failed_login_attempts, locked_until, totp_enabled, totp_verified`

func (s *UserStore) scanUser(row *sql.Row) (*User, error) {
	var u User
	var role string
	var lastLogin sql.NullTime
	var lockedUntil sql.NullTime
	var passwordHash sql.NullString
	var oidcSubject sql.NullString

	err := row.Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &role,
		&u.AuthProvider, &oidcSubject, &u.CreatedAt, &lastLogin, &u.Disabled,
		&u.FailedLoginAttempts, &lockedUntil, &u.TOTPEnabled, &u.TOTPVerified)
	if err != nil {
		return nil, err
	}
	u.Role = Role(role)
	if lastLogin.Valid {
		u.LastLogin = lastLogin.Time
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	if oidcSubject.Valid {
		u.OIDCSubject = oidcSubject.String
	}
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	return &u, nil
}

func (s *UserStore) scanUserRow(rows *sql.Rows) (*User, error) {
	var u User
	var role string
	var lastLogin sql.NullTime
	var lockedUntil sql.NullTime
	var passwordHash sql.NullString
	var oidcSubject sql.NullString

	err := rows.Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &role,
		&u.AuthProvider, &oidcSubject, &u.CreatedAt, &lastLogin, &u.Disabled,
		&u.FailedLoginAttempts, &lockedUntil, &u.TOTPEnabled, &u.TOTPVerified)
	if err != nil {
		return nil, err
	}
	u.Role = Role(role)
	if lastLogin.Valid {
		u.LastLogin = lastLogin.Time
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	if oidcSubject.Valid {
		u.OIDCSubject = oidcSubject.String
	}
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	return &u, nil
}

// migrations for the auth module.
var migrations = []plugin.Migration{
	{
		Version:     1,
		Description: "create auth_users table",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE auth_users (
					id            TEXT PRIMARY KEY,
					username      TEXT NOT NULL UNIQUE,
					email         TEXT NOT NULL UNIQUE,
					password_hash TEXT,
					role          TEXT NOT NULL DEFAULT 'viewer',
					auth_provider TEXT NOT NULL DEFAULT 'local',
					oidc_subject  TEXT,
					created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					last_login    DATETIME,
					disabled      INTEGER NOT NULL DEFAULT 0
				)`)
			return err
		},
	},
	{
		Version:     2,
		Description: "create auth_refresh_tokens table",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				CREATE TABLE auth_refresh_tokens (
					id         TEXT PRIMARY KEY,
					user_id    TEXT NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
					token_hash TEXT NOT NULL UNIQUE,
					expires_at DATETIME NOT NULL,
					created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
					revoked    INTEGER NOT NULL DEFAULT 0
				)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_refresh_tokens_user ON auth_refresh_tokens(user_id)`)
			return err
		},
	},
	{
		Version:     3,
		Description: "add account lockout columns",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`ALTER TABLE auth_users ADD COLUMN failed_login_attempts INTEGER NOT NULL DEFAULT 0`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`ALTER TABLE auth_users ADD COLUMN locked_until DATETIME`)
			return err
		},
	},
	{
		Version:     4,
		Description: "add MFA columns and tables",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`ALTER TABLE auth_users ADD COLUMN totp_secret TEXT`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`ALTER TABLE auth_users ADD COLUMN totp_enabled INTEGER NOT NULL DEFAULT 0`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`ALTER TABLE auth_users ADD COLUMN totp_verified INTEGER NOT NULL DEFAULT 0`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE TABLE auth_recovery_codes (
				id         TEXT PRIMARY KEY,
				user_id    TEXT NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
				code_hash  TEXT NOT NULL,
				used       INTEGER NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_recovery_codes_user ON auth_recovery_codes(user_id)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE TABLE auth_mfa_tokens (
				token_hash TEXT PRIMARY KEY,
				user_id    TEXT NOT NULL,
				expires_at DATETIME NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`)
			return err
		},
	},
}

// GetTOTPSecret returns the encrypted TOTP secret for a user.
func (s *UserStore) GetTOTPSecret(ctx context.Context, userID string) (string, error) {
	var secret sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT totp_secret FROM auth_users WHERE id = ?`, userID).Scan(&secret)
	if err != nil {
		return "", fmt.Errorf("get TOTP secret: %w", err)
	}
	if !secret.Valid {
		return "", nil
	}
	return secret.String, nil
}

// SetTOTPSecret stores an encrypted TOTP secret for a user.
func (s *UserStore) SetTOTPSecret(ctx context.Context, userID, encryptedSecret string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_users SET totp_secret = ? WHERE id = ?`,
		encryptedSecret, userID)
	if err != nil {
		return fmt.Errorf("set TOTP secret: %w", err)
	}
	return nil
}

// EnableTOTP marks a user's TOTP as enabled and verified.
func (s *UserStore) EnableTOTP(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_users SET totp_enabled = 1, totp_verified = 1 WHERE id = ?`,
		userID)
	if err != nil {
		return fmt.Errorf("enable TOTP: %w", err)
	}
	return nil
}

// DisableTOTP clears TOTP for a user: disables, unverifies, and removes the secret.
func (s *UserStore) DisableTOTP(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_users SET totp_enabled = 0, totp_verified = 0, totp_secret = NULL WHERE id = ?`,
		userID)
	if err != nil {
		return fmt.Errorf("disable TOTP: %w", err)
	}
	// Also delete recovery codes.
	_, err = s.db.ExecContext(ctx, `DELETE FROM auth_recovery_codes WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete recovery codes: %w", err)
	}
	return nil
}

// SaveRecoveryCodes stores hashed recovery codes for a user, replacing any existing ones.
func (s *UserStore) SaveRecoveryCodes(ctx context.Context, userID string, codeHashes []string) error {
	// Delete existing codes first.
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_recovery_codes WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("clear old recovery codes: %w", err)
	}

	for _, hash := range codeHashes {
		id := uuid.New().String()
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO auth_recovery_codes (id, user_id, code_hash) VALUES (?, ?, ?)`,
			id, userID, hash)
		if err != nil {
			return fmt.Errorf("save recovery code: %w", err)
		}
	}
	return nil
}

// ValidateRecoveryCode checks if a hashed recovery code exists and is unused for the user.
func (s *UserStore) ValidateRecoveryCode(ctx context.Context, userID, codeHash string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM auth_recovery_codes WHERE user_id = ? AND code_hash = ? AND used = 0`,
		userID, codeHash).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("validate recovery code: %w", err)
	}
	return count > 0, nil
}

// MarkRecoveryCodeUsed marks a recovery code as used.
func (s *UserStore) MarkRecoveryCodeUsed(ctx context.Context, codeHash string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_recovery_codes SET used = 1 WHERE code_hash = ?`, codeHash)
	if err != nil {
		return fmt.Errorf("mark recovery code used: %w", err)
	}
	return nil
}

// SaveMFAToken stores a hashed MFA token with its expiration.
func (s *UserStore) SaveMFAToken(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO auth_mfa_tokens (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		tokenHash, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("save MFA token: %w", err)
	}
	return nil
}

// GetMFAToken looks up an MFA token by hash and returns the user ID if valid and not expired.
func (s *UserStore) GetMFAToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at FROM auth_mfa_tokens WHERE token_hash = ?`,
		tokenHash).Scan(&userID, &expiresAt)
	if err != nil {
		return "", fmt.Errorf("get MFA token: %w", err)
	}
	if expiresAt.Before(time.Now()) {
		return "", fmt.Errorf("MFA token expired")
	}
	return userID, nil
}

// RevokeMFAToken deletes an MFA token.
func (s *UserStore) RevokeMFAToken(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM auth_mfa_tokens WHERE token_hash = ?`, tokenHash)
	return err
}
