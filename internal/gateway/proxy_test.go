package gateway

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestCreateProxy(t *testing.T) {
	pm := NewReverseProxyManager(zap.NewNop())

	session := &Session{
		ID:       "s1",
		DeviceID: "dev-1",
		Target:   ProxyTarget{Host: "127.0.0.1", Port: 8080},
	}

	if err := pm.CreateProxy(session, "http"); err != nil {
		t.Fatalf("CreateProxy() error = %v", err)
	}

	if pm.Count() != 1 {
		t.Errorf("Count() = %d, want 1", pm.Count())
	}
}

func TestCreateProxyServe(t *testing.T) {
	// Start a backend server that returns a known response.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "test")
		fmt.Fprintf(w, "hello from backend: %s", r.URL.Path)
	}))
	defer backend.Close()

	// Parse backend host and port.
	host, port := parseHostPort(t, backend.URL)

	pm := NewReverseProxyManager(zap.NewNop())
	session := &Session{
		ID:       "s1",
		DeviceID: "dev-1",
		Target:   ProxyTarget{Host: host, Port: port},
	}

	if err := pm.CreateProxy(session, "http"); err != nil {
		t.Fatalf("CreateProxy() error = %v", err)
	}

	// Serve a request through the proxy.
	req := httptest.NewRequest(http.MethodGet, "/test-path", http.NoBody)
	rr := httptest.NewRecorder()

	if err := pm.ServeProxy("s1", rr, req); err != nil {
		t.Fatalf("ServeProxy() error = %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body, _ := io.ReadAll(rr.Body)
	want := "hello from backend: /test-path"
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}

	if rr.Header().Get("X-Backend") != "test" {
		t.Errorf("X-Backend = %q, want %q", rr.Header().Get("X-Backend"), "test")
	}
}

func TestServeProxyNotFound(t *testing.T) {
	pm := NewReverseProxyManager(zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	err := pm.ServeProxy("nonexistent", rr, req)
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
}

func TestRemoveProxy(t *testing.T) {
	pm := NewReverseProxyManager(zap.NewNop())

	session := &Session{
		ID:       "s1",
		DeviceID: "dev-1",
		Target:   ProxyTarget{Host: "127.0.0.1", Port: 8080},
	}
	_ = pm.CreateProxy(session, "http")

	pm.RemoveProxy("s1")

	if pm.Count() != 0 {
		t.Errorf("Count() = %d after remove, want 0", pm.Count())
	}

	// Serving after removal should fail.
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	if err := pm.ServeProxy("s1", rr, req); err == nil {
		t.Error("expected error after proxy removal")
	}
}

func TestRemoveProxy_Nonexistent(t *testing.T) {
	pm := NewReverseProxyManager(zap.NewNop())

	// Should not panic.
	pm.RemoveProxy("nonexistent")

	if pm.Count() != 0 {
		t.Errorf("Count() = %d, want 0", pm.Count())
	}
}

func TestCloseAll(t *testing.T) {
	pm := NewReverseProxyManager(zap.NewNop())

	for i := range 5 {
		s := &Session{
			ID:       fmt.Sprintf("s%d", i),
			DeviceID: "dev-1",
			Target:   ProxyTarget{Host: "127.0.0.1", Port: 8080 + i},
		}
		_ = pm.CreateProxy(s, "http")
	}

	if pm.Count() != 5 {
		t.Fatalf("Count() = %d, want 5", pm.Count())
	}

	pm.CloseAll()

	if pm.Count() != 0 {
		t.Errorf("Count() after CloseAll = %d, want 0", pm.Count())
	}
}

func TestProxyConcurrent(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer backend.Close()

	host, port := parseHostPort(t, backend.URL)
	pm := NewReverseProxyManager(zap.NewNop())

	// Create multiple sessions.
	const numSessions = 10
	for i := range numSessions {
		s := &Session{
			ID:       fmt.Sprintf("s%d", i),
			DeviceID: "dev-1",
			Target:   ProxyTarget{Host: host, Port: port},
		}
		if err := pm.CreateProxy(s, "http"); err != nil {
			t.Fatalf("CreateProxy(s%d) error = %v", i, err)
		}
	}

	// Concurrently serve requests and remove proxies.
	var wg sync.WaitGroup
	for i := range numSessions {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sessionID := fmt.Sprintf("s%d", idx)

			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			rr := httptest.NewRecorder()
			_ = pm.ServeProxy(sessionID, rr, req)

			pm.RemoveProxy(sessionID)
		}(i)
	}
	wg.Wait()

	if pm.Count() != 0 {
		t.Errorf("Count() after concurrent removal = %d, want 0", pm.Count())
	}
}

func TestProxyForwardsHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo the custom header back.
		w.Header().Set("X-Echo", r.Header.Get("X-Custom"))
		fmt.Fprint(w, "ok")
	}))
	defer backend.Close()

	host, port := parseHostPort(t, backend.URL)
	pm := NewReverseProxyManager(zap.NewNop())

	session := &Session{
		ID:       "s1",
		DeviceID: "dev-1",
		Target:   ProxyTarget{Host: host, Port: port},
	}
	if err := pm.CreateProxy(session, "http"); err != nil {
		t.Fatalf("CreateProxy() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Custom", "test-value")
	rr := httptest.NewRecorder()

	if err := pm.ServeProxy("s1", rr, req); err != nil {
		t.Fatalf("ServeProxy() error = %v", err)
	}

	if rr.Header().Get("X-Echo") != "test-value" {
		t.Errorf("X-Echo = %q, want %q", rr.Header().Get("X-Echo"), "test-value")
	}
}

func TestProxyDefaultScheme(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer backend.Close()

	host, port := parseHostPort(t, backend.URL)
	pm := NewReverseProxyManager(zap.NewNop())

	session := &Session{
		ID:       "s1",
		DeviceID: "dev-1",
		Target:   ProxyTarget{Host: host, Port: port},
	}

	// Empty scheme should default to "http".
	if err := pm.CreateProxy(session, ""); err != nil {
		t.Fatalf("CreateProxy() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	if err := pm.ServeProxy("s1", rr, req); err != nil {
		t.Fatalf("ServeProxy() error = %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestProxyErrorHandler(t *testing.T) {
	pm := NewReverseProxyManager(zap.NewNop())

	// Create a proxy targeting a non-existent backend.
	session := &Session{
		ID:       "s1",
		DeviceID: "dev-1",
		Target:   ProxyTarget{Host: "127.0.0.1", Port: 1}, // Port 1 should refuse connections
	}
	if err := pm.CreateProxy(session, "http"); err != nil {
		t.Fatalf("CreateProxy() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()

	// Should not panic even though the backend is unreachable.
	if err := pm.ServeProxy("s1", rr, req); err != nil {
		t.Fatalf("ServeProxy() error = %v", err)
	}

	// Custom error handler returns 502 Bad Gateway.
	if rr.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
}

// parseHostPort splits an httptest server URL into host and port.
func parseHostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	// httptest URLs look like "http://127.0.0.1:PORT"
	var host string
	var port int
	// Strip scheme.
	addr := rawURL
	if len(addr) > 7 && addr[:7] == "http://" {
		addr = addr[7:]
	}
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			host = addr[:i]
			p := 0
			for _, c := range addr[i+1:] {
				p = p*10 + int(c-'0')
			}
			port = p
			break
		}
	}
	if host == "" {
		t.Fatalf("failed to parse host:port from %q", rawURL)
	}
	return host, port
}
