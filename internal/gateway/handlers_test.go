package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// newTestModule creates a gateway Module backed by an in-memory SQLite DB
// with sessions manager ready for handler testing.
func newTestModule(t *testing.T) *Module {
	t.Helper()

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "gateway", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	m := &Module{
		logger:   zap.NewNop(),
		store:    NewGatewayStore(db.DB()),
		cfg:      DefaultConfig(),
		sessions: NewSessionManager(100),
		proxies:  NewReverseProxyManager(zap.NewNop()),
		ctx:      ctx,
	}
	return m
}

// testEventBus is a synchronous event bus for testing.
type testEventBus struct {
	mu     sync.Mutex
	events []plugin.Event
}

func (b *testEventBus) Publish(_ context.Context, event plugin.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
	return nil
}

func (b *testEventBus) PublishAsync(_ context.Context, event plugin.Event) {
	_ = b.Publish(context.Background(), event)
}

func (b *testEventBus) Subscribe(_ string, _ plugin.EventHandler) func() {
	return func() {}
}

func (b *testEventBus) SubscribeAll(_ plugin.EventHandler) func() {
	return func() {}
}

func (b *testEventBus) lastEvent() *plugin.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.events) == 0 {
		return nil
	}
	return &b.events[len(b.events)-1]
}

// --- List Sessions ---

func TestHandleListSessions_Empty(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/sessions", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleListSessions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var sessions []sessionView
	if err := json.NewDecoder(rr.Body).Decode(&sessions); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("len = %d, want 0", len(sessions))
	}
}

func TestHandleListSessions_WithData(t *testing.T) {
	m := newTestModule(t)
	s := newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute))
	s.BytesIn.Store(100)
	_ = m.sessions.Create(s)

	req := httptest.NewRequest(http.MethodGet, "/sessions", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleListSessions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var sessions []sessionView
	if err := json.NewDecoder(rr.Body).Decode(&sessions); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len = %d, want 1", len(sessions))
	}
	if sessions[0].ID != "s1" {
		t.Errorf("ID = %q, want %q", sessions[0].ID, "s1")
	}
	if sessions[0].BytesIn != 100 {
		t.Errorf("BytesIn = %d, want 100", sessions[0].BytesIn)
	}
}

// --- Get Session ---

func TestHandleGetSession_Found(t *testing.T) {
	m := newTestModule(t)
	_ = m.sessions.Create(newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute)))

	req := httptest.NewRequest(http.MethodGet, "/sessions/s1", http.NoBody)
	req.SetPathValue("id", "s1")
	rr := httptest.NewRecorder()
	m.handleGetSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var sv sessionView
	if err := json.NewDecoder(rr.Body).Decode(&sv); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if sv.ID != "s1" {
		t.Errorf("ID = %q, want %q", sv.ID, "s1")
	}
	if sv.DeviceID != "dev-1" {
		t.Errorf("DeviceID = %q, want %q", sv.DeviceID, "dev-1")
	}
}

func TestHandleGetSession_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/sessions/missing", http.NoBody)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	m.handleGetSession(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleGetSession_MissingID(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/sessions/", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleGetSession(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// --- Delete Session ---

func TestHandleDeleteSession_Success(t *testing.T) {
	m := newTestModule(t)
	bus := &testEventBus{}
	m.bus = bus

	_ = m.sessions.Create(newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute)))

	req := httptest.NewRequest(http.MethodDelete, "/sessions/s1", http.NoBody)
	req.SetPathValue("id", "s1")
	rr := httptest.NewRecorder()
	m.handleDeleteSession(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}

	// Verify session is removed.
	if m.sessions.Count() != 0 {
		t.Errorf("session count = %d, want 0", m.sessions.Count())
	}

	// Verify event was published.
	if e := bus.lastEvent(); e == nil || e.Topic != TopicSessionClosed {
		t.Error("expected gateway.session.closed event")
	}

	// Verify audit entry was created.
	entries, err := m.store.ListAuditEntries(context.Background(), "", 100)
	if err != nil {
		t.Fatalf("ListAuditEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(entries))
	}
	if entries[0].Action != "closed:manual" {
		t.Errorf("audit action = %q, want %q", entries[0].Action, "closed:manual")
	}
}

func TestHandleDeleteSession_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodDelete, "/sessions/missing", http.NoBody)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	m.handleDeleteSession(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDeleteSession_MissingID(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodDelete, "/sessions/", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleDeleteSession(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// --- Status ---

func TestHandleStatus(t *testing.T) {
	m := newTestModule(t)
	_ = m.sessions.Create(newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute)))

	req := httptest.NewRequest(http.MethodGet, "/status", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["active_sessions"] != float64(1) {
		t.Errorf("active_sessions = %v, want 1", result["active_sessions"])
	}
	if result["max_sessions"] != float64(100) {
		t.Errorf("max_sessions = %v, want 100", result["max_sessions"])
	}
	if result["store"] != "connected" {
		t.Errorf("store = %v, want %q", result["store"], "connected")
	}
}

func TestHandleStatus_NoStore(t *testing.T) {
	m := &Module{
		logger:   zap.NewNop(),
		cfg:      DefaultConfig(),
		sessions: NewSessionManager(100),
		ctx:      context.Background(),
	}

	req := httptest.NewRequest(http.MethodGet, "/status", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["store"] != "unavailable" {
		t.Errorf("store = %v, want %q", result["store"], "unavailable")
	}
}

// --- Audit ---

func TestHandleListAudit_Success(t *testing.T) {
	m := newTestModule(t)
	now := time.Now().UTC()

	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		SessionID: "s1", DeviceID: "dev-1", SessionType: "http_proxy",
		Target: "192.168.1.1:80", Action: "created", Timestamp: now,
	})
	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		SessionID: "s2", DeviceID: "dev-2", SessionType: "ssh",
		Target: "192.168.1.2:22", Action: "created", Timestamp: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/audit", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleListAudit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var entries []AuditEntry
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("len = %d, want 2", len(entries))
	}
}

func TestHandleListAudit_WithDeviceFilter(t *testing.T) {
	m := newTestModule(t)
	now := time.Now().UTC()

	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		SessionID: "s1", DeviceID: "dev-1", SessionType: "http_proxy",
		Target: "192.168.1.1:80", Action: "created", Timestamp: now,
	})
	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		SessionID: "s2", DeviceID: "dev-2", SessionType: "ssh",
		Target: "192.168.1.2:22", Action: "created", Timestamp: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/audit?device_id=dev-1", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleListAudit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var entries []AuditEntry
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1", len(entries))
	}
}

func TestHandleListAudit_NilStore(t *testing.T) {
	m := &Module{
		logger:   zap.NewNop(),
		cfg:      DefaultConfig(),
		sessions: NewSessionManager(100),
		ctx:      context.Background(),
	}

	req := httptest.NewRequest(http.MethodGet, "/audit", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleListAudit(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

// --- Create Proxy ---

func TestHandleCreateProxy_Success(t *testing.T) {
	m := newTestModule(t)
	bus := &testEventBus{}
	m.bus = bus

	body := `{"port": 8080, "scheme": "http", "target": "192.168.1.100"}`
	req := httptest.NewRequest(http.MethodPost, "/proxy/dev-1", strings.NewReader(body))
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()

	m.handleCreateProxy(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusCreated, rr.Body.String())
	}

	var resp createProxyResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Session.DeviceID != "dev-1" {
		t.Errorf("DeviceID = %q, want %q", resp.Session.DeviceID, "dev-1")
	}
	if resp.Session.Target.Host != "192.168.1.100" {
		t.Errorf("Target.Host = %q, want %q", resp.Session.Target.Host, "192.168.1.100")
	}
	if resp.Session.Target.Port != 8080 {
		t.Errorf("Target.Port = %d, want %d", resp.Session.Target.Port, 8080)
	}
	if resp.ProxyURL == "" {
		t.Error("ProxyURL should not be empty")
	}

	// Verify session was created.
	if m.sessions.Count() != 1 {
		t.Errorf("session count = %d, want 1", m.sessions.Count())
	}

	// Verify event was published.
	if e := bus.lastEvent(); e == nil || e.Topic != TopicSessionCreated {
		t.Error("expected gateway.session.created event")
	}

	// Verify audit entry.
	entries, err := m.store.ListAuditEntries(context.Background(), "", 100)
	if err != nil {
		t.Fatalf("ListAuditEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(entries))
	}
	if entries[0].Action != "created" {
		t.Errorf("audit action = %q, want %q", entries[0].Action, "created")
	}
}

func TestHandleCreateProxy_DefaultPort(t *testing.T) {
	m := newTestModule(t)

	body := `{"target": "192.168.1.100"}`
	req := httptest.NewRequest(http.MethodPost, "/proxy/dev-1", strings.NewReader(body))
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()

	m.handleCreateProxy(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusCreated, rr.Body.String())
	}

	var resp createProxyResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Should use DefaultProxyPort from config (80).
	if resp.Session.Target.Port != 80 {
		t.Errorf("Target.Port = %d, want %d (default)", resp.Session.Target.Port, 80)
	}
}

func TestHandleCreateProxy_MissingTarget(t *testing.T) {
	m := newTestModule(t)

	body := `{"port": 80}`
	req := httptest.NewRequest(http.MethodPost, "/proxy/dev-1", strings.NewReader(body))
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()

	m.handleCreateProxy(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateProxy_NilSessions(t *testing.T) {
	m := &Module{
		logger: zap.NewNop(),
		cfg:    DefaultConfig(),
		ctx:    context.Background(),
	}

	body := `{"port": 80, "target": "192.168.1.1"}`
	req := httptest.NewRequest(http.MethodPost, "/proxy/dev-1", strings.NewReader(body))
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()

	m.handleCreateProxy(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleCreateProxy_InvalidBody(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodPost, "/proxy/dev-1", strings.NewReader("not json"))
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()

	m.handleCreateProxy(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateProxy_MissingDeviceID(t *testing.T) {
	m := newTestModule(t)

	body := `{"port": 80, "target": "192.168.1.1"}`
	req := httptest.NewRequest(http.MethodPost, "/proxy/", strings.NewReader(body))
	rr := httptest.NewRecorder()

	m.handleCreateProxy(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// --- Proxy Traffic ---

func TestHandleProxyTraffic_Success(t *testing.T) {
	// Start a backend that echoes path.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	}))
	defer backend.Close()

	m := newTestModule(t)

	// Parse backend address.
	host, port := parseHostPort(t, backend.URL)

	// Create a session and proxy manually.
	session := &Session{
		ID:          "test-session",
		DeviceID:    "dev-1",
		UserID:      "user-1",
		SessionType: SessionTypeProxy,
		Target:      ProxyTarget{Host: host, Port: port},
		SourceIP:    "10.0.0.1",
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(30 * time.Minute),
	}
	_ = m.sessions.Create(session)
	_ = m.proxies.CreateProxy(session, "http")

	req := httptest.NewRequest(http.MethodGet, "/proxy/s/test-session/some/resource", http.NoBody)
	req.SetPathValue("session_id", "test-session")
	req.SetPathValue("path", "some/resource")
	rr := httptest.NewRecorder()

	m.handleProxyTraffic(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body, _ := io.ReadAll(rr.Body)
	want := "path=/some/resource"
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}
}

func TestHandleProxyTraffic_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/proxy/s/nonexistent/path", http.NoBody)
	req.SetPathValue("session_id", "nonexistent")
	req.SetPathValue("path", "path")
	rr := httptest.NewRecorder()

	m.handleProxyTraffic(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleProxyTraffic_ExpiredSession(t *testing.T) {
	m := newTestModule(t)

	// Create an already-expired session.
	session := &Session{
		ID:          "expired-session",
		DeviceID:    "dev-1",
		UserID:      "user-1",
		SessionType: SessionTypeProxy,
		Target:      ProxyTarget{Host: "127.0.0.1", Port: 8080},
		SourceIP:    "10.0.0.1",
		CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
		ExpiresAt:   time.Now().UTC().Add(-1 * time.Minute),
	}
	_ = m.sessions.Create(session)
	_ = m.proxies.CreateProxy(session, "http")

	req := httptest.NewRequest(http.MethodGet, "/proxy/s/expired-session/path", http.NoBody)
	req.SetPathValue("session_id", "expired-session")
	req.SetPathValue("path", "path")
	rr := httptest.NewRecorder()

	m.handleProxyTraffic(rr, req)

	if rr.Code != http.StatusGone {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusGone)
	}

	// Session should be cleaned up.
	if m.sessions.Count() != 0 {
		t.Errorf("session count = %d, want 0 after expiry", m.sessions.Count())
	}
}

func TestHandleProxyTraffic_NilSessions(t *testing.T) {
	m := &Module{
		logger: zap.NewNop(),
		cfg:    DefaultConfig(),
		ctx:    context.Background(),
	}

	req := httptest.NewRequest(http.MethodGet, "/proxy/s/s1/path", http.NoBody)
	req.SetPathValue("session_id", "s1")
	req.SetPathValue("path", "path")
	rr := httptest.NewRecorder()

	m.handleProxyTraffic(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleProxyTraffic_RootPath(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	}))
	defer backend.Close()

	m := newTestModule(t)
	host, port := parseHostPort(t, backend.URL)

	session := &Session{
		ID:          "test-session",
		DeviceID:    "dev-1",
		UserID:      "user-1",
		SessionType: SessionTypeProxy,
		Target:      ProxyTarget{Host: host, Port: port},
		SourceIP:    "10.0.0.1",
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(30 * time.Minute),
	}
	_ = m.sessions.Create(session)
	_ = m.proxies.CreateProxy(session, "http")

	// Empty path should forward to "/".
	req := httptest.NewRequest(http.MethodGet, "/proxy/s/test-session/", http.NoBody)
	req.SetPathValue("session_id", "test-session")
	req.SetPathValue("path", "")
	rr := httptest.NewRecorder()

	m.handleProxyTraffic(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body, _ := io.ReadAll(rr.Body)
	want := "path=/"
	if string(body) != want {
		t.Errorf("body = %q, want %q", string(body), want)
	}
}

// --- Helper Tests ---

func TestGatewayParseLimit(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		defLimit int
		want     int
	}{
		{"default", "", 100, 100},
		{"valid", "?limit=50", 100, 50},
		{"too_large", "?limit=5000", 100, 100},
		{"negative", "?limit=-1", 100, 100},
		{"non_numeric", "?limit=abc", 100, 100},
		{"zero", "?limit=0", 100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/audit"+tt.query, http.NoBody)
			got := gatewayParseLimit(req, tt.defLimit)
			if got != tt.want {
				t.Errorf("gatewayParseLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}
