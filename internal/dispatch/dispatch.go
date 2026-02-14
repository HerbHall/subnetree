package dispatch

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"github.com/HerbHall/subnetree/internal/ca"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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

	m.logger.Info("dispatch module initialized",
		zap.String("grpc_addr", m.cfg.GRPCAddr),
		zap.Duration("agent_timeout", m.cfg.AgentTimeout),
		zap.Duration("enrollment_token_expiry", m.cfg.EnrollmentTokenExpiry),
		zap.Bool("ca_enabled", m.authority != nil),
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

	m.grpcServer = grpc.NewServer()
	scoutpb.RegisterScoutServiceServer(m.grpcServer, &scoutServer{
		store:     m.store,
		bus:       m.bus,
		logger:    m.logger.Named("grpc"),
		cfg:       m.cfg,
		authority: m.authority,
	})

	go func() {
		m.logger.Info("gRPC server listening", zap.String("addr", m.cfg.GRPCAddr))
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

// Routes is implemented in handlers.go.
