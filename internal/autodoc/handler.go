package autodoc

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
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

// ChangelogListResponse is the paginated response for GET /changes.
type ChangelogListResponse struct {
	Entries []ChangelogEntry `json:"entries"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

// handleListChanges returns a paginated list of changelog entries.
//
//	@Summary		List changelog entries
//	@Description	Returns a paginated list of auto-documentation changelog entries with optional filters.
//	@Tags			autodoc
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Page number"		default(1)
//	@Param			per_page	query		int		false	"Items per page"	default(50)
//	@Param			event_type	query		string	false	"Filter by event type"
//	@Param			since		query		string	false	"Start date (RFC3339)"
//	@Param			until		query		string	false	"End date (RFC3339)"
//	@Success		200			{object}	ChangelogListResponse
//	@Failure		500			{object}	map[string]any
//	@Router			/autodoc/changes [get]
func (m *Module) handleListChanges(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{
		Page:      queryInt(r, "page", 1),
		PerPage:   queryInt(r, "per_page", 50),
		EventType: r.URL.Query().Get("event_type"),
	}

	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			filter.Since = &t
		}
	}
	if s := r.URL.Query().Get("until"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			filter.Until = &t
		}
	}

	entries, total, err := m.store.ListEntries(r.Context(), filter)
	if err != nil {
		m.logger.Error("failed to list changelog entries", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to list changelog entries")
		return
	}

	writeJSON(w, http.StatusOK, ChangelogListResponse{
		Entries: entries,
		Total:   total,
		Page:    filter.Page,
		PerPage: filter.PerPage,
	})
}

// handleExport returns a markdown export of changelog entries.
//
//	@Summary		Export changelog as markdown
//	@Description	Generates a markdown summary of changelog entries for the given time range.
//	@Tags			autodoc
//	@Produce		text/markdown
//	@Security		BearerAuth
//	@Param			range	query		string	false	"Time range: 7d, 30d, or custom"	default(7d)
//	@Param			since	query		string	false	"Custom start date (RFC3339)"
//	@Param			until	query		string	false	"Custom end date (RFC3339)"
//	@Success		200		{string}	string	"Markdown text"
//	@Failure		500		{object}	map[string]any
//	@Router			/autodoc/export [get]
func (m *Module) handleExport(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	rangeParam := r.URL.Query().Get("range")

	var since, until time.Time

	// Parse "since" and "until" if provided directly.
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}
	if s := r.URL.Query().Get("until"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			until = t
		}
	}

	// If no custom dates, use the range parameter.
	if since.IsZero() {
		dur := ParseDuration(rangeParam, 7*24*time.Hour)
		since = now.Add(-dur)
	}
	if until.IsZero() {
		until = now
	}

	entries, err := m.store.ListEntriesBetween(r.Context(), since, until)
	if err != nil {
		m.logger.Error("failed to list entries for export", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to export changelog")
		return
	}

	md := GenerateMarkdown(entries)

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="changelog.md"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(md))
}

// handleStats returns summary statistics about the changelog.
//
//	@Summary		Changelog statistics
//	@Description	Returns aggregate statistics about the auto-documentation changelog.
//	@Tags			autodoc
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	Stats
//	@Failure		500	{object}	map[string]any
//	@Router			/autodoc/stats [get]
func (m *Module) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := m.store.GetStats(r.Context())
	if err != nil {
		m.logger.Error("failed to get changelog stats", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get changelog stats")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// handleDeviceDoc generates a per-device Markdown document.
//
//	@Summary		Generate device documentation
//	@Description	Produces a comprehensive Markdown document for a single device including hardware, services, alerts, and changelog.
//	@Tags			autodoc
//	@Produce		text/markdown
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Device ID"
//	@Success		200	{string}	string	"Markdown text"
//	@Failure		404	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Router			/autodoc/devices/{id} [get]
func (m *Module) handleDeviceDoc(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device ID is required")
		return
	}

	if m.deviceReader == nil {
		writeError(w, http.StatusServiceUnavailable, "device data source not configured")
		return
	}

	ctx := r.Context()

	device, err := m.deviceReader.GetDevice(ctx, deviceID)
	if err != nil {
		m.logger.Error("failed to get device", zap.String("device_id", deviceID), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to get device")
		return
	}
	if device == nil {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	data := m.assembleDeviceDocData(ctx, device)

	md, renderErr := RenderDeviceDoc(data)
	if renderErr != nil {
		m.logger.Error("failed to render device doc", zap.String("device_id", deviceID), zap.Error(renderErr))
		writeError(w, http.StatusInternalServerError, "failed to render device document")
		return
	}

	filename := deviceDocFilename(device)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(md))
}

// handleBulkExport generates a zip archive of per-device Markdown documents.
//
//	@Summary		Bulk export device documentation
//	@Description	Generates a zip archive containing a Markdown document for every device.
//	@Tags			autodoc
//	@Produce		application/zip
//	@Security		BearerAuth
//	@Success		200	{file}		file	"Zip archive"
//	@Failure		500	{object}	map[string]any
//	@Router			/autodoc/devices [get]
func (m *Module) handleBulkExport(w http.ResponseWriter, r *http.Request) {
	if m.deviceReader == nil {
		writeError(w, http.StatusServiceUnavailable, "device data source not configured")
		return
	}

	ctx := r.Context()

	devices, err := m.deviceReader.ListAllDevices(ctx)
	if err != nil {
		m.logger.Error("failed to list devices for bulk export", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to list devices")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="device-docs.zip"`)
	w.WriteHeader(http.StatusOK)

	zw := zip.NewWriter(w)
	defer func() { _ = zw.Close() }()

	for i := range devices {
		data := m.assembleDeviceDocData(ctx, &devices[i])
		md, renderErr := RenderDeviceDoc(data)
		if renderErr != nil {
			m.logger.Warn("failed to render device doc, skipping",
				zap.String("device_id", devices[i].ID),
				zap.Error(renderErr),
			)
			continue
		}

		filename := deviceDocFilename(&devices[i])
		fw, createErr := zw.Create(filename)
		if createErr != nil {
			m.logger.Warn("failed to create zip entry, skipping",
				zap.String("filename", filename),
				zap.Error(createErr),
			)
			continue
		}
		_, _ = fw.Write([]byte(md))
	}
}

// assembleDeviceDocData gathers all data sources for a single device document.
func (m *Module) assembleDeviceDocData(ctx context.Context, device *models.Device) DeviceDocData {
	data := DeviceDocData{
		Device:      device,
		GeneratedAt: time.Now().UTC(),
	}

	if m.deviceReader != nil {
		if hw, err := m.deviceReader.GetDeviceHardware(ctx, device.ID); err == nil {
			data.Hardware = hw
		}
		if storage, err := m.deviceReader.GetDeviceStorage(ctx, device.ID); err == nil {
			data.Storage = storage
		}
		if gpus, err := m.deviceReader.GetDeviceGPU(ctx, device.ID); err == nil {
			data.GPUs = gpus
		}
		if services, err := m.deviceReader.GetDeviceServices(ctx, device.ID); err == nil {
			data.Services = services
		}
		if children, err := m.deviceReader.GetChildDevices(ctx, device.ID); err == nil {
			data.Children = children
		}
	}

	if m.alertReader != nil {
		if alerts, err := m.alertReader.ListDeviceAlerts(ctx, device.ID, 20); err == nil {
			data.Alerts = alerts
		}
	}

	// Get recent changelog entries for this device (last 30 days).
	if m.store != nil {
		since := time.Now().UTC().Add(-30 * 24 * time.Hour)
		entries, _, err := m.store.ListEntries(ctx, ListFilter{
			Page:     1,
			PerPage:  50,
			DeviceID: device.ID,
			Since:    &since,
		})
		if err == nil {
			data.RecentChanges = entries
		}
	}

	return data
}

// deviceDocFilename generates a safe filename for a device document.
func deviceDocFilename(d *models.Device) string {
	name := d.Hostname
	if name == "" && len(d.IPAddresses) > 0 {
		name = d.IPAddresses[0]
	}
	if name == "" {
		name = d.ID
	}
	// Replace characters that are unsafe in filenames.
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return replacer.Replace(name) + ".md"
}

// queryInt extracts an integer query parameter with a default value.
func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return defaultVal
	}
	return v
}
