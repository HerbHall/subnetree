package docs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CollectionResult summarises a collection run.
type CollectionResult struct {
	AppsDiscovered   int      `json:"apps_discovered"`
	SnapshotsCreated int      `json:"snapshots_created"`
	Errors           []string `json:"errors"`
}

// RunCollection runs all available collectors, upserting discovered
// applications and creating snapshots when configuration content has changed.
func (m *Module) RunCollection(ctx context.Context) *CollectionResult {
	return m.runCollectors(ctx, m.collectors)
}

// RunCollectorByName runs collection for a single named collector.
func (m *Module) RunCollectorByName(ctx context.Context, name string) (*CollectionResult, error) {
	for _, c := range m.collectors {
		if c.Name() == name {
			return m.runCollectors(ctx, []Collector{c}), nil
		}
	}
	return nil, fmt.Errorf("collector %q not found", name)
}

// runCollectors executes the collection loop for the given set of collectors.
func (m *Module) runCollectors(ctx context.Context, collectors []Collector) *CollectionResult {
	result := &CollectionResult{Errors: []string{}}

	for _, c := range collectors {
		if !c.Available() {
			m.logger.Debug("collector not available, skipping", zap.String("collector", c.Name()))
			continue
		}

		apps, err := c.Discover(ctx)
		if err != nil {
			msg := fmt.Sprintf("%s: discover: %v", c.Name(), err)
			m.logger.Warn("collector discover failed", zap.String("collector", c.Name()), zap.Error(err))
			result.Errors = append(result.Errors, msg)
			continue
		}

		result.AppsDiscovered += len(apps)

		for i := range apps {
			app := &apps[i]

			if m.store != nil {
				if err := m.store.UpsertApplication(ctx, app); err != nil {
					msg := fmt.Sprintf("%s: upsert app %s: %v", c.Name(), app.ID, err)
					m.logger.Warn("upsert application failed", zap.Error(err))
					result.Errors = append(result.Errors, msg)
					continue
				}
			}

			cfg, err := c.Collect(ctx, app.ID)
			if err != nil {
				msg := fmt.Sprintf("%s: collect %s: %v", c.Name(), app.ID, err)
				m.logger.Warn("collect failed", zap.String("app_id", app.ID), zap.Error(err))
				result.Errors = append(result.Errors, msg)
				continue
			}

			hash := sha256.Sum256([]byte(cfg.Content))
			contentHash := hex.EncodeToString(hash[:])

			if m.store != nil {
				latest, err := m.store.GetLatestSnapshot(ctx, app.ID)
				if err != nil {
					msg := fmt.Sprintf("%s: get latest snapshot for %s: %v", c.Name(), app.ID, err)
					m.logger.Warn("get latest snapshot failed", zap.Error(err))
					result.Errors = append(result.Errors, msg)
					continue
				}

				// Skip if content hasn't changed.
				if latest != nil && latest.ContentHash == contentHash {
					continue
				}

				snap := &Snapshot{
					ID:            uuid.New().String(),
					ApplicationID: app.ID,
					ContentHash:   contentHash,
					Content:       cfg.Content,
					Format:        cfg.Format,
					SizeBytes:     len(cfg.Content),
					Source:        c.Name(),
					CapturedAt:    time.Now().UTC(),
				}

				if err := m.store.InsertSnapshot(ctx, snap); err != nil {
					msg := fmt.Sprintf("%s: insert snapshot for %s: %v", c.Name(), app.ID, err)
					m.logger.Warn("insert snapshot failed", zap.Error(err))
					result.Errors = append(result.Errors, msg)
					continue
				}

				result.SnapshotsCreated++

				m.publishEvent(ctx, TopicSnapshotCreated, snap)
			}
		}
	}

	return result
}
