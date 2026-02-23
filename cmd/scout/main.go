package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/HerbHall/subnetree/internal/scout"
	"github.com/HerbHall/subnetree/internal/scout/service"
	"github.com/HerbHall/subnetree/internal/version"
	"go.uber.org/zap"
)

func main() {
	if len(os.Args) < 2 {
		// No subcommand: default to "run" for backward compatibility.
		runCmd(os.Args[1:])
		return
	}

	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "install":
		installCmd(os.Args[2:])
	case "uninstall":
		uninstallCmd()
	case "version":
		versionCmd()
	default:
		// Backward compatibility: if first arg starts with "-", treat as "run" flags.
		if len(os.Args[1]) > 0 && os.Args[1][0] == '-' {
			runCmd(os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "unknown command: %s\nUsage: scout [run|install|uninstall|version] [flags]\n", os.Args[1])
		os.Exit(1)
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	serverAddr := fs.String("server", "localhost:9090", "SubNetree server gRPC address")
	interval := fs.Int("interval", 30, "Check-in interval in seconds")
	enrollToken := fs.String("enroll-token", "", "Enrollment token for initial registration")
	agentID := fs.String("agent-id", "", "Agent ID (auto-assigned during enrollment if empty)")
	certPath := fs.String("cert", "", "Path to agent TLS certificate")
	keyPath := fs.String("key", "", "Path to agent TLS private key")
	caCert := fs.String("ca-cert", "", "Path to CA certificate for TLS verification")
	insecureFlag := fs.Bool("insecure", false, "Use insecure gRPC transport (dev/testing only)")
	autoRestart := fs.Bool("auto-restart", false, "Enable auto-restart on version rejection (requires init system support)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

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
		AutoRestart:   *autoRestart,
	}

	// Check if running as a Windows service.
	isSvc, svcErr := service.IsService()
	if svcErr != nil {
		logger.Warn("failed to detect service mode", zap.Error(svcErr))
	}
	if isSvc {
		logger.Info("running as Windows service")
		if err := service.RunAsService(config, logger); err != nil {
			logger.Fatal("service error", zap.Error(err))
		}
		return
	}

	// Interactive mode (existing behavior).
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

func installCmd(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	serverAddr := fs.String("server", "localhost:9090", "SubNetree server gRPC address")
	interval := fs.Int("interval", 30, "Check-in interval in seconds")
	certPath := fs.String("cert", "", "Path to agent TLS certificate")
	keyPath := fs.String("key", "", "Path to agent TLS private key")
	caCert := fs.String("ca-cert", "", "Path to CA certificate")
	insecureFlag := fs.Bool("insecure", false, "Use insecure gRPC transport")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve executable path: %v\n", err)
		os.Exit(1)
	}

	config := &scout.Config{
		ServerAddr:    *serverAddr,
		CheckInterval: *interval,
		CertPath:      *certPath,
		KeyPath:       *keyPath,
		CACertPath:    *caCert,
		Insecure:      *insecureFlag,
	}

	if err := service.InstallService(exePath, config); err != nil {
		fmt.Fprintf(os.Stderr, "install failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("SubNetree Scout service installed successfully")
	fmt.Println("Start with: sc start SubNetreeScout")
}

func uninstallCmd() {
	if err := service.UninstallService(); err != nil {
		fmt.Fprintf(os.Stderr, "uninstall failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("SubNetree Scout service removed successfully")
}

func versionCmd() {
	fmt.Printf("SubNetree Scout %s\n", version.Version)
	fmt.Printf("  Commit: %s\n", version.GitCommit)
	fmt.Printf("  Built:  %s\n", version.BuildDate)
}
