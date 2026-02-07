package correlation

import "time"

// Alert represents an alert for correlation analysis.
type Alert struct {
	DeviceID  string
	Metric    string
	Timestamp time.Time
}

// TopologyEdge represents a network link between two devices.
type TopologyEdge struct {
	SourceID string
	TargetID string
}

// Group represents a group of correlated alerts.
type Group struct {
	DeviceIDs []string
	Alerts    []Alert
	RootCause string // Device ID with earliest alert in group
}

// Correlate groups alerts that are topologically connected and occur within the given time window.
// topology provides the network links between devices.
// Returns groups of correlated alerts (single-device groups are excluded).
func Correlate(alerts []Alert, topology []TopologyEdge, window time.Duration) []Group {
	if len(alerts) < 2 {
		return nil
	}

	// Build adjacency from topology
	adj := make(map[string]map[string]bool)
	for _, e := range topology {
		if adj[e.SourceID] == nil {
			adj[e.SourceID] = make(map[string]bool)
		}
		if adj[e.TargetID] == nil {
			adj[e.TargetID] = make(map[string]bool)
		}
		adj[e.SourceID][e.TargetID] = true
		adj[e.TargetID][e.SourceID] = true
	}

	// Union-Find
	parent := make(map[string]string)
	find := func(x string) string {
		for parent[x] != "" && parent[x] != x {
			parent[x] = parent[parent[x]] // Path compression
			x = parent[x]
		}
		return x
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	// Initialize parents
	for _, a := range alerts {
		if parent[a.DeviceID] == "" {
			parent[a.DeviceID] = a.DeviceID
		}
	}

	// Group alerts on connected devices within the time window
	for i := 0; i < len(alerts); i++ {
		for j := i + 1; j < len(alerts); j++ {
			a, b := alerts[i], alerts[j]
			// Check time window
			if absDuration(a.Timestamp.Sub(b.Timestamp)) > window {
				continue
			}
			// Check topology adjacency
			if a.DeviceID == b.DeviceID || adj[a.DeviceID][b.DeviceID] {
				union(a.DeviceID, b.DeviceID)
			}
		}
	}

	// Collect groups
	groups := make(map[string]*Group)
	for _, a := range alerts {
		root := find(a.DeviceID)
		g, ok := groups[root]
		if !ok {
			g = &Group{}
			groups[root] = g
		}
		g.Alerts = append(g.Alerts, a)
	}

	// Build result -- only multi-device groups
	result := make([]Group, 0, len(groups))
	for _, g := range groups {
		deviceSet := make(map[string]bool)
		var earliest Alert
		for i, a := range g.Alerts {
			deviceSet[a.DeviceID] = true
			if i == 0 || a.Timestamp.Before(earliest.Timestamp) {
				earliest = a
			}
		}
		if len(deviceSet) < 2 {
			continue
		}
		devices := make([]string, 0, len(deviceSet))
		for d := range deviceSet {
			devices = append(devices, d)
		}
		g.DeviceIDs = devices
		g.RootCause = earliest.DeviceID
		result = append(result, *g)
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
