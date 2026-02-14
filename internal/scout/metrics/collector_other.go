//go:build !windows

package metrics

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

// linuxCollector gathers system metrics from /proc on Linux.
// On other non-Windows platforms (macOS, FreeBSD), /proc may not exist;
// each method returns partial data and logs warnings on failure.
type linuxCollector struct {
	logger *zap.Logger
}

// Compile-time guard.
var _ Collector = (*linuxCollector)(nil)

func newPlatformCollector(logger *zap.Logger) Collector {
	return &linuxCollector{logger: logger}
}

func (c *linuxCollector) Collect(ctx context.Context) (*scoutpb.SystemMetrics, error) {
	m := &scoutpb.SystemMetrics{}

	cpu, err := c.collectCPU(ctx)
	if err != nil {
		c.logger.Debug("cpu metrics failed", zap.Error(err))
	} else {
		m.CpuPercent = cpu
	}

	memPercent, memUsed, memTotal, err := c.collectMemory()
	if err != nil {
		c.logger.Debug("memory metrics failed", zap.Error(err))
	} else {
		m.MemoryPercent = memPercent
		m.MemoryUsedBytes = memUsed
		m.MemoryTotalBytes = memTotal
	}

	disks, err := c.collectDisks()
	if err != nil {
		c.logger.Debug("disk metrics failed", zap.Error(err))
	} else {
		m.Disks = disks
	}

	networks, err := c.collectNetwork()
	if err != nil {
		c.logger.Debug("network metrics failed", zap.Error(err))
	} else {
		m.Networks = networks
	}

	return m, nil
}

// collectCPU reads /proc/stat twice with a 200ms delta to calculate CPU usage.
func (c *linuxCollector) collectCPU(ctx context.Context) (float64, error) {
	idle1, total1, err := readCPUStat()
	if err != nil {
		return 0, fmt.Errorf("first /proc/stat read: %w", err)
	}

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-time.After(200 * time.Millisecond):
	}

	idle2, total2, err := readCPUStat()
	if err != nil {
		return 0, fmt.Errorf("second /proc/stat read: %w", err)
	}

	idleDelta := idle2 - idle1
	totalDelta := total2 - total1
	if totalDelta == 0 {
		return 0, nil
	}

	cpuPercent := (1.0 - float64(idleDelta)/float64(totalDelta)) * 100.0
	if cpuPercent < 0 {
		cpuPercent = 0
	}
	if cpuPercent > 100 {
		cpuPercent = 100
	}
	return cpuPercent, nil
}

// readCPUStat parses the first "cpu" line from /proc/stat and returns
// (idle ticks, total ticks). Fields: user nice system idle iowait irq softirq steal.
func readCPUStat() (idle, total uint64, err error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	return parseProcStat(string(data))
}

// parseProcStat extracts idle and total CPU ticks from /proc/stat content.
func parseProcStat(content string) (idle, total uint64, err error) {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0, fmt.Errorf("unexpected /proc/stat cpu line: %q", line)
		}
		// fields[0] = "cpu", fields[1..] = user nice system idle [iowait irq softirq steal ...]
		var sum uint64
		var idleVal uint64
		for i := 1; i < len(fields); i++ {
			v, parseErr := strconv.ParseUint(fields[i], 10, 64)
			if parseErr != nil {
				continue
			}
			sum += v
			if i == 4 { // idle field
				idleVal = v
			}
			if i == 5 { // iowait -- counted as idle time
				idleVal += v
			}
		}
		return idleVal, sum, nil
	}
	return 0, 0, fmt.Errorf("/proc/stat has no aggregate cpu line")
}

// collectMemory reads /proc/meminfo and returns (percent, used, total).
func (c *linuxCollector) collectMemory() (percent, used, total float64, err error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, err
	}
	return parseMeminfo(string(data))
}

// parseMeminfo parses /proc/meminfo content and returns (percent, used, total) in bytes.
func parseMeminfo(content string) (percent, used, total float64, err error) {
	fields := make(map[string]uint64)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		valStr = strings.TrimSuffix(valStr, " kB")
		valStr = strings.TrimSpace(valStr)

		v, parseErr := strconv.ParseUint(valStr, 10, 64)
		if parseErr != nil {
			continue
		}
		fields[key] = v
	}

	memTotal, ok := fields["MemTotal"]
	if !ok || memTotal == 0 {
		return 0, 0, 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
	}

	// MemAvailable is the best estimate of usable memory (kernel 3.14+).
	// Fall back to MemFree + Buffers + Cached on older kernels.
	memAvailable, ok := fields["MemAvailable"]
	if !ok {
		memAvailable = fields["MemFree"] + fields["Buffers"] + fields["Cached"]
	}

	// Convert kB to bytes.
	totalBytes := float64(memTotal) * 1024
	usedBytes := float64(memTotal-memAvailable) * 1024

	pct := usedBytes / totalBytes * 100.0
	return pct, usedBytes, totalBytes, nil
}

// collectDisks reads /proc/mounts and uses syscall.Statfs to get disk usage.
func (c *linuxCollector) collectDisks() ([]*scoutpb.DiskMetric, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil, err
	}
	return c.parseMountsAndStat(string(data))
}

// skipFSTypes contains filesystem types that should be ignored.
var skipFSTypes = map[string]bool{
	"tmpfs":     true,
	"devtmpfs":  true,
	"sysfs":     true,
	"proc":      true,
	"devpts":    true,
	"cgroup":    true,
	"cgroup2":   true,
	"pstore":    true,
	"securityfs": true,
	"debugfs":   true,
	"hugetlbfs": true,
	"mqueue":    true,
	"fusectl":   true,
	"configfs":  true,
	"binfmt_misc": true,
	"autofs":    true,
	"tracefs":   true,
	"overlay":   true,
	"squashfs":  true,
	"nsfs":      true,
	"efivarfs":  true,
	"bpf":       true,
	"rpc_pipefs": true,
	"nfsd":      true,
	"sunrpc":    true,
	"ramfs":     true,
}

func (c *linuxCollector) parseMountsAndStat(content string) ([]*scoutpb.DiskMetric, error) {
	seen := make(map[string]bool)
	var disks []*scoutpb.DiskMetric

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		mountPoint := fields[1]
		fsType := fields[2]

		if skipFSTypes[fsType] {
			continue
		}
		if seen[mountPoint] {
			continue
		}
		seen[mountPoint] = true

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mountPoint, &stat); err != nil {
			c.logger.Debug("statfs failed", zap.String("mount", mountPoint), zap.Error(err))
			continue
		}

		totalBytes := float64(stat.Blocks) * float64(stat.Bsize)
		freeBytes := float64(stat.Bfree) * float64(stat.Bsize)
		if totalBytes == 0 {
			continue
		}

		disks = append(disks, &scoutpb.DiskMetric{
			MountPoint: mountPoint,
			TotalBytes: totalBytes,
			UsedBytes:  totalBytes - freeBytes,
			FreeBytes:  freeBytes,
		})
	}

	return disks, nil
}

// collectNetwork reads /proc/net/dev for interface traffic counters.
func (c *linuxCollector) collectNetwork() ([]*scoutpb.NetworkMetric, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	return parseProcNetDev(string(data))
}

// parseProcNetDev parses /proc/net/dev content into network metrics.
// Format: "iface: rx_bytes rx_packets ... tx_bytes tx_packets ..."
func parseProcNetDev(content string) ([]*scoutpb.NetworkMetric, error) {
	var networks []*scoutpb.NetworkMetric

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		// Each interface line contains "iface:" followed by stats.
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}

		ifName := strings.TrimSpace(line[:colonIdx])
		if ifName == "lo" {
			continue
		}

		statsStr := strings.TrimSpace(line[colonIdx+1:])
		fields := strings.Fields(statsStr)
		if len(fields) < 10 {
			continue
		}

		rxBytes, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		// tx_bytes is the 9th field (index 8) in the stats portion.
		txBytes, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			continue
		}

		networks = append(networks, &scoutpb.NetworkMetric{
			InterfaceName: ifName,
			BytesRecv:     float64(rxBytes),
			BytesSent:     float64(txBytes),
		})
	}

	return networks, nil
}
