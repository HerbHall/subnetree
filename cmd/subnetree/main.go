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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/HerbHall/subnetree/api/swagger"
	"github.com/HerbHall/subnetree/internal/auth"
	"github.com/HerbHall/subnetree/internal/autodoc"
	mcpmod "github.com/HerbHall/subnetree/internal/mcp"
	"github.com/HerbHall/subnetree/internal/catalog"
	"github.com/HerbHall/subnetree/internal/config"
	"github.com/HerbHall/subnetree/internal/dashboard"
	"github.com/HerbHall/subnetree/internal/dispatch"
	"github.com/HerbHall/subnetree/internal/docs"
	"github.com/HerbHall/subnetree/internal/event"
	"github.com/HerbHall/subnetree/internal/gateway"
	"github.com/HerbHall/subnetree/internal/insight"
	"github.com/HerbHall/subnetree/internal/llm"
	"github.com/HerbHall/subnetree/internal/mqtt"
	"github.com/HerbHall/subnetree/internal/pulse"
	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/internal/seed"
	"github.com/HerbHall/subnetree/internal/registry"
	"github.com/HerbHall/subnetree/internal/server"
	"github.com/HerbHall/subnetree/internal/services"
	"github.com/HerbHall/subnetree/internal/settings"
	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/internal/svcmap"
	tsmod "github.com/HerbHall/subnetree/internal/tailscale"
	"github.com/HerbHall/subnetree/internal/tier"
	"github.com/HerbHall/subnetree/internal/vault"
	"github.com/HerbHall/subnetree/internal/version"
	"github.com/HerbHall/subnetree/internal/webhook"
	"github.com/HerbHall/subnetree/internal/ws"
	pkgcatalog "github.com/HerbHall/subnetree/pkg/catalog"
	"github.com/HerbHall/subnetree/pkg/models"
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
		case "mcp":
			runMCPStdio()
			return
		}
	}

	configPath := flag.String("config", "", "path to configuration file")
	showVersion := flag.Bool("version", false, "print version information and exit")
	seedData := flag.Bool("seed", false, "populate database with demo network data")
	demoMode := flag.Bool("demo", false, "enable demo mode (read-only, no auth required)")
	flag.Parse()

	// Demo mode forces seed data on and can be set via environment variable.
	isDemoMode := *demoMode || os.Getenv("NV_DEMO_MODE") == "true"
	if isDemoMode {
		*seedData = true
	}

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

	// Detect hardware tier and apply tier-specific defaults.
	detectedTier := tier.DetectTier()
	tier.ApplyDefaults(viperCfg, detectedTier)

	// Initialize logger from configuration.
	logger, err := config.NewLogger(viperCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("SubNetree server starting", zap.String("version", version.Short()))

	logger.Info("hardware tier detected",
		zap.Int("tier", int(detectedTier)),
		zap.String("tier_name", tier.Name(detectedTier)),
	)

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

	// Check database schema version compatibility.
	if err := db.CheckVersion(context.Background(), version.Short()); err != nil {
		logger.Fatal("database version check failed",
			zap.Error(err),
			zap.String("binary_version", version.Short()),
		)
	}

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
		mqtt.New(),
		autodoc.New(),
		mcpmod.New(),
		tsmod.New(),
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
	totpSvc := auth.NewTOTPService([]byte(jwtSecret))
	authService := auth.NewService(authStore, tokens, totpSvc, logger.Named("auth"))
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
	var pulseMod *pulse.Module
	for _, m := range modules {
		switch mod := m.(type) {
		case *recon.Module:
			reconMod = mod
		case *vault.Module:
			vaultMod = mod
		case *pulse.Module:
			pulseMod = mod
		}
	}
	if reconMod != nil && vaultMod != nil {
		reconMod.SetCredentialAccessor(recon.NewVaultCredentialAdapter(&vaultDecryptAdapter{vault: vaultMod}))
		reconMod.SetCredentialProvider(vaultMod)
		logger.Info("SNMP credential adapter wired", zap.String("component", "recon"))
	}

	// Wire hardware profile bridge: dispatch -> recon.
	if reconMod != nil {
		profileAdapter := &profileSourceAdapter{store: dispatchProfileStore}
		reconMod.SetProfileSource(profileAdapter)
		logger.Info("hardware profile bridge wired", zap.String("component", "recon"))
	}

	// Wire MCP device querier and service querier: mcp -> recon store, svcmap store.
	if reconMod != nil {
		for _, m := range modules {
			if mcpMod, ok := m.(*mcpmod.Module); ok {
				mcpMod.SetQuerier(&mcpDeviceAdapter{store: reconMod.Store()})
				mcpMod.SetServiceQuerier(&mcpServiceAdapter{store: svcmapStore})
				logger.Info("MCP device and service queriers wired", zap.String("component", "mcp"))
				break
			}
		}
	}

	// Wire Tailscale adapters: tailscale -> recon store, vault.
	if reconMod != nil && vaultMod != nil {
		for _, m := range modules {
			if ts, ok := m.(*tsmod.Module); ok {
				ts.SetDeviceStore(&tailscaleDeviceAdapter{store: reconMod.Store()})
				ts.SetCredentialDecrypter(&vaultDecryptAdapter{vault: vaultMod})
				logger.Info("tailscale adapters wired", zap.String("component", "tailscale"))
				break
			}
		}
	}

	// Wire AutoDoc device and alert readers: autodoc -> recon store, pulse store.
	if reconMod != nil {
		for _, m := range modules {
			if adMod, ok := m.(*autodoc.Module); ok {
				adMod.SetDeviceReader(&autodocDeviceAdapter{store: reconMod.Store()})
				if pulseMod != nil && pulseMod.Store() != nil {
					adMod.SetAlertReader(&autodocAlertAdapter{store: pulseMod.Store()})
				}
				logger.Info("autodoc device and alert readers wired", zap.String("component", "autodoc"))
				break
			}
		}
	}

	// Seed demo data if requested via --seed flag or NV_SEED_DATA env var.
	if *seedData || os.Getenv("NV_SEED_DATA") == "true" {
		if reconMod != nil {
			if seedErr := seed.SeedDemoNetwork(context.Background(), reconMod.Store()); seedErr != nil {
				logger.Error("failed to seed demo data", zap.Error(seedErr))
			} else {
				logger.Info("demo network seeded successfully")
			}
		} else {
			logger.Warn("cannot seed demo data: recon module not available")
		}

		// Seed Proxmox VM/container data under the proxmox-host device.
		if reconMod != nil {
			if seedErr := seed.SeedProxmoxData(context.Background(), reconMod.Store(), db.DB()); seedErr != nil {
				logger.Error("failed to seed proxmox data", zap.Error(seedErr))
			} else {
				logger.Info("proxmox VM/container data seeded successfully")
			}
		}

		// Seed Pulse monitoring data (checks, results, alerts) for demo.
		if pulseMod != nil && pulseMod.Store() != nil {
			if seedErr := seed.SeedPulseData(context.Background(), pulseMod.Store(), db.DB()); seedErr != nil {
				logger.Error("failed to seed pulse data", zap.Error(seedErr))
			} else {
				logger.Info("pulse monitoring data seeded successfully")
			}
		}
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

	// Create catalog recommendation handler.
	cat := pkgcatalog.NewCatalog()
	catalogEngine := catalog.NewEngine(cat)
	catalogHandler := catalog.NewHandler(catalogEngine, logger.Named("catalog"))

	extraRoutes := []server.SimpleRouteRegistrar{settingsHandler, wsHandler, svcmapHandler, catalogHandler}
	if sshHandler != nil {
		extraRoutes = append(extraRoutes, sshHandler)
	}
	// In demo mode, use DemoAuthMiddleware instead of JWT validation.
	var authRegistrar server.RouteRegistrar
	if isDemoMode {
		authRegistrar = &demoAuthRegistrar{}
		logger.Warn("demo mode enabled: authentication bypassed, read-only access enforced")
	} else {
		authRegistrar = authHandler
	}

	srv := server.New(addr, reg, logger, readyCheck, authRegistrar, dashboardHandler, devMode, isDemoMode, extraRoutes...)

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

// profileSourceAdapter adapts dispatch.DispatchStore to recon.ProfileSource.
type profileSourceAdapter struct {
	store *dispatch.DispatchStore
}

func (a *profileSourceAdapter) GetAgent(ctx context.Context, agentID string) (*recon.AgentInfo, error) {
	agent, err := a.store.GetAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, nil
	}
	return &recon.AgentInfo{
		ID:       agent.ID,
		DeviceID: agent.DeviceID,
		Platform: agent.Platform,
		Hostname: agent.Hostname,
	}, nil
}

func (a *profileSourceAdapter) GetHardwareProfile(ctx context.Context, agentID string) (*recon.HardwareProfileData, error) {
	hw, err := a.store.GetHardwareProfile(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if hw == nil {
		return nil, nil
	}
	result := &recon.HardwareProfileData{
		CPUModel:           hw.CpuModel,
		CPUCores:           hw.CpuCores,
		CPUThreads:         hw.CpuThreads,
		RAMBytes:           hw.RamBytes,
		BIOSVersion:        hw.BiosVersion,
		SystemManufacturer: hw.SystemManufacturer,
		SystemModel:        hw.SystemModel,
		SerialNumber:       hw.SerialNumber,
	}
	for _, d := range hw.Disks {
		result.Disks = append(result.Disks, recon.DiskData{
			Name:      d.Name,
			SizeBytes: d.SizeBytes,
			DiskType:  d.DiskType,
			Model:     d.Model,
		})
	}
	for _, g := range hw.Gpus {
		result.GPUs = append(result.GPUs, recon.GPUData{
			Model:         g.Model,
			VRAMBytes:     g.VramBytes,
			DriverVersion: g.DriverVersion,
		})
	}
	return result, nil
}

func (a *profileSourceAdapter) GetServices(ctx context.Context, agentID string) ([]*recon.ServiceData, error) {
	protos, err := a.store.GetServices(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if protos == nil {
		return nil, nil
	}
	result := make([]*recon.ServiceData, len(protos))
	for i, p := range protos {
		result[i] = &recon.ServiceData{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Status:      p.Status,
			StartType:   p.StartType,
			CPUPercent:  p.CpuPercent,
			MemoryBytes: p.MemoryBytes,
			Ports:       p.Ports,
		}
	}
	return result, nil
}

// mcpDeviceAdapter adapts recon.ReconStore to mcp.DeviceQuerier.
// Lives in the composition root to avoid coupling mcp -> recon.
type mcpDeviceAdapter struct {
	store *recon.ReconStore
}

func (a *mcpDeviceAdapter) GetDevice(ctx context.Context, id string) (*models.Device, error) {
	return a.store.GetDevice(ctx, id)
}

func (a *mcpDeviceAdapter) ListDevices(ctx context.Context, limit, offset int) ([]models.Device, int, error) {
	return a.store.ListDevices(ctx, recon.ListDevicesOptions{
		Limit:  limit,
		Offset: offset,
	})
}

func (a *mcpDeviceAdapter) GetDeviceHardware(ctx context.Context, deviceID string) (*models.DeviceHardware, error) {
	return a.store.GetDeviceHardware(ctx, deviceID)
}

func (a *mcpDeviceAdapter) GetHardwareSummary(ctx context.Context) (*models.HardwareSummary, error) {
	return a.store.GetHardwareSummary(ctx)
}

func (a *mcpDeviceAdapter) QueryDevicesByHardware(ctx context.Context, query models.HardwareQuery) ([]models.Device, int, error) {
	return a.store.QueryDevicesByHardware(ctx, query)
}

func (a *mcpDeviceAdapter) FindStaleDevices(ctx context.Context, threshold time.Time) ([]models.Device, error) {
	return a.store.FindStaleDevices(ctx, threshold)
}

// mcpServiceAdapter adapts svcmap.Store to mcp.ServiceQuerier.
// Lives in the composition root to avoid coupling mcp -> svcmap.
type mcpServiceAdapter struct {
	store *svcmap.Store
}

func (a *mcpServiceAdapter) ListServicesFiltered(ctx context.Context, deviceID, serviceType, status string) ([]models.Service, error) {
	return a.store.ListServicesFiltered(ctx, svcmap.ServiceFilter{
		DeviceID:    deviceID,
		ServiceType: serviceType,
		Status:      status,
	})
}

// tailscaleDeviceAdapter adapts recon.ReconStore to tailscale.DeviceStore.
type tailscaleDeviceAdapter struct {
	store *recon.ReconStore
}

func (a *tailscaleDeviceAdapter) UpsertDevice(ctx context.Context, d *models.Device) (bool, error) {
	return a.store.UpsertDevice(ctx, d)
}

func (a *tailscaleDeviceAdapter) ListDevices(ctx context.Context, limit, offset int) ([]models.Device, int, error) {
	return a.store.ListDevices(ctx, recon.ListDevicesOptions{Limit: limit, Offset: offset})
}

func (a *tailscaleDeviceAdapter) GetDeviceByMAC(ctx context.Context, mac string) (*models.Device, error) {
	return a.store.GetDeviceByMAC(ctx, mac)
}

// demoAuthRegistrar implements server.RouteRegistrar for demo mode.
// It registers no routes (login/setup not needed) and provides the
// DemoAuthMiddleware that injects synthetic viewer claims on every API request.
type demoAuthRegistrar struct{}

func (d *demoAuthRegistrar) RegisterRoutes(_ *http.ServeMux) {
	// No auth routes needed in demo mode.
}

func (d *demoAuthRegistrar) Middleware() func(http.Handler) http.Handler {
	return auth.DemoAuthMiddleware()
}

// autodocDeviceAdapter adapts recon.ReconStore to autodoc.DeviceReader.
// Lives in the composition root to avoid coupling autodoc -> recon.
type autodocDeviceAdapter struct {
	store *recon.ReconStore
}

func (a *autodocDeviceAdapter) GetDevice(ctx context.Context, id string) (*models.Device, error) {
	return a.store.GetDevice(ctx, id)
}

func (a *autodocDeviceAdapter) ListAllDevices(ctx context.Context) ([]models.Device, error) {
	return a.store.ListAllDevices(ctx)
}

func (a *autodocDeviceAdapter) GetDeviceHardware(ctx context.Context, deviceID string) (*models.DeviceHardware, error) {
	return a.store.GetDeviceHardware(ctx, deviceID)
}

func (a *autodocDeviceAdapter) GetDeviceStorage(ctx context.Context, deviceID string) ([]models.DeviceStorage, error) {
	return a.store.GetDeviceStorage(ctx, deviceID)
}

func (a *autodocDeviceAdapter) GetDeviceGPU(ctx context.Context, deviceID string) ([]models.DeviceGPU, error) {
	return a.store.GetDeviceGPU(ctx, deviceID)
}

func (a *autodocDeviceAdapter) GetDeviceServices(ctx context.Context, deviceID string) ([]models.DeviceService, error) {
	return a.store.GetDeviceServices(ctx, deviceID)
}

func (a *autodocDeviceAdapter) GetChildDevices(ctx context.Context, parentID string) ([]models.Device, error) {
	all, err := a.store.ListAllDevices(ctx)
	if err != nil {
		return nil, err
	}
	children := make([]models.Device, 0)
	for i := range all {
		if all[i].ParentDeviceID == parentID {
			children = append(children, all[i])
		}
	}
	return children, nil
}

// autodocAlertAdapter adapts pulse.PulseStore to autodoc.AlertReader.
// Lives in the composition root to avoid coupling autodoc -> pulse.
type autodocAlertAdapter struct {
	store *pulse.PulseStore
}

func (a *autodocAlertAdapter) ListDeviceAlerts(ctx context.Context, deviceID string, limit int) ([]autodoc.DeviceAlert, error) {
	alerts, err := a.store.ListAlerts(ctx, pulse.AlertFilters{
		DeviceID: deviceID,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}
	result := make([]autodoc.DeviceAlert, len(alerts))
	for i := range alerts {
		result[i] = autodoc.DeviceAlert{
			Severity:    alerts[i].Severity,
			Message:     alerts[i].Message,
			TriggeredAt: alerts[i].TriggeredAt,
			ResolvedAt:  alerts[i].ResolvedAt,
		}
	}
	return result, nil
}
