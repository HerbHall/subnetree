package autodoc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/internal/testutil"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"go.uber.org/zap"
)

func TestPluginContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

func newTestModule(t *testing.T) *Module {
	t.Helper()
	db := testutil.NewStore(t)

	m := New()
	logger, _ := zap.NewDevelopment()
	bus := testutil.NewMockBus()

	err := m.Init(context.Background(), plugin.Dependencies{
		Logger: logger.Named("autodoc"),
		Store:  db,
		Bus:    bus,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return m
}

func TestHandleDeviceDiscovered(t *testing.T) {
	m := newTestModule(t)

	device := &models.Device{
		ID:          "dev-001",
		Hostname:    "web-server-01",
		IPAddresses: []string{"192.168.1.10"},
	}

	event := plugin.Event{
		Topic:     TopicDeviceDiscovered,
		Source:    "recon",
		Timestamp: time.Now().UTC(),
		Payload:   &recon.DeviceEvent{ScanID: "scan-1", Device: device},
	}

	m.handleDeviceDiscovered(context.Background(), event)

	entries, total, err := m.store.ListEntries(context.Background(), ListFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 entry, got %d", total)
	}
	if entries[0].EventType != TopicDeviceDiscovered {
		t.Errorf("event_type = %q, want %q", entries[0].EventType, TopicDeviceDiscovered)
	}
	if !strings.Contains(entries[0].Summary, "web-server-01") {
		t.Errorf("summary = %q, expected to contain hostname", entries[0].Summary)
	}
	if !strings.Contains(entries[0].Summary, "192.168.1.10") {
		t.Errorf("summary = %q, expected to contain IP", entries[0].Summary)
	}
	if entries[0].DeviceID == nil || *entries[0].DeviceID != "dev-001" {
		t.Errorf("device_id = %v, want %q", entries[0].DeviceID, "dev-001")
	}
	if entries[0].SourceModule != "recon" {
		t.Errorf("source_module = %q, want %q", entries[0].SourceModule, "recon")
	}
}

func TestHandleDeviceLost(t *testing.T) {
	m := newTestModule(t)

	event := plugin.Event{
		Topic:     TopicDeviceLost,
		Source:    "recon",
		Timestamp: time.Now().UTC(),
		Payload: recon.DeviceLostEvent{
			DeviceID: "dev-002",
			IP:       "192.168.1.20",
			LastSeen: time.Now().Add(-1 * time.Hour),
		},
	}

	m.handleDeviceLost(context.Background(), event)

	entries, total, err := m.store.ListEntries(context.Background(), ListFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 entry, got %d", total)
	}
	if entries[0].EventType != TopicDeviceLost {
		t.Errorf("event_type = %q, want %q", entries[0].EventType, TopicDeviceLost)
	}
	if !strings.Contains(entries[0].Summary, "192.168.1.20") {
		t.Errorf("summary = %q, expected to contain IP", entries[0].Summary)
	}
}

func TestHandleScanCompleted(t *testing.T) {
	m := newTestModule(t)

	scan := &models.ScanResult{
		ID:     "scan-001",
		Subnet: "192.168.1.0/24",
		Status: "completed",
		Total:  12,
		Online: 8,
	}

	event := plugin.Event{
		Topic:     TopicScanCompleted,
		Source:    "recon",
		Timestamp: time.Now().UTC(),
		Payload:   scan,
	}

	m.handleScanCompleted(context.Background(), event)

	entries, total, err := m.store.ListEntries(context.Background(), ListFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 entry, got %d", total)
	}
	if !strings.Contains(entries[0].Summary, "192.168.1.0/24") {
		t.Errorf("summary = %q, expected to contain subnet", entries[0].Summary)
	}
	if !strings.Contains(entries[0].Summary, "12 devices found") {
		t.Errorf("summary = %q, expected to contain device count", entries[0].Summary)
	}
	// Scan events have no device ID.
	if entries[0].DeviceID != nil {
		t.Errorf("device_id = %v, want nil for scan events", entries[0].DeviceID)
	}
}

func TestListEntriesPagination(t *testing.T) {
	m := newTestModule(t)

	// Create 5 entries.
	for i := 0; i < 5; i++ {
		entry := ChangelogEntry{
			ID:           testID(i),
			EventType:    TopicDeviceDiscovered,
			Summary:      "Test entry",
			Details:      json.RawMessage("{}"),
			SourceModule: "test",
			CreatedAt:    time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		if err := m.store.SaveEntry(context.Background(), entry); err != nil {
			t.Fatalf("SaveEntry %d: %v", i, err)
		}
	}

	// Page 1, 2 per page.
	entries, total, err := m.store.ListEntries(context.Background(), ListFilter{Page: 1, PerPage: 2})
	if err != nil {
		t.Fatalf("ListEntries page 1: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}

	// Page 3, 2 per page (should have 1 entry).
	entries, _, err = m.store.ListEntries(context.Background(), ListFilter{Page: 3, PerPage: 2})
	if err != nil {
		t.Fatalf("ListEntries page 3: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("entries = %d, want 1", len(entries))
	}
}

func TestListEntriesFilterByEventType(t *testing.T) {
	m := newTestModule(t)

	// Create entries of different types.
	types := []string{TopicDeviceDiscovered, TopicDeviceLost, TopicDeviceDiscovered, TopicAlertTriggered}
	for i, et := range types {
		entry := ChangelogEntry{
			ID:           testID(i),
			EventType:    et,
			Summary:      "Test",
			Details:      json.RawMessage("{}"),
			SourceModule: "test",
			CreatedAt:    time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		if err := m.store.SaveEntry(context.Background(), entry); err != nil {
			t.Fatalf("SaveEntry %d: %v", i, err)
		}
	}

	entries, total, err := m.store.ListEntries(context.Background(), ListFilter{
		Page:      1,
		PerPage:   50,
		EventType: TopicDeviceDiscovered,
	})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

func TestGetStats(t *testing.T) {
	m := newTestModule(t)

	// Create a few entries.
	types := []string{TopicDeviceDiscovered, TopicDeviceLost, TopicDeviceDiscovered}
	for i, et := range types {
		entry := ChangelogEntry{
			ID:           testID(i),
			EventType:    et,
			Summary:      "Test",
			Details:      json.RawMessage("{}"),
			SourceModule: "test",
			CreatedAt:    time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		if err := m.store.SaveEntry(context.Background(), entry); err != nil {
			t.Fatalf("SaveEntry %d: %v", i, err)
		}
	}

	stats, err := m.store.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalEntries != 3 {
		t.Errorf("TotalEntries = %d, want 3", stats.TotalEntries)
	}
	if stats.EntriesByType[TopicDeviceDiscovered] != 2 {
		t.Errorf("EntriesByType[discovered] = %d, want 2", stats.EntriesByType[TopicDeviceDiscovered])
	}
	if stats.EntriesByType[TopicDeviceLost] != 1 {
		t.Errorf("EntriesByType[lost] = %d, want 1", stats.EntriesByType[TopicDeviceLost])
	}
	if stats.LatestEntry == nil {
		t.Error("LatestEntry should not be nil")
	}
	if stats.OldestEntry == nil {
		t.Error("OldestEntry should not be nil")
	}
}

func TestGenerateMarkdown(t *testing.T) {
	now := time.Now().UTC()

	entries := []ChangelogEntry{
		{
			ID:           "e1",
			EventType:    TopicDeviceDiscovered,
			Summary:      "New device discovered: web-01 (192.168.1.10)",
			SourceModule: "recon",
			CreatedAt:    now.Add(-2 * time.Hour),
		},
		{
			ID:           "e2",
			EventType:    TopicScanCompleted,
			Summary:      "Network scan completed on 192.168.1.0/24: 5 devices found (3 online)",
			SourceModule: "recon",
			CreatedAt:    now.Add(-1 * time.Hour),
		},
		{
			ID:           "e3",
			EventType:    TopicAlertTriggered,
			Summary:      "Alert triggered: host unreachable (severity: warning)",
			SourceModule: "pulse",
			CreatedAt:    now,
		},
	}

	md := GenerateMarkdown(entries)

	if !strings.Contains(md, "# Infrastructure Changelog") {
		t.Error("markdown should contain title")
	}
	if !strings.Contains(md, "web-01") {
		t.Error("markdown should contain device hostname")
	}
	if !strings.Contains(md, "192.168.1.0/24") {
		t.Error("markdown should contain subnet")
	}
	if !strings.Contains(md, "[ALERT]") {
		t.Error("markdown should contain alert icon")
	}
	if !strings.Contains(md, "_(via recon)_") {
		t.Error("markdown should contain source module tag")
	}
}

func TestGenerateMarkdownEmpty(t *testing.T) {
	md := GenerateMarkdown(nil)
	if !strings.Contains(md, "No changes recorded") {
		t.Error("empty markdown should show no changes message")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"7d", 7 * 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"1d", 24 * time.Hour},
		{"2h", 2 * time.Hour},
		{"", 7 * 24 * time.Hour},
		{"invalid", 7 * 24 * time.Hour},
	}

	fallback := 7 * 24 * time.Hour
	for _, tc := range tests {
		got := ParseDuration(tc.input, fallback)
		if got != tc.expected {
			t.Errorf("ParseDuration(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestModuleRoutes(t *testing.T) {
	m := New()
	routes := m.Routes()
	if len(routes) != 5 {
		t.Fatalf("Routes() = %d, want 5", len(routes))
	}

	expected := map[string]string{
		"GET /changes":      "",
		"GET /export":       "",
		"GET /stats":        "",
		"GET /devices/{id}": "",
		"GET /devices":      "",
	}

	for _, r := range routes {
		key := r.Method + " " + r.Path
		if _, ok := expected[key]; !ok {
			t.Errorf("unexpected route: %s", key)
		}
	}
}

func TestModuleSubscriptions(t *testing.T) {
	m := New()
	subs := m.Subscriptions()
	if len(subs) != 6 {
		t.Fatalf("Subscriptions() = %d, want 6", len(subs))
	}

	expectedTopics := map[string]bool{
		TopicDeviceDiscovered: false,
		TopicDeviceUpdated:    false,
		TopicDeviceLost:       false,
		TopicScanCompleted:    false,
		TopicAlertTriggered:   false,
		TopicAlertResolved:    false,
	}

	for _, s := range subs {
		if _, ok := expectedTopics[s.Topic]; !ok {
			t.Errorf("unexpected subscription topic: %s", s.Topic)
		}
		expectedTopics[s.Topic] = true
	}

	for topic, found := range expectedTopics {
		if !found {
			t.Errorf("missing subscription for topic: %s", topic)
		}
	}
}

func TestDeviceLabel(t *testing.T) {
	tests := []struct {
		hostname string
		ip       string
		expected string
	}{
		{"web-01", "192.168.1.10", "web-01 (192.168.1.10)"},
		{"web-01", "", "web-01"},
		{"", "192.168.1.10", "192.168.1.10"},
		{"", "", "unknown"},
	}

	for _, tc := range tests {
		got := deviceLabel(tc.hostname, tc.ip)
		if got != tc.expected {
			t.Errorf("deviceLabel(%q, %q) = %q, want %q", tc.hostname, tc.ip, got, tc.expected)
		}
	}
}

// testID generates a deterministic test ID.
func testID(n int) string {
	return strings.Repeat("0", 35) + string(rune('a'+n))
}
