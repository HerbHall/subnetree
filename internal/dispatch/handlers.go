package dispatch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	_ "github.com/HerbHall/subnetree/pkg/models" // swagger type reference
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/agents", Handler: m.handleListAgents},
		{Method: "GET", Path: "/agents/{id}", Handler: m.handleGetAgent},
		{Method: "POST", Path: "/enroll", Handler: m.handleCreateEnrollmentToken},
		{Method: "DELETE", Path: "/agents/{id}", Handler: m.handleDeleteAgent},
	}
}

// handleListAgents returns all connected Scout agents.
//
//	@Summary		List agents
//	@Description	Returns all registered Scout agents.
//	@Tags			dispatch
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}	models.AgentInfo
//	@Router			/dispatch/agents [get]
func (m *Module) handleListAgents(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		dispatchWriteError(w, http.StatusServiceUnavailable, "dispatch store not available")
		return
	}

	agents, err := m.store.ListAgents(r.Context())
	if err != nil {
		m.logger.Warn("failed to list agents", zap.Error(err))
		dispatchWriteError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	if agents == nil {
		agents = []Agent{}
	}
	dispatchWriteJSON(w, http.StatusOK, agents)
}

// handleGetAgent returns a single Scout agent by ID.
//
//	@Summary		Get agent
//	@Description	Returns a single Scout agent by ID.
//	@Tags			dispatch
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	models.AgentInfo
//	@Failure		404	{object}	models.APIProblem
//	@Router			/dispatch/agents/{id} [get]
func (m *Module) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		dispatchWriteError(w, http.StatusServiceUnavailable, "dispatch store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		dispatchWriteError(w, http.StatusBadRequest, "agent id is required")
		return
	}

	agent, err := m.store.GetAgent(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get agent", zap.String("id", id), zap.Error(err))
		dispatchWriteError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if agent == nil {
		dispatchWriteError(w, http.StatusNotFound, "agent not found")
		return
	}
	dispatchWriteJSON(w, http.StatusOK, agent)
}

// handleDeleteAgent removes a Scout agent by ID.
//
//	@Summary		Delete agent
//	@Description	Removes a Scout agent by ID.
//	@Tags			dispatch
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Agent ID"
//	@Success		204
//	@Failure		404	{object}	models.APIProblem
//	@Router			/dispatch/agents/{id} [delete]
func (m *Module) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		dispatchWriteError(w, http.StatusServiceUnavailable, "dispatch store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		dispatchWriteError(w, http.StatusBadRequest, "agent id is required")
		return
	}

	if err := m.store.DeleteAgent(r.Context(), id); err != nil {
		m.logger.Warn("failed to delete agent", zap.String("id", id), zap.Error(err))
		dispatchWriteError(w, http.StatusNotFound, "agent not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// enrollTokenRequest is the JSON body for creating enrollment tokens.
type enrollTokenRequest struct {
	Description string `json:"description"`
	MaxUses     int    `json:"max_uses,omitempty"`
	ExpiresIn   string `json:"expires_in,omitempty"` // e.g. "24h", "7d"
}

// enrollTokenResponse is returned after creating an enrollment token.
type enrollTokenResponse struct {
	ID        string     `json:"id"`
	Token     string     `json:"token"` // raw token (only returned once)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	MaxUses   int        `json:"max_uses"`
}

// handleCreateEnrollmentToken creates a new enrollment token.
//
//	@Summary		Create enrollment token
//	@Description	Creates a new enrollment token for agent registration.
//	@Tags			dispatch
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		enrollTokenRequest	true	"Token parameters"
//	@Success		201		{object}	enrollTokenResponse
//	@Failure		400		{object}	models.APIProblem
//	@Router			/dispatch/enroll [post]
func (m *Module) handleCreateEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		dispatchWriteError(w, http.StatusServiceUnavailable, "dispatch store not available")
		return
	}

	var req enrollTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dispatchWriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.MaxUses <= 0 {
		req.MaxUses = 1
	}

	// Generate a random token.
	rawToken := uuid.New().String()

	// Hash the token for storage.
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	now := time.Now().UTC()
	token := &EnrollmentToken{
		ID:          uuid.New().String(),
		TokenHash:   tokenHash,
		Description: req.Description,
		CreatedAt:   now,
		MaxUses:     req.MaxUses,
	}

	// Parse expiry duration if provided, otherwise use default.
	expiresIn := m.cfg.EnrollmentTokenExpiry
	if req.ExpiresIn != "" {
		d, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			dispatchWriteError(w, http.StatusBadRequest, "invalid expires_in duration")
			return
		}
		expiresIn = d
	}
	expiresAt := now.Add(expiresIn)
	token.ExpiresAt = &expiresAt

	if err := m.store.CreateEnrollmentToken(r.Context(), token); err != nil {
		m.logger.Warn("failed to create enrollment token", zap.Error(err))
		dispatchWriteError(w, http.StatusInternalServerError, "failed to create enrollment token")
		return
	}

	dispatchWriteJSON(w, http.StatusCreated, enrollTokenResponse{
		ID:        token.ID,
		Token:     rawToken,
		ExpiresAt: token.ExpiresAt,
		MaxUses:   token.MaxUses,
	})
}

// -- helpers --

func dispatchWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func dispatchWriteError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
