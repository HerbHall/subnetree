package pulse

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"go.uber.org/zap"
)

// correlationTestStore creates an in-memory store with recon_devices table
// including parent_device_id for correlation tests.
func correlationTestStore(t *testing.T) *PulseStore {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "pulse", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Create recon_devices table with parent_device_id column.
	_, err = db.DB().ExecContext(ctx, `CREATE TABLE IF NOT EXISTS recon_devices (
		id               TEXT PRIMARY KEY,
		hostname         TEXT NOT NULL DEFAULT '',
		ip_addresses     TEXT NOT NULL DEFAULT '[]',
		mac_address      TEXT NOT NULL DEFAULT '',
		manufacturer     TEXT NOT NULL DEFAULT '',
		device_type      TEXT NOT NULL DEFAULT 'unknown',
		os               TEXT NOT NULL DEFAULT '',
		status           TEXT NOT NULL DEFAULT 'unknown',
		discovery_method TEXT NOT NULL DEFAULT 'icmp',
		agent_id         TEXT NOT NULL DEFAULT '',
		first_seen       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_seen        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		notes            TEXT NOT NULL DEFAULT '',
		tags             TEXT NOT NULL DEFAULT '[]',
		custom_fields    TEXT NOT NULL DEFAULT '{}',
		parent_device_id TEXT NOT NULL DEFAULT '',
		network_layer    INTEGER NOT NULL DEFAULT 0,
		connection_type  TEXT NOT NULL DEFAULT ''
	)`)
	if err != nil {
		t.Fatalf("create recon_devices table: %v", err)
	}

	return NewPulseStore(db.DB())
}

// insertTestDevice inserts a device with optional parent into the recon_devices table.
func insertTestDevice(t *testing.T, ps *PulseStore, id, hostname, parentID string) {
	t.Helper()
	_, err := ps.db.ExecContext(context.Background(),
		`INSERT INTO recon_devices (id, hostname, ip_addresses, parent_device_id) VALUES (?, ?, '["10.0.0.1"]', ?)`,
		id, hostname, parentID,
	)
	if err != nil {
		t.Fatalf("insert test device %s: %v", id, err)
	}
}

// insertTestAlert inserts an alert directly into the store for testing.
func insertTestAlert(t *testing.T, ps *PulseStore, alert *Alert) {
	t.Helper()
	if err := ps.InsertAlert(context.Background(), alert); err != nil {
		t.Fatalf("insert test alert: %v", err)
	}
}

func TestCorrelation_ParentAlerting_SuppressesChild(t *testing.T) {
	ps := correlationTestStore(t)
	ctx := context.Background()
	logger := zap.NewNop()

	// Setup: router (parent) -> switch (child).
	insertTestDevice(t, ps, "router-1", "router", "")
	insertTestDevice(t, ps, "switch-1", "switch", "router-1")

	// Create checks for both devices.
	insertCorrelationCheck(t, ps, "chk-router", "router-1")
	insertCorrelationCheck(t, ps, "chk-switch", "switch-1")

	// Parent has an active alert.
	now := time.Now().UTC()
	insertTestAlert(t, ps, &Alert{
		ID:                  "alert-router-1",
		CheckID:             "chk-router",
		DeviceID:            "router-1",
		Severity:            "critical",
		Message:             "router unreachable",
		TriggeredAt:         now.Add(-2 * time.Minute),
		ConsecutiveFailures: 5,
	})

	engine := NewCorrelationEngine(ps, 5*time.Minute, logger)
	result, err := engine.Check(ctx, "switch-1")
	if err != nil {
		t.Fatalf("correlation check: %v", err)
	}
	if !result.Suppressed {
		t.Error("expected child alert to be suppressed, got not suppressed")
	}
	if result.ParentDeviceID != "router-1" {
		t.Errorf("ParentDeviceID = %q, want %q", result.ParentDeviceID, "router-1")
	}
}

func TestCorrelation_NoParentAlert_NotSuppressed(t *testing.T) {
	ps := correlationTestStore(t)
	ctx := context.Background()
	logger := zap.NewNop()

	// Setup: router (parent) -> switch (child), no alerts on parent.
	insertTestDevice(t, ps, "router-1", "router", "")
	insertTestDevice(t, ps, "switch-1", "switch", "router-1")

	engine := NewCorrelationEngine(ps, 5*time.Minute, logger)
	result, err := engine.Check(ctx, "switch-1")
	if err != nil {
		t.Fatalf("correlation check: %v", err)
	}
	if result.Suppressed {
		t.Error("expected alert not to be suppressed when parent has no alerts")
	}
}

func TestCorrelation_WindowExpiry_NotSuppressed(t *testing.T) {
	ps := correlationTestStore(t)
	ctx := context.Background()
	logger := zap.NewNop()

	// Setup: router (parent) -> switch (child).
	insertTestDevice(t, ps, "router-1", "router", "")
	insertTestDevice(t, ps, "switch-1", "switch", "router-1")

	insertCorrelationCheck(t, ps, "chk-router", "router-1")

	// Parent alert was triggered 10 minutes ago (outside 5-minute window).
	insertTestAlert(t, ps, &Alert{
		ID:                  "alert-router-old",
		CheckID:             "chk-router",
		DeviceID:            "router-1",
		Severity:            "critical",
		Message:             "router unreachable",
		TriggeredAt:         time.Now().UTC().Add(-10 * time.Minute),
		ConsecutiveFailures: 5,
	})

	engine := NewCorrelationEngine(ps, 5*time.Minute, logger)
	result, err := engine.Check(ctx, "switch-1")
	if err != nil {
		t.Fatalf("correlation check: %v", err)
	}
	if result.Suppressed {
		t.Error("expected alert not to be suppressed when parent alert is outside correlation window")
	}
}

func TestCorrelation_NoParentDevice_NotSuppressed(t *testing.T) {
	ps := correlationTestStore(t)
	ctx := context.Background()
	logger := zap.NewNop()

	// Device has no parent_device_id set.
	insertTestDevice(t, ps, "standalone-1", "standalone", "")

	engine := NewCorrelationEngine(ps, 5*time.Minute, logger)
	result, err := engine.Check(ctx, "standalone-1")
	if err != nil {
		t.Fatalf("correlation check: %v", err)
	}
	if result.Suppressed {
		t.Error("expected alert not to be suppressed when device has no parent")
	}
}

func TestCorrelation_DeviceNotInDB_NotSuppressed(t *testing.T) {
	ps := correlationTestStore(t)
	ctx := context.Background()
	logger := zap.NewNop()

	// Device doesn't exist in the database at all.
	engine := NewCorrelationEngine(ps, 5*time.Minute, logger)
	result, err := engine.Check(ctx, "nonexistent-device")
	if err != nil {
		t.Fatalf("correlation check: %v", err)
	}
	if result.Suppressed {
		t.Error("expected alert not to be suppressed for nonexistent device")
	}
}

func TestCorrelation_ParentAlertResolved_NotSuppressed(t *testing.T) {
	ps := correlationTestStore(t)
	ctx := context.Background()
	logger := zap.NewNop()

	insertTestDevice(t, ps, "router-1", "router", "")
	insertTestDevice(t, ps, "switch-1", "switch", "router-1")

	insertCorrelationCheck(t, ps, "chk-router", "router-1")

	// Parent has a resolved alert (not active).
	now := time.Now().UTC()
	resolvedAt := now.Add(-1 * time.Minute)
	insertTestAlert(t, ps, &Alert{
		ID:                  "alert-router-resolved",
		CheckID:             "chk-router",
		DeviceID:            "router-1",
		Severity:            "critical",
		Message:             "router unreachable",
		TriggeredAt:         now.Add(-3 * time.Minute),
		ResolvedAt:          &resolvedAt,
		ConsecutiveFailures: 5,
	})

	engine := NewCorrelationEngine(ps, 5*time.Minute, logger)
	result, err := engine.Check(ctx, "switch-1")
	if err != nil {
		t.Fatalf("correlation check: %v", err)
	}
	if result.Suppressed {
		t.Error("expected alert not to be suppressed when parent alert is resolved")
	}
}

func TestGetCorrelatedAlerts_GroupsByParent(t *testing.T) {
	ps := correlationTestStore(t)
	ctx := context.Background()

	insertTestDevice(t, ps, "router-1", "router", "")
	insertTestDevice(t, ps, "switch-1", "switch", "router-1")
	insertTestDevice(t, ps, "switch-2", "switch-2", "router-1")

	insertCorrelationCheck(t, ps, "chk-router", "router-1")
	insertCorrelationCheck(t, ps, "chk-switch1", "switch-1")
	insertCorrelationCheck(t, ps, "chk-switch2", "switch-2")

	now := time.Now().UTC()

	// Parent alert (not suppressed).
	insertTestAlert(t, ps, &Alert{
		ID:                  "alert-router-1",
		CheckID:             "chk-router",
		DeviceID:            "router-1",
		Severity:            "critical",
		Message:             "router unreachable",
		TriggeredAt:         now.Add(-2 * time.Minute),
		ConsecutiveFailures: 5,
	})

	// Child alerts (suppressed by router-1).
	insertTestAlert(t, ps, &Alert{
		ID:                  "alert-switch-1",
		CheckID:             "chk-switch1",
		DeviceID:            "switch-1",
		Severity:            "warning",
		Message:             "switch unreachable",
		TriggeredAt:         now.Add(-1 * time.Minute),
		ConsecutiveFailures: 3,
		Suppressed:          true,
		SuppressedBy:        "router-1",
	})
	insertTestAlert(t, ps, &Alert{
		ID:                  "alert-switch-2",
		CheckID:             "chk-switch2",
		DeviceID:            "switch-2",
		Severity:            "warning",
		Message:             "switch-2 unreachable",
		TriggeredAt:         now.Add(-1 * time.Minute),
		ConsecutiveFailures: 3,
		Suppressed:          true,
		SuppressedBy:        "router-1",
	})

	groups, err := ps.GetCorrelatedAlerts(ctx)
	if err != nil {
		t.Fatalf("GetCorrelatedAlerts: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}

	group := groups[0]
	if group.ParentAlert.ID != "alert-router-1" {
		t.Errorf("parent alert ID = %q, want %q", group.ParentAlert.ID, "alert-router-1")
	}
	if len(group.SuppressedChildren) != 2 {
		t.Errorf("got %d suppressed children, want 2", len(group.SuppressedChildren))
	}
}

func TestGetParentActiveAlerts_ReturnsParentAlerts(t *testing.T) {
	ps := correlationTestStore(t)
	ctx := context.Background()

	insertTestDevice(t, ps, "router-1", "router", "")
	insertTestDevice(t, ps, "switch-1", "switch", "router-1")

	insertCorrelationCheck(t, ps, "chk-router", "router-1")

	now := time.Now().UTC()
	insertTestAlert(t, ps, &Alert{
		ID:                  "alert-router-1",
		CheckID:             "chk-router",
		DeviceID:            "router-1",
		Severity:            "critical",
		Message:             "router down",
		TriggeredAt:         now.Add(-1 * time.Minute),
		ConsecutiveFailures: 5,
	})

	alerts, parentID, err := ps.GetParentActiveAlerts(ctx, "switch-1", 5*time.Minute)
	if err != nil {
		t.Fatalf("GetParentActiveAlerts: %v", err)
	}
	if parentID != "router-1" {
		t.Errorf("parentID = %q, want %q", parentID, "router-1")
	}
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].ID != "alert-router-1" {
		t.Errorf("alert ID = %q, want %q", alerts[0].ID, "alert-router-1")
	}
}

func TestAlerter_CorrelationIntegration(t *testing.T) {
	ps := correlationTestStore(t)
	bus := &mockEventBus{}
	threshold := 3
	alerter := NewAlerter(ps, bus, threshold, zap.NewNop())

	// Enable correlation.
	corr := NewCorrelationEngine(ps, 5*time.Minute, zap.NewNop())
	alerter.SetCorrelation(corr)

	ctx := context.Background()

	// Setup: router -> switch.
	insertTestDevice(t, ps, "router-1", "router", "")
	insertTestDevice(t, ps, "switch-1", "switch", "router-1")

	routerCheck := makeCorrelationCheck(t, ps, "router-1", "ping", "10.0.0.1")
	switchCheck := makeCorrelationCheck(t, ps, "switch-1", "ping", "10.0.0.2")

	failResult := func(check Check) *CheckResult {
		return &CheckResult{
			CheckID:      check.ID,
			DeviceID:     check.DeviceID,
			Success:      false,
			ErrorMessage: "timeout",
			CheckedAt:    time.Now().UTC(),
		}
	}

	// Trigger alert on router first.
	for i := 0; i < threshold; i++ {
		alerter.ProcessResult(ctx, routerCheck, failResult(routerCheck))
	}

	// Verify router alert exists and is not suppressed.
	routerAlert, err := ps.GetActiveAlert(ctx, routerCheck.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert router: %v", err)
	}
	if routerAlert == nil {
		t.Fatal("router alert should exist")
	}
	if routerAlert.Suppressed {
		t.Error("router alert should not be suppressed")
	}

	// Reset bus events.
	bus.events = nil

	// Now trigger alert on switch (child).
	for i := 0; i < threshold; i++ {
		alerter.ProcessResult(ctx, switchCheck, failResult(switchCheck))
	}

	// Switch alert should be suppressed due to parent correlation.
	switchAlert, err := ps.GetActiveAlert(ctx, switchCheck.ID)
	if err != nil {
		t.Fatalf("GetActiveAlert switch: %v", err)
	}
	if switchAlert == nil {
		t.Fatal("switch alert should exist")
	}
	if !switchAlert.Suppressed {
		t.Error("switch alert should be suppressed via correlation")
	}
	if switchAlert.SuppressedBy != "router-1" {
		t.Errorf("SuppressedBy = %q, want %q", switchAlert.SuppressedBy, "router-1")
	}

	// Should publish suppressed event, not triggered.
	var suppressedCount int
	for _, e := range bus.events {
		if e.Topic == TopicAlertSuppressed {
			suppressedCount++
		}
	}
	if suppressedCount != 1 {
		t.Errorf("got %d suppressed events, want 1", suppressedCount)
	}
}

// insertCorrelationCheck inserts a check for correlation tests.
func insertCorrelationCheck(t *testing.T, ps *PulseStore, checkID, deviceID string) {
	t.Helper()
	now := time.Now().UTC()
	if err := ps.InsertCheck(context.Background(), &Check{
		ID:              checkID,
		DeviceID:        deviceID,
		CheckType:       "icmp",
		Target:          "10.0.0.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("insert check %s: %v", checkID, err)
	}
}

// makeCorrelationCheck inserts a check with a unique ID and returns it.
func makeCorrelationCheck(t *testing.T, ps *PulseStore, deviceID, checkType, target string) Check {
	t.Helper()
	now := time.Now().UTC()
	check := Check{
		ID:              fmt.Sprintf("corr-check-%s-%s", deviceID, checkType),
		DeviceID:        deviceID,
		CheckType:       checkType,
		Target:          target,
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := ps.InsertCheck(context.Background(), &check); err != nil {
		t.Fatalf("insert check: %v", err)
	}
	return check
}
