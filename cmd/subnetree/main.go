package main

//	@title						SubNetree API
//	@version					0.1.0
//	@description				Network monitoring and management platform API.
//	@BasePath					/api/v1
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				JWT Bearer token. Format: "Bearer {token}"

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/HerbHall/subnetree/api/swagger"
	"github.com/HerbHall/subnetree/internal/auth"
	"github.com/HerbHall/subnetree/internal/config"
	"github.com/HerbHall/subnetree/internal/dashboard"
	"github.com/HerbHall/subnetree/internal/dispatch"
	"github.com/HerbHall/subnetree/internal/docs"
	"github.com/HerbHall/subnetree/internal/event"
	"github.com/HerbHall/subnetree/internal/gateway"
	"github.com/HerbHall/subnetree/internal/insight"
	"github.com/HerbHall/subnetree/internal/llm"
	"github.com/HerbHall/subnetree/internal/pulse"
	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/internal/registry"
	"github.com/HerbHall/subnetree/internal/server"
	"github.com/HerbHall/subnetree/internal/services"
	"github.com/HerbHall/subnetree/internal/settings"
	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/internal/svcmap"
	"github.com/HerbHall/subnetree/internal/vault"
	"github.com/HerbHall/subnetree/internal/version"
	"github.com/HerbHall/subnetree/internal/webhook"
	"github.com/HerbHall/subnetree/internal/ws"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

func main() {
	// Subcommand dispatch (before flag.Parse).
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "backup":
			runBackup(os.Args[2:])
			return
		case "restore":
			runRestore(os.Args[2:])
			return
		case "version":
			fmt.Println(version.Info())
			return
		}
	}

	configPath := flag.String("config", "", "path to configuration file")
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Info())
		os.Exit(0)
	}

	// Load configuration (before logger, so log level/format can be configured).
	viperCfg, err := server.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	cfg := config.New(viperCfg)

	// Initialize logger from configuration.
	logger, err := config.NewLogger(viperCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("SubNetree server starting", zap.String("version", version.Short()))

	if f := viperCfg.ConfigFileUsed(); f != "" {
		logger.Info("configuration loaded",
			zap.String("component", "config"),
			zap.String("source", f),
		)
	} else {
		logger.Warn("no configuration file found, using defaults",
			zap.String("component", "config"),
		)
	}

	// Open database
	dbPath := viperCfg.GetString("database.path")
	if dbPath == "" {
		dbPath = "subnetree.db"
	}
	db, err := store.New(dbPath)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("database initialized",
		zap.String("component", "database"),
		zap.String("path", dbPath),
	)

	// Create shared services
	bus := event.NewBus(logger.Named("event"))
	logger.Info("event bus created", zap.String("component", "event"))

	// Create plugin registry
	reg := registry.New(logger.Named("registry"))
	logger.Info("plugin registry created", zap.String("component", "registry"))

	// Register all plugins (compile-time composition)
	modules := []plugin.Plugin{
		recon.New(),
		pulse.New(),
		dispatch.New(),
		vault.New(),
		gateway.New(),
		webhook.New(),
		llm.New(),
		insight.New(),
		docs.New(),
	}
	for _, m := range modules {
		if err := reg.Register(m); err != nil {
			logger.Fatal("failed to register plugin", zap.Error(err))
		}
	}

	// Validate dependency graph and API versions
	if err := reg.Validate(); err != nil {
		logger.Fatal("plugin validation failed", zap.Error(err))
	}

	// Initialize all plugins with dependencies
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := reg.InitAll(ctx, func(name string) plugin.Dependencies {
		pluginCfg := cfg.Sub("plugins." + name)
		return plugin.Dependencies{
			Config:  pluginCfg,
			Logger:  logger.Named(name),
			Store:   db,
			Bus:     bus,
			Plugins: reg,
		}
	}); err != nil {
		logger.Fatal("failed to initialize plugins", zap.Error(err))
	}

	// Start plugins
	if err := reg.StartAll(ctx); err != nil {
		logger.Fatal("failed to start plugins", zap.Error(err))
	}

	// Create service mapping (svcmap) store, correlator, handler, and scheduler.
	svcmapStore, err := svcmap.NewStore(db.DB())
	if err != nil {
		logger.Fatal("failed to initialize svcmap store", zap.Error(err))
	}
	logger.Info("svcmap store initialized", zap.String("component", "svcmap"))

	dispatchProfileStore := dispatch.NewDispatchStore(db.DB())
	svcCorrelator := svcmap.NewCorrelator(svcmapStore, logger.Named("svcmap"))

	svcSourceAdapter := &serviceSourceAdapter{store: dispatchProfileStore}
	hwAdapter := &hwSourceAdapter{store: dispatchProfileStore}
	agentAdapter := &agentListerAdapter{store: dispatchProfileStore}

	svcmapHandler := svcmap.NewHandler(svcmapStore, hwAdapter, logger.Named("svcmap"))

	correlateInterval := viperCfg.GetDuration("svcmap.correlate_interval")
	if correlateInterval == 0 {
		correlateInterval = 60 * time.Second
	}
	svcmapScheduler := svcmap.NewScheduler(svcCorrelator, svcSourceAdapter, nil, agentAdapter, correlateInterval, logger.Named("svcmap"))
	svcmapScheduler.Start(ctx)

	// Create auth service
	authStore, err := auth.NewUserStore(ctx, db)
	if err != nil {
		logger.Fatal("failed to initialize auth store", zap.Error(err))
	}
	logger.Info("auth store initialized", zap.String("component", "auth"))

	jwtSecret := viperCfg.GetString("auth.jwt_secret")
	if jwtSecret == "" {
		// Generate an ephemeral secret -- tokens won't survive restarts.
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			logger.Fatal("failed to generate JWT secret", zap.Error(err))
		}
		jwtSecret = hex.EncodeToString(b)
		logger.Info("using auto-generated JWT secret (normal for first run; set auth.jwt_secret in config to persist sessions across restarts)",
			zap.String("component", "auth"),
		)
	} else {
		logger.Info("JWT secret loaded from configuration", zap.String("component", "auth"))
	}

	accessTTL := viperCfg.GetDuration("auth.access_token_ttl")
	if accessTTL == 0 {
		accessTTL = 15 * time.Minute
	}
	refreshTTL := viperCfg.GetDuration("auth.refresh_token_ttl")
	if refreshTTL == 0 {
		refreshTTL = 7 * 24 * time.Hour
	}

	tokens := auth.NewTokenService([]byte(jwtSecret), accessTTL, refreshTTL)
	authService := auth.NewService(authStore, tokens, logger.Named("auth"))
	authHandler := auth.NewHandler(authService, logger.Named("auth"))
	logger.Info("auth service initialized",
		zap.String("component", "auth"),
		zap.Duration("access_token_ttl", accessTTL),
		zap.Duration("refresh_token_ttl", refreshTTL),
	)

	// Create settings service
	settingsRepo, err := services.NewSQLiteSettingsRepository(ctx, db)
	if err != nil {
		logger.Fatal("failed to initialize settings repository", zap.Error(err))
	}
	settingsHandler := settings.NewHandler(settingsRepo, logger.Named("settings"))
	logger.Info("settings service initialized", zap.String("component", "settings"))

	// Create WebSocket handler for real-time scan updates
	wsHandler := ws.NewHandler(tokens, bus, logger.Named("ws"))
	logger.Info("websocket handler initialized", zap.String("component", "ws"))

	// Wire SNMP credential adapter: recon -> vault.
	var reconMod *recon.Module
	var vaultMod *vault.Module
	for _, m := range modules {
		switch mod := m.(type) {
		case *recon.Module:
			reconMod = mod
		case *vault.Module:
			vaultMod = mod
		}
	}
	if reconMod != nil && vaultMod != nil {
		reconMod.SetCredentialAccessor(recon.NewVaultCredentialAdapter(&vaultDecryptAdapter{vault: vaultMod}))
		reconMod.SetCredentialProvider(vaultMod)
		logger.Info("SNMP credential adapter wired", zap.String("component", "recon"))
	}

	// Create Gateway SSH WebSocket handler.
	// Find the gateway module in the registered plugins for SSH handler wiring.
	var gw *gateway.Module
	for _, m := range modules {
		if g, ok := m.(*gateway.Module); ok {
			gw = g
			break
		}
	}
	var sshHandler *gateway.SSHWebSocketHandler
	if gw != nil {
		sshHandler = gateway.NewSSHWebSocketHandler(gw, &tokenAdapter{tokens}, logger.Named("gateway-ssh"))
		logger.Info("gateway SSH handler initialized", zap.String("component", "gateway"))
	}

	// Create and start HTTP server
	addr := viperCfg.GetString("server.host") + ":" + viperCfg.GetString("server.port")
	if addr == ":" {
		addr = "0.0.0.0:8080"
	}
	logger.Info("HTTP server configured",
		zap.String("component", "server"),
		zap.String("addr", addr),
	)
	devMode := viperCfg.GetBool("server.dev_mode")
	readyCheck := server.ReadinessChecker(func(ctx context.Context) error {
		return db.DB().PingContext(ctx)
	})
	dashboardHandler := dashboard.Handler()
	extraRoutes := []server.SimpleRouteRegistrar{settingsHandler, wsHandler, svcmapHandler}
	if sshHandler != nil {
		extraRoutes = append(extraRoutes, sshHandler)
	}
	srv := server.New(addr, reg, logger, readyCheck, authHandler, dashboardHandler, devMode, extraRoutes...)

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	logger.Info("SubNetree server ready", zap.String("addr", addr))

	// Print human-readable banner for users watching docker logs.
	port := viperCfg.GetString("server.port")
	if port == "" {
		port = "8080"
	}
	fmt.Fprintf(os.Stderr, "\n  SubNetree %s is ready!\n  Open http://localhost:%s in your browser.\n\n", version.Short(), port)

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	svcmapScheduler.Stop()
	reg.StopAll(shutdownCtx)

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}

	logger.Info("SubNetree server stopped")
}

// vaultDecryptAdapter adapts vault.Module to the recon.CredentialDecrypter interface.
// Lives in the composition root to avoid coupling recon -> vault.
type vaultDecryptAdapter struct {
	vault *vault.Module
}

func (a *vaultDecryptAdapter) DecryptCredential(ctx context.Context, id string) (map[string]any, error) {
	return a.vault.DecryptCredentialData(ctx, id)
}

// tokenAdapter adapts auth.TokenService to the gateway.TokenValidator interface.
// Lives in the composition root to avoid coupling gateway -> auth.
type tokenAdapter struct {
	svc *auth.TokenService
}

func (a *tokenAdapter) ValidateAccessToken(token string) (*gateway.TokenClaims, error) {
	claims, err := a.svc.ValidateAccessToken(token)
	if err != nil {
		return nil, err
	}
	return &gateway.TokenClaims{UserID: claims.UserID}, nil
}

// serviceSourceAdapter adapts dispatch.DispatchStore to svcmap.ServiceSource.
type serviceSourceAdapter struct {
	store *dispatch.DispatchStore
}

func (a *serviceSourceAdapter) GetServices(ctx context.Context, agentID string) ([]svcmap.ScoutService, error) {
	protos, err := a.store.GetServices(ctx, agentID)
	if err != nil {
		return nil, err
	}
	result := make([]svcmap.ScoutService, len(protos))
	for i := range protos {
		ports := make([]string, len(protos[i].Ports))
		for j, p := range protos[i].Ports {
			ports[j] = fmt.Sprintf("%d", p)
		}
		result[i] = svcmap.ScoutService{
			Name:        protos[i].Name,
			DisplayName: protos[i].DisplayName,
			Status:      protos[i].Status,
			StartType:   protos[i].StartType,
			CPUPercent:  protos[i].CpuPercent,
			MemoryBytes: protos[i].MemoryBytes,
			Ports:       ports,
		}
	}
	return result, nil
}

// hwSourceAdapter adapts dispatch.DispatchStore to svcmap.HardwareSource.
type hwSourceAdapter struct {
	store *dispatch.DispatchStore
}

func (a *hwSourceAdapter) GetHardwareProfile(ctx context.Context, agentID string) (*svcmap.HardwareInfo, error) {
	hw, err := a.store.GetHardwareProfile(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if hw == nil {
		return nil, nil
	}
	var totalDisk int64
	for i := range hw.Disks {
		totalDisk += hw.Disks[i].SizeBytes
	}
	return &svcmap.HardwareInfo{
		TotalMemoryBytes: hw.RamBytes,
		TotalDiskBytes:   totalDisk,
		CPUCores:         int(hw.CpuCores),
	}, nil
}

// agentListerAdapter adapts dispatch.DispatchStore to svcmap.AgentLister.
type agentListerAdapter struct {
	store *dispatch.DispatchStore
}

func (a *agentListerAdapter) ListAgentsWithDevice(ctx context.Context) ([]svcmap.AgentRef, error) {
	agents, err := a.store.ListAgents(ctx)
	if err != nil {
		return nil, err
	}
	refs := make([]svcmap.AgentRef, len(agents))
	for i := range agents {
		refs[i] = svcmap.AgentRef{
			AgentID:  agents[i].ID,
			DeviceID: agents[i].DeviceID,
			Hostname: agents[i].Hostname,
		}
	}
	return refs, nil
}
