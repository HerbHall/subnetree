package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/HerbHall/subnetree/internal/testutil"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/plugin/plugintest"
	"go.uber.org/zap"
)

func TestPluginContract(t *testing.T) {
	plugintest.TestPluginContract(t, func() plugin.Plugin { return New() })
}

// mockQuerier implements DeviceQuerier for testing.
type mockQuerier struct {
	devices         map[string]*models.Device
	hardware        map[string]*models.DeviceHardware
	summary         *models.HardwareSummary
	allDevices      []models.Device
	queryDevicesErr error
}

// mockServiceQuerier implements ServiceQuerier for testing.
type mockServiceQuerier struct {
	services []models.Service
}

func newMockServiceQuerier() *mockServiceQuerier {
	now := time.Now().UTC()
	return &mockServiceQuerier{
		services: []models.Service{
			{
				ID:          "svc-001",
				Name:        "nginx",
				DisplayName: "NGINX Web Server",
				ServiceType: models.ServiceTypeDockerContainer,
				DeviceID:    "dev-001",
				Status:      models.ServiceStatusRunning,
				FirstSeen:   now,
				LastSeen:    now,
			},
			{
				ID:          "svc-002",
				Name:        "postgres",
				DisplayName: "PostgreSQL",
				ServiceType: models.ServiceTypeDockerContainer,
				DeviceID:    "dev-001",
				Status:      models.ServiceStatusRunning,
				FirstSeen:   now,
				LastSeen:    now,
			},
			{
				ID:          "svc-003",
				Name:        "sshd",
				DisplayName: "SSH Daemon",
				ServiceType: models.ServiceTypeSystemdService,
				DeviceID:    "dev-002",
				Status:      models.ServiceStatusRunning,
				FirstSeen:   now,
				LastSeen:    now,
			},
		},
	}
}

func (q *mockServiceQuerier) ListServicesFiltered(_ context.Context, deviceID, serviceType, status string) ([]models.Service, error) {
	result := make([]models.Service, 0, len(q.services))
	for i := range q.services {
		if deviceID != "" && q.services[i].DeviceID != deviceID {
			continue
		}
		if serviceType != "" && string(q.services[i].ServiceType) != serviceType {
			continue
		}
		if status != "" && string(q.services[i].Status) != status {
			continue
		}
		result = append(result, q.services[i])
	}
	return result, nil
}

func newMockQuerier() *mockQuerier {
	now := time.Now().UTC()
	staleTime := now.Add(-48 * time.Hour)
	devices := map[string]*models.Device{
		"dev-001": {
			ID:          "dev-001",
			Hostname:    "web-server-01",
			IPAddresses: []string{"192.168.1.10"},
			DeviceType:  models.DeviceTypeServer,
			Status:      models.DeviceStatusOnline,
			FirstSeen:   now,
			LastSeen:    now,
		},
		"dev-002": {
			ID:          "dev-002",
			Hostname:    "nas-01",
			IPAddresses: []string{"192.168.1.20"},
			DeviceType:  models.DeviceTypeNAS,
			Status:      models.DeviceStatusOnline,
			FirstSeen:   now,
			LastSeen:    now,
		},
		"dev-003": {
			ID:          "dev-003",
			Hostname:    "stale-printer",
			IPAddresses: []string{"192.168.1.30"},
			DeviceType:  models.DeviceTypePrinter,
			Status:      models.DeviceStatusOnline,
			FirstSeen:   staleTime,
			LastSeen:    staleTime,
		},
	}

	allDevices := make([]models.Device, 0, len(devices))
	for _, d := range devices {
		allDevices = append(allDevices, *d)
	}

	return &mockQuerier{
		devices: devices,
		hardware: map[string]*models.DeviceHardware{
			"dev-001": {
				DeviceID: "dev-001",
				CPUModel: "Intel Core i9-10900K",
				CPUCores: 10,
				RAMTotalMB: 32768,
				OSName:   "Ubuntu 24.04",
			},
		},
		summary: &models.HardwareSummary{
			TotalWithHardware: 1,
			TotalRAMMB:        32768,
			TotalStorageGB:    4000,
			ByOS:              map[string]int{"Ubuntu 24.04": 1},
			ByCPUModel:        map[string]int{"Intel Core i9-10900K": 1},
		},
		allDevices: allDevices,
	}
}

func (q *mockQuerier) GetDevice(_ context.Context, id string) (*models.Device, error) {
	d, ok := q.devices[id]
	if !ok {
		return nil, nil
	}
	return d, nil
}

func (q *mockQuerier) ListDevices(_ context.Context, limit, offset int) ([]models.Device, int, error) {
	total := len(q.allDevices)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return q.allDevices[offset:end], total, nil
}

func (q *mockQuerier) GetDeviceHardware(_ context.Context, deviceID string) (*models.DeviceHardware, error) {
	hw, ok := q.hardware[deviceID]
	if !ok {
		return nil, nil
	}
	return hw, nil
}

func (q *mockQuerier) GetHardwareSummary(_ context.Context) (*models.HardwareSummary, error) {
	return q.summary, nil
}

func (q *mockQuerier) QueryDevicesByHardware(_ context.Context, query models.HardwareQuery) ([]models.Device, int, error) {
	if q.queryDevicesErr != nil {
		return nil, 0, q.queryDevicesErr
	}
	result := append([]models.Device{}, q.allDevices...)
	return result, len(result), nil
}

func (q *mockQuerier) FindStaleDevices(_ context.Context, threshold time.Time) ([]models.Device, error) {
	var stale []models.Device
	for i := range q.allDevices {
		if q.allDevices[i].Status == models.DeviceStatusOnline && q.allDevices[i].LastSeen.Before(threshold) {
			stale = append(stale, q.allDevices[i])
		}
	}
	return stale, nil
}

func newTestModule(t *testing.T) *Module {
	t.Helper()
	m := New()
	logger, _ := zap.NewDevelopment()
	bus := testutil.NewMockBus()

	err := m.Init(context.Background(), plugin.Dependencies{
		Logger: logger.Named("mcp"),
		Bus:    bus,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	m.SetQuerier(newMockQuerier())
	m.SetServiceQuerier(newMockServiceQuerier())

	err = m.Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		_ = m.Stop(context.Background())
	})

	return m
}

func TestInfo(t *testing.T) {
	m := New()
	info := m.Info()

	if info.Name != "mcp" {
		t.Errorf("Name = %q, want %q", info.Name, "mcp")
	}
	if info.Version != "0.1.0" {
		t.Errorf("Version = %q, want %q", info.Version, "0.1.0")
	}
	if len(info.Dependencies) != 1 || info.Dependencies[0] != "recon" {
		t.Errorf("Dependencies = %v, want [recon]", info.Dependencies)
	}
}

func TestRoutes(t *testing.T) {
	m := New()
	routes := m.Routes()
	if len(routes) != 4 {
		t.Fatalf("Routes() = %d, want 4", len(routes))
	}

	expected := map[string]bool{
		"POST /":      false,
		"GET /":       false,
		"DELETE /":    false,
		"GET /audit":  false,
	}
	for _, r := range routes {
		key := r.Method + " " + r.Path
		if _, ok := expected[key]; !ok {
			t.Errorf("unexpected route: %s", key)
		}
		expected[key] = true
	}
	for key, found := range expected {
		if !found {
			t.Errorf("missing route: %s", key)
		}
	}
}

func TestToolHandlers(t *testing.T) {
	m := newTestModule(t)

	tests := []struct {
		name       string
		handler    func() (*CallToolResultCheck, error)
		wantErr    bool
		checkJSON  func(t *testing.T, data string)
	}{
		{
			name: "get_device_found",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetDevice(context.Background(), nil, getDeviceInput{DeviceID: "dev-001"})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var dev models.Device
				if err := json.Unmarshal([]byte(data), &dev); err != nil {
					t.Fatalf("unmarshal device: %v", err)
				}
				if dev.ID != "dev-001" {
					t.Errorf("device ID = %q, want %q", dev.ID, "dev-001")
				}
				if dev.Hostname != "web-server-01" {
					t.Errorf("hostname = %q, want %q", dev.Hostname, "web-server-01")
				}
			},
		},
		{
			name: "get_device_not_found",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetDevice(context.Background(), nil, getDeviceInput{DeviceID: "nonexistent"})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				if data == "" {
					t.Error("expected non-empty response for missing device")
				}
			},
		},
		{
			name: "list_devices",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleListDevices(context.Background(), nil, listDevicesInput{Limit: 10})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var resp struct {
					Devices []models.Device `json:"devices"`
					Total   int             `json:"total"`
				}
				if err := json.Unmarshal([]byte(data), &resp); err != nil {
					t.Fatalf("unmarshal list: %v", err)
				}
				if resp.Total != 3 {
					t.Errorf("total = %d, want 3", resp.Total)
				}
				if len(resp.Devices) != 3 {
					t.Errorf("devices count = %d, want 3", len(resp.Devices))
				}
			},
		},
		{
			name: "get_hardware_profile",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetHardwareProfile(context.Background(), nil, getHardwareProfileInput{DeviceID: "dev-001"})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var hw models.DeviceHardware
				if err := json.Unmarshal([]byte(data), &hw); err != nil {
					t.Fatalf("unmarshal hardware: %v", err)
				}
				if hw.CPUModel != "Intel Core i9-10900K" {
					t.Errorf("cpu_model = %q, want %q", hw.CPUModel, "Intel Core i9-10900K")
				}
			},
		},
		{
			name: "get_hardware_profile_not_found",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetHardwareProfile(context.Background(), nil, getHardwareProfileInput{DeviceID: "nonexistent"})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				if data == "" {
					t.Error("expected non-empty response for missing hardware")
				}
			},
		},
		{
			name: "get_fleet_summary",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetFleetSummary(context.Background(), nil, struct{}{})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var summary models.HardwareSummary
				if err := json.Unmarshal([]byte(data), &summary); err != nil {
					t.Fatalf("unmarshal summary: %v", err)
				}
				if summary.TotalWithHardware != 1 {
					t.Errorf("total_with_hardware = %d, want 1", summary.TotalWithHardware)
				}
			},
		},
		{
			name: "query_devices",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleQueryDevices(context.Background(), nil, queryDevicesInput{MinRAMMB: 8192})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var resp struct {
					Devices []models.Device `json:"devices"`
					Total   int             `json:"total"`
				}
				if err := json.Unmarshal([]byte(data), &resp); err != nil {
					t.Fatalf("unmarshal query result: %v", err)
				}
				// Mock returns all devices regardless of query filter.
				if resp.Total < 1 {
					t.Errorf("total = %d, want >= 1", resp.Total)
				}
			},
		},
		{
			name: "get_stale_devices_default",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetStaleDevices(context.Background(), nil, getStaleDevicesInput{})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var resp struct {
					Devices         []models.Device `json:"devices"`
					Count           int             `json:"count"`
					StaleAfterHours int             `json:"stale_after_hours"`
				}
				if err := json.Unmarshal([]byte(data), &resp); err != nil {
					t.Fatalf("unmarshal stale result: %v", err)
				}
				if resp.StaleAfterHours != 24 {
					t.Errorf("stale_after_hours = %d, want 24", resp.StaleAfterHours)
				}
				if resp.Count != 1 {
					t.Errorf("count = %d, want 1 (stale-printer)", resp.Count)
				}
				if len(resp.Devices) > 0 && resp.Devices[0].Hostname != "stale-printer" {
					t.Errorf("hostname = %q, want %q", resp.Devices[0].Hostname, "stale-printer")
				}
			},
		},
		{
			name: "get_stale_devices_custom_hours",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetStaleDevices(context.Background(), nil, getStaleDevicesInput{StaleAfterHours: 100})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var resp struct {
					Count           int `json:"count"`
					StaleAfterHours int `json:"stale_after_hours"`
				}
				if err := json.Unmarshal([]byte(data), &resp); err != nil {
					t.Fatalf("unmarshal stale result: %v", err)
				}
				if resp.StaleAfterHours != 100 {
					t.Errorf("stale_after_hours = %d, want 100", resp.StaleAfterHours)
				}
				// With 100h window, stale-printer (48h old) is NOT stale.
				if resp.Count != 0 {
					t.Errorf("count = %d, want 0 (100h window should include 48h-old device)", resp.Count)
				}
			},
		},
		{
			name: "get_service_inventory_all",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetServiceInventory(context.Background(), nil, getServiceInventoryInput{})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var resp struct {
					ServicesByDevice map[string][]models.Service `json:"services_by_device"`
					TotalServices    int                         `json:"total_services"`
				}
				if err := json.Unmarshal([]byte(data), &resp); err != nil {
					t.Fatalf("unmarshal services result: %v", err)
				}
				if resp.TotalServices != 3 {
					t.Errorf("total_services = %d, want 3", resp.TotalServices)
				}
				if len(resp.ServicesByDevice) != 2 {
					t.Errorf("device groups = %d, want 2", len(resp.ServicesByDevice))
				}
			},
		},
		{
			name: "get_service_inventory_filtered_by_device",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetServiceInventory(context.Background(), nil, getServiceInventoryInput{DeviceID: "dev-001"})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var resp struct {
					TotalServices int `json:"total_services"`
				}
				if err := json.Unmarshal([]byte(data), &resp); err != nil {
					t.Fatalf("unmarshal services result: %v", err)
				}
				if resp.TotalServices != 2 {
					t.Errorf("total_services = %d, want 2 (nginx + postgres on dev-001)", resp.TotalServices)
				}
			},
		},
		{
			name: "get_service_inventory_filtered_by_type",
			handler: func() (*CallToolResultCheck, error) {
				result, _, err := m.handleGetServiceInventory(context.Background(), nil, getServiceInventoryInput{ServiceType: "systemd-service"})
				return toCheck(result), err
			},
			checkJSON: func(t *testing.T, data string) {
				var resp struct {
					TotalServices int `json:"total_services"`
				}
				if err := json.Unmarshal([]byte(data), &resp); err != nil {
					t.Fatalf("unmarshal services result: %v", err)
				}
				if resp.TotalServices != 1 {
					t.Errorf("total_services = %d, want 1 (sshd)", resp.TotalServices)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.handler()
			if err != nil {
				if !tc.wantErr {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.wantErr {
				t.Fatal("expected error, got nil")
			}
			if result == nil || result.text == "" {
				t.Fatal("expected non-nil result with text content")
			}
			tc.checkJSON(t, result.text)
		})
	}
}

func TestNoQuerier(t *testing.T) {
	m := New()
	logger, _ := zap.NewDevelopment()
	bus := testutil.NewMockBus()

	err := m.Init(context.Background(), plugin.Dependencies{
		Logger: logger.Named("mcp"),
		Bus:    bus,
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	// Do NOT call SetQuerier -- querier is nil.

	err = m.Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = m.Stop(context.Background()) }()

	result, _, err := m.handleGetDevice(context.Background(), nil, getDeviceInput{DeviceID: "dev-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	check := toCheck(result)
	if check.text == "" {
		t.Fatal("expected non-empty response when querier is nil")
	}

	// Also test service inventory with nil serviceQuerier.
	svcResult, _, svcErr := m.handleGetServiceInventory(context.Background(), nil, getServiceInventoryInput{})
	if svcErr != nil {
		t.Fatalf("unexpected error: %v", svcErr)
	}
	svcCheck := toCheck(svcResult)
	if svcCheck.text == "" {
		t.Fatal("expected non-empty response when serviceQuerier is nil")
	}
}

func TestAPIKeyMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		authHeader string
		wantStatus int
	}{
		{
			name:       "no_key_configured_allows_all",
			apiKey:     "",
			authHeader: "",
			wantStatus: http.StatusServiceUnavailable, // server not started, but passes auth
		},
		{
			name:       "valid_key",
			apiKey:     "test-secret-key",
			authHeader: "Bearer test-secret-key",
			wantStatus: http.StatusServiceUnavailable, // passes auth, server not started
		},
		{
			name:       "invalid_key",
			apiKey:     "test-secret-key",
			authHeader: "Bearer wrong-key",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing_key_when_required",
			apiKey:     "test-secret-key",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed_auth_header",
			apiKey:     "test-secret-key",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := New()
			logger, _ := zap.NewDevelopment()

			err := m.Init(context.Background(), plugin.Dependencies{
				Logger: logger.Named("mcp"),
			})
			if err != nil {
				t.Fatalf("Init: %v", err)
			}
			m.apiKey = tc.apiKey

			req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/", http.NoBody)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rr := httptest.NewRecorder()

			m.handleMCP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

func TestPublishToolCall(t *testing.T) {
	bus := testutil.NewMockBus()
	m := &Module{bus: bus}

	m.publishToolCall("get_device", map[string]string{"device_id": "dev-001"})

	events := bus.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Topic != "mcp.tool.called" {
		t.Errorf("topic = %q, want %q", events[0].Topic, "mcp.tool.called")
	}
	if events[0].Source != "mcp" {
		t.Errorf("source = %q, want %q", events[0].Source, "mcp")
	}

	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any payload")
	}
	if payload["tool"] != "get_device" {
		t.Errorf("tool = %v, want %q", payload["tool"], "get_device")
	}
}

// newAuditStoreForTest creates an AuditStore backed by an in-memory SQLite
// database with the MCP migrations already applied.
func newAuditStoreForTest(t *testing.T) *AuditStore {
	t.Helper()
	pluginStore := testutil.NewStore(t)
	if err := pluginStore.Migrate(context.Background(), "mcp", migrations()); err != nil {
		t.Fatalf("migrate audit store: %v", err)
	}
	return NewAuditStore(pluginStore.DB())
}

func TestAuditStore_InsertAndList(t *testing.T) {
	s := newAuditStoreForTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	entries := []AuditEntry{
		{Timestamp: now.Add(-2 * time.Second), ToolName: "get_device", InputJSON: `{"device_id":"d1"}`, UserID: "http", DurationMs: 5, Success: true},
		{Timestamp: now.Add(-1 * time.Second), ToolName: "list_devices", InputJSON: `{"limit":10}`, UserID: "http", DurationMs: 8, Success: true},
		{Timestamp: now, ToolName: "get_device", InputJSON: `{"device_id":"d2"}`, UserID: "stdio", DurationMs: 3, Success: false, ErrorMessage: "not found"},
	}

	for i := range entries {
		if err := s.Insert(ctx, entries[i]); err != nil {
			t.Fatalf("Insert entry %d: %v", i, err)
		}
	}

	got, total, err := s.List(ctx, "", 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(got) != 3 {
		t.Errorf("len(entries) = %d, want 3", len(got))
	}

	// Entries are returned newest-first (ORDER BY timestamp DESC).
	if got[0].ToolName != "get_device" || got[0].UserID != "stdio" {
		t.Errorf("first entry = {%s %s}, want {get_device stdio}", got[0].ToolName, got[0].UserID)
	}
	if got[0].Success {
		t.Error("first entry success = true, want false")
	}
	if got[0].ErrorMessage != "not found" {
		t.Errorf("error_message = %q, want %q", got[0].ErrorMessage, "not found")
	}
}

func TestAuditStore_FilterByTool(t *testing.T) {
	s := newAuditStoreForTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	for i, toolName := range []string{"get_device", "list_devices", "get_device", "get_fleet_summary"} {
		if err := s.Insert(ctx, AuditEntry{
			Timestamp: now.Add(time.Duration(i) * time.Second),
			ToolName:  toolName,
			InputJSON: "{}",
			UserID:    "http",
			Success:   true,
		}); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	got, total, err := s.List(ctx, "get_device", 50, 0)
	if err != nil {
		t.Fatalf("List with filter: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(got) != 2 {
		t.Errorf("len(entries) = %d, want 2", len(got))
	}
	for _, e := range got {
		if e.ToolName != "get_device" {
			t.Errorf("unexpected tool_name %q in filtered results", e.ToolName)
		}
	}

	// Unfiltered should return all 4.
	_, allTotal, err := s.List(ctx, "", 50, 0)
	if err != nil {
		t.Fatalf("List unfiltered: %v", err)
	}
	if allTotal != 4 {
		t.Errorf("unfiltered total = %d, want 4", allTotal)
	}
}

func TestAuditStore_Pagination(t *testing.T) {
	s := newAuditStoreForTest(t)
	ctx := context.Background()

	now := time.Now().UTC()
	for i := 0; i < 10; i++ {
		if err := s.Insert(ctx, AuditEntry{
			Timestamp: now.Add(time.Duration(i) * time.Second),
			ToolName:  "list_devices",
			InputJSON: "{}",
			UserID:    "http",
			Success:   true,
		}); err != nil {
			t.Fatalf("Insert %d: %v", i, err)
		}
	}

	// First page: limit=3, offset=0.
	page1, total, err := s.List(ctx, "", 3, 0)
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if total != 10 {
		t.Errorf("total = %d, want 10", total)
	}
	if len(page1) != 3 {
		t.Errorf("page1 len = %d, want 3", len(page1))
	}

	// Second page: limit=3, offset=3.
	page2, _, err := s.List(ctx, "", 3, 3)
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(page2) != 3 {
		t.Errorf("page2 len = %d, want 3", len(page2))
	}

	// Pages should not overlap (different IDs since ordering is newest-first).
	if page1[0].ID == page2[0].ID {
		t.Error("page1 and page2 share first entry ID -- pagination is broken")
	}

	// Last page: limit=3, offset=9 => only 1 result.
	last, _, err := s.List(ctx, "", 3, 9)
	if err != nil {
		t.Fatalf("List last page: %v", err)
	}
	if len(last) != 1 {
		t.Errorf("last page len = %d, want 1", len(last))
	}
}

func TestAuditListHandler_NoStore(t *testing.T) {
	m := New()
	logger, _ := zap.NewDevelopment()
	_ = m.Init(context.Background(), plugin.Dependencies{Logger: logger.Named("mcp")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/audit", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleAuditList(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestAuditListHandler_WithStore(t *testing.T) {
	m := New()
	logger, _ := zap.NewDevelopment()
	pluginStore := testutil.NewStore(t)
	_ = m.Init(context.Background(), plugin.Dependencies{
		Logger: logger.Named("mcp"),
		Store:  pluginStore,
	})

	// Insert a couple of entries directly.
	now := time.Now().UTC()
	_ = m.auditStore.Insert(context.Background(), AuditEntry{
		Timestamp: now,
		ToolName:  "get_device",
		InputJSON: `{"device_id":"x"}`,
		UserID:    "http",
		Success:   true,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/audit?limit=10&offset=0", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleAuditList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["total"] == nil {
		t.Error("response missing 'total' field")
	}
	if resp["entries"] == nil {
		t.Error("response missing 'entries' field")
	}
}

// CallToolResultCheck is a helper to extract text from CallToolResult for testing.
type CallToolResultCheck struct {
	text    string
	isError bool
}

func toCheck(result *sdkmcp.CallToolResult) *CallToolResultCheck {
	if result == nil {
		return nil
	}
	check := &CallToolResultCheck{isError: result.IsError}
	for _, c := range result.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			check.text = tc.Text
			break
		}
	}
	return check
}
