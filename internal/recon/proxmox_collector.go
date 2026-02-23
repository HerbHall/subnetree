package recon

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ProxmoxCollector collects hardware information from a Proxmox VE REST API.
// This provides agentless hardware profiling for Proxmox-managed infrastructure.
type ProxmoxCollector struct {
	baseURL     string
	tokenID     string //nolint:gosec // G101: token identifier, not a credential
	tokenSecret string //nolint:gosec // G101: field name, not a credential
	httpClient  *http.Client
	logger      *zap.Logger
}

// ProxmoxNode represents a node returned by the Proxmox /nodes endpoint.
type ProxmoxNode struct {
	Node    string  `json:"node"`
	Status  string  `json:"status"`
	CPU     float64 `json:"cpu"`
	Maxcpu  int     `json:"maxcpu"`
	Mem     int64   `json:"mem"`
	Maxmem  int64   `json:"maxmem"`
	Disk    int64   `json:"disk"`
	Maxdisk int64   `json:"maxdisk"`
	Uptime  int64   `json:"uptime"`
}

// ProxmoxVM represents a QEMU VM from the Proxmox /qemu endpoint.
type ProxmoxVM struct {
	VMID   int    `json:"vmid"`
	Name   string `json:"name"`
	Status string `json:"status"`
	CPU    int    `json:"cpus"`
	Maxmem int64  `json:"maxmem"`
}

// ProxmoxContainer represents an LXC container from the Proxmox /lxc endpoint.
type ProxmoxContainer struct {
	VMID   int    `json:"vmid"`
	Name   string `json:"name"`
	Status string `json:"status"`
	CPU    int    `json:"cpus"`
	Maxmem int64  `json:"maxmem"`
}

// proxmoxDisk represents a disk from the Proxmox /disks/list endpoint.
type proxmoxDisk struct {
	DevPath string `json:"devpath"`
	Model   string `json:"model"`
	Serial  string `json:"serial"`
	Size    int64  `json:"size"`
	Type    string `json:"type"` // "ssd", "hdd", "nvme"
}

// proxmoxResponse wraps the standard Proxmox API response envelope.
type proxmoxResponse struct {
	Data json.RawMessage `json:"data"`
}

// NewProxmoxCollector creates a new collector for querying the Proxmox VE REST API.
// The tokenID should be in the format "USER@REALM!TOKENID" and tokenSecret is
// the corresponding API token secret.
func NewProxmoxCollector(baseURL, tokenID, tokenSecret string, logger *zap.Logger) *ProxmoxCollector {
	return &ProxmoxCollector{
		baseURL:     baseURL,
		tokenID:     tokenID,
		tokenSecret: tokenSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
					//nolint:gosec // G402: Proxmox commonly uses self-signed certs in homelab environments.
					InsecureSkipVerify: true,
				},
			},
		},
		logger: logger,
	}
}

// CollectNodes returns all nodes in the Proxmox cluster.
func (c *ProxmoxCollector) CollectNodes(ctx context.Context) ([]ProxmoxNode, error) {
	body, err := c.apiGet(ctx, "/api2/json/nodes")
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	var nodes []ProxmoxNode
	if err := json.Unmarshal(body, &nodes); err != nil {
		return nil, fmt.Errorf("parse nodes response: %w", err)
	}

	return nodes, nil
}

// CollectNodeHardware retrieves hardware details for a specific Proxmox node.
// Returns the mapped DeviceHardware and DeviceStorage slices.
func (c *ProxmoxCollector) CollectNodeHardware(ctx context.Context, node string) (*models.DeviceHardware, []models.DeviceStorage, error) {
	// Get node status for CPU/RAM.
	statusBody, err := c.apiGet(ctx, "/api2/json/nodes/"+node+"/status")
	if err != nil {
		return nil, nil, fmt.Errorf("get node status: %w", err)
	}

	var status struct {
		CPU struct {
			Model string `json:"model"`
			Cores int    `json:"cores"`
			CPUs  int    `json:"cpus"`
		} `json:"cpuinfo"`
		Memory struct {
			Total int64 `json:"total"`
		} `json:"memory"`
		Uptime int64 `json:"uptime"`
	}
	if err := json.Unmarshal(statusBody, &status); err != nil {
		return nil, nil, fmt.Errorf("parse node status: %w", err)
	}

	now := time.Now().UTC()
	hw := &models.DeviceHardware{
		Hostname:         node,
		CPUModel:         status.CPU.Model,
		CPUCores:         status.CPU.Cores,
		CPUThreads:       status.CPU.CPUs,
		RAMTotalMB:       int(status.Memory.Total / (1024 * 1024)),
		PlatformType:     "baremetal",
		Hypervisor:       "proxmox",
		CollectionSource: "proxmox-api",
		CollectedAt:      &now,
	}

	// Get disk list.
	var storage []models.DeviceStorage
	diskBody, err := c.apiGet(ctx, "/api2/json/nodes/"+node+"/disks/list")
	if err != nil {
		c.logger.Debug("failed to get proxmox disk list (may require root permissions)",
			zap.String("node", node),
			zap.Error(err),
		)
	} else {
		var disks []proxmoxDisk
		if err := json.Unmarshal(diskBody, &disks); err != nil {
			c.logger.Debug("failed to parse proxmox disk list", zap.Error(err))
		} else {
			for _, d := range disks {
				storage = append(storage, models.DeviceStorage{
					ID:               uuid.New().String(),
					Name:             d.DevPath,
					DiskType:         normalizeDiskType(d.Type),
					CapacityGB:       int(d.Size / (1024 * 1024 * 1024)),
					Model:            d.Model,
					CollectionSource: "proxmox-api",
					CollectedAt:      &now,
				})
			}
		}
	}

	return hw, storage, nil
}

// CollectVMs returns all QEMU VMs on a Proxmox node.
func (c *ProxmoxCollector) CollectVMs(ctx context.Context, node string) ([]ProxmoxVM, error) {
	body, err := c.apiGet(ctx, "/api2/json/nodes/"+node+"/qemu")
	if err != nil {
		return nil, fmt.Errorf("list VMs: %w", err)
	}

	var vms []ProxmoxVM
	if err := json.Unmarshal(body, &vms); err != nil {
		return nil, fmt.Errorf("parse VMs response: %w", err)
	}

	return vms, nil
}

// CollectContainers returns all LXC containers on a Proxmox node.
func (c *ProxmoxCollector) CollectContainers(ctx context.Context, node string) ([]ProxmoxContainer, error) {
	body, err := c.apiGet(ctx, "/api2/json/nodes/"+node+"/lxc")
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var containers []ProxmoxContainer
	if err := json.Unmarshal(body, &containers); err != nil {
		return nil, fmt.Errorf("parse containers response: %w", err)
	}

	return containers, nil
}

// ProxmoxResourceStatus holds resource utilisation metrics for a VM or container.
type ProxmoxResourceStatus struct {
	CPUPercent  float64
	MemUsedMB   int
	MemTotalMB  int
	DiskUsedGB  int
	DiskTotalGB int
	UptimeSec   int64
	NetInBytes  int64
	NetOutBytes int64
}

// CollectVMStatus returns live resource utilisation for a QEMU VM.
func (c *ProxmoxCollector) CollectVMStatus(ctx context.Context, node string, vmid int) (*ProxmoxResourceStatus, error) {
	path := fmt.Sprintf("/api2/json/nodes/%s/qemu/%d/status/current", node, vmid)
	return c.collectResourceStatus(ctx, path)
}

// CollectContainerStatus returns live resource utilisation for an LXC container.
func (c *ProxmoxCollector) CollectContainerStatus(ctx context.Context, node string, vmid int) (*ProxmoxResourceStatus, error) {
	path := fmt.Sprintf("/api2/json/nodes/%s/lxc/%d/status/current", node, vmid)
	return c.collectResourceStatus(ctx, path)
}

// collectResourceStatus fetches and maps a Proxmox status/current response.
func (c *ProxmoxCollector) collectResourceStatus(ctx context.Context, path string) (*ProxmoxResourceStatus, error) {
	body, err := c.apiGet(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get resource status: %w", err)
	}

	var raw struct {
		CPU     float64 `json:"cpu"`
		Mem     int64   `json:"mem"`
		Maxmem  int64   `json:"maxmem"`
		Disk    int64   `json:"disk"`
		Maxdisk int64   `json:"maxdisk"`
		Uptime  int64   `json:"uptime"`
		Netin   int64   `json:"netin"`
		Netout  int64   `json:"netout"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse resource status: %w", err)
	}

	return &ProxmoxResourceStatus{
		CPUPercent:  raw.CPU * 100,
		MemUsedMB:   int(raw.Mem / (1024 * 1024)),
		MemTotalMB:  int(raw.Maxmem / (1024 * 1024)),
		DiskUsedGB:  int(raw.Disk / (1024 * 1024 * 1024)),
		DiskTotalGB: int(raw.Maxdisk / (1024 * 1024 * 1024)),
		UptimeSec:   raw.Uptime,
		NetInBytes:  raw.Netin,
		NetOutBytes: raw.Netout,
	}, nil
}

// apiGet performs an authenticated GET request to the Proxmox API and
// returns the unwrapped "data" field from the response envelope.
func (c *ProxmoxCollector) apiGet(ctx context.Context, path string) (json.RawMessage, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Proxmox API token authentication.
	req.Header.Set("Authorization", "PVEAPIToken="+c.tokenID+"="+c.tokenSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxmox API returned %d: %s", resp.StatusCode, string(body))
	}

	var envelope proxmoxResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse response envelope: %w", err)
	}

	return envelope.Data, nil
}

// normalizeDiskType converts Proxmox disk type strings to the standard format.
func normalizeDiskType(pveType string) string {
	switch pveType {
	case "ssd":
		return "SSD"
	case "hdd":
		return "HDD"
	case "nvme":
		return "NVMe"
	default:
		return "Unknown"
	}
}
