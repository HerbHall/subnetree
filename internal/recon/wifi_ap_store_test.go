package recon

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

func TestUpsertWiFiClient_RoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create a parent AP device and a client device (FK requirement).
	ap := &models.Device{
		ID:              "ap-1",
		Hostname:        "test-ap",
		MACAddress:      "00:11:22:33:44:55",
		DeviceType:      models.DeviceTypeAccessPoint,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryWiFi,
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
	}
	if _, err := s.UpsertDevice(ctx, ap); err != nil {
		t.Fatalf("upsert AP: %v", err)
	}

	client := &models.Device{
		ID:              "client-1",
		Hostname:        "my-phone",
		MACAddress:      "aa:bb:cc:dd:ee:ff",
		DeviceType:      models.DeviceTypePhone,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryWiFi,
		ParentDeviceID:  "ap-1",
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
	}
	if _, err := s.UpsertDevice(ctx, client); err != nil {
		t.Fatalf("upsert client: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	snap := &WiFiClientSnapshot{
		DeviceID:     "client-1",
		SignalDBm:    -65,
		SignalAvgDBm: -67,
		ConnectedSec: 7200,
		InactiveSec:  5,
		RxBitrate:    300_000_000,
		TxBitrate:    200_000_000,
		RxBytes:      1_048_576,
		TxBytes:      524_288,
		APBSSID:      "00:11:22:33:44:55",
		APSSID:       "TestNetwork",
		CollectedAt:  now,
	}

	// Upsert.
	if err := s.UpsertWiFiClient(ctx, snap); err != nil {
		t.Fatalf("UpsertWiFiClient: %v", err)
	}

	// Get it back.
	got, err := s.GetWiFiClient(ctx, "client-1")
	if err != nil {
		t.Fatalf("GetWiFiClient: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if got.SignalDBm != -65 {
		t.Errorf("SignalDBm = %d, want -65", got.SignalDBm)
	}
	if got.SignalAvgDBm != -67 {
		t.Errorf("SignalAvgDBm = %d, want -67", got.SignalAvgDBm)
	}
	if got.ConnectedSec != 7200 {
		t.Errorf("ConnectedSec = %d, want 7200", got.ConnectedSec)
	}
	if got.RxBitrate != 300_000_000 {
		t.Errorf("RxBitrate = %d, want 300000000", got.RxBitrate)
	}
	if got.TxBitrate != 200_000_000 {
		t.Errorf("TxBitrate = %d, want 200000000", got.TxBitrate)
	}
	if got.RxBytes != 1_048_576 {
		t.Errorf("RxBytes = %d, want 1048576", got.RxBytes)
	}
	if got.APBSSID != "00:11:22:33:44:55" {
		t.Errorf("APBSSID = %q, want %q", got.APBSSID, "00:11:22:33:44:55")
	}
	if got.APSSID != "TestNetwork" {
		t.Errorf("APSSID = %q, want %q", got.APSSID, "TestNetwork")
	}
}

func TestUpsertWiFiClient_ReplaceOnConflict(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	dev := &models.Device{
		ID:              "client-1",
		Hostname:        "test-device",
		DeviceType:      models.DeviceTypePhone,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryWiFi,
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
	}
	if _, err := s.UpsertDevice(ctx, dev); err != nil {
		t.Fatalf("upsert device: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	snap := &WiFiClientSnapshot{
		DeviceID:    "client-1",
		SignalDBm:   -55,
		CollectedAt: now,
	}
	if err := s.UpsertWiFiClient(ctx, snap); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Upsert again with different signal.
	snap.SignalDBm = -72
	if err := s.UpsertWiFiClient(ctx, snap); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	got, err := s.GetWiFiClient(ctx, "client-1")
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.SignalDBm != -72 {
		t.Errorf("SignalDBm after update = %d, want -72", got.SignalDBm)
	}
}

func TestListWiFiClients_All(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create two devices with wifi client snapshots.
	for _, d := range []struct {
		id, mac string
	}{
		{"c-1", "aa:bb:cc:11:11:11"},
		{"c-2", "aa:bb:cc:22:22:22"},
	} {
		dev := &models.Device{
			ID:              d.id,
			MACAddress:      d.mac,
			DeviceType:      models.DeviceTypePhone,
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryWiFi,
			FirstSeen:       now,
			LastSeen:        now,
		}
		if _, err := s.UpsertDevice(ctx, dev); err != nil {
			t.Fatalf("upsert %s: %v", d.id, err)
		}
		snap := &WiFiClientSnapshot{
			DeviceID:    d.id,
			SignalDBm:   -60,
			CollectedAt: now,
		}
		if err := s.UpsertWiFiClient(ctx, snap); err != nil {
			t.Fatalf("upsert snap %s: %v", d.id, err)
		}
	}

	// List all.
	all, err := s.ListWiFiClients(ctx, "")
	if err != nil {
		t.Fatalf("ListWiFiClients (all): %v", err)
	}
	if len(all) != 2 {
		t.Errorf("all count = %d, want 2", len(all))
	}
}

func TestListWiFiClients_FilterByAP(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Create AP device.
	ap := &models.Device{
		ID:              "ap-1",
		Hostname:        "test-ap",
		DeviceType:      models.DeviceTypeAccessPoint,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryWiFi,
		FirstSeen:       now,
		LastSeen:        now,
	}
	if _, err := s.UpsertDevice(ctx, ap); err != nil {
		t.Fatalf("upsert AP: %v", err)
	}

	// Create two clients: one under the AP, one not.
	for _, d := range []struct {
		id, mac, parent string
	}{
		{"c-1", "aa:bb:cc:11:11:11", "ap-1"},
		{"c-2", "aa:bb:cc:22:22:22", ""},
	} {
		dev := &models.Device{
			ID:              d.id,
			MACAddress:      d.mac,
			DeviceType:      models.DeviceTypePhone,
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryWiFi,
			ParentDeviceID:  d.parent,
			FirstSeen:       now,
			LastSeen:        now,
		}
		if _, err := s.UpsertDevice(ctx, dev); err != nil {
			t.Fatalf("upsert %s: %v", d.id, err)
		}
		snap := &WiFiClientSnapshot{
			DeviceID:    d.id,
			SignalDBm:   -60,
			CollectedAt: now,
		}
		if err := s.UpsertWiFiClient(ctx, snap); err != nil {
			t.Fatalf("upsert snap %s: %v", d.id, err)
		}
	}

	// Filter by AP.
	filtered, err := s.ListWiFiClients(ctx, "ap-1")
	if err != nil {
		t.Fatalf("ListWiFiClients (filtered): %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("filtered count = %d, want 1", len(filtered))
	}
	if len(filtered) == 1 && filtered[0].DeviceID != "c-1" {
		t.Errorf("filtered device = %q, want %q", filtered[0].DeviceID, "c-1")
	}
}

func TestGetWiFiClient_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetWiFiClient(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent device")
	}
}
