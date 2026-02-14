//go:build !windows

package profiler

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

func collectHardware(_ context.Context, logger *zap.Logger) (*scoutpb.HardwareProfile, error) {
	hw := &scoutpb.HardwareProfile{}

	// CPU info from /proc/cpuinfo.
	cpuModel, physCores, logicalCPUs := readCPUInfo(logger)
	hw.CpuModel = cpuModel
	hw.CpuCores = physCores
	hw.CpuThreads = logicalCPUs

	// RAM from /proc/meminfo.
	hw.RamBytes = readTotalRAM(logger)

	// Disk info from /sys/block/.
	hw.Disks = readBlockDevices(logger)

	// NIC info from /sys/class/net/.
	hw.Nics = readNetworkInterfaces(logger)

	// System info from DMI (may not exist in containers/VMs).
	readDMIInfo(logger, hw)

	return hw, nil
}

// readCPUInfo parses /proc/cpuinfo for CPU model, physical cores, and logical CPUs.
func readCPUInfo(logger *zap.Logger) (model string, physCores int32, logicalCPUs int32) {
	logicalCPUs = int32(runtime.NumCPU())

	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		logger.Debug("failed to read /proc/cpuinfo", zap.Error(err))
		return "", 0, logicalCPUs
	}

	return parseCPUInfo(string(data), logicalCPUs)
}

// parseCPUInfo extracts CPU model and physical core count from /proc/cpuinfo content.
func parseCPUInfo(content string, logicalCPUs int32) (model string, physCores int32, logical int32) {
	logical = logicalCPUs

	// Track unique physical cores via (physical_id, core_id) pairs.
	type coreKey struct {
		physID int
		coreID int
	}
	uniqueCores := make(map[coreKey]bool)

	var currentPhysID, currentCoreID int
	var hasPhysID, hasCoreID bool

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// End of a processor block -- record the core if we have both IDs.
			if hasPhysID && hasCoreID {
				uniqueCores[coreKey{currentPhysID, currentCoreID}] = true
			}
			hasPhysID = false
			hasCoreID = false
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "model name":
			if model == "" {
				model = val
			}
		case "physical id":
			if v, e := strconv.Atoi(val); e == nil {
				currentPhysID = v
				hasPhysID = true
			}
		case "core id":
			if v, e := strconv.Atoi(val); e == nil {
				currentCoreID = v
				hasCoreID = true
			}
		}
	}

	// Handle last processor block.
	if hasPhysID && hasCoreID {
		uniqueCores[coreKey{currentPhysID, currentCoreID}] = true
	}

	if len(uniqueCores) > 0 {
		physCores = int32(len(uniqueCores))
	} else {
		// Single-socket or container without physical/core IDs.
		physCores = logical
	}

	return model, physCores, logical
}

// readTotalRAM reads MemTotal from /proc/meminfo and returns bytes.
func readTotalRAM(logger *zap.Logger) int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		logger.Debug("failed to read /proc/meminfo for RAM", zap.Error(err))
		return 0
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		v, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		return v * 1024 // kB to bytes
	}
	return 0
}

// readBlockDevices reads disk info from /sys/block/.
func readBlockDevices(logger *zap.Logger) []*scoutpb.DiskInfo {
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		logger.Debug("failed to read /sys/block", zap.Error(err))
		return nil
	}

	var disks []*scoutpb.DiskInfo
	for _, entry := range entries {
		name := entry.Name()

		// Skip virtual devices.
		if strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "dm-") ||
			strings.HasPrefix(name, "zram") {
			continue
		}

		basePath := filepath.Join("/sys/block", name)

		// Read size (in 512-byte sectors).
		sizeData, err := os.ReadFile(filepath.Join(basePath, "size"))
		if err != nil {
			continue
		}
		sectors, err := strconv.ParseInt(strings.TrimSpace(string(sizeData)), 10, 64)
		if err != nil || sectors == 0 {
			continue
		}
		sizeBytes := sectors * 512

		disk := &scoutpb.DiskInfo{
			Name:      name,
			SizeBytes: sizeBytes,
		}

		// Read model (may not exist for all devices).
		modelData, err := os.ReadFile(filepath.Join(basePath, "device", "model"))
		if err == nil {
			disk.Model = strings.TrimSpace(string(modelData))
		}

		// Classify disk type.
		disk.DiskType = classifyLinuxDiskType(name, basePath)

		disks = append(disks, disk)
	}

	return disks
}

// classifyLinuxDiskType determines if a block device is NVMe, SSD, or HDD.
func classifyLinuxDiskType(name, basePath string) string {
	if strings.HasPrefix(name, "nvme") {
		return "NVMe"
	}

	// Check rotational flag: 0 = SSD, 1 = HDD.
	rotData, err := os.ReadFile(filepath.Join(basePath, "queue", "rotational"))
	if err == nil {
		if strings.TrimSpace(string(rotData)) == "0" {
			return "SSD"
		}
		return "HDD"
	}

	return "Unknown"
}

// readNetworkInterfaces reads NIC info from /sys/class/net/.
func readNetworkInterfaces(logger *zap.Logger) []*scoutpb.NICInfo {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		logger.Debug("failed to read /sys/class/net", zap.Error(err))
		return nil
	}

	var nics []*scoutpb.NICInfo
	for _, entry := range entries {
		name := entry.Name()
		if name == "lo" {
			continue
		}

		basePath := filepath.Join("/sys/class/net", name)
		nic := &scoutpb.NICInfo{
			Name: name,
		}

		// MAC address.
		macData, err := os.ReadFile(filepath.Join(basePath, "address"))
		if err == nil {
			mac := strings.TrimSpace(string(macData))
			if mac != "00:00:00:00:00:00" {
				nic.MacAddress = mac
			}
		}

		// Speed in Mbps (may fail for down or virtual interfaces).
		speedData, err := os.ReadFile(filepath.Join(basePath, "speed"))
		if err == nil {
			if v, e := strconv.ParseInt(strings.TrimSpace(string(speedData)), 10, 64); e == nil && v > 0 {
				nic.SpeedMbps = v
			}
		}

		// Interface type: /sys/class/net/DEV/type (1=ethernet, 801=wifi, 772=loopback).
		typeData, err := os.ReadFile(filepath.Join(basePath, "type"))
		if err == nil {
			nic.NicType = classifyLinuxNICType(name, strings.TrimSpace(string(typeData)))
		}

		nics = append(nics, nic)
	}

	return nics
}

// classifyLinuxNICType determines NIC type from interface name and kernel type value.
func classifyLinuxNICType(name, kernelType string) string {
	// Check name patterns first for common virtual interfaces.
	n := strings.ToLower(name)
	switch {
	case strings.HasPrefix(n, "wl") || strings.HasPrefix(n, "wlan"):
		return "wifi"
	case strings.HasPrefix(n, "veth") || strings.HasPrefix(n, "docker") ||
		strings.HasPrefix(n, "br-") || strings.HasPrefix(n, "virbr"):
		return "virtual"
	}

	// Kernel type codes.
	switch kernelType {
	case "1": // ARPHRD_ETHER
		return "ethernet"
	case "801": // ARPHRD_IEEE80211 (wifi)
		return "wifi"
	default:
		return "ethernet"
	}
}

// readDMIInfo reads system manufacturer, model, serial, and BIOS version from
// /sys/class/dmi/id/. These files may not exist in containers or some VMs.
func readDMIInfo(logger *zap.Logger, hw *scoutpb.HardwareProfile) {
	dmiPath := "/sys/class/dmi/id"

	readDMIField := func(filename string) string {
		data, err := os.ReadFile(filepath.Join(dmiPath, filename))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}

	hw.SystemManufacturer = readDMIField("board_vendor")
	hw.SystemModel = readDMIField("product_name")
	hw.SerialNumber = readDMIField("product_serial")
	hw.BiosVersion = readDMIField("bios_version")

	// Log if DMI is completely unavailable (common in containers).
	if hw.SystemManufacturer == "" && hw.SystemModel == "" {
		if _, err := os.Stat(dmiPath); err != nil {
			logger.Debug("DMI info unavailable (container or VM without DMI)", zap.Error(err))
		}
	}

	// Filter placeholder values that some VMs use.
	placeholders := []string{"To Be Filled By O.E.M.", "Not Specified", "Default string", "System Product Name"}
	for _, p := range placeholders {
		if hw.SystemManufacturer == p {
			hw.SystemManufacturer = ""
		}
		if hw.SystemModel == p {
			hw.SystemModel = ""
		}
		if hw.SerialNumber == p {
			hw.SerialNumber = ""
		}
	}

}
