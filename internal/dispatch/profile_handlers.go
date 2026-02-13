package dispatch

import (
	"net/http"

	"go.uber.org/zap"
)

// handleGetHardwareProfile returns the hardware profile for an agent.
//
//	@Summary		Get agent hardware profile
//	@Description	Returns the stored hardware profile for a specific agent.
//	@Tags			dispatch
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	object
//	@Failure		404	{object}	object
//	@Router			/dispatch/agents/{id}/hardware [get]
func (m *Module) handleGetHardwareProfile(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		dispatchWriteError(w, http.StatusServiceUnavailable, "dispatch store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		dispatchWriteError(w, http.StatusBadRequest, "agent id is required")
		return
	}

	hw, err := m.store.GetHardwareProfile(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get hardware profile", zap.String("agent_id", id), zap.Error(err))
		dispatchWriteError(w, http.StatusInternalServerError, "failed to get hardware profile")
		return
	}
	if hw == nil {
		dispatchWriteError(w, http.StatusNotFound, "hardware profile not found")
		return
	}
	dispatchWriteJSON(w, http.StatusOK, hw)
}

// handleGetSoftwareInventory returns the software inventory for an agent.
//
//	@Summary		Get agent software inventory
//	@Description	Returns the stored software inventory for a specific agent.
//	@Tags			dispatch
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	object
//	@Failure		404	{object}	object
//	@Router			/dispatch/agents/{id}/software [get]
func (m *Module) handleGetSoftwareInventory(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		dispatchWriteError(w, http.StatusServiceUnavailable, "dispatch store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		dispatchWriteError(w, http.StatusBadRequest, "agent id is required")
		return
	}

	sw, err := m.store.GetSoftwareInventory(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get software inventory", zap.String("agent_id", id), zap.Error(err))
		dispatchWriteError(w, http.StatusInternalServerError, "failed to get software inventory")
		return
	}
	if sw == nil {
		dispatchWriteError(w, http.StatusNotFound, "software inventory not found")
		return
	}
	dispatchWriteJSON(w, http.StatusOK, sw)
}

// handleGetServices returns the services list for an agent.
//
//	@Summary		Get agent services
//	@Description	Returns the stored services list for a specific agent.
//	@Tags			dispatch
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{array}		object
//	@Failure		404	{object}	object
//	@Router			/dispatch/agents/{id}/services [get]
func (m *Module) handleGetServices(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		dispatchWriteError(w, http.StatusServiceUnavailable, "dispatch store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		dispatchWriteError(w, http.StatusBadRequest, "agent id is required")
		return
	}

	services, err := m.store.GetServices(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get services", zap.String("agent_id", id), zap.Error(err))
		dispatchWriteError(w, http.StatusInternalServerError, "failed to get services")
		return
	}
	if services == nil {
		dispatchWriteError(w, http.StatusNotFound, "services not found")
		return
	}
	dispatchWriteJSON(w, http.StatusOK, services)
}
