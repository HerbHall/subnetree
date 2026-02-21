package recon

import (
	"context"
	"fmt"
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
)

// ---------------------------------------------------------------------------
// Hardware profile store tests
// ---------------------------------------------------------------------------

func TestUpsertDeviceHardware_CreateNew(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create a parent device first (FK constraint).
	device := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	hw := &models.DeviceHardware{
		DeviceID:           device.ID,
		Hostname:           "web-server-01",
		OSName:             "Ubuntu 24.04",
		OSVersion:          "24.04",
		OSArch:             "amd64",
		CPUModel:           "Intel Core i7-12700K",
		CPUCores:           12,
		CPUThreads:         20,
		RAMTotalMB:         32768,
		PlatformType:       "baremetal",
		SystemManufacturer: "Dell Inc.",
		CollectionSource:   "scout-linux",
	}
	if err := s.UpsertDeviceHardware(ctx, hw); err != nil {
		t.Fatalf("UpsertDeviceHardware: %v", err)
	}

	got, err := s.GetDeviceHardware(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceHardware: %v", err)
	}
	if got == nil {
		t.Fatal("expected hardware, got nil")
	}
	if got.Hostname != "web-server-01" {
		t.Errorf("Hostname = %q, want web-server-01", got.Hostname)
	}
	if got.CPUCores != 12 {
		t.Errorf("CPUCores = %d, want 12", got.CPUCores)
	}
	if got.RAMTotalMB != 32768 {
		t.Errorf("RAMTotalMB = %d, want 32768", got.RAMTotalMB)
	}
	if got.CollectionSource != "scout-linux" {
		t.Errorf("CollectionSource = %q, want scout-linux", got.CollectionSource)
	}
	if got.CollectedAt == nil {
		t.Error("expected non-nil CollectedAt")
	}
	if got.UpdatedAt == nil {
		t.Error("expected non-nil UpdatedAt")
	}
}

func TestUpsertDeviceHardware_UpdateExisting(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.2"},
		MACAddress:      "AA:BB:CC:DD:EE:02",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	// First insert.
	hw := &models.DeviceHardware{
		DeviceID:         device.ID,
		OSName:           "Ubuntu 22.04",
		CPUCores:         8,
		RAMTotalMB:       16384,
		CollectionSource: "scout-linux",
	}
	if err := s.UpsertDeviceHardware(ctx, hw); err != nil {
		t.Fatalf("first UpsertDeviceHardware: %v", err)
	}

	// Update with new data.
	hw2 := &models.DeviceHardware{
		DeviceID:         device.ID,
		OSName:           "Ubuntu 24.04",
		CPUCores:         12,
		RAMTotalMB:       32768,
		CollectionSource: "scout-linux",
	}
	if err := s.UpsertDeviceHardware(ctx, hw2); err != nil {
		t.Fatalf("second UpsertDeviceHardware: %v", err)
	}

	got, err := s.GetDeviceHardware(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceHardware: %v", err)
	}
	if got.OSName != "Ubuntu 24.04" {
		t.Errorf("OSName = %q, want Ubuntu 24.04", got.OSName)
	}
	if got.CPUCores != 12 {
		t.Errorf("CPUCores = %d, want 12", got.CPUCores)
	}
	if got.RAMTotalMB != 32768 {
		t.Errorf("RAMTotalMB = %d, want 32768", got.RAMTotalMB)
	}
}

func TestUpsertDeviceHardware_ManualOverridePreserved(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.3"},
		MACAddress:      "AA:BB:CC:DD:EE:03",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	// Insert manual hardware data.
	manual := &models.DeviceHardware{
		DeviceID:         device.ID,
		Hostname:         "manual-host",
		OSName:           "FreeBSD 14",
		CPUCores:         4,
		RAMTotalMB:       8192,
		PlatformType:     "baremetal",
		CollectionSource: "manual",
	}
	if err := s.UpsertDeviceHardware(ctx, manual); err != nil {
		t.Fatalf("insert manual hardware: %v", err)
	}

	// Auto-collected data should not overwrite manual fields.
	auto := &models.DeviceHardware{
		DeviceID:         device.ID,
		Hostname:         "auto-host",
		OSName:           "Linux 6.5",
		CPUCores:         16,
		RAMTotalMB:       65536,
		PlatformType:     "vm",
		Kernel:           "6.5.0-44-generic",
		CollectionSource: "scout-linux",
	}
	if err := s.UpsertDeviceHardware(ctx, auto); err != nil {
		t.Fatalf("upsert auto hardware: %v", err)
	}

	got, err := s.GetDeviceHardware(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceHardware: %v", err)
	}

	// Manual fields should be preserved.
	if got.Hostname != "manual-host" {
		t.Errorf("Hostname = %q, want manual-host (manual override)", got.Hostname)
	}
	if got.OSName != "FreeBSD 14" {
		t.Errorf("OSName = %q, want FreeBSD 14 (manual override)", got.OSName)
	}
	if got.CPUCores != 4 {
		t.Errorf("CPUCores = %d, want 4 (manual override)", got.CPUCores)
	}
	if got.RAMTotalMB != 8192 {
		t.Errorf("RAMTotalMB = %d, want 8192 (manual override)", got.RAMTotalMB)
	}
	if got.PlatformType != "baremetal" {
		t.Errorf("PlatformType = %q, want baremetal (manual override)", got.PlatformType)
	}
	// Empty field should be filled by auto data.
	if got.Kernel != "6.5.0-44-generic" {
		t.Errorf("Kernel = %q, want 6.5.0-44-generic (auto-filled empty field)", got.Kernel)
	}
	// Source should remain manual.
	if got.CollectionSource != "manual" {
		t.Errorf("CollectionSource = %q, want manual", got.CollectionSource)
	}
}

func TestUpsertDeviceHardware_ManualOverridesManual(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.4"},
		MACAddress:      "AA:BB:CC:DD:EE:04",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	// First manual insert.
	hw1 := &models.DeviceHardware{
		DeviceID:         device.ID,
		Hostname:         "manual-v1",
		RAMTotalMB:       8192,
		CollectionSource: "manual",
	}
	if err := s.UpsertDeviceHardware(ctx, hw1); err != nil {
		t.Fatalf("first manual insert: %v", err)
	}

	// Second manual insert should fully replace.
	hw2 := &models.DeviceHardware{
		DeviceID:         device.ID,
		Hostname:         "manual-v2",
		RAMTotalMB:       16384,
		CollectionSource: "manual",
	}
	if err := s.UpsertDeviceHardware(ctx, hw2); err != nil {
		t.Fatalf("second manual insert: %v", err)
	}

	got, err := s.GetDeviceHardware(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceHardware: %v", err)
	}
	if got.Hostname != "manual-v2" {
		t.Errorf("Hostname = %q, want manual-v2", got.Hostname)
	}
	if got.RAMTotalMB != 16384 {
		t.Errorf("RAMTotalMB = %d, want 16384", got.RAMTotalMB)
	}
}

func TestGetDeviceHardware_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetDeviceHardware(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing hardware, got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Storage tests
// ---------------------------------------------------------------------------

func TestUpsertDeviceStorage_Create(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.10"},
		MACAddress:      "AA:BB:CC:DD:10:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	disks := []models.DeviceStorage{
		{
			DeviceID:         device.ID,
			Name:             "Samsung 990 Pro",
			DiskType:         "nvme",
			Interface:        "pcie4",
			CapacityGB:       2000,
			Model:            "Samsung SSD 990 PRO",
			Role:             "boot",
			CollectionSource: "scout-linux",
		},
		{
			DeviceID:         device.ID,
			Name:             "WD Red 8TB",
			DiskType:         "hdd",
			Interface:        "sata",
			CapacityGB:       8000,
			Model:            "WDC WD80EFAX",
			Role:             "data",
			CollectionSource: "scout-linux",
		},
	}
	if err := s.UpsertDeviceStorage(ctx, device.ID, disks); err != nil {
		t.Fatalf("UpsertDeviceStorage: %v", err)
	}

	got, err := s.GetDeviceStorage(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceStorage: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("storage count = %d, want 2", len(got))
	}
}

func TestUpsertDeviceStorage_ReplaceNonManual(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.11"},
		MACAddress:      "AA:BB:CC:DD:10:02",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	// Insert initial auto-collected disks.
	initial := []models.DeviceStorage{
		{DeviceID: device.ID, Name: "Old Disk", DiskType: "hdd", CollectionSource: "scout-linux"},
	}
	if err := s.UpsertDeviceStorage(ctx, device.ID, initial); err != nil {
		t.Fatalf("initial UpsertDeviceStorage: %v", err)
	}

	// Replace with new auto-collected disks.
	replacement := []models.DeviceStorage{
		{DeviceID: device.ID, Name: "New Disk 1", DiskType: "nvme", CollectionSource: "scout-linux"},
		{DeviceID: device.ID, Name: "New Disk 2", DiskType: "ssd", CollectionSource: "scout-linux"},
	}
	if err := s.UpsertDeviceStorage(ctx, device.ID, replacement); err != nil {
		t.Fatalf("replacement UpsertDeviceStorage: %v", err)
	}

	got, err := s.GetDeviceStorage(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceStorage: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("storage count = %d, want 2 (old auto disk should be gone)", len(got))
	}
	names := map[string]bool{}
	for _, d := range got {
		names[d.Name] = true
	}
	if names["Old Disk"] {
		t.Error("Old Disk should have been replaced")
	}
	if !names["New Disk 1"] || !names["New Disk 2"] {
		t.Errorf("expected New Disk 1 and New Disk 2, got %v", names)
	}
}

func TestUpsertDeviceStorage_PreservesManual(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.12"},
		MACAddress:      "AA:BB:CC:DD:10:03",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	// Insert a manual disk.
	manual := []models.DeviceStorage{
		{DeviceID: device.ID, Name: "Manual Disk", DiskType: "ssd", CollectionSource: "manual"},
	}
	if err := s.UpsertDeviceStorage(ctx, device.ID, manual); err != nil {
		t.Fatalf("manual UpsertDeviceStorage: %v", err)
	}

	// Insert auto-collected disks (should not remove manual).
	auto := []models.DeviceStorage{
		{DeviceID: device.ID, Name: "Auto Disk", DiskType: "nvme", CollectionSource: "scout-linux"},
	}
	if err := s.UpsertDeviceStorage(ctx, device.ID, auto); err != nil {
		t.Fatalf("auto UpsertDeviceStorage: %v", err)
	}

	got, err := s.GetDeviceStorage(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceStorage: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("storage count = %d, want 2 (manual + auto)", len(got))
	}
}

func TestGetDeviceStorage_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetDeviceStorage(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}

// ---------------------------------------------------------------------------
// GPU tests
// ---------------------------------------------------------------------------

func TestUpsertDeviceGPU_Create(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.20"},
		MACAddress:      "AA:BB:CC:DD:20:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	gpus := []models.DeviceGPU{
		{
			DeviceID:         device.ID,
			Model:            "NVIDIA RTX 3090",
			Vendor:           "nvidia",
			VRAMMB:           24576,
			DriverVersion:    "535.183.01",
			CollectionSource: "scout-linux",
		},
	}
	if err := s.UpsertDeviceGPU(ctx, device.ID, gpus); err != nil {
		t.Fatalf("UpsertDeviceGPU: %v", err)
	}

	got, err := s.GetDeviceGPU(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceGPU: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("gpu count = %d, want 1", len(got))
	}
	if got[0].Model != "NVIDIA RTX 3090" {
		t.Errorf("Model = %q, want NVIDIA RTX 3090", got[0].Model)
	}
	if got[0].VRAMMB != 24576 {
		t.Errorf("VRAMMB = %d, want 24576", got[0].VRAMMB)
	}
}

func TestUpsertDeviceGPU_Replace(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.21"},
		MACAddress:      "AA:BB:CC:DD:20:02",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	// Insert initial GPU.
	initial := []models.DeviceGPU{
		{DeviceID: device.ID, Model: "Old GPU", Vendor: "amd", CollectionSource: "scout-linux"},
	}
	if err := s.UpsertDeviceGPU(ctx, device.ID, initial); err != nil {
		t.Fatalf("initial UpsertDeviceGPU: %v", err)
	}

	// Replace with new GPU.
	replacement := []models.DeviceGPU{
		{DeviceID: device.ID, Model: "New GPU", Vendor: "nvidia", CollectionSource: "scout-linux"},
	}
	if err := s.UpsertDeviceGPU(ctx, device.ID, replacement); err != nil {
		t.Fatalf("replacement UpsertDeviceGPU: %v", err)
	}

	got, err := s.GetDeviceGPU(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceGPU: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("gpu count = %d, want 1", len(got))
	}
	if got[0].Model != "New GPU" {
		t.Errorf("Model = %q, want New GPU", got[0].Model)
	}
}

func TestGetDeviceGPU_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetDeviceGPU(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}

// ---------------------------------------------------------------------------
// Services tests
// ---------------------------------------------------------------------------

func TestUpsertDeviceServices_Create(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.30"},
		MACAddress:      "AA:BB:CC:DD:30:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	svcs := []models.DeviceService{
		{
			DeviceID:         device.ID,
			Name:             "plex",
			ServiceType:      "docker",
			Port:             32400,
			URL:              "http://10.0.0.30:32400",
			Version:          "1.40.0",
			Status:           "running",
			CollectionSource: "scout-linux",
		},
		{
			DeviceID:         device.ID,
			Name:             "nginx",
			ServiceType:      "systemd",
			Port:             443,
			Status:           "running",
			CollectionSource: "scout-linux",
		},
	}
	if err := s.UpsertDeviceServices(ctx, device.ID, svcs); err != nil {
		t.Fatalf("UpsertDeviceServices: %v", err)
	}

	got, err := s.GetDeviceServices(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceServices: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("service count = %d, want 2", len(got))
	}
}

func TestUpsertDeviceServices_Replace(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.31"},
		MACAddress:      "AA:BB:CC:DD:30:02",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	initial := []models.DeviceService{
		{DeviceID: device.ID, Name: "old-svc", CollectionSource: "scout-linux"},
	}
	if err := s.UpsertDeviceServices(ctx, device.ID, initial); err != nil {
		t.Fatalf("initial UpsertDeviceServices: %v", err)
	}

	replacement := []models.DeviceService{
		{DeviceID: device.ID, Name: "new-svc-1", CollectionSource: "scout-linux"},
		{DeviceID: device.ID, Name: "new-svc-2", CollectionSource: "scout-linux"},
	}
	if err := s.UpsertDeviceServices(ctx, device.ID, replacement); err != nil {
		t.Fatalf("replacement UpsertDeviceServices: %v", err)
	}

	got, err := s.GetDeviceServices(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceServices: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("service count = %d, want 2", len(got))
	}
	names := map[string]bool{}
	for _, svc := range got {
		names[svc.Name] = true
	}
	if names["old-svc"] {
		t.Error("old-svc should have been replaced")
	}
}

func TestGetDeviceServices_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetDeviceServices(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}

// ---------------------------------------------------------------------------
// Hardware summary tests
// ---------------------------------------------------------------------------

func TestGetHardwareSummary_Empty(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	summary, err := s.GetHardwareSummary(ctx)
	if err != nil {
		t.Fatalf("GetHardwareSummary: %v", err)
	}
	if summary.TotalWithHardware != 0 {
		t.Errorf("TotalWithHardware = %d, want 0", summary.TotalWithHardware)
	}
	if summary.TotalRAMMB != 0 {
		t.Errorf("TotalRAMMB = %d, want 0", summary.TotalRAMMB)
	}
	if summary.TotalStorageGB != 0 {
		t.Errorf("TotalStorageGB = %d, want 0", summary.TotalStorageGB)
	}
	if summary.TotalGPUs != 0 {
		t.Errorf("TotalGPUs = %d, want 0", summary.TotalGPUs)
	}
}

func TestGetHardwareSummary_WithData(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create two devices with hardware profiles.
	d1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:DD:50:01",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	d2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:DD:50:02",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, d1); err != nil {
		t.Fatalf("create d1: %v", err)
	}
	if _, err := s.UpsertDevice(ctx, d2); err != nil {
		t.Fatalf("create d2: %v", err)
	}

	// Hardware profiles.
	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d1.ID, OSName: "Ubuntu 24.04", CPUModel: "Intel i7", RAMTotalMB: 32768,
		PlatformType: "baremetal", CollectionSource: "scout-linux",
	}); err != nil {
		t.Fatalf("hardware d1: %v", err)
	}
	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d2.ID, OSName: "Ubuntu 24.04", CPUModel: "AMD Ryzen 9", RAMTotalMB: 65536,
		PlatformType: "baremetal", CollectionSource: "scout-linux",
	}); err != nil {
		t.Fatalf("hardware d2: %v", err)
	}

	// Storage.
	if err := s.UpsertDeviceStorage(ctx, d1.ID, []models.DeviceStorage{
		{DeviceID: d1.ID, CapacityGB: 2000, CollectionSource: "scout-linux"},
	}); err != nil {
		t.Fatalf("storage d1: %v", err)
	}
	if err := s.UpsertDeviceStorage(ctx, d2.ID, []models.DeviceStorage{
		{DeviceID: d2.ID, CapacityGB: 4000, CollectionSource: "scout-linux"},
		{DeviceID: d2.ID, CapacityGB: 8000, CollectionSource: "scout-linux"},
	}); err != nil {
		t.Fatalf("storage d2: %v", err)
	}

	// GPU on d1 only.
	if err := s.UpsertDeviceGPU(ctx, d1.ID, []models.DeviceGPU{
		{DeviceID: d1.ID, Model: "RTX 3090", Vendor: "nvidia", VRAMMB: 24576, CollectionSource: "scout-linux"},
	}); err != nil {
		t.Fatalf("gpu d1: %v", err)
	}

	summary, err := s.GetHardwareSummary(ctx)
	if err != nil {
		t.Fatalf("GetHardwareSummary: %v", err)
	}
	if summary.TotalWithHardware != 2 {
		t.Errorf("TotalWithHardware = %d, want 2", summary.TotalWithHardware)
	}
	if summary.TotalRAMMB != 98304 { // 32768 + 65536
		t.Errorf("TotalRAMMB = %d, want 98304", summary.TotalRAMMB)
	}
	if summary.TotalStorageGB != 14000 { // 2000 + 4000 + 8000
		t.Errorf("TotalStorageGB = %d, want 14000", summary.TotalStorageGB)
	}
	if summary.TotalGPUs != 1 {
		t.Errorf("TotalGPUs = %d, want 1", summary.TotalGPUs)
	}
	if summary.ByOS["Ubuntu 24.04"] != 2 {
		t.Errorf("ByOS[Ubuntu 24.04] = %d, want 2", summary.ByOS["Ubuntu 24.04"])
	}
	if summary.ByCPUModel["Intel i7"] != 1 {
		t.Errorf("ByCPUModel[Intel i7] = %d, want 1", summary.ByCPUModel["Intel i7"])
	}
	if summary.ByPlatformType["baremetal"] != 2 {
		t.Errorf("ByPlatformType[baremetal] = %d, want 2", summary.ByPlatformType["baremetal"])
	}
	if summary.ByGPUVendor["nvidia"] != 1 {
		t.Errorf("ByGPUVendor[nvidia] = %d, want 1", summary.ByGPUVendor["nvidia"])
	}
}

// ---------------------------------------------------------------------------
// Query devices by hardware tests
// ---------------------------------------------------------------------------

func TestQueryDevicesByHardware_FilterByRAM(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:DD:60:01",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	d2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:DD:60:02",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, d1); err != nil {
		t.Fatalf("create d1: %v", err)
	}
	if _, err := s.UpsertDevice(ctx, d2); err != nil {
		t.Fatalf("create d2: %v", err)
	}

	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d1.ID, RAMTotalMB: 8192, CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("hw d1: %v", err)
	}
	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d2.ID, RAMTotalMB: 65536, CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("hw d2: %v", err)
	}

	// Query for devices with >= 32768 MB RAM.
	devices, total, err := s.QueryDevicesByHardware(ctx, models.HardwareQuery{MinRAMMB: 32768})
	if err != nil {
		t.Fatalf("QueryDevicesByHardware: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 {
		t.Fatalf("count = %d, want 1", len(devices))
	}
	if devices[0].ID != d2.ID {
		t.Errorf("expected d2, got %s", devices[0].ID)
	}
}

func TestQueryDevicesByHardware_FilterByOSAndCPU(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:DD:61:01",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	d2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:DD:61:02",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, d1); err != nil {
		t.Fatalf("create d1: %v", err)
	}
	if _, err := s.UpsertDevice(ctx, d2); err != nil {
		t.Fatalf("create d2: %v", err)
	}

	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d1.ID, OSName: "Ubuntu 24.04", CPUModel: "Intel Core i7", CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("hw d1: %v", err)
	}
	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d2.ID, OSName: "Windows 11", CPUModel: "AMD Ryzen 9", CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("hw d2: %v", err)
	}

	// Filter by OS.
	devices, total, err := s.QueryDevicesByHardware(ctx, models.HardwareQuery{OSName: "Ubuntu"})
	if err != nil {
		t.Fatalf("QueryDevicesByHardware OS: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 || devices[0].ID != d1.ID {
		t.Errorf("expected d1 for Ubuntu filter")
	}

	// Filter by CPU.
	devices, total, err = s.QueryDevicesByHardware(ctx, models.HardwareQuery{CPUModel: "Ryzen"})
	if err != nil {
		t.Fatalf("QueryDevicesByHardware CPU: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 || devices[0].ID != d2.ID {
		t.Errorf("expected d2 for Ryzen filter")
	}
}

func TestQueryDevicesByHardware_FilterByPlatformType(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:DD:62:01",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	d2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:DD:62:02",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, d1); err != nil {
		t.Fatalf("create d1: %v", err)
	}
	if _, err := s.UpsertDevice(ctx, d2); err != nil {
		t.Fatalf("create d2: %v", err)
	}

	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d1.ID, PlatformType: "baremetal", CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("hw d1: %v", err)
	}
	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d2.ID, PlatformType: "vm", CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("hw d2: %v", err)
	}

	devices, total, err := s.QueryDevicesByHardware(ctx, models.HardwareQuery{PlatformType: "vm"})
	if err != nil {
		t.Fatalf("QueryDevicesByHardware: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 || devices[0].ID != d2.ID {
		t.Errorf("expected d2 for vm filter")
	}
}

func TestQueryDevicesByHardware_FilterByGPU(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:DD:63:01",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	d2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:DD:63:02",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, d1); err != nil {
		t.Fatalf("create d1: %v", err)
	}
	if _, err := s.UpsertDevice(ctx, d2); err != nil {
		t.Fatalf("create d2: %v", err)
	}

	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d1.ID, CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("hw d1: %v", err)
	}
	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: d2.ID, CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("hw d2: %v", err)
	}

	// Add GPU to d1 only.
	if err := s.UpsertDeviceGPU(ctx, d1.ID, []models.DeviceGPU{
		{DeviceID: d1.ID, Model: "RTX 3090", Vendor: "nvidia", CollectionSource: "scout"},
	}); err != nil {
		t.Fatalf("gpu d1: %v", err)
	}

	// Filter has_gpu = true.
	hasGPU := true
	devices, total, err := s.QueryDevicesByHardware(ctx, models.HardwareQuery{HasGPU: &hasGPU})
	if err != nil {
		t.Fatalf("QueryDevicesByHardware has_gpu=true: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 || devices[0].ID != d1.ID {
		t.Errorf("expected d1 for has_gpu=true")
	}

	// Filter has_gpu = false.
	noGPU := false
	devices, total, err = s.QueryDevicesByHardware(ctx, models.HardwareQuery{HasGPU: &noGPU})
	if err != nil {
		t.Fatalf("QueryDevicesByHardware has_gpu=false: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 || devices[0].ID != d2.ID {
		t.Errorf("expected d2 for has_gpu=false")
	}

	// Filter by GPU vendor.
	devices, total, err = s.QueryDevicesByHardware(ctx, models.HardwareQuery{GPUVendor: "nvidia"})
	if err != nil {
		t.Fatalf("QueryDevicesByHardware gpu_vendor: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 || devices[0].ID != d1.ID {
		t.Errorf("expected d1 for nvidia filter")
	}
}

func TestQueryDevicesByHardware_Pagination(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create 3 devices with hardware.
	for i := 0; i < 3; i++ {
		d := &models.Device{
			IPAddresses:     []string{fmt.Sprintf("10.0.0.%d", i+1)},
			MACAddress:      fmt.Sprintf("AA:BB:CC:DD:64:%02X", i),
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryICMP,
		}
		if _, err := s.UpsertDevice(ctx, d); err != nil {
			t.Fatalf("create device %d: %v", i, err)
		}
		if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
			DeviceID: d.ID, RAMTotalMB: (i + 1) * 8192, CollectionSource: "scout",
		}); err != nil {
			t.Fatalf("hardware %d: %v", i, err)
		}
	}

	// Page 1.
	devices, total, err := s.QueryDevicesByHardware(ctx, models.HardwareQuery{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("QueryDevicesByHardware page 1: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(devices) != 2 {
		t.Errorf("page 1 count = %d, want 2", len(devices))
	}

	// Page 2.
	devices, _, err = s.QueryDevicesByHardware(ctx, models.HardwareQuery{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("QueryDevicesByHardware page 2: %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("page 2 count = %d, want 1", len(devices))
	}
}

// ---------------------------------------------------------------------------
// Delete and cascade tests
// ---------------------------------------------------------------------------

func TestDeleteDeviceHardware(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.40"},
		MACAddress:      "AA:BB:CC:DD:40:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}
	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: device.ID, OSName: "Ubuntu", CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("insert hardware: %v", err)
	}

	if err := s.DeleteDeviceHardware(ctx, device.ID); err != nil {
		t.Fatalf("DeleteDeviceHardware: %v", err)
	}

	got, err := s.GetDeviceHardware(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceHardware after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestDeleteDeviceHardware_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	err := s.DeleteDeviceHardware(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent hardware")
	}
}

func TestCascadeDeleteDevice_RemovesAllHardware(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.50"},
		MACAddress:      "AA:BB:CC:DD:50:99",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, device); err != nil {
		t.Fatalf("create device: %v", err)
	}

	// Add hardware, storage, GPU, and services.
	if err := s.UpsertDeviceHardware(ctx, &models.DeviceHardware{
		DeviceID: device.ID, OSName: "Ubuntu", CollectionSource: "scout",
	}); err != nil {
		t.Fatalf("insert hardware: %v", err)
	}
	if err := s.UpsertDeviceStorage(ctx, device.ID, []models.DeviceStorage{
		{DeviceID: device.ID, Name: "Disk 1", CollectionSource: "scout"},
	}); err != nil {
		t.Fatalf("insert storage: %v", err)
	}
	if err := s.UpsertDeviceGPU(ctx, device.ID, []models.DeviceGPU{
		{DeviceID: device.ID, Model: "GPU 1", Vendor: "nvidia", CollectionSource: "scout"},
	}); err != nil {
		t.Fatalf("insert gpu: %v", err)
	}
	if err := s.UpsertDeviceServices(ctx, device.ID, []models.DeviceService{
		{DeviceID: device.ID, Name: "svc-1", CollectionSource: "scout"},
	}); err != nil {
		t.Fatalf("insert services: %v", err)
	}

	// Delete the parent device -- CASCADE should remove all children.
	if err := s.DeleteDevice(ctx, device.ID); err != nil {
		t.Fatalf("DeleteDevice: %v", err)
	}

	// Verify all hardware tables are empty for this device.
	hw, err := s.GetDeviceHardware(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceHardware: %v", err)
	}
	if hw != nil {
		t.Error("hardware should be cascaded")
	}

	storage, err := s.GetDeviceStorage(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceStorage: %v", err)
	}
	if len(storage) != 0 {
		t.Errorf("storage count = %d, want 0 (cascaded)", len(storage))
	}

	gpus, err := s.GetDeviceGPU(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceGPU: %v", err)
	}
	if len(gpus) != 0 {
		t.Errorf("gpu count = %d, want 0 (cascaded)", len(gpus))
	}

	svcs, err := s.GetDeviceServices(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceServices: %v", err)
	}
	if len(svcs) != 0 {
		t.Errorf("service count = %d, want 0 (cascaded)", len(svcs))
	}
}
