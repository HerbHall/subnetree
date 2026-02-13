package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/HerbHall/subnetree/internal/scout"
	"go.uber.org/zap"
)

func main() {
	serverAddr := flag.String("server", "localhost:9090", "SubNetree server gRPC address")
	interval := flag.Int("interval", 30, "Check-in interval in seconds")
	enrollToken := flag.String("enroll-token", "", "Enrollment token for initial registration")
	agentID := flag.String("agent-id", "", "Agent ID (auto-assigned during enrollment if empty)")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	config := &scout.Config{
		ServerAddr:    *serverAddr,
		CheckInterval: *interval,
		AgentID:       *agentID,
		EnrollToken:   *enrollToken,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	agent := scout.NewAgent(config, logger)
	if err := agent.Run(ctx); err != nil {
		logger.Fatal("agent error", zap.Error(err))
	}

	logger.Info("scout agent stopped")
}
