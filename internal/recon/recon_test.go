package recon

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/HerbHall/netvantage/internal/store"
	"github.com/HerbHall/netvantage/pkg/models"
	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// mockEventBus records published events for verification.
type mockEventBus struct {
	mu     sync.Mutex
	events []plugin.Event
}

func (b *mockEventBus) Publish(_ context.Context, event plugin.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
	return nil
}

func (b *mockEventBus) Subscribe(_ string, _ plugin.EventHandler) (unsubscribe func()) {
	return func() {}
}

func (b *mockEventBus) PublishAsync(_ context.Context, event plugin.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
}

func (b *mockEventBus) SubscribeAll(_ plugin.EventHandler) (unsubscribe func()) {
	return func() {}
}

func (b *mockEventBus) Events() []plugin.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]plugin.Event, len(b.events))
	copy(cp, b.events)
	return cp
}

// setupTestModule creates a Module wired to an in-memory store and mock bus
// with a very short DeviceLostAfter threshold for testing.
func setupTestModule(t *testing.T) (*Module, *ReconStore, *mockEventBus) {
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

	s := NewReconStore(db.DB())
	bus := &mockEventBus{}
	logger := zap.NewNop()

	m := &Module{
		logger: logger,
		cfg: ReconConfig{
			DeviceLostAfter: 100 * time.Millisecond,
		},
		store: s,
		bus:   bus,
	}

	return m, s, bus
}

func TestCheckForLostDevices_DetectsStaleDevices(t *testing.T) {
	m, s, bus := setupTestModule(t)
	ctx := context.Background()

	// Create an online device and backdate it beyond the threshold.
	d := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:00:00:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, d); err != nil {
		t.Fatalf("UpsertDevice: %v", err)
	}

	// Backdate last_seen to well beyond the threshold.
	oldTime := time.Now().Add(-1 * time.Hour)
	if _, err := s.db.ExecContext(ctx, "UPDATE recon_devices SET last_seen = ? WHERE id = ?", oldTime, d.ID); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	// Set up scan context for the checker.
	m.scanCtx, m.scanCancel = context.WithCancel(context.Background())
	defer m.scanCancel()

	// Run the check directly.
	m.checkForLostDevices()

	// Verify device is now offline.
	got, err := s.GetDevice(ctx, d.ID)
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if got.Status != models.DeviceStatusOffline {
		t.Errorf("device status = %q, want %q", got.Status, models.DeviceStatusOffline)
	}

	// Verify DeviceLost event was published.
	events := bus.Events()
	if len(events) != 1 {
		t.Fatalf("published %d events, want 1", len(events))
	}
	if events[0].Topic != TopicDeviceLost {
		t.Errorf("event topic = %q, want %q", events[0].Topic, TopicDeviceLost)
	}
	if events[0].Source != "recon" {
		t.Errorf("event source = %q, want recon", events[0].Source)
	}
	payload, ok := events[0].Payload.(DeviceLostEvent)
	if !ok {
		t.Fatalf("payload type = %T, want DeviceLostEvent", events[0].Payload)
	}
	if payload.DeviceID != d.ID {
		t.Errorf("payload.DeviceID = %q, want %q", payload.DeviceID, d.ID)
	}
	if payload.IP != "10.0.0.1" {
		t.Errorf("payload.IP = %q, want 10.0.0.1", payload.IP)
	}
}

func TestCheckForLostDevices_SkipsFreshDevices(t *testing.T) {
	m, s, bus := setupTestModule(t)
	ctx := context.Background()

	// Create an online device that was just seen (fresh).
	d := &models.Device{
		IPAddresses:     []string{"10.0.0.1"},
		MACAddress:      "AA:BB:CC:00:00:01",
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, d); err != nil {
		t.Fatalf("UpsertDevice: %v", err)
	}

	m.scanCtx, m.scanCancel = context.WithCancel(context.Background())
	defer m.scanCancel()

	m.checkForLostDevices()

	// Device should still be online.
	got, _ := s.GetDevice(ctx, d.ID)
	if got.Status != models.DeviceStatusOnline {
		t.Errorf("device status = %q, want %q", got.Status, models.DeviceStatusOnline)
	}

	// No events should be published.
	if len(bus.Events()) != 0 {
		t.Errorf("published %d events, want 0", len(bus.Events()))
	}
}

func TestDeviceLostChecker_StopsOnCancel(t *testing.T) {
	m, _, _ := setupTestModule(t)

	m.scanCtx, m.scanCancel = context.WithCancel(context.Background())

	m.wg.Add(1)
	go m.runDeviceLostChecker()

	// Cancel the context and verify the goroutine exits promptly.
	m.scanCancel()

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Goroutine stopped cleanly.
	case <-time.After(2 * time.Second):
		t.Fatal("device lost checker did not stop within 2 seconds after context cancellation")
	}
}
