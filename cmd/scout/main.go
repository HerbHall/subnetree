package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/HerbHall/netvantage/internal/scout"
	"go.uber.org/zap"
)

func main() {
	serverAddr := flag.String("server", "localhost:9090", "NetVantage server address")
	interval := flag.Int("interval", 30, "check-in interval in seconds")
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	config := &scout.Config{
		ServerAddr:    *serverAddr,
		CheckInterval: *interval,
	}

	agent := scout.NewAgent(config, logger)

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	if err := agent.Run(ctx); err != nil {
		logger.Fatal("agent error", zap.Error(err))
	}

	logger.Info("scout agent stopped")
}
