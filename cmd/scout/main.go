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
	certPath := flag.String("cert", "", "Path to agent TLS certificate")
	keyPath := flag.String("key", "", "Path to agent TLS private key")
	caCert := flag.String("ca-cert", "", "Path to CA certificate for TLS verification")
	insecureFlag := flag.Bool("insecure", false, "Use insecure gRPC transport (dev/testing only)")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	// Default to insecure when no TLS paths are configured.
	useInsecure := *insecureFlag
	if *certPath == "" && *keyPath == "" && *caCert == "" && !*insecureFlag {
		useInsecure = true
	}

	config := &scout.Config{
		ServerAddr:    *serverAddr,
		CheckInterval: *interval,
		AgentID:       *agentID,
		EnrollToken:   *enrollToken,
		CertPath:      *certPath,
		KeyPath:       *keyPath,
		CACertPath:    *caCert,
		Insecure:      useInsecure,
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
