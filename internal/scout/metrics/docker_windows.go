//go:build windows

package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

// dockerCLIStats represents a single line from `docker stats --no-stream --format json`.
type dockerCLIStats struct {
	Container string `json:"Container"`
	Name      string `json:"Name"`
	CPUPerc   string `json:"CPUPerc"`
	MemUsage  string `json:"MemUsage"`
	MemPerc   string `json:"MemPerc"`
	NetIO     string `json:"NetIO"`
	BlockIO   string `json:"BlockIO"`
	PIDs      string `json:"PIDs"`
}

// collectDockerStats collects per-container stats via Docker CLI on Windows.
// Returns nil if Docker is not available (graceful degradation).
func collectDockerStats(ctx context.Context, logger *zap.Logger) []*scoutpb.DockerContainerStats {
	// Check if docker CLI is available.
	if _, err := exec.LookPath("docker"); err != nil {
		return nil
	}

	// Use docker stats CLI with JSON format.
	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", "{{json .}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		logger.Debug("docker stats command failed", zap.Error(err))
		return nil
	}

	var results []*scoutpb.DockerContainerStats
	decoder := json.NewDecoder(&stdout)
	for decoder.More() {
		var s dockerCLIStats
		if err := decoder.Decode(&s); err != nil {
			break
		}
		results = append(results, parseCLIStats(&s))
	}

	return results
}

// parseCLIStats converts Docker CLI stats output to protobuf.
// Note: CLI stats are string-formatted (e.g., "0.50%", "100MiB / 1GiB").
// We extract approximate values where possible.
func parseCLIStats(s *dockerCLIStats) *scoutpb.DockerContainerStats {
	return &scoutpb.DockerContainerStats{
		ContainerId:   s.Container,
		Name:          s.Name,
		CpuPercent:    parsePercent(s.CPUPerc),
		MemoryPercent: parsePercent(s.MemPerc),
	}
}

// parsePercent parses a percentage string like "0.50%" to a float64.
func parsePercent(s string) float64 {
	// Remove % suffix and parse.
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
