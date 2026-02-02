package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/HerbHall/netvantage/pkg/plugin"
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
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, role, auth_provider, oidc_subject, created_at, last_login, disabled
		FROM auth_users WHERE id = ?`, id))
}

// GetUserByUsername returns a user by username.
func (s *UserStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, role, auth_provider, oidc_subject, created_at, last_login, disabled
		FROM auth_users WHERE username = ?`, username))
}

// ListUsers returns all users.
func (s *UserStore) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, email, password_hash, role, auth_provider, oidc_subject, created_at, last_login, disabled
		FROM auth_users ORDER BY created_at`)
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

func (s *UserStore) scanUser(row *sql.Row) (*User, error) {
	var u User
	var role string
	var lastLogin sql.NullTime
	var passwordHash sql.NullString
	var oidcSubject sql.NullString

	err := row.Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &role,
		&u.AuthProvider, &oidcSubject, &u.CreatedAt, &lastLogin, &u.Disabled)
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
	return &u, nil
}

func (s *UserStore) scanUserRow(rows *sql.Rows) (*User, error) {
	var u User
	var role string
	var lastLogin sql.NullTime
	var passwordHash sql.NullString
	var oidcSubject sql.NullString

	err := rows.Scan(&u.ID, &u.Username, &u.Email, &passwordHash, &role,
		&u.AuthProvider, &oidcSubject, &u.CreatedAt, &lastLogin, &u.Disabled)
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
}
