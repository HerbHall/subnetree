package recon

import (
	"context"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	pruneRetentionDays = 30
	checkInterval      = 1 * time.Hour
)

// ScanConsolidator rolls up raw scan metrics into weekly and monthly aggregates
// and prunes raw metrics older than the retention period.
type ScanConsolidator struct {
	store  *ReconStore
	logger *zap.Logger
}

// NewScanConsolidator creates a new ScanConsolidator.
func NewScanConsolidator(store *ReconStore, logger *zap.Logger) *ScanConsolidator {
	return &ScanConsolidator{
		store:  store,
		logger: logger,
	}
}

// Run starts the consolidation loop, checking every hour whether it's time
// to consolidate. Weekly consolidation runs on Mondays at 03:00 UTC.
func (c *ScanConsolidator) Run(ctx context.Context) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	c.logger.Info("scan consolidator started",
		zap.Duration("check_interval", checkInterval),
		zap.Int("retention_days", pruneRetentionDays),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("scan consolidator stopped")
			return
		case now := <-ticker.C:
			now = now.UTC()
			if now.Weekday() == time.Monday && now.Hour() == 3 {
				c.runConsolidation(ctx, now)
			}
		}
	}
}

// RunOnce performs a single consolidation pass. Exposed for testing.
func (c *ScanConsolidator) RunOnce(ctx context.Context, now time.Time) {
	c.runConsolidation(ctx, now.UTC())
}

func (c *ScanConsolidator) runConsolidation(ctx context.Context, now time.Time) {
	c.logger.Info("starting metrics consolidation")

	if err := c.consolidateWeekly(ctx, now); err != nil {
		c.logger.Error("weekly consolidation failed", zap.Error(err))
	}

	if err := c.consolidateMonthly(ctx, now); err != nil {
		c.logger.Error("monthly consolidation failed", zap.Error(err))
	}

	if err := c.pruneOldMetrics(ctx, now); err != nil {
		c.logger.Error("metrics pruning failed", zap.Error(err))
	}

	c.logger.Info("metrics consolidation complete")
}

// consolidateWeekly aggregates raw metrics from the previous week (Monday to Sunday).
func (c *ScanConsolidator) consolidateWeekly(ctx context.Context, now time.Time) error {
	weekEnd := startOfWeek(now)
	weekStart := weekEnd.AddDate(0, 0, -7)

	raw, err := c.store.GetRawMetricsInRange(ctx, weekStart, weekEnd)
	if err != nil {
		return err
	}

	if len(raw) == 0 {
		c.logger.Debug("no raw metrics for weekly consolidation",
			zap.Time("week_start", weekStart),
			zap.Time("week_end", weekEnd),
		)
		return nil
	}

	agg := c.aggregateRawMetrics(raw, "weekly", weekStart, weekEnd)
	if err := c.store.SaveScanMetricsAggregate(ctx, agg); err != nil {
		return err
	}

	c.logger.Info("weekly aggregate saved",
		zap.String("period_start", agg.PeriodStart),
		zap.Int("scan_count", agg.ScanCount),
	)
	return nil
}

// consolidateMonthly aggregates weekly aggregates from the previous month.
func (c *ScanConsolidator) consolidateMonthly(ctx context.Context, now time.Time) error {
	monthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Only run monthly consolidation at the start of a new month (first week).
	if now.Day() > 7 {
		return nil
	}

	weeklies, err := c.store.GetWeeklyAggregatesInRange(ctx, monthStart, monthEnd)
	if err != nil {
		return err
	}

	if len(weeklies) == 0 {
		c.logger.Debug("no weekly aggregates for monthly consolidation",
			zap.Time("month_start", monthStart),
			zap.Time("month_end", monthEnd),
		)
		return nil
	}

	agg := c.aggregateWeeklyAggregates(weeklies, monthStart, monthEnd)
	if err := c.store.SaveScanMetricsAggregate(ctx, agg); err != nil {
		return err
	}

	c.logger.Info("monthly aggregate saved",
		zap.String("period_start", agg.PeriodStart),
		zap.Int("scan_count", agg.ScanCount),
	)
	return nil
}

// pruneOldMetrics removes raw scan metrics older than the retention period.
func (c *ScanConsolidator) pruneOldMetrics(ctx context.Context, now time.Time) error {
	cutoff := now.AddDate(0, 0, -pruneRetentionDays)
	pruned, err := c.store.PruneMetricsBefore(ctx, cutoff)
	if err != nil {
		return err
	}

	if pruned > 0 {
		c.logger.Info("pruned old scan metrics",
			zap.Int64("deleted", pruned),
			zap.Time("cutoff", cutoff),
		)
	}
	return nil
}

// aggregateRawMetrics computes an aggregate from a slice of raw scan metrics.
func (c *ScanConsolidator) aggregateRawMetrics(raw []models.ScanMetrics, period string, periodStart, periodEnd time.Time) *ScanMetricsAggregate {
	agg := &ScanMetricsAggregate{
		ID:          uuid.New().String(),
		Period:      period,
		PeriodStart: periodStart.Format(time.RFC3339),
		PeriodEnd:   periodEnd.Format(time.RFC3339),
		ScanCount:   len(raw),
	}

	var (
		totalDuration  float64
		totalPing      float64
		totalEnrich    float64
		totalDevices   float64
		totalAlive     float64
		totalNewDevs   int
		maxDevs        int
		minDevs        = -1
		failedCount    int
	)

	for i := range raw {
		m := &raw[i]
		totalDuration += float64(m.DurationMs)
		totalPing += float64(m.PingPhaseMs)
		totalEnrich += float64(m.EnrichPhaseMs)

		devicesFound := m.DevicesCreated + m.DevicesUpdated
		totalDevices += float64(devicesFound)
		totalAlive += float64(m.HostsAlive)
		totalNewDevs += m.DevicesCreated

		if devicesFound > maxDevs {
			maxDevs = devicesFound
		}
		if minDevs < 0 || devicesFound < minDevs {
			minDevs = devicesFound
		}

		if m.DurationMs == 0 && m.HostsScanned == 0 {
			failedCount++
		}
	}

	if minDevs < 0 {
		minDevs = 0
	}

	count := float64(agg.ScanCount)
	agg.AvgDurationMs = totalDuration / count
	agg.AvgPingPhaseMs = totalPing / count
	agg.AvgEnrichMs = totalEnrich / count
	agg.AvgDevicesFound = totalDevices / count
	agg.AvgHostsAlive = totalAlive / count
	agg.MaxDevicesFound = maxDevs
	agg.MinDevicesFound = minDevs
	agg.TotalNewDevices = totalNewDevs
	agg.FailedScans = failedCount

	return agg
}

// aggregateWeeklyAggregates rolls up weekly aggregates into a monthly aggregate.
func (c *ScanConsolidator) aggregateWeeklyAggregates(weeklies []ScanMetricsAggregate, monthStart, monthEnd time.Time) *ScanMetricsAggregate {
	agg := &ScanMetricsAggregate{
		ID:          uuid.New().String(),
		Period:      "monthly",
		PeriodStart: monthStart.Format(time.RFC3339),
		PeriodEnd:   monthEnd.Format(time.RFC3339),
	}

	var (
		totalScans     int
		weightedDur    float64
		weightedPing   float64
		weightedEnrich float64
		weightedDevs   float64
		weightedAlive  float64
		totalNewDevs   int
		totalFailed    int
		maxDevs        int
		minDevs        = -1
	)

	for i := range weeklies {
		w := &weeklies[i]
		n := w.ScanCount
		totalScans += n

		weight := float64(n)
		weightedDur += w.AvgDurationMs * weight
		weightedPing += w.AvgPingPhaseMs * weight
		weightedEnrich += w.AvgEnrichMs * weight
		weightedDevs += w.AvgDevicesFound * weight
		weightedAlive += w.AvgHostsAlive * weight
		totalNewDevs += w.TotalNewDevices
		totalFailed += w.FailedScans

		if w.MaxDevicesFound > maxDevs {
			maxDevs = w.MaxDevicesFound
		}
		if minDevs < 0 || w.MinDevicesFound < minDevs {
			minDevs = w.MinDevicesFound
		}
	}

	if minDevs < 0 {
		minDevs = 0
	}

	agg.ScanCount = totalScans
	if totalScans > 0 {
		total := float64(totalScans)
		agg.AvgDurationMs = weightedDur / total
		agg.AvgPingPhaseMs = weightedPing / total
		agg.AvgEnrichMs = weightedEnrich / total
		agg.AvgDevicesFound = weightedDevs / total
		agg.AvgHostsAlive = weightedAlive / total
	}
	agg.MaxDevicesFound = maxDevs
	agg.MinDevicesFound = minDevs
	agg.TotalNewDevices = totalNewDevs
	agg.FailedScans = totalFailed

	return agg
}

// startOfWeek returns midnight UTC of the Monday at or before the given time.
func startOfWeek(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysBack := weekday - 1
	monday := t.AddDate(0, 0, -daysBack)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}
