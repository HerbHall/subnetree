package recon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

func TestProxmoxSyncer_Sync(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create the parent host device.
	host := &models.Device{
		ID:              "pve-host-1",
		Hostname:        "proxmox-host",
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, host); err != nil {
		t.Fatalf("upsert host: %v", err)
	}

	// Mock Proxmox API server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp any
		switch r.URL.Path {
		case "/api2/json/nodes":
			resp = map[string]any{
				"data": []map[string]any{
					{"node": "pve1", "status": "online", "cpu": 0.1, "maxcpu": 8,
						"mem": 4294967296, "maxmem": 17179869184,
						"disk": 10737418240, "maxdisk": 500107862016, "uptime": 100000},
				},
			}
		case "/api2/json/nodes/pve1/qemu":
			resp = map[string]any{
				"data": []map[string]any{
					{"vmid": 100, "name": "web-server", "status": "running", "cpus": 4, "maxmem": 8589934592},
					{"vmid": 101, "name": "stopped-vm", "status": "stopped", "cpus": 2, "maxmem": 4294967296},
				},
			}
		case "/api2/json/nodes/pve1/lxc":
			resp = map[string]any{
				"data": []map[string]any{
					{"vmid": 200, "name": "nginx-ct", "status": "running", "cpus": 2, "maxmem": 536870912},
				},
			}
		case "/api2/json/nodes/pve1/qemu/100/status/current":
			resp = map[string]any{
				"data": map[string]any{
					"cpu": 0.35, "mem": 4294967296, "maxmem": 8589934592,
					"disk": 5368709120, "maxdisk": 53687091200,
					"uptime": 86400, "netin": 1000000, "netout": 500000,
				},
			}
		case "/api2/json/nodes/pve1/lxc/200/status/current":
			resp = map[string]any{
				"data": map[string]any{
					"cpu": 0.05, "mem": 268435456, "maxmem": 536870912,
					"disk": 1073741824, "maxdisk": 10737418240,
					"uptime": 3600, "netin": 100000, "netout": 50000,
				},
			}
		default:
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	collector := NewProxmoxCollector(srv.URL, "test@pve!token", "secret", zap.NewNop())
	syncer := NewProxmoxSyncer(s, zap.NewNop())

	result, err := syncer.Sync(ctx, collector, "pve-host-1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Verify sync result counts.
	if result.NodesScanned != 1 {
		t.Errorf("NodesScanned = %d, want 1", result.NodesScanned)
	}
	if result.VMsFound != 2 {
		t.Errorf("VMsFound = %d, want 2", result.VMsFound)
	}
	if result.LXCsFound != 1 {
		t.Errorf("LXCsFound = %d, want 1", result.LXCsFound)
	}
	if result.Created != 3 {
		t.Errorf("Created = %d, want 3", result.Created)
	}

	// Verify running VM has resource snapshot.
	webVM, err := s.FindDeviceByHostnameAndParent(ctx, "web-server", "pve-host-1")
	if err != nil {
		t.Fatalf("find web-server: %v", err)
	}
	if webVM == nil {
		t.Fatal("expected web-server device to exist")
	}
	if webVM.Status != models.DeviceStatusOnline {
		t.Errorf("web-server status = %q, want online", webVM.Status)
	}
	if webVM.DeviceType != models.DeviceTypeVM {
		t.Errorf("web-server type = %q, want virtual_machine", webVM.DeviceType)
	}

	res, err := s.GetProxmoxResource(ctx, webVM.ID)
	if err != nil {
		t.Fatalf("get web-server resource: %v", err)
	}
	if res == nil {
		t.Fatal("expected resource snapshot for running VM")
	}
	if res.CPUPercent != 35.0 {
		t.Errorf("CPUPercent = %f, want 35.0", res.CPUPercent)
	}

	// Verify stopped VM exists but has no resource snapshot.
	stoppedVM, err := s.FindDeviceByHostnameAndParent(ctx, "stopped-vm", "pve-host-1")
	if err != nil {
		t.Fatalf("find stopped-vm: %v", err)
	}
	if stoppedVM == nil {
		t.Fatal("expected stopped-vm device to exist")
	}
	if stoppedVM.Status != models.DeviceStatusOffline {
		t.Errorf("stopped-vm status = %q, want offline", stoppedVM.Status)
	}

	stoppedRes, err := s.GetProxmoxResource(ctx, stoppedVM.ID)
	if err != nil {
		t.Fatalf("get stopped-vm resource: %v", err)
	}
	if stoppedRes != nil {
		t.Error("expected no resource snapshot for stopped VM")
	}

	// Verify container has resource snapshot.
	nginxCT, err := s.FindDeviceByHostnameAndParent(ctx, "nginx-ct", "pve-host-1")
	if err != nil {
		t.Fatalf("find nginx-ct: %v", err)
	}
	if nginxCT == nil {
		t.Fatal("expected nginx-ct device to exist")
	}
	if nginxCT.DeviceType != models.DeviceTypeContainer {
		t.Errorf("nginx-ct type = %q, want container", nginxCT.DeviceType)
	}

	ctRes, err := s.GetProxmoxResource(ctx, nginxCT.ID)
	if err != nil {
		t.Fatalf("get nginx-ct resource: %v", err)
	}
	if ctRes == nil {
		t.Fatal("expected resource snapshot for running container")
	}
	if ctRes.CPUPercent != 5.0 {
		t.Errorf("container CPUPercent = %f, want 5.0", ctRes.CPUPercent)
	}
}

func TestProxmoxSyncer_Sync_UpdateExisting(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	host := &models.Device{
		ID:              "pve-host-1",
		Hostname:        "proxmox-host",
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, host); err != nil {
		t.Fatalf("upsert host: %v", err)
	}

	// Pre-create a VM that the sync will update.
	existing := &models.Device{
		ID:              "existing-vm",
		Hostname:        "web-server",
		DeviceType:      models.DeviceTypeVM,
		Status:          models.DeviceStatusOffline,
		DiscoveryMethod: models.DiscoveryProxmox,
		ParentDeviceID:  "pve-host-1",
	}
	if _, err := s.UpsertDevice(ctx, existing); err != nil {
		t.Fatalf("upsert existing: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp any
		switch r.URL.Path {
		case "/api2/json/nodes":
			resp = map[string]any{
				"data": []map[string]any{
					{"node": "pve1", "status": "online", "cpu": 0.1, "maxcpu": 4,
						"mem": 4294967296, "maxmem": 8589934592,
						"disk": 1073741824, "maxdisk": 107374182400, "uptime": 100},
				},
			}
		case "/api2/json/nodes/pve1/qemu":
			resp = map[string]any{
				"data": []map[string]any{
					{"vmid": 100, "name": "web-server", "status": "running", "cpus": 4, "maxmem": 8589934592},
				},
			}
		case "/api2/json/nodes/pve1/lxc":
			resp = map[string]any{"data": []map[string]any{}}
		case "/api2/json/nodes/pve1/qemu/100/status/current":
			resp = map[string]any{
				"data": map[string]any{
					"cpu": 0.5, "mem": 4294967296, "maxmem": 8589934592,
					"disk": 5368709120, "maxdisk": 53687091200,
					"uptime": 100, "netin": 100, "netout": 50,
				},
			}
		default:
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	collector := NewProxmoxCollector(srv.URL, "test@pve!token", "secret", zap.NewNop())
	syncer := NewProxmoxSyncer(s, zap.NewNop())

	result, err := syncer.Sync(ctx, collector, "pve-host-1")
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if result.Created != 0 {
		t.Errorf("Created = %d, want 0 (existing device should be updated)", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}

	// Verify status was updated from offline to online.
	updated, err := s.FindDeviceByHostnameAndParent(ctx, "web-server", "pve-host-1")
	if err != nil {
		t.Fatalf("find updated device: %v", err)
	}
	if updated.Status != models.DeviceStatusOnline {
		t.Errorf("status = %q, want online", updated.Status)
	}
}

func TestProxmoxSyncer_MarkUnseen(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	host := &models.Device{
		ID:              "pve-host-1",
		Hostname:        "proxmox-host",
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	if _, err := s.UpsertDevice(ctx, host); err != nil {
		t.Fatalf("upsert host: %v", err)
	}

	// Create two devices: one will be "seen", one will not.
	for _, d := range []struct {
		id, hostname string
	}{
		{"vm-seen", "seen-vm"},
		{"vm-gone", "gone-vm"},
	} {
		dev := &models.Device{
			ID:              d.id,
			Hostname:        d.hostname,
			DeviceType:      models.DeviceTypeVM,
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryProxmox,
			ParentDeviceID:  "pve-host-1",
		}
		if _, err := s.UpsertDevice(ctx, dev); err != nil {
			t.Fatalf("upsert %s: %v", d.id, err)
		}
	}

	// Sync that only returns "seen-vm".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp any
		switch r.URL.Path {
		case "/api2/json/nodes":
			resp = map[string]any{
				"data": []map[string]any{
					{"node": "pve1", "status": "online", "cpu": 0.1, "maxcpu": 4,
						"mem": 4294967296, "maxmem": 8589934592,
						"disk": 1073741824, "maxdisk": 107374182400, "uptime": 100},
				},
			}
		case "/api2/json/nodes/pve1/qemu":
			resp = map[string]any{
				"data": []map[string]any{
					{"vmid": 100, "name": "seen-vm", "status": "running", "cpus": 2, "maxmem": 4294967296},
				},
			}
		case "/api2/json/nodes/pve1/lxc":
			resp = map[string]any{"data": []map[string]any{}}
		case "/api2/json/nodes/pve1/qemu/100/status/current":
			resp = map[string]any{
				"data": map[string]any{
					"cpu": 0.1, "mem": 1073741824, "maxmem": 4294967296,
					"disk": 1073741824, "maxdisk": 10737418240,
					"uptime": 100, "netin": 0, "netout": 0,
				},
			}
		default:
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	collector := NewProxmoxCollector(srv.URL, "test@pve!token", "secret", zap.NewNop())
	syncer := NewProxmoxSyncer(s, zap.NewNop())

	if _, err := syncer.Sync(ctx, collector, "pve-host-1"); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	// Verify "gone-vm" was marked offline.
	goneDevice, err := s.FindDeviceByHostnameAndParent(ctx, "gone-vm", "pve-host-1")
	if err != nil {
		t.Fatalf("find gone-vm: %v", err)
	}
	if goneDevice == nil {
		t.Fatal("expected gone-vm to still exist")
	}
	if goneDevice.Status != models.DeviceStatusOffline {
		t.Errorf("gone-vm status = %q, want offline", goneDevice.Status)
	}

	// Verify "seen-vm" remains online.
	seenDevice, err := s.FindDeviceByHostnameAndParent(ctx, "seen-vm", "pve-host-1")
	if err != nil {
		t.Fatalf("find seen-vm: %v", err)
	}
	if seenDevice == nil {
		t.Fatal("expected seen-vm to exist")
	}
	if seenDevice.Status != models.DeviceStatusOnline {
		t.Errorf("seen-vm status = %q, want online", seenDevice.Status)
	}
}
