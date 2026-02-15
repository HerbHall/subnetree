package pulse

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// createCheckRequest is the JSON body for POST /checks.
type createCheckRequest struct {
	DeviceID        string `json:"device_id"`
	CheckType       string `json:"check_type"`
	Target          string `json:"target"`
	IntervalSeconds int    `json:"interval_seconds"`
}

// updateCheckRequest is the JSON body for PUT /checks/{id}.
type updateCheckRequest struct {
	Target          string `json:"target,omitempty"`
	CheckType       string `json:"check_type,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

// createNotificationRequest is the JSON body for POST /notifications.
type createNotificationRequest struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Config string `json:"config"`
}

// updateNotificationRequest is the JSON body for PUT /notifications/{id}.
type updateNotificationRequest struct {
	Name    string `json:"name,omitempty"`
	Config  string `json:"config,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
}

// createMaintWindowRequest is the JSON body for POST /maintenance-windows.
type createMaintWindowRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	StartTime   string   `json:"start_time"`
	EndTime     string   `json:"end_time"`
	Recurrence  string   `json:"recurrence"`
	DeviceIDs   []string `json:"device_ids"`
}

// updateMaintWindowRequest is the JSON body for PUT /maintenance-windows/{id}.
type updateMaintWindowRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	StartTime   string   `json:"start_time,omitempty"`
	EndTime     string   `json:"end_time,omitempty"`
	Recurrence  string   `json:"recurrence,omitempty"`
	DeviceIDs   []string `json:"device_ids,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/checks", Handler: m.handleListChecks},
		{Method: "POST", Path: "/checks", Handler: m.handleCreateCheck},
		{Method: "GET", Path: "/checks/{device_id}", Handler: m.handleDeviceChecks},
		{Method: "PUT", Path: "/checks/{id}", Handler: m.handleUpdateCheck},
		{Method: "DELETE", Path: "/checks/{id}", Handler: m.handleDeleteCheck},
		{Method: "PATCH", Path: "/checks/{id}/toggle", Handler: m.handleToggleCheck},
		{Method: "GET", Path: "/checks/{check_id}/dependencies", Handler: m.handleListCheckDependencies},
		{Method: "POST", Path: "/checks/{check_id}/dependencies", Handler: m.handleAddCheckDependency},
		{Method: "DELETE", Path: "/checks/{check_id}/dependencies/{device_id}", Handler: m.handleRemoveCheckDependency},
		{Method: "GET", Path: "/results/{device_id}", Handler: m.handleDeviceResults},
		{Method: "GET", Path: "/metrics/{device_id}", Handler: m.handleDeviceMetrics},
		{Method: "GET", Path: "/alerts", Handler: m.handleListAlerts},
		{Method: "GET", Path: "/alerts/{id}", Handler: m.handleGetAlert},
		{Method: "POST", Path: "/alerts/{id}/acknowledge", Handler: m.handleAcknowledgeAlert},
		{Method: "POST", Path: "/alerts/{id}/resolve", Handler: m.handleResolveAlert},
		{Method: "GET", Path: "/status/{device_id}", Handler: m.handleDeviceStatus},
		{Method: "GET", Path: "/notifications", Handler: m.handleListNotifications},
		{Method: "GET", Path: "/notifications/{id}", Handler: m.handleGetNotification},
		{Method: "POST", Path: "/notifications", Handler: m.handleCreateNotification},
		{Method: "PUT", Path: "/notifications/{id}", Handler: m.handleUpdateNotification},
		{Method: "DELETE", Path: "/notifications/{id}", Handler: m.handleDeleteNotification},
		{Method: "POST", Path: "/notifications/{id}/test", Handler: m.handleTestNotification},
		{Method: "GET", Path: "/maintenance-windows", Handler: m.handleListMaintWindows},
		{Method: "POST", Path: "/maintenance-windows", Handler: m.handleCreateMaintWindow},
		{Method: "GET", Path: "/maintenance-windows/{id}", Handler: m.handleGetMaintWindow},
		{Method: "PUT", Path: "/maintenance-windows/{id}", Handler: m.handleUpdateMaintWindow},
		{Method: "DELETE", Path: "/maintenance-windows/{id}", Handler: m.handleDeleteMaintWindow},
	}
}

// handleListChecks returns all registered monitoring checks.
//
//	@Summary		List checks
//	@Description	Returns all monitoring checks (enabled and disabled).
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
	checks, err := m.store.ListAllChecks(r.Context())
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

// handleCreateCheck creates a new monitoring check.
//
//	@Summary		Create check
//	@Description	Creates a new monitoring check for a device.
//	@Tags			pulse
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body body createCheckRequest true "Check definition"
//	@Success		201 {object} Check
//	@Failure		400 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks [post]
func (m *Module) handleCreateCheck(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	var req createCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pulseWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.DeviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	// Validate check_type.
	switch req.CheckType {
	case "icmp", "tcp", "http":
		// valid
	default:
		pulseWriteError(w, http.StatusBadRequest, "check_type must be icmp, tcp, or http")
		return
	}

	// Validate target based on check type.
	if req.Target == "" {
		pulseWriteError(w, http.StatusBadRequest, "target is required")
		return
	}
	if err := validateTarget(req.CheckType, req.Target); err != nil {
		pulseWriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = 30
	}

	now := time.Now().UTC()
	check := &Check{
		ID:              fmt.Sprintf("pulse-%s-%s-%d", req.DeviceID, req.CheckType, now.UnixMilli()),
		DeviceID:        req.DeviceID,
		CheckType:       req.CheckType,
		Target:          req.Target,
		IntervalSeconds: req.IntervalSeconds,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := m.store.InsertCheck(r.Context(), check); err != nil {
		m.logger.Warn("failed to create check", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to create check")
		return
	}

	pulseWriteJSON(w, http.StatusCreated, check)
}

// handleUpdateCheck updates an existing monitoring check.
//
//	@Summary		Update check
//	@Description	Updates fields on an existing monitoring check.
//	@Tags			pulse
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Check ID"
//	@Param			body body updateCheckRequest true "Fields to update"
//	@Success		200 {object} Check
//	@Failure		400 {object} map[string]any
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{id} [put]
func (m *Module) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	existing, err := m.store.GetCheck(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get check for update", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get check")
		return
	}
	if existing == nil {
		pulseWriteError(w, http.StatusNotFound, "check not found")
		return
	}

	var req updateCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pulseWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.CheckType != "" {
		switch req.CheckType {
		case "icmp", "tcp", "http":
			existing.CheckType = req.CheckType
		default:
			pulseWriteError(w, http.StatusBadRequest, "check_type must be icmp, tcp, or http")
			return
		}
	}
	if req.Target != "" {
		if err := validateTarget(existing.CheckType, req.Target); err != nil {
			pulseWriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		existing.Target = req.Target
	}
	if req.IntervalSeconds > 0 {
		existing.IntervalSeconds = req.IntervalSeconds
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := m.store.UpdateCheck(r.Context(), existing); err != nil {
		m.logger.Warn("failed to update check", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to update check")
		return
	}

	pulseWriteJSON(w, http.StatusOK, existing)
}

// handleDeleteCheck deletes a monitoring check and its results.
//
//	@Summary		Delete check
//	@Description	Deletes a monitoring check and cascade-deletes its results.
//	@Tags			pulse
//	@Security		BearerAuth
//	@Param			id path string true "Check ID"
//	@Success		204
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{id} [delete]
func (m *Module) handleDeleteCheck(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := m.store.DeleteCheck(r.Context(), id); err != nil {
		m.logger.Warn("failed to delete check", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to delete check")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleToggleCheck toggles the enabled state of a check.
//
//	@Summary		Toggle check
//	@Description	Toggles the enabled/disabled state of a monitoring check.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Check ID"
//	@Success		200 {object} Check
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{id}/toggle [patch]
func (m *Module) handleToggleCheck(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	existing, err := m.store.GetCheck(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get check for toggle", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get check")
		return
	}
	if existing == nil {
		pulseWriteError(w, http.StatusNotFound, "check not found")
		return
	}

	newEnabled := !existing.Enabled
	if err := m.store.UpdateCheckEnabled(r.Context(), id, newEnabled); err != nil {
		m.logger.Warn("failed to toggle check", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to toggle check")
		return
	}

	existing.Enabled = newEnabled
	pulseWriteJSON(w, http.StatusOK, existing)
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

// handleListAlerts returns alerts with optional filtering.
//
//	@Summary		List alerts
//	@Description	Returns monitoring alerts with optional filters.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id query string false "Filter by device ID"
//	@Param			severity query string false "Filter by severity (warning, critical)"
//	@Param			active query bool false "Only active (unresolved) alerts" default(true)
//	@Param			limit query int false "Maximum alerts" default(50)
//	@Success		200 {array} Alert
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts [get]
func (m *Module) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	filters := AlertFilters{
		DeviceID:   r.URL.Query().Get("device_id"),
		Severity:   r.URL.Query().Get("severity"),
		ActiveOnly: true,
		Limit:      pulseParseLimit(r, 50),
	}

	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		filters.ActiveOnly = activeStr != "false"
	}
	if suppressedStr := r.URL.Query().Get("suppressed"); suppressedStr != "" {
		v := suppressedStr == "true"
		filters.Suppressed = &v
	}

	alerts, err := m.store.ListAlerts(r.Context(), filters)
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

// handleGetAlert returns a single alert by ID.
//
//	@Summary		Get alert
//	@Description	Returns a single monitoring alert by ID.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Alert ID"
//	@Success		200 {object} Alert
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts/{id} [get]
func (m *Module) handleGetAlert(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	alert, err := m.store.GetAlert(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get alert", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get alert")
		return
	}
	if alert == nil {
		pulseWriteError(w, http.StatusNotFound, "alert not found")
		return
	}

	pulseWriteJSON(w, http.StatusOK, alert)
}

// handleAcknowledgeAlert acknowledges an alert.
//
//	@Summary		Acknowledge alert
//	@Description	Marks an alert as acknowledged.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Alert ID"
//	@Success		200 {object} Alert
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts/{id}/acknowledge [post]
func (m *Module) handleAcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := m.store.AcknowledgeAlert(r.Context(), id); err != nil {
		m.logger.Warn("failed to acknowledge alert", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to acknowledge alert")
		return
	}

	alert, err := m.store.GetAlert(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get alert after acknowledge", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get alert")
		return
	}
	if alert == nil {
		pulseWriteError(w, http.StatusNotFound, "alert not found")
		return
	}

	pulseWriteJSON(w, http.StatusOK, alert)
}

// handleResolveAlert resolves an alert.
//
//	@Summary		Resolve alert
//	@Description	Marks an alert as resolved.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Alert ID"
//	@Success		200 {object} Alert
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/alerts/{id}/resolve [post]
func (m *Module) handleResolveAlert(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	now := time.Now().UTC()
	if err := m.store.ResolveAlert(r.Context(), id, now); err != nil {
		m.logger.Warn("failed to resolve alert", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to resolve alert")
		return
	}

	alert, err := m.store.GetAlert(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get alert after resolve", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get alert")
		return
	}
	if alert == nil {
		pulseWriteError(w, http.StatusNotFound, "alert not found")
		return
	}

	pulseWriteJSON(w, http.StatusOK, alert)
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

// -- Metrics handler --

// handleDeviceMetrics returns time-series metrics for a device with automatic downsampling.
//
//	@Summary		Device metrics
//	@Description	Returns time-series metrics for a device with automatic downsampling.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id path string true "Device ID"
//	@Param			metric query string true "Metric name" Enums(latency, packet_loss, success_rate)
//	@Param			range query string false "Time range" Enums(1h, 6h, 24h, 7d, 30d) default(24h)
//	@Success		200 {object} MetricSeries
//	@Failure		400 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/metrics/{device_id} [get]
func (m *Module) handleDeviceMetrics(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}

	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	metric := r.URL.Query().Get("metric")
	if metric == "" {
		pulseWriteError(w, http.StatusBadRequest, "metric query parameter is required")
		return
	}
	if !validMetrics[metric] {
		pulseWriteError(w, http.StatusBadRequest, "metric must be latency, packet_loss, or success_rate")
		return
	}

	timeRange := r.URL.Query().Get("range")
	if timeRange == "" {
		timeRange = "24h"
	}
	if _, ok := validRanges[timeRange]; !ok {
		pulseWriteError(w, http.StatusBadRequest, "range must be 1h, 6h, 24h, 7d, or 30d")
		return
	}

	series, err := m.store.QueryMetrics(r.Context(), deviceID, metric, timeRange)
	if err != nil {
		m.logger.Warn("failed to query metrics",
			zap.String("device_id", deviceID),
			zap.String("metric", metric),
			zap.String("range", timeRange),
			zap.Error(err),
		)
		pulseWriteError(w, http.StatusInternalServerError, "failed to query metrics")
		return
	}

	pulseWriteJSON(w, http.StatusOK, series)
}

// -- Check dependency handlers --

// addDependencyRequest is the JSON body for POST /checks/{check_id}/dependencies.
type addDependencyRequest struct {
	DependsOnDeviceID string `json:"depends_on_device_id"`
}

// handleListCheckDependencies returns all dependencies for a check.
//
//	@Summary		List check dependencies
//	@Description	Returns all upstream device dependencies for a check.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			check_id path string true "Check ID"
//	@Success		200 {array} CheckDependency
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{check_id}/dependencies [get]
func (m *Module) handleListCheckDependencies(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	checkID := r.PathValue("check_id")
	if checkID == "" {
		pulseWriteError(w, http.StatusBadRequest, "check_id is required")
		return
	}
	deps, err := m.store.ListCheckDependencies(r.Context(), checkID)
	if err != nil {
		m.logger.Warn("failed to list check dependencies", zap.String("check_id", checkID), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list dependencies")
		return
	}
	if deps == nil {
		deps = []CheckDependency{}
	}
	pulseWriteJSON(w, http.StatusOK, deps)
}

// handleAddCheckDependency adds a dependency between a check and an upstream device.
//
//	@Summary		Add check dependency
//	@Description	Adds an upstream device dependency. When the upstream device has an active critical alert, this check's alerts are suppressed.
//	@Tags			pulse
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			check_id path string true "Check ID"
//	@Param			body body addDependencyRequest true "Dependency definition"
//	@Success		201 {object} map[string]any
//	@Failure		400 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{check_id}/dependencies [post]
func (m *Module) handleAddCheckDependency(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	checkID := r.PathValue("check_id")
	if checkID == "" {
		pulseWriteError(w, http.StatusBadRequest, "check_id is required")
		return
	}
	var req addDependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pulseWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.DependsOnDeviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "depends_on_device_id is required")
		return
	}
	if err := m.store.AddCheckDependency(r.Context(), checkID, req.DependsOnDeviceID); err != nil {
		m.logger.Warn("failed to add check dependency",
			zap.String("check_id", checkID),
			zap.String("depends_on", req.DependsOnDeviceID),
			zap.Error(err),
		)
		pulseWriteError(w, http.StatusInternalServerError, "failed to add dependency")
		return
	}
	pulseWriteJSON(w, http.StatusCreated, map[string]any{
		"check_id":             checkID,
		"depends_on_device_id": req.DependsOnDeviceID,
	})
}

// handleRemoveCheckDependency removes a dependency between a check and an upstream device.
//
//	@Summary		Remove check dependency
//	@Description	Removes an upstream device dependency from a check.
//	@Tags			pulse
//	@Security		BearerAuth
//	@Param			check_id path string true "Check ID"
//	@Param			device_id path string true "Upstream device ID"
//	@Success		204
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/checks/{check_id}/dependencies/{device_id} [delete]
func (m *Module) handleRemoveCheckDependency(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	checkID := r.PathValue("check_id")
	deviceID := r.PathValue("device_id")
	if checkID == "" || deviceID == "" {
		pulseWriteError(w, http.StatusBadRequest, "check_id and device_id are required")
		return
	}
	if err := m.store.RemoveCheckDependency(r.Context(), checkID, deviceID); err != nil {
		m.logger.Warn("failed to remove check dependency",
			zap.String("check_id", checkID),
			zap.String("device_id", deviceID),
			zap.Error(err),
		)
		pulseWriteError(w, http.StatusInternalServerError, "failed to remove dependency")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// -- Notification channel handlers --

// handleListNotifications returns all notification channels.
//
//	@Summary		List notification channels
//	@Description	Returns all notification channels with sensitive fields masked.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200 {array} NotificationChannel
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/notifications [get]
func (m *Module) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	channels, err := m.store.ListChannels(r.Context())
	if err != nil {
		m.logger.Warn("failed to list notification channels", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list channels")
		return
	}
	if channels == nil {
		channels = []NotificationChannel{}
	}
	for i := range channels {
		channels[i].Config = maskChannelConfig(channels[i].Config)
	}
	pulseWriteJSON(w, http.StatusOK, channels)
}

// handleGetNotification returns a single notification channel.
//
//	@Summary		Get notification channel
//	@Description	Returns a single notification channel by ID with sensitive fields masked.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Channel ID"
//	@Success		200 {object} NotificationChannel
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/notifications/{id} [get]
func (m *Module) handleGetNotification(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	ch, err := m.store.GetChannel(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get notification channel", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get channel")
		return
	}
	if ch == nil {
		pulseWriteError(w, http.StatusNotFound, "channel not found")
		return
	}
	ch.Config = maskChannelConfig(ch.Config)
	pulseWriteJSON(w, http.StatusOK, ch)
}

// handleCreateNotification creates a new notification channel.
//
//	@Summary		Create notification channel
//	@Description	Creates a new notification channel (webhook or email).
//	@Tags			pulse
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body body createNotificationRequest true "Channel definition"
//	@Success		201 {object} NotificationChannel
//	@Failure		400 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/notifications [post]
func (m *Module) handleCreateNotification(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	var req createNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pulseWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		pulseWriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Type != "webhook" && req.Type != "email" {
		pulseWriteError(w, http.StatusBadRequest, "type must be webhook or email")
		return
	}
	if req.Config == "" {
		pulseWriteError(w, http.StatusBadRequest, "config is required")
		return
	}
	// Validate config is valid JSON.
	if !json.Valid([]byte(req.Config)) {
		pulseWriteError(w, http.StatusBadRequest, "config must be valid JSON")
		return
	}

	now := time.Now().UTC()
	ch := &NotificationChannel{
		ID:        fmt.Sprintf("notif-%s-%d", req.Type, now.UnixMilli()),
		Name:      req.Name,
		Type:      req.Type,
		Config:    req.Config,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := m.store.InsertChannel(r.Context(), ch); err != nil {
		m.logger.Warn("failed to create notification channel", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to create channel")
		return
	}

	ch.Config = maskChannelConfig(ch.Config)
	pulseWriteJSON(w, http.StatusCreated, ch)
}

// handleUpdateNotification updates an existing notification channel.
//
//	@Summary		Update notification channel
//	@Description	Updates fields on an existing notification channel.
//	@Tags			pulse
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Channel ID"
//	@Param			body body updateNotificationRequest true "Fields to update"
//	@Success		200 {object} NotificationChannel
//	@Failure		400 {object} map[string]any
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/notifications/{id} [put]
func (m *Module) handleUpdateNotification(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	existing, err := m.store.GetChannel(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get channel for update", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get channel")
		return
	}
	if existing == nil {
		pulseWriteError(w, http.StatusNotFound, "channel not found")
		return
	}

	var req updateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pulseWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Config != "" {
		if !json.Valid([]byte(req.Config)) {
			pulseWriteError(w, http.StatusBadRequest, "config must be valid JSON")
			return
		}
		existing.Config = req.Config
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := m.store.UpdateChannel(r.Context(), existing); err != nil {
		m.logger.Warn("failed to update channel", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to update channel")
		return
	}

	existing.Config = maskChannelConfig(existing.Config)
	pulseWriteJSON(w, http.StatusOK, existing)
}

// handleDeleteNotification deletes a notification channel.
//
//	@Summary		Delete notification channel
//	@Description	Deletes a notification channel by ID.
//	@Tags			pulse
//	@Security		BearerAuth
//	@Param			id path string true "Channel ID"
//	@Success		204
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/notifications/{id} [delete]
func (m *Module) handleDeleteNotification(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := m.store.DeleteChannel(r.Context(), id); err != nil {
		m.logger.Warn("failed to delete channel", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to delete channel")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleTestNotification sends a synthetic test alert through a notification channel.
//
//	@Summary		Test notification channel
//	@Description	Sends a test notification through the specified channel.
//	@Tags			pulse
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id path string true "Channel ID"
//	@Success		200 {object} map[string]any
//	@Failure		404 {object} map[string]any
//	@Failure		500 {object} map[string]any
//	@Router			/pulse/notifications/{id}/test [post]
func (m *Module) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	ch, err := m.store.GetChannel(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get channel for test", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get channel")
		return
	}
	if ch == nil {
		pulseWriteError(w, http.StatusNotFound, "channel not found")
		return
	}

	// Build a synthetic test alert.
	testAlert := &Alert{
		ID:                  "test-alert",
		CheckID:             "test-check",
		DeviceID:            "test-device",
		Severity:            "warning",
		Message:             "This is a test notification from SubNetree",
		TriggeredAt:         time.Now().UTC(),
		ConsecutiveFailures: 1,
	}

	notifier, err := buildNotifier(*ch)
	if err != nil {
		m.logger.Warn("failed to build notifier for test", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to build notifier: "+err.Error())
		return
	}
	if notifier == nil {
		pulseWriteJSON(w, http.StatusOK, map[string]any{
			"status":  "skipped",
			"message": ch.Type + " notifications are not yet implemented",
		})
		return
	}

	if err := notifier.Notify(r.Context(), testAlert, "test"); err != nil {
		m.logger.Warn("test notification failed", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "test notification failed: "+err.Error())
		return
	}

	pulseWriteJSON(w, http.StatusOK, map[string]any{
		"status":  "sent",
		"message": "Test notification delivered successfully",
	})
}

// maskChannelConfig replaces sensitive values (secret, password) with "****" in config JSON.
func maskChannelConfig(cfgJSON string) string {
	var raw map[string]any
	if err := json.Unmarshal([]byte(cfgJSON), &raw); err != nil {
		return cfgJSON
	}
	for _, key := range []string{"secret", "password"} {
		if _, ok := raw[key]; ok {
			if v, isStr := raw[key].(string); isStr && v != "" {
				raw[key] = "****"
			}
		}
	}
	masked, err := json.Marshal(raw)
	if err != nil {
		return cfgJSON
	}
	return string(masked)
}

// buildNotifier creates a Notifier from a NotificationChannel configuration.
func buildNotifier(ch NotificationChannel) (Notifier, error) {
	switch ch.Type {
	case "webhook":
		var cfg WebhookConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			return nil, fmt.Errorf("unmarshal webhook config: %w", err)
		}
		if cfg.URL == "" {
			return nil, fmt.Errorf("webhook URL is required")
		}
		return NewWebhookNotifier(cfg), nil
	case "email":
		// Email notifications are stubbed for future implementation.
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported notification type: %s", ch.Type)
	}
}

// -- Maintenance window handlers --

// handleListMaintWindows returns all maintenance windows.
func (m *Module) handleListMaintWindows(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	windows, err := m.store.ListMaintWindows(r.Context())
	if err != nil {
		m.logger.Warn("failed to list maintenance windows", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to list maintenance windows")
		return
	}
	if windows == nil {
		windows = []MaintWindow{}
	}
	pulseWriteJSON(w, http.StatusOK, windows)
}

// handleCreateMaintWindow creates a new maintenance window.
func (m *Module) handleCreateMaintWindow(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	var req createMaintWindowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pulseWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		pulseWriteError(w, http.StatusBadRequest, "name is required")
		return
	}
	if !validRecurrence[req.Recurrence] {
		pulseWriteError(w, http.StatusBadRequest, "recurrence must be once, daily, weekly, or monthly")
		return
	}
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		pulseWriteError(w, http.StatusBadRequest, "start_time must be RFC3339 format")
		return
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		pulseWriteError(w, http.StatusBadRequest, "end_time must be RFC3339 format")
		return
	}
	if !endTime.After(startTime) {
		pulseWriteError(w, http.StatusBadRequest, "end_time must be after start_time")
		return
	}
	if len(req.DeviceIDs) == 0 {
		pulseWriteError(w, http.StatusBadRequest, "device_ids must not be empty")
		return
	}

	now := time.Now().UTC()
	mw := &MaintWindow{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		StartTime:   startTime.UTC(),
		EndTime:     endTime.UTC(),
		Recurrence:  req.Recurrence,
		DeviceIDs:   req.DeviceIDs,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := m.store.InsertMaintWindow(r.Context(), mw); err != nil {
		m.logger.Warn("failed to create maintenance window", zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to create maintenance window")
		return
	}
	pulseWriteJSON(w, http.StatusCreated, mw)
}

// handleGetMaintWindow returns a single maintenance window by ID.
func (m *Module) handleGetMaintWindow(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	mw, err := m.store.GetMaintWindow(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get maintenance window", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get maintenance window")
		return
	}
	if mw == nil {
		pulseWriteError(w, http.StatusNotFound, "maintenance window not found")
		return
	}
	pulseWriteJSON(w, http.StatusOK, mw)
}

// handleUpdateMaintWindow updates an existing maintenance window.
func (m *Module) handleUpdateMaintWindow(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	existing, err := m.store.GetMaintWindow(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get maintenance window for update", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to get maintenance window")
		return
	}
	if existing == nil {
		pulseWriteError(w, http.StatusNotFound, "maintenance window not found")
		return
	}

	var req updateMaintWindowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pulseWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			pulseWriteError(w, http.StatusBadRequest, "start_time must be RFC3339 format")
			return
		}
		existing.StartTime = t.UTC()
	}
	if req.EndTime != "" {
		t, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			pulseWriteError(w, http.StatusBadRequest, "end_time must be RFC3339 format")
			return
		}
		existing.EndTime = t.UTC()
	}
	if req.Recurrence != "" {
		if !validRecurrence[req.Recurrence] {
			pulseWriteError(w, http.StatusBadRequest, "recurrence must be once, daily, weekly, or monthly")
			return
		}
		existing.Recurrence = req.Recurrence
	}
	if len(req.DeviceIDs) > 0 {
		existing.DeviceIDs = req.DeviceIDs
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := m.store.UpdateMaintWindow(r.Context(), existing); err != nil {
		m.logger.Warn("failed to update maintenance window", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to update maintenance window")
		return
	}
	pulseWriteJSON(w, http.StatusOK, existing)
}

// handleDeleteMaintWindow deletes a maintenance window.
func (m *Module) handleDeleteMaintWindow(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		pulseWriteError(w, http.StatusServiceUnavailable, "pulse store not available")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		pulseWriteError(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := m.store.DeleteMaintWindow(r.Context(), id); err != nil {
		m.logger.Warn("failed to delete maintenance window", zap.String("id", id), zap.Error(err))
		pulseWriteError(w, http.StatusInternalServerError, "failed to delete maintenance window")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// validateTarget validates a check target based on the check type.
func validateTarget(checkType, target string) error {
	switch checkType {
	case "icmp":
		if net.ParseIP(target) == nil {
			// Not an IP -- check it's a non-empty hostname.
			if strings.TrimSpace(target) == "" {
				return fmt.Errorf("icmp target must be a valid IP or hostname")
			}
		}
	case "tcp":
		if _, _, err := net.SplitHostPort(target); err != nil {
			return fmt.Errorf("tcp target must be host:port format")
		}
	case "http":
		u, err := url.Parse(target)
		if err != nil {
			return fmt.Errorf("http target must be a valid URL")
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("http target must have http or https scheme")
		}
	}
	return nil
}
