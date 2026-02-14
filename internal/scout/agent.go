package scout

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"github.com/HerbHall/subnetree/internal/ca"
	"github.com/HerbHall/subnetree/internal/scout/metrics"
	"github.com/HerbHall/subnetree/internal/scout/profiler"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	grpcinsecure "google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Agent is the Scout monitoring agent.
type Agent struct {
	config    *Config
	logger    *zap.Logger
	cancel    context.CancelFunc
	conn      *grpc.ClientConn
	client    scoutpb.ScoutServiceClient
	collector metrics.Collector
	profiler  *profiler.Profiler
}

// NewAgent creates a new Scout agent instance.
func NewAgent(config *Config, logger *zap.Logger) *Agent {
	return &Agent{
		config:    config,
		logger:    logger,
		collector: metrics.NewCollector(logger),
		profiler:  profiler.NewProfiler(logger.Named("profiler")),
	}
}

// Run starts the agent and blocks until the context is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	ctx, a.cancel = context.WithCancel(ctx)

	// Load persisted agent ID.
	a.loadAgentID()

	// Connect with exponential backoff.
	// For new agents (no agent ID), connect insecure for enrollment.
	// For existing agents with certs, connect with mTLS.
	if err := a.connectWithBackoff(ctx); err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}
	defer func() {
		if a.conn != nil {
			a.conn.Close()
		}
	}()

	// If no agent ID, attempt enrollment.
	if a.config.AgentID == "" {
		if a.config.EnrollToken == "" {
			return fmt.Errorf("no agent ID and no enrollment token provided")
		}
		if err := a.enroll(ctx); err != nil {
			return fmt.Errorf("enrollment: %w", err)
		}

		// After enrollment, if we received certificates, reconnect with mTLS.
		if !a.config.Insecure && a.hasTLSCredentials() {
			a.logger.Info("reconnecting with mTLS after enrollment")
			a.conn.Close()
			if err := a.connectWithBackoff(ctx); err != nil {
				return fmt.Errorf("reconnect with mTLS: %w", err)
			}
		}
	}

	a.logger.Info("agent running",
		zap.String("agent_id", a.config.AgentID),
		zap.String("server", a.config.ServerAddr),
		zap.Int("interval", a.config.CheckInterval),
	)

	ticker := time.NewTicker(time.Duration(a.config.CheckInterval) * time.Second)
	defer ticker.Stop()

	// Profile collection: hardware at startup, full refresh periodically.
	const profileInterval = 6 * time.Hour
	profileTicker := time.NewTicker(profileInterval)
	defer profileTicker.Stop()

	// Initial check-in.
	a.checkIn(ctx)

	// Initial profile collection after startup.
	a.collectAndSendProfile(ctx)

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("agent shutting down")
			return nil
		case <-ticker.C:
			a.checkIn(ctx)
		case <-profileTicker.C:
			a.collectAndSendProfile(ctx)
		}
	}
}

// Stop signals the agent to shut down.
func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

// dialGRPC creates a gRPC client connection with the appropriate transport credentials.
// In insecure mode or when no TLS files exist, it uses plaintext.
// Otherwise, it loads the agent certificate and CA cert for mTLS.
func (a *Agent) dialGRPC() (*grpc.ClientConn, error) {
	if a.config.Insecure || !a.hasTLSCredentials() {
		return grpc.NewClient(
			a.config.ServerAddr,
			grpc.WithTransportCredentials(grpcinsecure.NewCredentials()),
		)
	}

	tlsCfg, err := a.buildTLSConfig()
	if err != nil {
		a.logger.Warn("failed to build TLS config, falling back to insecure", zap.Error(err))
		return grpc.NewClient(
			a.config.ServerAddr,
			grpc.WithTransportCredentials(grpcinsecure.NewCredentials()),
		)
	}

	return grpc.NewClient(
		a.config.ServerAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
	)
}

// buildTLSConfig loads the agent certificate, key, and CA certificate to create
// a TLS configuration for mTLS connections.
func (a *Agent) buildTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(a.config.CertPath, a.config.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("load agent certificate/key: %w", err)
	}

	caCertPath := a.config.ResolveCACertPath()
	caCertPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("read CA certificate %s: %w", caCertPath, err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate from %s", caCertPath)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// hasTLSCredentials returns true if the agent has cert and key files on disk.
func (a *Agent) hasTLSCredentials() bool {
	if a.config.CertPath == "" || a.config.KeyPath == "" {
		return false
	}
	if _, err := os.Stat(a.config.CertPath); err != nil {
		return false
	}
	if _, err := os.Stat(a.config.KeyPath); err != nil {
		return false
	}
	return true
}

func (a *Agent) connectWithBackoff(ctx context.Context) error {
	backoff := time.Second
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := a.dialGRPC()
		if err == nil {
			a.conn = conn
			a.client = scoutpb.NewScoutServiceClient(conn)
			a.logger.Info("connected to server",
				zap.String("addr", a.config.ServerAddr),
				zap.Bool("tls", !a.config.Insecure && a.hasTLSCredentials()),
			)
			return nil
		}

		a.logger.Warn("connection failed, retrying",
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
	}
}

func (a *Agent) enroll(ctx context.Context) error {
	req := &scoutpb.CheckInRequest{
		Hostname:     hostname(),
		Platform:     agentPlatform(),
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		EnrollToken:  a.config.EnrollToken,
	}

	// Generate keypair and CSR for mTLS enrollment when TLS paths are configured.
	if !a.config.Insecure && a.config.CertPath != "" && a.config.KeyPath != "" {
		key, _, err := ca.GenerateKeypair()
		if err != nil {
			a.logger.Warn("failed to generate keypair for enrollment, continuing without CSR", zap.Error(err))
		} else {
			csrDER, csrErr := ca.CreateCSR(key, "", hostname())
			if csrErr != nil {
				a.logger.Warn("failed to create CSR for enrollment, continuing without CSR", zap.Error(csrErr))
			} else {
				// Save private key to disk before sending CSR.
				keyPEM, encErr := ca.EncodeKeyPEM(key)
				if encErr != nil {
					a.logger.Warn("failed to encode private key, continuing without CSR", zap.Error(encErr))
				} else {
					keyDir := filepath.Dir(a.config.KeyPath)
					if mkErr := os.MkdirAll(keyDir, 0o700); mkErr != nil {
						a.logger.Warn("failed to create key directory", zap.Error(mkErr))
					} else if wErr := os.WriteFile(a.config.KeyPath, keyPEM, 0o600); wErr != nil {
						a.logger.Warn("failed to save private key", zap.Error(wErr))
					} else {
						req.CertificateRequest = csrDER
						a.logger.Info("generated keypair and CSR for enrollment")
					}
				}
			}
		}
	}

	resp, err := a.client.CheckIn(ctx, req)
	if err != nil {
		return fmt.Errorf("enrollment check-in: %w", err)
	}

	if !resp.Acknowledged {
		return fmt.Errorf("enrollment rejected: %s", resp.UpgradeMessage)
	}

	if resp.AssignedAgentId == "" {
		return fmt.Errorf("server did not assign an agent ID")
	}

	a.config.AgentID = resp.AssignedAgentId
	a.saveAgentID()

	// Save certificate artifacts if the server returned them.
	if len(resp.SignedCertificate) > 0 {
		if err := a.saveCertificates(resp.SignedCertificate, resp.CaCertificate); err != nil {
			a.logger.Warn("failed to save enrollment certificates", zap.Error(err))
		}
	}

	a.logger.Info("enrolled successfully", zap.String("agent_id", a.config.AgentID))
	return nil
}

// saveCertificates writes the signed agent certificate and CA certificate to disk.
func (a *Agent) saveCertificates(certDER, caCertDER []byte) error {
	// Save agent certificate.
	certPEM, err := ca.EncodeCertPEM(certDER)
	if err != nil {
		return fmt.Errorf("encode agent certificate: %w", err)
	}

	certDir := filepath.Dir(a.config.CertPath)
	if err := os.MkdirAll(certDir, 0o700); err != nil {
		return fmt.Errorf("create cert directory: %w", err)
	}
	if err := os.WriteFile(a.config.CertPath, certPEM, 0o600); err != nil {
		return fmt.Errorf("write agent certificate: %w", err)
	}

	// Parse and log certificate details.
	cert, parseErr := x509.ParseCertificate(certDER)
	if parseErr == nil {
		a.logger.Info("saved agent certificate",
			zap.String("cert_cn", cert.Subject.CommonName),
			zap.Time("cert_expires", cert.NotAfter),
			zap.String("cert_path", a.config.CertPath),
		)
	}

	// Save CA certificate if provided.
	if len(caCertDER) > 0 {
		caCertPEM, encErr := ca.EncodeCertPEM(caCertDER)
		if encErr != nil {
			return fmt.Errorf("encode CA certificate: %w", encErr)
		}

		caCertPath := a.config.ResolveCACertPath()
		caCertDir := filepath.Dir(caCertPath)
		if err := os.MkdirAll(caCertDir, 0o700); err != nil {
			return fmt.Errorf("create CA cert directory: %w", err)
		}
		if err := os.WriteFile(caCertPath, caCertPEM, 0o600); err != nil {
			return fmt.Errorf("write CA certificate: %w", err)
		}
		a.logger.Info("saved CA certificate", zap.String("ca_cert_path", caCertPath))
	}

	return nil
}

func (a *Agent) checkIn(ctx context.Context) {
	var sysMetrics *scoutpb.SystemMetrics
	if a.collector != nil {
		m, err := a.collector.Collect(ctx)
		if err != nil {
			a.logger.Warn("metrics collection failed", zap.Error(err))
		} else {
			sysMetrics = m
		}
	}

	// Check if certificate renewal is needed before check-in.
	var renewalCSR []byte
	var renewalKey crypto.PrivateKey
	csrDER, newKey, renewErr := a.checkCertificateRenewal()
	if renewErr != nil {
		a.logger.Warn("certificate renewal check failed", zap.Error(renewErr))
	} else if csrDER != nil {
		renewalCSR = csrDER
		renewalKey = newKey
	}

	req := &scoutpb.CheckInRequest{
		AgentId:            a.config.AgentID,
		Hostname:           hostname(),
		Platform:           agentPlatform(),
		AgentVersion:       "0.1.0",
		ProtoVersion:       1,
		Metrics:            sysMetrics,
		CertificateRequest: renewalCSR,
	}

	resp, err := a.client.CheckIn(ctx, req)
	if err != nil {
		a.logger.Warn("check-in failed", zap.Error(err))
		return
	}

	if !resp.Acknowledged {
		a.logger.Warn("check-in not acknowledged",
			zap.String("version_status", resp.VersionStatus.String()),
			zap.String("message", resp.UpgradeMessage),
		)
	}

	// Handle certificate renewal response.
	if renewalKey != nil && len(resp.SignedCertificate) > 0 {
		// Save new private key first (atomic swap: key then cert).
		if err := a.saveRenewalKey(renewalKey); err != nil {
			a.logger.Error("failed to save renewal key, keeping old certificate", zap.Error(err))
			return
		}

		// Save new certificate and CA cert.
		if err := a.saveCertificates(resp.SignedCertificate, resp.CaCertificate); err != nil {
			a.logger.Error("failed to save renewal certificates", zap.Error(err))
			return
		}

		// Log the new certificate details.
		if newCert, parseErr := x509.ParseCertificate(resp.SignedCertificate); parseErr == nil {
			a.logger.Info("certificate renewed",
				zap.Time("expires", newCert.NotAfter),
			)
		}

		// Reconnect with the new credentials.
		if err := a.reconnectWithNewCerts(ctx); err != nil {
			a.logger.Error("failed to reconnect with new certificates", zap.Error(err))
		}
	}
}

// checkCertificateRenewal checks if the current certificate is expiring soon
// and generates a new CSR if renewal is needed. Returns nil values if no renewal
// is needed or if the agent is in insecure mode.
func (a *Agent) checkCertificateRenewal() (csrDER []byte, newKey crypto.PrivateKey, err error) {
	// Only attempt renewal when TLS is configured and credentials exist.
	if a.config.Insecure || !a.hasTLSCredentials() {
		return nil, nil, nil
	}

	// Load current certificate from disk.
	certDER, loadErr := ca.LoadPEM(a.config.CertPath, "CERTIFICATE")
	if loadErr != nil {
		return nil, nil, fmt.Errorf("load current certificate: %w", loadErr)
	}

	cert, parseErr := x509.ParseCertificate(certDER)
	if parseErr != nil {
		return nil, nil, fmt.Errorf("parse current certificate: %w", parseErr)
	}

	// Check if the certificate is expiring soon.
	if !ca.IsCertificateExpiringSoon(cert, a.config.RenewalThreshold) {
		return nil, nil, nil
	}

	a.logger.Info("certificate expiring soon, initiating renewal",
		zap.Time("expires", cert.NotAfter),
		zap.Duration("threshold", a.config.RenewalThreshold),
	)

	// Generate new keypair and CSR.
	key, _, genErr := ca.GenerateKeypair()
	if genErr != nil {
		return nil, nil, fmt.Errorf("generate renewal keypair: %w", genErr)
	}

	csr, csrErr := ca.CreateCSR(key, a.config.AgentID, hostname())
	if csrErr != nil {
		return nil, nil, fmt.Errorf("create renewal CSR: %w", csrErr)
	}

	return csr, key, nil
}

// reconnectWithNewCerts closes the existing gRPC connection and reconnects
// using the newly saved TLS credentials.
func (a *Agent) reconnectWithNewCerts(ctx context.Context) error {
	if a.conn != nil {
		a.conn.Close()
	}
	return a.connectWithBackoff(ctx)
}

// saveRenewalKey saves the new private key to disk, replacing the old key.
// This should only be called after the server has returned a signed certificate.
func (a *Agent) saveRenewalKey(newKey crypto.PrivateKey) error {
	keyPEM, err := ca.EncodeKeyPEM(newKey)
	if err != nil {
		return fmt.Errorf("encode renewal key: %w", err)
	}

	keyDir := filepath.Dir(a.config.KeyPath)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("create key directory: %w", err)
	}

	if err := os.WriteFile(a.config.KeyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("write renewal key: %w", err)
	}

	return nil
}

func (a *Agent) collectAndSendProfile(ctx context.Context) {
	if a.profiler == nil || a.client == nil {
		return
	}

	profile, err := a.profiler.CollectProfile(ctx)
	if err != nil {
		a.logger.Warn("profile collection failed", zap.Error(err))
		return
	}

	ack, err := a.client.ReportProfile(ctx, &scoutpb.ProfileReport{
		AgentId:     a.config.AgentID,
		CollectedAt: timestamppb.Now(),
		Profile:     profile,
	})
	if err != nil {
		a.logger.Warn("profile report failed", zap.Error(err))
		return
	}

	a.logger.Info("profile reported",
		zap.Bool("success", ack.GetSuccess()),
	)
}

// Agent ID persistence -- simple JSON file in config directory.
type agentState struct {
	AgentID string `json:"agent_id"`
}

func (a *Agent) statePath() string {
	dir := filepath.Dir(a.config.CertPath)
	if dir == "" || dir == "." {
		dir, _ = os.UserConfigDir()
		if dir == "" {
			dir = "."
		}
		dir = filepath.Join(dir, "subnetree-scout")
	}
	return filepath.Join(dir, "agent-state.json")
}

func (a *Agent) loadAgentID() {
	data, err := os.ReadFile(a.statePath())
	if err != nil {
		return
	}
	var state agentState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	if state.AgentID != "" && a.config.AgentID == "" {
		a.config.AgentID = state.AgentID
		a.logger.Info("loaded persisted agent ID", zap.String("agent_id", state.AgentID))
	}
}

func (a *Agent) saveAgentID() {
	data, err := json.Marshal(agentState{AgentID: a.config.AgentID})
	if err != nil {
		a.logger.Warn("failed to marshal agent state", zap.Error(err))
		return
	}
	dir := filepath.Dir(a.statePath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		a.logger.Warn("failed to create state dir", zap.Error(err))
		return
	}
	if err := os.WriteFile(a.statePath(), data, 0o600); err != nil {
		a.logger.Warn("failed to save agent state", zap.Error(err))
	}
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func agentPlatform() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
