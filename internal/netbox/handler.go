package netbox

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an RFC 7807 problem detail response.
func writeError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

// StatusResponse is the response for the GET /status endpoint.
type StatusResponse struct {
	Configured bool   `json:"configured"`
	URL        string `json:"url,omitempty"`
	TagName    string `json:"tag_name"`
	DryRun     bool   `json:"dry_run"`
}

// handleSync triggers a full sync of all SubNetree devices to NetBox.
//
//	@Summary		Sync all devices to NetBox
//	@Description	Triggers a full sync of all SubNetree devices to a NetBox CMDB instance.
//	@Tags			netbox
//	@Produce		json
//	@Security		BearerAuth
//	@Param			dry_run	query		bool	false	"Dry run mode (no changes made)"
//	@Success		200		{object}	SyncResult
//	@Failure		500		{object}	map[string]any
//	@Failure		503		{object}	map[string]any
//	@Router			/netbox/sync [post]
func (m *Module) handleSync(w http.ResponseWriter, r *http.Request) {
	if m.client == nil {
		writeError(w, http.StatusServiceUnavailable, "netbox module not configured (set plugins.netbox.url and plugins.netbox.token)")
		return
	}
	if m.deviceReader == nil {
		writeError(w, http.StatusServiceUnavailable, "device reader not available")
		return
	}

	dryRun := r.URL.Query().Get("dry_run") == "true" || m.cfg.DryRun

	engine := newSyncEngine(m.client, m.deviceReader, m.cfg, m.logger)
	result, err := engine.SyncAll(r.Context(), dryRun)
	if err != nil {
		m.logger.Error("sync failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleSyncDevice triggers a sync for a single device.
//
//	@Summary		Sync a single device to NetBox
//	@Description	Syncs one SubNetree device to NetBox by device ID.
//	@Tags			netbox
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"Device ID"
//	@Param			dry_run	query		bool	false	"Dry run mode"
//	@Success		200		{object}	SyncResult
//	@Failure		400		{object}	map[string]any
//	@Failure		500		{object}	map[string]any
//	@Failure		503		{object}	map[string]any
//	@Router			/netbox/sync/{id} [post]
func (m *Module) handleSyncDevice(w http.ResponseWriter, r *http.Request) {
	if m.client == nil {
		writeError(w, http.StatusServiceUnavailable, "netbox module not configured")
		return
	}
	if m.deviceReader == nil {
		writeError(w, http.StatusServiceUnavailable, "device reader not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device id is required")
		return
	}

	dryRun := r.URL.Query().Get("dry_run") == "true" || m.cfg.DryRun

	engine := newSyncEngine(m.client, m.deviceReader, m.cfg, m.logger)
	result, err := engine.SyncDevice(r.Context(), id, dryRun)
	if err != nil {
		m.logger.Error("device sync failed", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleStatus returns the current NetBox integration configuration status.
//
//	@Summary		Get NetBox integration status
//	@Description	Returns whether the NetBox integration is configured and connection details.
//	@Tags			netbox
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	StatusResponse
//	@Router			/netbox/status [get]
func (m *Module) handleStatus(w http.ResponseWriter, _ *http.Request) {
	resp := StatusResponse{
		Configured: m.client != nil,
		TagName:    m.cfg.TagName,
		DryRun:     m.cfg.DryRun,
	}
	if m.client != nil {
		resp.URL = m.cfg.URL
	}
	writeJSON(w, http.StatusOK, resp)
}
