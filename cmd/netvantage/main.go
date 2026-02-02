package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("NetVantage server starting", zap.String("version", version.Short()))

	// Load configuration
	viperCfg, err := server.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}
	cfg := config.New(viperCfg)

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

	// Create shared services
	bus := event.NewBus(logger.Named("event"))

	// Create plugin registry
	reg := registry.New(logger.Named("registry"))

	// Register all plugins (compile-time composition)
	modules := []plugin.Plugin{
		recon.New(),
		pulse.New(),
		dispatch.New(),
		vault.New(),
		gateway.New(),
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

	// Create and start HTTP server
	addr := viperCfg.GetString("server.host") + ":" + viperCfg.GetString("server.port")
	if addr == ":" {
		addr = "0.0.0.0:8080"
	}
	readyCheck := server.ReadinessChecker(func(ctx context.Context) error {
		return db.DB().PingContext(ctx)
	})
	srv := server.New(addr, reg, logger, readyCheck)

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
