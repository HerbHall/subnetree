package metrics

import (
	"strings"
	"testing"
)

func TestCalculateCPUPercent(t *testing.T) {
	tests := []struct {
		name    string
		stats   *dockerStats
		wantMin float64
		wantMax float64
	}{
		{
			name: "zero delta",
			stats: &dockerStats{
				CPUStats:    dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 100}, SystemUsage: 1000, OnlineCPUs: 2},
				PreCPUStats: dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 100}, SystemUsage: 1000},
			},
			wantMin: 0,
			wantMax: 0.001,
		},
		{
			name: "50 percent single cpu",
			stats: &dockerStats{
				CPUStats:    dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 200}, SystemUsage: 2000, OnlineCPUs: 1},
				PreCPUStats: dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 100}, SystemUsage: 1800},
			},
			wantMin: 49.0,
			wantMax: 51.0,
		},
		{
			name: "multi cpu",
			stats: &dockerStats{
				CPUStats:    dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 200}, SystemUsage: 2000, OnlineCPUs: 4},
				PreCPUStats: dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 100}, SystemUsage: 1800},
			},
			wantMin: 190.0,
			wantMax: 210.0,
		},
		{
			name: "negative cpu delta returns zero",
			stats: &dockerStats{
				CPUStats:    dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 50}, SystemUsage: 2000, OnlineCPUs: 1},
				PreCPUStats: dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 100}, SystemUsage: 1800},
			},
			wantMin: 0,
			wantMax: 0.001,
		},
		{
			name: "zero system delta returns zero",
			stats: &dockerStats{
				CPUStats:    dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 200}, SystemUsage: 1000, OnlineCPUs: 1},
				PreCPUStats: dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 100}, SystemUsage: 1000},
			},
			wantMin: 0,
			wantMax: 0.001,
		},
		{
			name: "zero online cpus defaults to one",
			stats: &dockerStats{
				CPUStats:    dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 200}, SystemUsage: 2000, OnlineCPUs: 0},
				PreCPUStats: dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 100}, SystemUsage: 1800},
			},
			wantMin: 49.0,
			wantMax: 51.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCPUPercent(tt.stats)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateCPUPercent() = %f, want between %f and %f", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestParseContainerList(t *testing.T) {
	input := `[{"Id":"abc123def456","Names":["/myapp"]},{"Id":"xyz789","Names":["/db"]}]`
	containers, err := parseContainerList(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseContainerList() error: %v", err)
	}
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}
	if containers[0].ID != "abc123def456" {
		t.Errorf("container[0].ID = %q, want %q", containers[0].ID, "abc123def456")
	}
	if containers[1].Names[0] != "/db" {
		t.Errorf("container[1].Names[0] = %q, want %q", containers[1].Names[0], "/db")
	}
}

func TestParseContainerListEmpty(t *testing.T) {
	input := `[]`
	containers, err := parseContainerList(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseContainerList() error: %v", err)
	}
	if len(containers) != 0 {
		t.Fatalf("expected 0 containers, got %d", len(containers))
	}
}

func TestParseContainerListInvalid(t *testing.T) {
	input := `not json`
	_, err := parseContainerList(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseContainerStats(t *testing.T) {
	input := `{
		"cpu_stats": {"cpu_usage": {"total_usage": 500}, "system_cpu_usage": 5000, "online_cpus": 2},
		"precpu_stats": {"cpu_usage": {"total_usage": 400}, "system_cpu_usage": 4800},
		"memory_stats": {"usage": 104857600, "limit": 1073741824},
		"networks": {"eth0": {"rx_bytes": 1024, "tx_bytes": 2048}},
		"blkio_stats": {"io_service_bytes_recursive": [{"op": "read", "value": 4096}, {"op": "write", "value": 8192}]},
		"pids_stats": {"current": 5}
	}`
	stats, err := parseContainerStats(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseContainerStats() error: %v", err)
	}
	if stats.MemoryStats.Usage != 104857600 {
		t.Errorf("memory usage = %d, want 104857600", stats.MemoryStats.Usage)
	}
	if stats.MemoryStats.Limit != 1073741824 {
		t.Errorf("memory limit = %d, want 1073741824", stats.MemoryStats.Limit)
	}
	if stats.PidsStats.Current != 5 {
		t.Errorf("pids = %d, want 5", stats.PidsStats.Current)
	}
}

func TestStatsToProto(t *testing.T) {
	stats := &dockerStats{
		CPUStats:    dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 200}, SystemUsage: 2000, OnlineCPUs: 1},
		PreCPUStats: dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 100}, SystemUsage: 1800},
		MemoryStats: dockerMemStats{Usage: 100 * 1024 * 1024, Limit: 1024 * 1024 * 1024},
		Networks: map[string]dockerNetStats{
			"eth0": {RxBytes: 1024, TxBytes: 2048},
			"eth1": {RxBytes: 512, TxBytes: 256},
		},
		BlkioStats: dockerBlkioStats{
			IoServiceBytesRecursive: []dockerBlkioEntry{
				{Op: "Read", Value: 4096},
				{Op: "Write", Value: 8192},
			},
		},
		PidsStats: dockerPidsStats{Current: 10},
	}

	proto := statsToProto("abc123def456", "myapp", stats)

	if proto.ContainerId != "abc123def456" {
		t.Errorf("ContainerId = %q, want %q", proto.ContainerId, "abc123def456")
	}
	if proto.Name != "myapp" {
		t.Errorf("Name = %q, want %q", proto.Name, "myapp")
	}
	if proto.NetworkRxBytes != 1536 { // 1024 + 512
		t.Errorf("NetworkRxBytes = %d, want 1536", proto.NetworkRxBytes)
	}
	if proto.NetworkTxBytes != 2304 { // 2048 + 256
		t.Errorf("NetworkTxBytes = %d, want 2304", proto.NetworkTxBytes)
	}
	if proto.BlockReadBytes != 4096 {
		t.Errorf("BlockReadBytes = %d, want 4096", proto.BlockReadBytes)
	}
	if proto.BlockWriteBytes != 8192 {
		t.Errorf("BlockWriteBytes = %d, want 8192", proto.BlockWriteBytes)
	}
	if proto.Pids != 10 {
		t.Errorf("Pids = %d, want 10", proto.Pids)
	}
	if proto.MemoryPercent < 9.0 || proto.MemoryPercent > 10.0 {
		t.Errorf("MemoryPercent = %f, want ~9.77", proto.MemoryPercent)
	}
}

func TestStatsToProtoNoNetworks(t *testing.T) {
	stats := &dockerStats{
		CPUStats:    dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 200}, SystemUsage: 2000, OnlineCPUs: 1},
		PreCPUStats: dockerCPUStats{CPUUsage: dockerCPUUsage{TotalUsage: 100}, SystemUsage: 1800},
		MemoryStats: dockerMemStats{Usage: 0, Limit: 0},
	}

	proto := statsToProto("abc123", "test", stats)

	if proto.NetworkRxBytes != 0 {
		t.Errorf("NetworkRxBytes = %d, want 0", proto.NetworkRxBytes)
	}
	if proto.MemoryPercent != 0 {
		t.Errorf("MemoryPercent = %f, want 0 (no limit)", proto.MemoryPercent)
	}
}

func TestContainerName(t *testing.T) {
	tests := []struct {
		names []string
		want  string
	}{
		{[]string{"/myapp"}, "myapp"},
		{[]string{"/db"}, "db"},
		{[]string{"noprefix"}, "noprefix"},
		{nil, ""},
		{[]string{}, ""},
	}
	for _, tt := range tests {
		got := containerName(tt.names)
		if got != tt.want {
			t.Errorf("containerName(%v) = %q, want %q", tt.names, got, tt.want)
		}
	}
}
