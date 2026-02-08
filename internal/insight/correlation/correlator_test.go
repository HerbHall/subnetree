package correlation

import (
	"sort"
	"testing"
	"time"
)

func TestCorrelate_SingleAlert(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
	}

	result := Correlate(alerts, nil, 5*time.Minute)

	if result != nil {
		t.Errorf("expected nil for single alert, got %d groups", len(result))
	}
}

func TestCorrelate_ConnectedDevices(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
		{DeviceID: "dev2", Metric: "memory", Timestamp: now.Add(1 * time.Minute)},
	}
	topology := []TopologyEdge{
		{SourceID: "dev1", TargetID: "dev2"},
	}

	result := Correlate(alerts, topology, 5*time.Minute)

	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}

	group := result[0]
	if len(group.DeviceIDs) != 2 {
		t.Errorf("group has %d devices, want 2", len(group.DeviceIDs))
	}
	if len(group.Alerts) != 2 {
		t.Errorf("group has %d alerts, want 2", len(group.Alerts))
	}
	if group.RootCause != "dev1" {
		t.Errorf("RootCause = %s, want dev1 (earliest alert)", group.RootCause)
	}
}

func TestCorrelate_DisconnectedDevices(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
		{DeviceID: "dev2", Metric: "memory", Timestamp: now.Add(1 * time.Minute)},
	}
	// No topology edges - devices are disconnected

	result := Correlate(alerts, nil, 5*time.Minute)

	// Single-device groups are excluded
	if result != nil {
		t.Errorf("expected nil (single-device groups excluded), got %d groups", len(result))
	}
}

func TestCorrelate_OutsideTimeWindow(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
		{DeviceID: "dev2", Metric: "memory", Timestamp: now.Add(10 * time.Minute)},
	}
	topology := []TopologyEdge{
		{SourceID: "dev1", TargetID: "dev2"},
	}

	// Window is 5 minutes, but alerts are 10 minutes apart
	result := Correlate(alerts, topology, 5*time.Minute)

	// Should not correlate - outside time window
	if result != nil {
		t.Errorf("expected nil (outside time window), got %d groups", len(result))
	}
}

func TestCorrelate_CascadeEffect(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
		{DeviceID: "dev2", Metric: "memory", Timestamp: now.Add(1 * time.Minute)},
		{DeviceID: "dev3", Metric: "disk", Timestamp: now.Add(2 * time.Minute)},
	}
	topology := []TopologyEdge{
		{SourceID: "dev1", TargetID: "dev2"},
		{SourceID: "dev2", TargetID: "dev3"},
	}

	result := Correlate(alerts, topology, 5*time.Minute)

	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}

	group := result[0]
	if len(group.DeviceIDs) != 3 {
		t.Errorf("group has %d devices, want 3 (cascade A->B->C)", len(group.DeviceIDs))
	}
	if len(group.Alerts) != 3 {
		t.Errorf("group has %d alerts, want 3", len(group.Alerts))
	}

	// Verify all device IDs are present
	deviceSet := make(map[string]bool)
	for _, id := range group.DeviceIDs {
		deviceSet[id] = true
	}
	for _, expected := range []string{"dev1", "dev2", "dev3"} {
		if !deviceSet[expected] {
			t.Errorf("missing device %s in group", expected)
		}
	}
}

func TestCorrelate_RootCause(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		{DeviceID: "dev2", Metric: "memory", Timestamp: now.Add(2 * time.Minute)},
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now}, // Earliest
		{DeviceID: "dev3", Metric: "disk", Timestamp: now.Add(1 * time.Minute)},
	}
	topology := []TopologyEdge{
		{SourceID: "dev1", TargetID: "dev2"},
		{SourceID: "dev2", TargetID: "dev3"},
	}

	result := Correlate(alerts, topology, 5*time.Minute)

	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}

	group := result[0]
	if group.RootCause != "dev1" {
		t.Errorf("RootCause = %s, want dev1 (earliest alert)", group.RootCause)
	}
}

func TestCorrelate_NoTopology(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
		{DeviceID: "dev1", Metric: "memory", Timestamp: now.Add(1 * time.Minute)},
	}

	// No topology edges, but same device
	result := Correlate(alerts, nil, 5*time.Minute)

	// Single-device groups are excluded
	if result != nil {
		t.Errorf("expected nil (single-device group excluded), got %d groups", len(result))
	}
}

func TestCorrelate_EmptyAlerts(t *testing.T) {
	t.Parallel()

	result := Correlate([]Alert{}, nil, 5*time.Minute)

	if result != nil {
		t.Errorf("expected nil for empty alerts, got %d groups", len(result))
	}
}

func TestCorrelate_MultipleGroups(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		// Group 1: dev1 <-> dev2
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
		{DeviceID: "dev2", Metric: "memory", Timestamp: now.Add(1 * time.Minute)},
		// Group 2: dev3 <-> dev4
		{DeviceID: "dev3", Metric: "disk", Timestamp: now.Add(2 * time.Minute)},
		{DeviceID: "dev4", Metric: "network", Timestamp: now.Add(3 * time.Minute)},
	}
	topology := []TopologyEdge{
		{SourceID: "dev1", TargetID: "dev2"},
		{SourceID: "dev3", TargetID: "dev4"},
		// No edge between groups
	}

	result := Correlate(alerts, topology, 10*time.Minute)

	if len(result) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result))
	}

	// Sort groups by first device ID for deterministic comparison
	sort.Slice(result, func(i, j int) bool {
		return result[i].DeviceIDs[0] < result[j].DeviceIDs[0]
	})

	// Check each group has 2 devices
	for i, group := range result {
		if len(group.DeviceIDs) != 2 {
			t.Errorf("group %d has %d devices, want 2", i, len(group.DeviceIDs))
		}
		if len(group.Alerts) != 2 {
			t.Errorf("group %d has %d alerts, want 2", i, len(group.Alerts))
		}
	}
}

func TestCorrelate_SameDeviceMultipleAlerts(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
		{DeviceID: "dev1", Metric: "memory", Timestamp: now.Add(30 * time.Second)},
		{DeviceID: "dev2", Metric: "disk", Timestamp: now.Add(1 * time.Minute)},
	}
	topology := []TopologyEdge{
		{SourceID: "dev1", TargetID: "dev2"},
	}

	result := Correlate(alerts, topology, 5*time.Minute)

	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}

	group := result[0]
	if len(group.DeviceIDs) != 2 {
		t.Errorf("group has %d unique devices, want 2", len(group.DeviceIDs))
	}
	if len(group.Alerts) != 3 {
		t.Errorf("group has %d alerts, want 3", len(group.Alerts))
	}
}

func TestCorrelate_BidirectionalEdge(t *testing.T) {
	t.Parallel()

	now := time.Now()
	alerts := []Alert{
		{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
		{DeviceID: "dev2", Metric: "memory", Timestamp: now.Add(1 * time.Minute)},
	}
	// Only one edge defined, but correlation treats topology as undirected
	topology := []TopologyEdge{
		{SourceID: "dev1", TargetID: "dev2"},
	}

	result := Correlate(alerts, topology, 5*time.Minute)

	if len(result) != 1 {
		t.Fatalf("expected 1 group (bidirectional edge), got %d", len(result))
	}
}

func TestCorrelate_TimeWindowBoundary(t *testing.T) {
	t.Parallel()

	now := time.Now()
	window := 5 * time.Minute

	tests := []struct {
		name      string
		offset    time.Duration
		wantGroup bool
	}{
		{
			name:      "just inside window",
			offset:    4*time.Minute + 59*time.Second,
			wantGroup: true,
		},
		{
			name:      "exactly at window boundary",
			offset:    5 * time.Minute,
			wantGroup: true, // inclusive: absDuration == window is within window
		},
		{
			name:      "just outside window",
			offset:    5*time.Minute + 1*time.Second,
			wantGroup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alerts := []Alert{
				{DeviceID: "dev1", Metric: "cpu", Timestamp: now},
				{DeviceID: "dev2", Metric: "memory", Timestamp: now.Add(tt.offset)},
			}
			topology := []TopologyEdge{
				{SourceID: "dev1", TargetID: "dev2"},
			}

			result := Correlate(alerts, topology, window)

			if tt.wantGroup && result == nil {
				t.Error("expected group, got nil")
			}
			if !tt.wantGroup && result != nil {
				t.Errorf("expected no group, got %d groups", len(result))
			}
		})
	}
}
