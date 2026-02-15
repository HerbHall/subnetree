package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
)

// dockerContainer represents a container from the Docker API list endpoint.
type dockerContainer struct {
	ID    string   `json:"Id"`
	Names []string `json:"Names"`
}

// dockerStats represents the Docker API stats response.
type dockerStats struct {
	Read        string                       `json:"read"`
	PreRead     string                       `json:"preread"`
	CPUStats    dockerCPUStats               `json:"cpu_stats"`
	PreCPUStats dockerCPUStats               `json:"precpu_stats"`
	MemoryStats dockerMemStats               `json:"memory_stats"`
	Networks    map[string]dockerNetStats    `json:"networks"`
	BlkioStats  dockerBlkioStats             `json:"blkio_stats"`
	PidsStats   dockerPidsStats              `json:"pids_stats"`
}

type dockerCPUStats struct {
	CPUUsage    dockerCPUUsage `json:"cpu_usage"`
	SystemUsage uint64         `json:"system_cpu_usage"`
	OnlineCPUs  uint32         `json:"online_cpus"`
}

type dockerCPUUsage struct {
	TotalUsage uint64 `json:"total_usage"`
}

type dockerMemStats struct {
	Usage uint64 `json:"usage"`
	Limit uint64 `json:"limit"`
}

type dockerNetStats struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

type dockerBlkioStats struct {
	IoServiceBytesRecursive []dockerBlkioEntry `json:"io_service_bytes_recursive"`
}

type dockerBlkioEntry struct {
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}

type dockerPidsStats struct {
	Current uint64 `json:"current"`
}

// parseContainerList parses the JSON response from /containers/json.
func parseContainerList(r io.Reader) ([]dockerContainer, error) {
	var containers []dockerContainer
	if err := json.NewDecoder(r).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decode container list: %w", err)
	}
	return containers, nil
}

// parseContainerStats parses the JSON response from /containers/{id}/stats.
func parseContainerStats(r io.Reader) (*dockerStats, error) {
	var stats dockerStats
	if err := json.NewDecoder(r).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode container stats: %w", err)
	}
	return &stats, nil
}

// calculateCPUPercent computes the CPU usage percentage from Docker stats.
// Formula from Docker CLI: https://github.com/moby/moby/blob/master/api/types/stats.go
func calculateCPUPercent(stats *dockerStats) float64 {
	// Guard against uint64 underflow: if current < previous, delta is invalid.
	if stats.CPUStats.CPUUsage.TotalUsage < stats.PreCPUStats.CPUUsage.TotalUsage {
		return 0.0
	}
	if stats.CPUStats.SystemUsage <= stats.PreCPUStats.SystemUsage {
		return 0.0
	}
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	cpus := float64(stats.CPUStats.OnlineCPUs)
	if cpus == 0 {
		cpus = 1
	}
	return (cpuDelta / systemDelta) * cpus * 100.0
}

// statsToProto converts parsed Docker stats into protobuf format.
func statsToProto(containerID, name string, stats *dockerStats) *scoutpb.DockerContainerStats {
	// Sum network bytes across all interfaces.
	var rxBytes, txBytes uint64
	for _, net := range stats.Networks {
		rxBytes += net.RxBytes
		txBytes += net.TxBytes
	}

	// Sum block I/O bytes.
	var readBytes, writeBytes uint64
	for _, entry := range stats.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(entry.Op) {
		case "read":
			readBytes += entry.Value
		case "write":
			writeBytes += entry.Value
		}
	}

	// Memory percent.
	var memPercent float64
	if stats.MemoryStats.Limit > 0 {
		memPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}

	return &scoutpb.DockerContainerStats{
		ContainerId:      containerID,
		Name:             name,
		CpuPercent:       calculateCPUPercent(stats),
		MemoryUsageBytes: stats.MemoryStats.Usage,
		MemoryLimitBytes: stats.MemoryStats.Limit,
		MemoryPercent:    memPercent,
		NetworkRxBytes:   rxBytes,
		NetworkTxBytes:   txBytes,
		BlockReadBytes:   readBytes,
		BlockWriteBytes:  writeBytes,
		Pids:             uint32(stats.PidsStats.Current),
	}
}

// containerName extracts a clean name from Docker container names (removes leading /).
func containerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}
