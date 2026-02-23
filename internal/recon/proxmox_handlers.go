package recon

import (
	"encoding/json"
	"net/http"

	// models is imported for swagger annotation resolution (models.APIProblem).
	_ "github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// ProxmoxSyncRequest is the request body for POST /recon/proxmox/sync.
type ProxmoxSyncRequest struct {
	BaseURL      string `json:"base_url" example:"https://pve:8006"`
	TokenID      string `json:"token_id" example:"user@pam!token"`       //nolint:gosec // G101: field name, not a credential
	TokenSecret  string `json:"token_secret" example:"uuid-secret-here"` //nolint:gosec // G101: field name, not a credential
	HostDeviceID string `json:"host_device_id" example:"device-uuid"`
}

// handleProxmoxSync triggers a Proxmox VM/container sync.
//
//	@Summary		Trigger Proxmox sync
//	@Description	Connects to a Proxmox VE host, enumerates VMs and containers, and syncs them as child devices with resource snapshots.
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		ProxmoxSyncRequest	true	"Proxmox connection details"
//	@Success		200		{object}	ProxmoxSyncResult
//	@Failure		400		{object}	models.APIProblem
//	@Failure		500		{object}	models.APIProblem
//	@Router			/recon/proxmox/sync [post]
func (m *Module) handleProxmoxSync(w http.ResponseWriter, r *http.Request) {
	var req ProxmoxSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.BaseURL == "" || req.TokenID == "" || req.TokenSecret == "" || req.HostDeviceID == "" {
		writeError(w, http.StatusBadRequest, "base_url, token_id, token_secret, and host_device_id are required")
		return
	}

	collector := NewProxmoxCollector(req.BaseURL, req.TokenID, req.TokenSecret, m.logger.Named("proxmox"))
	result, err := m.proxmoxSyncer.Sync(r.Context(), collector, req.HostDeviceID)
	if err != nil {
		m.logger.Error("proxmox sync failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "proxmox sync failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleListProxmoxVMs returns all Proxmox VMs/containers with resource snapshots.
//
//	@Summary		List Proxmox VMs and containers
//	@Description	Returns all Proxmox-managed VMs and containers with their latest resource snapshots.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			parent_id	query		string	false	"Filter by parent host device ID"
//	@Param			status		query		string	false	"Filter by status (online/offline)"
//	@Success		200			{array}		ProxmoxResource
//	@Failure		500			{object}	models.APIProblem
//	@Router			/recon/proxmox/vms [get]
func (m *Module) handleListProxmoxVMs(w http.ResponseWriter, r *http.Request) {
	parentID := r.URL.Query().Get("parent_id")
	statusFilter := r.URL.Query().Get("status")

	resources, err := m.store.ListProxmoxResources(r.Context(), parentID, statusFilter)
	if err != nil {
		m.logger.Error("failed to list proxmox resources", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to list proxmox resources")
		return
	}
	if resources == nil {
		resources = []ProxmoxResource{}
	}
	writeJSON(w, http.StatusOK, resources)
}

// handleGetProxmoxVMResources returns the resource snapshot for a specific VM/container.
//
//	@Summary		Get VM/container resources
//	@Description	Returns the latest resource snapshot for a specific Proxmox VM or container.
//	@Tags			recon
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Device ID of the VM/container"
//	@Success		200	{object}	ProxmoxResource
//	@Failure		400	{object}	models.APIProblem
//	@Failure		404	{object}	models.APIProblem
//	@Failure		500	{object}	models.APIProblem
//	@Router			/recon/proxmox/vms/{id}/resources [get]
func (m *Module) handleGetProxmoxVMResources(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	resource, err := m.store.GetProxmoxResource(r.Context(), id)
	if err != nil {
		m.logger.Error("failed to get proxmox resource", zap.String("device_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get proxmox resource")
		return
	}
	if resource == nil {
		writeError(w, http.StatusNotFound, "no resource snapshot found for device")
		return
	}
	writeJSON(w, http.StatusOK, resource)
}
