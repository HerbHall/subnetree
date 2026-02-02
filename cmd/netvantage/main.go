package main

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

	"github.com/HerbHall/netvantage/internal/auth"
	"github.com/HerbHall/netvantage/internal/config"
	"github.com/HerbHall/netvantage/internal/dispatch"
	"github.com/HerbHall/netvantage/internal/event"
	"github.com/HerbHall/netvantage/internal/gateway"
	"github.com/HerbHall/netvantage/internal/pulse"
	"github.com/HerbHall/netvantage/internal/recon"
	"github.com/HerbHall/netvantage/internal/registry"
	"github.com/HerbHall/netvantage/internal/server"
	"github.com/HerbHall/netvantage/internal/store"
	"github.com/HerbHall/netvantage/internal/vault"
	"github.com/HerbHall/netvantage/internal/webhook"
	"github.com/HerbHall/netvantage/internal/version"
	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

func main() {
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

	logger.Info("NetVantage server starting", zap.String("version", version.Short()))

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
		dbPath = "netvantage.db"
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
		logger.Warn("no auth.jwt_secret configured; using ephemeral secret (tokens will not survive restarts)",
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

	// Create and start HTTP server
	addr := viperCfg.GetString("server.host") + ":" + viperCfg.GetString("server.port")
	if addr == ":" {
		addr = "0.0.0.0:8080"
	}
	logger.Info("HTTP server configured",
		zap.String("component", "server"),
		zap.String("addr", addr),
	)
	readyCheck := server.ReadinessChecker(func(ctx context.Context) error {
		return db.DB().PingContext(ctx)
	})
	srv := server.New(addr, reg, logger, readyCheck, authHandler)

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	logger.Info("NetVantage server ready", zap.String("addr", addr))

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	logger.Info("received shutdown signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	reg.StopAll(shutdownCtx)

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}

	logger.Info("NetVantage server stopped")
}
