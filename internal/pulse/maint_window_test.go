package pulse

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// isTimeInWindow – table-driven
// ---------------------------------------------------------------------------

func TestIsTimeInWindow(t *testing.T) {
	base := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC) // Sunday

	tests := []struct {
		name       string
		now        time.Time
		start      time.Time
		end        time.Time
		recurrence string
		want       bool
	}{
		{
			name:       "once: within range",
			now:        base.Add(3 * time.Hour),
			start:      base.Add(2 * time.Hour),
			end:        base.Add(4 * time.Hour),
			recurrence: "once",
			want:       true,
		},
		{
			name:       "once: before range",
			now:        base.Add(1 * time.Hour),
			start:      base.Add(2 * time.Hour),
			end:        base.Add(4 * time.Hour),
			recurrence: "once",
			want:       false,
		},
		{
			name:       "once: after range",
			now:        base.Add(5 * time.Hour),
			start:      base.Add(2 * time.Hour),
			end:        base.Add(4 * time.Hour),
			recurrence: "once",
			want:       false,
		},
		{
			name:       "daily: within time of day",
			now:        time.Date(2025, 7, 10, 3, 0, 0, 0, time.UTC),
			start:      base.Add(2 * time.Hour), // 02:00
			end:        base.Add(4 * time.Hour), // 04:00
			recurrence: "daily",
			want:       true,
		},
		{
			name:       "daily: outside time of day",
			now:        time.Date(2025, 7, 10, 5, 0, 0, 0, time.UTC),
			start:      base.Add(2 * time.Hour), // 02:00
			end:        base.Add(4 * time.Hour), // 04:00
			recurrence: "daily",
			want:       false,
		},
		{
			name:       "daily: midnight crossing within",
			now:        time.Date(2025, 7, 10, 23, 30, 0, 0, time.UTC),
			start:      base.Add(22 * time.Hour), // 22:00
			end:        base.Add(2 * time.Hour),   // 02:00 next day
			recurrence: "daily",
			want:       true,
		},
		{
			name:       "daily: midnight crossing within (after midnight)",
			now:        time.Date(2025, 7, 11, 1, 30, 0, 0, time.UTC),
			start:      base.Add(22 * time.Hour), // 22:00
			end:        base.Add(2 * time.Hour),   // 02:00
			recurrence: "daily",
			want:       true,
		},
		{
			name:       "weekly: correct weekday within time",
			now:        time.Date(2025, 6, 22, 3, 0, 0, 0, time.UTC), // Sunday
			start:      base.Add(2 * time.Hour),                       // Sunday 02:00
			end:        base.Add(4 * time.Hour),                       // Sunday 04:00
			recurrence: "weekly",
			want:       true,
		},
		{
			name:       "weekly: wrong weekday",
			now:        time.Date(2025, 6, 23, 3, 0, 0, 0, time.UTC), // Monday
			start:      base.Add(2 * time.Hour),                       // Sunday 02:00
			end:        base.Add(4 * time.Hour),                       // Sunday 04:00
			recurrence: "weekly",
			want:       false,
		},
		{
			name:       "monthly: correct day within time",
			now:        time.Date(2025, 7, 15, 3, 0, 0, 0, time.UTC), // 15th
			start:      base.Add(2 * time.Hour),                       // 15th 02:00
			end:        base.Add(4 * time.Hour),                       // 15th 04:00
			recurrence: "monthly",
			want:       true,
		},
		{
			name:       "monthly: wrong day",
			now:        time.Date(2025, 7, 16, 3, 0, 0, 0, time.UTC), // 16th
			start:      base.Add(2 * time.Hour),                       // 15th 02:00
			end:        base.Add(4 * time.Hour),                       // 15th 04:00
			recurrence: "monthly",
			want:       false,
		},
		{
			name:       "unknown recurrence",
			now:        base.Add(3 * time.Hour),
			start:      base.Add(2 * time.Hour),
			end:        base.Add(4 * time.Hour),
			recurrence: "yearly",
			want:       false,
		},
		{
			name:       "once: exact start boundary",
			now:        base.Add(2 * time.Hour),
			start:      base.Add(2 * time.Hour),
			end:        base.Add(4 * time.Hour),
			recurrence: "once",
			want:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTimeInWindow(tt.now, tt.start, tt.end, tt.recurrence)
			if got != tt.want {
				t.Errorf("isTimeInWindow() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Store CRUD
// ---------------------------------------------------------------------------

func TestMaintWindow_CRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mw := &MaintWindow{
		ID:          "mw-1",
		Name:        "Weekly Patch Window",
		Description: "Server patching",
		StartTime:   time.Date(2025, 6, 15, 2, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2025, 6, 15, 4, 0, 0, 0, time.UTC),
		Recurrence:  "weekly",
		DeviceIDs:   []string{"dev-a", "dev-b"},
		Enabled:     true,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Insert.
	if err := s.InsertMaintWindow(ctx, mw); err != nil {
		t.Fatalf("InsertMaintWindow: %v", err)
	}

	// Get.
	got, err := s.GetMaintWindow(ctx, "mw-1")
	if err != nil {
		t.Fatalf("GetMaintWindow: %v", err)
	}
	if got == nil {
		t.Fatal("GetMaintWindow returned nil")
	}
	if got.Name != "Weekly Patch Window" {
		t.Errorf("Name = %q, want %q", got.Name, "Weekly Patch Window")
	}
	if len(got.DeviceIDs) != 2 {
		t.Errorf("DeviceIDs len = %d, want 2", len(got.DeviceIDs))
	}
	if !got.Enabled {
		t.Error("Enabled = false, want true")
	}

	// List.
	all, err := s.ListMaintWindows(ctx)
	if err != nil {
		t.Fatalf("ListMaintWindows: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("ListMaintWindows len = %d, want 1", len(all))
	}

	// Update.
	mw.Name = "Updated Window"
	mw.Enabled = false
	mw.UpdatedAt = time.Now().UTC()
	if err := s.UpdateMaintWindow(ctx, mw); err != nil {
		t.Fatalf("UpdateMaintWindow: %v", err)
	}
	got, err = s.GetMaintWindow(ctx, "mw-1")
	if err != nil {
		t.Fatalf("GetMaintWindow after update: %v", err)
	}
	if got.Name != "Updated Window" {
		t.Errorf("Name after update = %q, want %q", got.Name, "Updated Window")
	}
	if got.Enabled {
		t.Error("Enabled after update = true, want false")
	}

	// Delete.
	if err := s.DeleteMaintWindow(ctx, "mw-1"); err != nil {
		t.Fatalf("DeleteMaintWindow: %v", err)
	}
	got, err = s.GetMaintWindow(ctx, "mw-1")
	if err != nil {
		t.Fatalf("GetMaintWindow after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestGetMaintWindow_NotFound(t *testing.T) {
	s := testStore(t)
	got, err := s.GetMaintWindow(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent window")
	}
}

// ---------------------------------------------------------------------------
// IsDeviceInMaintenanceWindow
// ---------------------------------------------------------------------------

func TestIsDeviceInMaintenanceWindow_Active(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	mw := &MaintWindow{
		ID:         "mw-active",
		Name:       "Active Window",
		StartTime:  now.Add(-1 * time.Hour),
		EndTime:    now.Add(1 * time.Hour),
		Recurrence: "once",
		DeviceIDs:  []string{"dev-1", "dev-2"},
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.InsertMaintWindow(ctx, mw); err != nil {
		t.Fatalf("InsertMaintWindow: %v", err)
	}

	inMaint, err := s.IsDeviceInMaintenanceWindow(ctx, "dev-1")
	if err != nil {
		t.Fatalf("IsDeviceInMaintenanceWindow: %v", err)
	}
	if !inMaint {
		t.Error("expected device to be in maintenance window")
	}
}

func TestIsDeviceInMaintenanceWindow_Disabled(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	mw := &MaintWindow{
		ID:         "mw-disabled",
		Name:       "Disabled Window",
		StartTime:  now.Add(-1 * time.Hour),
		EndTime:    now.Add(1 * time.Hour),
		Recurrence: "once",
		DeviceIDs:  []string{"dev-1"},
		Enabled:    false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.InsertMaintWindow(ctx, mw); err != nil {
		t.Fatalf("InsertMaintWindow: %v", err)
	}

	inMaint, err := s.IsDeviceInMaintenanceWindow(ctx, "dev-1")
	if err != nil {
		t.Fatalf("IsDeviceInMaintenanceWindow: %v", err)
	}
	if inMaint {
		t.Error("expected device NOT to be in maintenance window (disabled)")
	}
}

func TestIsDeviceInMaintenanceWindow_Expired(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	mw := &MaintWindow{
		ID:         "mw-expired",
		Name:       "Expired Window",
		StartTime:  now.Add(-3 * time.Hour),
		EndTime:    now.Add(-1 * time.Hour),
		Recurrence: "once",
		DeviceIDs:  []string{"dev-1"},
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.InsertMaintWindow(ctx, mw); err != nil {
		t.Fatalf("InsertMaintWindow: %v", err)
	}

	inMaint, err := s.IsDeviceInMaintenanceWindow(ctx, "dev-1")
	if err != nil {
		t.Fatalf("IsDeviceInMaintenanceWindow: %v", err)
	}
	if inMaint {
		t.Error("expected device NOT to be in maintenance window (expired)")
	}
}

func TestIsDeviceInMaintenanceWindow_NoWindows(t *testing.T) {
	s := testStore(t)
	inMaint, err := s.IsDeviceInMaintenanceWindow(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("IsDeviceInMaintenanceWindow: %v", err)
	}
	if inMaint {
		t.Error("expected device NOT to be in maintenance window (no windows)")
	}
}

func TestIsDeviceInMaintenanceWindow_DeviceNotInList(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	mw := &MaintWindow{
		ID:         "mw-other",
		Name:       "Other Devices",
		StartTime:  now.Add(-1 * time.Hour),
		EndTime:    now.Add(1 * time.Hour),
		Recurrence: "once",
		DeviceIDs:  []string{"dev-a", "dev-b"},
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.InsertMaintWindow(ctx, mw); err != nil {
		t.Fatalf("InsertMaintWindow: %v", err)
	}

	inMaint, err := s.IsDeviceInMaintenanceWindow(ctx, "dev-1")
	if err != nil {
		t.Fatalf("IsDeviceInMaintenanceWindow: %v", err)
	}
	if inMaint {
		t.Error("expected device NOT to be in maintenance window (not in list)")
	}
}
