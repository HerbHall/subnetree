package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// createProxyRequest is the JSON body for POST /proxy/{device_id}.
type createProxyRequest struct {
	Port   int    `json:"port"`
	Scheme string `json:"scheme"`
	Target string `json:"target"` // Optional fallback IP when DeviceLookup unavailable
}

// createProxyResponse is the JSON response for a newly created proxy session.
type createProxyResponse struct {
	Session  sessionView `json:"session"`
	ProxyURL string      `json:"proxy_url"`
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/sessions", Handler: m.handleListSessions},
		{Method: "GET", Path: "/sessions/{id}", Handler: m.handleGetSession},
		{Method: "DELETE", Path: "/sessions/{id}", Handler: m.handleDeleteSession},
		{Method: "GET", Path: "/status", Handler: m.handleStatus},
		{Method: "GET", Path: "/audit", Handler: m.handleListAudit},
		{Method: "POST", Path: "/proxy/{device_id}", Handler: m.handleCreateProxy},
		{Method: "GET", Path: "/proxy/s/{session_id}/{path...}", Handler: m.handleProxyTraffic},
		{Method: "POST", Path: "/proxy/s/{session_id}/{path...}", Handler: m.handleProxyTraffic},
		{Method: "PUT", Path: "/proxy/s/{session_id}/{path...}", Handler: m.handleProxyTraffic},
		{Method: "DELETE", Path: "/proxy/s/{session_id}/{path...}", Handler: m.handleProxyTraffic},
		{Method: "PATCH", Path: "/proxy/s/{session_id}/{path...}", Handler: m.handleProxyTraffic},
	}
}

// handleListSessions returns all active sessions.
func (m *Module) handleListSessions(w http.ResponseWriter, _ *http.Request) {
	if m.sessions == nil {
		gatewayWriteJSON(w, http.StatusOK, []any{})
		return
	}

	sessions := m.sessions.List()
	views := make([]sessionView, len(sessions))
	for i, s := range sessions {
		views[i] = s.toView()
	}
	gatewayWriteJSON(w, http.StatusOK, views)
}

// handleGetSession returns a single session by ID.
func (m *Module) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		gatewayWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if m.sessions == nil {
		gatewayWriteError(w, http.StatusNotFound, "session not found")
		return
	}

	session, ok := m.sessions.Get(id)
	if !ok {
		gatewayWriteError(w, http.StatusNotFound, "session not found")
		return
	}

	gatewayWriteJSON(w, http.StatusOK, session.toView())
}

// handleDeleteSession closes and removes a session.
func (m *Module) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		gatewayWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if m.sessions == nil {
		gatewayWriteError(w, http.StatusNotFound, "session not found")
		return
	}

	session, ok := m.sessions.Get(id)
	if !ok {
		gatewayWriteError(w, http.StatusNotFound, "session not found")
		return
	}

	m.sessions.Delete(id)
	if m.proxies != nil {
		m.proxies.RemoveProxy(id)
	}
	m.logSessionClosed(session, "manual")

	w.WriteHeader(http.StatusNoContent)
}

// handleStatus returns gateway status including session count and capacity.
func (m *Module) handleStatus(w http.ResponseWriter, _ *http.Request) {
	sessionCount := 0
	if m.sessions != nil {
		sessionCount = m.sessions.Count()
	}

	storeStatus := "unavailable"
	if m.store != nil {
		storeStatus = "connected"
	}

	gatewayWriteJSON(w, http.StatusOK, map[string]any{
		"active_sessions": sessionCount,
		"max_sessions":    m.cfg.MaxSessions,
		"store":           storeStatus,
	})
}

// handleListAudit returns audit log entries with optional device filtering.
func (m *Module) handleListAudit(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		gatewayWriteError(w, http.StatusServiceUnavailable, "gateway store not available")
		return
	}

	deviceID := r.URL.Query().Get("device_id")
	limit := gatewayParseLimit(r, 100)

	entries, err := m.store.ListAuditEntries(r.Context(), deviceID, limit)
	if err != nil {
		m.logger.Warn("failed to list gateway audit entries", zap.Error(err))
		gatewayWriteError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	gatewayWriteJSON(w, http.StatusOK, entries)
}

// --- Proxy Handlers ---

// handleCreateProxy creates a new proxy session for a device.
// POST /proxy/{device_id} with JSON body: {"port": 80, "scheme": "http", "target": "192.168.1.1"}
func (m *Module) handleCreateProxy(w http.ResponseWriter, r *http.Request) {
	if m.sessions == nil {
		gatewayWriteError(w, http.StatusServiceUnavailable, "gateway not ready")
		return
	}

	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		gatewayWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	var body createProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		gatewayWriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Port <= 0 {
		body.Port = m.cfg.DefaultProxyPort
	}
	if body.Scheme == "" {
		body.Scheme = "http"
	}

	// Resolve the target host: try DeviceLookup first, fall back to body.Target.
	targetHost := body.Target
	if m.deviceLookup != nil {
		device, err := m.deviceLookup.DeviceByID(r.Context(), deviceID)
		if err == nil && device != nil && len(device.IPAddresses) > 0 {
			targetHost = device.IPAddresses[0]
		}
	}

	if targetHost == "" {
		gatewayWriteError(w, http.StatusBadRequest, "target is required when device lookup is unavailable")
		return
	}

	// Create session.
	session := &Session{
		ID:          generateSessionID(),
		DeviceID:    deviceID,
		UserID:      "anonymous", // Populated by auth middleware in production
		SessionType: SessionTypeProxy,
		Target:      ProxyTarget{Host: targetHost, Port: body.Port},
		SourceIP:    r.RemoteAddr,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(m.cfg.SessionTimeout),
	}

	if err := m.sessions.Create(session); err != nil {
		gatewayWriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	// Create reverse proxy.
	if m.proxies != nil {
		if err := m.proxies.CreateProxy(session, body.Scheme); err != nil {
			m.sessions.Delete(session.ID)
			m.logger.Warn("failed to create reverse proxy", zap.Error(err))
			gatewayWriteError(w, http.StatusInternalServerError, "failed to create proxy")
			return
		}
	}

	// Record audit entry.
	if m.store != nil {
		entry := &AuditEntry{
			SessionID:   session.ID,
			DeviceID:    deviceID,
			UserID:      session.UserID,
			SessionType: string(SessionTypeProxy),
			Target:      fmt.Sprintf("%s:%d", targetHost, body.Port),
			Action:      "created",
			SourceIP:    r.RemoteAddr,
			Timestamp:   time.Now().UTC(),
		}
		if err := m.store.InsertAuditEntry(r.Context(), entry); err != nil {
			m.logger.Warn("failed to write proxy creation audit entry", zap.Error(err))
		}
	}

	// Publish event.
	m.publishEvent(TopicSessionCreated, map[string]string{
		"session_id":   session.ID,
		"device_id":    deviceID,
		"session_type": string(SessionTypeProxy),
		"target":       fmt.Sprintf("%s:%d", targetHost, body.Port),
	})

	proxyURL := fmt.Sprintf("/proxy/s/%s/", session.ID)

	gatewayWriteJSON(w, http.StatusCreated, createProxyResponse{
		Session:  session.toView(),
		ProxyURL: proxyURL,
	})
}

// handleProxyTraffic forwards HTTP traffic through an active proxy session.
func (m *Module) handleProxyTraffic(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		gatewayWriteError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	if m.sessions == nil {
		gatewayWriteError(w, http.StatusServiceUnavailable, "gateway not ready")
		return
	}

	// Verify session exists and is not expired.
	session, ok := m.sessions.Get(sessionID)
	if !ok {
		gatewayWriteError(w, http.StatusNotFound, "session not found")
		return
	}
	if time.Now().After(session.ExpiresAt) {
		m.sessions.Delete(sessionID)
		if m.proxies != nil {
			m.proxies.RemoveProxy(sessionID)
		}
		gatewayWriteError(w, http.StatusGone, "session expired")
		return
	}

	if m.proxies == nil {
		gatewayWriteError(w, http.StatusServiceUnavailable, "proxy manager not available")
		return
	}

	// Strip the gateway proxy prefix so the target device sees relative paths.
	// The incoming path includes /proxy/s/{session_id}/{path...} relative to
	// the plugin mount. We need to forward only the {path...} portion.
	remainingPath := r.PathValue("path")
	r.URL.Path = "/" + remainingPath
	r.URL.RawPath = ""

	// Clear the RequestURI to avoid conflicts with the modified URL.
	r.RequestURI = ""

	if err := m.proxies.ServeProxy(sessionID, w, r); err != nil {
		gatewayWriteError(w, http.StatusNotFound, "proxy session not found")
		return
	}
}

// generateSessionID returns a unique session identifier.
func generateSessionID() string {
	return fmt.Sprintf("gw-%d", time.Now().UnixNano())
}

// --- Helpers ---

// gatewayWriteJSON writes a JSON response with the given status code.
func gatewayWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// gatewayWriteError writes a problem+json error response.
func gatewayWriteError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   fmt.Sprintf("https://subnetree.com/problems/%s", http.StatusText(status)),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

// gatewayParseLimit extracts a limit query parameter with a default value.
func gatewayParseLimit(r *http.Request, defaultLimit int) int {
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 1000 {
			return n
		}
	}
	return defaultLimit
}
