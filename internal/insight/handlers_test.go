package insight

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/analytics"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

func newTestModule(t *testing.T) *Module {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := New()
	err = m.Init(context.Background(), plugin.Dependencies{
		Logger: zap.NewNop(),
		Store:  db,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	return m
}

func TestHandleListAnomalies_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/anomalies", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListAnomalies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got []analytics.Anomaly
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty array, got %d items", len(got))
	}
}

func TestHandleDeviceAnomalies_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/anomalies/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceAnomalies(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got []analytics.Anomaly
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty array, got %d items", len(got))
	}
}

func TestHandleDeviceForecasts_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/forecasts/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceForecasts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got []analytics.Forecast
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty array, got %d items", len(got))
	}
}

func TestHandleListCorrelations_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/correlations", http.NoBody)
	w := httptest.NewRecorder()

	m.handleListCorrelations(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got []analytics.AlertGroup
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty array, got %d items", len(got))
	}
}

func TestHandleDeviceBaselines_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/baselines/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	w := httptest.NewRecorder()

	m.handleDeviceBaselines(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got []analytics.Baseline
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty array, got %d items", len(got))
	}
}

func TestHandleNLQuery_Stub(t *testing.T) {
	m := newTestModule(t)

	body := strings.NewReader(`{"query":"show me anomalies"}`)
	req := httptest.NewRequest(http.MethodPost, "/query", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	m.handleNLQuery(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var got map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["detail"] == nil || got["detail"] == "" {
		t.Error("expected error detail in response body")
	}
}

func TestHandleNLQuery_EmptyBody(t *testing.T) {
	m := newTestModule(t)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/query", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	m.handleNLQuery(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var got map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["detail"] != "query is required" {
		t.Errorf("detail = %q, want %q", got["detail"], "query is required")
	}
}

func TestHandleDeviceAnomalies_MissingID(t *testing.T) {
	m := newTestModule(t)

	// No SetPathValue -- simulates missing path parameter.
	req := httptest.NewRequest(http.MethodGet, "/anomalies/", http.NoBody)
	w := httptest.NewRecorder()

	m.handleDeviceAnomalies(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var got map[string]any
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["detail"] != "device_id is required" {
		t.Errorf("detail = %q, want %q", got["detail"], "device_id is required")
	}
}
