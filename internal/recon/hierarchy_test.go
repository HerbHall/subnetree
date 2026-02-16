package recon

import (
	"context"
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

func TestInferHierarchyFromData(t *testing.T) {
	tests := []struct {
		name    string
		devices []models.Device
		links   []TopologyLink
		check   func(t *testing.T, result []HierarchyAssignment)
	}{
		{
			name:    "empty devices returns nil",
			devices: nil,
			links:   nil,
			check: func(t *testing.T, result []HierarchyAssignment) {
				t.Helper()
				if result != nil {
					t.Errorf("expected nil, got %d assignments", len(result))
				}
			},
		},
		{
			name: "flat network: router plus 5 endpoints",
			devices: []models.Device{
				{ID: "router-1", DeviceType: models.DeviceTypeRouter},
				{ID: "server-1", DeviceType: models.DeviceTypeServer},
				{ID: "desktop-1", DeviceType: models.DeviceTypeDesktop},
				{ID: "laptop-1", DeviceType: models.DeviceTypeLaptop},
				{ID: "iot-1", DeviceType: models.DeviceTypeIoT},
				{ID: "phone-1", DeviceType: models.DeviceTypePhone},
			},
			links: []TopologyLink{
				{SourceDeviceID: "server-1", TargetDeviceID: "router-1", LinkType: "arp"},
				{SourceDeviceID: "desktop-1", TargetDeviceID: "router-1", LinkType: "arp"},
				{SourceDeviceID: "laptop-1", TargetDeviceID: "router-1", LinkType: "arp"},
				{SourceDeviceID: "iot-1", TargetDeviceID: "router-1", LinkType: "arp"},
				{SourceDeviceID: "phone-1", TargetDeviceID: "router-1", LinkType: "arp"},
			},
			check: func(t *testing.T, result []HierarchyAssignment) {
				t.Helper()
				m := assignmentMap(result)

				assertLayer(t, m, "router-1", models.NetworkLayerGateway)
				assertParent(t, m, "router-1", "")

				for _, id := range []string{"server-1", "desktop-1", "laptop-1", "iot-1", "phone-1"} {
					assertLayer(t, m, id, models.NetworkLayerEndpoint)
					assertParent(t, m, id, "router-1")
				}
			},
		},
		{
			name: "network with switch: router -> switch -> 3 endpoints",
			devices: []models.Device{
				{ID: "router-1", DeviceType: models.DeviceTypeRouter},
				{ID: "switch-1", DeviceType: models.DeviceTypeSwitch},
				{ID: "server-1", DeviceType: models.DeviceTypeServer},
				{ID: "server-2", DeviceType: models.DeviceTypeServer},
				{ID: "server-3", DeviceType: models.DeviceTypeServer},
			},
			links: []TopologyLink{
				{SourceDeviceID: "router-1", TargetDeviceID: "switch-1", LinkType: "ethernet"},
				{SourceDeviceID: "switch-1", TargetDeviceID: "server-1", LinkType: "fdb"},
				{SourceDeviceID: "switch-1", TargetDeviceID: "server-2", LinkType: "fdb"},
				{SourceDeviceID: "switch-1", TargetDeviceID: "server-3", LinkType: "fdb"},
			},
			check: func(t *testing.T, result []HierarchyAssignment) {
				t.Helper()
				m := assignmentMap(result)

				assertLayer(t, m, "router-1", models.NetworkLayerGateway)
				assertLayer(t, m, "switch-1", models.NetworkLayerDistribution)
				assertParent(t, m, "switch-1", "router-1")

				for _, id := range []string{"server-1", "server-2", "server-3"} {
					assertLayer(t, m, id, models.NetworkLayerEndpoint)
					assertParent(t, m, id, "switch-1")
				}
			},
		},
		{
			name: "multiple switches: router -> dist-switch -> access-switch -> endpoints",
			devices: []models.Device{
				{ID: "router-1", DeviceType: models.DeviceTypeRouter},
				{ID: "switch-dist", DeviceType: models.DeviceTypeSwitch},
				{ID: "switch-access", DeviceType: models.DeviceTypeSwitch},
				{ID: "desktop-1", DeviceType: models.DeviceTypeDesktop},
				{ID: "desktop-2", DeviceType: models.DeviceTypeDesktop},
			},
			links: []TopologyLink{
				// Router -> distribution switch
				{SourceDeviceID: "router-1", TargetDeviceID: "switch-dist", LinkType: "ethernet"},
				// Distribution -> access switch
				{SourceDeviceID: "switch-dist", TargetDeviceID: "switch-access", LinkType: "ethernet"},
				// Access switch -> endpoints via FDB
				{SourceDeviceID: "switch-access", TargetDeviceID: "desktop-1", LinkType: "fdb"},
				{SourceDeviceID: "switch-access", TargetDeviceID: "desktop-2", LinkType: "fdb"},
			},
			check: func(t *testing.T, result []HierarchyAssignment) {
				t.Helper()
				m := assignmentMap(result)

				assertLayer(t, m, "router-1", models.NetworkLayerGateway)
				assertLayer(t, m, "switch-dist", models.NetworkLayerDistribution)
				assertParent(t, m, "switch-dist", "router-1")

				assertLayer(t, m, "switch-access", models.NetworkLayerAccess)
				assertParent(t, m, "switch-access", "switch-dist")

				for _, id := range []string{"desktop-1", "desktop-2"} {
					assertLayer(t, m, id, models.NetworkLayerEndpoint)
					assertParent(t, m, id, "switch-access")
				}
			},
		},
		{
			name: "no router: graceful degradation to layer 0 for gateway, 4 for endpoints",
			devices: []models.Device{
				{ID: "server-1", DeviceType: models.DeviceTypeServer},
				{ID: "server-2", DeviceType: models.DeviceTypeServer},
			},
			links: []TopologyLink{
				{SourceDeviceID: "server-1", TargetDeviceID: "server-2", LinkType: "arp"},
			},
			check: func(t *testing.T, result []HierarchyAssignment) {
				t.Helper()
				m := assignmentMap(result)

				// No gateway found, so all are endpoints with no parent.
				assertLayer(t, m, "server-1", models.NetworkLayerEndpoint)
				assertLayer(t, m, "server-2", models.NetworkLayerEndpoint)
				assertParent(t, m, "server-1", "")
				assertParent(t, m, "server-2", "")
			},
		},
		{
			name: "access point gets layer 3 with switch parent",
			devices: []models.Device{
				{ID: "router-1", DeviceType: models.DeviceTypeRouter},
				{ID: "switch-1", DeviceType: models.DeviceTypeSwitch},
				{ID: "ap-1", DeviceType: models.DeviceTypeAccessPoint},
			},
			links: []TopologyLink{
				{SourceDeviceID: "router-1", TargetDeviceID: "switch-1", LinkType: "ethernet"},
				{SourceDeviceID: "switch-1", TargetDeviceID: "ap-1", LinkType: "ethernet"},
			},
			check: func(t *testing.T, result []HierarchyAssignment) {
				t.Helper()
				m := assignmentMap(result)

				assertLayer(t, m, "ap-1", models.NetworkLayerAccess)
				assertParent(t, m, "ap-1", "switch-1")
			},
		},
		{
			name: "firewall gets gateway layer with router as parent",
			devices: []models.Device{
				{ID: "fw-1", DeviceType: models.DeviceTypeFirewall},
				{ID: "router-1", DeviceType: models.DeviceTypeRouter},
				{ID: "server-1", DeviceType: models.DeviceTypeServer},
			},
			links: []TopologyLink{
				{SourceDeviceID: "fw-1", TargetDeviceID: "router-1", LinkType: "ethernet"},
				{SourceDeviceID: "server-1", TargetDeviceID: "router-1", LinkType: "arp"},
			},
			check: func(t *testing.T, result []HierarchyAssignment) {
				t.Helper()
				m := assignmentMap(result)

				assertLayer(t, m, "fw-1", models.NetworkLayerGateway)
				assertParent(t, m, "fw-1", "router-1")

				assertLayer(t, m, "router-1", models.NetworkLayerGateway)

				assertLayer(t, m, "server-1", models.NetworkLayerEndpoint)
				assertParent(t, m, "server-1", "router-1")
			},
		},
		{
			name: "two parallel switches under router",
			devices: []models.Device{
				{ID: "router-1", DeviceType: models.DeviceTypeRouter},
				{ID: "switch-a", DeviceType: models.DeviceTypeSwitch},
				{ID: "switch-b", DeviceType: models.DeviceTypeSwitch},
				{ID: "pc-1", DeviceType: models.DeviceTypeDesktop},
				{ID: "pc-2", DeviceType: models.DeviceTypeDesktop},
			},
			links: []TopologyLink{
				{SourceDeviceID: "router-1", TargetDeviceID: "switch-a", LinkType: "ethernet"},
				{SourceDeviceID: "router-1", TargetDeviceID: "switch-b", LinkType: "ethernet"},
				{SourceDeviceID: "switch-a", TargetDeviceID: "pc-1", LinkType: "fdb"},
				{SourceDeviceID: "switch-b", TargetDeviceID: "pc-2", LinkType: "fdb"},
			},
			check: func(t *testing.T, result []HierarchyAssignment) {
				t.Helper()
				m := assignmentMap(result)

				assertLayer(t, m, "switch-a", models.NetworkLayerDistribution)
				assertParent(t, m, "switch-a", "router-1")
				assertLayer(t, m, "switch-b", models.NetworkLayerDistribution)
				assertParent(t, m, "switch-b", "router-1")

				assertLayer(t, m, "pc-1", models.NetworkLayerEndpoint)
				assertParent(t, m, "pc-1", "switch-a")
				assertLayer(t, m, "pc-2", models.NetworkLayerEndpoint)
				assertParent(t, m, "pc-2", "switch-b")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := InferHierarchyFromData(tc.devices, tc.links)
			tc.check(t, result)
		})
	}
}

func TestInferHierarchy_Integration(t *testing.T) {
	_, store, _ := setupTestModule(t)
	ctx := context.Background()

	// Create a simple network: router + switch + 2 servers.
	router := &models.Device{ID: "r1", Hostname: "router", DeviceType: models.DeviceTypeRouter, IPAddresses: []string{"10.0.0.1"}, Status: models.DeviceStatusOnline}
	sw := &models.Device{ID: "s1", Hostname: "switch", DeviceType: models.DeviceTypeSwitch, IPAddresses: []string{"10.0.0.2"}, Status: models.DeviceStatusOnline}
	srv1 := &models.Device{ID: "srv1", Hostname: "server1", DeviceType: models.DeviceTypeServer, IPAddresses: []string{"10.0.0.10"}, Status: models.DeviceStatusOnline}
	srv2 := &models.Device{ID: "srv2", Hostname: "server2", DeviceType: models.DeviceTypeServer, IPAddresses: []string{"10.0.0.11"}, Status: models.DeviceStatusOnline}

	for _, d := range []*models.Device{router, sw, srv1, srv2} {
		if _, err := store.UpsertDevice(ctx, d); err != nil {
			t.Fatalf("upsert %s: %v", d.Hostname, err)
		}
	}

	// Create topology links.
	if err := store.UpsertTopologyLink(ctx, &TopologyLink{SourceDeviceID: "r1", TargetDeviceID: "s1", LinkType: "ethernet"}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertTopologyLink(ctx, &TopologyLink{SourceDeviceID: "s1", TargetDeviceID: "srv1", LinkType: "fdb"}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertTopologyLink(ctx, &TopologyLink{SourceDeviceID: "s1", TargetDeviceID: "srv2", LinkType: "fdb"}); err != nil {
		t.Fatal(err)
	}

	// Run hierarchy inference.
	inferrer := NewHierarchyInferrer(store, zapNop())
	if err := inferrer.InferHierarchy(ctx); err != nil {
		t.Fatalf("InferHierarchy: %v", err)
	}

	// Verify via store.
	tree, err := store.GetDeviceTree(ctx)
	if err != nil {
		t.Fatalf("GetDeviceTree: %v", err)
	}

	treeMap := make(map[string]DeviceTreeNode, len(tree))
	for _, n := range tree {
		treeMap[n.ID] = n
	}

	if got := treeMap["r1"].NetworkLayer; got != models.NetworkLayerGateway {
		t.Errorf("router layer: got %d, want %d", got, models.NetworkLayerGateway)
	}
	if got := treeMap["s1"].NetworkLayer; got != models.NetworkLayerDistribution {
		t.Errorf("switch layer: got %d, want %d", got, models.NetworkLayerDistribution)
	}
	if got := treeMap["s1"].ParentDeviceID; got != "r1" {
		t.Errorf("switch parent: got %q, want %q", got, "r1")
	}
	for _, id := range []string{"srv1", "srv2"} {
		if got := treeMap[id].NetworkLayer; got != models.NetworkLayerEndpoint {
			t.Errorf("%s layer: got %d, want %d", id, got, models.NetworkLayerEndpoint)
		}
		if got := treeMap[id].ParentDeviceID; got != "s1" {
			t.Errorf("%s parent: got %q, want %q", id, got, "s1")
		}
	}

	// Verify child count.
	if got := treeMap["s1"].ChildCount; got != 2 {
		t.Errorf("switch child count: got %d, want 2", got)
	}
}

// Helpers

func assignmentMap(result []HierarchyAssignment) map[string]HierarchyAssignment {
	m := make(map[string]HierarchyAssignment, len(result))
	for _, a := range result {
		m[a.DeviceID] = a
	}
	return m
}

func assertLayer(t *testing.T, m map[string]HierarchyAssignment, deviceID string, expectedLayer int) {
	t.Helper()
	a, ok := m[deviceID]
	if !ok {
		t.Errorf("device %q not found in assignments", deviceID)
		return
	}
	if a.NetworkLayer != expectedLayer {
		t.Errorf("device %q: layer = %d, want %d", deviceID, a.NetworkLayer, expectedLayer)
	}
}

func assertParent(t *testing.T, m map[string]HierarchyAssignment, deviceID, expectedParent string) {
	t.Helper()
	a, ok := m[deviceID]
	if !ok {
		t.Errorf("device %q not found in assignments", deviceID)
		return
	}
	if a.ParentDeviceID != expectedParent {
		t.Errorf("device %q: parent = %q, want %q", deviceID, a.ParentDeviceID, expectedParent)
	}
}

func zapNop() *zap.Logger {
	return zap.NewNop()
}
