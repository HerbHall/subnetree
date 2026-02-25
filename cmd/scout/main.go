package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/HerbHall/subnetree/internal/scout"
	"github.com/HerbHall/subnetree/internal/scout/service"
	"github.com/HerbHall/subnetree/internal/scout/updater"
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
	case "update":
		updateCmd(os.Args[2:])
	case "version":
		versionCmd()
	default:
		// Backward compatibility: if first arg starts with "-", treat as "run" flags.
		if os.Args[1] != "" && os.Args[1][0] == '-' {
			runCmd(os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "unknown command: %s\nUsage: scout [run|install|uninstall|update|version] [flags]\n", os.Args[1])
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

func updateCmd(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	serverAddr := fs.String("server", "http://localhost:8080", "SubNetree server HTTP address")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create logger")
		os.Exit(1)
	}

	if err := runUpdate(logger, *serverAddr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		_ = logger.Sync()
		os.Exit(1)
	}
	_ = logger.Sync()
}

func runUpdate(logger *zap.Logger, serverAddr string) error {
	ctx := context.Background()

	// Fetch update manifest from server.
	manifestURL := serverAddr + "/api/v1/dispatch/updates/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("manifest request failed (%d): %s", resp.StatusCode, body)
	}

	var manifest struct {
		Version   string `json:"version"`
		Platforms map[string]struct {
			URL string `json:"url"`
		} `json:"platforms"`
		ChecksumsURL string `json:"checksums_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}

	if manifest.Version == version.Version || manifest.Version == "dev" {
		fmt.Printf("Already up to date (%s)\n", version.Version)
		return nil
	}

	platform := runtime.GOOS + "/" + runtime.GOARCH
	info, ok := manifest.Platforms[platform]
	if !ok {
		return fmt.Errorf("no update available for platform %s", platform)
	}

	u, err := updater.New(logger)
	if err != nil {
		return fmt.Errorf("init updater: %w", err)
	}

	fmt.Printf("Updating from %s to %s...\n", version.Version, manifest.Version)
	if err := u.Apply(ctx, info.URL, manifest.ChecksumsURL); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}
	fmt.Println("Update applied successfully. Restart Scout to use the new version.")
	return nil
}

func versionCmd() {
	fmt.Printf("SubNetree Scout %s\n", version.Version)
	fmt.Printf("  Commit: %s\n", version.GitCommit)
	fmt.Printf("  Built:  %s\n", version.BuildDate)
}
