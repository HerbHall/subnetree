package svcmap

import (
	"context"
	"fmt"

	"github.com/HerbHall/subnetree/pkg/models"
)

// HardwareSource provides hardware profile data for utilization calculations.
type HardwareSource interface {
	GetHardwareProfile(ctx context.Context, agentID string) (*HardwareInfo, error)
}

// DeviceInfo provides device metadata for utilization computations.
type DeviceInfo struct {
	DeviceID string
	Hostname string
	AgentID  string
}

// HardwareInfo is a lightweight DTO for hardware profile data relevant to utilization.
type HardwareInfo struct {
	TotalMemoryBytes int64
	TotalDiskBytes   int64
	UsedDiskBytes    int64
	CPUCores         int
}

// ComputeGrade assigns a letter grade (A-F) based on the highest resource usage.
// A: <30%, B: <60%, C: 60-80%, D: 80-90%, F: >90%
func ComputeGrade(cpu, mem, disk float64) string {
	peak := cpu
	if mem > peak {
		peak = mem
	}
	if disk > peak {
		peak = disk
	}

	switch {
	case peak < 30:
		return "A"
	case peak < 60:
		return "B"
	case peak < 80:
		return "C"
	case peak < 90:
		return "D"
	default:
		return "F"
	}
}

// ComputeDeviceUtilization calculates utilization for a single device.
func ComputeDeviceUtilization(
	ctx context.Context,
	store *Store,
	device DeviceInfo,
	hwSource HardwareSource,
) (*models.UtilizationSummary, error) {
	// Get aggregated service stats from the store.
	allStats, err := store.GetUtilizationSummaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("get service stats: %w", err)
	}

	var stats *DeviceServiceStats
	for i := range allStats {
		if allStats[i].DeviceID == device.DeviceID {
			stats = &allStats[i]
			break
		}
	}

	summary := &models.UtilizationSummary{
		DeviceID: device.DeviceID,
		Hostname: device.Hostname,
	}

	if stats == nil {
		summary.Grade = "A"
		summary.Headroom = 100
		return summary, nil
	}

	summary.ServiceCount = stats.ServiceCount
	summary.CPUPercent = stats.TotalCPU

	// Get hardware info for percentage calculations.
	if device.AgentID != "" && hwSource != nil {
		hw, err := hwSource.GetHardwareProfile(ctx, device.AgentID)
		if err == nil && hw != nil {
			if hw.TotalMemoryBytes > 0 {
				summary.MemoryPercent = float64(stats.TotalMemory) / float64(hw.TotalMemoryBytes) * 100
			}
			if hw.TotalDiskBytes > 0 {
				summary.DiskPercent = float64(hw.UsedDiskBytes) / float64(hw.TotalDiskBytes) * 100
			}
		}
	}

	summary.Grade = ComputeGrade(summary.CPUPercent, summary.MemoryPercent, summary.DiskPercent)

	// Headroom is 100 minus the highest utilization percentage.
	peak := summary.CPUPercent
	if summary.MemoryPercent > peak {
		peak = summary.MemoryPercent
	}
	if summary.DiskPercent > peak {
		peak = summary.DiskPercent
	}
	summary.Headroom = 100 - peak

	return summary, nil
}

// ComputeFleetSummary aggregates utilization across all devices.
func ComputeFleetSummary(summaries []models.UtilizationSummary) *models.FleetSummary {
	fleet := &models.FleetSummary{
		ByGrade: make(map[string]int),
	}

	if len(summaries) == 0 {
		return fleet
	}

	fleet.TotalDevices = len(summaries)

	var totalCPU, totalMem float64
	for i := range summaries {
		s := &summaries[i]
		fleet.TotalServices += s.ServiceCount
		totalCPU += s.CPUPercent
		totalMem += s.MemoryPercent
		fleet.ByGrade[s.Grade]++

		switch s.Grade {
		case "A":
			fleet.Underutilized = append(fleet.Underutilized, s.Hostname)
		case "D", "F":
			fleet.Overloaded = append(fleet.Overloaded, s.Hostname)
		}
	}

	fleet.AvgCPU = totalCPU / float64(fleet.TotalDevices)
	fleet.AvgMemory = totalMem / float64(fleet.TotalDevices)

	return fleet
}
