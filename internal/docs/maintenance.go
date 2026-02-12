package docs

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// MaintenanceWorker periodically cleans up old snapshots based on retention
// policy and enforces the max snapshots per application limit.
type MaintenanceWorker struct {
	store  *DocsStore
	cfg    Config
	logger *zap.Logger
}

// NewMaintenanceWorker creates a new MaintenanceWorker.
func NewMaintenanceWorker(store *DocsStore, cfg Config, logger *zap.Logger) *MaintenanceWorker {
	return &MaintenanceWorker{
		store:  store,
		cfg:    cfg,
		logger: logger,
	}
}

// Run starts the maintenance loop, running cleanup on every tick of the
// maintenance interval. It blocks until ctx is cancelled.
func (w *MaintenanceWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.MaintenanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runCleanup(ctx)
		}
	}
}

// runCleanup executes a single maintenance cycle: deletes snapshots older than
// the retention period and enforces max_snapshots_per_app.
func (w *MaintenanceWorker) runCleanup(ctx context.Context) {
	if w.store == nil {
		return
	}

	cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Delete snapshots past the retention period.
	cutoff := time.Now().Add(-w.cfg.RetentionPeriod)
	deleted, err := w.store.DeleteOldSnapshots(cleanupCtx, cutoff)
	if err != nil {
		w.logger.Warn("failed to delete old snapshots", zap.Error(err))
	} else if deleted > 0 {
		w.logger.Info("purged expired snapshots", zap.Int64("count", deleted))
	}

	// Enforce max snapshots per application.
	w.enforceMaxSnapshots(cleanupCtx)
}

// enforceMaxSnapshots trims each application's snapshots to the configured maximum.
func (w *MaintenanceWorker) enforceMaxSnapshots(ctx context.Context) {
	apps, _, err := w.store.ListApplications(ctx, ListApplicationsParams{Limit: 1000})
	if err != nil {
		w.logger.Warn("failed to list applications for snapshot cap", zap.Error(err))
		return
	}

	for i := range apps {
		deleted, err := w.store.DeleteExcessSnapshots(ctx, apps[i].ID, w.cfg.MaxSnapshotsPerApp)
		if err != nil {
			w.logger.Warn("failed to cap snapshots",
				zap.String("application_id", apps[i].ID),
				zap.Error(err),
			)
			continue
		}
		if deleted > 0 {
			w.logger.Info("capped snapshots for application",
				zap.String("application_id", apps[i].ID),
				zap.Int64("deleted", deleted),
			)
		}
	}
}
