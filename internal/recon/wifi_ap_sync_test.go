package recon

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

func TestWiFiAPSyncer_Sync_Unavailable(t *testing.T) {
	s := testStore(t)
	syncer := NewWiFiAPSyncer(s, nil, zap.NewNop())

	enumerator := &mockAPClientEnumerator{available: false}
	result, err := syncer.Sync(context.Background(), enumerator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClientsFound != 0 {
		t.Errorf("ClientsFound = %d, want 0", result.ClientsFound)
	}
	if result.Created != 0 {
		t.Errorf("Created = %d, want 0", result.Created)
	}
}

func TestWiFiAPSyncer_Sync_CreatesChildDevices(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create the AP device so the syncer can find it.
	apDevice := &models.Device{
		ID:              "ap-1",
		Hostname:        "test-ap",
		MACAddress:      "00:11:22:33:44:55",
		DeviceType:      models.DeviceTypeAccessPoint,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryWiFi,
		FirstSeen:       time.Now().UTC(),
		LastSeen:        time.Now().UTC(),
	}
	if _, err := s.UpsertDevice(ctx, apDevice); err != nil {
		t.Fatalf("upsert AP: %v", err)
	}

	enumerator := &mockAPClientEnumerator{
		available: true,
		clients: []APClientInfo{
			{
				MACAddress: "aa:bb:cc:11:22:33",
				Signal:     -55,
				Connected:  2 * time.Hour,
				RxBitrate:  300_000_000,
				TxBitrate:  200_000_000,
				APBSSID:    "00:11:22:33:44:55",
				APSSID:     "TestNet",
			},
			{
				MACAddress: "aa:bb:cc:44:55:66",
				Signal:     -72,
				Connected:  30 * time.Minute,
				RxBitrate:  144_000_000,
				TxBitrate:  130_000_000,
				APBSSID:    "00:11:22:33:44:55",
				APSSID:     "TestNet",
			},
		},
	}

	syncer := NewWiFiAPSyncer(s, nil, zap.NewNop())
	result, err := syncer.Sync(ctx, enumerator)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if result.ClientsFound != 2 {
		t.Errorf("ClientsFound = %d, want 2", result.ClientsFound)
	}
	if result.Created != 2 {
		t.Errorf("Created = %d, want 2", result.Created)
	}
	if result.APsChecked != 1 {
		t.Errorf("APsChecked = %d, want 1", result.APsChecked)
	}

	// Verify first client device was created correctly.
	client1, err := s.GetDeviceByMAC(ctx, "aa:bb:cc:11:22:33")
	if err != nil {
		t.Fatalf("get client1: %v", err)
	}
	if client1 == nil {
		t.Fatal("expected client1 to exist")
	}
	if client1.ParentDeviceID != "ap-1" {
		t.Errorf("client1 ParentDeviceID = %q, want %q", client1.ParentDeviceID, "ap-1")
	}
	if client1.DiscoveryMethod != models.DiscoveryWiFi {
		t.Errorf("client1 DiscoveryMethod = %q, want wifi", client1.DiscoveryMethod)
	}
	if client1.ConnectionType != models.ConnectionWiFi {
		t.Errorf("client1 ConnectionType = %q, want wifi", client1.ConnectionType)
	}
	if client1.Status != models.DeviceStatusOnline {
		t.Errorf("client1 Status = %q, want online", client1.Status)
	}

	// Verify WiFi client snapshot was stored.
	snap, err := s.GetWiFiClient(ctx, client1.ID)
	if err != nil {
		t.Fatalf("get wifi client snap: %v", err)
	}
	if snap == nil {
		t.Fatal("expected wifi client snapshot")
	}
	if snap.SignalDBm != -55 {
		t.Errorf("SignalDBm = %d, want -55", snap.SignalDBm)
	}
	if snap.RxBitrate != 300_000_000 {
		t.Errorf("RxBitrate = %d, want 300000000", snap.RxBitrate)
	}
}

func TestWiFiAPSyncer_Sync_UpdatesExisting(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create AP device.
	apDevice := &models.Device{
		ID:              "ap-1",
		Hostname:        "test-ap",
		MACAddress:      "00:11:22:33:44:55",
		DeviceType:      models.DeviceTypeAccessPoint,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryWiFi,
		FirstSeen:       time.Now().UTC(),
		LastSeen:        time.Now().UTC(),
	}
	if _, err := s.UpsertDevice(ctx, apDevice); err != nil {
		t.Fatalf("upsert AP: %v", err)
	}

	// Pre-create a client device matching the MAC that will be enumerated.
	existing := &models.Device{
		ID:              "existing-client",
		Hostname:        "my-phone",
		MACAddress:      "aa:bb:cc:11:22:33",
		DeviceType:      models.DeviceTypePhone,
		Status:          models.DeviceStatusOffline,
		DiscoveryMethod: models.DiscoveryWiFi,
		ConnectionType:  models.ConnectionUnknown,
		ParentDeviceID:  "ap-1",
		FirstSeen:       time.Now().UTC(),
		LastSeen:        time.Now().UTC(),
	}
	if _, err := s.UpsertDevice(ctx, existing); err != nil {
		t.Fatalf("upsert existing: %v", err)
	}

	enumerator := &mockAPClientEnumerator{
		available: true,
		clients: []APClientInfo{
			{
				MACAddress: "aa:bb:cc:11:22:33",
				Signal:     -60,
				Connected:  1 * time.Hour,
				APBSSID:    "00:11:22:33:44:55",
				APSSID:     "TestNet",
			},
		},
	}

	syncer := NewWiFiAPSyncer(s, nil, zap.NewNop())
	result, err := syncer.Sync(ctx, enumerator)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if result.Created != 0 {
		t.Errorf("Created = %d, want 0 (existing device should be updated)", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}

	// Verify the device was updated.
	updated, err := s.GetDeviceByMAC(ctx, "aa:bb:cc:11:22:33")
	if err != nil {
		t.Fatalf("get updated device: %v", err)
	}
	if updated.Status != models.DeviceStatusOnline {
		t.Errorf("status = %q, want online", updated.Status)
	}
	if updated.ConnectionType != models.ConnectionWiFi {
		t.Errorf("connection_type = %q, want wifi", updated.ConnectionType)
	}
}

func TestWiFiAPSyncer_Sync_MarksUnseenOffline(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create AP device.
	apDevice := &models.Device{
		ID:              "ap-1",
		Hostname:        "test-ap",
		MACAddress:      "00:11:22:33:44:55",
		DeviceType:      models.DeviceTypeAccessPoint,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryWiFi,
		FirstSeen:       time.Now().UTC(),
		LastSeen:        time.Now().UTC(),
	}
	if _, err := s.UpsertDevice(ctx, apDevice); err != nil {
		t.Fatalf("upsert AP: %v", err)
	}

	// Create two WiFi child devices -- one will be "seen", one will not.
	for _, d := range []struct {
		id, mac string
	}{
		{"client-seen", "aa:bb:cc:11:11:11"},
		{"client-gone", "aa:bb:cc:22:22:22"},
	} {
		dev := &models.Device{
			ID:              d.id,
			MACAddress:      d.mac,
			DeviceType:      models.DeviceTypePhone,
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryWiFi,
			ConnectionType:  models.ConnectionWiFi,
			ParentDeviceID:  "ap-1",
			FirstSeen:       time.Now().UTC(),
			LastSeen:        time.Now().UTC(),
		}
		if _, err := s.UpsertDevice(ctx, dev); err != nil {
			t.Fatalf("upsert %s: %v", d.id, err)
		}
	}

	// Only enumerate the "seen" client.
	enumerator := &mockAPClientEnumerator{
		available: true,
		clients: []APClientInfo{
			{
				MACAddress: "aa:bb:cc:11:11:11",
				Signal:     -55,
				APBSSID:    "00:11:22:33:44:55",
				APSSID:     "TestNet",
			},
		},
	}

	syncer := NewWiFiAPSyncer(s, nil, zap.NewNop())
	if _, err := syncer.Sync(ctx, enumerator); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Verify "client-gone" was marked offline.
	goneDevice, err := s.GetDeviceByMAC(ctx, "aa:bb:cc:22:22:22")
	if err != nil {
		t.Fatalf("get gone device: %v", err)
	}
	if goneDevice == nil {
		t.Fatal("expected client-gone to still exist")
	}
	if goneDevice.Status != models.DeviceStatusOffline {
		t.Errorf("client-gone status = %q, want offline", goneDevice.Status)
	}

	// Verify "client-seen" remains online.
	seenDevice, err := s.GetDeviceByMAC(ctx, "aa:bb:cc:11:11:11")
	if err != nil {
		t.Fatalf("get seen device: %v", err)
	}
	if seenDevice == nil {
		t.Fatal("expected client-seen to exist")
	}
	if seenDevice.Status != models.DeviceStatusOnline {
		t.Errorf("client-seen status = %q, want online", seenDevice.Status)
	}
}

func TestWiFiAPSyncer_Sync_SetsConfidence100(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create AP device.
	apDevice := &models.Device{
		ID:              "ap-1",
		Hostname:        "test-ap",
		MACAddress:      "00:11:22:33:44:55",
		DeviceType:      models.DeviceTypeAccessPoint,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryWiFi,
		FirstSeen:       time.Now().UTC(),
		LastSeen:        time.Now().UTC(),
	}
	if _, err := s.UpsertDevice(ctx, apDevice); err != nil {
		t.Fatalf("upsert AP: %v", err)
	}

	enumerator := &mockAPClientEnumerator{
		available: true,
		clients: []APClientInfo{
			{
				MACAddress: "aa:bb:cc:99:88:77",
				Signal:     -60,
				APBSSID:    "00:11:22:33:44:55",
				APSSID:     "TestNet",
			},
		},
	}

	syncer := NewWiFiAPSyncer(s, nil, zap.NewNop())
	if _, err := syncer.Sync(ctx, enumerator); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	device, err := s.GetDeviceByMAC(ctx, "aa:bb:cc:99:88:77")
	if err != nil {
		t.Fatalf("get device: %v", err)
	}
	if device == nil {
		t.Fatal("expected device to exist")
	}
	if device.ClassificationConfidence != 100 {
		t.Errorf("ClassificationConfidence = %d, want 100", device.ClassificationConfidence)
	}
	if device.ClassificationSource != "wifi_ap" {
		t.Errorf("ClassificationSource = %q, want %q", device.ClassificationSource, "wifi_ap")
	}
}
