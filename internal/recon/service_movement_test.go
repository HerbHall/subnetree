package recon

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// detectServiceMovements unit tests
// ---------------------------------------------------------------------------

func TestDetectServiceMovements_SingleMove(t *testing.T) {
	previous := map[string][]int{
		"device-a": {22, 80},
		"device-b": {443},
	}
	current := map[string][]int{
		"device-a": {22},
		"device-b": {443, 80},
	}

	movements := detectServiceMovements(previous, current)
	if len(movements) != 1 {
		t.Fatalf("got %d movements, want 1", len(movements))
	}

	m := movements[0]
	if m.Port != 80 {
		t.Errorf("Port = %d, want 80", m.Port)
	}
	if m.FromDevice != "device-a" {
		t.Errorf("FromDevice = %q, want device-a", m.FromDevice)
	}
	if m.ToDevice != "device-b" {
		t.Errorf("ToDevice = %q, want device-b", m.ToDevice)
	}
	if m.Protocol != "tcp" {
		t.Errorf("Protocol = %q, want tcp", m.Protocol)
	}
	if m.ServiceName != "http" {
		t.Errorf("ServiceName = %q, want http", m.ServiceName)
	}
	if m.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestDetectServiceMovements_ServiceDisappears(t *testing.T) {
	previous := map[string][]int{
		"device-a": {22, 80},
		"device-b": {443},
	}
	current := map[string][]int{
		"device-a": {22},
		"device-b": {443},
	}

	movements := detectServiceMovements(previous, current)
	if len(movements) != 0 {
		t.Errorf("got %d movements, want 0 (port 80 disappeared, not moved)", len(movements))
	}
}

func TestDetectServiceMovements_NewServiceAppears(t *testing.T) {
	previous := map[string][]int{
		"device-a": {22},
	}
	current := map[string][]int{
		"device-a": {22},
		"device-b": {80},
	}

	movements := detectServiceMovements(previous, current)
	if len(movements) != 0 {
		t.Errorf("got %d movements, want 0 (port 80 is new, not moved)", len(movements))
	}
}

func TestDetectServiceMovements_MultipleTargets(t *testing.T) {
	// Port 80 was on device-a, now appears on device-b AND device-c.
	// This is replication, not movement.
	previous := map[string][]int{
		"device-a": {80},
	}
	current := map[string][]int{
		"device-b": {80},
		"device-c": {80},
	}

	movements := detectServiceMovements(previous, current)
	if len(movements) != 0 {
		t.Errorf("got %d movements, want 0 (port appeared on multiple devices)", len(movements))
	}
}

func TestDetectServiceMovements_MultipleSources(t *testing.T) {
	// Port 80 was on device-a AND device-b, now only on device-c.
	// Multiple sources removed -- not a clear 1:1 movement.
	previous := map[string][]int{
		"device-a": {80},
		"device-b": {80},
	}
	current := map[string][]int{
		"device-c": {80},
	}

	movements := detectServiceMovements(previous, current)
	if len(movements) != 0 {
		t.Errorf("got %d movements, want 0 (multiple sources removed)", len(movements))
	}
}

func TestDetectServiceMovements_MultipleSimultaneous(t *testing.T) {
	previous := map[string][]int{
		"device-a": {22, 80, 443},
		"device-b": {3306},
	}
	current := map[string][]int{
		"device-a": {443},
		"device-b": {22, 80, 3306},
	}

	movements := detectServiceMovements(previous, current)
	if len(movements) != 2 {
		t.Fatalf("got %d movements, want 2", len(movements))
	}

	// Check both ports are represented (order is not guaranteed due to map iteration).
	ports := make(map[int]bool)
	for _, m := range movements {
		ports[m.Port] = true
		if m.FromDevice != "device-a" {
			t.Errorf("FromDevice = %q, want device-a", m.FromDevice)
		}
		if m.ToDevice != "device-b" {
			t.Errorf("ToDevice = %q, want device-b", m.ToDevice)
		}
	}
	if !ports[22] {
		t.Error("expected port 22 movement")
	}
	if !ports[80] {
		t.Error("expected port 80 movement")
	}
}

func TestDetectServiceMovements_EmptyMaps(t *testing.T) {
	movements := detectServiceMovements(map[string][]int{}, map[string][]int{})
	if len(movements) != 0 {
		t.Errorf("got %d movements, want 0", len(movements))
	}
}

func TestDetectServiceMovements_NilMaps(t *testing.T) {
	movements := detectServiceMovements(nil, nil)
	if len(movements) != 0 {
		t.Errorf("got %d movements, want 0", len(movements))
	}
}

func TestDetectServiceMovements_NoChange(t *testing.T) {
	services := map[string][]int{
		"device-a": {22, 80},
		"device-b": {443, 3306},
	}

	movements := detectServiceMovements(services, services)
	if len(movements) != 0 {
		t.Errorf("got %d movements, want 0 (no changes)", len(movements))
	}
}

func TestDetectServiceMovements_SamePortStaysOnOriginal(t *testing.T) {
	// Port 80 stays on device-a but also appears on device-b.
	// device-a still has it, so it's not a movement.
	previous := map[string][]int{
		"device-a": {80},
	}
	current := map[string][]int{
		"device-a": {80},
		"device-b": {80},
	}

	movements := detectServiceMovements(previous, current)
	if len(movements) != 0 {
		t.Errorf("got %d movements, want 0 (port still on original device)", len(movements))
	}
}

// ---------------------------------------------------------------------------
// lookupServiceName tests
// ---------------------------------------------------------------------------

func TestLookupServiceName_Known(t *testing.T) {
	tests := []struct {
		port int
		want string
	}{
		{22, "ssh"},
		{80, "http"},
		{443, "https"},
		{3306, "mysql"},
		{5432, "postgres"},
	}

	for _, tt := range tests {
		got := lookupServiceName(tt.port)
		if got != tt.want {
			t.Errorf("lookupServiceName(%d) = %q, want %q", tt.port, got, tt.want)
		}
	}
}

func TestLookupServiceName_Unknown(t *testing.T) {
	got := lookupServiceName(99999)
	if got != "" {
		t.Errorf("lookupServiceName(99999) = %q, want empty", got)
	}
}

// ---------------------------------------------------------------------------
// Store integration tests
// ---------------------------------------------------------------------------

func TestSaveAndListServiceMovements(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	movement := ServiceMovement{
		ID:          "test-movement-1",
		Port:        80,
		Protocol:    "tcp",
		ServiceName: "http",
		FromDevice:  "device-a",
		ToDevice:    "device-b",
		DetectedAt:  time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
	}

	if err := s.SaveServiceMovement(ctx, movement); err != nil {
		t.Fatalf("SaveServiceMovement: %v", err)
	}

	movements, err := s.ListServiceMovements(ctx, 50)
	if err != nil {
		t.Fatalf("ListServiceMovements: %v", err)
	}
	if len(movements) != 1 {
		t.Fatalf("got %d movements, want 1", len(movements))
	}

	got := movements[0]
	if got.ID != "test-movement-1" {
		t.Errorf("ID = %q, want test-movement-1", got.ID)
	}
	if got.Port != 80 {
		t.Errorf("Port = %d, want 80", got.Port)
	}
	if got.Protocol != "tcp" {
		t.Errorf("Protocol = %q, want tcp", got.Protocol)
	}
	if got.ServiceName != "http" {
		t.Errorf("ServiceName = %q, want http", got.ServiceName)
	}
	if got.FromDevice != "device-a" {
		t.Errorf("FromDevice = %q, want device-a", got.FromDevice)
	}
	if got.ToDevice != "device-b" {
		t.Errorf("ToDevice = %q, want device-b", got.ToDevice)
	}
	if !got.DetectedAt.Equal(movement.DetectedAt) {
		t.Errorf("DetectedAt = %v, want %v", got.DetectedAt, movement.DetectedAt)
	}
}

func TestListServiceMovements_OrderByDetectedAt(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Insert older movement first.
	older := ServiceMovement{
		ID:         "older",
		Port:       22,
		Protocol:   "tcp",
		FromDevice: "a",
		ToDevice:   "b",
		DetectedAt: time.Date(2026, 2, 14, 10, 0, 0, 0, time.UTC),
	}
	newer := ServiceMovement{
		ID:         "newer",
		Port:       80,
		Protocol:   "tcp",
		FromDevice: "c",
		ToDevice:   "d",
		DetectedAt: time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
	}

	if err := s.SaveServiceMovement(ctx, older); err != nil {
		t.Fatalf("SaveServiceMovement older: %v", err)
	}
	if err := s.SaveServiceMovement(ctx, newer); err != nil {
		t.Fatalf("SaveServiceMovement newer: %v", err)
	}

	movements, err := s.ListServiceMovements(ctx, 50)
	if err != nil {
		t.Fatalf("ListServiceMovements: %v", err)
	}
	if len(movements) != 2 {
		t.Fatalf("got %d movements, want 2", len(movements))
	}
	if movements[0].ID != "newer" {
		t.Errorf("first movement ID = %q, want newer (most recent first)", movements[0].ID)
	}
	if movements[1].ID != "older" {
		t.Errorf("second movement ID = %q, want older", movements[1].ID)
	}
}

func TestListServiceMovements_Limit(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		m := ServiceMovement{
			ID:         "m-" + string(rune('0'+i)),
			Port:       22 + i,
			Protocol:   "tcp",
			FromDevice: "a",
			ToDevice:   "b",
			DetectedAt: time.Date(2026, 2, 15, i, 0, 0, 0, time.UTC),
		}
		if err := s.SaveServiceMovement(ctx, m); err != nil {
			t.Fatalf("SaveServiceMovement %d: %v", i, err)
		}
	}

	movements, err := s.ListServiceMovements(ctx, 3)
	if err != nil {
		t.Fatalf("ListServiceMovements: %v", err)
	}
	if len(movements) != 3 {
		t.Errorf("got %d movements, want 3 (limited)", len(movements))
	}
}

func TestListServiceMovements_Empty(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	movements, err := s.ListServiceMovements(ctx, 50)
	if err != nil {
		t.Fatalf("ListServiceMovements: %v", err)
	}
	if movements != nil {
		t.Errorf("got %v, want nil for empty table", movements)
	}
}

func TestGetPreviousServiceMap_NoScans(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	serviceMap, err := s.GetPreviousServiceMap(ctx)
	if err != nil {
		t.Fatalf("GetPreviousServiceMap: %v", err)
	}
	if serviceMap == nil {
		t.Error("expected non-nil map")
	}
	if len(serviceMap) != 0 {
		t.Errorf("got %d entries, want 0", len(serviceMap))
	}
}
