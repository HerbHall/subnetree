package autodoc

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
//	@Failure		500			{object}	models.APIProblem
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
//	@Failure		500		{object}	models.APIProblem
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
//	@Failure		500	{object}	models.APIProblem
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
