package pulse

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
)

func testStore(t *testing.T) *PulseStore {
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
	return NewPulseStore(db.DB())
}

// insertTestCheck is a helper that inserts a check and fails the test on error.
func insertTestCheck(t *testing.T, s *PulseStore, c *Check) {
	t.Helper()
	if err := s.InsertCheck(context.Background(), c); err != nil {
		t.Fatalf("InsertCheck: %v", err)
	}
}

// -- Checks --

func TestInsertCheck_AndGetCheck(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	c := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c)

	got, err := s.GetCheck(ctx, "chk-001")
	if err != nil {
		t.Fatalf("GetCheck: %v", err)
	}
	if got == nil {
		t.Fatal("GetCheck returned nil, want non-nil")
	}
	if got.ID != "chk-001" {
		t.Errorf("ID = %q, want %q", got.ID, "chk-001")
	}
	if got.DeviceID != "dev-001" {
		t.Errorf("DeviceID = %q, want %q", got.DeviceID, "dev-001")
	}
	if got.CheckType != "icmp" {
		t.Errorf("CheckType = %q, want %q", got.CheckType, "icmp")
	}
	if got.Target != "192.168.1.1" {
		t.Errorf("Target = %q, want %q", got.Target, "192.168.1.1")
	}
	if got.IntervalSeconds != 30 {
		t.Errorf("IntervalSeconds = %d, want %d", got.IntervalSeconds, 30)
	}
	if !got.Enabled {
		t.Errorf("Enabled = false, want true")
	}
}

func TestGetCheck_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetCheck(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetCheck: %v", err)
	}
	if got != nil {
		t.Errorf("GetCheck = %+v, want nil", got)
	}
}

func TestGetCheckByDeviceID(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	c := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-ABC",
		CheckType:       "tcp",
		Target:          "10.0.0.1:443",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c)

	got, err := s.GetCheckByDeviceID(ctx, "dev-ABC")
	if err != nil {
		t.Fatalf("GetCheckByDeviceID: %v", err)
	}
	if got == nil {
		t.Fatal("GetCheckByDeviceID returned nil, want non-nil")
	}
	if got.ID != "chk-001" {
		t.Errorf("ID = %q, want %q", got.ID, "chk-001")
	}
	if got.DeviceID != "dev-ABC" {
		t.Errorf("DeviceID = %q, want %q", got.DeviceID, "dev-ABC")
	}
	if got.CheckType != "tcp" {
		t.Errorf("CheckType = %q, want %q", got.CheckType, "tcp")
	}
	if got.Target != "10.0.0.1:443" {
		t.Errorf("Target = %q, want %q", got.Target, "10.0.0.1:443")
	}
	if got.IntervalSeconds != 60 {
		t.Errorf("IntervalSeconds = %d, want %d", got.IntervalSeconds, 60)
	}
}

func TestGetCheckByDeviceID_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetCheckByDeviceID(ctx, "nonexistent-device")
	if err != nil {
		t.Fatalf("GetCheckByDeviceID: %v", err)
	}
	if got != nil {
		t.Errorf("GetCheckByDeviceID = %+v, want nil", got)
	}
}

func TestListEnabledChecks(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert an enabled check.
	enabled := &Check{
		ID:              "chk-enabled",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, enabled)

	// Insert a disabled check.
	disabled := &Check{
		ID:              "chk-disabled",
		DeviceID:        "dev-002",
		CheckType:       "tcp",
		Target:          "192.168.1.2:22",
		IntervalSeconds: 60,
		Enabled:         false,
		CreatedAt:       now.Add(time.Second),
		UpdatedAt:       now.Add(time.Second),
	}
	insertTestCheck(t, s, disabled)

	checks, err := s.ListEnabledChecks(ctx)
	if err != nil {
		t.Fatalf("ListEnabledChecks: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("expected 1 enabled check, got %d", len(checks))
	}
	if checks[0].ID != "chk-enabled" {
		t.Errorf("ID = %q, want %q", checks[0].ID, "chk-enabled")
	}
	if !checks[0].Enabled {
		t.Errorf("Enabled = false, want true")
	}
}

func TestUpdateCheckEnabled(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	c := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c)

	// Disable the check.
	if err := s.UpdateCheckEnabled(ctx, "chk-001", false); err != nil {
		t.Fatalf("UpdateCheckEnabled: %v", err)
	}

	got, err := s.GetCheck(ctx, "chk-001")
	if err != nil {
		t.Fatalf("GetCheck: %v", err)
	}
	if got == nil {
		t.Fatal("GetCheck returned nil after update")
	}
	if got.Enabled {
		t.Errorf("Enabled = true after disabling, want false")
	}

	// Verify it no longer appears in enabled list.
	enabled, err := s.ListEnabledChecks(ctx)
	if err != nil {
		t.Fatalf("ListEnabledChecks: %v", err)
	}
	if len(enabled) != 0 {
		t.Errorf("expected 0 enabled checks after disabling, got %d", len(enabled))
	}
}

// -- Results --

func TestInsertResult_AndListResults(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert a check first (results reference checks via check_id).
	c := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c)

	// Insert 3 results with different timestamps.
	r1 := &CheckResult{
		CheckID:      "chk-001",
		DeviceID:     "dev-001",
		Success:      true,
		LatencyMs:    12.5,
		PacketLoss:   0.0,
		ErrorMessage: "",
		CheckedAt:    now.Add(-2 * time.Minute),
	}
	r2 := &CheckResult{
		CheckID:      "chk-001",
		DeviceID:     "dev-001",
		Success:      true,
		LatencyMs:    15.3,
		PacketLoss:   0.0,
		ErrorMessage: "",
		CheckedAt:    now.Add(-1 * time.Minute),
	}
	r3 := &CheckResult{
		CheckID:      "chk-001",
		DeviceID:     "dev-001",
		Success:      false,
		LatencyMs:    0.0,
		PacketLoss:   100.0,
		ErrorMessage: "timeout",
		CheckedAt:    now,
	}

	for i, r := range []*CheckResult{r1, r2, r3} {
		if err := s.InsertResult(ctx, r); err != nil {
			t.Fatalf("InsertResult[%d]: %v", i, err)
		}
	}

	// List with limit 2 -- should return the 2 most recent in DESC order.
	results, err := s.ListResults(ctx, "dev-001", 2)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First result should be the most recent (r3).
	if results[0].Success {
		t.Errorf("results[0].Success = true, want false")
	}
	if results[0].LatencyMs != 0.0 {
		t.Errorf("results[0].LatencyMs = %f, want %f", results[0].LatencyMs, 0.0)
	}
	if results[0].PacketLoss != 100.0 {
		t.Errorf("results[0].PacketLoss = %f, want %f", results[0].PacketLoss, 100.0)
	}
	if results[0].ErrorMessage != "timeout" {
		t.Errorf("results[0].ErrorMessage = %q, want %q", results[0].ErrorMessage, "timeout")
	}

	// Second result should be r2.
	if !results[1].Success {
		t.Errorf("results[1].Success = false, want true")
	}
	if results[1].LatencyMs != 15.3 {
		t.Errorf("results[1].LatencyMs = %f, want %f", results[1].LatencyMs, 15.3)
	}
}

func TestListResults_DefaultLimit(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert a check.
	c := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c)

	// Insert a single result.
	r := &CheckResult{
		CheckID:   "chk-001",
		DeviceID:  "dev-001",
		Success:   true,
		LatencyMs: 10.0,
		CheckedAt: now,
	}
	if err := s.InsertResult(ctx, r); err != nil {
		t.Fatalf("InsertResult: %v", err)
	}

	// Pass 0 for limit -- should default to 100 and still return the result.
	results, err := s.ListResults(ctx, "dev-001", 0)
	if err != nil {
		t.Fatalf("ListResults with limit 0: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result with default limit, got %d", len(results))
	}
	if results[0].LatencyMs != 10.0 {
		t.Errorf("LatencyMs = %f, want %f", results[0].LatencyMs, 10.0)
	}
}

func TestDeleteOldResults(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert a check.
	c := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c)

	// Insert an old result (48 hours ago).
	old := &CheckResult{
		CheckID:   "chk-001",
		DeviceID:  "dev-001",
		Success:   true,
		LatencyMs: 10.0,
		CheckedAt: now.Add(-48 * time.Hour),
	}
	if err := s.InsertResult(ctx, old); err != nil {
		t.Fatalf("InsertResult (old): %v", err)
	}

	// Insert a recent result (1 hour ago).
	recent := &CheckResult{
		CheckID:   "chk-001",
		DeviceID:  "dev-001",
		Success:   true,
		LatencyMs: 12.0,
		CheckedAt: now.Add(-1 * time.Hour),
	}
	if err := s.InsertResult(ctx, recent); err != nil {
		t.Fatalf("InsertResult (recent): %v", err)
	}

	// Delete results older than 24 hours.
	cutoff := now.Add(-24 * time.Hour)
	deleted, err := s.DeleteOldResults(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOldResults: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify only the recent result remains.
	remaining, err := s.ListResults(ctx, "dev-001", 100)
	if err != nil {
		t.Fatalf("ListResults: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining result, got %d", len(remaining))
	}
	if remaining[0].LatencyMs != 12.0 {
		t.Errorf("remaining LatencyMs = %f, want %f", remaining[0].LatencyMs, 12.0)
	}
}

// -- Alerts --

func TestInsertAlert_AndGetActive(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert a check (alerts reference checks via check_id).
	c := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c)

	a := &Alert{
		ID:                  "alert-001",
		CheckID:             "chk-001",
		DeviceID:            "dev-001",
		Severity:            "critical",
		Message:             "Host unreachable for 5 consecutive checks",
		TriggeredAt:         now,
		ResolvedAt:          nil,
		ConsecutiveFailures: 5,
	}
	if err := s.InsertAlert(ctx, a); err != nil {
		t.Fatalf("InsertAlert: %v", err)
	}

	got, err := s.GetActiveAlert(ctx, "chk-001")
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if got == nil {
		t.Fatal("GetActiveAlert returned nil, want non-nil")
	}
	if got.ID != "alert-001" {
		t.Errorf("ID = %q, want %q", got.ID, "alert-001")
	}
	if got.CheckID != "chk-001" {
		t.Errorf("CheckID = %q, want %q", got.CheckID, "chk-001")
	}
	if got.DeviceID != "dev-001" {
		t.Errorf("DeviceID = %q, want %q", got.DeviceID, "dev-001")
	}
	if got.Severity != "critical" {
		t.Errorf("Severity = %q, want %q", got.Severity, "critical")
	}
	if got.Message != "Host unreachable for 5 consecutive checks" {
		t.Errorf("Message = %q, want %q", got.Message, "Host unreachable for 5 consecutive checks")
	}
	if got.ConsecutiveFailures != 5 {
		t.Errorf("ConsecutiveFailures = %d, want %d", got.ConsecutiveFailures, 5)
	}
	if got.ResolvedAt != nil {
		t.Errorf("ResolvedAt = %v, want nil", got.ResolvedAt)
	}
}

func TestGetActiveAlert_None(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetActiveAlert(ctx, "nonexistent-check")
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if got != nil {
		t.Errorf("GetActiveAlert = %+v, want nil", got)
	}
}

func TestResolveAlert(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert a check.
	c := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c)

	// Insert an active alert.
	a := &Alert{
		ID:                  "alert-001",
		CheckID:             "chk-001",
		DeviceID:            "dev-001",
		Severity:            "warning",
		Message:             "High latency detected",
		TriggeredAt:         now,
		ConsecutiveFailures: 3,
	}
	if err := s.InsertAlert(ctx, a); err != nil {
		t.Fatalf("InsertAlert: %v", err)
	}

	// Resolve the alert.
	resolvedAt := now.Add(10 * time.Minute)
	if err := s.ResolveAlert(ctx, "alert-001", resolvedAt); err != nil {
		t.Fatalf("ResolveAlert: %v", err)
	}

	// GetActiveAlert should return nil since the alert is now resolved.
	got, err := s.GetActiveAlert(ctx, "chk-001")
	if err != nil {
		t.Fatalf("GetActiveAlert after resolve: %v", err)
	}
	if got != nil {
		t.Errorf("GetActiveAlert after resolve = %+v, want nil", got)
	}
}

func TestListActiveAlerts_AllDevices(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert checks for two devices.
	c1 := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	c2 := &Check{
		ID:              "chk-002",
		DeviceID:        "dev-002",
		CheckType:       "icmp",
		Target:          "192.168.1.2",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c1)
	insertTestCheck(t, s, c2)

	// Insert 2 active alerts on different devices.
	a1 := &Alert{
		ID:                  "alert-001",
		CheckID:             "chk-001",
		DeviceID:            "dev-001",
		Severity:            "critical",
		Message:             "Host down",
		TriggeredAt:         now,
		ConsecutiveFailures: 5,
	}
	a2 := &Alert{
		ID:                  "alert-002",
		CheckID:             "chk-002",
		DeviceID:            "dev-002",
		Severity:            "warning",
		Message:             "High latency",
		TriggeredAt:         now.Add(time.Second),
		ConsecutiveFailures: 3,
	}
	if err := s.InsertAlert(ctx, a1); err != nil {
		t.Fatalf("InsertAlert a1: %v", err)
	}
	if err := s.InsertAlert(ctx, a2); err != nil {
		t.Fatalf("InsertAlert a2: %v", err)
	}

	// Insert a resolved alert (should NOT be returned).
	resolvedAt := now.Add(5 * time.Minute)
	a3 := &Alert{
		ID:                  "alert-003",
		CheckID:             "chk-001",
		DeviceID:            "dev-001",
		Severity:            "info",
		Message:             "Packet loss spike",
		TriggeredAt:         now.Add(-10 * time.Minute),
		ResolvedAt:          &resolvedAt,
		ConsecutiveFailures: 2,
	}
	if err := s.InsertAlert(ctx, a3); err != nil {
		t.Fatalf("InsertAlert a3: %v", err)
	}

	// List all active alerts (empty deviceID).
	alerts, err := s.ListActiveAlerts(ctx, "")
	if err != nil {
		t.Fatalf("ListActiveAlerts: %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 active alerts, got %d", len(alerts))
	}

	// Verify DESC order by triggered_at (a2 is more recent).
	if alerts[0].ID != "alert-002" {
		t.Errorf("alerts[0].ID = %q, want %q", alerts[0].ID, "alert-002")
	}
	if alerts[1].ID != "alert-001" {
		t.Errorf("alerts[1].ID = %q, want %q", alerts[1].ID, "alert-001")
	}
}

func TestListActiveAlerts_FilterByDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert checks for two devices.
	c1 := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	c2 := &Check{
		ID:              "chk-002",
		DeviceID:        "dev-002",
		CheckType:       "icmp",
		Target:          "192.168.1.2",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c1)
	insertTestCheck(t, s, c2)

	// Insert active alerts on different devices.
	a1 := &Alert{
		ID:                  "alert-001",
		CheckID:             "chk-001",
		DeviceID:            "dev-001",
		Severity:            "critical",
		Message:             "Host down",
		TriggeredAt:         now,
		ConsecutiveFailures: 5,
	}
	a2 := &Alert{
		ID:                  "alert-002",
		CheckID:             "chk-002",
		DeviceID:            "dev-002",
		Severity:            "warning",
		Message:             "High latency",
		TriggeredAt:         now.Add(time.Second),
		ConsecutiveFailures: 3,
	}
	if err := s.InsertAlert(ctx, a1); err != nil {
		t.Fatalf("InsertAlert a1: %v", err)
	}
	if err := s.InsertAlert(ctx, a2); err != nil {
		t.Fatalf("InsertAlert a2: %v", err)
	}

	// Filter by dev-001 only.
	alerts, err := s.ListActiveAlerts(ctx, "dev-001")
	if err != nil {
		t.Fatalf("ListActiveAlerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 active alert for dev-001, got %d", len(alerts))
	}
	if alerts[0].ID != "alert-001" {
		t.Errorf("ID = %q, want %q", alerts[0].ID, "alert-001")
	}
	if alerts[0].DeviceID != "dev-001" {
		t.Errorf("DeviceID = %q, want %q", alerts[0].DeviceID, "dev-001")
	}
}

func TestDeleteOldAlerts(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert checks.
	c1 := &Check{
		ID:              "chk-001",
		DeviceID:        "dev-001",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	c2 := &Check{
		ID:              "chk-002",
		DeviceID:        "dev-002",
		CheckType:       "icmp",
		Target:          "192.168.1.2",
		IntervalSeconds: 30,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	insertTestCheck(t, s, c1)
	insertTestCheck(t, s, c2)

	// 1. Old resolved alert (resolved 48 hours ago) -- should be deleted.
	oldResolvedAt := now.Add(-48 * time.Hour)
	oldResolved := &Alert{
		ID:                  "alert-old-resolved",
		CheckID:             "chk-001",
		DeviceID:            "dev-001",
		Severity:            "warning",
		Message:             "Old resolved alert",
		TriggeredAt:         now.Add(-72 * time.Hour),
		ResolvedAt:          &oldResolvedAt,
		ConsecutiveFailures: 2,
	}
	if err := s.InsertAlert(ctx, oldResolved); err != nil {
		t.Fatalf("InsertAlert (old resolved): %v", err)
	}

	// 2. Recent resolved alert (resolved 1 hour ago) -- should NOT be deleted.
	recentResolvedAt := now.Add(-1 * time.Hour)
	recentResolved := &Alert{
		ID:                  "alert-recent-resolved",
		CheckID:             "chk-001",
		DeviceID:            "dev-001",
		Severity:            "info",
		Message:             "Recent resolved alert",
		TriggeredAt:         now.Add(-2 * time.Hour),
		ResolvedAt:          &recentResolvedAt,
		ConsecutiveFailures: 1,
	}
	if err := s.InsertAlert(ctx, recentResolved); err != nil {
		t.Fatalf("InsertAlert (recent resolved): %v", err)
	}

	// 3. Active (unresolved) alert triggered 72 hours ago -- should NOT be deleted
	//    because DeleteOldAlerts only deletes resolved alerts.
	activeOld := &Alert{
		ID:                  "alert-active-old",
		CheckID:             "chk-002",
		DeviceID:            "dev-002",
		Severity:            "critical",
		Message:             "Active old alert",
		TriggeredAt:         now.Add(-72 * time.Hour),
		ConsecutiveFailures: 10,
	}
	if err := s.InsertAlert(ctx, activeOld); err != nil {
		t.Fatalf("InsertAlert (active old): %v", err)
	}

	// Delete resolved alerts older than 24 hours.
	cutoff := now.Add(-24 * time.Hour)
	deleted, err := s.DeleteOldAlerts(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOldAlerts: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify the active old alert is still present.
	activeAlert, err := s.GetActiveAlert(ctx, "chk-002")
	if err != nil {
		t.Fatalf("GetActiveAlert: %v", err)
	}
	if activeAlert == nil {
		t.Fatal("active old alert was deleted, but should have been preserved")
	}
	if activeAlert.ID != "alert-active-old" {
		t.Errorf("active alert ID = %q, want %q", activeAlert.ID, "alert-active-old")
	}

	// Verify the recent resolved alert is still present (list all active = only the active one).
	// We can't use ListActiveAlerts to check resolved alerts, so list all for dev-001
	// via a direct query approach: list active for dev-001 should be empty (both were resolved).
	dev1Active, err := s.ListActiveAlerts(ctx, "dev-001")
	if err != nil {
		t.Fatalf("ListActiveAlerts dev-001: %v", err)
	}
	if len(dev1Active) != 0 {
		t.Errorf("expected 0 active alerts for dev-001, got %d", len(dev1Active))
	}

	// The active old alert for dev-002 should still be listed.
	dev2Active, err := s.ListActiveAlerts(ctx, "dev-002")
	if err != nil {
		t.Fatalf("ListActiveAlerts dev-002: %v", err)
	}
	if len(dev2Active) != 1 {
		t.Fatalf("expected 1 active alert for dev-002, got %d", len(dev2Active))
	}
	if dev2Active[0].ID != "alert-active-old" {
		t.Errorf("dev-002 active alert ID = %q, want %q", dev2Active[0].ID, "alert-active-old")
	}
}
