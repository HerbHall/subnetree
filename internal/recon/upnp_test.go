package recon

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"go.uber.org/zap"
)

func TestNewUPNPDiscoverer(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := NewReconStore(db.DB())
	bus := &mockEventBus{}
	logger := zap.NewNop()

	disc := NewUPNPDiscoverer(s, bus, logger, 5*time.Minute)
	if disc == nil {
		t.Fatal("NewUPNPDiscoverer returned nil")
	}
	if disc.store != s {
		t.Error("store not set")
	}
	if disc.bus != bus {
		t.Error("bus not set")
	}
	if disc.interval != 5*time.Minute {
		t.Errorf("interval = %v, want 5m", disc.interval)
	}
	if disc.seen == nil {
		t.Error("seen map not initialized")
	}
}

func TestUPNPDiscoverer_RunStopsOnCancel(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := NewReconStore(db.DB())
	bus := &mockEventBus{}
	logger := zap.NewNop()

	// Use a very long interval so we only test cancellation, not ticks.
	disc := NewUPNPDiscoverer(s, bus, logger, 10*time.Minute)

	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		disc.Run(runCtx)
		close(done)
	}()

	// Cancel and verify the goroutine exits promptly.
	cancel()

	select {
	case <-done:
		// Run exited cleanly.
	case <-time.After(10 * time.Second):
		t.Fatal("UPNPDiscoverer.Run did not stop within 10 seconds after cancellation")
	}
}

func TestUPNPDiscoverer_Deduplication(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := NewReconStore(db.DB())
	bus := &mockEventBus{}
	logger := zap.NewNop()

	disc := NewUPNPDiscoverer(s, bus, logger, 60*time.Second)

	udn := "uuid:12345678-1234-1234-1234-123456789abc"

	// First time: not recently seen.
	if disc.recentlySeen(udn) {
		t.Error("UDN should not be recently seen before marking")
	}

	// Mark and check.
	disc.markSeen(udn)
	if !disc.recentlySeen(udn) {
		t.Error("UDN should be recently seen after marking")
	}

	// Second check: still recently seen.
	if !disc.recentlySeen(udn) {
		t.Error("UDN should still be recently seen")
	}
}

func TestUPNPDiscoverer_CleanSeen(t *testing.T) {
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := NewReconStore(db.DB())
	logger := zap.NewNop()

	// Very short interval so entries expire quickly.
	disc := NewUPNPDiscoverer(s, nil, logger, 10*time.Millisecond)

	disc.markSeen("uuid:device-1")
	disc.markSeen("uuid:device-2")

	// Both should exist.
	disc.mu.Lock()
	if len(disc.seen) != 2 {
		t.Errorf("seen map has %d entries, want 2", len(disc.seen))
	}
	disc.mu.Unlock()

	// Wait for entries to expire (2x interval = 20ms).
	time.Sleep(30 * time.Millisecond)

	disc.cleanSeen()

	disc.mu.Lock()
	if len(disc.seen) != 0 {
		t.Errorf("seen map has %d entries after clean, want 0", len(disc.seen))
	}
	disc.mu.Unlock()
}

func TestInferDeviceTypeFromUPnP(t *testing.T) {
	tests := []struct {
		deviceType string
		want       string
	}{
		{"urn:schemas-upnp-org:device:MediaRenderer:1", "nas"},
		{"urn:schemas-upnp-org:device:MediaServer:1", "nas"},
		{"urn:schemas-upnp-org:device:Printer:1", "printer"},
		{"urn:schemas-upnp-org:device:InternetGatewayDevice:1", "router"},
		{"urn:schemas-upnp-org:device:WANDevice:1", "router"},
		{"urn:schemas-upnp-org:device:WANConnectionDevice:1", "router"},
		{"urn:schemas-upnp-org:device:WLANAccessPointDevice:1", "access_point"},
		{"urn:schemas-upnp-org:device:DigitalSecurityCamera:1", "camera"},
		{"urn:schemas-upnp-org:device:BinaryLight:1", "iot"},
		{"urn:schemas-upnp-org:device:DimmableLight:1", "iot"},
		{"urn:schemas-upnp-org:device:HVAC_System:1", "iot"},
		{"urn:schemas-upnp-org:device:SensorManagement:1", "iot"},
		{"urn:schemas-upnp-org:device:LightingControls:1", "iot"},
		{"urn:schemas-upnp-org:device:Basic:1", "unknown"},
		{"", "unknown"},
		{"some-custom-device-type", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.deviceType, func(t *testing.T) {
			got := string(inferDeviceTypeFromUPnP(tt.deviceType))
			if got != tt.want {
				t.Errorf("inferDeviceTypeFromUPnP(%q) = %q, want %q", tt.deviceType, got, tt.want)
			}
		})
	}
}
