package vault

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// VaultStore provides database access for the Vault module.
type VaultStore struct {
	db *sql.DB
}

// NewVaultStore creates a new VaultStore wrapping the given database connection.
func NewVaultStore(db *sql.DB) *VaultStore {
	return &VaultStore{db: db}
}

// --- Master Key ---

// GetMasterKeyRecord returns the singleton master key record, or nil if
// no master key has been configured yet.
func (s *VaultStore) GetMasterKeyRecord(ctx context.Context) (*MasterKeyRecord, error) {
	var rec MasterKeyRecord
	err := s.db.QueryRowContext(ctx, `
		SELECT id, salt, verification_blob, created_at, updated_at
		FROM vault_master WHERE id = 1`,
	).Scan(&rec.ID, &rec.Salt, &rec.VerificationBlob, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get master key record: %w", err)
	}
	return &rec, nil
}

// UpsertMasterKeyRecord inserts or updates the singleton master key record.
func (s *VaultStore) UpsertMasterKeyRecord(ctx context.Context, salt, verification []byte) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vault_master (id, salt, verification_blob, created_at, updated_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			salt = excluded.salt,
			verification_blob = excluded.verification_blob,
			updated_at = excluded.updated_at`,
		salt, verification, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert master key record: %w", err)
	}
	return nil
}

// --- Credentials ---

// InsertCredential inserts a new credential record.
func (s *VaultStore) InsertCredential(ctx context.Context, cred *CredentialRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vault_credentials (id, name, type, device_id, description, encrypted_data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		cred.ID, cred.Name, cred.Type, cred.DeviceID, cred.Description,
		cred.EncryptedData, cred.CreatedAt, cred.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert credential: %w", err)
	}
	return nil
}

// GetCredential returns a credential by ID, including encrypted data.
// Returns nil, nil if not found.
func (s *VaultStore) GetCredential(ctx context.Context, id string) (*CredentialRecord, error) {
	var c CredentialRecord
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, type, device_id, description, encrypted_data, created_at, updated_at
		FROM vault_credentials WHERE id = ?`,
		id,
	).Scan(&c.ID, &c.Name, &c.Type, &c.DeviceID, &c.Description,
		&c.EncryptedData, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get credential: %w", err)
	}
	return &c, nil
}

// ListCredentials returns metadata for all credentials (no encrypted data).
func (s *VaultStore) ListCredentials(ctx context.Context) ([]CredentialMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, type, device_id, description, created_at, updated_at
		FROM vault_credentials ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}
	defer rows.Close()

	var metas []CredentialMeta
	for rows.Next() {
		var m CredentialMeta
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.DeviceID, &m.Description,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan credential row: %w", err)
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

// ListCredentialsByDevice returns metadata for credentials associated with a device.
func (s *VaultStore) ListCredentialsByDevice(ctx context.Context, deviceID string) ([]CredentialMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, type, device_id, description, created_at, updated_at
		FROM vault_credentials WHERE device_id = ? ORDER BY created_at`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list credentials by device: %w", err)
	}
	defer rows.Close()

	var metas []CredentialMeta
	for rows.Next() {
		var m CredentialMeta
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.DeviceID, &m.Description,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan credential row: %w", err)
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

// ListCredentialsByType returns metadata for credentials of a given type.
func (s *VaultStore) ListCredentialsByType(ctx context.Context, credType string) ([]CredentialMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, type, device_id, description, created_at, updated_at
		FROM vault_credentials WHERE type = ? ORDER BY created_at`,
		credType,
	)
	if err != nil {
		return nil, fmt.Errorf("list credentials by type: %w", err)
	}
	defer rows.Close()

	var metas []CredentialMeta
	for rows.Next() {
		var m CredentialMeta
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.DeviceID, &m.Description,
			&m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan credential row: %w", err)
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

// UpdateCredential updates an existing credential's metadata and encrypted data.
func (s *VaultStore) UpdateCredential(ctx context.Context, cred *CredentialRecord) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE vault_credentials SET
			name = ?, type = ?, device_id = ?, description = ?,
			encrypted_data = ?, updated_at = ?
		WHERE id = ?`,
		cred.Name, cred.Type, cred.DeviceID, cred.Description,
		cred.EncryptedData, cred.UpdatedAt, cred.ID,
	)
	if err != nil {
		return fmt.Errorf("update credential: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("credential %q not found", cred.ID)
	}
	return nil
}

// DeleteCredential deletes a credential by ID.
func (s *VaultStore) DeleteCredential(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM vault_credentials WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete credential: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("credential %q not found", id)
	}
	return nil
}

// CredentialCount returns the total number of stored credentials.
func (s *VaultStore) CredentialCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vault_credentials`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count credentials: %w", err)
	}
	return count, nil
}

// --- Keys ---

// InsertKey stores a wrapped DEK for a credential.
func (s *VaultStore) InsertKey(ctx context.Context, credentialID string, wrappedKey []byte) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vault_keys (credential_id, wrapped_key, created_at, updated_at)
		VALUES (?, ?, ?, ?)`,
		credentialID, wrappedKey, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert key: %w", err)
	}
	return nil
}

// GetKey returns the wrapped DEK for a credential. Returns nil, nil if not found.
func (s *VaultStore) GetKey(ctx context.Context, credentialID string) (*CredentialKey, error) {
	var k CredentialKey
	err := s.db.QueryRowContext(ctx, `
		SELECT credential_id, wrapped_key FROM vault_keys WHERE credential_id = ?`,
		credentialID,
	).Scan(&k.CredentialID, &k.WrappedKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get key: %w", err)
	}
	return &k, nil
}

// UpdateKey updates the wrapped DEK for a credential.
func (s *VaultStore) UpdateKey(ctx context.Context, credentialID string, wrappedKey []byte) error {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE vault_keys SET wrapped_key = ?, updated_at = ?
		WHERE credential_id = ?`,
		wrappedKey, now, credentialID,
	)
	if err != nil {
		return fmt.Errorf("update key: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("key for credential %q not found", credentialID)
	}
	return nil
}

// ListAllKeys returns all wrapped DEKs (for key rotation).
func (s *VaultStore) ListAllKeys(ctx context.Context) ([]CredentialKey, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT credential_id, wrapped_key FROM vault_keys`)
	if err != nil {
		return nil, fmt.Errorf("list all keys: %w", err)
	}
	defer rows.Close()

	var keys []CredentialKey
	for rows.Next() {
		var k CredentialKey
		if err := rows.Scan(&k.CredentialID, &k.WrappedKey); err != nil {
			return nil, fmt.Errorf("scan key row: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// DeleteKey deletes the wrapped DEK for a credential.
func (s *VaultStore) DeleteKey(ctx context.Context, credentialID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM vault_keys WHERE credential_id = ?`, credentialID)
	if err != nil {
		return fmt.Errorf("delete key: %w", err)
	}
	return nil
}

// --- Audit ---

// InsertAuditEntry records a credential access event.
func (s *VaultStore) InsertAuditEntry(ctx context.Context, entry *AuditEntry) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vault_audit_log (credential_id, user_id, action, purpose, source_ip, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)`,
		entry.CredentialID, entry.UserID, entry.Action,
		entry.Purpose, entry.SourceIP, entry.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert audit entry: %w", err)
	}
	return nil
}

// ListAuditEntries returns audit entries, optionally filtered by credential ID.
// Pass empty credentialID to list all entries.
func (s *VaultStore) ListAuditEntries(ctx context.Context, credentialID string, limit int) ([]AuditEntry, error) {
	var query string
	var args []any

	if credentialID != "" {
		query = `SELECT id, credential_id, user_id, action, purpose, source_ip, timestamp
			FROM vault_audit_log WHERE credential_id = ? ORDER BY timestamp DESC LIMIT ?`
		args = []any{credentialID, limit}
	} else {
		query = `SELECT id, credential_id, user_id, action, purpose, source_ip, timestamp
			FROM vault_audit_log ORDER BY timestamp DESC LIMIT ?`
		args = []any{limit}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit entries: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.CredentialID, &e.UserID, &e.Action,
			&e.Purpose, &e.SourceIP, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("scan audit row: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// DeleteOldAuditEntries deletes audit entries older than the given time.
func (s *VaultStore) DeleteOldAuditEntries(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM vault_audit_log WHERE timestamp < ?`, before)
	if err != nil {
		return 0, fmt.Errorf("delete old audit entries: %w", err)
	}
	return result.RowsAffected()
}
