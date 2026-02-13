package dispatch

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"testing"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func testGRPCServer(t *testing.T) (scoutpb.ScoutServiceClient, *DispatchStore) {
	t.Helper()

	store := testStore(t)

	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { lis.Close() })

	srv := grpc.NewServer()
	scoutpb.RegisterScoutServiceServer(srv, &scoutServer{
		store:  store,
		logger: zap.NewNop(),
		cfg:    DefaultConfig(),
	})
	t.Cleanup(func() { srv.Stop() })

	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return scoutpb.NewScoutServiceClient(conn), store
}

func TestGRPC_CheckIn_WithAgentID(t *testing.T) {
	client, store := testGRPCServer(t)
	ctx := context.Background()

	// Pre-create an agent in the store.
	now := time.Now().UTC()
	if err := store.UpsertAgent(ctx, &Agent{
		ID:           "agent-001",
		Hostname:     "test-host",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		Status:       "connected",
		EnrolledAt:   now,
		ConfigJSON:   "{}",
	}); err != nil {
		t.Fatalf("setup agent: %v", err)
	}

	resp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		AgentId:      "agent-001",
		Hostname:     "test-host",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
	})
	if err != nil {
		t.Fatalf("CheckIn: %v", err)
	}

	if !resp.Acknowledged {
		t.Error("expected acknowledged=true")
	}
	if resp.VersionStatus != scoutpb.VersionStatus_VERSION_OK {
		t.Errorf("version_status = %v, want VERSION_OK", resp.VersionStatus)
	}
	if resp.AssignedAgentId != "" {
		t.Errorf("assigned_agent_id = %q, want empty (not enrolling)", resp.AssignedAgentId)
	}
}

func TestGRPC_CheckIn_Enrollment(t *testing.T) {
	client, store := testGRPCServer(t)
	ctx := context.Background()

	// Create enrollment token.
	rawToken := "test-enrollment-token-12345"
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	if err := store.CreateEnrollmentToken(ctx, &EnrollmentToken{
		ID:          "tok-001",
		TokenHash:   tokenHash,
		Description: "test token",
		CreatedAt:   now,
		ExpiresAt:   &expires,
		MaxUses:     1,
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Enroll with the token.
	resp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		Hostname:     "new-agent-host",
		Platform:     "windows/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		EnrollToken:  rawToken,
	})
	if err != nil {
		t.Fatalf("CheckIn (enroll): %v", err)
	}

	if !resp.Acknowledged {
		t.Error("expected acknowledged=true")
	}
	if resp.AssignedAgentId == "" {
		t.Fatal("expected non-empty assigned_agent_id")
	}

	// Verify agent was created in store.
	agent, err := store.GetAgent(ctx, resp.AssignedAgentId)
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent in store")
	}
	if agent.Hostname != "new-agent-host" {
		t.Errorf("hostname = %q, want %q", agent.Hostname, "new-agent-host")
	}
	if agent.Platform != "windows/amd64" {
		t.Errorf("platform = %q, want %q", agent.Platform, "windows/amd64")
	}
}

func TestGRPC_CheckIn_InvalidToken(t *testing.T) {
	client, _ := testGRPCServer(t)
	ctx := context.Background()

	_, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		Hostname:     "rogue-agent",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		EnrollToken:  "invalid-token",
	})
	if err == nil {
		t.Fatal("expected error for invalid enrollment token")
	}
}

func TestGRPC_CheckIn_NoIDNoToken(t *testing.T) {
	client, _ := testGRPCServer(t)
	ctx := context.Background()

	_, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		Hostname:     "no-id-host",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
	})
	if err == nil {
		t.Fatal("expected error when no agent_id and no enroll_token")
	}
}

func TestGRPC_CheckIn_VersionNegotiation(t *testing.T) {
	client, store := testGRPCServer(t)
	ctx := context.Background()

	// Pre-create agent.
	now := time.Now().UTC()
	_ = store.UpsertAgent(ctx, &Agent{
		ID: "agent-ver", Hostname: "h", Platform: "p",
		AgentVersion: "0.1.0", ProtoVersion: 1,
		Status: "connected", EnrolledAt: now, ConfigJSON: "{}",
	})

	tests := []struct {
		name         string
		protoVersion uint32
		wantStatus   scoutpb.VersionStatus
		wantAcked    bool
	}{
		{"current version", 1, scoutpb.VersionStatus_VERSION_OK, true},
		{"deprecated (current-1)", 0, scoutpb.VersionStatus_VERSION_DEPRECATED, true},
		{"rejected (too old)", 99, scoutpb.VersionStatus_VERSION_REJECTED, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
				AgentId:      "agent-ver",
				Hostname:     "h",
				Platform:     "p",
				AgentVersion: "0.1.0",
				ProtoVersion: tc.protoVersion,
			})
			if err != nil {
				t.Fatalf("CheckIn: %v", err)
			}
			if resp.VersionStatus != tc.wantStatus {
				t.Errorf("version_status = %v, want %v", resp.VersionStatus, tc.wantStatus)
			}
			if resp.Acknowledged != tc.wantAcked {
				t.Errorf("acknowledged = %v, want %v", resp.Acknowledged, tc.wantAcked)
			}
		})
	}
}
