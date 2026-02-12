package recon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// newTestModule creates a Module wired with in-memory SQLite and mock scanners.
func newTestModule(t *testing.T) *Module {
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

	logger, _ := zap.NewDevelopment()
	reconStore := NewReconStore(db.DB())
	oui := NewOUITable()
	bus := newTestBus(logger)

	pinger := &mockPingScanner{results: []HostResult{}}
	arp := &mockARPReader{table: map[string]string{}}

	m := &Module{
		logger:      logger,
		cfg:         DefaultConfig(),
		store:       reconStore,
		bus:         bus,
		oui:         oui,
		orchestrator: NewScanOrchestrator(reconStore, bus, oui, pinger, arp, logger),
	}
	// Start sets up scanCtx.
	m.scanCtx, m.scanCancel = context.WithCancel(context.Background())
	t.Cleanup(func() { m.scanCancel() })

	return m
}

func TestHandleScan_ValidCIDR(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`{"subnet":"192.168.1.0/24"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusAccepted, w.Body.String())
	}

	var resp models.ScanResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty scan ID")
	}
	if resp.Status != "running" {
		t.Errorf("status = %q, want running", resp.Status)
	}

	// Wait for background scan to finish.
	m.wg.Wait()
}

func TestHandleScan_InvalidCIDR(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`{"subnet":"not-a-cidr"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}

func TestHandleScan_MissingSubnet(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleScan_SubnetTooLarge(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`{"subnet":"10.0.0.0/8"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleScan_InvalidBody(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("POST", "/scan", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	m.handleScan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleListScans_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("GET", "/scans", http.NoBody)
	w := httptest.NewRecorder()
	m.handleListScans(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var scans []models.ScanResult
	if err := json.NewDecoder(w.Body).Decode(&scans); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(scans) != 0 {
		t.Errorf("scan count = %d, want 0", len(scans))
	}
}

func TestHandleListScans_WithData(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()

	// Create some scans.
	for i := 0; i < 3; i++ {
		_ = m.store.CreateScan(ctx, &models.ScanResult{Subnet: "10.0.0.0/24"})
	}

	req := httptest.NewRequest("GET", "/scans?limit=2&offset=0", http.NoBody)
	w := httptest.NewRecorder()
	m.handleListScans(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var scans []models.ScanResult
	_ = json.NewDecoder(w.Body).Decode(&scans)
	if len(scans) != 2 {
		t.Errorf("scan count = %d, want 2 (paginated)", len(scans))
	}
}

func TestHandleGetScan_Found(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()

	scan := &models.ScanResult{ID: "test-scan-1", Subnet: "10.0.0.0/24", Status: "completed"}
	_ = m.store.CreateScan(ctx, scan)

	// Use Go 1.22+ mux for path parameter support.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /scans/{id}", m.handleGetScan)

	req := httptest.NewRequest("GET", "/scans/test-scan-1", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var got models.ScanResult
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.ID != "test-scan-1" {
		t.Errorf("scan ID = %q, want test-scan-1", got.ID)
	}
}

func TestHandleGetScan_NotFound(t *testing.T) {
	m := newTestModule(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /scans/{id}", m.handleGetScan)

	req := httptest.NewRequest("GET", "/scans/nonexistent", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleTopology_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest("GET", "/topology", http.NoBody)
	w := httptest.NewRecorder()
	m.handleTopology(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var graph TopologyGraph
	if err := json.NewDecoder(w.Body).Decode(&graph); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(graph.Nodes) != 0 {
		t.Errorf("nodes = %d, want 0", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Errorf("edges = %d, want 0", len(graph.Edges))
	}
}

func TestHandleTopology_WithData(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()

	d1 := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:00:00:00:00:01",
		Hostname: "router", DeviceType: models.DeviceTypeRouter,
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryARP,
	}
	d2 := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:00:00:00:00:02",
		Hostname: "switch", DeviceType: models.DeviceTypeSwitch,
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryARP,
	}
	_, _ = m.store.UpsertDevice(ctx, d1)
	_, _ = m.store.UpsertDevice(ctx, d2)
	_ = m.store.UpsertTopologyLink(ctx, &TopologyLink{
		SourceDeviceID: d1.ID, TargetDeviceID: d2.ID, LinkType: "arp",
	})

	req := httptest.NewRequest("GET", "/topology", http.NoBody)
	w := httptest.NewRecorder()
	m.handleTopology(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var graph TopologyGraph
	_ = json.NewDecoder(w.Body).Decode(&graph)
	if len(graph.Nodes) != 2 {
		t.Errorf("nodes = %d, want 2", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Errorf("edges = %d, want 1", len(graph.Edges))
	}
}

func TestWriteError_Format(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["detail"] != "test error" {
		t.Errorf("detail = %v, want test error", resp["detail"])
	}
	if resp["status"] != float64(http.StatusBadRequest) {
		t.Errorf("status = %v, want %d", resp["status"], http.StatusBadRequest)
	}
}

// ---------------------------------------------------------------------------
// Device CRUD handler tests
// ---------------------------------------------------------------------------

// deviceMux returns an http.ServeMux with all device routes registered
// so path parameters like {id} are resolved correctly.
func deviceMux(m *Module) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /devices", m.handleListDevices)
	mux.HandleFunc("POST /devices", m.handleCreateDevice)
	mux.HandleFunc("GET /devices/{id}", m.handleGetDevice)
	mux.HandleFunc("PUT /devices/{id}", m.handleUpdateDevice)
	mux.HandleFunc("DELETE /devices/{id}", m.handleDeleteDevice)
	mux.HandleFunc("GET /devices/{id}/history", m.handleDeviceHistory)
	mux.HandleFunc("GET /devices/{id}/scans", m.handleDeviceScans)
	return mux
}

func TestHandleListDevices_Empty(t *testing.T) {
	m := newTestModule(t)
	mux := deviceMux(m)

	req := httptest.NewRequest("GET", "/devices", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp DeviceListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
	if len(resp.Devices) != 0 {
		t.Errorf("devices = %d, want 0", len(resp.Devices))
	}
}

func TestHandleListDevices_WithDevices(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()
	mux := deviceMux(m)

	for i := 0; i < 3; i++ {
		d := &models.Device{
			Hostname:        "dev-" + string(rune('a'+i)),
			IPAddresses:     []string{"10.0.0." + string(rune('1'+i))},
			MACAddress:      "AA:BB:CC:DD:EE:0" + string(rune('1'+i)),
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryICMP,
		}
		_, _ = m.store.UpsertDevice(ctx, d)
	}

	req := httptest.NewRequest("GET", "/devices?limit=2", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp DeviceListResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 3 {
		t.Errorf("total = %d, want 3", resp.Total)
	}
	if len(resp.Devices) != 2 {
		t.Errorf("devices = %d, want 2 (paginated)", len(resp.Devices))
	}
}

func TestHandleListDevices_FilterByStatus(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()
	mux := deviceMux(m)

	online := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:00:00:01",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = m.store.UpsertDevice(ctx, online)

	offline := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:00:00:02",
		Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = m.store.UpsertDevice(ctx, offline)
	_, _ = m.store.db.ExecContext(ctx, "UPDATE recon_devices SET status = 'offline' WHERE id = ?", offline.ID)

	req := httptest.NewRequest("GET", "/devices?status=online", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp DeviceListResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
}

func TestHandleListDevices_FilterByType(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()
	mux := deviceMux(m)

	server := &models.Device{
		IPAddresses: []string{"10.0.0.1"}, MACAddress: "AA:BB:CC:00:00:01",
		DeviceType: models.DeviceTypeServer, Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	router := &models.Device{
		IPAddresses: []string{"10.0.0.2"}, MACAddress: "AA:BB:CC:00:00:02",
		DeviceType: models.DeviceTypeRouter, Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = m.store.UpsertDevice(ctx, server)
	_, _ = m.store.UpsertDevice(ctx, router)

	req := httptest.NewRequest("GET", "/devices?type=router", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp DeviceListResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
	if len(resp.Devices) != 1 {
		t.Fatalf("devices = %d, want 1", len(resp.Devices))
	}
	if resp.Devices[0].DeviceType != models.DeviceTypeRouter {
		t.Errorf("DeviceType = %q, want router", resp.Devices[0].DeviceType)
	}
}

func TestHandleGetDevice_Found(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()
	mux := deviceMux(m)

	d := &models.Device{
		Hostname: "get-me", IPAddresses: []string{"10.0.0.1"},
		MACAddress: "AA:BB:CC:DD:EE:01", Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = m.store.UpsertDevice(ctx, d)

	req := httptest.NewRequest("GET", "/devices/"+d.ID, http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var got models.Device
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.Hostname != "get-me" {
		t.Errorf("Hostname = %q, want get-me", got.Hostname)
	}
}

func TestHandleGetDevice_NotFound(t *testing.T) {
	m := newTestModule(t)
	mux := deviceMux(m)

	req := httptest.NewRequest("GET", "/devices/nonexistent", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleUpdateDevice_Success(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()
	mux := deviceMux(m)

	d := &models.Device{
		Hostname: "update-me", IPAddresses: []string{"10.0.0.1"},
		MACAddress: "AA:BB:CC:DD:EE:01", Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = m.store.UpsertDevice(ctx, d)

	body := `{"notes":"new notes","tags":["web","prod"]}`
	req := httptest.NewRequest("PUT", "/devices/"+d.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var got models.Device
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.Notes != "new notes" {
		t.Errorf("Notes = %q, want new notes", got.Notes)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(got.Tags))
	}
}

func TestHandleUpdateDevice_NotFound(t *testing.T) {
	m := newTestModule(t)
	mux := deviceMux(m)

	body := `{"notes":"test"}`
	req := httptest.NewRequest("PUT", "/devices/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleUpdateDevice_InvalidJSON(t *testing.T) {
	m := newTestModule(t)
	mux := deviceMux(m)

	req := httptest.NewRequest("PUT", "/devices/some-id", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeleteDevice_Success(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()
	mux := deviceMux(m)

	d := &models.Device{
		Hostname: "delete-me", IPAddresses: []string{"10.0.0.1"},
		MACAddress: "AA:BB:CC:DD:EE:01", Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = m.store.UpsertDevice(ctx, d)

	req := httptest.NewRequest("DELETE", "/devices/"+d.ID, http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify it's gone.
	req2 := httptest.NewRequest("GET", "/devices/"+d.ID, http.NoBody)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", w2.Code, http.StatusNotFound)
	}
}

func TestHandleDeleteDevice_NotFound(t *testing.T) {
	m := newTestModule(t)
	mux := deviceMux(m)

	req := httptest.NewRequest("DELETE", "/devices/nonexistent", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleCreateDevice_Success(t *testing.T) {
	m := newTestModule(t)
	mux := deviceMux(m)

	body := `{"hostname":"new-device","ip_addresses":["10.0.0.99"],"device_type":"server"}`
	req := httptest.NewRequest("POST", "/devices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var got models.Device
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.ID == "" {
		t.Error("expected non-empty ID")
	}
	if got.Hostname != "new-device" {
		t.Errorf("Hostname = %q, want new-device", got.Hostname)
	}
	if got.DiscoveryMethod != models.DiscoveryManual {
		t.Errorf("DiscoveryMethod = %q, want manual", got.DiscoveryMethod)
	}
}

func TestHandleCreateDevice_InvalidJSON(t *testing.T) {
	m := newTestModule(t)
	mux := deviceMux(m)

	req := httptest.NewRequest("POST", "/devices", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateDevice_MissingHostname(t *testing.T) {
	m := newTestModule(t)
	mux := deviceMux(m)

	body := `{"ip_addresses":["10.0.0.1"]}`
	req := httptest.NewRequest("POST", "/devices", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceHistory_Empty(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()
	mux := deviceMux(m)

	d := &models.Device{
		Hostname: "hist-test", IPAddresses: []string{"10.0.0.1"},
		MACAddress: "AA:BB:CC:DD:EE:01", Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = m.store.UpsertDevice(ctx, d)

	req := httptest.NewRequest("GET", "/devices/"+d.ID+"/history", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var events []DeviceStatusEvent
	_ = json.NewDecoder(w.Body).Decode(&events)
	if len(events) != 0 {
		t.Errorf("events = %d, want 0", len(events))
	}
}

func TestHandleDeviceHistory_WithData(t *testing.T) {
	m := newTestModule(t)
	ctx := context.Background()
	mux := deviceMux(m)

	d := &models.Device{
		Hostname: "hist-test", IPAddresses: []string{"10.0.0.1"},
		MACAddress: "AA:BB:CC:DD:EE:01", Status: models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryICMP,
	}
	_, _ = m.store.UpsertDevice(ctx, d)
	_ = m.store.MarkDeviceOffline(ctx, d.ID) // Creates a history entry.

	req := httptest.NewRequest("GET", "/devices/"+d.ID+"/history", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var events []DeviceStatusEvent
	_ = json.NewDecoder(w.Body).Decode(&events)
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Status != "offline" {
		t.Errorf("Status = %q, want offline", events[0].Status)
	}
	if events[0].DeviceID != d.ID {
		t.Errorf("DeviceID = %q, want %q", events[0].DeviceID, d.ID)
	}
	if events[0].Timestamp == "" {
		t.Error("expected non-empty Timestamp")
	}
}
