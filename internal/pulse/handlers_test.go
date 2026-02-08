package pulse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// -- handleListChecks tests --

func TestHandleListChecks_Empty(t *testing.T) {
	m, _ := newTestModule(t)
	req := httptest.NewRequest(http.MethodGet, "/checks", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListChecks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var checks []Check
	if err := json.NewDecoder(w.Body).Decode(&checks); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(checks) != 0 {
		t.Errorf("len(checks) = %d, want 0", len(checks))
	}
}

func TestHandleListChecks_WithData(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/checks", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListChecks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var checks []Check
	if err := json.NewDecoder(w.Body).Decode(&checks); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("len(checks) = %d, want 1", len(checks))
	}
	if checks[0].ID != "check-1" {
		t.Errorf("checks[0].ID = %q, want %q", checks[0].ID, "check-1")
	}
}

func TestHandleListChecks_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/checks", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListChecks(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleDeviceChecks tests --

func TestHandleDeviceChecks_Found(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/checks/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceChecks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got Check
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "check-1" {
		t.Errorf("check.ID = %q, want %q", got.ID, "check-1")
	}
	if got.DeviceID != "dev-1" {
		t.Errorf("check.DeviceID = %q, want %q", got.DeviceID, "dev-1")
	}
}

func TestHandleDeviceChecks_NotFound(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/checks/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceChecks(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleDeviceChecks_EmptyDeviceID(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/checks/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleDeviceChecks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceChecks_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/checks/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceChecks(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleDeviceResults tests --

func TestHandleDeviceResults_Empty(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/results/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var results []CheckResult
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestHandleDeviceResults_WithData(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	// Insert check first (foreign key constraint).
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	result := &CheckResult{
		CheckID:      "check-1",
		DeviceID:     "dev-1",
		Success:      true,
		LatencyMs:    12.5,
		PacketLoss:   0.0,
		ErrorMessage: "",
		CheckedAt:    now,
	}
	if err := m.store.InsertResult(context.Background(), result); err != nil {
		t.Fatalf("insert result: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/results/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var results []CheckResult
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].DeviceID != "dev-1" {
		t.Errorf("results[0].DeviceID = %q, want %q", results[0].DeviceID, "dev-1")
	}
	if results[0].Success != true {
		t.Errorf("results[0].Success = %v, want true", results[0].Success)
	}
}

func TestHandleDeviceResults_WithLimit(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	// Insert check first (foreign key constraint).
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	// Insert 10 results.
	for i := 0; i < 10; i++ {
		result := &CheckResult{
			CheckID:    "check-1",
			DeviceID:   "dev-1",
			Success:    true,
			LatencyMs:  float64(i) + 1.0,
			PacketLoss: 0.0,
			CheckedAt:  now.Add(time.Duration(i) * time.Second),
		}
		if err := m.store.InsertResult(context.Background(), result); err != nil {
			t.Fatalf("insert result %d: %v", i, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/results/dev-1?limit=5", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var results []CheckResult
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("len(results) = %d, want 5", len(results))
	}
}

func TestHandleDeviceResults_EmptyDeviceID(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/results/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceResults_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/results/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceResults(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleListAlerts tests --

func TestHandleListAlerts_Empty(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/alerts", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var alerts []Alert
	if err := json.NewDecoder(w.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("len(alerts) = %d, want 0", len(alerts))
	}
}

func TestHandleListAlerts_WithData(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	// Insert check first (foreign key constraint).
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	alert := &Alert{
		ID:                  "alert-1",
		CheckID:             "check-1",
		DeviceID:            "dev-1",
		Severity:            "warning",
		Message:             "Device unreachable",
		TriggeredAt:         now,
		ResolvedAt:          nil,
		ConsecutiveFailures: 3,
	}
	if err := m.store.InsertAlert(context.Background(), alert); err != nil {
		t.Fatalf("insert alert: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/alerts", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var alerts []Alert
	if err := json.NewDecoder(w.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(alerts))
	}
	if alerts[0].ID != "alert-1" {
		t.Errorf("alerts[0].ID = %q, want %q", alerts[0].ID, "alert-1")
	}
	if alerts[0].Message != "Device unreachable" {
		t.Errorf("alerts[0].Message = %q, want %q", alerts[0].Message, "Device unreachable")
	}
}

func TestHandleListAlerts_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/alerts", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListAlerts(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleDeviceAlerts tests --

func TestHandleDeviceAlerts_Empty(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/alerts/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var alerts []Alert
	if err := json.NewDecoder(w.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("len(alerts) = %d, want 0", len(alerts))
	}
}

func TestHandleDeviceAlerts_WithData(t *testing.T) {
	m, _ := newTestModule(t)

	now := time.Now().UTC().Truncate(time.Second)
	// Insert checks first (foreign key constraint).
	check1 := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	check2 := &Check{
		ID:              "check-2",
		DeviceID:        "dev-2",
		CheckType:       "icmp",
		Target:          "192.168.1.2",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check1); err != nil {
		t.Fatalf("insert check1: %v", err)
	}
	if err := m.store.InsertCheck(context.Background(), check2); err != nil {
		t.Fatalf("insert check2: %v", err)
	}

	alert1 := &Alert{
		ID:                  "alert-1",
		CheckID:             "check-1",
		DeviceID:            "dev-1",
		Severity:            "warning",
		Message:             "Device dev-1 unreachable",
		TriggeredAt:         now,
		ResolvedAt:          nil,
		ConsecutiveFailures: 3,
	}
	alert2 := &Alert{
		ID:                  "alert-2",
		CheckID:             "check-2",
		DeviceID:            "dev-2",
		Severity:            "critical",
		Message:             "Device dev-2 unreachable",
		TriggeredAt:         now,
		ResolvedAt:          nil,
		ConsecutiveFailures: 5,
	}
	if err := m.store.InsertAlert(context.Background(), alert1); err != nil {
		t.Fatalf("insert alert1: %v", err)
	}
	if err := m.store.InsertAlert(context.Background(), alert2); err != nil {
		t.Fatalf("insert alert2: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/alerts/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var alerts []Alert
	if err := json.NewDecoder(w.Body).Decode(&alerts); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("len(alerts) = %d, want 1", len(alerts))
	}
	if alerts[0].DeviceID != "dev-1" {
		t.Errorf("alerts[0].DeviceID = %q, want %q", alerts[0].DeviceID, "dev-1")
	}
}

func TestHandleDeviceAlerts_EmptyDeviceID(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/alerts/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleDeviceAlerts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleDeviceAlerts_NilStore(t *testing.T) {
	m := &Module{logger: zap.NewNop()}
	req := httptest.NewRequest(http.MethodGet, "/alerts/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceAlerts(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// -- handleDeviceStatus tests --

func TestHandleDeviceStatus_WithData(t *testing.T) {
	m, _ := newTestModule(t)

	// Start the module so the Status method works.
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = m.Stop(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	// Insert check first (foreign key constraint).
	check := &Check{
		ID:              "check-1",
		DeviceID:        "dev-1",
		CheckType:       "icmp",
		Target:          "192.168.1.1",
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.InsertCheck(context.Background(), check); err != nil {
		t.Fatalf("insert check: %v", err)
	}

	result := &CheckResult{
		CheckID:      "check-1",
		DeviceID:     "dev-1",
		Success:      true,
		LatencyMs:    15.3,
		PacketLoss:   0.0,
		ErrorMessage: "",
		CheckedAt:    now,
	}
	if err := m.store.InsertResult(context.Background(), result); err != nil {
		t.Fatalf("insert result: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/status/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var status roles.MonitorStatus
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if status.DeviceID != "dev-1" {
		t.Errorf("status.DeviceID = %q, want %q", status.DeviceID, "dev-1")
	}
	if !status.Healthy {
		t.Errorf("status.Healthy = %v, want true", status.Healthy)
	}
	if status.Message == "" {
		t.Error("status.Message is empty, want non-empty")
	}
}

func TestHandleDeviceStatus_EmptyDeviceID(t *testing.T) {
	m, _ := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/status/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleDeviceStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// -- pulseParseLimit tests --

func TestPulseParseLimit(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		defaultVal int
		want       int
	}{
		{
			name:       "no param returns default",
			query:      "",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "valid param",
			query:      "limit=50",
			defaultVal: 100,
			want:       50,
		},
		{
			name:       "out of range high returns default",
			query:      "limit=2000",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "out of range low returns default",
			query:      "limit=0",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "negative returns default",
			query:      "limit=-10",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "non-numeric returns default",
			query:      "limit=abc",
			defaultVal: 100,
			want:       100,
		},
		{
			name:       "max allowed value",
			query:      "limit=1000",
			defaultVal: 100,
			want:       1000,
		},
		{
			name:       "min allowed value",
			query:      "limit=1",
			defaultVal: 100,
			want:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, http.NoBody)
			got := pulseParseLimit(req, tt.defaultVal)
			if got != tt.want {
				t.Errorf("pulseParseLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}
