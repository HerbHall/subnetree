package netbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// newMockNetBox creates a test HTTP server that mimics NetBox API responses.
// Returns the server and a map of recorded request paths for verification.
func newMockNetBox(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var requests []string

	mux := http.NewServeMux()

	// Tags
	mux.HandleFunc("GET /api/extras/tags/", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, "GET /api/extras/tags/")
		slug := r.URL.Query().Get("slug")
		if slug == "subnetree-managed" {
			writeTestJSON(w, ListResponse[NBTag]{Count: 1, Results: []NBTag{{ID: 10, Name: "subnetree-managed", Slug: "subnetree-managed"}}})
			return
		}
		writeTestJSON(w, ListResponse[NBTag]{Count: 0, Results: []NBTag{}})
	})
	mux.HandleFunc("POST /api/extras/tags/", func(w http.ResponseWriter, _ *http.Request) {
		requests = append(requests, "POST /api/extras/tags/")
		w.WriteHeader(http.StatusCreated)
		writeTestJSON(w, NBTag{ID: 99, Name: "new-tag", Slug: "new-tag"})
	})

	// Sites
	mux.HandleFunc("GET /api/dcim/sites/", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, "GET /api/dcim/sites/")
		slug := r.URL.Query().Get("slug")
		if slug == "home-lab" {
			writeTestJSON(w, ListResponse[NBSite]{Count: 1, Results: []NBSite{{ID: 1, Name: "Home Lab", Slug: "home-lab"}}})
			return
		}
		writeTestJSON(w, ListResponse[NBSite]{Count: 0, Results: []NBSite{}})
	})
	mux.HandleFunc("POST /api/dcim/sites/", func(w http.ResponseWriter, _ *http.Request) {
		requests = append(requests, "POST /api/dcim/sites/")
		w.WriteHeader(http.StatusCreated)
		writeTestJSON(w, NBSite{ID: 50, Name: "New Site", Slug: "new-site"})
	})

	// Manufacturers
	mux.HandleFunc("GET /api/dcim/manufacturers/", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, "GET /api/dcim/manufacturers/")
		slug := r.URL.Query().Get("slug")
		if slug == "dell-inc" {
			writeTestJSON(w, ListResponse[NBManufacturer]{Count: 1, Results: []NBManufacturer{{ID: 5, Name: "Dell Inc.", Slug: "dell-inc"}}})
			return
		}
		writeTestJSON(w, ListResponse[NBManufacturer]{Count: 0, Results: []NBManufacturer{}})
	})
	mux.HandleFunc("POST /api/dcim/manufacturers/", func(w http.ResponseWriter, _ *http.Request) {
		requests = append(requests, "POST /api/dcim/manufacturers/")
		w.WriteHeader(http.StatusCreated)
		writeTestJSON(w, NBManufacturer{ID: 20, Name: "New Mfr", Slug: "new-mfr"})
	})

	// Device Types
	mux.HandleFunc("GET /api/dcim/device-types/", func(w http.ResponseWriter, _ *http.Request) {
		requests = append(requests, "GET /api/dcim/device-types/")
		writeTestJSON(w, ListResponse[NBDeviceType]{Count: 0, Results: []NBDeviceType{}})
	})
	mux.HandleFunc("POST /api/dcim/device-types/", func(w http.ResponseWriter, _ *http.Request) {
		requests = append(requests, "POST /api/dcim/device-types/")
		w.WriteHeader(http.StatusCreated)
		writeTestJSON(w, NBDeviceType{ID: 30, Model: "server", Slug: "server"})
	})

	// Device Roles
	mux.HandleFunc("GET /api/dcim/device-roles/", func(w http.ResponseWriter, _ *http.Request) {
		requests = append(requests, "GET /api/dcim/device-roles/")
		writeTestJSON(w, ListResponse[NBDeviceRole]{Count: 0, Results: []NBDeviceRole{}})
	})
	mux.HandleFunc("POST /api/dcim/device-roles/", func(w http.ResponseWriter, _ *http.Request) {
		requests = append(requests, "POST /api/dcim/device-roles/")
		w.WriteHeader(http.StatusCreated)
		writeTestJSON(w, NBDeviceRole{ID: 40, Name: "Server", Slug: "server"})
	})

	// Devices
	mux.HandleFunc("GET /api/dcim/devices/", func(w http.ResponseWriter, _ *http.Request) {
		requests = append(requests, "GET /api/dcim/devices/")
		writeTestJSON(w, ListResponse[NBDevice]{Count: 0, Results: []NBDevice{}})
	})
	mux.HandleFunc("POST /api/dcim/devices/", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, "POST /api/dcim/devices/")
		var req NBDeviceCreateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		writeTestJSON(w, NBDevice{
			ID:   100,
			Name: req.Name,
			Status: &NBStatusValue{Value: req.Status},
		})
	})

	// Interfaces
	mux.HandleFunc("POST /api/dcim/interfaces/", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, "POST /api/dcim/interfaces/")
		var req NBInterfaceCreateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		writeTestJSON(w, NBInterface{ID: 200, Name: req.Name, MACAddress: req.MACAddress})
	})

	// IP Addresses
	mux.HandleFunc("POST /api/ipam/ip-addresses/", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, "POST /api/ipam/ip-addresses/")
		var req NBIPAddressCreateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		writeTestJSON(w, NBIPAddress{ID: 300, Address: req.Address})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &requests
}

func writeTestJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func TestCreateDevice(t *testing.T) {
	srv, requests := newMockNetBox(t)
	client := NewClient(srv.URL, "test-token", 5*time.Second)

	req := NBDeviceCreateRequest{
		Name:       "test-device",
		DeviceType: 1,
		Role:       2,
		Site:       3,
		Status:     "active",
	}

	device, err := client.CreateDevice(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateDevice error: %v", err)
	}
	if device.ID != 100 {
		t.Errorf("device ID = %d, want 100", device.ID)
	}
	if device.Name != "test-device" {
		t.Errorf("device Name = %q, want %q", device.Name, "test-device")
	}

	found := false
	for _, r := range *requests {
		if r == "POST /api/dcim/devices/" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected POST /api/dcim/devices/ request")
	}
}

func TestGetOrCreateManufacturer_Exists(t *testing.T) {
	srv, requests := newMockNetBox(t)
	client := NewClient(srv.URL, "test-token", 5*time.Second)

	id, err := client.GetOrCreateManufacturer(context.Background(), "Dell Inc.")
	if err != nil {
		t.Fatalf("GetOrCreateManufacturer error: %v", err)
	}
	if id != 5 {
		t.Errorf("manufacturer ID = %d, want 5", id)
	}

	// Should only GET, not POST.
	for _, r := range *requests {
		if r == "POST /api/dcim/manufacturers/" {
			t.Error("should not POST when manufacturer exists")
		}
	}
}

func TestGetOrCreateManufacturer_Creates(t *testing.T) {
	srv, requests := newMockNetBox(t)
	client := NewClient(srv.URL, "test-token", 5*time.Second)

	id, err := client.GetOrCreateManufacturer(context.Background(), "New Manufacturer")
	if err != nil {
		t.Fatalf("GetOrCreateManufacturer error: %v", err)
	}
	if id != 20 {
		t.Errorf("manufacturer ID = %d, want 20", id)
	}

	hasPost := false
	for _, r := range *requests {
		if r == "POST /api/dcim/manufacturers/" {
			hasPost = true
		}
	}
	if !hasPost {
		t.Error("expected POST /api/dcim/manufacturers/ when creating new")
	}
}

func TestGetOrCreateDeviceRole(t *testing.T) {
	srv, _ := newMockNetBox(t)
	client := NewClient(srv.URL, "test-token", 5*time.Second)

	id, err := client.GetOrCreateDeviceRole(context.Background(), "Server")
	if err != nil {
		t.Fatalf("GetOrCreateDeviceRole error: %v", err)
	}
	if id != 40 {
		t.Errorf("role ID = %d, want 40", id)
	}
}

func TestEnsureTag_Exists(t *testing.T) {
	srv, requests := newMockNetBox(t)
	client := NewClient(srv.URL, "test-token", 5*time.Second)

	id, err := client.EnsureTag(context.Background(), "subnetree-managed")
	if err != nil {
		t.Fatalf("EnsureTag error: %v", err)
	}
	if id != 10 {
		t.Errorf("tag ID = %d, want 10", id)
	}

	for _, r := range *requests {
		if r == "POST /api/extras/tags/" {
			t.Error("should not POST when tag exists")
		}
	}
}

func TestEnsureTag_Creates(t *testing.T) {
	srv, _ := newMockNetBox(t)
	client := NewClient(srv.URL, "test-token", 5*time.Second)

	id, err := client.EnsureTag(context.Background(), "brand-new-tag")
	if err != nil {
		t.Fatalf("EnsureTag error: %v", err)
	}
	if id != 99 {
		t.Errorf("tag ID = %d, want 99", id)
	}
}

func TestSyncAll(t *testing.T) {
	srv, requests := newMockNetBox(t)
	client := NewClient(srv.URL, "test-token", 5*time.Second)

	devices := []models.Device{
		{
			ID:           "dev-001",
			Hostname:     "web-server-01",
			IPAddresses:  []string{"192.168.1.10"},
			MACAddress:   "00:1a:2b:3c:4d:5e",
			Manufacturer: "Dell Inc.",
			DeviceType:   models.DeviceTypeServer,
			Status:       models.DeviceStatusOnline,
		},
		{
			ID:           "dev-002",
			Hostname:     "switch-01",
			IPAddresses:  []string{"192.168.1.1"},
			Manufacturer: "Unknown",
			DeviceType:   models.DeviceTypeSwitch,
			Status:       models.DeviceStatusOnline,
		},
	}

	reader := &mockDeviceReader{devices: devices}
	cfg := Config{
		URL:      srv.URL,
		Token:    "test-token",
		SiteName: "Home Lab",
		TagName:  "subnetree-managed",
	}

	engine := newSyncEngine(client, reader, cfg, zapNop())
	result, err := engine.SyncAll(context.Background(), false)
	if err != nil {
		t.Fatalf("SyncAll error: %v", err)
	}

	if result.Created != 2 {
		t.Errorf("Created = %d, want 2", result.Created)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0 (errors: %v)", result.Failed, result.Errors)
	}

	// Verify key API calls were made.
	hasTagGet := false
	hasSiteGet := false
	hasDeviceCreate := false
	hasInterfaceCreate := false
	hasIPCreate := false
	for _, r := range *requests {
		switch {
		case r == "GET /api/extras/tags/":
			hasTagGet = true
		case r == "GET /api/dcim/sites/":
			hasSiteGet = true
		case r == "POST /api/dcim/devices/":
			hasDeviceCreate = true
		case r == "POST /api/dcim/interfaces/":
			hasInterfaceCreate = true
		case r == "POST /api/ipam/ip-addresses/":
			hasIPCreate = true
		}
	}

	if !hasTagGet {
		t.Error("expected GET /api/extras/tags/")
	}
	if !hasSiteGet {
		t.Error("expected GET /api/dcim/sites/")
	}
	if !hasDeviceCreate {
		t.Error("expected POST /api/dcim/devices/")
	}
	if !hasInterfaceCreate {
		t.Error("expected POST /api/dcim/interfaces/")
	}
	if !hasIPCreate {
		t.Error("expected POST /api/ipam/ip-addresses/")
	}
}

func TestSyncAll_DryRun(t *testing.T) {
	srv, requests := newMockNetBox(t)
	client := NewClient(srv.URL, "test-token", 5*time.Second)

	devices := []models.Device{
		{
			ID:         "dev-001",
			Hostname:   "server-01",
			DeviceType: models.DeviceTypeServer,
			Status:     models.DeviceStatusOnline,
		},
	}

	reader := &mockDeviceReader{devices: devices}
	cfg := Config{
		URL:      srv.URL,
		Token:    "test-token",
		SiteName: "Home Lab",
		TagName:  "subnetree-managed",
	}

	engine := newSyncEngine(client, reader, cfg, zapNop())
	result, err := engine.SyncAll(context.Background(), true)
	if err != nil {
		t.Fatalf("SyncAll dry-run error: %v", err)
	}

	if !result.DryRun {
		t.Error("expected DryRun = true")
	}
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1", result.Created)
	}

	// Verify no device was actually created.
	for _, r := range *requests {
		if r == "POST /api/dcim/devices/" {
			t.Error("dry run should not POST devices")
		}
	}
}

func TestClientAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeTestJSON(w, ListResponse[NBTag]{Count: 0, Results: []NBTag{}})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "my-secret-token", 5*time.Second)
	_, _ = client.EnsureTag(context.Background(), "test")

	if gotAuth != "Token my-secret-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Token my-secret-token")
	}
}

func TestClientErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"detail":"Authentication credentials were not provided."}`))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "bad-token", 5*time.Second)
	_, err := client.EnsureTag(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention 403, got: %v", err)
	}
}

// mockDeviceReader implements DeviceReader for testing.
type mockDeviceReader struct {
	devices []models.Device
}

func (m *mockDeviceReader) ListAllDevices(_ context.Context) ([]models.Device, error) {
	return m.devices, nil
}

func (m *mockDeviceReader) GetDevice(_ context.Context, id string) (*models.Device, error) {
	for i := range m.devices {
		if m.devices[i].ID == id {
			return &m.devices[i], nil
		}
	}
	return nil, nil
}

// zapNop returns a no-op zap logger for tests.
func zapNop() *zap.Logger {
	return zap.NewNop()
}
