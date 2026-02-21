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

func newMockQuerier() *mockQuerier {
	now := time.Now().UTC()
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
	// Simple filter: return all devices that pass basic checks.
	var result []models.Device
	for _, d := range q.allDevices {
		result = append(result, d)
	}
	return result, len(result), nil
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
	if len(routes) != 3 {
		t.Fatalf("Routes() = %d, want 3", len(routes))
	}

	expected := map[string]bool{
		"POST /": false,
		"GET /":  false,
		"DELETE /": false,
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
				if resp.Total != 2 {
					t.Errorf("total = %d, want 2", resp.Total)
				}
				if len(resp.Devices) != 2 {
					t.Errorf("devices count = %d, want 2", len(resp.Devices))
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
				if resp.Total < 1 {
					t.Errorf("total = %d, want >= 1", resp.Total)
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

			req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/", nil)
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
