package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// mockPluginSource satisfies the PluginSource interface for testing.
type mockPluginSource struct {
	plugins []plugin.Plugin
	routes  map[string][]plugin.Route
}

func (m *mockPluginSource) AllRoutes() map[string][]plugin.Route {
	if m.routes != nil {
		return m.routes
	}
	return map[string][]plugin.Route{}
}

func (m *mockPluginSource) All() []plugin.Plugin {
	return m.plugins
}

// stubPlugin satisfies plugin.Plugin for testing.
type stubPlugin struct {
	info plugin.PluginInfo
}

func (s *stubPlugin) Info() plugin.PluginInfo                        { return s.info }
func (s *stubPlugin) Init(_ context.Context, _ plugin.Dependencies) error { return nil }
func (s *stubPlugin) Start(_ context.Context) error                  { return nil }
func (s *stubPlugin) Stop(_ context.Context) error                   { return nil }

func newTestServer(ready ReadinessChecker) *Server {
	logger, _ := zap.NewDevelopment()
	plugins := &mockPluginSource{
		plugins: []plugin.Plugin{
			&stubPlugin{info: plugin.PluginInfo{
				Name:        "test-plugin",
				Version:     "1.0.0",
				Description: "A test plugin",
			}},
		},
	}
	return New("127.0.0.1:0", plugins, logger, ready, nil)
}

func TestHandleHealthz(t *testing.T) {
	srv := newTestServer(nil)

	req := httptest.NewRequest("GET", "/healthz", http.NoBody)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "alive" {
		t.Errorf("status = %q, want %q", body["status"], "alive")
	}
}

func TestHandleReadyz_Healthy(t *testing.T) {
	ready := ReadinessChecker(func(_ context.Context) error {
		return nil
	})
	srv := newTestServer(ready)

	req := httptest.NewRequest("GET", "/readyz", http.NoBody)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ready" {
		t.Errorf("status = %q, want %q", body["status"], "ready")
	}
}

func TestHandleReadyz_Unhealthy(t *testing.T) {
	ready := ReadinessChecker(func(_ context.Context) error {
		return errors.New("database unreachable")
	})
	srv := newTestServer(ready)

	req := httptest.NewRequest("GET", "/readyz", http.NoBody)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "not ready" {
		t.Errorf("status = %q, want %q", body["status"], "not ready")
	}
	if !strings.Contains(body["error"], "database unreachable") {
		t.Errorf("error = %q, want it to contain %q", body["error"], "database unreachable")
	}
}

func TestHandleReadyz_NilChecker(t *testing.T) {
	srv := newTestServer(nil)

	req := httptest.NewRequest("GET", "/readyz", http.NoBody)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleHealth(t *testing.T) {
	srv := newTestServer(nil)

	req := httptest.NewRequest("GET", "/api/v1/health", http.NoBody)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status = %v, want %q", body["status"], "ok")
	}
	if body["service"] != "netvantage" {
		t.Errorf("service = %v, want %q", body["service"], "netvantage")
	}
	if body["version"] == nil {
		t.Error("expected version field in response")
	}
}

func TestHandlePlugins(t *testing.T) {
	srv := newTestServer(nil)

	req := httptest.NewRequest("GET", "/api/v1/plugins", http.NoBody)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var plugins []map[string]string
	json.NewDecoder(w.Body).Decode(&plugins)
	if len(plugins) != 1 {
		t.Fatalf("len(plugins) = %d, want 1", len(plugins))
	}
	if plugins[0]["name"] != "test-plugin" {
		t.Errorf("name = %q, want %q", plugins[0]["name"], "test-plugin")
	}
	if plugins[0]["version"] != "1.0.0" {
		t.Errorf("version = %q, want %q", plugins[0]["version"], "1.0.0")
	}
}

func TestHandleMetrics(t *testing.T) {
	srv := newTestServer(nil)

	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "go_goroutines") {
		t.Error("expected prometheus Go runtime metrics in /metrics output")
	}
}

func TestMiddlewareChain_Integration(t *testing.T) {
	srv := newTestServer(nil)

	req := httptest.NewRequest("GET", "/healthz", http.NoBody)
	w := httptest.NewRecorder()

	// Use the full handler (with middleware chain) instead of just the mux.
	srv.httpServer.Handler.ServeHTTP(w, req)

	// Check that middleware headers are present.
	if v := w.Header().Get("X-NetVantage-Version"); v == "" {
		t.Error("expected X-NetVantage-Version header from middleware")
	}
	if v := w.Header().Get("X-Request-ID"); v == "" {
		t.Error("expected X-Request-ID header from middleware")
	}
	if v := w.Header().Get("X-Content-Type-Options"); v != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", v, "nosniff")
	}
	if v := w.Header().Get("X-Frame-Options"); v != "DENY" {
		t.Errorf("X-Frame-Options = %q, want %q", v, "DENY")
	}
}

func TestPluginRoutes_Mounted(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	plugins := &mockPluginSource{
		plugins: []plugin.Plugin{},
		routes: map[string][]plugin.Route{
			"recon": {
				{
					Method: "POST",
					Path:   "/scan",
					Handler: func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusAccepted)
					},
				},
			},
		},
	}
	srv := New("127.0.0.1:0", plugins, logger, nil, nil)

	req := httptest.NewRequest("POST", "/api/v1/recon/scan", http.NoBody)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
}
