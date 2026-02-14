//go:build !windows

package profiler

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

func collectServices(ctx context.Context, logger *zap.Logger) ([]*scoutpb.ServiceInfo, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "systemctl", "list-units",
		"--type=service", "--no-pager", "--no-legend", "--plain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		logger.Debug("systemctl not available", zap.Error(err))
		return nil, nil
	}

	var services []*scoutpb.ServiceInfo
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Each line: UNIT LOAD ACTIVE SUB DESCRIPTION...
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		unitName := fields[0]
		sub := fields[3]

		// Build display name from the description (fields[4:]).
		displayName := unitName
		if len(fields) > 4 {
			displayName = strings.Join(fields[4:], " ")
		}

		si := &scoutpb.ServiceInfo{
			Name:        unitName,
			DisplayName: displayName,
			Status:      mapSubState(sub),
		}

		// Get start type (best-effort, skip on failure to avoid slowing down).
		si.StartType = getServiceStartType(cmdCtx, unitName, logger)

		services = append(services, si)
	}

	logger.Debug("collected services", zap.Int("count", len(services)))
	return services, nil
}

// mapSubState maps systemd SUB states to our simplified status values.
func mapSubState(sub string) string {
	switch sub {
	case "running":
		return "running"
	case "dead", "inactive":
		return "stopped"
	case "failed":
		return "failed"
	case "exited":
		return "stopped"
	case "waiting", "start-pre", "start", "start-post":
		return "start_pending"
	case "stop-pre", "stop", "stop-post":
		return "stop_pending"
	default:
		return "unknown"
	}
}

// getServiceStartType runs systemctl is-enabled to determine the start type.
func getServiceStartType(ctx context.Context, unit string, logger *zap.Logger) string {
	cmd := exec.CommandContext(ctx, "systemctl", "is-enabled", unit)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// is-enabled returns non-zero for disabled/masked/static, so we don't check error.
	_ = cmd.Run()

	result := strings.TrimSpace(stdout.String())
	switch result {
	case "enabled":
		return "auto"
	case "disabled":
		return "disabled"
	case "masked":
		return "disabled"
	case "static":
		return "manual"
	case "indirect":
		return "manual"
	default:
		return "unknown"
	}
}
