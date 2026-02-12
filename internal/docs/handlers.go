package docs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/applications", Handler: m.handleListApplications},
		{Method: "GET", Path: "/applications/{id}", Handler: m.handleGetApplication},
		{Method: "GET", Path: "/applications/{id}/history", Handler: m.handleApplicationHistory},
		{Method: "GET", Path: "/snapshots", Handler: m.handleListSnapshots},
		{Method: "GET", Path: "/snapshots/{id}", Handler: m.handleGetSnapshot},
		{Method: "POST", Path: "/snapshots", Handler: m.handleCreateSnapshot},
		{Method: "DELETE", Path: "/snapshots/{id}", Handler: m.handleDeleteSnapshot},
		{Method: "GET", Path: "/snapshots/{id}/diff/{other_id}", Handler: m.handleSnapshotDiff},
		{Method: "GET", Path: "/collectors", Handler: m.handleListCollectors},
		{Method: "POST", Path: "/collect", Handler: m.handleTriggerCollection},
		{Method: "POST", Path: "/collect/{collector}", Handler: m.handleTriggerCollectorByName},
	}
}

// handleListApplications returns a paginated, filtered list of applications.
//
//	@Summary		List applications
//	@Description	Returns a paginated list of discovered infrastructure applications.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int		false	"Max results"	default(50)
//	@Param			offset	query		int		false	"Offset"		default(0)
//	@Param			type	query		string	false	"Filter by app_type"
//	@Param			status	query		string	false	"Filter by status"
//	@Success		200		{object}	map[string]any
//	@Failure		500		{object}	map[string]any
//	@Router			/docs/applications [get]
func (m *Module) handleListApplications(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	params := ListApplicationsParams{
		Limit:   docsQueryInt(r, "limit", 50),
		Offset:  docsQueryInt(r, "offset", 0),
		AppType: r.URL.Query().Get("type"),
		Status:  r.URL.Query().Get("status"),
	}

	apps, total, err := m.store.ListApplications(r.Context(), params)
	if err != nil {
		m.logger.Warn("failed to list applications", zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to list applications")
		return
	}
	if apps == nil {
		apps = []Application{}
	}
	docsWriteJSON(w, http.StatusOK, map[string]any{
		"items": apps,
		"total": total,
	})
}

// handleGetApplication returns a single application by ID.
//
//	@Summary		Get application
//	@Description	Returns a single infrastructure application by its ID.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Application ID"
//	@Success		200	{object}	Application
//	@Failure		400	{object}	map[string]any
//	@Failure		404	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Router			/docs/applications/{id} [get]
func (m *Module) handleGetApplication(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		docsWriteError(w, http.StatusBadRequest, "application ID is required")
		return
	}

	app, err := m.store.GetApplication(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get application", zap.String("id", id), zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to get application")
		return
	}
	if app == nil {
		docsWriteError(w, http.StatusNotFound, "application not found")
		return
	}
	docsWriteJSON(w, http.StatusOK, app)
}

// handleListSnapshots returns a filtered, paginated list of snapshots.
//
//	@Summary		List snapshots
//	@Description	Returns a paginated list of configuration snapshots, optionally filtered by application.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			application_id	query		string	false	"Filter by application ID"
//	@Param			limit			query		int		false	"Max results"	default(50)
//	@Param			offset			query		int		false	"Offset"		default(0)
//	@Success		200				{array}		Snapshot
//	@Failure		500				{object}	map[string]any
//	@Router			/docs/snapshots [get]
func (m *Module) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	params := ListSnapshotsParams{
		ApplicationID: r.URL.Query().Get("application_id"),
		Limit:         docsQueryInt(r, "limit", 50),
		Offset:        docsQueryInt(r, "offset", 0),
	}

	snapshots, err := m.store.ListSnapshots(r.Context(), params)
	if err != nil {
		m.logger.Warn("failed to list snapshots", zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to list snapshots")
		return
	}
	if snapshots == nil {
		snapshots = []Snapshot{}
	}
	docsWriteJSON(w, http.StatusOK, snapshots)
}

// handleGetSnapshot returns a single snapshot by ID.
//
//	@Summary		Get snapshot
//	@Description	Returns a single configuration snapshot including its content.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Snapshot ID"
//	@Success		200	{object}	Snapshot
//	@Failure		400	{object}	map[string]any
//	@Failure		404	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Router			/docs/snapshots/{id} [get]
func (m *Module) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		docsWriteError(w, http.StatusBadRequest, "snapshot ID is required")
		return
	}

	snap, err := m.store.GetSnapshot(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get snapshot", zap.String("id", id), zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to get snapshot")
		return
	}
	if snap == nil {
		docsWriteError(w, http.StatusNotFound, "snapshot not found")
		return
	}
	docsWriteJSON(w, http.StatusOK, snap)
}

// CreateSnapshotRequest is the request body for POST /snapshots.
type CreateSnapshotRequest struct {
	ApplicationID string `json:"application_id"`
	Content       string `json:"content"`
	Format        string `json:"format"`
}

// handleCreateSnapshot creates a new configuration snapshot.
//
//	@Summary		Create snapshot
//	@Description	Creates a new configuration snapshot for an application.
//	@Tags			docs
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateSnapshotRequest	true	"Snapshot to create"
//	@Success		201		{object}	Snapshot
//	@Failure		400		{object}	map[string]any
//	@Failure		500		{object}	map[string]any
//	@Router			/docs/snapshots [post]
func (m *Module) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	var req CreateSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		docsWriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.ApplicationID == "" {
		docsWriteError(w, http.StatusBadRequest, "application_id is required")
		return
	}
	if req.Content == "" {
		docsWriteError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.Format == "" {
		req.Format = "json"
	}

	hash := sha256.Sum256([]byte(req.Content))

	snap := &Snapshot{
		ID:            uuid.New().String(),
		ApplicationID: req.ApplicationID,
		ContentHash:   hex.EncodeToString(hash[:]),
		Content:       req.Content,
		Format:        req.Format,
		SizeBytes:     len(req.Content),
		Source:        "manual",
		CapturedAt:    time.Now().UTC(),
	}

	if err := m.store.InsertSnapshot(r.Context(), snap); err != nil {
		m.logger.Error("failed to create snapshot", zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to create snapshot")
		return
	}

	m.publishEvent(r.Context(), TopicSnapshotCreated, snap)

	docsWriteJSON(w, http.StatusCreated, snap)
}

// CollectorInfo describes a registered collector and its availability.
type CollectorInfo struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

// handleListCollectors returns the registered collectors with availability status.
//
//	@Summary		List collectors
//	@Description	Returns all registered infrastructure collectors and whether they are currently available.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		CollectorInfo
//	@Router			/docs/collectors [get]
func (m *Module) handleListCollectors(w http.ResponseWriter, _ *http.Request) {
	infos := make([]CollectorInfo, 0, len(m.collectors))
	for _, c := range m.collectors {
		infos = append(infos, CollectorInfo{
			Name:      c.Name(),
			Available: c.Available(),
		})
	}
	docsWriteJSON(w, http.StatusOK, infos)
}

// handleTriggerCollection runs collection from all available collectors.
//
//	@Summary		Trigger collection
//	@Description	Runs configuration collection from all available collectors immediately.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	CollectionResult
//	@Failure		503	{object}	map[string]any
//	@Router			/docs/collect [post]
func (m *Module) handleTriggerCollection(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	result := m.RunCollection(r.Context())
	docsWriteJSON(w, http.StatusOK, result)
}

// handleTriggerCollectorByName runs collection from a single named collector.
//
//	@Summary		Trigger collector by name
//	@Description	Runs configuration collection from a specific named collector.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			collector	path		string	true	"Collector name"
//	@Success		200			{object}	CollectionResult
//	@Failure		404			{object}	map[string]any
//	@Failure		503			{object}	map[string]any
//	@Router			/docs/collect/{collector} [post]
func (m *Module) handleTriggerCollectorByName(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	name := r.PathValue("collector")
	if name == "" {
		docsWriteError(w, http.StatusBadRequest, "collector name is required")
		return
	}

	result, err := m.RunCollectorByName(r.Context(), name)
	if err != nil {
		docsWriteError(w, http.StatusNotFound, err.Error())
		return
	}
	docsWriteJSON(w, http.StatusOK, result)
}

// handleApplicationHistory returns paginated snapshot history for an application.
//
//	@Summary		Application history
//	@Description	Returns paginated snapshot history for an application with total count.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"Application ID"
//	@Param			limit	query		int		false	"Max results"	default(20)
//	@Param			offset	query		int		false	"Offset"		default(0)
//	@Success		200		{object}	map[string]any
//	@Failure		400		{object}	map[string]any
//	@Failure		404		{object}	map[string]any
//	@Failure		500		{object}	map[string]any
//	@Router			/docs/applications/{id}/history [get]
func (m *Module) handleApplicationHistory(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		docsWriteError(w, http.StatusBadRequest, "application ID is required")
		return
	}

	app, err := m.store.GetApplication(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get application for history", zap.String("id", id), zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to get application")
		return
	}
	if app == nil {
		docsWriteError(w, http.StatusNotFound, "application not found")
		return
	}

	limit := docsQueryInt(r, "limit", 20)
	offset := docsQueryInt(r, "offset", 0)

	snapshots, total, err := m.store.ListSnapshotHistory(r.Context(), id, limit, offset)
	if err != nil {
		m.logger.Warn("failed to list snapshot history", zap.String("id", id), zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to list snapshot history")
		return
	}
	if snapshots == nil {
		snapshots = []Snapshot{}
	}

	docsWriteJSON(w, http.StatusOK, map[string]any{
		"snapshots": snapshots,
		"total":     total,
	})
}

// DiffResponse is the response body for GET /snapshots/{id}/diff/{other_id}.
type DiffResponse struct {
	DiffText      string `json:"diff_text"`
	OldSnapshotID string `json:"old_snapshot_id"`
	NewSnapshotID string `json:"new_snapshot_id"`
}

// handleSnapshotDiff returns a unified diff between two snapshots.
//
//	@Summary		Diff snapshots
//	@Description	Returns a unified diff between two configuration snapshots.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string	true	"Base snapshot ID"
//	@Param			other_id	path		string	true	"Comparison snapshot ID"
//	@Success		200			{object}	DiffResponse
//	@Failure		400			{object}	map[string]any
//	@Failure		404			{object}	map[string]any
//	@Failure		500			{object}	map[string]any
//	@Router			/docs/snapshots/{id}/diff/{other_id} [get]
func (m *Module) handleSnapshotDiff(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	id := r.PathValue("id")
	otherID := r.PathValue("other_id")
	if id == "" || otherID == "" {
		docsWriteError(w, http.StatusBadRequest, "both snapshot IDs are required")
		return
	}

	oldSnap, err := m.store.GetSnapshot(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get old snapshot", zap.String("id", id), zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to get snapshot")
		return
	}
	if oldSnap == nil {
		docsWriteError(w, http.StatusNotFound, "base snapshot not found")
		return
	}

	newSnap, err := m.store.GetSnapshot(r.Context(), otherID)
	if err != nil {
		m.logger.Warn("failed to get new snapshot", zap.String("id", otherID), zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to get snapshot")
		return
	}
	if newSnap == nil {
		docsWriteError(w, http.StatusNotFound, "comparison snapshot not found")
		return
	}

	lines := ComputeDiff(oldSnap.Content, newSnap.Content)
	diffText := FormatUnifiedDiff(lines, 3)

	docsWriteJSON(w, http.StatusOK, DiffResponse{
		DiffText:      diffText,
		OldSnapshotID: id,
		NewSnapshotID: otherID,
	})
}

// handleDeleteSnapshot deletes a single snapshot by ID.
//
//	@Summary		Delete snapshot
//	@Description	Deletes a configuration snapshot by its ID.
//	@Tags			docs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Snapshot ID"
//	@Success		204
//	@Failure		400	{object}	map[string]any
//	@Failure		404	{object}	map[string]any
//	@Failure		500	{object}	map[string]any
//	@Router			/docs/snapshots/{id} [delete]
func (m *Module) handleDeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		docsWriteError(w, http.StatusServiceUnavailable, "docs store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		docsWriteError(w, http.StatusBadRequest, "snapshot ID is required")
		return
	}

	snap, err := m.store.GetSnapshot(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get snapshot for delete", zap.String("id", id), zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to get snapshot")
		return
	}
	if snap == nil {
		docsWriteError(w, http.StatusNotFound, "snapshot not found")
		return
	}

	if err := m.store.DeleteSnapshot(r.Context(), id); err != nil {
		m.logger.Error("failed to delete snapshot", zap.String("id", id), zap.Error(err))
		docsWriteError(w, http.StatusInternalServerError, "failed to delete snapshot")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -- helpers --

func docsWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func docsWriteError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

func docsQueryInt(r *http.Request, key string, defaultVal int) int {
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
