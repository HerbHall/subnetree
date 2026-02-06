package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
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
	return New("127.0.0.1:0", plugins, logger, ready, nil, nil, false)
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
	if body["service"] != "subnetree" {
		t.Errorf("service = %v, want %q", body["service"], "subnetree")
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
	if v := w.Header().Get("X-SubNetree-Version"); v == "" {
		t.Error("expected X-SubNetree-Version header from middleware")
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
	srv := New("127.0.0.1:0", plugins, logger, nil, nil, nil, false)

	req := httptest.NewRequest("POST", "/api/v1/recon/scan", http.NoBody)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
}

// --- Graceful Shutdown Tests ---

// newTestServerWithListener creates a server bound to an available port.
func newTestServerWithListener(t *testing.T, slowHandler http.HandlerFunc) (*Server, net.Listener, string) {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	plugins := &mockPluginSource{
		plugins: []plugin.Plugin{},
		routes:  map[string][]plugin.Route{},
	}
	if slowHandler != nil {
		plugins.routes = map[string][]plugin.Route{
			"test": {{Method: "GET", Path: "/slow", Handler: slowHandler}},
		}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	addr := listener.Addr().String()
	srv := New(addr, plugins, logger, nil, nil, nil, false)

	return srv, listener, addr
}

func TestShutdown_DrainsInFlightRequests(t *testing.T) {
	// Test: HTTP server drains in-flight requests before closing.
	requestStarted := make(chan struct{})
	requestCanContinue := make(chan struct{})
	var requestCompleted atomic.Bool

	slowHandler := func(w http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-requestCanContinue
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("done"))
		requestCompleted.Store(true)
	}

	srv, listener, addr := newTestServerWithListener(t, slowHandler)

	// Start server in background.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.httpServer.Serve(listener)
	}()

	// Give server time to start.
	time.Sleep(50 * time.Millisecond)

	// Start an in-flight request.
	client := &http.Client{Timeout: 5 * time.Second}
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)

	go func() {
		req, _ := http.NewRequestWithContext(context.Background(), "GET", fmt.Sprintf("http://%s/api/v1/test/slow", addr), http.NoBody)
		resp, err := client.Do(req) //nolint:bodyclose // Body closed in select below after channel receive
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	// Wait for request to start processing.
	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start within timeout")
	}

	// Initiate shutdown while request is in-flight.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- srv.Shutdown(shutdownCtx)
	}()

	// Allow the request to complete.
	close(requestCanContinue)

	// Wait for the request to finish.
	select {
	case resp := <-respCh:
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("response status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if string(body) != "done" {
			t.Errorf("response body = %q, want %q", string(body), "done")
		}
	case err := <-errCh:
		t.Fatalf("request failed: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("request did not complete within timeout")
	}

	// Wait for shutdown to complete.
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("shutdown error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not complete within timeout")
	}

	if !requestCompleted.Load() {
		t.Error("request handler did not complete")
	}
}

func TestShutdown_RejectsNewConnections(t *testing.T) {
	// After shutdown starts, new connections should be rejected.
	srv, listener, addr := newTestServerWithListener(t, nil)

	// Start server.
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.httpServer.Serve(listener)
	}()

	time.Sleep(50 * time.Millisecond)

	// Verify server is accepting connections.
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", fmt.Sprintf("http://%s/healthz", addr), http.NoBody)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("pre-shutdown request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pre-shutdown status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Initiate shutdown.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}

	// New requests should fail after shutdown.
	req2, _ := http.NewRequestWithContext(context.Background(), "GET", fmt.Sprintf("http://%s/healthz", addr), http.NoBody)
	resp2, err := client.Do(req2)
	if err == nil {
		resp2.Body.Close()
		t.Error("expected request after shutdown to fail")
	}
}

func TestShutdown_CompletesWithinTimeout(t *testing.T) {
	// Test: Shutdown completes within configured maximum timeout.
	srv, listener, _ := newTestServerWithListener(t, nil)

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.httpServer.Serve(listener)
	}()

	time.Sleep(50 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	err := srv.Shutdown(shutdownCtx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("shutdown error: %v", err)
	}

	if elapsed > 500*time.Millisecond {
		t.Errorf("shutdown took %v, expected < 500ms for idle server", elapsed)
	}
}

func TestShutdown_TimeoutEnforced(t *testing.T) {
	// Test: Shutdown times out if requests don't complete.
	blockForever := make(chan struct{})
	slowHandler := func(w http.ResponseWriter, _ *http.Request) {
		<-blockForever // Never returns
	}

	srv, listener, addr := newTestServerWithListener(t, slowHandler)

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.httpServer.Serve(listener)
	}()

	time.Sleep(50 * time.Millisecond)

	// Start a request that will block forever.
	go func() {
		client := &http.Client{Timeout: 0}
		req, _ := http.NewRequestWithContext(context.Background(), "GET", fmt.Sprintf("http://%s/api/v1/test/slow", addr), http.NoBody)
		resp, err := client.Do(req)
		if err == nil && resp != nil {
			resp.Body.Close()
		}
	}()

	time.Sleep(50 * time.Millisecond)

	// Shutdown with a short timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := srv.Shutdown(shutdownCtx)
	elapsed := time.Since(start)

	// Should return with context deadline exceeded.
	if err == nil {
		t.Error("expected shutdown to return error due to timeout")
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}

	// Should have taken approximately the timeout duration.
	if elapsed < 90*time.Millisecond || elapsed > 300*time.Millisecond {
		t.Errorf("shutdown took %v, expected ~100ms (timeout duration)", elapsed)
	}

	close(blockForever)
}

func TestShutdown_MultipleInFlightRequests(t *testing.T) {
	// Multiple in-flight requests should all be drained.
	var activeRequests atomic.Int32
	var completedRequests atomic.Int32
	requestGate := make(chan struct{})

	slowHandler := func(w http.ResponseWriter, _ *http.Request) {
		activeRequests.Add(1)
		<-requestGate
		w.WriteHeader(http.StatusOK)
		completedRequests.Add(1)
	}

	srv, listener, addr := newTestServerWithListener(t, slowHandler)

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.httpServer.Serve(listener)
	}()

	time.Sleep(50 * time.Millisecond)

	// Start multiple concurrent requests.
	const numRequests = 5
	var wg sync.WaitGroup
	client := &http.Client{Timeout: 5 * time.Second}

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", fmt.Sprintf("http://%s/api/v1/test/slow", addr), http.NoBody)
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}()
	}

	// Wait for all requests to be in-flight.
	for activeRequests.Load() < numRequests {
		time.Sleep(10 * time.Millisecond)
	}

	// Start shutdown.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- srv.Shutdown(shutdownCtx)
	}()

	// Let all requests complete.
	close(requestGate)

	// Wait for shutdown.
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("shutdown error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not complete")
	}

	wg.Wait()

	if completed := completedRequests.Load(); completed != numRequests {
		t.Errorf("completed requests = %d, want %d", completed, numRequests)
	}
}
