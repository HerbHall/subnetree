package recon

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

// ARPReader reads the system ARP table to get IP-to-MAC mappings.
type ARPReader struct {
	logger *zap.Logger
}

// NewARPReader creates a new ARP table reader.
func NewARPReader(logger *zap.Logger) *ARPReader {
	return &ARPReader{logger: logger}
}

// ReadTable returns a map of IP address to MAC address from the system ARP cache.
// Returns an empty map (not an error) if ARP reading is unavailable.
func (r *ARPReader) ReadTable(ctx context.Context) map[string]string {
	switch runtime.GOOS {
	case "linux":
		return r.readLinuxARP(ctx)
	case "windows":
		return r.readWindowsARP(ctx)
	case "darwin":
		return r.readDarwinARP(ctx)
	default:
		r.logger.Warn("ARP table reading not supported on this platform",
			zap.String("os", runtime.GOOS))
		return map[string]string{}
	}
}

// readLinuxARP parses /proc/net/arp.
// Format: IP address HW type Flags HW address Mask Device
func (r *ARPReader) readLinuxARP(_ context.Context) map[string]string {
	out, err := exec.Command("cat", "/proc/net/arp").Output()
	if err != nil {
		r.logger.Debug("failed to read /proc/net/arp", zap.Error(err))
		return map[string]string{}
	}

	table := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	// Skip header line.
	scanner.Scan()
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		ip := fields[0]
		mac := strings.ToUpper(fields[3])
		// Skip incomplete entries (00:00:00:00:00:00).
		if mac == "00:00:00:00:00:00" {
			continue
		}
		table[ip] = mac
	}
	return table
}

// readWindowsARP parses `arp -a` output on Windows.
// Format: Internet Address Physical Address Type
func (r *ARPReader) readWindowsARP(ctx context.Context) map[string]string {
	cmd := exec.CommandContext(ctx, "arp", "-a")
	out, err := cmd.Output()
	if err != nil {
		r.logger.Debug("failed to run arp -a", zap.Error(err))
		return map[string]string{}
	}

	table := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Windows arp -a lines look like: 192.168.1.1 aa-bb-cc-dd-ee-ff dynamic
		ip := fields[0]
		// Validate IP-like format (starts with digit).
		if ip == "" || ip[0] < '0' || ip[0] > '9' {
			continue
		}
		mac := strings.ToUpper(strings.ReplaceAll(fields[1], "-", ":"))
		// Skip broadcast and incomplete.
		if mac == "FF:FF:FF:FF:FF:FF" || mac == "00:00:00:00:00:00" {
			continue
		}
		table[ip] = mac
	}
	return table
}

// readDarwinARP parses `arp -a` output on macOS.
// Format: hostname (ip) at mac on iface [...]
func (r *ARPReader) readDarwinARP(ctx context.Context) map[string]string {
	cmd := exec.CommandContext(ctx, "arp", "-a")
	out, err := cmd.Output()
	if err != nil {
		r.logger.Debug("failed to run arp -a", zap.Error(err))
		return map[string]string{}
	}

	table := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: hostname (ip) at mac on iface
		parenStart := strings.Index(line, "(")
		parenEnd := strings.Index(line, ")")
		if parenStart < 0 || parenEnd < 0 || parenEnd <= parenStart {
			continue
		}
		ip := line[parenStart+1 : parenEnd]

		atIdx := strings.Index(line[parenEnd:], " at ")
		if atIdx < 0 {
			continue
		}
		rest := line[parenEnd+atIdx+4:]
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		mac := strings.ToUpper(fields[0])
		if mac == "(INCOMPLETE)" || mac == "FF:FF:FF:FF:FF:FF" {
			continue
		}
		table[ip] = mac
	}
	return table
}

// ParseARPOutput parses platform-specific ARP output. Exported for testing.
func ParseARPOutput(output, platform string) map[string]string {
	switch platform {
	case "linux":
		return parseLinuxARPOutput(output)
	case "windows":
		return parseWindowsARPOutput(output)
	case "darwin":
		return parseDarwinARPOutput(output)
	default:
		return map[string]string{}
	}
}

func parseLinuxARPOutput(output string) map[string]string {
	table := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Scan() // Skip header.
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		mac := strings.ToUpper(fields[3])
		if mac == "00:00:00:00:00:00" {
			continue
		}
		table[fields[0]] = mac
	}
	return table
}

func parseWindowsARPOutput(output string) map[string]string {
	table := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		ip := fields[0]
		if ip == "" || ip[0] < '0' || ip[0] > '9' {
			continue
		}
		mac := strings.ToUpper(strings.ReplaceAll(fields[1], "-", ":"))
		if mac == "FF:FF:FF:FF:FF:FF" || mac == "00:00:00:00:00:00" {
			continue
		}
		table[ip] = mac
	}
	return table
}

func parseDarwinARPOutput(output string) map[string]string {
	table := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parenStart := strings.Index(line, "(")
		parenEnd := strings.Index(line, ")")
		if parenStart < 0 || parenEnd < 0 || parenEnd <= parenStart {
			continue
		}
		ip := line[parenStart+1 : parenEnd]

		atIdx := strings.Index(line[parenEnd:], " at ")
		if atIdx < 0 {
			continue
		}
		rest := line[parenEnd+atIdx+4:]
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		mac := strings.ToUpper(fields[0])
		if mac == "(INCOMPLETE)" || mac == "FF:FF:FF:FF:FF:FF" {
			continue
		}
		table[ip] = mac
	}
	return table
}
