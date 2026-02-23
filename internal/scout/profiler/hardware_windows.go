//go:build windows

package profiler

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

func collectHardware(ctx context.Context, logger *zap.Logger) (*scoutpb.HardwareProfile, error) {
	hw := &scoutpb.HardwareProfile{}

	// CPU info.
	cpuRows, err := runWMIC(ctx, "cpu", "Name,NumberOfCores,NumberOfLogicalProcessors")
	if err != nil {
		logger.Debug("wmic cpu failed", zap.Error(err))
	} else if len(cpuRows) > 0 {
		row := cpuRows[0]
		hw.CpuModel = row["Name"]
		if v, e := strconv.ParseInt(row["NumberOfCores"], 10, 32); e == nil {
			hw.CpuCores = int32(v)
		}
		if v, e := strconv.ParseInt(row["NumberOfLogicalProcessors"], 10, 32); e == nil {
			hw.CpuThreads = int32(v)
		}
	}

	// RAM info.
	memRows, err := runWMIC(ctx, "memorychip", "Capacity")
	if err != nil {
		logger.Debug("wmic memorychip failed", zap.Error(err))
	} else {
		var totalRAM int64
		for _, row := range memRows {
			if v, e := strconv.ParseInt(row["Capacity"], 10, 64); e == nil {
				totalRAM += v
			}
		}
		hw.RamBytes = totalRAM
	}

	// Disk info.
	diskRows, err := runWMIC(ctx, "diskdrive", "Model,Size,MediaType,SerialNumber")
	if err != nil {
		logger.Debug("wmic diskdrive failed", zap.Error(err))
	} else {
		for _, row := range diskRows {
			disk := &scoutpb.DiskInfo{
				Model:  row["Model"],
				Serial: row["SerialNumber"],
			}
			if v, e := strconv.ParseInt(row["Size"], 10, 64); e == nil {
				disk.SizeBytes = v
			}
			disk.DiskType = classifyDiskType(row["MediaType"], row["Model"])
			disk.Name = row["Model"]
			hw.Disks = append(hw.Disks, disk)
		}
	}

	// GPU info.
	gpuRows, err := runWMIC(ctx, "path win32_videocontroller", "Name,AdapterRAM,DriverVersion")
	if err != nil {
		logger.Debug("wmic gpu failed", zap.Error(err))
	} else {
		for _, row := range gpuRows {
			gpu := &scoutpb.GPUInfo{
				Model:         row["Name"],
				DriverVersion: row["DriverVersion"],
			}
			if v, e := strconv.ParseInt(row["AdapterRAM"], 10, 64); e == nil {
				gpu.VramBytes = v
			}
			hw.Gpus = append(hw.Gpus, gpu)
		}
	}

	// NIC info.
	nicRows, err := runWMIC(ctx, "nic where \"NetEnabled=true\"", "Name,Speed,MACAddress,NetConnectionID")
	if err != nil {
		logger.Debug("wmic nic failed", zap.Error(err))
	} else {
		for _, row := range nicRows {
			nic := &scoutpb.NICInfo{
				Name:       row["NetConnectionID"],
				MacAddress: row["MACAddress"],
				NicType:    classifyNICType(row["Name"]),
			}
			if v, e := strconv.ParseInt(row["Speed"], 10, 64); e == nil {
				nic.SpeedMbps = v / 1_000_000
			}
			hw.Nics = append(hw.Nics, nic)
		}
	}

	// BIOS version.
	biosRows, err := runWMIC(ctx, "bios", "SMBIOSBIOSVersion,SerialNumber")
	if err != nil {
		logger.Debug("wmic bios failed", zap.Error(err))
	} else if len(biosRows) > 0 {
		hw.BiosVersion = biosRows[0]["SMBIOSBIOSVersion"]
		hw.SerialNumber = biosRows[0]["SerialNumber"]
	}

	// System manufacturer and model.
	csRows, err := runWMIC(ctx, "computersystem", "Manufacturer,Model")
	if err != nil {
		logger.Debug("wmic computersystem failed", zap.Error(err))
	} else if len(csRows) > 0 {
		hw.SystemManufacturer = csRows[0]["Manufacturer"]
		hw.SystemModel = csRows[0]["Model"]
	}

	return hw, nil
}

// runWMIC queries Windows hardware info via PowerShell Get-CimInstance (preferred)
// with a fallback to the legacy wmic command for older Windows versions.
func runWMIC(ctx context.Context, wmicClass, fields string) ([]map[string]string, error) {
	// Try PowerShell Get-CimInstance first (wmic removed in Windows 11 24H2+).
	if className, filter, ok := wmicToCIMClass(wmicClass); ok {
		rows, err := runCIMQuery(ctx, className, filter, fields)
		if err == nil && len(rows) > 0 {
			return rows, nil
		}
	}

	// Fall back to legacy wmic.
	args := strings.Fields(wmicClass)
	args = append(args, "get", fields, "/format:csv")

	cmd := exec.CommandContext(ctx, "wmic", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("wmic %s: %w (stderr: %s)", wmicClass, err, stderr.String())
	}

	return parseWMICCSV(stdout.Bytes())
}

// wmicToCIMClass maps a wmic alias to the corresponding CIM class name and optional filter.
func wmicToCIMClass(wmicClass string) (className, filter string, ok bool) {
	mapping := map[string][2]string{
		"cpu":                        {"Win32_Processor", ""},
		"memorychip":                 {"Win32_PhysicalMemory", ""},
		"diskdrive":                  {"Win32_DiskDrive", ""},
		"os":                         {"Win32_OperatingSystem", ""},
		"bios":                       {"Win32_BIOS", ""},
		"computersystem":             {"Win32_ComputerSystem", ""},
		"path win32_videocontroller": {"Win32_VideoController", ""},
	}

	if m, found := mapping[wmicClass]; found {
		return m[0], m[1], true
	}

	// Handle "nic where ..." pattern.
	if strings.HasPrefix(wmicClass, "nic") {
		return "Win32_NetworkAdapter", "NetEnabled=True", true
	}

	return "", "", false
}

// runCIMQuery executes a PowerShell Get-CimInstance query and parses the CSV output.
func runCIMQuery(ctx context.Context, className, filter, fields string) ([]map[string]string, error) {
	var psCmd string
	if filter != "" {
		psCmd = fmt.Sprintf(
			"Get-CimInstance %s -Filter '%s' | Select-Object %s | ConvertTo-Csv -NoTypeInformation",
			className, filter, fields)
	} else {
		psCmd = fmt.Sprintf(
			"Get-CimInstance %s | Select-Object %s | ConvertTo-Csv -NoTypeInformation",
			className, fields)
	}

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", psCmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("powershell CIM %s: %w (stderr: %s)", className, err, stderr.String())
	}

	return parseWMICCSV(stdout.Bytes())
}

// parseWMICCSV parses wmic /format:csv output.
// The output has a blank line, then "Node,Field1,Field2,...", then data rows.
func parseWMICCSV(data []byte) ([]map[string]string, error) {
	// Remove BOM and normalize line endings.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))

	// Skip leading blank lines.
	lines := strings.Split(string(data), "\n")
	var nonEmpty []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			nonEmpty = append(nonEmpty, trimmed)
		}
	}

	if len(nonEmpty) < 2 {
		return nil, nil
	}

	reader := csv.NewReader(strings.NewReader(strings.Join(nonEmpty, "\n")))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv headers: %w", err)
	}

	// Trim whitespace from headers.
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	var results []map[string]string
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			continue
		}

		row := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(record) {
				row[h] = strings.TrimSpace(record[i])
			}
		}
		results = append(results, row)
	}

	return results, nil
}

func classifyDiskType(mediaType, model string) string {
	mt := strings.ToLower(mediaType)
	mdl := strings.ToLower(model)

	if strings.Contains(mdl, "nvme") {
		return "NVMe"
	}
	if strings.Contains(mdl, "ssd") || strings.Contains(mt, "ssd") || strings.Contains(mt, "solid") {
		return "SSD"
	}
	if strings.Contains(mt, "fixed") || strings.Contains(mt, "hard") {
		return "HDD"
	}
	return "Unknown"
}

func classifyNICType(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "wi-fi") || strings.Contains(n, "wifi") || strings.Contains(n, "wireless"):
		return "wifi"
	case strings.Contains(n, "virtual") || strings.Contains(n, "hyper-v") || strings.Contains(n, "vmware"):
		return "virtual"
	default:
		return "ethernet"
	}
}
