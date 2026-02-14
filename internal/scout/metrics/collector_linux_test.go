//go:build !windows

package metrics

import (
	"testing"
)

func TestParseProcStat(t *testing.T) {
	// Fixture from a real /proc/stat.
	content := `cpu  10132153 290696 3084719 46828483 16683 0 25195 0 0 0
cpu0  1393280 32966 572056 13343292 6130 0 17875 0 0 0
cpu1  1335089 34612 543823 11287525 1641 0 3580 0 0 0
cpu2  3714081 137036 979346 7259327 8372 0 2764 0 0 0
cpu3  3689703 86082 989494 14938339 540 0 976 0 0 0
`

	tests := []struct {
		name        string
		content     string
		wantIdle    uint64
		wantTotal   uint64
		wantErr     bool
	}{
		{
			name:    "normal proc stat",
			content: content,
			// idle=46828483, iowait=16683, so idle = 46828483+16683 = 46845166
			// total = 10132153+290696+3084719+46828483+16683+0+25195+0+0+0 = 60377929
			wantIdle:  46845166,
			wantTotal: 60377929,
		},
		{
			name:    "minimal fields",
			content: "cpu  100 0 50 350\n",
			// idle=350 (field 4), no iowait field
			wantIdle:  350,
			wantTotal: 500,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
		{
			name:    "no cpu line",
			content: "processes 12345\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idle, total, err := parseProcStat(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if idle != tt.wantIdle {
				t.Errorf("idle: got %d, want %d", idle, tt.wantIdle)
			}
			if total != tt.wantTotal {
				t.Errorf("total: got %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestParseMeminfo(t *testing.T) {
	content := `MemTotal:       16384000 kB
MemFree:         2048000 kB
MemAvailable:    8192000 kB
Buffers:          512000 kB
Cached:          4096000 kB
SwapCached:            0 kB
`

	tests := []struct {
		name        string
		content     string
		wantPercent float64
		wantUsed    float64
		wantTotal   float64
		wantErr     bool
	}{
		{
			name:    "normal meminfo",
			content: content,
			// total = 16384000 kB = 16777216000 bytes
			// available = 8192000 kB = 8388608000 bytes
			// used = (16384000 - 8192000) * 1024 = 8388608000 bytes
			// percent = 8388608000 / 16777216000 * 100 = 50%
			wantPercent: 50.0,
			wantUsed:    8388608000,
			wantTotal:   16777216000,
		},
		{
			name: "no MemAvailable (old kernel fallback)",
			content: `MemTotal:       16384000 kB
MemFree:         2048000 kB
Buffers:          512000 kB
Cached:          4096000 kB
`,
			// available = MemFree + Buffers + Cached = 6656000 kB
			// used = (16384000 - 6656000) * 1024 = 9961472000
			// total = 16384000 * 1024 = 16777216000
			// percent = 9961472000 / 16777216000 * 100 ~= 59.375
			wantPercent: 59.375,
			wantUsed:    9961472000,
			wantTotal:   16777216000,
		},
		{
			name:    "empty content",
			content: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct, used, total, err := parseMeminfo(tt.content)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pct != tt.wantPercent {
				t.Errorf("percent: got %f, want %f", pct, tt.wantPercent)
			}
			if used != tt.wantUsed {
				t.Errorf("used: got %f, want %f", used, tt.wantUsed)
			}
			if total != tt.wantTotal {
				t.Errorf("total: got %f, want %f", total, tt.wantTotal)
			}
		})
	}
}

func TestParseProcNetDev(t *testing.T) {
	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000000   10000    0    0    0     0          0         0  1000000   10000    0    0    0     0       0          0
  eth0: 5000000   30000    0    0    0     0          0         0  2000000   20000    0    0    0     0       0          0
wlan0: 3000000   15000    0    0    0     0          0         0  1500000   12000    0    0    0     0       0          0
`

	networks, err := parseProcNetDev(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(networks) != 2 {
		t.Fatalf("expected 2 interfaces (lo excluded), got %d", len(networks))
	}

	// Check eth0.
	eth0 := networks[0]
	if eth0.InterfaceName != "eth0" {
		t.Errorf("expected eth0, got %s", eth0.InterfaceName)
	}
	if eth0.BytesRecv != 5000000 {
		t.Errorf("eth0 BytesRecv: got %f, want 5000000", eth0.BytesRecv)
	}
	if eth0.BytesSent != 2000000 {
		t.Errorf("eth0 BytesSent: got %f, want 2000000", eth0.BytesSent)
	}

	// Check wlan0.
	wlan0 := networks[1]
	if wlan0.InterfaceName != "wlan0" {
		t.Errorf("expected wlan0, got %s", wlan0.InterfaceName)
	}
	if wlan0.BytesRecv != 3000000 {
		t.Errorf("wlan0 BytesRecv: got %f, want 3000000", wlan0.BytesRecv)
	}
	if wlan0.BytesSent != 1500000 {
		t.Errorf("wlan0 BytesSent: got %f, want 1500000", wlan0.BytesSent)
	}
}

func TestParseProcNetDev_Empty(t *testing.T) {
	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
`

	networks, err := parseProcNetDev(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(networks) != 0 {
		t.Errorf("expected 0 interfaces, got %d", len(networks))
	}
}
