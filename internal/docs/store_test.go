package docs

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
)

// testStore creates an in-memory SQLite database, runs docs migrations,
// and returns a DocsStore ready for testing.
func testStore(t *testing.T) *DocsStore {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "docs", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewStore(db.DB())
}

// insertTestApp is a helper that inserts an application and fails the test on error.
func insertTestApp(t *testing.T, s *DocsStore, a *Application) {
	t.Helper()
	if err := s.InsertApplication(context.Background(), a); err != nil {
		t.Fatalf("InsertApplication: %v", err)
	}
}

// insertTestSnap is a helper that inserts a snapshot and fails the test on error.
func insertTestSnap(t *testing.T, s *DocsStore, snap *Snapshot) {
	t.Helper()
	if err := s.InsertSnapshot(context.Background(), snap); err != nil {
		t.Fatalf("InsertSnapshot: %v", err)
	}
}

// -- Applications --

func TestInsertAndGetApplication(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID:           "app-001",
		Name:         "Proxmox VE",
		AppType:      "hypervisor",
		DeviceID:     "dev-001",
		Collector:    "docker",
		Status:       "active",
		Metadata:     `{"version":"8.0"}`,
		DiscoveredAt: now,
		UpdatedAt:    now,
	}
	insertTestApp(t, s, app)

	got, err := s.GetApplication(ctx, "app-001")
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if got == nil {
		t.Fatal("GetApplication returned nil, want non-nil")
	}
	if got.ID != "app-001" {
		t.Errorf("ID = %q, want %q", got.ID, "app-001")
	}
	if got.Name != "Proxmox VE" {
		t.Errorf("Name = %q, want %q", got.Name, "Proxmox VE")
	}
	if got.AppType != "hypervisor" {
		t.Errorf("AppType = %q, want %q", got.AppType, "hypervisor")
	}
	if got.DeviceID != "dev-001" {
		t.Errorf("DeviceID = %q, want %q", got.DeviceID, "dev-001")
	}
	if got.Collector != "docker" {
		t.Errorf("Collector = %q, want %q", got.Collector, "docker")
	}
	if got.Status != "active" {
		t.Errorf("Status = %q, want %q", got.Status, "active")
	}
	if got.Metadata != `{"version":"8.0"}` {
		t.Errorf("Metadata = %q, want %q", got.Metadata, `{"version":"8.0"}`)
	}
}

func TestGetApplication_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetApplication(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if got != nil {
		t.Errorf("GetApplication = %+v, want nil", got)
	}
}

func TestListApplications(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	apps := []*Application{
		{
			ID: "app-001", Name: "Proxmox", AppType: "hypervisor",
			Collector: "docker", Status: "active", Metadata: "{}",
			DiscoveredAt: now, UpdatedAt: now,
		},
		{
			ID: "app-002", Name: "Nginx", AppType: "web_server",
			Collector: "docker", Status: "active", Metadata: "{}",
			DiscoveredAt: now, UpdatedAt: now.Add(time.Second),
		},
		{
			ID: "app-003", Name: "Old App", AppType: "hypervisor",
			Collector: "manual", Status: "removed", Metadata: "{}",
			DiscoveredAt: now, UpdatedAt: now.Add(2 * time.Second),
		},
	}
	for _, a := range apps {
		insertTestApp(t, s, a)
	}

	tests := []struct {
		name      string
		params    ListApplicationsParams
		wantCount int
		wantTotal int
	}{
		{
			name:      "all applications",
			params:    ListApplicationsParams{Limit: 50},
			wantCount: 3,
			wantTotal: 3,
		},
		{
			name:      "filter by type hypervisor",
			params:    ListApplicationsParams{Limit: 50, AppType: "hypervisor"},
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name:      "filter by type web_server",
			params:    ListApplicationsParams{Limit: 50, AppType: "web_server"},
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name:      "filter by status active",
			params:    ListApplicationsParams{Limit: 50, Status: "active"},
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name:      "filter by status removed",
			params:    ListApplicationsParams{Limit: 50, Status: "removed"},
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name:      "filter by type and status",
			params:    ListApplicationsParams{Limit: 50, AppType: "hypervisor", Status: "active"},
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name:      "pagination limit",
			params:    ListApplicationsParams{Limit: 2},
			wantCount: 2,
			wantTotal: 3,
		},
		{
			name:      "pagination offset",
			params:    ListApplicationsParams{Limit: 50, Offset: 2},
			wantCount: 1,
			wantTotal: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := s.ListApplications(ctx, tt.params)
			if err != nil {
				t.Fatalf("ListApplications: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("len(apps) = %d, want %d", len(got), tt.wantCount)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestListApplications_Empty(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, total, err := s.ListApplications(ctx, ListApplicationsParams{Limit: 50})
	if err != nil {
		t.Fatalf("ListApplications: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil slice for empty result, got %v", got)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

func TestListApplications_DefaultLimit(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Test", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app)

	// Pass 0 for limit -- should default to 50 and still return the result.
	got, total, err := s.ListApplications(ctx, ListApplicationsParams{Limit: 0})
	if err != nil {
		t.Fatalf("ListApplications with limit 0: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("len(apps) = %d, want 1", len(got))
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
}

func TestUpdateApplication(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	app := &Application{
		ID: "app-001", Name: "Proxmox", AppType: "hypervisor",
		DeviceID: "dev-001", Collector: "docker", Status: "active",
		Metadata: "{}", DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app)

	// Update some fields.
	app.Name = "Proxmox VE Updated"
	app.Status = "removed"
	app.Metadata = `{"version":"8.1"}`
	app.UpdatedAt = now.Add(time.Minute)

	if err := s.UpdateApplication(ctx, app); err != nil {
		t.Fatalf("UpdateApplication: %v", err)
	}

	got, err := s.GetApplication(ctx, "app-001")
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if got == nil {
		t.Fatal("GetApplication returned nil after update")
	}
	if got.Name != "Proxmox VE Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Proxmox VE Updated")
	}
	if got.Status != "removed" {
		t.Errorf("Status = %q, want %q", got.Status, "removed")
	}
	if got.Metadata != `{"version":"8.1"}` {
		t.Errorf("Metadata = %q, want %q", got.Metadata, `{"version":"8.1"}`)
	}
}

// -- Snapshots --

func TestInsertAndGetSnapshot(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	// Insert application first (foreign key constraint).
	app := &Application{
		ID: "app-001", Name: "Test App", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app)

	snap := &Snapshot{
		ID:            "snap-001",
		ApplicationID: "app-001",
		ContentHash:   "abc123hash",
		Content:       `{"config":"value"}`,
		Format:        "json",
		SizeBytes:     18,
		Source:        "manual",
		CapturedAt:    now,
	}
	insertTestSnap(t, s, snap)

	got, err := s.GetSnapshot(ctx, "snap-001")
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if got == nil {
		t.Fatal("GetSnapshot returned nil, want non-nil")
	}
	if got.ID != "snap-001" {
		t.Errorf("ID = %q, want %q", got.ID, "snap-001")
	}
	if got.ApplicationID != "app-001" {
		t.Errorf("ApplicationID = %q, want %q", got.ApplicationID, "app-001")
	}
	if got.ContentHash != "abc123hash" {
		t.Errorf("ContentHash = %q, want %q", got.ContentHash, "abc123hash")
	}
	if got.Content != `{"config":"value"}` {
		t.Errorf("Content = %q, want %q", got.Content, `{"config":"value"}`)
	}
	if got.Format != "json" {
		t.Errorf("Format = %q, want %q", got.Format, "json")
	}
	if got.SizeBytes != 18 {
		t.Errorf("SizeBytes = %d, want %d", got.SizeBytes, 18)
	}
	if got.Source != "manual" {
		t.Errorf("Source = %q, want %q", got.Source, "manual")
	}
}

func TestGetSnapshot_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetSnapshot(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if got != nil {
		t.Errorf("GetSnapshot = %+v, want nil", got)
	}
}

func TestListSnapshots(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	// Insert two applications.
	app1 := &Application{
		ID: "app-001", Name: "App 1", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	app2 := &Application{
		ID: "app-002", Name: "App 2", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app1)
	insertTestApp(t, s, app2)

	// Insert snapshots: 2 for app-001, 1 for app-002.
	snaps := []*Snapshot{
		{
			ID: "snap-001", ApplicationID: "app-001", ContentHash: "h1",
			Content: "c1", Format: "json", SizeBytes: 2, Source: "auto",
			CapturedAt: now,
		},
		{
			ID: "snap-002", ApplicationID: "app-001", ContentHash: "h2",
			Content: "c2", Format: "json", SizeBytes: 2, Source: "auto",
			CapturedAt: now.Add(time.Second),
		},
		{
			ID: "snap-003", ApplicationID: "app-002", ContentHash: "h3",
			Content: "c3", Format: "yaml", SizeBytes: 2, Source: "manual",
			CapturedAt: now.Add(2 * time.Second),
		},
	}
	for _, sn := range snaps {
		insertTestSnap(t, s, sn)
	}

	tests := []struct {
		name      string
		params    ListSnapshotsParams
		wantCount int
	}{
		{
			name:      "all snapshots",
			params:    ListSnapshotsParams{Limit: 50},
			wantCount: 3,
		},
		{
			name:      "filter by app-001",
			params:    ListSnapshotsParams{ApplicationID: "app-001", Limit: 50},
			wantCount: 2,
		},
		{
			name:      "filter by app-002",
			params:    ListSnapshotsParams{ApplicationID: "app-002", Limit: 50},
			wantCount: 1,
		},
		{
			name:      "pagination limit",
			params:    ListSnapshotsParams{Limit: 2},
			wantCount: 2,
		},
		{
			name:      "pagination offset",
			params:    ListSnapshotsParams{Limit: 50, Offset: 2},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.ListSnapshots(ctx, tt.params)
			if err != nil {
				t.Fatalf("ListSnapshots: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("len(snapshots) = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestListSnapshots_Empty(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.ListSnapshots(ctx, ListSnapshotsParams{Limit: 50})
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil slice for empty result, got %v", got)
	}
}

func TestGetLatestSnapshot(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	app := &Application{
		ID: "app-001", Name: "Test App", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app)

	// Insert multiple snapshots with different timestamps.
	snaps := []*Snapshot{
		{
			ID: "snap-old", ApplicationID: "app-001", ContentHash: "h1",
			Content: "old config", Format: "json", SizeBytes: 10,
			Source: "auto", CapturedAt: now.Add(-2 * time.Hour),
		},
		{
			ID: "snap-mid", ApplicationID: "app-001", ContentHash: "h2",
			Content: "mid config", Format: "json", SizeBytes: 10,
			Source: "auto", CapturedAt: now.Add(-1 * time.Hour),
		},
		{
			ID: "snap-new", ApplicationID: "app-001", ContentHash: "h3",
			Content: "new config", Format: "json", SizeBytes: 10,
			Source: "auto", CapturedAt: now,
		},
	}
	for _, sn := range snaps {
		insertTestSnap(t, s, sn)
	}

	got, err := s.GetLatestSnapshot(ctx, "app-001")
	if err != nil {
		t.Fatalf("GetLatestSnapshot: %v", err)
	}
	if got == nil {
		t.Fatal("GetLatestSnapshot returned nil, want non-nil")
	}
	if got.ID != "snap-new" {
		t.Errorf("ID = %q, want %q", got.ID, "snap-new")
	}
	if got.Content != "new config" {
		t.Errorf("Content = %q, want %q", got.Content, "new config")
	}
}

func TestGetLatestSnapshot_NotFound(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	got, err := s.GetLatestSnapshot(ctx, "nonexistent-app")
	if err != nil {
		t.Fatalf("GetLatestSnapshot: %v", err)
	}
	if got != nil {
		t.Errorf("GetLatestSnapshot = %+v, want nil", got)
	}
}

func TestDeleteSnapshot(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	app := &Application{
		ID: "app-001", Name: "Test App", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app)

	snap := &Snapshot{
		ID: "snap-001", ApplicationID: "app-001", ContentHash: "h1",
		Content: "config", Format: "json", SizeBytes: 6,
		Source: "manual", CapturedAt: now,
	}
	insertTestSnap(t, s, snap)

	// Verify it exists.
	got, err := s.GetSnapshot(ctx, "snap-001")
	if err != nil {
		t.Fatalf("GetSnapshot before delete: %v", err)
	}
	if got == nil {
		t.Fatal("snapshot should exist before deletion")
	}

	// Delete it.
	if err := s.DeleteSnapshot(ctx, "snap-001"); err != nil {
		t.Fatalf("DeleteSnapshot: %v", err)
	}

	// Verify it is gone.
	got, err = s.GetSnapshot(ctx, "snap-001")
	if err != nil {
		t.Fatalf("GetSnapshot after delete: %v", err)
	}
	if got != nil {
		t.Errorf("GetSnapshot after delete = %+v, want nil", got)
	}
}

func TestCountSnapshots(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	app := &Application{
		ID: "app-001", Name: "Test App", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app)

	// Count when empty.
	count, err := s.CountSnapshots(ctx, "app-001")
	if err != nil {
		t.Fatalf("CountSnapshots: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	// Insert 3 snapshots.
	for i := 0; i < 3; i++ {
		snap := &Snapshot{
			ID: "snap-" + string(rune('a'+i)), ApplicationID: "app-001",
			ContentHash: "h", Content: "c", Format: "json", SizeBytes: 1,
			Source: "auto", CapturedAt: now.Add(time.Duration(i) * time.Second),
		}
		insertTestSnap(t, s, snap)
	}

	count, err = s.CountSnapshots(ctx, "app-001")
	if err != nil {
		t.Fatalf("CountSnapshots: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestDeleteOldSnapshots(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	app := &Application{
		ID: "app-001", Name: "Test App", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app)

	// Insert an old snapshot (48 hours ago).
	old := &Snapshot{
		ID: "snap-old", ApplicationID: "app-001", ContentHash: "h1",
		Content: "old", Format: "json", SizeBytes: 3,
		Source: "auto", CapturedAt: now.Add(-48 * time.Hour),
	}
	insertTestSnap(t, s, old)

	// Insert a recent snapshot (1 hour ago).
	recent := &Snapshot{
		ID: "snap-recent", ApplicationID: "app-001", ContentHash: "h2",
		Content: "recent", Format: "json", SizeBytes: 6,
		Source: "auto", CapturedAt: now.Add(-1 * time.Hour),
	}
	insertTestSnap(t, s, recent)

	// Delete snapshots older than 24 hours.
	cutoff := now.Add(-24 * time.Hour)
	deleted, err := s.DeleteOldSnapshots(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOldSnapshots: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	// Verify only the recent snapshot remains.
	remaining, err := s.ListSnapshots(ctx, ListSnapshotsParams{ApplicationID: "app-001", Limit: 50})
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining snapshot, got %d", len(remaining))
	}
	if remaining[0].ID != "snap-recent" {
		t.Errorf("remaining snapshot ID = %q, want %q", remaining[0].ID, "snap-recent")
	}
}

func TestDeleteOldSnapshots_NoneToDelete(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	app := &Application{
		ID: "app-001", Name: "Test App", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app)

	// Insert a recent snapshot.
	snap := &Snapshot{
		ID: "snap-001", ApplicationID: "app-001", ContentHash: "h1",
		Content: "config", Format: "json", SizeBytes: 6,
		Source: "auto", CapturedAt: now,
	}
	insertTestSnap(t, s, snap)

	// Delete snapshots older than 24 hours -- none should match.
	cutoff := now.Add(-24 * time.Hour)
	deleted, err := s.DeleteOldSnapshots(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOldSnapshots: %v", err)
	}
	if deleted != 0 {
		t.Errorf("deleted = %d, want 0", deleted)
	}
}

func TestListSnapshots_OrderByCapturedAtDesc(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	app := &Application{
		ID: "app-001", Name: "Test App", AppType: "test",
		Collector: "manual", Status: "active", Metadata: "{}",
		DiscoveredAt: now, UpdatedAt: now,
	}
	insertTestApp(t, s, app)

	// Insert snapshots out of chronological order.
	snaps := []*Snapshot{
		{
			ID: "snap-mid", ApplicationID: "app-001", ContentHash: "h2",
			Content: "mid", Format: "json", SizeBytes: 3,
			Source: "auto", CapturedAt: now.Add(-1 * time.Hour),
		},
		{
			ID: "snap-old", ApplicationID: "app-001", ContentHash: "h1",
			Content: "old", Format: "json", SizeBytes: 3,
			Source: "auto", CapturedAt: now.Add(-2 * time.Hour),
		},
		{
			ID: "snap-new", ApplicationID: "app-001", ContentHash: "h3",
			Content: "new", Format: "json", SizeBytes: 3,
			Source: "auto", CapturedAt: now,
		},
	}
	for _, sn := range snaps {
		insertTestSnap(t, s, sn)
	}

	got, err := s.ListSnapshots(ctx, ListSnapshotsParams{ApplicationID: "app-001", Limit: 50})
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(snapshots) = %d, want 3", len(got))
	}

	// Verify DESC order: new, mid, old.
	if got[0].ID != "snap-new" {
		t.Errorf("got[0].ID = %q, want %q", got[0].ID, "snap-new")
	}
	if got[1].ID != "snap-mid" {
		t.Errorf("got[1].ID = %q, want %q", got[1].ID, "snap-mid")
	}
	if got[2].ID != "snap-old" {
		t.Errorf("got[2].ID = %q, want %q", got[2].ID, "snap-old")
	}
}
