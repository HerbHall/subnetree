package svcmap

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// Handler provides HTTP endpoints for the service mapping module.
type Handler struct {
	store    *Store
	hwSource HardwareSource
	logger   *zap.Logger
}

// NewHandler creates a new svcmap Handler.
func NewHandler(store *Store, hwSource HardwareSource, logger *zap.Logger) *Handler {
	return &Handler{store: store, hwSource: hwSource, logger: logger}
}

// RegisterRoutes registers svcmap HTTP routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/svcmap/services", h.handleListServices)
	mux.HandleFunc("GET /api/v1/svcmap/services/{id}", h.handleGetService)
	mux.HandleFunc("PATCH /api/v1/svcmap/services/{id}", h.handleUpdateDesiredState)
	mux.HandleFunc("GET /api/v1/svcmap/devices/{device_id}/services", h.handleDeviceServices)
	mux.HandleFunc("GET /api/v1/svcmap/devices/{device_id}/utilization", h.handleDeviceUtilization)
	mux.HandleFunc("GET /api/v1/svcmap/utilization/fleet", h.handleFleetSummary)
}

// handleListServices returns all services, optionally filtered by query params.
//
//	@Summary		List services
//	@Description	Returns all tracked services with optional filtering.
//	@Tags			svcmap
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id		query	string	false	"Filter by device ID"
//	@Param			service_type	query	string	false	"Filter by service type"
//	@Param			status			query	string	false	"Filter by status"
//	@Success		200	{array}		models.Service
//	@Router			/svcmap/services [get]
func (h *Handler) handleListServices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := ServiceFilter{
		DeviceID:    q.Get("device_id"),
		ServiceType: q.Get("service_type"),
		Status:      q.Get("status"),
	}

	services, err := h.store.ListServicesFiltered(r.Context(), filter)
	if err != nil {
		h.logger.Warn("failed to list services", zap.Error(err))
		svcmapWriteError(w, http.StatusInternalServerError, "failed to list services")
		return
	}
	if services == nil {
		services = []models.Service{}
	}
	svcmapWriteJSON(w, http.StatusOK, services)
}

// handleGetService returns a single service by ID.
//
//	@Summary		Get service
//	@Description	Returns a single tracked service by ID.
//	@Tags			svcmap
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Service ID"
//	@Success		200	{object}	models.Service
//	@Failure		404	{object}	models.APIProblem
//	@Router			/svcmap/services/{id} [get]
func (h *Handler) handleGetService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		svcmapWriteError(w, http.StatusBadRequest, "service id is required")
		return
	}

	svc, err := h.store.GetService(r.Context(), id)
	if err != nil {
		h.logger.Warn("failed to get service", zap.String("id", id), zap.Error(err))
		svcmapWriteError(w, http.StatusInternalServerError, "failed to get service")
		return
	}
	if svc == nil {
		svcmapWriteError(w, http.StatusNotFound, "service not found")
		return
	}
	svcmapWriteJSON(w, http.StatusOK, svc)
}

// desiredStateRequest is the request body for updating desired state.
type desiredStateRequest struct {
	DesiredState models.DesiredState `json:"desired_state"`
}

// handleUpdateDesiredState updates the desired state annotation for a service.
//
//	@Summary		Update desired state
//	@Description	Updates the desired operational state for a service.
//	@Tags			svcmap
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Service ID"
//	@Param			body	body		desiredStateRequest	true	"Desired state"
//	@Success		200		{object}	models.Service
//	@Failure		400		{object}	models.APIProblem
//	@Failure		404		{object}	models.APIProblem
//	@Router			/svcmap/services/{id} [patch]
func (h *Handler) handleUpdateDesiredState(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		svcmapWriteError(w, http.StatusBadRequest, "service id is required")
		return
	}

	var req desiredStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		svcmapWriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	switch req.DesiredState {
	case models.DesiredStateShouldRun, models.DesiredStateShouldStop, models.DesiredStateMonitoringOnly:
		// valid
	default:
		svcmapWriteError(w, http.StatusBadRequest, "invalid desired_state value")
		return
	}

	err := h.store.UpdateDesiredState(r.Context(), id, req.DesiredState)
	if err != nil {
		if err == sql.ErrNoRows {
			svcmapWriteError(w, http.StatusNotFound, "service not found")
			return
		}
		h.logger.Warn("failed to update desired state", zap.String("id", id), zap.Error(err))
		svcmapWriteError(w, http.StatusInternalServerError, "failed to update desired state")
		return
	}

	svc, err := h.store.GetService(r.Context(), id)
	if err != nil {
		h.logger.Warn("failed to fetch updated service", zap.String("id", id), zap.Error(err))
		svcmapWriteError(w, http.StatusInternalServerError, "failed to fetch updated service")
		return
	}
	svcmapWriteJSON(w, http.StatusOK, svc)
}

// handleDeviceServices returns all services for a specific device.
//
//	@Summary		Device services
//	@Description	Returns all tracked services for a specific device.
//	@Tags			svcmap
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{array}		models.Service
//	@Router			/svcmap/devices/{device_id}/services [get]
func (h *Handler) handleDeviceServices(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		svcmapWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	services, err := h.store.ListServicesByDevice(r.Context(), deviceID)
	if err != nil {
		h.logger.Warn("failed to list device services", zap.String("device_id", deviceID), zap.Error(err))
		svcmapWriteError(w, http.StatusInternalServerError, "failed to list device services")
		return
	}
	if services == nil {
		services = []models.Service{}
	}
	svcmapWriteJSON(w, http.StatusOK, services)
}

// handleDeviceUtilization returns the utilization summary for a device.
//
//	@Summary		Device utilization
//	@Description	Returns resource utilization summary and grade for a device.
//	@Tags			svcmap
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{object}	models.UtilizationSummary
//	@Router			/svcmap/devices/{device_id}/utilization [get]
func (h *Handler) handleDeviceUtilization(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		svcmapWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	device := DeviceInfo{
		DeviceID: deviceID,
		Hostname: r.URL.Query().Get("hostname"),
		AgentID:  r.URL.Query().Get("agent_id"),
	}

	summary, err := ComputeDeviceUtilization(r.Context(), h.store, device, h.hwSource)
	if err != nil {
		h.logger.Warn("failed to compute utilization", zap.String("device_id", deviceID), zap.Error(err))
		svcmapWriteError(w, http.StatusInternalServerError, "failed to compute utilization")
		return
	}
	svcmapWriteJSON(w, http.StatusOK, summary)
}

// handleFleetSummary returns fleet-wide utilization summary.
//
//	@Summary		Fleet summary
//	@Description	Returns fleet-wide utilization summary with grade distribution.
//	@Tags			svcmap
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	models.FleetSummary
//	@Router			/svcmap/utilization/fleet [get]
func (h *Handler) handleFleetSummary(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetUtilizationSummaries(r.Context())
	if err != nil {
		h.logger.Warn("failed to get utilization summaries", zap.Error(err))
		svcmapWriteError(w, http.StatusInternalServerError, "failed to compute fleet summary")
		return
	}

	summaries := make([]models.UtilizationSummary, 0, len(stats))
	for i := range stats {
		summaries = append(summaries, models.UtilizationSummary{
			DeviceID:     stats[i].DeviceID,
			CPUPercent:   stats[i].TotalCPU,
			ServiceCount: stats[i].ServiceCount,
			Grade:        ComputeGrade(stats[i].TotalCPU, 0, 0),
		})
	}

	fleet := ComputeFleetSummary(summaries)
	svcmapWriteJSON(w, http.StatusOK, fleet)
}

func svcmapWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func svcmapWriteError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
