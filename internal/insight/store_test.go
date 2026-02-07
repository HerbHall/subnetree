package insight

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/analytics"
)

func testStore(t *testing.T) *InsightStore {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "insight", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewInsightStore(db.DB())
}

// -- Baselines --

func TestUpsertBaseline_CreateAndRetrieve(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	b := &analytics.Baseline{
		DeviceID:   "dev-001",
		MetricName: "cpu_usage",
		Algorithm:  "ewma",
		Mean:       45.5,
		StdDev:     3.2,
		Samples:    100,
		Stable:     true,
		UpdatedAt:  now,
	}

	if err := s.UpsertBaseline(ctx, b); err != nil {
		t.Fatalf("UpsertBaseline: %v", err)
	}

	baselines, err := s.GetBaselines(ctx, "dev-001")
	if err != nil {
		t.Fatalf("GetBaselines: %v", err)
	}
	if len(baselines) != 1 {
		t.Fatalf("expected 1 baseline, got %d", len(baselines))
	}

	got := baselines[0]
	if got.DeviceID != "dev-001" {
		t.Errorf("DeviceID = %q, want %q", got.DeviceID, "dev-001")
	}
	if got.MetricName != "cpu_usage" {
		t.Errorf("MetricName = %q, want %q", got.MetricName, "cpu_usage")
	}
	if got.Algorithm != "ewma" {
		t.Errorf("Algorithm = %q, want %q", got.Algorithm, "ewma")
	}
	if got.Mean != 45.5 {
		t.Errorf("Mean = %f, want %f", got.Mean, 45.5)
	}
	if got.StdDev != 3.2 {
		t.Errorf("StdDev = %f, want %f", got.StdDev, 3.2)
	}
	if got.Samples != 100 {
		t.Errorf("Samples = %d, want %d", got.Samples, 100)
	}
	if !got.Stable {
		t.Errorf("Stable = false, want true")
	}
}

func TestUpsertBaseline_Update(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	b := &analytics.Baseline{
		DeviceID:   "dev-001",
		MetricName: "cpu_usage",
		Algorithm:  "ewma",
		Mean:       45.5,
		StdDev:     3.2,
		Samples:    100,
		Stable:     false,
		UpdatedAt:  now,
	}
	if err := s.UpsertBaseline(ctx, b); err != nil {
		t.Fatalf("UpsertBaseline (initial): %v", err)
	}

	// Update with new values.
	b.Mean = 50.0
	b.StdDev = 2.1
	b.Samples = 200
	b.Stable = true
	b.UpdatedAt = now.Add(time.Hour)
	if err := s.UpsertBaseline(ctx, b); err != nil {
		t.Fatalf("UpsertBaseline (update): %v", err)
	}

	baselines, err := s.GetBaselines(ctx, "dev-001")
	if err != nil {
		t.Fatalf("GetBaselines: %v", err)
	}
	if len(baselines) != 1 {
		t.Fatalf("expected 1 baseline after upsert, got %d", len(baselines))
	}

	got := baselines[0]
	if got.Mean != 50.0 {
		t.Errorf("Mean = %f, want %f", got.Mean, 50.0)
	}
	if got.StdDev != 2.1 {
		t.Errorf("StdDev = %f, want %f", got.StdDev, 2.1)
	}
	if got.Samples != 200 {
		t.Errorf("Samples = %d, want %d", got.Samples, 200)
	}
	if !got.Stable {
		t.Errorf("Stable = false, want true")
	}
}

func TestGetBaselines_EmptyResults(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	baselines, err := s.GetBaselines(ctx, "nonexistent-device")
	if err != nil {
		t.Fatalf("GetBaselines: %v", err)
	}
	if baselines != nil {
		t.Errorf("expected nil slice for nonexistent device, got %v", baselines)
	}
}

// -- Anomalies --

func TestInsertAnomaly_AndList(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	a := &analytics.Anomaly{
		ID:          "anom-001",
		DeviceID:    "dev-001",
		MetricName:  "cpu_usage",
		Severity:    "warning",
		Type:        "zscore",
		Value:       95.3,
		Expected:    45.5,
		Deviation:   15.6,
		Description: "CPU spike detected",
		DetectedAt:  now,
	}

	if err := s.InsertAnomaly(ctx, a); err != nil {
		t.Fatalf("InsertAnomaly: %v", err)
	}

	anomalies, err := s.ListAnomalies(ctx, "", 50)
	if err != nil {
		t.Fatalf("ListAnomalies: %v", err)
	}
	if len(anomalies) != 1 {
		t.Fatalf("expected 1 anomaly, got %d", len(anomalies))
	}

	got := anomalies[0]
	if got.ID != "anom-001" {
		t.Errorf("ID = %q, want %q", got.ID, "anom-001")
	}
	if got.DeviceID != "dev-001" {
		t.Errorf("DeviceID = %q, want %q", got.DeviceID, "dev-001")
	}
	if got.MetricName != "cpu_usage" {
		t.Errorf("MetricName = %q, want %q", got.MetricName, "cpu_usage")
	}
	if got.Severity != "warning" {
		t.Errorf("Severity = %q, want %q", got.Severity, "warning")
	}
	if got.Type != "zscore" {
		t.Errorf("Type = %q, want %q", got.Type, "zscore")
	}
	if got.Value != 95.3 {
		t.Errorf("Value = %f, want %f", got.Value, 95.3)
	}
	if got.Expected != 45.5 {
		t.Errorf("Expected = %f, want %f", got.Expected, 45.5)
	}
	if got.Deviation != 15.6 {
		t.Errorf("Deviation = %f, want %f", got.Deviation, 15.6)
	}
	if got.Description != "CPU spike detected" {
		t.Errorf("Description = %q, want %q", got.Description, "CPU spike detected")
	}
	if got.ResolvedAt != nil {
		t.Errorf("ResolvedAt = %v, want nil", got.ResolvedAt)
	}
}

func TestListAnomalies_FilterByDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert anomaly for device A.
	a1 := &analytics.Anomaly{
		ID:         "anom-001",
		DeviceID:   "dev-A",
		MetricName: "cpu_usage",
		Severity:   "warning",
		Type:       "zscore",
		Value:      90.0,
		Expected:   50.0,
		Deviation:  8.0,
		DetectedAt: now,
	}
	if err := s.InsertAnomaly(ctx, a1); err != nil {
		t.Fatalf("InsertAnomaly a1: %v", err)
	}

	// Insert anomaly for device B.
	a2 := &analytics.Anomaly{
		ID:         "anom-002",
		DeviceID:   "dev-B",
		MetricName: "mem_usage",
		Severity:   "critical",
		Type:       "cusum",
		Value:      99.0,
		Expected:   60.0,
		Deviation:  13.0,
		DetectedAt: now.Add(time.Second),
	}
	if err := s.InsertAnomaly(ctx, a2); err != nil {
		t.Fatalf("InsertAnomaly a2: %v", err)
	}

	// List for device A only.
	anomalies, err := s.ListAnomalies(ctx, "dev-A", 50)
	if err != nil {
		t.Fatalf("ListAnomalies: %v", err)
	}
	if len(anomalies) != 1 {
		t.Fatalf("expected 1 anomaly for dev-A, got %d", len(anomalies))
	}
	if anomalies[0].ID != "anom-001" {
		t.Errorf("ID = %q, want %q", anomalies[0].ID, "anom-001")
	}

	// List all (empty device filter).
	all, err := s.ListAnomalies(ctx, "", 50)
	if err != nil {
		t.Fatalf("ListAnomalies (all): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 anomalies total, got %d", len(all))
	}
}

func TestResolveAnomaly(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	a := &analytics.Anomaly{
		ID:         "anom-001",
		DeviceID:   "dev-001",
		MetricName: "cpu_usage",
		Severity:   "warning",
		Type:       "zscore",
		Value:      95.0,
		Expected:   50.0,
		Deviation:  9.0,
		DetectedAt: now,
	}
	if err := s.InsertAnomaly(ctx, a); err != nil {
		t.Fatalf("InsertAnomaly: %v", err)
	}

	resolvedAt := now.Add(30 * time.Minute)
	if err := s.ResolveAnomaly(ctx, "anom-001", resolvedAt); err != nil {
		t.Fatalf("ResolveAnomaly: %v", err)
	}

	anomalies, err := s.ListAnomalies(ctx, "dev-001", 50)
	if err != nil {
		t.Fatalf("ListAnomalies: %v", err)
	}
	if len(anomalies) != 1 {
		t.Fatalf("expected 1 anomaly, got %d", len(anomalies))
	}
	if anomalies[0].ResolvedAt == nil {
		t.Fatal("ResolvedAt = nil, want non-nil after resolve")
	}
}

func TestDeleteOldAnomalies(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Old resolved anomaly (48 hours ago).
	oldResolved := now.Add(-48 * time.Hour)
	a1 := &analytics.Anomaly{
		ID:         "anom-old",
		DeviceID:   "dev-001",
		MetricName: "cpu_usage",
		Severity:   "warning",
		Type:       "zscore",
		Value:      90.0,
		Expected:   50.0,
		Deviation:  8.0,
		DetectedAt: now.Add(-72 * time.Hour),
		ResolvedAt: &oldResolved,
	}
	if err := s.InsertAnomaly(ctx, a1); err != nil {
		t.Fatalf("InsertAnomaly (old): %v", err)
	}

	// Recent resolved anomaly (1 hour ago).
	recentResolved := now.Add(-1 * time.Hour)
	a2 := &analytics.Anomaly{
		ID:         "anom-recent",
		DeviceID:   "dev-001",
		MetricName: "mem_usage",
		Severity:   "info",
		Type:       "zscore",
		Value:      80.0,
		Expected:   60.0,
		Deviation:  4.0,
		DetectedAt: now.Add(-2 * time.Hour),
		ResolvedAt: &recentResolved,
	}
	if err := s.InsertAnomaly(ctx, a2); err != nil {
		t.Fatalf("InsertAnomaly (recent): %v", err)
	}

	// Delete anomalies resolved before 24 hours ago.
	cutoff := now.Add(-24 * time.Hour)
	deleted, err := s.DeleteOldAnomalies(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOldAnomalies: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify only the recent anomaly remains.
	remaining, err := s.ListAnomalies(ctx, "", 50)
	if err != nil {
		t.Fatalf("ListAnomalies: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining anomaly, got %d", len(remaining))
	}
	if remaining[0].ID != "anom-recent" {
		t.Errorf("remaining ID = %q, want %q", remaining[0].ID, "anom-recent")
	}
}

// -- Forecasts --

func TestUpsertForecast_CreateAndRetrieve(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	ttl := 72 * time.Hour
	f := &analytics.Forecast{
		DeviceID:        "dev-001",
		MetricName:      "disk_usage",
		CurrentValue:    65.0,
		PredictedValue:  90.0,
		TimeToThreshold: &ttl,
		Threshold:       95.0,
		Slope:           0.35,
		Confidence:      0.87,
		GeneratedAt:     now,
	}

	if err := s.UpsertForecast(ctx, f); err != nil {
		t.Fatalf("UpsertForecast: %v", err)
	}

	forecasts, err := s.GetForecasts(ctx, "dev-001")
	if err != nil {
		t.Fatalf("GetForecasts: %v", err)
	}
	if len(forecasts) != 1 {
		t.Fatalf("expected 1 forecast, got %d", len(forecasts))
	}

	got := forecasts[0]
	if got.DeviceID != "dev-001" {
		t.Errorf("DeviceID = %q, want %q", got.DeviceID, "dev-001")
	}
	if got.MetricName != "disk_usage" {
		t.Errorf("MetricName = %q, want %q", got.MetricName, "disk_usage")
	}
	if got.CurrentValue != 65.0 {
		t.Errorf("CurrentValue = %f, want %f", got.CurrentValue, 65.0)
	}
	if got.PredictedValue != 90.0 {
		t.Errorf("PredictedValue = %f, want %f", got.PredictedValue, 90.0)
	}
	if got.Threshold != 95.0 {
		t.Errorf("Threshold = %f, want %f", got.Threshold, 95.0)
	}
	if got.Slope != 0.35 {
		t.Errorf("Slope = %f, want %f", got.Slope, 0.35)
	}
	if got.Confidence != 0.87 {
		t.Errorf("Confidence = %f, want %f", got.Confidence, 0.87)
	}
	if got.TimeToThreshold == nil {
		t.Fatal("TimeToThreshold = nil, want non-nil")
	}
	// Duration is stored as seconds, so compare in seconds.
	wantSecs := int64(ttl.Seconds())
	gotSecs := int64(got.TimeToThreshold.Seconds())
	if gotSecs != wantSecs {
		t.Errorf("TimeToThreshold = %v (%d s), want %v (%d s)", *got.TimeToThreshold, gotSecs, ttl, wantSecs)
	}
}

func TestUpsertForecast_NilTimeToThreshold(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	f := &analytics.Forecast{
		DeviceID:       "dev-001",
		MetricName:     "disk_usage",
		CurrentValue:   30.0,
		PredictedValue: 35.0,
		Threshold:      95.0,
		Slope:          0.01,
		Confidence:     0.5,
		GeneratedAt:    now,
	}

	if err := s.UpsertForecast(ctx, f); err != nil {
		t.Fatalf("UpsertForecast: %v", err)
	}

	forecasts, err := s.GetForecasts(ctx, "dev-001")
	if err != nil {
		t.Fatalf("GetForecasts: %v", err)
	}
	if len(forecasts) != 1 {
		t.Fatalf("expected 1 forecast, got %d", len(forecasts))
	}
	if forecasts[0].TimeToThreshold != nil {
		t.Errorf("TimeToThreshold = %v, want nil", forecasts[0].TimeToThreshold)
	}
}

// -- Correlations --

func TestInsertCorrelation_AndList(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	g := &analytics.AlertGroup{
		ID:          "corr-001",
		RootCause:   "dev-switch-01",
		DeviceIDs:   []string{"dev-001", "dev-002", "dev-003"},
		AlertCount:  5,
		Description: "Network switch failure affecting downstream devices",
		CreatedAt:   now,
	}

	if err := s.InsertCorrelation(ctx, g); err != nil {
		t.Fatalf("InsertCorrelation: %v", err)
	}

	groups, err := s.ListActiveCorrelations(ctx)
	if err != nil {
		t.Fatalf("ListActiveCorrelations: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 correlation group, got %d", len(groups))
	}

	got := groups[0]
	if got.ID != "corr-001" {
		t.Errorf("ID = %q, want %q", got.ID, "corr-001")
	}
	if got.RootCause != "dev-switch-01" {
		t.Errorf("RootCause = %q, want %q", got.RootCause, "dev-switch-01")
	}
	if got.AlertCount != 5 {
		t.Errorf("AlertCount = %d, want %d", got.AlertCount, 5)
	}
	if got.Description != "Network switch failure affecting downstream devices" {
		t.Errorf("Description = %q, want %q", got.Description, "Network switch failure affecting downstream devices")
	}
	if len(got.DeviceIDs) != 3 {
		t.Fatalf("DeviceIDs length = %d, want 3", len(got.DeviceIDs))
	}
	if got.DeviceIDs[0] != "dev-001" {
		t.Errorf("DeviceIDs[0] = %q, want %q", got.DeviceIDs[0], "dev-001")
	}
	if got.DeviceIDs[1] != "dev-002" {
		t.Errorf("DeviceIDs[1] = %q, want %q", got.DeviceIDs[1], "dev-002")
	}
	if got.DeviceIDs[2] != "dev-003" {
		t.Errorf("DeviceIDs[2] = %q, want %q", got.DeviceIDs[2], "dev-003")
	}
}

// -- Metrics --

func TestInsertMetric_AndWindow(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert 3 metric points spread across time.
	points := []analytics.MetricPoint{
		{
			DeviceID:   "dev-001",
			MetricName: "cpu_usage",
			Value:      45.0,
			Timestamp:  now.Add(-2 * time.Hour),
			Tags:       map[string]string{"core": "0"},
		},
		{
			DeviceID:   "dev-001",
			MetricName: "cpu_usage",
			Value:      55.0,
			Timestamp:  now.Add(-1 * time.Hour),
			Tags:       map[string]string{"core": "0"},
		},
		{
			DeviceID:   "dev-001",
			MetricName: "cpu_usage",
			Value:      65.0,
			Timestamp:  now,
			Tags:       map[string]string{"core": "0"},
		},
	}

	for i := range points {
		if err := s.InsertMetric(ctx, &points[i]); err != nil {
			t.Fatalf("InsertMetric[%d]: %v", i, err)
		}
	}

	// Retrieve window from 90 minutes ago -- should get 2 points.
	since := now.Add(-90 * time.Minute)
	result, err := s.GetMetricWindow(ctx, "dev-001", "cpu_usage", since)
	if err != nil {
		t.Fatalf("GetMetricWindow: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 metric points, got %d", len(result))
	}

	// Verify ascending order by timestamp.
	if result[0].Value != 55.0 {
		t.Errorf("result[0].Value = %f, want %f", result[0].Value, 55.0)
	}
	if result[1].Value != 65.0 {
		t.Errorf("result[1].Value = %f, want %f", result[1].Value, 65.0)
	}

	// Verify tags are preserved.
	if result[0].Tags["core"] != "0" {
		t.Errorf("result[0].Tags[\"core\"] = %q, want %q", result[0].Tags["core"], "0")
	}
}

func TestDeleteOldMetrics(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Insert old metric (48 hours ago).
	old := &analytics.MetricPoint{
		DeviceID:   "dev-001",
		MetricName: "cpu_usage",
		Value:      40.0,
		Timestamp:  now.Add(-48 * time.Hour),
		Tags:       map[string]string{},
	}
	if err := s.InsertMetric(ctx, old); err != nil {
		t.Fatalf("InsertMetric (old): %v", err)
	}

	// Insert recent metric (1 hour ago).
	recent := &analytics.MetricPoint{
		DeviceID:   "dev-001",
		MetricName: "cpu_usage",
		Value:      60.0,
		Timestamp:  now.Add(-1 * time.Hour),
		Tags:       map[string]string{},
	}
	if err := s.InsertMetric(ctx, recent); err != nil {
		t.Fatalf("InsertMetric (recent): %v", err)
	}

	// Delete metrics older than 24 hours.
	cutoff := now.Add(-24 * time.Hour)
	deleted, err := s.DeleteOldMetrics(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOldMetrics: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify only the recent metric remains.
	remaining, err := s.GetMetricWindow(ctx, "dev-001", "cpu_usage", now.Add(-72*time.Hour))
	if err != nil {
		t.Fatalf("GetMetricWindow: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining metric, got %d", len(remaining))
	}
	if remaining[0].Value != 60.0 {
		t.Errorf("remaining Value = %f, want %f", remaining[0].Value, 60.0)
	}
}
