package pulse

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/checks", Handler: m.handleListChecks},
		{Method: "GET", Path: "/checks/{device_id}", Handler: m.handleDeviceChecks},
		{Method: "GET", Path: "/results/{device_id}", Handler: m.handleDeviceResults},
		{Method: "GET", Path: "/alerts", Handler: m.handleListAlerts},
		{Method: "GET", Path: "/alerts/{device_id}", Handler: m.handleDeviceAlerts},
		{Method: "GET", Path: "/status/{device_id}", Handler: m.handleDeviceStatus},
	}
}

// handleListChecks returns all registered monitoring checks.
//
//	@Summary		List checks
//	@Description	Returns all enabled monitoring checks.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200 {array} Check
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks [get]
func (m *Module) handleListChecks(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	checks, err := m.store.ListEnabledChecks(r.Context())
	if err != nil {
		m.logger.Warn("failed to list checks", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list checks")
		return
	}
	if checks == nil {
		checks = []Check{}
	}
	pulseWriteJSON(w, http.StatusOK, checks)
}

// handleDeviceChecks returns checks for a specific device.
//
//	@Summary		Device checks
//	@Description	Returns monitoring checks for a specific device.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Success		200 {object} Check
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{device_id} [get]
func (m *Module) handleDeviceChecks(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	check, err := m.store.GetCheckByDeviceID(r.Context(), deviceID)
	if err != nil {
		m.logger.Warn("failed to get device check", zap.String("device_id", deviceID), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get check")
		return
	}
	if check == nil {
		pulseWriteError(w, http.StatusNotFound, "no check found for device")
		return
	}
	pulseWriteJSON(w, http.StatusOK, check)
}

// handleDeviceResults returns recent check results for a device.
//
//	@Summary		Device results
//	@Description	Returns recent check results for a specific device.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Param			limit query int false "Maximum results" default(100)
//	@Success		200 {array} CheckResult
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/results/{device_id} [get]
func (m *Module) handleDeviceResults(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	limit := pulseParseLimit(r, 100)
	results, err := m.store.ListResults(r.Context(), deviceID, limit)
	if err != nil {
		m.logger.Warn("failed to list results", zap.String("device_id", deviceID), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list results")
		return
	}
	if results == nil {
		results = []CheckResult{}
	}
	pulseWriteJSON(w, http.StatusOK, results)
}

// handleListAlerts returns all active (unresolved) alerts.
//
//	@Summary		List alerts
//	@Description	Returns all active (unresolved) monitoring alerts.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200 {array} Alert
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts [get]
func (m *Module) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	alerts, err := m.store.ListActiveAlerts(r.Context(), "")
	if err != nil {
		m.logger.Warn("failed to list alerts", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list alerts")
		return
	}
	if alerts == nil {
		alerts = []Alert{}
	}
	pulseWriteJSON(w, http.StatusOK, alerts)
}

// handleDeviceAlerts returns active alerts for a specific device.
//
//	@Summary		Device alerts
//	@Description	Returns active (unresolved) alerts for a specific device.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Success		200 {array} Alert
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts/{device_id} [get]
func (m *Module) handleDeviceAlerts(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	alerts, err := m.store.ListActiveAlerts(r.Context(), deviceID)
	if err != nil {
		m.logger.Warn("failed to list device alerts", zap.String("device_id", deviceID), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list alerts")
		return
	}
	if alerts == nil {
		alerts = []Alert{}
	}
	pulseWriteJSON(w, http.StatusOK, alerts)
}

// handleDeviceStatus returns composite monitoring status for a device.
//
//	@Summary		Monitoring status
//	@Description	Returns composite health status for a specific device, including latest check result and active alerts.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Success		200 {object} github_com_HerbHall_subnetree_pkg_roles.MonitorStatus
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/status/{device_id} [get]
func (m *Module) handleDeviceStatus(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	status, err := m.Status(r.Context(), deviceID)
	if err != nil {
		m.logger.Warn("failed to get device status", zap.String("device_id", deviceID), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get status")
		return
	}
	pulseWriteJSON(w, http.StatusOK, status)
}

// -- helpers --

func pulseWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func pulseWriteError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

func pulseParseLimit(r *http.Request, defaultLimit int) int {
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 1000 {
			return n
		}
	}
	return defaultLimit
}
