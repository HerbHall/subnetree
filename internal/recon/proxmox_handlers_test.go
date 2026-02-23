package recon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// newTestModuleWithProxmox creates a Module with proxmoxSyncer initialised.
func newTestModuleWithProxmox(t *testing.T) *Module {
	t.Helper()

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	logger := zap.NewNop()
	reconStore := NewReconStore(db.DB())

	m := &Module{
		logger:        logger,
		store:         reconStore,
		proxmoxSyncer: NewProxmoxSyncer(reconStore, logger.Named("proxmox-sync")),
	}
	return m
}

func TestHandleListProxmoxVMs_Empty(t *testing.T) {
	m := newTestModuleWithProxmox(t)

	req := httptest.NewRequest("GET", "/recon/proxmox/vms", http.NoBody)
	w := httptest.NewRecorder()
	m.handleListProxmoxVMs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result []ProxmoxResource
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestHandleListProxmoxVMs_WithData(t *testing.T) {
	m := newTestModuleWithProxmox(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Seed a host and VM with resource data.
	host := &models.Device{
		ID:              "host-1",
		Hostname:        "pve-host",
		DeviceType:      models.DeviceTypeServer,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
		FirstSeen:       now,
		LastSeen:        now,
	}
	if _, err := m.store.UpsertDevice(ctx, host); err != nil {
		t.Fatalf("upsert host: %v", err)
	}

	vm := &models.Device{
		ID:              "vm-1",
		Hostname:        "web-vm",
		DeviceType:      models.DeviceTypeVM,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryProxmox,
		ParentDeviceID:  "host-1",
		FirstSeen:       now,
		LastSeen:        now,
	}
	if _, err := m.store.UpsertDevice(ctx, vm); err != nil {
		t.Fatalf("upsert vm: %v", err)
	}

	r := &ProxmoxResource{
		DeviceID:    "vm-1",
		CPUPercent:  15.0,
		MemUsedMB:   2048,
		MemTotalMB:  4096,
		DiskUsedGB:  10,
		DiskTotalGB: 50,
		UptimeSec:   86400,
		CollectedAt: now,
	}
	if err := m.store.UpsertProxmoxResource(ctx, r); err != nil {
		t.Fatalf("upsert resource: %v", err)
	}

	req := httptest.NewRequest("GET", "/recon/proxmox/vms?parent_id=host-1", http.NoBody)
	w := httptest.NewRecorder()
	m.handleListProxmoxVMs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result []ProxmoxResource
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].DeviceName != "web-vm" {
		t.Errorf("DeviceName = %q, want %q", result[0].DeviceName, "web-vm")
	}
	if result[0].CPUPercent != 15.0 {
		t.Errorf("CPUPercent = %f, want 15.0", result[0].CPUPercent)
	}
}

func TestHandleGetProxmoxVMResources_Found(t *testing.T) {
	m := newTestModuleWithProxmox(t)
	ctx := context.Background()
	now := time.Now().UTC()

	vm := &models.Device{
		ID:              "vm-1",
		Hostname:        "test-vm",
		DeviceType:      models.DeviceTypeVM,
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryProxmox,
		FirstSeen:       now,
		LastSeen:        now,
	}
	if _, err := m.store.UpsertDevice(ctx, vm); err != nil {
		t.Fatalf("upsert vm: %v", err)
	}

	r := &ProxmoxResource{
		DeviceID:    "vm-1",
		CPUPercent:  42.0,
		MemUsedMB:   8192,
		MemTotalMB:  16384,
		DiskUsedGB:  25,
		DiskTotalGB: 100,
		UptimeSec:   172800,
		NetInBytes:  5000000,
		NetOutBytes: 2500000,
		CollectedAt: now,
	}
	if err := m.store.UpsertProxmoxResource(ctx, r); err != nil {
		t.Fatalf("upsert resource: %v", err)
	}

	req := httptest.NewRequest("GET", "/recon/proxmox/vms/vm-1/resources", http.NoBody)
	req.SetPathValue("id", "vm-1")
	w := httptest.NewRecorder()
	m.handleGetProxmoxVMResources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result ProxmoxResource
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.CPUPercent != 42.0 {
		t.Errorf("CPUPercent = %f, want 42.0", result.CPUPercent)
	}
	if result.MemUsedMB != 8192 {
		t.Errorf("MemUsedMB = %d, want 8192", result.MemUsedMB)
	}
}

func TestHandleGetProxmoxVMResources_NotFound(t *testing.T) {
	m := newTestModuleWithProxmox(t)

	req := httptest.NewRequest("GET", "/recon/proxmox/vms/nonexistent/resources", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	m.handleGetProxmoxVMResources(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleGetProxmoxVMResources_MissingID(t *testing.T) {
	m := newTestModuleWithProxmox(t)

	req := httptest.NewRequest("GET", "/recon/proxmox/vms//resources", http.NoBody)
	// Don't set PathValue -- simulates missing ID.
	w := httptest.NewRecorder()
	m.handleGetProxmoxVMResources(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleProxmoxSync_MissingFields(t *testing.T) {
	m := newTestModuleWithProxmox(t)

	body := `{"base_url":"https://pve:8006"}`
	req := httptest.NewRequest("POST", "/recon/proxmox/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleProxmoxSync(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestHandleProxmoxSync_InvalidJSON(t *testing.T) {
	m := newTestModuleWithProxmox(t)

	req := httptest.NewRequest("POST", "/recon/proxmox/sync", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleProxmoxSync(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
