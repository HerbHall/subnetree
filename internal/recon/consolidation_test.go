package recon

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// insertTestScanWithMetrics creates a parent scan record and associated metrics.
func insertTestScanWithMetrics(t *testing.T, s *ReconStore, ctx context.Context, m *models.ScanMetrics) {
	t.Helper()
	scan := &models.ScanResult{Subnet: "10.0.0.0/24", Status: "completed"}
	if err := s.CreateScan(ctx, scan); err != nil {
		t.Fatalf("CreateScan: %v", err)
	}
	m.ScanID = scan.ID
	if err := s.SaveScanMetrics(ctx, m); err != nil {
		t.Fatalf("SaveScanMetrics: %v", err)
	}
}

func TestConsolidateWeekly(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	logger := zap.NewNop()
	c := NewScanConsolidator(s, logger)

	// Use a fixed reference Monday.
	now := time.Date(2026, 2, 16, 3, 0, 0, 0, time.UTC) // Monday 2026-02-16

	// Insert raw metrics in the previous week (Mon 2/9 to Sun 2/15).
	weekStart := time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC)
	for i := range 5 {
		ts := weekStart.Add(time.Duration(i) * 24 * time.Hour).Format(time.RFC3339)
		insertTestScanWithMetrics(t, s, ctx, &models.ScanMetrics{
			DurationMs:     1000 + int64(i*100),
			PingPhaseMs:    500 + int64(i*50),
			EnrichPhaseMs:  200 + int64(i*20),
			PostProcessMs:  100,
			HostsScanned:   254,
			HostsAlive:     10 + i,
			DevicesCreated: 2,
			DevicesUpdated: 8 + i,
			CreatedAt:      ts,
		})
	}

	if err := c.consolidateWeekly(ctx, now); err != nil {
		t.Fatalf("consolidateWeekly: %v", err)
	}

	aggs, err := s.ListScanMetricsAggregates(ctx, "weekly", 10)
	if err != nil {
		t.Fatalf("ListScanMetricsAggregates: %v", err)
	}
	if len(aggs) != 1 {
		t.Fatalf("expected 1 weekly aggregate, got %d", len(aggs))
	}

	agg := aggs[0]
	if agg.ScanCount != 5 {
		t.Errorf("ScanCount = %d, want 5", agg.ScanCount)
	}
	if agg.TotalNewDevices != 10 {
		t.Errorf("TotalNewDevices = %d, want 10", agg.TotalNewDevices)
	}
	if agg.Period != "weekly" {
		t.Errorf("Period = %q, want weekly", agg.Period)
	}

	// Verify averages are reasonable.
	expectedAvgDuration := (1000.0 + 1100.0 + 1200.0 + 1300.0 + 1400.0) / 5.0
	if math.Abs(agg.AvgDurationMs-expectedAvgDuration) > 0.01 {
		t.Errorf("AvgDurationMs = %f, want %f", agg.AvgDurationMs, expectedAvgDuration)
	}

	// MaxDevicesFound = max(2+8, 2+9, 2+10, 2+11, 2+12) = 14
	if agg.MaxDevicesFound != 14 {
		t.Errorf("MaxDevicesFound = %d, want 14", agg.MaxDevicesFound)
	}
	// MinDevicesFound = min(10, 11, 12, 13, 14) = 10
	if agg.MinDevicesFound != 10 {
		t.Errorf("MinDevicesFound = %d, want 10", agg.MinDevicesFound)
	}
}

func TestConsolidateMonthly(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	logger := zap.NewNop()
	c := NewScanConsolidator(s, logger)

	// Insert weekly aggregates for January 2026.
	janWeek1 := &ScanMetricsAggregate{
		Period:          "weekly",
		PeriodStart:     time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		PeriodEnd:       time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		ScanCount:       10,
		AvgDurationMs:   1200,
		AvgPingPhaseMs:  600,
		AvgEnrichMs:     300,
		AvgDevicesFound: 12,
		MaxDevicesFound: 15,
		MinDevicesFound: 8,
		AvgHostsAlive:   20,
		TotalNewDevices: 5,
		FailedScans:     1,
	}
	if err := s.SaveScanMetricsAggregate(ctx, janWeek1); err != nil {
		t.Fatalf("save week1: %v", err)
	}

	janWeek2 := &ScanMetricsAggregate{
		Period:          "weekly",
		PeriodStart:     time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		PeriodEnd:       time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		ScanCount:       14,
		AvgDurationMs:   1400,
		AvgPingPhaseMs:  700,
		AvgEnrichMs:     350,
		AvgDevicesFound: 14,
		MaxDevicesFound: 20,
		MinDevicesFound: 10,
		AvgHostsAlive:   25,
		TotalNewDevices: 8,
		FailedScans:     0,
	}
	if err := s.SaveScanMetricsAggregate(ctx, janWeek2); err != nil {
		t.Fatalf("save week2: %v", err)
	}

	// Run monthly consolidation from early February (day <= 7).
	now := time.Date(2026, 2, 2, 3, 0, 0, 0, time.UTC)
	if err := c.consolidateMonthly(ctx, now); err != nil {
		t.Fatalf("consolidateMonthly: %v", err)
	}

	aggs, err := s.ListScanMetricsAggregates(ctx, "monthly", 10)
	if err != nil {
		t.Fatalf("ListScanMetricsAggregates: %v", err)
	}
	if len(aggs) != 1 {
		t.Fatalf("expected 1 monthly aggregate, got %d", len(aggs))
	}

	agg := aggs[0]
	if agg.ScanCount != 24 {
		t.Errorf("ScanCount = %d, want 24", agg.ScanCount)
	}
	if agg.TotalNewDevices != 13 {
		t.Errorf("TotalNewDevices = %d, want 13", agg.TotalNewDevices)
	}
	if agg.MaxDevicesFound != 20 {
		t.Errorf("MaxDevicesFound = %d, want 20", agg.MaxDevicesFound)
	}
	if agg.MinDevicesFound != 8 {
		t.Errorf("MinDevicesFound = %d, want 8", agg.MinDevicesFound)
	}
	if agg.FailedScans != 1 {
		t.Errorf("FailedScans = %d, want 1", agg.FailedScans)
	}

	// Weighted average: (1200*10 + 1400*14) / 24 = (12000 + 19600) / 24 = 1316.67
	expectedAvg := (1200.0*10.0 + 1400.0*14.0) / 24.0
	if math.Abs(agg.AvgDurationMs-expectedAvg) > 0.01 {
		t.Errorf("AvgDurationMs = %f, want %f", agg.AvgDurationMs, expectedAvg)
	}
}

func TestPruneOldMetrics(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	logger := zap.NewNop()
	c := NewScanConsolidator(s, logger)

	now := time.Date(2026, 3, 15, 3, 0, 0, 0, time.UTC)

	// Insert an old metric (45 days ago) -- should be pruned.
	oldTS := now.AddDate(0, 0, -45).Format(time.RFC3339)
	insertTestScanWithMetrics(t, s, ctx, &models.ScanMetrics{
		DurationMs:    1000,
		PingPhaseMs:   500,
		EnrichPhaseMs: 200,
		PostProcessMs: 100,
		HostsScanned:  254,
		HostsAlive:    10,
		CreatedAt:     oldTS,
	})

	// Insert a recent metric (5 days ago) -- should be kept.
	recentTS := now.AddDate(0, 0, -5).Format(time.RFC3339)
	insertTestScanWithMetrics(t, s, ctx, &models.ScanMetrics{
		DurationMs:    2000,
		PingPhaseMs:   800,
		EnrichPhaseMs: 400,
		PostProcessMs: 200,
		HostsScanned:  254,
		HostsAlive:    15,
		CreatedAt:     recentTS,
	})

	if err := c.pruneOldMetrics(ctx, now); err != nil {
		t.Fatalf("pruneOldMetrics: %v", err)
	}

	// Verify only the recent metric remains.
	cutoff := now.AddDate(0, 0, -60)
	remaining, err := s.GetRawMetricsSince(ctx, cutoff)
	if err != nil {
		t.Fatalf("GetRawMetricsSince: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining metric, got %d", len(remaining))
	}
	if remaining[0].DurationMs != 2000 {
		t.Errorf("remaining DurationMs = %d, want 2000", remaining[0].DurationMs)
	}
}

func TestConsolidationIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	logger := zap.NewNop()
	c := NewScanConsolidator(s, logger)

	now := time.Date(2026, 2, 16, 3, 0, 0, 0, time.UTC)
	weekStart := time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC)

	// Insert raw metrics in the previous week.
	for i := range 3 {
		ts := weekStart.Add(time.Duration(i) * 24 * time.Hour).Format(time.RFC3339)
		insertTestScanWithMetrics(t, s, ctx, &models.ScanMetrics{
			DurationMs:     1000,
			PingPhaseMs:    500,
			EnrichPhaseMs:  200,
			PostProcessMs:  100,
			HostsScanned:   254,
			HostsAlive:     10,
			DevicesCreated: 1,
			DevicesUpdated: 5,
			CreatedAt:      ts,
		})
	}

	// Run consolidation twice.
	if err := c.consolidateWeekly(ctx, now); err != nil {
		t.Fatalf("first consolidateWeekly: %v", err)
	}
	if err := c.consolidateWeekly(ctx, now); err != nil {
		t.Fatalf("second consolidateWeekly: %v", err)
	}

	// Verify only one aggregate exists (INSERT OR IGNORE).
	aggs, err := s.ListScanMetricsAggregates(ctx, "weekly", 10)
	if err != nil {
		t.Fatalf("ListScanMetricsAggregates: %v", err)
	}
	if len(aggs) != 1 {
		t.Fatalf("expected 1 aggregate after idempotent runs, got %d", len(aggs))
	}
}

func TestStartOfWeek(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want time.Time
	}{
		{
			name: "Monday stays Monday",
			in:   time.Date(2026, 2, 16, 15, 30, 0, 0, time.UTC),
			want: time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "Wednesday goes back to Monday",
			in:   time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC),
			want: time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "Sunday goes back to Monday",
			in:   time.Date(2026, 2, 22, 23, 59, 0, 0, time.UTC),
			want: time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "Saturday goes back to Monday",
			in:   time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC),
			want: time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := startOfWeek(tt.in)
			if !got.Equal(tt.want) {
				t.Errorf("startOfWeek(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
