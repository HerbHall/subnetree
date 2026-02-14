//go:build !windows

package profiler

import (
	"testing"
)

func TestParseCPUInfo(t *testing.T) {
	// Fixture: dual-core hyperthreaded CPU (2 physical cores, 4 threads).
	content := `processor	: 0
vendor_id	: GenuineIntel
cpu family	: 6
model		: 142
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
physical id	: 0
core id		: 0

processor	: 1
vendor_id	: GenuineIntel
cpu family	: 6
model		: 142
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
physical id	: 0
core id		: 1

processor	: 2
vendor_id	: GenuineIntel
cpu family	: 6
model		: 142
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
physical id	: 0
core id		: 0

processor	: 3
vendor_id	: GenuineIntel
cpu family	: 6
model		: 142
model name	: Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz
physical id	: 0
core id		: 1
`

	tests := []struct {
		name        string
		content     string
		logical     int32
		wantModel   string
		wantPhys    int32
		wantLogical int32
	}{
		{
			name:        "hyperthreaded dual-core",
			content:     content,
			logical:     4,
			wantModel:   "Intel(R) Core(TM) i7-8550U CPU @ 1.80GHz",
			wantPhys:    2,
			wantLogical: 4,
		},
		{
			name: "single processor no physical id",
			content: `processor	: 0
model name	: ARM Cortex-A72
`,
			logical:     1,
			wantModel:   "ARM Cortex-A72",
			wantPhys:    1, // Falls back to logical count.
			wantLogical: 1,
		},
		{
			name:        "empty content",
			content:     "",
			logical:     4,
			wantModel:   "",
			wantPhys:    4, // Falls back to logical count.
			wantLogical: 4,
		},
		{
			name: "dual socket",
			content: `processor	: 0
model name	: Intel Xeon E5-2680 v4
physical id	: 0
core id		: 0

processor	: 1
model name	: Intel Xeon E5-2680 v4
physical id	: 0
core id		: 1

processor	: 2
model name	: Intel Xeon E5-2680 v4
physical id	: 1
core id		: 0

processor	: 3
model name	: Intel Xeon E5-2680 v4
physical id	: 1
core id		: 1
`,
			logical:     4,
			wantModel:   "Intel Xeon E5-2680 v4",
			wantPhys:    4, // 2 sockets x 2 cores = 4 unique (physID, coreID) pairs.
			wantLogical: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, phys, logical := parseCPUInfo(tt.content, tt.logical)
			if model != tt.wantModel {
				t.Errorf("model: got %q, want %q", model, tt.wantModel)
			}
			if phys != tt.wantPhys {
				t.Errorf("physCores: got %d, want %d", phys, tt.wantPhys)
			}
			if logical != tt.wantLogical {
				t.Errorf("logicalCPUs: got %d, want %d", logical, tt.wantLogical)
			}
		})
	}
}

func TestClassifyLinuxDiskType(t *testing.T) {
	tests := []struct {
		name     string
		devName  string
		wantType string
	}{
		{name: "nvme device", devName: "nvme0n1", wantType: "NVMe"},
		{name: "nvme partition", devName: "nvme1n1", wantType: "NVMe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For NVMe, the function returns early based on name alone
			// without reading /sys/block files.
			got := classifyLinuxDiskType(tt.devName, "/nonexistent")
			if got != tt.wantType {
				t.Errorf("classifyLinuxDiskType(%q): got %q, want %q", tt.devName, got, tt.wantType)
			}
		})
	}
}

func TestClassifyLinuxNICType(t *testing.T) {
	tests := []struct {
		name       string
		ifName     string
		kernelType string
		wantType   string
	}{
		{name: "wifi wl prefix", ifName: "wlp3s0", kernelType: "1", wantType: "wifi"},
		{name: "wifi wlan prefix", ifName: "wlan0", kernelType: "1", wantType: "wifi"},
		{name: "virtual veth", ifName: "veth123abc", kernelType: "1", wantType: "virtual"},
		{name: "virtual docker", ifName: "docker0", kernelType: "1", wantType: "virtual"},
		{name: "virtual bridge", ifName: "br-abc123", kernelType: "1", wantType: "virtual"},
		{name: "ethernet", ifName: "eth0", kernelType: "1", wantType: "ethernet"},
		{name: "ethernet eno", ifName: "eno1", kernelType: "1", wantType: "ethernet"},
		{name: "wifi kernel type", ifName: "ath0", kernelType: "801", wantType: "wifi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLinuxNICType(tt.ifName, tt.kernelType)
			if got != tt.wantType {
				t.Errorf("classifyLinuxNICType(%q, %q): got %q, want %q", tt.ifName, tt.kernelType, got, tt.wantType)
			}
		})
	}
}
