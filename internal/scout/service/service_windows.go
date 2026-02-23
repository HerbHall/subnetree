//go:build windows

// Package service provides Windows service lifecycle management for the Scout agent.
package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/HerbHall/subnetree/internal/scout"
	"go.uber.org/zap"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	// ServiceName is the Windows service name used for SCM registration.
	ServiceName = "SubNetreeScout"
	// ServiceDisplayName is the human-readable name shown in services.msc.
	ServiceDisplayName = "SubNetree Scout Agent"
	// ServiceDescription is the description shown in service properties.
	ServiceDescription = "SubNetree network monitoring agent"
)

// scoutService implements svc.Handler for the Windows Service Control Manager.
type scoutService struct {
	config *scout.Config
	logger *zap.Logger
}

// Execute implements svc.Handler. It manages the service lifecycle:
// StartPending -> Running -> (handles Stop/Shutdown) -> StopPending -> exit.
func (s *scoutService) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agent := scout.NewAgent(s.config, s.logger)

	// Run agent in a goroutine so we can respond to SCM commands.
	agentDone := make(chan error, 1)
	go func() {
		agentDone <- agent.Run(ctx)
	}()

	changes <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for {
		select {
		case err := <-agentDone:
			if err != nil {
				s.logger.Error("agent exited with error", zap.Error(err))
				return false, 1
			}
			return false, 0

		case cr := <-r:
			switch cr.Cmd {
			case svc.Stop, svc.Shutdown:
				s.logger.Info("service stop requested")
				changes <- svc.Status{State: svc.StopPending}
				cancel()

				// Wait for agent to finish with a timeout.
				select {
				case <-agentDone:
				case <-time.After(30 * time.Second):
					s.logger.Warn("agent did not stop within timeout")
				}
				return false, 0

			case svc.Interrogate:
				changes <- cr.CurrentStatus
			}
		}
	}
}

// RunAsService starts the Scout agent as a Windows service.
// This should only be called when the process is launched by the SCM.
func RunAsService(config *scout.Config, logger *zap.Logger) error {
	return svc.Run(ServiceName, &scoutService{
		config: config,
		logger: logger,
	})
}

// InstallService registers the Scout agent as a Windows service.
// exePath is the absolute path to the scout executable.
func InstallService(exePath string, config *scout.Config) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()

	// Build service arguments: "run" subcommand with config flags.
	args := buildServiceArgs(config)

	svcConfig := mgr.Config{
		DisplayName:  ServiceDisplayName,
		Description:  ServiceDescription,
		ServiceType:  windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
	}

	s, err := m.CreateService(ServiceName, exePath, svcConfig, args...)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer s.Close()

	// Configure recovery actions: restart with increasing delays.
	recoveryActions := []mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}
	if err := s.SetRecoveryActions(recoveryActions, 86400); err != nil {
		// Non-fatal: service works without recovery actions.
		fmt.Fprintf(os.Stderr, "warning: failed to set recovery actions: %v\n", err)
	}

	// Enable recovery on non-crash failures (clean exit with non-zero code).
	if err := s.SetRecoveryActionsOnNonCrashFailures(true); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to set non-crash recovery: %v\n", err)
	}

	// Install eventlog source for structured Windows event logging.
	if err := eventlog.InstallAsEventCreate(ServiceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to install eventlog source: %v\n", err)
	}

	return nil
}

// UninstallService removes the Scout Windows service registration.
func UninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("open service: %w", err)
	}
	defer s.Close()

	// If the service is running, stop it first.
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("query service status: %w", err)
	}

	if status.State != svc.Stopped {
		if _, err := s.Control(svc.Stop); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to stop service: %v\n", err)
		}

		// Wait for the service to stop.
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			status, err = s.Query()
			if err != nil {
				break
			}
			if status.State == svc.Stopped {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}

	// Remove eventlog source.
	if err := eventlog.Remove(ServiceName); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to remove eventlog source: %v\n", err)
	}

	return nil
}

// IsService reports whether the current process is running as a Windows service.
func IsService() (bool, error) {
	return svc.IsWindowsService()
}

// buildServiceArgs constructs the command-line arguments for the service binary.
// These are passed after the executable path when the SCM starts the service.
func buildServiceArgs(config *scout.Config) []string {
	args := []string{"run"}

	if config.ServerAddr != "" {
		args = append(args, "--server", config.ServerAddr)
	}
	if config.CheckInterval > 0 {
		args = append(args, "--interval", fmt.Sprintf("%d", config.CheckInterval))
	}
	if config.CertPath != "" {
		args = append(args, "--cert", config.CertPath)
	}
	if config.KeyPath != "" {
		args = append(args, "--key", config.KeyPath)
	}
	if config.CACertPath != "" {
		args = append(args, "--ca-cert", config.CACertPath)
	}
	if config.Insecure {
		args = append(args, "--insecure")
	}

	return args
}
