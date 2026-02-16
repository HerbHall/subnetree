package recon

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// mockPingScanner returns predetermined results.
type mockPingScanner struct {
	results []HostResult
	err     error
}

func (m *mockPingScanner) Scan(ctx context.Context, _ *net.IPNet, results chan<- HostResult) error {
	if m.err != nil {
		return m.err
	}
	for _, r := range m.results {
		select {
		case results <- r:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// mockARPReader returns a predetermined ARP table.
type mockARPReader struct {
	table map[string]string
}

func (m *mockARPReader) ReadTable(_ context.Context) map[string]string {
	if m.table == nil {
		return map[string]string{}
	}
	return m.table
}

// mockOUI returns a fixed manufacturer.
type mockOUI struct {
	table map[string]string
}

func (m *mockOUI) Lookup(mac string) string {
	prefix := normalizeMAC(mac)
	return m.table[prefix]
}

// eventCollector records published events for assertion.
type eventCollector struct {
	mu     sync.Mutex
	events []plugin.Event
}

func (c *eventCollector) handler(_ context.Context, e plugin.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

func (c *eventCollector) byTopic(topic string) []plugin.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []plugin.Event
	for _, e := range c.events {
		if e.Topic == topic {
			out = append(out, e)
		}
	}
	return out
}

func setupOrchestrator(t *testing.T, pinger *mockPingScanner, arp *mockARPReader, oui *mockOUI) (*ScanOrchestrator, *ReconStore, *eventCollector) {
	t.Helper()

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	reconStore := NewReconStore(db.DB())
	logger, _ := zap.NewDevelopment()
	bus := newTestBus(logger)

	collector := &eventCollector{}
	bus.SubscribeAll(collector.handler)

	orch := NewScanOrchestrator(reconStore, bus, oui, pinger, arp, logger)
	return orch, reconStore, collector
}

// newTestBus creates a real event bus for testing.
func newTestBus(logger *zap.Logger) plugin.EventBus {
	// Import the event package's NewBus.
	// Since we can't import internal/event from internal/recon (both internal),
	// use a minimal in-test implementation.
	return &testEventBus{logger: logger}
}

type testEventBus struct {
	mu       sync.RWMutex
	handlers []plugin.EventHandler
	logger   *zap.Logger
}

func (b *testEventBus) Publish(ctx context.Context, event plugin.Event) error {
	b.mu.RLock()
	handlers := make([]plugin.EventHandler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()
	for _, h := range handlers {
		h(ctx, event)
	}
	return nil
}

func (b *testEventBus) PublishAsync(ctx context.Context, event plugin.Event) {
	// For tests, run synchronously to avoid race conditions.
	_ = b.Publish(ctx, event)
}

func (b *testEventBus) Subscribe(topic string, handler plugin.EventHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	wrapped := func(ctx context.Context, e plugin.Event) {
		if e.Topic == topic {
			handler(ctx, e)
		}
	}
	b.handlers = append(b.handlers, wrapped)
	return func() {}
}

func (b *testEventBus) SubscribeAll(handler plugin.EventHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
	return func() {}
}

func TestScanOrchestrator_BasicScan(t *testing.T) {
	pinger := &mockPingScanner{
		results: []HostResult{
			{IP: "192.168.1.1", Alive: true, RTT: 5 * time.Millisecond, Method: "icmp"},
			{IP: "192.168.1.2", Alive: true, RTT: 10 * time.Millisecond, Method: "icmp"},
		},
	}
	arp := &mockARPReader{
		table: map[string]string{
			"192.168.1.1": "AA:BB:CC:DD:EE:01",
		},
	}
	oui := &mockOUI{
		table: map[string]string{
			"AA:BB:CC": "TestVendor",
		},
	}

	orch, reconStore, collector := setupOrchestrator(t, pinger, arp, oui)
	ctx := context.Background()

	// Create a scan record first.
	scan := &models.ScanResult{ID: "scan-1", Subnet: "192.168.1.0/24", Status: "running"}
	if err := reconStore.CreateScan(ctx, scan); err != nil {
		t.Fatalf("CreateScan: %v", err)
	}

	orch.RunScan(ctx, "scan-1", "192.168.1.0/24")

	// Verify scan completed.
	got, err := reconStore.GetScan(ctx, "scan-1")
	if err != nil {
		t.Fatalf("GetScan: %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("scan status = %q, want completed", got.Status)
	}
	if got.Total != 2 {
		t.Errorf("scan total = %d, want 2", got.Total)
	}
	if got.Online != 2 {
		t.Errorf("scan online = %d, want 2", got.Online)
	}

	// Verify devices created.
	devices, total, err := reconStore.ListDevices(ctx, ListDevicesOptions{ScanID: "scan-1"})
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if total != 2 {
		t.Errorf("device count = %d, want 2", total)
	}

	// Device with ARP should have manufacturer.
	for _, d := range devices {
		if d.MACAddress == "AA:BB:CC:DD:EE:01" {
			if d.Manufacturer != "TestVendor" {
				t.Errorf("manufacturer = %q, want TestVendor", d.Manufacturer)
			}
		}
	}

	// Verify events.
	// Allow time for async events.
	time.Sleep(50 * time.Millisecond)

	discovered := collector.byTopic(TopicDeviceDiscovered)
	if len(discovered) != 2 {
		t.Errorf("discovered events = %d, want 2", len(discovered))
	}

	completed := collector.byTopic(TopicScanCompleted)
	if len(completed) != 1 {
		t.Errorf("completed events = %d, want 1", len(completed))
	}
}

func TestScanOrchestrator_DeviceUpdate(t *testing.T) {
	pinger := &mockPingScanner{
		results: []HostResult{
			{IP: "10.0.0.1", Alive: true, RTT: 1 * time.Millisecond, Method: "icmp"},
		},
	}
	arp := &mockARPReader{table: map[string]string{"10.0.0.1": "AA:BB:CC:00:00:01"}}
	oui := &mockOUI{table: map[string]string{}}

	orch, reconStore, collector := setupOrchestrator(t, pinger, arp, oui)
	ctx := context.Background()

	// First scan: device should be created.
	scan1 := &models.ScanResult{ID: "scan-1", Subnet: "10.0.0.0/24", Status: "running"}
	_ = reconStore.CreateScan(ctx, scan1)
	orch.RunScan(ctx, "scan-1", "10.0.0.0/24")

	discovered := collector.byTopic(TopicDeviceDiscovered)
	if len(discovered) != 1 {
		t.Fatalf("first scan: discovered = %d, want 1", len(discovered))
	}

	// Second scan: same device should be updated, not created.
	scan2 := &models.ScanResult{ID: "scan-2", Subnet: "10.0.0.0/24", Status: "running"}
	_ = reconStore.CreateScan(ctx, scan2)
	orch.RunScan(ctx, "scan-2", "10.0.0.0/24")

	// Should have 1 discovered (first scan) + 1 updated (second scan).
	discovered = collector.byTopic(TopicDeviceDiscovered)
	updated := collector.byTopic(TopicDeviceUpdated)
	if len(discovered) != 1 {
		t.Errorf("total discovered = %d, want 1", len(discovered))
	}
	if len(updated) != 1 {
		t.Errorf("total updated = %d, want 1", len(updated))
	}
}

func TestScanOrchestrator_Cancellation(t *testing.T) {
	// Pinger that blocks until cancelled.
	pinger := &mockPingScanner{
		results: nil, // Will never send results.
		err:     context.Canceled,
	}

	orch, reconStore, _ := setupOrchestrator(t, pinger, &mockARPReader{}, &mockOUI{table: map[string]string{}})

	ctx, cancel := context.WithCancel(context.Background())

	scan := &models.ScanResult{ID: "scan-cancel", Subnet: "10.0.0.0/24", Status: "running"}
	_ = reconStore.CreateScan(ctx, scan)

	cancel() // Cancel immediately.
	orch.RunScan(ctx, "scan-cancel", "10.0.0.0/24")

	// Scan should be marked as failed/cancelled.
	got, _ := reconStore.GetScan(context.Background(), "scan-cancel")
	if got.Status != "failed" {
		t.Errorf("scan status = %q, want failed (cancelled)", got.Status)
	}
}

func TestScanOrchestrator_InvalidSubnet(t *testing.T) {
	orch, reconStore, _ := setupOrchestrator(t,
		&mockPingScanner{}, &mockARPReader{}, &mockOUI{table: map[string]string{}})
	ctx := context.Background()

	scan := &models.ScanResult{ID: "scan-bad", Subnet: "not-a-cidr", Status: "running"}
	_ = reconStore.CreateScan(ctx, scan)
	orch.RunScan(ctx, "scan-bad", "not-a-cidr")

	got, _ := reconStore.GetScan(ctx, "scan-bad")
	if got.Status != "failed" {
		t.Errorf("scan status = %q, want failed", got.Status)
	}
}

func TestScanOrchestrator_ResolveHostname(t *testing.T) {
	orch, _, _ := setupOrchestrator(t,
		&mockPingScanner{}, &mockARPReader{}, &mockOUI{table: map[string]string{}})

	// Private IP with no reverse DNS entry returns empty string.
	got := orch.resolveHostname("192.0.2.1") // TEST-NET-1, guaranteed no PTR record
	if got != "" {
		t.Errorf("resolveHostname(192.0.2.1) = %q, want empty", got)
	}

	// Localhost should resolve (platform-dependent, so just verify no panic
	// and the trailing dot is trimmed).
	name := orch.resolveHostname("127.0.0.1")
	if name != "" && name[len(name)-1] == '.' {
		t.Errorf("resolveHostname(127.0.0.1) = %q, trailing dot not trimmed", name)
	}
}

func TestScanOrchestrator_TopologyLinks(t *testing.T) {
	pinger := &mockPingScanner{
		results: []HostResult{
			{IP: "192.168.1.1", Alive: true, RTT: 1 * time.Millisecond, Method: "icmp"},
			{IP: "192.168.1.10", Alive: true, RTT: 5 * time.Millisecond, Method: "icmp"},
			{IP: "192.168.1.20", Alive: true, RTT: 8 * time.Millisecond, Method: "icmp"},
		},
	}
	arp := &mockARPReader{
		table: map[string]string{
			"192.168.1.1":  "AA:BB:CC:00:00:01",
			"192.168.1.10": "AA:BB:CC:00:00:0A",
			"192.168.1.20": "AA:BB:CC:00:00:14",
		},
	}
	oui := &mockOUI{table: map[string]string{}}

	orch, reconStore, _ := setupOrchestrator(t, pinger, arp, oui)
	ctx := context.Background()

	scan := &models.ScanResult{ID: "scan-topo", Subnet: "192.168.1.0/24", Status: "running"}
	if err := reconStore.CreateScan(ctx, scan); err != nil {
		t.Fatalf("CreateScan: %v", err)
	}

	orch.RunScan(ctx, "scan-topo", "192.168.1.0/24")

	// Verify scan completed.
	got, err := reconStore.GetScan(ctx, "scan-topo")
	if err != nil {
		t.Fatalf("GetScan: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("scan status = %q, want completed", got.Status)
	}

	// Verify topology links were created.
	links, err := reconStore.GetTopologyLinks(ctx)
	if err != nil {
		t.Fatalf("GetTopologyLinks: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("topology link count = %d, want 2", len(links))
	}

	// Look up the gateway device to verify link targets.
	gateway, err := reconStore.GetDeviceByIP(ctx, "192.168.1.1")
	if err != nil || gateway == nil {
		t.Fatalf("GetDeviceByIP(gateway): err=%v, device=%v", err, gateway)
	}

	for _, link := range links {
		if link.TargetDeviceID != gateway.ID {
			t.Errorf("link target = %q, want gateway %q", link.TargetDeviceID, gateway.ID)
		}
		if link.LinkType != "arp" {
			t.Errorf("link type = %q, want arp", link.LinkType)
		}
		if link.SourceDeviceID == gateway.ID {
			t.Errorf("link source should not be the gateway")
		}
	}
}

func TestFirstUsableIP(t *testing.T) {
	tests := []struct {
		cidr string
		want string
	}{
		{"192.168.1.0/24", "192.168.1.1"},
		{"10.0.0.0/8", "10.0.0.1"},
		{"172.16.0.0/16", "172.16.0.1"},
		{"192.168.100.0/24", "192.168.100.1"},
		{"10.10.10.0/30", "10.10.10.1"},
	}

	for _, tt := range tests {
		t.Run(tt.cidr, func(t *testing.T) {
			_, ipNet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("ParseCIDR: %v", err)
			}
			got := firstUsableIP(ipNet)
			if got != tt.want {
				t.Errorf("firstUsableIP(%s) = %q, want %q", tt.cidr, got, tt.want)
			}
		})
	}
}

func TestExpandSubnet(t *testing.T) {
	tests := []struct {
		cidr      string
		wantCount int
	}{
		{"192.168.1.0/30", 2},   // .1 and .2
		{"10.0.0.0/29", 6},     // .1 through .6
		{"172.16.0.0/31", 0},   // Only network + broadcast, no usable hosts (but /31 is special)
		{"192.168.1.0/24", 254}, // .1 through .254
	}

	for _, tt := range tests {
		t.Run(tt.cidr, func(t *testing.T) {
			_, ipNet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("ParseCIDR: %v", err)
			}
			hosts := expandSubnet(ipNet)
			if len(hosts) != tt.wantCount {
				t.Errorf("expandSubnet(%s) = %d hosts, want %d", tt.cidr, len(hosts), tt.wantCount)
			}
		})
	}
}

func TestExpandSubnet_TooLarge(t *testing.T) {
	_, ipNet, _ := net.ParseCIDR("10.0.0.0/15")
	hosts := expandSubnet(ipNet)
	if hosts != nil {
		t.Errorf("expected nil for /15 subnet (too large), got %d hosts", len(hosts))
	}
}

func TestRunStages_AllExecuted(t *testing.T) {
	orch, _, _ := setupOrchestrator(t,
		&mockPingScanner{}, &mockARPReader{}, &mockOUI{table: map[string]string{}})

	var executed []string
	stages := []scanStage{
		{"stage-a", func(_ context.Context) { executed = append(executed, "a") }},
		{"stage-b", func(_ context.Context) { executed = append(executed, "b") }},
		{"stage-c", func(_ context.Context) { executed = append(executed, "c") }},
	}

	orch.runStages(context.Background(), stages)

	if len(executed) != 3 {
		t.Fatalf("executed %d stages, want 3", len(executed))
	}
	for i, want := range []string{"a", "b", "c"} {
		if executed[i] != want {
			t.Errorf("stage %d = %q, want %q", i, executed[i], want)
		}
	}
}

func TestRunStages_CancelBetweenStages(t *testing.T) {
	orch, _, _ := setupOrchestrator(t,
		&mockPingScanner{}, &mockARPReader{}, &mockOUI{table: map[string]string{}})

	ctx, cancel := context.WithCancel(context.Background())

	var executed []string
	stages := []scanStage{
		{"stage-a", func(_ context.Context) {
			executed = append(executed, "a")
			cancel() // Cancel after first stage.
		}},
		{"stage-b", func(_ context.Context) { executed = append(executed, "b") }},
		{"stage-c", func(_ context.Context) { executed = append(executed, "c") }},
	}

	orch.runStages(ctx, stages)

	if len(executed) != 1 {
		t.Fatalf("executed %d stages, want 1 (only stage-a before cancel)", len(executed))
	}
	if executed[0] != "a" {
		t.Errorf("first stage = %q, want a", executed[0])
	}
}

func TestRunStages_EmptyStages(t *testing.T) {
	orch, _, _ := setupOrchestrator(t,
		&mockPingScanner{}, &mockARPReader{}, &mockOUI{table: map[string]string{}})

	// Should not panic on empty stages.
	orch.runStages(context.Background(), nil)
	orch.runStages(context.Background(), []scanStage{})
}
