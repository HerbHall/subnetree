package dispatch

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"github.com/HerbHall/subnetree/internal/ca"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// testCA creates a temporary CA for testing.
func testCA(t *testing.T) *ca.Authority {
	t.Helper()
	dir := t.TempDir()
	cfg := ca.Config{
		CertPath:     filepath.Join(dir, "ca.crt"),
		KeyPath:      filepath.Join(dir, "ca.key"),
		Organization: "TestOrg",
	}
	authority, err := ca.GenerateCA(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("generate test CA: %v", err)
	}
	return authority
}

// testCSR generates an ECDSA keypair and returns a DER-encoded CSR.
func testCSR(t *testing.T, agentID string) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate agent key: %v", err)
	}
	csrDER, err := ca.CreateCSR(key, agentID, "test-host")
	if err != nil {
		t.Fatalf("create CSR: %v", err)
	}
	return csrDER
}

func testGRPCServer(t *testing.T) (scoutpb.ScoutServiceClient, *DispatchStore) {
	t.Helper()
	return testGRPCServerWithCA(t, nil)
}

func testGRPCServerWithCA(t *testing.T, authority *ca.Authority) (scoutpb.ScoutServiceClient, *DispatchStore) {
	t.Helper()

	store := testStore(t)

	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { lis.Close() })

	srv := grpc.NewServer()
	scoutpb.RegisterScoutServiceServer(srv, &scoutServer{
		store:     store,
		logger:    zap.NewNop(),
		cfg:       DefaultConfig(),
		authority: authority,
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

	// Enroll with the token (no CSR -- backward compat).
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

	// No CSR was sent, so no certificate should be returned.
	if len(resp.SignedCertificate) != 0 {
		t.Errorf("expected empty signed_certificate without CSR, got %d bytes", len(resp.SignedCertificate))
	}
	if len(resp.CaCertificate) != 0 {
		t.Errorf("expected empty ca_certificate without CSR, got %d bytes", len(resp.CaCertificate))
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
	// No CSR => no cert fields.
	if agent.CertSerial != "" {
		t.Errorf("cert_serial = %q, want empty (no CSR)", agent.CertSerial)
	}
	if agent.CertExpires != nil {
		t.Errorf("cert_expires = %v, want nil (no CSR)", agent.CertExpires)
	}
}

func TestGRPC_CheckIn_EnrollmentWithCSR(t *testing.T) {
	authority := testCA(t)
	client, store := testGRPCServerWithCA(t, authority)
	ctx := context.Background()

	// Create enrollment token.
	rawToken := "test-enrollment-csr-token"
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	if err := store.CreateEnrollmentToken(ctx, &EnrollmentToken{
		ID:          "tok-csr",
		TokenHash:   tokenHash,
		Description: "CSR enrollment token",
		CreatedAt:   now,
		ExpiresAt:   &expires,
		MaxUses:     1,
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Generate a CSR for enrollment. Use a placeholder agent ID since
	// the real one is assigned by the server.
	csrDER := testCSR(t, "pending-agent")

	// Enroll with CSR.
	resp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		Hostname:           "csr-agent-host",
		Platform:           "linux/amd64",
		AgentVersion:       "0.1.0",
		ProtoVersion:       1,
		EnrollToken:        rawToken,
		CertificateRequest: csrDER,
	})
	if err != nil {
		t.Fatalf("CheckIn (enroll with CSR): %v", err)
	}

	if !resp.Acknowledged {
		t.Error("expected acknowledged=true")
	}
	if resp.AssignedAgentId == "" {
		t.Fatal("expected non-empty assigned_agent_id")
	}

	// Verify signed certificate was returned.
	if len(resp.SignedCertificate) == 0 {
		t.Fatal("expected non-empty signed_certificate")
	}
	if len(resp.CaCertificate) == 0 {
		t.Fatal("expected non-empty ca_certificate")
	}

	// Parse and verify the signed certificate.
	cert, err := x509.ParseCertificate(resp.SignedCertificate)
	if err != nil {
		t.Fatalf("parse signed certificate: %v", err)
	}
	if cert.Subject.CommonName != resp.AssignedAgentId {
		t.Errorf("cert CN = %q, want %q", cert.Subject.CommonName, resp.AssignedAgentId)
	}

	// Verify the CA cert can be parsed.
	caCert, err := x509.ParseCertificate(resp.CaCertificate)
	if err != nil {
		t.Fatalf("parse CA certificate: %v", err)
	}
	if !caCert.IsCA {
		t.Error("expected CA certificate to have IsCA=true")
	}

	// Verify the agent cert chains to the CA.
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	if _, err := cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		t.Errorf("agent cert does not chain to CA: %v", err)
	}

	// Verify agent record has cert fields populated.
	agent, err := store.GetAgent(ctx, resp.AssignedAgentId)
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent in store")
	}
	if agent.CertSerial == "" {
		t.Error("cert_serial is empty, expected populated after CSR enrollment")
	}
	if agent.CertExpires == nil {
		t.Error("cert_expires is nil, expected populated after CSR enrollment")
	}
	if agent.CertExpires != nil && agent.CertExpires.Before(time.Now()) {
		t.Error("cert_expires is in the past")
	}
}

func TestGRPC_CheckIn_RenewalWithCSR(t *testing.T) {
	authority := testCA(t)
	client, store := testGRPCServerWithCA(t, authority)
	ctx := context.Background()

	// Pre-create an agent with an existing certificate.
	now := time.Now().UTC()
	oldExpiry := now.Add(24 * time.Hour)
	if err := store.UpsertAgent(ctx, &Agent{
		ID:           "agent-renew",
		Hostname:     "renew-host",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		Status:       "connected",
		EnrolledAt:   now,
		CertSerial:   "old-serial-abc",
		CertExpires:  &oldExpiry,
		ConfigJSON:   "{}",
	}); err != nil {
		t.Fatalf("setup agent: %v", err)
	}

	// Generate a new CSR for renewal.
	csrDER := testCSR(t, "agent-renew")

	// Check in with CSR (certificate renewal).
	resp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		AgentId:            "agent-renew",
		Hostname:           "renew-host",
		Platform:           "linux/amd64",
		AgentVersion:       "0.1.0",
		ProtoVersion:       1,
		CertificateRequest: csrDER,
	})
	if err != nil {
		t.Fatalf("CheckIn (renewal): %v", err)
	}

	if !resp.Acknowledged {
		t.Error("expected acknowledged=true")
	}
	// Not a new enrollment -- assigned_agent_id should be empty.
	if resp.AssignedAgentId != "" {
		t.Errorf("assigned_agent_id = %q, want empty for renewal", resp.AssignedAgentId)
	}

	// Verify renewed certificate was returned.
	if len(resp.SignedCertificate) == 0 {
		t.Fatal("expected non-empty signed_certificate for renewal")
	}
	if len(resp.CaCertificate) == 0 {
		t.Fatal("expected non-empty ca_certificate for renewal")
	}

	// Verify the agent record was updated with new cert fields.
	agent, err := store.GetAgent(ctx, "agent-renew")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if agent == nil {
		t.Fatal("expected agent in store")
	}
	if agent.CertSerial == "old-serial-abc" {
		t.Error("cert_serial was not updated after renewal")
	}
	if agent.CertSerial == "" {
		t.Error("cert_serial is empty after renewal")
	}
	if agent.CertExpires == nil {
		t.Fatal("cert_expires is nil after renewal")
	}
	if !agent.CertExpires.After(oldExpiry) {
		t.Errorf("cert_expires (%v) should be after old expiry (%v)", agent.CertExpires, oldExpiry)
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
