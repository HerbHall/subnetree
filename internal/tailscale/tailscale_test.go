package tailscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"go.uber.org/zap"
)

func TestContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

// --- Mock store ---

type mockDeviceStore struct {
	mu      sync.Mutex
	devices []models.Device
}

func (s *mockDeviceStore) UpsertDevice(_ context.Context, d *models.Device) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.devices {
		if s.devices[i].ID == d.ID && d.ID != "" {
			s.devices[i] = *d
			return false, nil
		}
		if s.devices[i].Hostname != "" && strings.EqualFold(s.devices[i].Hostname, d.Hostname) {
			s.devices[i] = *d
			return false, nil
		}
	}
	if d.ID == "" {
		d.ID = fmt.Sprintf("dev-%d", len(s.devices)+1)
	}
	s.devices = append(s.devices, *d)
	return true, nil
}

func (s *mockDeviceStore) ListDevices(_ context.Context, limit, offset int) ([]models.Device, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	total := len(s.devices)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	result := make([]models.Device, end-offset)
	copy(result, s.devices[offset:end])
	return result, total, nil
}

func (s *mockDeviceStore) GetDeviceByMAC(_ context.Context, mac string) (*models.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.devices {
		if s.devices[i].MACAddress == mac {
			d := s.devices[i]
			return &d, nil
		}
	}
	return nil, nil
}

// --- Syncer tests ---

func newTestSyncer(store *mockDeviceStore) *Syncer {
	logger, _ := zap.NewDevelopment()
	return NewSyncer(store, logger)
}

func newTestServer(t *testing.T, devices []TailscaleDevice) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(devicesResponse{Devices: devices})
	}))
}

func TestSyncer_NewDevices(t *testing.T) {
	tsDevices := []TailscaleDevice{
		{
			ID:        "ts-1",
			Name:      "web-server.tail123.ts.net",
			Addresses: []string{"100.64.0.1"},
			Hostname:  "web-server",
			OS:        "linux",
			Online:    true,
			NodeKey:   "nodekey:abc",
		},
		{
			ID:        "ts-2",
			Name:      "db-server.tail123.ts.net",
			Addresses: []string{"100.64.0.2"},
			Hostname:  "db-server",
			OS:        "linux",
			Online:    false,
			NodeKey:   "nodekey:def",
		},
	}

	srv := newTestServer(t, tsDevices)
	defer srv.Close()

	store := &mockDeviceStore{}
	syncer := newTestSyncer(store)
	client := NewClient("test-key", srv.URL, "-")

	result, err := syncer.Sync(context.Background(), client)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.DevicesFound != 2 {
		t.Errorf("DevicesFound = %d, want 2", result.DevicesFound)
	}
	if result.Created != 2 {
		t.Errorf("Created = %d, want 2", result.Created)
	}
	if result.Updated != 0 {
		t.Errorf("Updated = %d, want 0", result.Updated)
	}

	// Verify devices in store.
	devices, total, _ := store.ListDevices(context.Background(), 100, 0)
	if total != 2 {
		t.Fatalf("store has %d devices, want 2", total)
	}
	for _, d := range devices {
		if d.DiscoveryMethod != models.DiscoveryTailscale {
			t.Errorf("device %s discovery_method = %s, want tailscale", d.Hostname, d.DiscoveryMethod)
		}
		if d.CustomFields["tailscale_node_key"] == "" {
			t.Errorf("device %s missing tailscale_node_key custom field", d.Hostname)
		}
	}
}

func TestSyncer_HostnameMatch(t *testing.T) {
	// Pre-existing device with the same hostname.
	store := &mockDeviceStore{
		devices: []models.Device{
			{
				ID:          "existing-1",
				Hostname:    "web-server",
				IPAddresses: []string{"192.168.1.10"},
				Status:      models.DeviceStatusOnline,
			},
		},
	}

	tsDevices := []TailscaleDevice{
		{
			ID:        "ts-1",
			Name:      "web-server.tail123.ts.net",
			Addresses: []string{"100.64.0.1"},
			Hostname:  "web-server",
			OS:        "linux",
			Online:    true,
			NodeKey:   "nodekey:abc",
		},
	}

	srv := newTestServer(t, tsDevices)
	defer srv.Close()

	syncer := newTestSyncer(store)
	client := NewClient("test-key", srv.URL, "-")

	result, err := syncer.Sync(context.Background(), client)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.Created != 0 {
		t.Errorf("Created = %d, want 0 (should merge)", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}

	// Verify merged IPs.
	devices, _, _ := store.ListDevices(context.Background(), 100, 0)
	if len(devices) != 1 {
		t.Fatalf("store has %d devices, want 1", len(devices))
	}
	d := devices[0]
	hasOrigIP := false
	hasTSIP := false
	for _, ip := range d.IPAddresses {
		if ip == "192.168.1.10" {
			hasOrigIP = true
		}
		if ip == "100.64.0.1" {
			hasTSIP = true
		}
	}
	if !hasOrigIP || !hasTSIP {
		t.Errorf("merged IPs = %v, want both 192.168.1.10 and 100.64.0.1", d.IPAddresses)
	}
}

func TestSyncer_IPMatch(t *testing.T) {
	// Pre-existing device sharing a Tailscale IP.
	store := &mockDeviceStore{
		devices: []models.Device{
			{
				ID:          "existing-1",
				Hostname:    "different-name",
				IPAddresses: []string{"100.64.0.1"},
				Status:      models.DeviceStatusOnline,
			},
		},
	}

	tsDevices := []TailscaleDevice{
		{
			ID:        "ts-1",
			Name:      "web-server.tail123.ts.net",
			Addresses: []string{"100.64.0.1", "fd7a::1"},
			Hostname:  "web-server",
			OS:        "linux",
			Online:    true,
			NodeKey:   "nodekey:abc",
		},
	}

	srv := newTestServer(t, tsDevices)
	defer srv.Close()

	syncer := newTestSyncer(store)
	client := NewClient("test-key", srv.URL, "-")

	result, err := syncer.Sync(context.Background(), client)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if result.Created != 0 {
		t.Errorf("Created = %d, want 0 (should merge by IP)", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("Updated = %d, want 1", result.Updated)
	}
}

func TestSyncer_MarkUnseen(t *testing.T) {
	// Pre-existing Tailscale device that is no longer in the tailnet.
	store := &mockDeviceStore{
		devices: []models.Device{
			{
				ID:              "existing-ts-1",
				Hostname:        "removed-host",
				IPAddresses:     []string{"100.64.0.99"},
				Status:          models.DeviceStatusOnline,
				DiscoveryMethod: models.DiscoveryTailscale,
			},
		},
	}

	// Tailscale returns a different device.
	tsDevices := []TailscaleDevice{
		{
			ID:        "ts-new",
			Name:      "new-host.tail123.ts.net",
			Addresses: []string{"100.64.0.2"},
			Hostname:  "new-host",
			OS:        "linux",
			Online:    true,
			NodeKey:   "nodekey:xyz",
		},
	}

	srv := newTestServer(t, tsDevices)
	defer srv.Close()

	syncer := newTestSyncer(store)
	client := NewClient("test-key", srv.URL, "-")

	_, err := syncer.Sync(context.Background(), client)
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	// The old Tailscale device should now be offline.
	devices, _, _ := store.ListDevices(context.Background(), 100, 0)
	for _, d := range devices {
		if d.ID == "existing-ts-1" {
			if d.Status != models.DeviceStatusOffline {
				t.Errorf("unseen device status = %s, want offline", d.Status)
			}
			return
		}
	}
	t.Error("existing Tailscale device not found in store after sync")
}

func TestExtractShortHostname(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"myhost.tail12345.ts.net", "myhost"},
		{"simple", "simple"},
		{"host.domain.com", "host"},
		{"", ""},
	}
	for _, tc := range tests {
		got := extractShortHostname(tc.input)
		if got != tc.want {
			t.Errorf("extractShortHostname(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
