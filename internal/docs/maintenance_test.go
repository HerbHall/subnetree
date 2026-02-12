package docs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMaintenanceWorker_RetentionCleanup(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Insert an application.
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(ctx, app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	// Insert old snapshot (past retention).
	oldSnap := &Snapshot{
		ID: "snap-old", ApplicationID: "app-001", ContentHash: "h1",
		Content: "old config", Format: "text", SizeBytes: 10,
		Source: "manual", CapturedAt: now.Add(-200 * 24 * time.Hour),
	}
	if err := m.store.InsertSnapshot(ctx, oldSnap); err != nil {
		t.Fatalf("insert old snapshot: %v", err)
	}

	// Insert recent snapshot (within retention).
	recentSnap := &Snapshot{
		ID: "snap-recent", ApplicationID: "app-001", ContentHash: "h2",
		Content: "new config", Format: "text", SizeBytes: 10,
		Source: "manual", CapturedAt: now,
	}
	if err := m.store.InsertSnapshot(ctx, recentSnap); err != nil {
		t.Fatalf("insert recent snapshot: %v", err)
	}

	worker := NewMaintenanceWorker(m.store, m.cfg, zap.NewNop())
	worker.runCleanup(ctx)

	// Old snapshot should be deleted.
	snap, err := m.store.GetSnapshot(ctx, "snap-old")
	if err != nil {
		t.Fatalf("get old snapshot: %v", err)
	}
	if snap != nil {
		t.Error("old snapshot should have been deleted")
	}

	// Recent snapshot should remain.
	snap, err = m.store.GetSnapshot(ctx, "snap-recent")
	if err != nil {
		t.Fatalf("get recent snapshot: %v", err)
	}
	if snap == nil {
		t.Error("recent snapshot should still exist")
	}
}

func TestMaintenanceWorker_MaxSnapshotsEnforcement(t *testing.T) {
	m := newTestModule(t)
	m.cfg.MaxSnapshotsPerApp = 3
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// Insert an application.
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	if err := m.store.InsertApplication(ctx, app); err != nil {
		t.Fatalf("insert application: %v", err)
	}

	// Insert 5 snapshots (all recent so retention doesn't delete them).
	for i := 0; i < 5; i++ {
		snap := &Snapshot{
			ID:            fmt.Sprintf("snap-%03d", i),
			ApplicationID: "app-001",
			ContentHash:   fmt.Sprintf("hash-%d", i),
			Content:       fmt.Sprintf("config %d", i),
			Format:        "text",
			SizeBytes:     8,
			Source:        "manual",
			CapturedAt:    now.Add(time.Duration(i) * time.Minute),
		}
		if err := m.store.InsertSnapshot(ctx, snap); err != nil {
			t.Fatalf("insert snapshot %d: %v", i, err)
		}
	}

	count, err := m.store.CountSnapshots(ctx, "app-001")
	if err != nil {
		t.Fatalf("count before cleanup: %v", err)
	}
	if count != 5 {
		t.Fatalf("count before cleanup = %d, want 5", count)
	}

	worker := NewMaintenanceWorker(m.store, m.cfg, zap.NewNop())
	worker.runCleanup(ctx)

	count, err = m.store.CountSnapshots(ctx, "app-001")
	if err != nil {
		t.Fatalf("count after cleanup: %v", err)
	}
	if count != 3 {
		t.Errorf("count after cleanup = %d, want 3", count)
	}

	// Verify the 3 most recent snapshots are kept.
	for i := 2; i < 5; i++ {
		snap, err := m.store.GetSnapshot(ctx, fmt.Sprintf("snap-%03d", i))
		if err != nil {
			t.Fatalf("get snapshot %d: %v", i, err)
		}
		if snap == nil {
			t.Errorf("snap-%03d should still exist (was recent)", i)
		}
	}

	// Verify the 2 oldest snapshots are deleted.
	for i := 0; i < 2; i++ {
		snap, err := m.store.GetSnapshot(ctx, fmt.Sprintf("snap-%03d", i))
		if err != nil {
			t.Fatalf("get snapshot %d: %v", i, err)
		}
		if snap != nil {
			t.Errorf("snap-%03d should have been deleted (was oldest)", i)
		}
	}
}

func TestMaintenanceWorker_NilStore(t *testing.T) {
	worker := NewMaintenanceWorker(nil, DefaultConfig(), zap.NewNop())
	// Should not panic.
	worker.runCleanup(context.Background())
}
