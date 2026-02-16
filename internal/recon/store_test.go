package recon

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/models"
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

// ---------------------------------------------------------------------------
// Device CRUD store tests
// ---------------------------------------------------------------------------

func TestUpdateDevice_Success(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d := &models.Device{
		Hostname:        "update-me",
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d)

	notes := "updated notes"
	tags := []string{"tag-a", "tag-b"}
	err := s.UpdateDevice(ctx, d.ID, UpdateDeviceParams{
		Notes: &notes,
		Tags:  &tags,
	})
	if err != nil {
		t.Fatalf("UpdateDevice: %v", err)
	}

	got, _ := s.GetDevice(ctx, d.ID)
	if got.Notes != "updated notes" {
		t.Errorf("Notes = %q, want %q", got.Notes, "updated notes")
	}
	if len(got.Tags) != 2 || got.Tags[0] != "tag-a" {
		t.Errorf("Tags = %v, want [tag-a tag-b]", got.Tags)
	}
}

func TestUpdateDevice_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	notes := "test"
	err := s.UpdateDevice(ctx, "nonexistent", UpdateDeviceParams{Notes: &notes})
	if err == nil {
		t.Error("expected error for nonexistent device")
	}
}

func TestUpdateDevice_PartialUpdate(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d := &models.Device{
		Hostname:        "partial-update",
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:02",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
		Notes:           "original notes",
	}
	_, _ = s.UpsertDevice(ctx, d)

	// Update only tags, leave notes untouched.
	tags := []string{"new-tag"}
	err := s.UpdateDevice(ctx, d.ID, UpdateDeviceParams{Tags: &tags})
	if err != nil {
		t.Fatalf("UpdateDevice: %v", err)
	}

	got, _ := s.GetDevice(ctx, d.ID)
	if got.Notes != "original notes" {
		t.Errorf("Notes = %q, want %q (should not change)", got.Notes, "original notes")
	}
	if len(got.Tags) != 1 || got.Tags[0] != "new-tag" {
		t.Errorf("Tags = %v, want [new-tag]", got.Tags)
	}
}

func TestDeleteDevice_Success(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d := &models.Device{
		Hostname:        "delete-me",
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:03",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d)

	err := s.DeleteDevice(ctx, d.ID)
	if err != nil {
		t.Fatalf("DeleteDevice: %v", err)
	}

	_, err = s.GetDevice(ctx, d.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestDeleteDevice_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	err := s.DeleteDevice(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent device")
	}
}

func TestInsertManualDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d := &models.Device{
		Hostname:    "manual-device",
		IPAddresses: []string{"10.0.0.99"},
		DeviceType:  models.DeviceTypeServer,
		Notes:       "manually added",
	}
	err := s.InsertManualDevice(ctx, d)
	if err != nil {
		t.Fatalf("InsertManualDevice: %v", err)
	}
	if d.ID == "" {
		t.Error("expected non-empty ID")
	}

	got, err := s.GetDevice(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.Hostname != "manual-device" {
		t.Errorf("Hostname = %q, want manual-device", got.Hostname)
	}
	if got.DiscoveryMethod != models.DiscoveryManual {
		t.Errorf("DiscoveryMethod = %q, want manual", got.DiscoveryMethod)
	}
	if got.Status != models.DeviceStatusUnknown {
		t.Errorf("Status = %q, want unknown", got.Status)
	}
}

func TestGetDeviceHistory_Empty(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	changes, total, err := s.GetDeviceHistory(ctx, "no-such-device", 50, 0)
	if err != nil {
		t.Fatalf("GetDeviceHistory: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(changes) != 0 {
		t.Errorf("changes = %d, want 0", len(changes))
	}
}

func TestGetDeviceHistory_WithChanges(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d := &models.Device{
		Hostname:        "history-test",
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:04",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d)

	// Mark offline -- should record a status change.
	if err := s.MarkDeviceOffline(ctx, d.ID); err != nil {
		t.Fatalf("MarkDeviceOffline: %v", err)
	}

	changes, total, err := s.GetDeviceHistory(ctx, d.ID, 50, 0)
	if err != nil {
		t.Fatalf("GetDeviceHistory: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(changes) != 1 {
		t.Fatalf("changes = %d, want 1", len(changes))
	}
	if changes[0].OldStatus != "online" {
		t.Errorf("OldStatus = %q, want online", changes[0].OldStatus)
	}
	if changes[0].NewStatus != "offline" {
		t.Errorf("NewStatus = %q, want offline", changes[0].NewStatus)
	}
}

func TestGetDeviceScans(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:05",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d)

	scan := &models.ScanResult{Subnet: "10.0.0.0/24", Status: "completed"}
	_ = s.CreateScan(ctx, scan)
	_ = s.LinkScanDevice(ctx, scan.ID, d.ID)

	scans, total, err := s.GetDeviceScans(ctx, d.ID, 50, 0)
	if err != nil {
		t.Fatalf("GetDeviceScans: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(scans) != 1 {
		t.Fatalf("scans = %d, want 1", len(scans))
	}
	if scans[0].Subnet != "10.0.0.0/24" {
		t.Errorf("Subnet = %q, want 10.0.0.0/24", scans[0].Subnet)
	}
}

func TestUpsertDevice_RecordsStatusChange(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create an online device.
	d := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:06",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d)

	// Manually set it offline so the next upsert detects a change.
	_, _ = s.db.ExecContext(ctx, "UPDATE recon_devices SET status = 'offline' WHERE id = ?", d.ID)

	// Upsert again -- status goes offline -> online.
	d2 := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:06",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d2)

	changes, total, err := s.GetDeviceHistory(ctx, d.ID, 50, 0)
	if err != nil {
		t.Fatalf("GetDeviceHistory: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if changes[0].OldStatus != "offline" {
		t.Errorf("OldStatus = %q, want offline", changes[0].OldStatus)
	}
	if changes[0].NewStatus != "online" {
		t.Errorf("NewStatus = %q, want online", changes[0].NewStatus)
	}
}

func TestUpsertDevice_UpdatesDeviceTypeFromUnknown(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create device with unknown type.
	device := &models.Device{
		Hostname:    "test-host",
		IPAddresses: []string{"192.168.1.100"},
		MACAddress:  "AA:BB:CC:DD:EE:FF",
		DeviceType:  models.DeviceTypeUnknown,
		Status:      models.DeviceStatusOnline,
	}
	created, err := s.UpsertDevice(ctx, device)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected device to be created")
	}

	// Upsert again with a classification.
	device2 := &models.Device{
		IPAddresses: []string{"192.168.1.100"},
		MACAddress:  "AA:BB:CC:DD:EE:FF",
		DeviceType:  models.DeviceTypeRouter,
		Status:      models.DeviceStatusOnline,
	}
	created2, err := s.UpsertDevice(ctx, device2)
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatal("expected update, not create")
	}

	// Verify device type was updated.
	got, err := s.GetDeviceByMAC(ctx, "AA:BB:CC:DD:EE:FF")
	if err != nil {
		t.Fatal(err)
	}
	if got.DeviceType != models.DeviceTypeRouter {
		t.Errorf("DeviceType = %q, want %q", got.DeviceType, models.DeviceTypeRouter)
	}
}

func TestUpsertDevice_DoesNotDowngradeDeviceType(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create device with router type.
	device := &models.Device{
		IPAddresses: []string{"192.168.1.200"},
		MACAddress:  "11:22:33:44:55:66",
		DeviceType:  models.DeviceTypeRouter,
		Status:      models.DeviceStatusOnline,
	}
	_, _ = s.UpsertDevice(ctx, device)

	// Upsert with a different type -- should NOT overwrite.
	device2 := &models.Device{
		IPAddresses: []string{"192.168.1.200"},
		MACAddress:  "11:22:33:44:55:66",
		DeviceType:  models.DeviceTypeSwitch,
		Status:      models.DeviceStatusOnline,
	}
	_, _ = s.UpsertDevice(ctx, device2)

	got, _ := s.GetDeviceByMAC(ctx, "11:22:33:44:55:66")
	if got.DeviceType != models.DeviceTypeRouter {
		t.Errorf("DeviceType = %q, want %q (should not be overwritten)", got.DeviceType, models.DeviceTypeRouter)
	}
}

func TestListDevices_FilterByType(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	server := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:00:00:01",
		DeviceType: models.DeviceTypeServer, Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	router := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:00:00:02",
		DeviceType: models.DeviceTypeRouter, Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, server)
	_, _ = s.UpsertDevice(ctx, router)

	devices, total, err := s.ListDevices(ctx, ListDevicesOptions{DeviceType: "server"})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 {
		t.Fatalf("count = %d, want 1", len(devices))
	}
	if devices[0].DeviceType != models.DeviceTypeServer {
		t.Errorf("DeviceType = %q, want server", devices[0].DeviceType)
	}
}

// ---------------------------------------------------------------------------
// Inventory management store tests
// ---------------------------------------------------------------------------

func TestUpdateDevice_InventoryFields(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d := &models.Device{
		Hostname:        "inv-test",
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:DD:EE:10",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d)

	location := "Rack A3, U12"
	category := "production"
	role := "web-server"
	owner := "platform-team"
	err := s.UpdateDevice(ctx, d.ID, UpdateDeviceParams{
		Location:    &location,
		Category:    &category,
		PrimaryRole: &role,
		Owner:       &owner,
	})
	if err != nil {
		t.Fatalf("UpdateDevice: %v", err)
	}

	got, _ := s.GetDevice(ctx, d.ID)
	if got.Location != "Rack A3, U12" {
		t.Errorf("Location = %q, want %q", got.Location, "Rack A3, U12")
	}
	if got.Category != "production" {
		t.Errorf("Category = %q, want %q", got.Category, "production")
	}
	if got.PrimaryRole != "web-server" {
		t.Errorf("PrimaryRole = %q, want %q", got.PrimaryRole, "web-server")
	}
	if got.Owner != "platform-team" {
		t.Errorf("Owner = %q, want %q", got.Owner, "platform-team")
	}
}

func TestListDevices_FilterByCategory(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	prod := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:00:00:01",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	dev := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:00:00:02",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, prod)
	_, _ = s.UpsertDevice(ctx, dev)

	cat1 := "production"
	cat2 := "development"
	_ = s.UpdateDevice(ctx, prod.ID, UpdateDeviceParams{Category: &cat1})
	_ = s.UpdateDevice(ctx, dev.ID, UpdateDeviceParams{Category: &cat2})

	devices, total, err := s.ListDevices(ctx, ListDevicesOptions{Category: "production"})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 {
		t.Fatalf("count = %d, want 1", len(devices))
	}
	if devices[0].Category != "production" {
		t.Errorf("Category = %q, want production", devices[0].Category)
	}
}

func TestListDevices_FilterByOwner(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:00:00:01",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	d2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:00:00:02",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d1)
	_, _ = s.UpsertDevice(ctx, d2)

	owner1 := "team-a"
	owner2 := "team-b"
	_ = s.UpdateDevice(ctx, d1.ID, UpdateDeviceParams{Owner: &owner1})
	_ = s.UpdateDevice(ctx, d2.ID, UpdateDeviceParams{Owner: &owner2})

	devices, total, err := s.ListDevices(ctx, ListDevicesOptions{Owner: "team-a"})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(devices) != 1 {
		t.Fatalf("count = %d, want 1", len(devices))
	}
	if devices[0].Owner != "team-a" {
		t.Errorf("Owner = %q, want team-a", devices[0].Owner)
	}
}

func TestGetInventorySummary(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create devices with different statuses and categories.
	online1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:00:00:01",
		DeviceType: models.DeviceTypeServer, Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	online2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:00:00:02",
		DeviceType: models.DeviceTypeRouter, Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, online1)
	_, _ = s.UpsertDevice(ctx, online2)

	// Set one device offline.
	_, _ = s.db.ExecContext(ctx, "UPDATE recon_devices SET status = 'offline' WHERE id = ?", online2.ID)

	// Set categories.
	cat := "production"
	_ = s.UpdateDevice(ctx, online1.ID, UpdateDeviceParams{Category: &cat})

	// Backdate one device to make it stale.
	oldTime := time.Now().Add(-48 * time.Hour)
	_, _ = s.db.ExecContext(ctx, "UPDATE recon_devices SET last_seen = ? WHERE id = ?", oldTime, online1.ID)

	summary, err := s.GetInventorySummary(ctx, 1)
	if err != nil {
		t.Fatalf("GetInventorySummary: %v", err)
	}
	if summary.TotalDevices != 2 {
		t.Errorf("TotalDevices = %d, want 2", summary.TotalDevices)
	}
	if summary.OnlineCount != 1 {
		t.Errorf("OnlineCount = %d, want 1", summary.OnlineCount)
	}
	if summary.OfflineCount != 1 {
		t.Errorf("OfflineCount = %d, want 1", summary.OfflineCount)
	}
	if summary.StaleCount != 1 {
		t.Errorf("StaleCount = %d, want 1", summary.StaleCount)
	}
	if summary.ByCategory["production"] != 1 {
		t.Errorf("ByCategory[production] = %d, want 1", summary.ByCategory["production"])
	}
	if summary.ByType["server"] != 1 {
		t.Errorf("ByType[server] = %d, want 1", summary.ByType["server"])
	}
	if summary.ByType["router"] != 1 {
		t.Errorf("ByType[router] = %d, want 1", summary.ByType["router"])
	}
}

func TestGetInventorySummary_Empty(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	summary, err := s.GetInventorySummary(ctx, 30)
	if err != nil {
		t.Fatalf("GetInventorySummary: %v", err)
	}
	if summary.TotalDevices != 0 {
		t.Errorf("TotalDevices = %d, want 0", summary.TotalDevices)
	}
	if summary.OnlineCount != 0 {
		t.Errorf("OnlineCount = %d, want 0", summary.OnlineCount)
	}
}

func TestBulkUpdateDevices(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create 3 devices.
	ids := make([]string, 3)
	for i := 0; i < 3; i++ {
		d := &models.Device{
			IPAddresses:     []string{fmt.Sprintf("10.0.0.%d", i+1)},
			MACAddress:      fmt.Sprintf("AA:BB:CC:DD:EE:%02X", i+10),
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryICMP,
		}
		_, _ = s.UpsertDevice(ctx, d)
		ids[i] = d.ID
	}

	// Bulk update category for first 2.
	cat := "production"
	owner := "ops-team"
	updated, err := s.BulkUpdateDevices(ctx, ids[:2], UpdateDeviceParams{
		Category: &cat,
		Owner:    &owner,
	})
	if err != nil {
		t.Fatalf("BulkUpdateDevices: %v", err)
	}
	if updated != 2 {
		t.Errorf("updated = %d, want 2", updated)
	}

	// Verify first two were updated.
	for _, id := range ids[:2] {
		got, _ := s.GetDevice(ctx, id)
		if got.Category != "production" {
			t.Errorf("device %s: Category = %q, want production", id, got.Category)
		}
		if got.Owner != "ops-team" {
			t.Errorf("device %s: Owner = %q, want ops-team", id, got.Owner)
		}
	}

	// Verify third was NOT updated.
	got, _ := s.GetDevice(ctx, ids[2])
	if got.Category != "" {
		t.Errorf("device %s: Category = %q, want empty", ids[2], got.Category)
	}
}

func TestBulkUpdateDevices_EmptyIDs(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	cat := "test"
	updated, err := s.BulkUpdateDevices(ctx, []string{}, UpdateDeviceParams{Category: &cat})
	if err != nil {
		t.Fatalf("BulkUpdateDevices: %v", err)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0", updated)
	}
}

func TestBulkUpdateDevices_NoFields(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	d := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:DD:EE:20",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = s.UpsertDevice(ctx, d)

	updated, err := s.BulkUpdateDevices(ctx, []string{d.ID}, UpdateDeviceParams{})
	if err != nil {
		t.Fatalf("BulkUpdateDevices: %v", err)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0 (no fields to update)", updated)
	}
}
