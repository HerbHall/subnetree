package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"
)

// Handler provides HTTP handlers for authentication endpoints.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates an auth Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RegisterRoutes registers auth-related routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Public auth endpoints (no JWT required).
	mux.HandleFunc("POST /api/v1/auth/login", h.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/logout", h.handleLogout)
	mux.HandleFunc("POST /api/v1/auth/setup", h.handleSetup)

	// Admin-only user management endpoints (auth enforced by middleware,
	// role checked in handlers).
	mux.HandleFunc("GET /api/v1/users", h.handleListUsers)
	mux.HandleFunc("GET /api/v1/users/{id}", h.handleGetUser)
	mux.HandleFunc("PUT /api/v1/users/{id}", h.handleUpdateUser)
	mux.HandleFunc("DELETE /api/v1/users/{id}", h.handleDeleteUser)
}

// Middleware returns the JWT authentication middleware.
func (h *Handler) Middleware() func(http.Handler) http.Handler {
	return AuthMiddleware(h.service.Tokens())
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeAuthError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	pair, err := h.service.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserDisabled) {
			writeAuthError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		h.logger.Error("login error", zap.Error(err))
		writeAuthError(w, http.StatusInternalServerError, "authentication failed")
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		writeAuthError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	pair, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrUserDisabled) {
			writeAuthError(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}
		h.logger.Error("refresh error", zap.Error(err))
		writeAuthError(w, http.StatusInternalServerError, "token refresh failed")
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		writeAuthError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		h.logger.Error("logout error", zap.Error(err))
		writeAuthError(w, http.StatusInternalServerError, "logout failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeAuthError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	user, err := h.service.Setup(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrSetupComplete) {
			writeAuthError(w, http.StatusConflict, "setup already completed")
			return
		}
		writeAuthError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		h.logger.Error("list users error", zap.Error(err))
		writeAuthError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	user, err := h.service.GetUser(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeAuthError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("get user error", zap.Error(err))
		writeAuthError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	var req struct {
		Email    string `json:"email"`
		Role     string `json:"role"`
		Disabled bool   `json:"disabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role := Role(req.Role)
	if !ValidRoles[role] {
		writeAuthError(w, http.StatusBadRequest, "invalid role: must be admin, operator, or viewer")
		return
	}

	user, err := h.service.UpdateUser(r.Context(), r.PathValue("id"), req.Email, role, req.Disabled)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeAuthError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("update user error", zap.Error(err))
		writeAuthError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	if err := h.service.DeleteUser(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeAuthError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("delete user error", zap.Error(err))
		writeAuthError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// requireAdmin checks that the authenticated user has admin role.
// Returns false (and writes an error response) if not authorized.
func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	user := UserFromContext(r.Context())
	if user == nil {
		writeAuthError(w, http.StatusUnauthorized, "authentication required")
		return false
	}
	if Role(user.Role) != RoleAdmin {
		writeAuthError(w, http.StatusForbidden, "admin role required")
		return false
	}
	return true
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeAuthError writes an RFC 7807 problem response.
func writeAuthError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://netvantage.io/problems/auth-error",
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
