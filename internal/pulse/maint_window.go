package pulse

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// MaintWindow represents a scheduled maintenance window during which
// monitoring alerts for specified devices are suppressed.
type MaintWindow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Recurrence  string    `json:"recurrence"` // "once", "daily", "weekly", "monthly"
	DeviceIDs   []string  `json:"device_ids"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var validRecurrence = map[string]bool{
	"once":    true,
	"daily":   true,
	"weekly":  true,
	"monthly": true,
}

// -- Maintenance Window CRUD --

// InsertMaintWindow inserts a new maintenance window.
func (s *PulseStore) InsertMaintWindow(ctx context.Context, mw *MaintWindow) error {
	enabled := 0
	if mw.Enabled {
		enabled = 1
	}
	deviceJSON, err := json.Marshal(mw.DeviceIDs)
	if err != nil {
		return fmt.Errorf("marshal device_ids: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO pulse_maint_windows (
			id, name, description, start_time, end_time, recurrence,
			device_ids, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mw.ID, mw.Name, mw.Description, mw.StartTime, mw.EndTime,
		mw.Recurrence, string(deviceJSON), enabled, mw.CreatedAt, mw.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert maint window: %w", err)
	}
	return nil
}

// GetMaintWindow returns a maintenance window by ID. Returns nil, nil if not found.
func (s *PulseStore) GetMaintWindow(ctx context.Context, id string) (*MaintWindow, error) {
	var mw MaintWindow
	var enabledInt int
	var deviceJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, start_time, end_time, recurrence,
			device_ids, enabled, created_at, updated_at
		FROM pulse_maint_windows WHERE id = ?`,
		id,
	).Scan(
		&mw.ID, &mw.Name, &mw.Description, &mw.StartTime, &mw.EndTime,
		&mw.Recurrence, &deviceJSON, &enabledInt, &mw.CreatedAt, &mw.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get maint window: %w", err)
	}
	mw.Enabled = enabledInt != 0
	if err := json.Unmarshal([]byte(deviceJSON), &mw.DeviceIDs); err != nil {
		return nil, fmt.Errorf("unmarshal device_ids: %w", err)
	}
	return &mw, nil
}

// ListMaintWindows returns all maintenance windows.
func (s *PulseStore) ListMaintWindows(ctx context.Context) ([]MaintWindow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, start_time, end_time, recurrence,
			device_ids, enabled, created_at, updated_at
		FROM pulse_maint_windows ORDER BY start_time DESC`)
	if err != nil {
		return nil, fmt.Errorf("list maint windows: %w", err)
	}
	defer rows.Close()

	var windows []MaintWindow
	for rows.Next() {
		var mw MaintWindow
		var enabledInt int
		var deviceJSON string
		if err := rows.Scan(
			&mw.ID, &mw.Name, &mw.Description, &mw.StartTime, &mw.EndTime,
			&mw.Recurrence, &deviceJSON, &enabledInt, &mw.CreatedAt, &mw.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan maint window: %w", err)
		}
		mw.Enabled = enabledInt != 0
		if err := json.Unmarshal([]byte(deviceJSON), &mw.DeviceIDs); err != nil {
			return nil, fmt.Errorf("unmarshal device_ids: %w", err)
		}
		windows = append(windows, mw)
	}
	return windows, rows.Err()
}

// UpdateMaintWindow updates an existing maintenance window.
func (s *PulseStore) UpdateMaintWindow(ctx context.Context, mw *MaintWindow) error {
	enabled := 0
	if mw.Enabled {
		enabled = 1
	}
	deviceJSON, err := json.Marshal(mw.DeviceIDs)
	if err != nil {
		return fmt.Errorf("marshal device_ids: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE pulse_maint_windows SET
			name = ?, description = ?, start_time = ?, end_time = ?,
			recurrence = ?, device_ids = ?, enabled = ?, updated_at = ?
		WHERE id = ?`,
		mw.Name, mw.Description, mw.StartTime, mw.EndTime,
		mw.Recurrence, string(deviceJSON), enabled, mw.UpdatedAt, mw.ID,
	)
	if err != nil {
		return fmt.Errorf("update maint window: %w", err)
	}
	return nil
}

// DeleteMaintWindow removes a maintenance window by ID.
func (s *PulseStore) DeleteMaintWindow(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pulse_maint_windows WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete maint window: %w", err)
	}
	return nil
}

// IsDeviceInMaintenanceWindow checks whether the given device is currently
// inside any enabled maintenance window, accounting for recurrence.
func (s *PulseStore) IsDeviceInMaintenanceWindow(ctx context.Context, deviceID string) (bool, error) {
	windows, err := s.ListMaintWindows(ctx)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	for i := range windows {
		if !windows[i].Enabled {
			continue
		}
		// Check if device is in this window's device list.
		found := false
		for _, did := range windows[i].DeviceIDs {
			if did == deviceID {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		if isTimeInWindow(now, windows[i].StartTime, windows[i].EndTime, windows[i].Recurrence) {
			return true, nil
		}
	}
	return false, nil
}

// isTimeInWindow returns true if t falls within the maintenance window
// defined by start/end with the given recurrence type.
func isTimeInWindow(t, start, end time.Time, recurrence string) bool {
	switch recurrence {
	case "once":
		return !t.Before(start) && !t.After(end)

	case "daily":
		return isTimeOfDayInRange(t, start, end)

	case "weekly":
		if t.Weekday() != start.Weekday() {
			return false
		}
		return isTimeOfDayInRange(t, start, end)

	case "monthly":
		if t.Day() != start.Day() {
			return false
		}
		return isTimeOfDayInRange(t, start, end)

	default:
		return false
	}
}

// isTimeOfDayInRange checks whether the time-of-day of t falls within
// the time-of-day range defined by start and end. Supports midnight
// crossing (e.g., 22:00-02:00).
func isTimeOfDayInRange(t, start, end time.Time) bool {
	tSec := timeOfDaySeconds(t)
	startSec := timeOfDaySeconds(start)
	endSec := timeOfDaySeconds(end)

	if startSec <= endSec {
		// Normal range (e.g., 02:00-06:00).
		return tSec >= startSec && tSec <= endSec
	}
	// Midnight-crossing range (e.g., 22:00-02:00).
	return tSec >= startSec || tSec <= endSec
}

// timeOfDaySeconds returns the number of seconds elapsed since midnight.
func timeOfDaySeconds(t time.Time) int {
	return t.Hour()*3600 + t.Minute()*60 + t.Second()
}
