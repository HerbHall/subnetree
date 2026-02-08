package pulse

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// startMaintenance launches a background goroutine that periodically
// deletes old check results and resolved alerts past the retention window.
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

	cutoff := time.Now().Add(-m.cfg.RetentionPeriod)

	// Purge old check results.
	deletedResults, err := m.store.DeleteOldResults(ctx, cutoff)
	if err != nil {
		m.logger.Warn("failed to delete old results", zap.Error(err))
	} else if deletedResults > 0 {
		m.logger.Info("purged old check results", zap.Int64("count", deletedResults))
	}

	// Purge old resolved alerts.
	deletedAlerts, err := m.store.DeleteOldAlerts(ctx, cutoff)
	if err != nil {
		m.logger.Warn("failed to delete old alerts", zap.Error(err))
	} else if deletedAlerts > 0 {
		m.logger.Info("purged old resolved alerts", zap.Int64("count", deletedAlerts))
	}
}
