package dispatch

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Agent represents a registered Scout agent.
type Agent struct {
	ID           string     `json:"id"`
	Hostname     string     `json:"hostname"`
	Platform     string     `json:"platform"`
	AgentVersion string     `json:"agent_version"`
	ProtoVersion int        `json:"proto_version"`
	DeviceID     string     `json:"device_id"`
	Status       string     `json:"status"` // pending, connected, disconnected
	LastCheckIn  *time.Time `json:"last_check_in,omitempty"`
	EnrolledAt   time.Time  `json:"enrolled_at"`
	CertSerial   string     `json:"cert_serial"`
	CertExpires  *time.Time `json:"cert_expires_at,omitempty"`
	ConfigJSON   string     `json:"config_json"`
}

// EnrollmentToken represents a one-time or multi-use enrollment token.
type EnrollmentToken struct {
	ID          string     `json:"id"`
	TokenHash   string     `json:"token_hash"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	UsedAt      *time.Time `json:"used_at,omitempty"`
	AgentID     string     `json:"agent_id,omitempty"`
	MaxUses     int        `json:"max_uses"`
	UseCount    int        `json:"use_count"`
}

// DispatchStore provides database operations for the Dispatch module.
type DispatchStore struct {
	db *sql.DB
}

// NewDispatchStore creates a new DispatchStore backed by the given database.
func NewDispatchStore(db *sql.DB) *DispatchStore {
	return &DispatchStore{db: db}
}

// -- Agent methods --

// UpsertAgent inserts a new agent or updates an existing one.
func (s *DispatchStore) UpsertAgent(ctx context.Context, agent *Agent) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dispatch_agents (
			id, hostname, platform, agent_version, proto_version,
			device_id, status, last_check_in, enrolled_at,
			cert_serial, cert_expires_at, config_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			hostname = excluded.hostname,
			platform = excluded.platform,
			agent_version = excluded.agent_version,
			proto_version = excluded.proto_version,
			device_id = excluded.device_id,
			status = excluded.status,
			last_check_in = excluded.last_check_in,
			cert_serial = excluded.cert_serial,
			cert_expires_at = excluded.cert_expires_at,
			config_json = excluded.config_json`,
		agent.ID, agent.Hostname, agent.Platform, agent.AgentVersion, agent.ProtoVersion,
		agent.DeviceID, agent.Status, nullTime(agent.LastCheckIn), agent.EnrolledAt,
		agent.CertSerial, nullTime(agent.CertExpires), agent.ConfigJSON,
	)
	if err != nil {
		return fmt.Errorf("upsert agent: %w", err)
	}
	return nil
}

// GetAgent returns an agent by ID. Returns nil, nil if not found.
func (s *DispatchStore) GetAgent(ctx context.Context, id string) (*Agent, error) {
	var a Agent
	var lastCheckIn, certExpires sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, hostname, platform, agent_version, proto_version,
			device_id, status, last_check_in, enrolled_at,
			cert_serial, cert_expires_at, config_json
		FROM dispatch_agents WHERE id = ?`, id,
	).Scan(
		&a.ID, &a.Hostname, &a.Platform, &a.AgentVersion, &a.ProtoVersion,
		&a.DeviceID, &a.Status, &lastCheckIn, &a.EnrolledAt,
		&a.CertSerial, &certExpires, &a.ConfigJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent: %w", err)
	}
	if lastCheckIn.Valid {
		a.LastCheckIn = &lastCheckIn.Time
	}
	if certExpires.Valid {
		a.CertExpires = &certExpires.Time
	}
	return &a, nil
}

// ListAgents returns all registered agents.
func (s *DispatchStore) ListAgents(ctx context.Context) ([]Agent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, hostname, platform, agent_version, proto_version,
			device_id, status, last_check_in, enrolled_at,
			cert_serial, cert_expires_at, config_json
		FROM dispatch_agents ORDER BY enrolled_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var a Agent
		var lastCheckIn, certExpires sql.NullTime
		if err := rows.Scan(
			&a.ID, &a.Hostname, &a.Platform, &a.AgentVersion, &a.ProtoVersion,
			&a.DeviceID, &a.Status, &lastCheckIn, &a.EnrolledAt,
			&a.CertSerial, &certExpires, &a.ConfigJSON,
		); err != nil {
			return nil, fmt.Errorf("scan agent row: %w", err)
		}
		if lastCheckIn.Valid {
			a.LastCheckIn = &lastCheckIn.Time
		}
		if certExpires.Valid {
			a.CertExpires = &certExpires.Time
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// UpdateCheckIn updates an agent's check-in timestamp and metadata.
func (s *DispatchStore) UpdateCheckIn(ctx context.Context, agentID, hostname, platform, version string, protoVersion int) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE dispatch_agents SET
			hostname = ?, platform = ?, agent_version = ?,
			proto_version = ?, status = 'connected', last_check_in = ?
		WHERE id = ?`,
		hostname, platform, version, protoVersion, now, agentID,
	)
	if err != nil {
		return fmt.Errorf("update check-in: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("agent %q not found", agentID)
	}
	return nil
}

// DeleteAgent removes an agent by ID. Returns an error if the agent does not exist.
func (s *DispatchStore) DeleteAgent(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM dispatch_agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// -- Enrollment token methods --

// CreateEnrollmentToken inserts a new enrollment token.
func (s *DispatchStore) CreateEnrollmentToken(ctx context.Context, token *EnrollmentToken) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dispatch_enrollment_tokens (
			id, token_hash, description, created_at, expires_at,
			used_at, agent_id, max_uses, use_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		token.ID, token.TokenHash, token.Description, token.CreatedAt,
		nullTime(token.ExpiresAt), nullTime(token.UsedAt), token.AgentID,
		token.MaxUses, token.UseCount,
	)
	if err != nil {
		return fmt.Errorf("create enrollment token: %w", err)
	}
	return nil
}

// ValidateEnrollmentToken looks up a token by hash and checks validity.
// Returns the token if found and valid, or an error if expired/exhausted/not found.
func (s *DispatchStore) ValidateEnrollmentToken(ctx context.Context, tokenHash string) (*EnrollmentToken, error) {
	var t EnrollmentToken
	var expiresAt, usedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, token_hash, description, created_at, expires_at,
			used_at, agent_id, max_uses, use_count
		FROM dispatch_enrollment_tokens WHERE token_hash = ?`,
		tokenHash,
	).Scan(
		&t.ID, &t.TokenHash, &t.Description, &t.CreatedAt, &expiresAt,
		&usedAt, &t.AgentID, &t.MaxUses, &t.UseCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("enrollment token not found")
		}
		return nil, fmt.Errorf("validate enrollment token: %w", err)
	}
	if expiresAt.Valid {
		t.ExpiresAt = &expiresAt.Time
	}
	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}

	// Check if token is expired.
	if t.ExpiresAt != nil && time.Now().UTC().After(*t.ExpiresAt) {
		return nil, fmt.Errorf("enrollment token expired")
	}

	// Check if token has reached max uses.
	if t.UseCount >= t.MaxUses {
		return nil, fmt.Errorf("enrollment token exhausted")
	}

	return &t, nil
}

// ConsumeEnrollmentToken increments the use count and records the agent ID.
func (s *DispatchStore) ConsumeEnrollmentToken(ctx context.Context, tokenHash, agentID string) error {
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		UPDATE dispatch_enrollment_tokens SET
			use_count = use_count + 1,
			used_at = ?,
			agent_id = ?
		WHERE token_hash = ?`,
		now, agentID, tokenHash,
	)
	if err != nil {
		return fmt.Errorf("consume enrollment token: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("enrollment token not found")
	}
	return nil
}

// -- helpers --

// nullTime converts a *time.Time to sql.NullTime for database operations.
func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
