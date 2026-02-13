package dispatch

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
)

func testStore(t *testing.T) *DispatchStore {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "dispatch", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewDispatchStore(db.DB())
}

// -- Agent tests --

func TestDispatchStore_UpsertAgent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	agent := &Agent{
		ID:           "agent-001",
		Hostname:     "web-server-01",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		DeviceID:     "dev-001",
		Status:       "connected",
		LastCheckIn:  &now,
		EnrolledAt:   now,
		CertSerial:   "abc123",
		ConfigJSON:   "{}",
	}

	// Insert new agent.
	if err := s.UpsertAgent(ctx, agent); err != nil {
		t.Fatalf("UpsertAgent (insert): %v", err)
	}

	// Verify it was inserted.
	got, err := s.GetAgent(ctx, "agent-001")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got == nil {
		t.Fatal("GetAgent returned nil, want non-nil")
	}
	if got.Hostname != "web-server-01" {
		t.Errorf("Hostname = %q, want %q", got.Hostname, "web-server-01")
	}
	if got.Platform != "linux/amd64" {
		t.Errorf("Platform = %q, want %q", got.Platform, "linux/amd64")
	}
	if got.Status != "connected" {
		t.Errorf("Status = %q, want %q", got.Status, "connected")
	}

	// Update existing agent.
	agent.Hostname = "web-server-02"
	agent.Status = "disconnected"
	if err := s.UpsertAgent(ctx, agent); err != nil {
		t.Fatalf("UpsertAgent (update): %v", err)
	}

	got, err = s.GetAgent(ctx, "agent-001")
	if err != nil {
		t.Fatalf("GetAgent after update: %v", err)
	}
	if got.Hostname != "web-server-02" {
		t.Errorf("Hostname after update = %q, want %q", got.Hostname, "web-server-02")
	}
	if got.Status != "disconnected" {
		t.Errorf("Status after update = %q, want %q", got.Status, "disconnected")
	}
}

func TestDispatchStore_GetAgent(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		setup   func(t *testing.T, s *DispatchStore)
		wantNil bool
	}{
		{
			name:    "not found",
			id:      "nonexistent",
			wantNil: true,
		},
		{
			name: "found",
			id:   "agent-001",
			setup: func(t *testing.T, s *DispatchStore) {
				t.Helper()
				now := time.Now().UTC().Truncate(time.Second)
				if err := s.UpsertAgent(context.Background(), &Agent{
					ID:           "agent-001",
					Hostname:     "test-host",
					Platform:     "darwin/arm64",
					AgentVersion: "0.2.0",
					ProtoVersion: 1,
					Status:       "pending",
					EnrolledAt:   now,
					ConfigJSON:   "{}",
				}); err != nil {
					t.Fatalf("setup: %v", err)
				}
			},
			wantNil: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := testStore(t)
			if tc.setup != nil {
				tc.setup(t, s)
			}

			got, err := s.GetAgent(context.Background(), tc.id)
			if err != nil {
				t.Fatalf("GetAgent: %v", err)
			}
			if tc.wantNil && got != nil {
				t.Errorf("GetAgent = %+v, want nil", got)
			}
			if !tc.wantNil && got == nil {
				t.Error("GetAgent returned nil, want non-nil")
			}
		})
	}
}

func TestDispatchStore_ListAgents(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Empty list initially.
	agents, err := s.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents (empty): %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("expected 0 agents, got %d", len(agents))
	}

	// Insert two agents.
	now := time.Now().UTC().Truncate(time.Second)
	for _, id := range []string{"agent-001", "agent-002"} {
		if err := s.UpsertAgent(ctx, &Agent{
			ID:           id,
			Hostname:     id + "-host",
			Platform:     "linux/amd64",
			AgentVersion: "0.1.0",
			ProtoVersion: 1,
			Status:       "connected",
			EnrolledAt:   now,
			ConfigJSON:   "{}",
		}); err != nil {
			t.Fatalf("UpsertAgent %s: %v", id, err)
		}
	}

	agents, err = s.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
}

func TestDispatchStore_UpdateCheckIn(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	// Insert an agent first.
	if err := s.UpsertAgent(ctx, &Agent{
		ID:           "agent-001",
		Hostname:     "old-host",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		Status:       "pending",
		EnrolledAt:   now,
		ConfigJSON:   "{}",
	}); err != nil {
		t.Fatalf("UpsertAgent: %v", err)
	}

	// Update check-in.
	if err := s.UpdateCheckIn(ctx, "agent-001", "new-host", "linux/arm64", "0.2.0", 2); err != nil {
		t.Fatalf("UpdateCheckIn: %v", err)
	}

	got, err := s.GetAgent(ctx, "agent-001")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got.Hostname != "new-host" {
		t.Errorf("Hostname = %q, want %q", got.Hostname, "new-host")
	}
	if got.Platform != "linux/arm64" {
		t.Errorf("Platform = %q, want %q", got.Platform, "linux/arm64")
	}
	if got.AgentVersion != "0.2.0" {
		t.Errorf("AgentVersion = %q, want %q", got.AgentVersion, "0.2.0")
	}
	if got.ProtoVersion != 2 {
		t.Errorf("ProtoVersion = %d, want %d", got.ProtoVersion, 2)
	}
	if got.Status != "connected" {
		t.Errorf("Status = %q, want %q", got.Status, "connected")
	}
	if got.LastCheckIn == nil {
		t.Error("LastCheckIn is nil, want non-nil")
	}

	// Update check-in for nonexistent agent.
	err = s.UpdateCheckIn(ctx, "nonexistent", "h", "p", "v", 1)
	if err == nil {
		t.Error("UpdateCheckIn for nonexistent agent should return error")
	}
}

func TestDispatchStore_DeleteAgent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	// Insert an agent.
	if err := s.UpsertAgent(ctx, &Agent{
		ID:           "agent-001",
		Hostname:     "test-host",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		Status:       "connected",
		EnrolledAt:   now,
		ConfigJSON:   "{}",
	}); err != nil {
		t.Fatalf("UpsertAgent: %v", err)
	}

	// Delete the agent.
	if err := s.DeleteAgent(ctx, "agent-001"); err != nil {
		t.Fatalf("DeleteAgent: %v", err)
	}

	// Verify it was deleted.
	got, err := s.GetAgent(ctx, "agent-001")
	if err != nil {
		t.Fatalf("GetAgent after delete: %v", err)
	}
	if got != nil {
		t.Errorf("agent still exists after delete: %+v", got)
	}

	// Delete nonexistent agent.
	err = s.DeleteAgent(ctx, "nonexistent")
	if err == nil {
		t.Error("DeleteAgent for nonexistent agent should return error")
	}
}

// -- Enrollment token tests --

func TestDispatchStore_EnrollmentTokenLifecycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	expiresAt := now.Add(24 * time.Hour)

	// Create a token with max_uses=2.
	token := &EnrollmentToken{
		ID:          "tok-001",
		TokenHash:   "hash-abc123",
		Description: "Test token",
		CreatedAt:   now,
		ExpiresAt:   &expiresAt,
		MaxUses:     2,
	}
	if err := s.CreateEnrollmentToken(ctx, token); err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}

	// Validate the token -- should succeed.
	got, err := s.ValidateEnrollmentToken(ctx, "hash-abc123")
	if err != nil {
		t.Fatalf("ValidateEnrollmentToken: %v", err)
	}
	if got.ID != "tok-001" {
		t.Errorf("ID = %q, want %q", got.ID, "tok-001")
	}
	if got.Description != "Test token" {
		t.Errorf("Description = %q, want %q", got.Description, "Test token")
	}
	if got.MaxUses != 2 {
		t.Errorf("MaxUses = %d, want %d", got.MaxUses, 2)
	}
	if got.UseCount != 0 {
		t.Errorf("UseCount = %d, want %d", got.UseCount, 0)
	}

	// Consume the token (first use).
	if err := s.ConsumeEnrollmentToken(ctx, "hash-abc123", "agent-001"); err != nil {
		t.Fatalf("ConsumeEnrollmentToken (first): %v", err)
	}

	// Validate again -- should succeed (use_count=1, max_uses=2).
	got, err = s.ValidateEnrollmentToken(ctx, "hash-abc123")
	if err != nil {
		t.Fatalf("ValidateEnrollmentToken after first use: %v", err)
	}
	if got.UseCount != 1 {
		t.Errorf("UseCount = %d, want %d", got.UseCount, 1)
	}
	if got.AgentID != "agent-001" {
		t.Errorf("AgentID = %q, want %q", got.AgentID, "agent-001")
	}
	if got.UsedAt == nil {
		t.Error("UsedAt is nil after consumption, want non-nil")
	}

	// Consume the token (second use).
	if err := s.ConsumeEnrollmentToken(ctx, "hash-abc123", "agent-002"); err != nil {
		t.Fatalf("ConsumeEnrollmentToken (second): %v", err)
	}

	// Validate again -- should fail (exhausted).
	_, err = s.ValidateEnrollmentToken(ctx, "hash-abc123")
	if err == nil {
		t.Error("ValidateEnrollmentToken should fail after exhaustion")
	}
}

func TestDispatchStore_ValidateEnrollmentToken_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	_, err := s.ValidateEnrollmentToken(ctx, "nonexistent-hash")
	if err == nil {
		t.Error("ValidateEnrollmentToken should fail for nonexistent token")
	}
}

func TestDispatchStore_ValidateEnrollmentToken_Expired(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	expiredAt := now.Add(-1 * time.Hour) // already expired

	token := &EnrollmentToken{
		ID:          "tok-expired",
		TokenHash:   "hash-expired",
		Description: "Expired token",
		CreatedAt:   now.Add(-24 * time.Hour),
		ExpiresAt:   &expiredAt,
		MaxUses:     1,
	}
	if err := s.CreateEnrollmentToken(ctx, token); err != nil {
		t.Fatalf("CreateEnrollmentToken: %v", err)
	}

	_, err := s.ValidateEnrollmentToken(ctx, "hash-expired")
	if err == nil {
		t.Error("ValidateEnrollmentToken should fail for expired token")
	}
}

func TestDispatchStore_ConsumeEnrollmentToken_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	err := s.ConsumeEnrollmentToken(ctx, "nonexistent-hash", "agent-001")
	if err == nil {
		t.Error("ConsumeEnrollmentToken should fail for nonexistent token")
	}
}
