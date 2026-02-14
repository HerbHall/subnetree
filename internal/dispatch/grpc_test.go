package dispatch

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"github.com/HerbHall/subnetree/internal/ca"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

// testTLSServerWithCA creates a TLS-enabled gRPC server and returns a client connected over TLS.
// The returned client does NOT present a client certificate (simulates enrollment).
// Uses InsecureSkipVerify on the client since the test CA-signed server cert
// lacks IP SANs (SignCSR doesn't copy SANs from the CSR).
func testTLSServerWithCA(t *testing.T, authority *ca.Authority) (scoutpb.ScoutServiceClient, *DispatchStore) {
	t.Helper()

	store := testStore(t)
	dir := t.TempDir()

	// Generate server certificate signed by the CA.
	serverKey, _, err := ca.GenerateKeypair()
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}
	serverCSR, err := ca.CreateCSR(serverKey, "dispatch-server", "localhost")
	if err != nil {
		t.Fatalf("create server CSR: %v", err)
	}
	serverCertDER, _, _, err := authority.SignCSR(serverCSR, "dispatch-server", 24*time.Hour)
	if err != nil {
		t.Fatalf("sign server cert: %v", err)
	}

	// Write server cert and key to temp files.
	serverCertPath := filepath.Join(dir, "server.crt")
	serverKeyPath := filepath.Join(dir, "server.key")
	if err := ca.SavePEM(serverCertPath, "CERTIFICATE", serverCertDER); err != nil {
		t.Fatalf("save server cert: %v", err)
	}
	serverKeyPEM, err := ca.EncodeKeyPEM(serverKey)
	if err != nil {
		t.Fatalf("encode server key: %v", err)
	}
	if err := os.WriteFile(serverKeyPath, serverKeyPEM, 0o600); err != nil {
		t.Fatalf("write server key: %v", err)
	}

	// Create TLS-enabled gRPC server.
	serverCert, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath)
	if err != nil {
		t.Fatalf("load server keypair: %v", err)
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(authority.CACertPEM())

	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS12,
	}

	// Use a real TCP listener for TLS (bufconn doesn't support TLS handshake).
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { lis.Close() })
	addr := lis.Addr().String()

	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(serverTLS)))
	scoutpb.RegisterScoutServiceServer(srv, &scoutServer{
		store:     store,
		logger:    zap.NewNop(),
		cfg:       DefaultConfig(),
		authority: authority,
	})
	t.Cleanup(func() { srv.Stop() })
	go func() { _ = srv.Serve(lis) }()

	// Client connects with TLS but WITHOUT a client certificate.
	// InsecureSkipVerify is used in tests because the CA-signed server cert
	// doesn't include IP SANs for 127.0.0.1 (SignCSR doesn't copy SANs).
	clientTLS := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // G402: test-only, server cert lacks IP SANs
		MinVersion:         tls.VersionTLS12,
	}
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(credentials.NewTLS(clientTLS)),
	)
	if err != nil {
		t.Fatalf("dial TLS: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return scoutpb.NewScoutServiceClient(conn), store
}

func TestGRPC_TLS_EnrollmentWithoutClientCert(t *testing.T) {
	authority := testCA(t)
	client, store := testTLSServerWithCA(t, authority)
	ctx := context.Background()

	// Create enrollment token.
	rawToken := "tls-enrollment-token"
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	if err := store.CreateEnrollmentToken(ctx, &EnrollmentToken{
		ID:          "tok-tls",
		TokenHash:   tokenHash,
		Description: "TLS enrollment token",
		CreatedAt:   now,
		ExpiresAt:   &expires,
		MaxUses:     1,
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Enroll over TLS without a client cert (first connection).
	resp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		Hostname:     "tls-agent-host",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		EnrollToken:  rawToken,
	})
	if err != nil {
		t.Fatalf("CheckIn (TLS enrollment): %v", err)
	}

	if !resp.Acknowledged {
		t.Error("expected acknowledged=true")
	}
	if resp.AssignedAgentId == "" {
		t.Fatal("expected non-empty assigned_agent_id")
	}
}

func TestGRPC_TLS_EnrollmentWithCSR(t *testing.T) {
	authority := testCA(t)
	client, store := testTLSServerWithCA(t, authority)
	ctx := context.Background()

	// Create enrollment token.
	rawToken := "tls-csr-enrollment-token"
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	if err := store.CreateEnrollmentToken(ctx, &EnrollmentToken{
		ID:          "tok-tls-csr",
		TokenHash:   tokenHash,
		Description: "TLS CSR enrollment token",
		CreatedAt:   now,
		ExpiresAt:   &expires,
		MaxUses:     1,
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	csrDER := testCSR(t, "pending-tls-agent")

	// Enroll over TLS with CSR.
	resp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		Hostname:           "tls-csr-agent",
		Platform:           "linux/amd64",
		AgentVersion:       "0.1.0",
		ProtoVersion:       1,
		EnrollToken:        rawToken,
		CertificateRequest: csrDER,
	})
	if err != nil {
		t.Fatalf("CheckIn (TLS enrollment with CSR): %v", err)
	}

	if !resp.Acknowledged {
		t.Error("expected acknowledged=true")
	}
	if resp.AssignedAgentId == "" {
		t.Fatal("expected non-empty assigned_agent_id")
	}
	if len(resp.SignedCertificate) == 0 {
		t.Fatal("expected signed certificate from TLS enrollment with CSR")
	}
	if len(resp.CaCertificate) == 0 {
		t.Fatal("expected CA certificate from TLS enrollment with CSR")
	}
}

func TestGRPC_FullEnrollmentThenRenewal(t *testing.T) {
	authority := testCA(t)
	client, store := testGRPCServerWithCA(t, authority)
	ctx := context.Background()

	// --- Step 1: Enrollment with CSR ---
	rawToken := "full-flow-enrollment-token"
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	if err := store.CreateEnrollmentToken(ctx, &EnrollmentToken{
		ID:          "tok-full-flow",
		TokenHash:   tokenHash,
		Description: "full flow enrollment token",
		CreatedAt:   now,
		ExpiresAt:   &expires,
		MaxUses:     1,
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	enrollCSR := testCSR(t, "pending-agent")

	enrollResp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		Hostname:           "full-flow-host",
		Platform:           "linux/amd64",
		AgentVersion:       "0.1.0",
		ProtoVersion:       1,
		EnrollToken:        rawToken,
		CertificateRequest: enrollCSR,
	})
	if err != nil {
		t.Fatalf("enrollment CheckIn: %v", err)
	}
	if !enrollResp.Acknowledged {
		t.Fatal("enrollment not acknowledged")
	}
	if enrollResp.AssignedAgentId == "" {
		t.Fatal("expected non-empty assigned_agent_id from enrollment")
	}
	if len(enrollResp.SignedCertificate) == 0 {
		t.Fatal("expected signed certificate from enrollment")
	}

	agentID := enrollResp.AssignedAgentId

	// Parse and record initial certificate details.
	initialCert, err := x509.ParseCertificate(enrollResp.SignedCertificate)
	if err != nil {
		t.Fatalf("parse initial cert: %v", err)
	}
	initialSerial := initialCert.SerialNumber.String()

	// Verify agent record.
	agent, err := store.GetAgent(ctx, agentID)
	if err != nil {
		t.Fatalf("GetAgent after enrollment: %v", err)
	}
	if agent.CertSerial == "" {
		t.Error("cert_serial should be populated after enrollment")
	}
	if agent.CertExpires == nil {
		t.Error("cert_expires should be populated after enrollment")
	}
	enrollSerial := agent.CertSerial

	// --- Step 2: Renewal with new CSR ---
	renewCSR := testCSR(t, agentID)

	renewResp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		AgentId:            agentID,
		Hostname:           "full-flow-host",
		Platform:           "linux/amd64",
		AgentVersion:       "0.1.0",
		ProtoVersion:       1,
		CertificateRequest: renewCSR,
	})
	if err != nil {
		t.Fatalf("renewal CheckIn: %v", err)
	}
	if !renewResp.Acknowledged {
		t.Fatal("renewal not acknowledged")
	}
	// Renewal should NOT assign a new agent ID.
	if renewResp.AssignedAgentId != "" {
		t.Errorf("assigned_agent_id = %q, want empty for renewal", renewResp.AssignedAgentId)
	}
	if len(renewResp.SignedCertificate) == 0 {
		t.Fatal("expected signed certificate from renewal")
	}
	if len(renewResp.CaCertificate) == 0 {
		t.Fatal("expected CA certificate from renewal")
	}

	// Parse renewal certificate.
	renewedCert, err := x509.ParseCertificate(renewResp.SignedCertificate)
	if err != nil {
		t.Fatalf("parse renewed cert: %v", err)
	}

	// Verify serial numbers differ.
	renewedSerial := renewedCert.SerialNumber.String()
	if initialSerial == renewedSerial {
		t.Errorf("renewed cert serial (%s) should differ from initial (%s)", renewedSerial, initialSerial)
	}

	// Verify CN matches agent ID.
	if renewedCert.Subject.CommonName != agentID {
		t.Errorf("renewed cert CN = %q, want %q", renewedCert.Subject.CommonName, agentID)
	}

	// Verify renewed cert chains to CA.
	caCert, err := x509.ParseCertificate(renewResp.CaCertificate)
	if err != nil {
		t.Fatalf("parse CA cert: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	if _, verifyErr := renewedCert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); verifyErr != nil {
		t.Errorf("renewed cert does not chain to CA: %v", verifyErr)
	}

	// Verify store was updated.
	agentAfter, err := store.GetAgent(ctx, agentID)
	if err != nil {
		t.Fatalf("GetAgent after renewal: %v", err)
	}
	if agentAfter.CertSerial == enrollSerial {
		t.Error("cert_serial should be different after renewal")
	}
	if agentAfter.CertSerial == "" {
		t.Error("cert_serial should not be empty after renewal")
	}
	if agentAfter.CertExpires == nil {
		t.Fatal("cert_expires should not be nil after renewal")
	}
	if agent.CertExpires != nil && !agentAfter.CertExpires.After(*agent.CertExpires) {
		t.Errorf("cert_expires after renewal (%v) should be after enrollment (%v)",
			agentAfter.CertExpires, agent.CertExpires)
	}
}

func TestGRPC_RenewalSerialDiffers(t *testing.T) {
	authority := testCA(t)
	client, store := testGRPCServerWithCA(t, authority)
	ctx := context.Background()

	// Pre-create an agent.
	now := time.Now().UTC()
	oldExpiry := now.Add(24 * time.Hour)
	if err := store.UpsertAgent(ctx, &Agent{
		ID:           "agent-serial-test",
		Hostname:     "serial-host",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		Status:       "connected",
		EnrolledAt:   now,
		CertSerial:   "initial-serial",
		CertExpires:  &oldExpiry,
		ConfigJSON:   "{}",
	}); err != nil {
		t.Fatalf("setup agent: %v", err)
	}

	// First renewal.
	csr1 := testCSR(t, "agent-serial-test")
	resp1, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		AgentId:            "agent-serial-test",
		Hostname:           "serial-host",
		Platform:           "linux/amd64",
		AgentVersion:       "0.1.0",
		ProtoVersion:       1,
		CertificateRequest: csr1,
	})
	if err != nil {
		t.Fatalf("first renewal: %v", err)
	}
	if len(resp1.SignedCertificate) == 0 {
		t.Fatal("expected cert from first renewal")
	}

	cert1, err := x509.ParseCertificate(resp1.SignedCertificate)
	if err != nil {
		t.Fatalf("parse first renewal cert: %v", err)
	}

	// Second renewal.
	csr2 := testCSR(t, "agent-serial-test")
	resp2, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		AgentId:            "agent-serial-test",
		Hostname:           "serial-host",
		Platform:           "linux/amd64",
		AgentVersion:       "0.1.0",
		ProtoVersion:       1,
		CertificateRequest: csr2,
	})
	if err != nil {
		t.Fatalf("second renewal: %v", err)
	}
	if len(resp2.SignedCertificate) == 0 {
		t.Fatal("expected cert from second renewal")
	}

	cert2, err := x509.ParseCertificate(resp2.SignedCertificate)
	if err != nil {
		t.Fatalf("parse second renewal cert: %v", err)
	}

	// Verify serials differ between the two renewals.
	if cert1.SerialNumber.Cmp(cert2.SerialNumber) == 0 {
		t.Error("serial numbers from two consecutive renewals should differ")
	}
}

func TestGRPC_CheckInWithoutCSR_NoCertReturned(t *testing.T) {
	authority := testCA(t)
	client, store := testGRPCServerWithCA(t, authority)
	ctx := context.Background()

	// Pre-create an agent (already enrolled, no CSR in check-in).
	now := time.Now().UTC()
	if err := store.UpsertAgent(ctx, &Agent{
		ID:           "agent-no-csr",
		Hostname:     "no-csr-host",
		Platform:     "linux/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		Status:       "connected",
		EnrolledAt:   now,
		ConfigJSON:   "{}",
	}); err != nil {
		t.Fatalf("setup agent: %v", err)
	}

	// Normal check-in without CSR.
	resp, err := client.CheckIn(ctx, &scoutpb.CheckInRequest{
		AgentId:      "agent-no-csr",
		Hostname:     "no-csr-host",
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
	if len(resp.SignedCertificate) != 0 {
		t.Errorf("expected no signed_certificate for normal check-in, got %d bytes", len(resp.SignedCertificate))
	}
	if len(resp.CaCertificate) != 0 {
		t.Errorf("expected no ca_certificate for normal check-in, got %d bytes", len(resp.CaCertificate))
	}
}

func TestModule_EnsureServerCert(t *testing.T) {
	dir := t.TempDir()
	authority := testCA(t)

	m := &Module{
		logger:    zap.NewNop(),
		authority: authority,
		cfg: DispatchConfig{
			TLSEnabled:     true,
			ServerCertPath: filepath.Join(dir, "server.crt"),
			ServerKeyPath:  filepath.Join(dir, "server.key"),
			CAConfig: ca.Config{
				CertPath: filepath.Join(dir, "ca.crt"),
				KeyPath:  filepath.Join(dir, "ca.key"),
			},
		},
	}

	if err := m.ensureServerCert(); err != nil {
		t.Fatalf("ensureServerCert: %v", err)
	}

	// Verify cert and key were created.
	if _, err := os.Stat(m.cfg.ServerCertPath); err != nil {
		t.Errorf("server cert not created: %v", err)
	}
	if _, err := os.Stat(m.cfg.ServerKeyPath); err != nil {
		t.Errorf("server key not created: %v", err)
	}

	// Verify the cert can be loaded.
	_, err := tls.LoadX509KeyPair(m.cfg.ServerCertPath, m.cfg.ServerKeyPath)
	if err != nil {
		t.Fatalf("load generated server keypair: %v", err)
	}

	// Calling again should be a no-op (cert already exists).
	if err := m.ensureServerCert(); err != nil {
		t.Fatalf("ensureServerCert (second call): %v", err)
	}
}

func TestModule_CreateGRPCServer_InsecureFallback(t *testing.T) {
	m := &Module{
		logger: zap.NewNop(),
		cfg: DispatchConfig{
			TLSEnabled: false,
		},
	}

	srv := m.createGRPCServer()
	if srv == nil {
		t.Fatal("expected non-nil gRPC server")
	}
	srv.Stop()
}

func TestExtractAgentIDFromCert_NoPeer(t *testing.T) {
	ctx := context.Background()
	cn, ok := extractAgentIDFromCert(ctx)
	if ok {
		t.Errorf("expected ok=false for context without peer, got cn=%q", cn)
	}
}
