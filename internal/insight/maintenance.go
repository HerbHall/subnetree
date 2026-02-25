package insight

import (
	"context"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/pkg/analytics"
	"go.uber.org/zap"
)

// startMaintenance launches a background goroutine that periodically:
// 1. Persists in-memory baselines to the database.
// 2. Deletes old resolved anomalies past the retention window.
// 3. Deletes old metric data past the retention window.
func (m *Module) startMaintenance() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.cfg.MaintenanceInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.runMaintenance()
			}
		}
	}()
}

// runMaintenance executes a single maintenance cycle.
func (m *Module) runMaintenance() {
	if m.store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	// Persist baselines
	m.persistBaselines(ctx)

	// Delete old anomalies
	cutoff := time.Now().Add(-m.cfg.AnomalyRetention)
	deleted, err := m.store.DeleteOldAnomalies(ctx, cutoff)
	if err != nil {
		m.logger.Warn("failed to delete old anomalies", zap.Error(err))
	} else if deleted > 0 {
		m.logger.Info("purged old anomalies", zap.Int64("count", deleted))
	}

	// Delete old metrics (keep 2x forecast window for regression)
	metricCutoff := time.Now().Add(-2 * m.cfg.ForecastWindow)
	deletedMetrics, err := m.store.DeleteOldMetrics(ctx, metricCutoff)
	if err != nil {
		m.logger.Warn("failed to delete old metrics", zap.Error(err))
	} else if deletedMetrics > 0 {
		m.logger.Info("purged old metrics", zap.Int64("count", deletedMetrics))
	}
}

// persistBaselines writes all in-memory baseline states to the database.
// Persists both EWMA and Holt-Winters baselines when the HW model is initialized.
func (m *Module) persistBaselines(ctx context.Context) {
	snap := m.states.snapshot()
	persisted := 0
	now := time.Now()
	for key, state := range snap {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		deviceID, metric := parts[0], parts[1]
		ewma := state.EWMA

		// Persist EWMA baseline
		b := &analytics.Baseline{
			DeviceID:   deviceID,
			MetricName: metric,
			Algorithm:  "ewma",
			Mean:       ewma.Mean,
			StdDev:     ewma.StdDev(),
			Samples:    ewma.Samples,
			Stable:     ewma.Samples >= m.cfg.MinSamplesStable,
			UpdatedAt:  now,
		}
		if err := m.store.UpsertBaseline(ctx, b); err != nil {
			m.logger.Warn("failed to persist baseline",
				zap.String("device_id", deviceID),
				zap.String("metric", metric),
				zap.String("algorithm", "ewma"),
				zap.Error(err),
			)
			continue
		}
		persisted++

		// Persist Holt-Winters baseline when the model has enough data
		hw := state.HoltWinters
		if hw.IsInitialized() && hw.Samples >= 2*m.cfg.HWSeasonLen {
			hwBaseline := &analytics.Baseline{
				DeviceID:   deviceID,
				MetricName: metric + ":hw",
				Algorithm:  "holt_winters",
				Mean:       hw.Level,
				StdDev:     hw.ResidualStdDev(),
				Samples:    hw.Samples,
				Stable:     hw.Samples >= 2*m.cfg.HWSeasonLen,
				UpdatedAt:  now,
			}
			if err := m.store.UpsertBaseline(ctx, hwBaseline); err != nil {
				m.logger.Warn("failed to persist baseline",
					zap.String("device_id", deviceID),
					zap.String("metric", metric),
					zap.String("algorithm", "holt_winters"),
					zap.Error(err),
				)
				continue
			}
			persisted++
		}
	}
	if persisted > 0 {
		m.logger.Debug("persisted baselines", zap.Int("count", persisted))
	}
}
