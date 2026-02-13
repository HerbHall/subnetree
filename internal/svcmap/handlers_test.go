package svcmap

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	_ "modernc.org/sqlite"
	"go.uber.org/zap"
)

// testHarness creates an in-memory store, handler, and a test mux with routes registered.
func testHarness(t *testing.T) (*Store, *http.ServeMux) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	handler := NewHandler(store, &stubHWSource{}, zap.NewNop())
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return store, mux
}

// stubHWSource is a no-op HardwareSource for handler tests.
type stubHWSource struct{}

func (s *stubHWSource) GetHardwareProfile(_ context.Context, _ string) (*HardwareInfo, error) {
	return &HardwareInfo{
		TotalMemoryBytes: 8 * 1024 * 1024 * 1024, // 8 GB
		TotalDiskBytes:   500 * 1024 * 1024 * 1024,
		CPUCores:         4,
	}, nil
}

func seedService(t *testing.T, store *Store, id, name, deviceID string, status models.ServiceStatus) {
	t.Helper()
	now := time.Now().UTC()
	svc := &models.Service{
		ID:           id,
		Name:         name,
		DisplayName:  name,
		ServiceType:  models.ServiceTypeSystemdService,
		DeviceID:     deviceID,
		Status:       status,
		DesiredState: models.DesiredStateShouldRun,
		Ports:        []string{},
		CPUPercent:   5.0,
		MemoryBytes:  1024 * 1024,
		FirstSeen:    now,
		LastSeen:     now,
	}
	if err := store.UpsertService(context.Background(), svc); err != nil {
		t.Fatalf("seed service %s: %v", id, err)
	}
}

func TestHandleListServices_Empty(t *testing.T) {
	_, mux := testHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/svcmap/services", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var services []models.Service
	if err := json.NewDecoder(rec.Body).Decode(&services); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(services) != 0 {
		t.Errorf("expected empty list, got %d services", len(services))
	}
}

func TestHandleListServices_WithData(t *testing.T) {
	store, mux := testHarness(t)
	seedService(t, store, "svc-1", "nginx", "dev-1", models.ServiceStatusRunning)
	seedService(t, store, "svc-2", "redis", "dev-1", models.ServiceStatusStopped)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/svcmap/services", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var services []models.Service
	if err := json.NewDecoder(rec.Body).Decode(&services); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}
}

func TestHandleListServices_FilterByStatus(t *testing.T) {
	store, mux := testHarness(t)
	seedService(t, store, "svc-1", "nginx", "dev-1", models.ServiceStatusRunning)
	seedService(t, store, "svc-2", "redis", "dev-1", models.ServiceStatusStopped)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/svcmap/services?status=running", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var services []models.Service
	if err := json.NewDecoder(rec.Body).Decode(&services); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("expected 1 running service, got %d", len(services))
	}
	if len(services) > 0 && services[0].Name != "nginx" {
		t.Errorf("expected nginx, got %s", services[0].Name)
	}
}

func TestHandleGetService_Found(t *testing.T) {
	store, mux := testHarness(t)
	seedService(t, store, "svc-1", "nginx", "dev-1", models.ServiceStatusRunning)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/svcmap/services/svc-1", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var svc models.Service
	if err := json.NewDecoder(rec.Body).Decode(&svc); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if svc.ID != "svc-1" {
		t.Errorf("expected svc-1, got %s", svc.ID)
	}
}

func TestHandleGetService_NotFound(t *testing.T) {
	_, mux := testHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/svcmap/services/nonexistent", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleUpdateDesiredState_Valid(t *testing.T) {
	store, mux := testHarness(t)
	seedService(t, store, "svc-1", "nginx", "dev-1", models.ServiceStatusRunning)

	body := `{"desired_state":"should-stop"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/svcmap/services/svc-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var svc models.Service
	if err := json.NewDecoder(rec.Body).Decode(&svc); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if svc.DesiredState != models.DesiredStateShouldStop {
		t.Errorf("expected should-stop, got %s", svc.DesiredState)
	}
}

func TestHandleUpdateDesiredState_InvalidBody(t *testing.T) {
	store, mux := testHarness(t)
	seedService(t, store, "svc-1", "nginx", "dev-1", models.ServiceStatusRunning)

	body := `{"desired_state":"invalid-value"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/svcmap/services/svc-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateDesiredState_NotFound(t *testing.T) {
	_, mux := testHarness(t)

	body := `{"desired_state":"should-run"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/svcmap/services/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDeviceServices(t *testing.T) {
	store, mux := testHarness(t)
	seedService(t, store, "svc-1", "nginx", "dev-1", models.ServiceStatusRunning)
	seedService(t, store, "svc-2", "redis", "dev-2", models.ServiceStatusRunning)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/svcmap/devices/dev-1/services", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var services []models.Service
	if err := json.NewDecoder(rec.Body).Decode(&services); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(services) != 1 {
		t.Errorf("expected 1 service for dev-1, got %d", len(services))
	}
	if len(services) > 0 && services[0].DeviceID != "dev-1" {
		t.Errorf("expected device dev-1, got %s", services[0].DeviceID)
	}
}

func TestHandleDeviceUtilization(t *testing.T) {
	store, mux := testHarness(t)
	seedService(t, store, "svc-1", "nginx", "dev-1", models.ServiceStatusRunning)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/svcmap/devices/dev-1/utilization?agent_id=agent-1", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var summary models.UtilizationSummary
	if err := json.NewDecoder(rec.Body).Decode(&summary); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if summary.DeviceID != "dev-1" {
		t.Errorf("expected device dev-1, got %s", summary.DeviceID)
	}
	if summary.ServiceCount != 1 {
		t.Errorf("expected 1 service, got %d", summary.ServiceCount)
	}
}

func TestHandleFleetSummary_Empty(t *testing.T) {
	_, mux := testHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/svcmap/utilization/fleet", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var fleet models.FleetSummary
	if err := json.NewDecoder(rec.Body).Decode(&fleet); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if fleet.TotalDevices != 0 {
		t.Errorf("expected 0 devices, got %d", fleet.TotalDevices)
	}
}

func TestHandleFleetSummary_WithData(t *testing.T) {
	store, mux := testHarness(t)
	seedService(t, store, "svc-1", "nginx", "dev-1", models.ServiceStatusRunning)
	seedService(t, store, "svc-2", "redis", "dev-1", models.ServiceStatusRunning)
	seedService(t, store, "svc-3", "postgres", "dev-2", models.ServiceStatusRunning)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/svcmap/utilization/fleet", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var fleet models.FleetSummary
	if err := json.NewDecoder(rec.Body).Decode(&fleet); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if fleet.TotalDevices != 2 {
		t.Errorf("expected 2 devices, got %d", fleet.TotalDevices)
	}
	if fleet.TotalServices != 3 {
		t.Errorf("expected 3 services, got %d", fleet.TotalServices)
	}
}
