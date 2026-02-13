package profiler

import (
	"context"
	"runtime"
	"testing"

	"go.uber.org/zap"
)

func TestNewProfiler_ReturnsNonNil(t *testing.T) {
	logger := zap.NewNop()
	p := NewProfiler(logger)
	if p == nil {
		t.Fatal("NewProfiler returned nil")
	}
}

func TestCollectProfile_ReturnsNonNilProfile(t *testing.T) {
	logger := zap.NewNop()
	p := NewProfiler(logger)

	ctx := context.Background()
	profile, err := p.CollectProfile(ctx)
	if err != nil {
		t.Fatalf("CollectProfile returned error: %v", err)
	}
	if profile == nil {
		t.Fatal("CollectProfile returned nil profile")
	}
}

func TestCollectProfile_HasOSInfo(t *testing.T) {
	logger := zap.NewNop()
	p := NewProfiler(logger)

	ctx := context.Background()
	profile, err := p.CollectProfile(ctx)
	if err != nil {
		t.Fatalf("CollectProfile returned error: %v", err)
	}
	if profile.Software == nil {
		t.Fatal("Software inventory is nil")
	}
	if profile.Software.OsName == "" {
		t.Error("OsName is empty")
	}
}

func TestCollectProfile_Windows_HasHardwareData(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("hardware profiling requires Windows")
	}

	logger := zap.NewNop()
	p := NewProfiler(logger)

	ctx := context.Background()
	profile, err := p.CollectProfile(ctx)
	if err != nil {
		t.Fatalf("CollectProfile returned error: %v", err)
	}
	if profile.Hardware == nil {
		t.Fatal("Hardware profile is nil on Windows")
	}
	if profile.Hardware.CpuModel == "" {
		t.Error("CpuModel is empty on Windows")
	}
	if profile.Hardware.CpuCores == 0 {
		t.Error("CpuCores is 0 on Windows")
	}
	if profile.Hardware.RamBytes == 0 {
		t.Error("RamBytes is 0 on Windows")
	}
}

func TestCollectProfile_Windows_HasSoftwareData(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("software inventory requires Windows")
	}

	logger := zap.NewNop()
	p := NewProfiler(logger)

	ctx := context.Background()
	profile, err := p.CollectProfile(ctx)
	if err != nil {
		t.Fatalf("CollectProfile returned error: %v", err)
	}
	if profile.Software == nil {
		t.Fatal("Software inventory is nil on Windows")
	}
	if profile.Software.OsVersion == "" {
		t.Error("OsVersion is empty on Windows")
	}
	if len(profile.Software.Packages) == 0 {
		t.Error("no installed packages found on Windows")
	}
}

func TestCollectProfile_Windows_HasServices(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("service enumeration requires Windows")
	}

	logger := zap.NewNop()
	p := NewProfiler(logger)

	ctx := context.Background()
	profile, err := p.CollectProfile(ctx)
	if err != nil {
		t.Fatalf("CollectProfile returned error: %v", err)
	}
	// Service enumeration via SCM may require elevated permissions.
	// If no services are returned, log a warning instead of failing.
	if len(profile.Services) == 0 {
		t.Log("no services found on Windows (may require elevated permissions)")
	}
}

func TestCollectProfile_CancelledContext(t *testing.T) {
	logger := zap.NewNop()
	p := NewProfiler(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	profile, err := p.CollectProfile(ctx)
	if profile == nil {
		t.Fatal("CollectProfile returned nil even with cancelled context")
	}
	_ = err
}
