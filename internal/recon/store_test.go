package recon

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/HerbHall/netvantage/internal/store"
	"github.com/HerbHall/netvantage/pkg/models"
)

func testStore(t *testing.T) *ReconStore {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewReconStore(db.DB())
}

func TestUpsertDevice_CreateNew(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	device := &models.Device{
		Hostname:        "server-1",
		IPAddresses:     []string{"192.168.1.10"},
		MACAddress:      "AA:BB:CC:DD:EE:FF",
		Manufacturer:    "TestCorp",
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}

	created, err := s.UpsertDevice(ctx, device)
	if err != nil {
		t.Fatalf("UpsertDevice: %v", err)
	}
	if !created {
		t.Error("expected created=true for new device")
	}
	if device.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestUpsertDevice_UpdateByMAC(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create first device.
	d1 := &models.Device{
		IPAddresses:     []string{"192.168.1.10"},
		MACAddress:      "AA:BB:CC:DD:EE:FF",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	created, err := s.UpsertDevice(ctx, d1)
	if err != nil {
		t.Fatalf("first UpsertDevice: %v", err)
	}
	if !created {
		t.Error("expected first insert to create")
	}
	firstID := d1.ID

	// Upsert same MAC with new IP.
	d2 := &models.Device{
		IPAddresses:     []string{"192.168.1.20"},
		MACAddress:      "AA:BB:CC:DD:EE:FF",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryARP,
	}
	created, err = s.UpsertDevice(ctx, d2)
	if err != nil {
		t.Fatalf("second UpsertDevice: %v", err)
	}
	if created {
		t.Error("expected created=false for existing MAC")
	}
	if d2.ID != firstID {
		t.Errorf("ID = %q, want %q (same device)", d2.ID, firstID)
	}

	// Verify merged IPs.
	got, err := s.GetDevice(ctx, firstID)
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if len(got.IPAddresses) != 2 {
		t.Errorf("IPAddresses count = %d, want 2", len(got.IPAddresses))
	}
}

func TestUpsertDevice_UpdateByIP(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create device without MAC (ICMP-only).
	d1 := &models.Device{
		IPAddresses:     []string{"10.0.0.5"},
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	created, _ := s.UpsertDevice(ctx, d1)
	if !created {
		t.Error("expected created=true")
	}

	// Upsert same IP, now with MAC.
	d2 := &models.Device{
		IPAddresses:     []string{"10.0.0.5"},
		MACAddress:      "11:22:33:44:55:66",
		Manufacturer:    "SomeCorp",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryARP,
	}
	created, _ = s.UpsertDevice(ctx, d2)
	if created {
		t.Error("expected created=false for same IP")
	}

	got, _ := s.GetDevice(ctx, d1.ID)
	if got.MACAddress != "11:22:33:44:55:66" {
		t.Errorf("MACAddress = %q, want 11:22:33:44:55:66", got.MACAddress)
	}
}

func TestGetDevice_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	_, err := s.GetDevice(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent device")
	}
}

func TestListDevices_Pagination(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create 5 devices.
	for i := 0; i < 5; i++ {
		d := &models.Device{
			IPAddresses:     []string{fmt.Sprintf("10.0.0.%d", i+1)},
			MACAddress:      fmt.Sprintf("AA:BB:CC:DD:EE:%02X", i),
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryICMP,
		}
		if _, err := s.UpsertDevice(ctx, d); err != nil {
			t.Fatalf("create device %d: %v", i, err)
		}
	}

	// Page 1.
	devices, total, err := s.ListDevices(ctx, ListDevicesOptions{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(devices) != 2 {
		t.Errorf("page 1 count = %d, want 2", len(devices))
	}

	// Page 3 (last device).
	devices, _, err = s.ListDevices(ctx, ListDevicesOptions{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("ListDevices page 3: %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("page 3 count = %d, want 1", len(devices))
	}
}

func TestListDevices_FilterByStatus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	online := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:00:00:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, online)

	offline := &models.Device{
		IPAddresses:     []string{"10.0.0.2"},
		MACAddress:      "AA:BB:CC:00:00:02",
		Status:          models.DeviceStatusOffline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, offline)
	// Manually set status to offline since upsert sets it to online.
	_, _ = s.db.ExecContext(ctx, "UPDATE recon_devices SET status = 'offline' WHERE id = ?", offline.ID)

	devices, total, err := s.ListDevices(ctx, ListDevicesOptions{Status: "online"})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 {
		t.Errorf("count = %d, want 1", len(devices))
	}
}

func TestScanCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	scan := &models.ScanResult{
		Subnet: "192.168.1.0/24",
		Status: "running",
	}
	if err := s.CreateScan(ctx, scan); err != nil {
		t.Fatalf("CreateScan: %v", err)
	}
	if scan.ID == "" {
		t.Error("expected non-empty scan ID")
	}

	// Get scan.
	got, err := s.GetScan(ctx, scan.ID)
	if err != nil {
		t.Fatalf("GetScan: %v", err)
	}
	if got.Subnet != "192.168.1.0/24" {
		t.Errorf("Subnet = %q, want 192.168.1.0/24", got.Subnet)
	}
	if got.Status != "running" {
		t.Errorf("Status = %q, want running", got.Status)
	}

	// Update scan.
	scan.Status = "completed"
	scan.EndedAt = "2026-01-01T00:00:00Z"
	scan.Total = 10
	scan.Online = 8
	if err := s.UpdateScan(ctx, scan); err != nil {
		t.Fatalf("UpdateScan: %v", err)
	}

	got, _ = s.GetScan(ctx, scan.ID)
	if got.Status != "completed" {
		t.Errorf("Status = %q, want completed", got.Status)
	}
	if got.Total != 10 {
		t.Errorf("Total = %d, want 10", got.Total)
	}
	if got.Online != 8 {
		t.Errorf("Online = %d, want 8", got.Online)
	}

	// List scans.
	scans, err := s.ListScans(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListScans: %v", err)
	}
	if len(scans) != 1 {
		t.Errorf("scan count = %d, want 1", len(scans))
	}
}

func TestGetScan_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	_, err := s.GetScan(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent scan")
	}
}

func TestLinkScanDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create a scan and a device.
	scan := &models.ScanResult{Subnet: "10.0.0.0/24"}
	_ = s.CreateScan(ctx, scan)

	device := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:00:00:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, device)

	// Link them.
	if err := s.LinkScanDevice(ctx, scan.ID, device.ID); err != nil {
		t.Fatalf("LinkScanDevice: %v", err)
	}

	// Linking again should be idempotent.
	if err := s.LinkScanDevice(ctx, scan.ID, device.ID); err != nil {
		t.Fatalf("LinkScanDevice (second): %v", err)
	}

	// List devices for this scan.
	devices, total, err := s.ListDevices(ctx, ListDevicesOptions{ScanID: scan.ID})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 {
		t.Errorf("count = %d, want 1", len(devices))
	}
}

func TestFindStaleDevices(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create two online devices.
	fresh := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:00:00:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	stale := &models.Device{
		IPAddresses:     []string{"10.0.0.2"},
		MACAddress:      "AA:BB:CC:00:00:02",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, fresh)
	_, _ = s.UpsertDevice(ctx, stale)

	// Backdate the stale device's last_seen to 48 hours ago.
	oldTime := time.Now().Add(-48 * time.Hour)
	_, _ = s.db.ExecContext(ctx, "UPDATE recon_devices SET last_seen = ? WHERE id = ?", oldTime, stale.ID)

	// Threshold = 24 hours ago. Only the stale device should be returned.
	threshold := time.Now().Add(-24 * time.Hour)
	devices, err := s.FindStaleDevices(ctx, threshold)
	if err != nil {
		t.Fatalf("FindStaleDevices: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("got %d stale devices, want 1", len(devices))
	}
	if devices[0].ID != stale.ID {
		t.Errorf("stale device ID = %q, want %q", devices[0].ID, stale.ID)
	}
}

func TestFindStaleDevices_IgnoresOffline(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create an online device and backdate it.
	d := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:00:00:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d)

	oldTime := time.Now().Add(-48 * time.Hour)
	_, _ = s.db.ExecContext(ctx, "UPDATE recon_devices SET last_seen = ?, status = ? WHERE id = ?",
		oldTime, string(models.DeviceStatusOffline), d.ID)

	// Already-offline devices should not be returned.
	threshold := time.Now().Add(-24 * time.Hour)
	devices, err := s.FindStaleDevices(ctx, threshold)
	if err != nil {
		t.Fatalf("FindStaleDevices: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("got %d stale devices, want 0 (already offline)", len(devices))
	}
}

func TestMarkDeviceOffline(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:00:00:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d)

	if err := s.MarkDeviceOffline(ctx, d.ID); err != nil {
		t.Fatalf("MarkDeviceOffline: %v", err)
	}

	got, err := s.GetDevice(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.Status != models.DeviceStatusOffline {
		t.Errorf("status = %q, want %q", got.Status, models.DeviceStatusOffline)
	}
}

func TestTopologyLinkCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create two devices.
	d1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:00:00:00:00:01",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryARP,
	}
	d2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:00:00:00:00:02",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryARP,
	}
	_, _ = s.UpsertDevice(ctx, d1)
	_, _ = s.UpsertDevice(ctx, d2)

	link := &TopologyLink{
		SourceDeviceID: d1.ID,
		TargetDeviceID: d2.ID,
		LinkType:       "arp",
	}
	if err := s.UpsertTopologyLink(ctx, link); err != nil {
		t.Fatalf("UpsertTopologyLink: %v", err)
	}

	// Upsert again should update last_confirmed without error.
	if err := s.UpsertTopologyLink(ctx, &TopologyLink{
		SourceDeviceID: d1.ID,
		TargetDeviceID: d2.ID,
		LinkType:       "arp",
	}); err != nil {
		t.Fatalf("UpsertTopologyLink (second): %v", err)
	}

	links, err := s.GetTopologyLinks(ctx)
	if err != nil {
		t.Fatalf("GetTopologyLinks: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("link count = %d, want 1 (unique constraint)", len(links))
	}
}
