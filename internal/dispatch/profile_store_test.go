package dispatch

import (
	"context"
	"testing"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
)

func testStoreWithAgent(t *testing.T) (store *DispatchStore, agentID string) {
	t.Helper()
	s := testStore(t)
	ctx := context.Background()

	agentID = "agent-profile-001"
	if err := s.UpsertAgent(ctx, &Agent{
		ID:           agentID,
		Hostname:     "profile-host",
		Platform:     "windows/amd64",
		AgentVersion: "0.1.0",
		ProtoVersion: 1,
		Status:       "connected",
		ConfigJSON:   "{}",
	}); err != nil {
		t.Fatalf("setup agent: %v", err)
	}
	return s, agentID
}

func TestProfileStore_UpsertAndGetHardwareProfile(t *testing.T) {
	s, agentID := testStoreWithAgent(t)
	ctx := context.Background()

	hw := &scoutpb.HardwareProfile{
		CpuModel:           "Intel Core i7-12700K",
		CpuCores:           12,
		CpuThreads:         20,
		RamBytes:           34359738368,
		BiosVersion:        "1.0.0",
		SystemManufacturer: "Dell",
		SystemModel:        "OptiPlex 7090",
		SerialNumber:       "SN12345",
		Disks: []*scoutpb.DiskInfo{
			{Name: "Samsung 980 PRO", SizeBytes: 1000204886016, DiskType: "NVMe", Model: "Samsung 980 PRO"},
		},
		Gpus: []*scoutpb.GPUInfo{
			{Model: "NVIDIA GeForce RTX 3080", VramBytes: 10737418240, DriverVersion: "531.68"},
		},
		Nics: []*scoutpb.NICInfo{
			{Name: "Ethernet", SpeedMbps: 1000, MacAddress: "AA:BB:CC:DD:EE:FF", NicType: "ethernet"},
		},
	}

	if err := s.UpsertHardwareProfile(ctx, agentID, hw); err != nil {
		t.Fatalf("UpsertHardwareProfile: %v", err)
	}

	got, err := s.GetHardwareProfile(ctx, agentID)
	if err != nil {
		t.Fatalf("GetHardwareProfile: %v", err)
	}
	if got == nil {
		t.Fatal("GetHardwareProfile returned nil")
	}
	if got.CpuModel != "Intel Core i7-12700K" {
		t.Errorf("CpuModel = %q, want %q", got.CpuModel, "Intel Core i7-12700K")
	}
	if got.CpuCores != 12 {
		t.Errorf("CpuCores = %d, want %d", got.CpuCores, 12)
	}
	if got.CpuThreads != 20 {
		t.Errorf("CpuThreads = %d, want %d", got.CpuThreads, 20)
	}
	if got.RamBytes != 34359738368 {
		t.Errorf("RamBytes = %d, want %d", got.RamBytes, int64(34359738368))
	}
	if len(got.Disks) != 1 {
		t.Errorf("Disks count = %d, want 1", len(got.Disks))
	}
	if len(got.Gpus) != 1 {
		t.Errorf("Gpus count = %d, want 1", len(got.Gpus))
	}
	if len(got.Nics) != 1 {
		t.Errorf("Nics count = %d, want 1", len(got.Nics))
	}
}

func TestProfileStore_UpsertAndGetSoftwareInventory(t *testing.T) {
	s, agentID := testStoreWithAgent(t)
	ctx := context.Background()

	sw := &scoutpb.SoftwareInventory{
		OsName:    "Microsoft Windows 11 Pro",
		OsVersion: "10.0.22631",
		OsBuild:   "22631",
		Packages: []*scoutpb.InstalledPackage{
			{Name: "Visual Studio Code", Version: "1.85.0", Publisher: "Microsoft"},
			{Name: "Git", Version: "2.43.0", Publisher: "The Git Development Community"},
		},
		DockerContainers: []*scoutpb.DockerContainer{
			{ContainerId: "abc123", Name: "postgres", Image: "postgres:16", Status: "Up 2 hours"},
		},
	}

	if err := s.UpsertSoftwareInventory(ctx, agentID, sw); err != nil {
		t.Fatalf("UpsertSoftwareInventory: %v", err)
	}

	got, err := s.GetSoftwareInventory(ctx, agentID)
	if err != nil {
		t.Fatalf("GetSoftwareInventory: %v", err)
	}
	if got == nil {
		t.Fatal("GetSoftwareInventory returned nil")
	}
	if got.OsName != "Microsoft Windows 11 Pro" {
		t.Errorf("OsName = %q, want %q", got.OsName, "Microsoft Windows 11 Pro")
	}
	if got.OsVersion != "10.0.22631" {
		t.Errorf("OsVersion = %q, want %q", got.OsVersion, "10.0.22631")
	}
	if len(got.Packages) != 2 {
		t.Errorf("Packages count = %d, want 2", len(got.Packages))
	}
	if len(got.DockerContainers) != 1 {
		t.Errorf("DockerContainers count = %d, want 1", len(got.DockerContainers))
	}
}

func TestProfileStore_UpsertAndGetServices(t *testing.T) {
	s, agentID := testStoreWithAgent(t)
	ctx := context.Background()

	services := []*scoutpb.ServiceInfo{
		{Name: "wuauserv", DisplayName: "Windows Update", Status: "running", StartType: "auto"},
		{Name: "sshd", DisplayName: "OpenSSH Server", Status: "running", StartType: "auto", Ports: []int32{22}},
		{Name: "Spooler", DisplayName: "Print Spooler", Status: "stopped", StartType: "manual"},
	}

	if err := s.UpsertServices(ctx, agentID, services); err != nil {
		t.Fatalf("UpsertServices: %v", err)
	}

	got, err := s.GetServices(ctx, agentID)
	if err != nil {
		t.Fatalf("GetServices: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Services count = %d, want 3", len(got))
	}
	if got[0].Name != "wuauserv" {
		t.Errorf("Services[0].Name = %q, want %q", got[0].Name, "wuauserv")
	}
	if got[1].Ports[0] != 22 {
		t.Errorf("Services[1].Ports[0] = %d, want 22", got[1].Ports[0])
	}
}

func TestProfileStore_UpsertFullProfile(t *testing.T) {
	s, agentID := testStoreWithAgent(t)
	ctx := context.Background()

	hw := &scoutpb.HardwareProfile{CpuModel: "AMD Ryzen 9", CpuCores: 16}
	sw := &scoutpb.SoftwareInventory{OsName: "Windows 11", OsVersion: "10.0.22631"}
	services := []*scoutpb.ServiceInfo{{Name: "sshd", Status: "running"}}

	if err := s.UpsertFullProfile(ctx, agentID, hw, sw, services); err != nil {
		t.Fatalf("UpsertFullProfile: %v", err)
	}

	gotHW, err := s.GetHardwareProfile(ctx, agentID)
	if err != nil {
		t.Fatalf("GetHardwareProfile: %v", err)
	}
	if gotHW.CpuModel != "AMD Ryzen 9" {
		t.Errorf("CpuModel = %q, want %q", gotHW.CpuModel, "AMD Ryzen 9")
	}

	gotSW, err := s.GetSoftwareInventory(ctx, agentID)
	if err != nil {
		t.Fatalf("GetSoftwareInventory: %v", err)
	}
	if gotSW.OsName != "Windows 11" {
		t.Errorf("OsName = %q, want %q", gotSW.OsName, "Windows 11")
	}

	gotSvc, err := s.GetServices(ctx, agentID)
	if err != nil {
		t.Fatalf("GetServices: %v", err)
	}
	if len(gotSvc) != 1 {
		t.Fatalf("Services count = %d, want 1", len(gotSvc))
	}
}

func TestProfileStore_GetHardwareProfile_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetHardwareProfile(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetHardwareProfile: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent agent, got %+v", got)
	}
}

func TestProfileStore_GetSoftwareInventory_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetSoftwareInventory(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetSoftwareInventory: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent agent, got %+v", got)
	}
}

func TestProfileStore_GetServices_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetServices(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetServices: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent agent, got %+v", got)
	}
}

func TestProfileStore_UpdateExistingProfile(t *testing.T) {
	s, agentID := testStoreWithAgent(t)
	ctx := context.Background()

	// Initial insert.
	hw1 := &scoutpb.HardwareProfile{CpuModel: "Intel i5", CpuCores: 4}
	if err := s.UpsertHardwareProfile(ctx, agentID, hw1); err != nil {
		t.Fatalf("first UpsertHardwareProfile: %v", err)
	}

	// Update with new data.
	hw2 := &scoutpb.HardwareProfile{CpuModel: "Intel i7", CpuCores: 8}
	if err := s.UpsertHardwareProfile(ctx, agentID, hw2); err != nil {
		t.Fatalf("second UpsertHardwareProfile: %v", err)
	}

	got, err := s.GetHardwareProfile(ctx, agentID)
	if err != nil {
		t.Fatalf("GetHardwareProfile: %v", err)
	}
	if got.CpuModel != "Intel i7" {
		t.Errorf("CpuModel = %q, want %q after update", got.CpuModel, "Intel i7")
	}
	if got.CpuCores != 8 {
		t.Errorf("CpuCores = %d, want 8 after update", got.CpuCores)
	}
}
