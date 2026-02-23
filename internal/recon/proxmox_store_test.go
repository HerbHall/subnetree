package recon

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

func TestUpsertAndGetProxmoxResource(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create a parent device first (needed for FK relationship in list queries).
	host := &models.Device{
		ID:              "host-1",
		Hostname:        "pve-host",
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
	}
	if _, err := s.UpsertDevice(ctx, host); err != nil {
		t.Fatalf("upsert host: %v", err)
	}

	// Create a VM device.
	vm := &models.Device{
		ID:              "vm-1",
		Hostname:        "docker-vm",
		DeviceType:      models.DeviceTypeVM,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryProxmox,
		ParentDeviceID:  "host-1",
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
	}
	if _, err := s.UpsertDevice(ctx, vm); err != nil {
		t.Fatalf("upsert vm: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	r := &ProxmoxResource{
		DeviceID:    "vm-1",
		CPUPercent:  25.5,
		MemUsedMB:   4096,
		MemTotalMB:  8192,
		DiskUsedGB:  10,
		DiskTotalGB: 50,
		UptimeSec:   86400,
		NetInBytes:  1073741824,
		NetOutBytes: 536870912,
		CollectedAt: now,
	}

	// Upsert resource.
	if err := s.UpsertProxmoxResource(ctx, r); err != nil {
		t.Fatalf("UpsertProxmoxResource: %v", err)
	}

	// Get it back.
	got, err := s.GetProxmoxResource(ctx, "vm-1")
	if err != nil {
		t.Fatalf("GetProxmoxResource: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil resource")
	}
	if got.DeviceName != "docker-vm" {
		t.Errorf("DeviceName = %q, want %q", got.DeviceName, "docker-vm")
	}
	if got.DeviceType != "virtual_machine" {
		t.Errorf("DeviceType = %q, want %q", got.DeviceType, "virtual_machine")
	}
	if got.CPUPercent != 25.5 {
		t.Errorf("CPUPercent = %f, want 25.5", got.CPUPercent)
	}
	if got.MemUsedMB != 4096 {
		t.Errorf("MemUsedMB = %d, want 4096", got.MemUsedMB)
	}
	if got.DiskTotalGB != 50 {
		t.Errorf("DiskTotalGB = %d, want 50", got.DiskTotalGB)
	}
	if got.UptimeSec != 86400 {
		t.Errorf("UptimeSec = %d, want 86400", got.UptimeSec)
	}

	// Upsert again (should replace).
	r.CPUPercent = 50.0
	if err := s.UpsertProxmoxResource(ctx, r); err != nil {
		t.Fatalf("UpsertProxmoxResource (update): %v", err)
	}
	got, err = s.GetProxmoxResource(ctx, "vm-1")
	if err != nil {
		t.Fatalf("GetProxmoxResource after update: %v", err)
	}
	if got.CPUPercent != 50.0 {
		t.Errorf("CPUPercent after update = %f, want 50.0", got.CPUPercent)
	}
}

func TestGetProxmoxResource_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetProxmoxResource(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent device")
	}
}

func TestListProxmoxResources(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create parent host.
	host := &models.Device{
		ID:              "host-1",
		Hostname:        "pve-host",
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
		FirstSeen:       now,
		LastSeen:        now,
	}
	if _, err := s.UpsertDevice(ctx, host); err != nil {
		t.Fatalf("upsert host: %v", err)
	}

	// Create two VMs under the host.
	for _, d := range []struct {
		id, hostname string
		status       models.DeviceStatus
	}{
		{"vm-1", "alpha-vm", models.DeviceStatusOnline},
		{"vm-2", "beta-vm", models.DeviceStatusOffline},
	} {
		dev := &models.Device{
			ID:              d.id,
			Hostname:        d.hostname,
			DeviceType:      models.DeviceTypeVM,
			Status:          d.status,
			DiscoveryMethod: models.DiscoveryProxmox,
			ParentDeviceID:  "host-1",
			FirstSeen:       now,
			LastSeen:        now,
		}
		if _, err := s.UpsertDevice(ctx, dev); err != nil {
			t.Fatalf("upsert %s: %v", d.id, err)
		}
		r := &ProxmoxResource{
			DeviceID:    d.id,
			CPUPercent:  10.0,
			MemUsedMB:   1024,
			MemTotalMB:  2048,
			CollectedAt: now,
		}
		if err := s.UpsertProxmoxResource(ctx, r); err != nil {
			t.Fatalf("upsert resource %s: %v", d.id, err)
		}
	}

	// List all.
	all, err := s.ListProxmoxResources(ctx, "", "")
	if err != nil {
		t.Fatalf("ListProxmoxResources (all): %v", err)
	}
	if len(all) != 2 {
		t.Errorf("all count = %d, want 2", len(all))
	}

	// Filter by parent.
	byParent, err := s.ListProxmoxResources(ctx, "host-1", "")
	if err != nil {
		t.Fatalf("ListProxmoxResources (parent): %v", err)
	}
	if len(byParent) != 2 {
		t.Errorf("by parent count = %d, want 2", len(byParent))
	}

	// Filter by status.
	onlineOnly, err := s.ListProxmoxResources(ctx, "", "online")
	if err != nil {
		t.Fatalf("ListProxmoxResources (online): %v", err)
	}
	if len(onlineOnly) != 1 {
		t.Errorf("online count = %d, want 1", len(onlineOnly))
	}
	if len(onlineOnly) == 1 && onlineOnly[0].DeviceName != "alpha-vm" {
		t.Errorf("online device = %q, want %q", onlineOnly[0].DeviceName, "alpha-vm")
	}

	// Filter by nonexistent parent.
	empty, err := s.ListProxmoxResources(ctx, "nonexistent", "")
	if err != nil {
		t.Fatalf("ListProxmoxResources (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("empty count = %d, want 0", len(empty))
	}
}

func TestFindDeviceByHostnameAndParent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create parent and child.
	host := &models.Device{
		ID:              "host-1",
		Hostname:        "pve-host",
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
		FirstSeen:       now,
		LastSeen:        now,
	}
	if _, err := s.UpsertDevice(ctx, host); err != nil {
		t.Fatalf("upsert host: %v", err)
	}

	child := &models.Device{
		ID:              "vm-1",
		Hostname:        "test-vm",
		DeviceType:      models.DeviceTypeVM,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryProxmox,
		ParentDeviceID:  "host-1",
		FirstSeen:       now,
		LastSeen:        now,
	}
	if _, err := s.UpsertDevice(ctx, child); err != nil {
		t.Fatalf("upsert child: %v", err)
	}

	// Found.
	got, err := s.FindDeviceByHostnameAndParent(ctx, "test-vm", "host-1")
	if err != nil {
		t.Fatalf("FindDeviceByHostnameAndParent: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil device")
	}
	if got.ID != "vm-1" {
		t.Errorf("ID = %q, want %q", got.ID, "vm-1")
	}

	// Not found (wrong parent).
	got, err = s.FindDeviceByHostnameAndParent(ctx, "test-vm", "wrong-parent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for wrong parent")
	}

	// Not found (wrong hostname).
	got, err = s.FindDeviceByHostnameAndParent(ctx, "nonexistent", "host-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for wrong hostname")
	}
}

func TestFindChildDevicesByDiscovery(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	host := &models.Device{
		ID:              "host-1",
		Hostname:        "pve-host",
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
		FirstSeen:       now,
		LastSeen:        now,
	}
	if _, err := s.UpsertDevice(ctx, host); err != nil {
		t.Fatalf("upsert host: %v", err)
	}

	// Two proxmox children plus one non-proxmox child.
	for _, d := range []struct {
		id, hostname string
		method       models.DiscoveryMethod
	}{
		{"vm-1", "vm-one", models.DiscoveryProxmox},
		{"vm-2", "vm-two", models.DiscoveryProxmox},
		{"other-1", "manual-device", models.DiscoveryICMP},
	} {
		dev := &models.Device{
			ID:              d.id,
			Hostname:        d.hostname,
			DeviceType:      models.DeviceTypeVM,
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: d.method,
			ParentDeviceID:  "host-1",
			FirstSeen:       now,
			LastSeen:        now,
		}
		if _, err := s.UpsertDevice(ctx, dev); err != nil {
			t.Fatalf("upsert %s: %v", d.id, err)
		}
	}

	// Find proxmox children only.
	devices, err := s.FindChildDevicesByDiscovery(ctx, "host-1", string(models.DiscoveryProxmox))
	if err != nil {
		t.Fatalf("FindChildDevicesByDiscovery: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("got %d devices, want 2", len(devices))
	}

	// No children under wrong parent.
	devices, err = s.FindChildDevicesByDiscovery(ctx, "nonexistent", string(models.DiscoveryProxmox))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("got %d devices, want 0", len(devices))
	}
}
