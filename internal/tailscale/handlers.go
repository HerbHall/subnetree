package tailscale

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/status", Handler: m.handleStatus},
		{Method: "POST", Path: "/sync", Handler: m.handleSync},
	}
}

// statusResponse is the JSON body returned by GET /status.
type statusResponse struct {
	Enabled        bool        `json:"enabled"`
	LastSyncTime   *time.Time  `json:"last_sync_time,omitempty"`
	LastSyncResult *SyncResult `json:"last_sync_result,omitempty"`
	Error          string      `json:"error,omitempty"`
}

// handleStatus returns the current Tailscale integration status.
//
//	@Summary		Get Tailscale integration status
//	@Description	Returns whether the integration is enabled, last sync time and result.
//	@Tags			tailscale
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	statusResponse
//	@Router			/tailscale/status [get]
func (m *Module) handleStatus(w http.ResponseWriter, _ *http.Request) {
	m.mu.RLock()
	resp := statusResponse{
		Enabled:        m.cfg.Enabled,
		LastSyncResult: m.lastSyncResult,
	}
	if !m.lastSyncTime.IsZero() {
		t := m.lastSyncTime
		resp.LastSyncTime = &t
	}
	if m.lastSyncErr != nil {
		resp.Error = m.lastSyncErr.Error()
	}
	m.mu.RUnlock()

	tsWriteJSON(w, http.StatusOK, resp)
}

// handleSync triggers an immediate device sync.
//
//	@Summary		Trigger Tailscale sync
//	@Description	Immediately syncs devices from the configured Tailscale tailnet.
//	@Tags			tailscale
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	SyncResult
//	@Failure		500	{object}	map[string]any
//	@Router			/tailscale/sync [post]
func (m *Module) handleSync(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	client := m.client
	syncer := m.syncer
	m.mu.RUnlock()

	if client == nil || syncer == nil {
		tsWriteError(w, http.StatusServiceUnavailable, "tailscale integration is not configured")
		return
	}

	result, err := syncer.Sync(r.Context(), client)
	if err != nil {
		m.mu.Lock()
		m.lastSyncErr = err
		m.lastSyncTime = time.Now().UTC()
		m.mu.Unlock()
		tsWriteError(w, http.StatusInternalServerError, "sync failed: "+err.Error())
		return
	}

	m.mu.Lock()
	m.lastSyncResult = result
	m.lastSyncErr = nil
	m.lastSyncTime = time.Now().UTC()
	m.mu.Unlock()

	tsWriteJSON(w, http.StatusOK, result)
}

// tsWriteJSON writes a JSON response with the given status code.
func tsWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// tsWriteError writes an RFC 7807 problem detail response.
func tsWriteError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "about:blank",
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
