package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
)

// DockerCollector discovers Docker containers and captures their configuration
// using the Docker Engine API over a Unix socket (Linux) or TCP (Windows).
// It uses raw net/http with no Docker SDK dependency.
type DockerCollector struct {
	client     *http.Client
	socketPath string
	endpoint   string
	logger     *zap.Logger
}

// Compile-time interface guard.
var _ Collector = (*DockerCollector)(nil)

// dockerContainer holds the subset of fields returned by GET /containers/json.
type dockerContainer struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Image  string            `json:"Image"`
	State  string            `json:"State"`
	Status string            `json:"Status"`
	Ports  []dockerPort      `json:"Ports"`
	Mounts []dockerMount     `json:"Mounts"`
	Labels map[string]string `json:"Labels"`
}

type dockerPort struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

type dockerMount struct {
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
}

// NewDockerCollector creates a DockerCollector. If socketPath is empty, the
// platform default is used: /var/run/docker.sock on Linux, or TCP
// http://localhost:2375 on Windows.
func NewDockerCollector(socketPath string, logger *zap.Logger) *DockerCollector {
	dc := &DockerCollector{
		logger: logger,
	}

	if socketPath != "" {
		dc.socketPath = socketPath
	} else {
		dc.socketPath = detectDockerSocket()
	}

	if strings.HasPrefix(dc.socketPath, "tcp://") || strings.HasPrefix(dc.socketPath, "http://") {
		// TCP endpoint (typically Windows or remote Docker).
		dc.endpoint = strings.TrimPrefix(dc.socketPath, "tcp://")
		dc.endpoint = "http://" + dc.endpoint
		dc.client = &http.Client{Timeout: 30 * time.Second}
	} else {
		// Unix socket (Linux/macOS).
		dc.endpoint = "http://docker"
		dc.client = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", dc.socketPath)
				},
			},
		}
	}

	return dc
}

// detectDockerSocket returns the platform-appropriate default Docker socket.
func detectDockerSocket() string {
	if runtime.GOOS == "windows" {
		return "tcp://localhost:2375"
	}
	return "/var/run/docker.sock"
}

func (d *DockerCollector) Name() string {
	return "docker"
}

// Available returns true if the Docker Engine API is reachable.
func (d *DockerCollector) Available() bool {
	// For Unix sockets, check the socket file exists first.
	if !strings.HasPrefix(d.socketPath, "tcp://") && !strings.HasPrefix(d.socketPath, "http://") {
		if _, err := os.Stat(d.socketPath); err != nil {
			return false
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint+"/_ping", http.NoBody)
	if err != nil {
		return false
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// Discover queries the Docker Engine for all containers and returns them as Applications.
func (d *DockerCollector) Discover(ctx context.Context) ([]Application, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.endpoint+"/containers/json?all=true", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("docker discover: create request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker discover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker discover: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var containers []dockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("docker discover: decode response: %w", err)
	}

	now := time.Now().UTC()
	apps := make([]Application, 0, len(containers))
	for i := range containers {
		c := &containers[i]

		name := c.ID[:12]
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		status := "active"
		if c.State != "running" {
			status = "inactive"
		}

		meta, _ := json.Marshal(map[string]any{
			"image":  c.Image,
			"status": c.Status,
			"ports":  c.Ports,
			"mounts": c.Mounts,
			"labels": c.Labels,
		})

		apps = append(apps, Application{
			ID:           c.ID,
			Name:         name,
			AppType:      "docker-container",
			Collector:    "docker",
			Status:       status,
			Metadata:     string(meta),
			DiscoveredAt: now,
			UpdatedAt:    now,
		})
	}

	return apps, nil
}

// Collect retrieves the full inspect output for a specific container.
func (d *DockerCollector) Collect(ctx context.Context, appID string) (*CollectedConfig, error) {
	url := fmt.Sprintf("%s/containers/%s/json", d.endpoint, appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("docker collect: create request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker collect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("docker collect: container %s not found", appID)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docker collect: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("docker collect: read body: %w", err)
	}

	return &CollectedConfig{
		Content: string(body),
		Format:  "json",
	}, nil
}
