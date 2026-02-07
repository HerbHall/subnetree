package insight

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/HerbHall/subnetree/pkg/analytics"
	"github.com/HerbHall/subnetree/pkg/plugin"
)

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/anomalies", Handler: m.handleListAnomalies},
		{Method: "GET", Path: "/anomalies/{device_id}", Handler: m.handleDeviceAnomalies},
		{Method: "GET", Path: "/forecasts/{device_id}", Handler: m.handleDeviceForecasts},
		{Method: "GET", Path: "/correlations", Handler: m.handleListCorrelations},
		{Method: "GET", Path: "/baselines/{device_id}", Handler: m.handleDeviceBaselines},
		{Method: "POST", Path: "/query", Handler: m.handleNLQuery},
	}
}

// handleListAnomalies returns all detected anomalies.
//
//	@Summary		List anomalies
//	@Description	Returns all detected anomalies across all devices.
//	@Tags			insight
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit query int false "Maximum results" default(50)
//	@Success		200 {array} analytics.Anomaly
//	@Failure		500 {object} map[string]any
//	@Router			/insight/anomalies [get]
func (m *Module) handleListAnomalies(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 50)
	anomalies, err := m.store.ListAnomalies(r.Context(), "", limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list anomalies")
		return
	}
	if anomalies == nil {
		anomalies = []analytics.Anomaly{}
	}
	writeJSON(w, http.StatusOK, anomalies)
}

// handleDeviceAnomalies returns anomalies for a specific device.
//
//	@Summary		Device anomalies
//	@Description	Returns anomalies for a specific device.
//	@Tags			insight
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Param			limit query int false "Maximum results" default(50)
//	@Success		200 {array} analytics.Anomaly
//	@Failure		500 {object} map[string]any
//	@Router			/insight/anomalies/{device_id} [get]
func (m *Module) handleDeviceAnomalies(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	limit := parseLimit(r, 50)
	anomalies, err := m.store.ListAnomalies(r.Context(), deviceID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list anomalies")
		return
	}
	if anomalies == nil {
		anomalies = []analytics.Anomaly{}
	}
	writeJSON(w, http.StatusOK, anomalies)
}

// handleDeviceForecasts returns capacity forecasts for a device.
//
//	@Summary		Device forecasts
//	@Description	Returns capacity forecasts for a specific device.
//	@Tags			insight
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Success		200 {array} analytics.Forecast
//	@Failure		500 {object} map[string]any
//	@Router			/insight/forecasts/{device_id} [get]
func (m *Module) handleDeviceForecasts(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	forecasts, err := m.store.GetForecasts(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get forecasts")
		return
	}
	if forecasts == nil {
		forecasts = []analytics.Forecast{}
	}
	writeJSON(w, http.StatusOK, forecasts)
}

// handleListCorrelations returns active alert correlation groups.
//
//	@Summary		List correlations
//	@Description	Returns active alert correlation groups.
//	@Tags			insight
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200 {array} analytics.AlertGroup
//	@Failure		500 {object} map[string]any
//	@Router			/insight/correlations [get]
func (m *Module) handleListCorrelations(w http.ResponseWriter, r *http.Request) {
	groups, err := m.store.ListActiveCorrelations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list correlations")
		return
	}
	if groups == nil {
		groups = []analytics.AlertGroup{}
	}
	writeJSON(w, http.StatusOK, groups)
}

// handleDeviceBaselines returns learned baselines for a device.
//
//	@Summary		Device baselines
//	@Description	Returns learned baselines for a specific device.
//	@Tags			insight
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Success		200 {array} analytics.Baseline
//	@Failure		500 {object} map[string]any
//	@Router			/insight/baselines/{device_id} [get]
func (m *Module) handleDeviceBaselines(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	baselines, err := m.store.GetBaselines(r.Context(), deviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get baselines")
		return
	}
	if baselines == nil {
		baselines = []analytics.Baseline{}
	}
	writeJSON(w, http.StatusOK, baselines)
}

// handleNLQuery processes a natural language query.
//
//	@Summary		Natural language query
//	@Description	Translate a natural language question into a structured query and return results.
//	@Tags			insight
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request body analytics.NLQueryRequest true "Query"
//	@Success		200 {object} analytics.NLQueryResponse
//	@Failure		400 {object} map[string]any
//	@Failure		503 {object} map[string]any
//	@Router			/insight/query [post]
func (m *Module) handleNLQuery(w http.ResponseWriter, r *http.Request) {
	var req analytics.NLQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	// NL query requires LLM provider -- stub returns 503 until PR 3 wires it up.
	writeError(w, http.StatusServiceUnavailable,
		"natural language queries require the LLM plugin; this feature is coming soon")
}

// -- helpers --

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

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

func parseLimit(r *http.Request, defaultLimit int) int {
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 1000 {
			return n
		}
	}
	return defaultLimit
}
