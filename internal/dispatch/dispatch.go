package dispatch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"github.com/HerbHall/subnetree/internal/ca"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin       = (*Module)(nil)
	_ plugin.HTTPProvider = (*Module)(nil)
)

// Module implements the Dispatch agent management plugin.
type Module struct {
	logger     *zap.Logger
	config     plugin.Config
	cfg        DispatchConfig
	store      *DispatchStore
	bus        plugin.EventBus
	authority  *ca.Authority
	grpcServer *grpc.Server
	grpcLis    net.Listener
}

// New creates a new Dispatch plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "dispatch",
		Version:     "0.1.0",
		Description: "Scout agent enrollment and management",
		Roles:       []string{"agent_management"},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.config = deps.Config
	m.logger = deps.Logger

	m.cfg = DefaultConfig()
	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&m.cfg); err != nil {
			return fmt.Errorf("unmarshal dispatch config: %w", err)
		}
	}

	if deps.Store != nil {
		if err := deps.Store.Migrate(context.Background(), "dispatch", migrations()); err != nil {
			return fmt.Errorf("dispatch migrations: %w", err)
		}
		m.store = NewDispatchStore(deps.Store.DB())
	}

	m.bus = deps.Bus

	// Initialize the internal CA for agent certificate management.
	// CA is optional -- enrollment works without it (no mTLS certs issued).
	if m.cfg.CAConfig.CertPath != "" && m.cfg.CAConfig.KeyPath != "" {
		authority, err := ca.LoadOrGenerate(m.cfg.CAConfig, m.logger.Named("ca"))
		if err != nil {
			m.logger.Warn("CA initialization failed; mTLS certificate issuance disabled",
				zap.Error(err),
			)
		} else {
			m.authority = authority
			caCert := m.authority.CACert()
			m.logger.Info("CA initialized",
				zap.String("ca_serial", hex.EncodeToString(caCert.SerialNumber.Bytes())),
				zap.Time("ca_expires", caCert.NotAfter),
			)
		}
	}

	// Generate server certificate for gRPC TLS if CA is available and TLS is enabled.
	if m.cfg.TLSEnabled && m.authority != nil {
		if err := m.ensureServerCert(); err != nil {
			m.logger.Warn("failed to generate server certificate; TLS will be disabled",
				zap.Error(err),
			)
			m.cfg.TLSEnabled = false
		}
	}

	m.logger.Info("dispatch module initialized",
		zap.String("grpc_addr", m.cfg.GRPCAddr),
		zap.Duration("agent_timeout", m.cfg.AgentTimeout),
		zap.Duration("enrollment_token_expiry", m.cfg.EnrollmentTokenExpiry),
		zap.Bool("ca_enabled", m.authority != nil),
		zap.Bool("tls_enabled", m.cfg.TLSEnabled),
	)
	return nil
}

func (m *Module) Start(_ context.Context) error {
	if m.store == nil {
		m.logger.Info("dispatch module started (no store, gRPC disabled)")
		return nil
	}

	lis, err := net.Listen("tcp", m.cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen gRPC %s: %w", m.cfg.GRPCAddr, err)
	}
	m.grpcLis = lis

	m.grpcServer = m.createGRPCServer()
	scoutpb.RegisterScoutServiceServer(m.grpcServer, &scoutServer{
		store:     m.store,
		bus:       m.bus,
		logger:    m.logger.Named("grpc"),
		cfg:       m.cfg,
		authority: m.authority,
	})

	go func() {
		m.logger.Info("gRPC server listening",
			zap.String("addr", m.cfg.GRPCAddr),
			zap.Bool("tls", m.cfg.TLSEnabled),
		)
		if err := m.grpcServer.Serve(lis); err != nil {
			m.logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	m.logger.Info("dispatch module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	if m.grpcServer != nil {
		m.grpcServer.GracefulStop()
		m.logger.Info("gRPC server stopped")
	}
	m.logger.Info("dispatch module stopped")
	return nil
}

// createGRPCServer creates a gRPC server with optional TLS/mTLS.
// When TLS is enabled and the CA is available, the server presents a certificate
// and optionally verifies client certificates (VerifyClientCertIfGiven).
// This allows new agents to connect without a cert for enrollment, while
// enrolled agents can present their client certificate for mTLS.
func (m *Module) createGRPCServer() *grpc.Server {
	if !m.cfg.TLSEnabled || m.authority == nil {
		return grpc.NewServer()
	}

	serverCert, err := tls.LoadX509KeyPair(m.cfg.ServerCertPath, m.cfg.ServerKeyPath)
	if err != nil {
		m.logger.Warn("failed to load server TLS certificate, falling back to insecure",
			zap.Error(err),
		)
		return grpc.NewServer()
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(m.authority.CACertPEM())

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS12,
	}

	return grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsCfg)))
}

// ensureServerCert generates the gRPC server certificate if it doesn't exist.
// The certificate is signed by the internal CA and valid for 365 days.
func (m *Module) ensureServerCert() error {
	// Check if cert already exists.
	if m.cfg.ServerCertPath != "" && m.cfg.ServerKeyPath != "" {
		if _, err := os.Stat(m.cfg.ServerCertPath); err == nil {
			m.logger.Info("server TLS certificate already exists",
				zap.String("cert_path", m.cfg.ServerCertPath),
			)
			return nil
		}
	}

	// Generate default paths if not configured.
	if m.cfg.ServerCertPath == "" {
		m.cfg.ServerCertPath = filepath.Join(filepath.Dir(m.cfg.CAConfig.CertPath), "server.crt")
	}
	if m.cfg.ServerKeyPath == "" {
		m.cfg.ServerKeyPath = filepath.Join(filepath.Dir(m.cfg.CAConfig.KeyPath), "server.key")
	}

	// Generate server keypair.
	key, _, err := ca.GenerateKeypair()
	if err != nil {
		return fmt.Errorf("generate server keypair: %w", err)
	}

	// Create CSR with server identity.
	host, _ := os.Hostname()
	csrDER, err := ca.CreateCSR(key, "dispatch-server", host)
	if err != nil {
		return fmt.Errorf("create server CSR: %w", err)
	}

	// Sign with the CA (365 days validity for server cert).
	const serverCertValidity = 365 * 24 * time.Hour
	certDER, serial, expiresAt, err := m.authority.SignCSR(csrDER, "dispatch-server", serverCertValidity)
	if err != nil {
		return fmt.Errorf("sign server certificate: %w", err)
	}

	// Save key.
	keyPEM, err := ca.EncodeKeyPEM(key)
	if err != nil {
		return fmt.Errorf("encode server key: %w", err)
	}
	keyDir := filepath.Dir(m.cfg.ServerKeyPath)
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("create server key directory: %w", err)
	}
	if err := os.WriteFile(m.cfg.ServerKeyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("write server key: %w", err)
	}

	// Save cert.
	if err := ca.SavePEM(m.cfg.ServerCertPath, "CERTIFICATE", certDER); err != nil {
		return fmt.Errorf("write server certificate: %w", err)
	}

	m.logger.Info("generated server TLS certificate",
		zap.String("cert_serial", serial),
		zap.Time("cert_expires", expiresAt),
		zap.String("cert_path", m.cfg.ServerCertPath),
		zap.String("key_path", m.cfg.ServerKeyPath),
	)

	return nil
}

// Routes is implemented in handlers.go.
