//go:build !windows

package metrics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

const dockerSocket = "/var/run/docker.sock"

// collectDockerStats collects per-container stats via the Docker API over Unix socket.
// Returns nil if Docker is not available (graceful degradation).
func collectDockerStats(ctx context.Context, logger *zap.Logger) []*scoutpb.DockerContainerStats {
	// Check if Docker socket exists.
	if _, err := os.Stat(dockerSocket); err != nil {
		return nil
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", dockerSocket, 5*time.Second)
			},
		},
		Timeout: 30 * time.Second,
	}

	// List running containers.
	listReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://localhost/v1.41/containers/json", http.NoBody)
	if err != nil {
		logger.Debug("docker: failed to create list request", zap.Error(err))
		return nil
	}

	listResp, err := client.Do(listReq)
	if err != nil {
		logger.Debug("docker: failed to list containers", zap.Error(err))
		return nil
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		logger.Debug("docker: list containers returned non-200",
			zap.Int("status", listResp.StatusCode))
		return nil
	}

	containers, err := parseContainerList(listResp.Body)
	if err != nil {
		logger.Debug("docker: failed to parse container list", zap.Error(err))
		return nil
	}

	if len(containers) == 0 {
		return nil
	}

	var results []*scoutpb.DockerContainerStats
	for _, c := range containers {
		stats := fetchContainerStats(ctx, client, logger, c.ID)
		if stats == nil {
			continue
		}
		results = append(results, statsToProto(c.ID[:12], containerName(c.Names), stats))
	}

	return results
}

// fetchContainerStats fetches stats for a single container.
func fetchContainerStats(ctx context.Context, client *http.Client, logger *zap.Logger, containerID string) *dockerStats {
	url := fmt.Sprintf("http://localhost/v1.41/containers/%s/stats?stream=false", containerID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		logger.Debug("docker: failed to create stats request",
			zap.String("container", containerID[:12]),
			zap.Error(err))
		return nil
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Debug("docker: failed to fetch stats",
			zap.String("container", containerID[:12]),
			zap.Error(err))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	stats, err := parseContainerStats(resp.Body)
	if err != nil {
		logger.Debug("docker: failed to parse stats",
			zap.String("container", containerID[:12]),
			zap.Error(err))
		return nil
	}

	return stats
}
