// Package settings provides HTTP handlers for application settings endpoints.
package settings

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/internal/services"
	"go.uber.org/zap"
)

// ScanInterfaceRequest represents a request to set the scan interface.
// @Description Request body for setting the network scan interface.
type ScanInterfaceRequest struct {
	InterfaceName string `json:"interface_name" example:"eth0"`
}

// ScanInterfaceResponse represents the current scan interface setting.
// @Description Response containing the configured scan interface.
type ScanInterfaceResponse struct {
	InterfaceName string `json:"interface_name" example:"eth0"`
}

// SettingsProblemDetail represents an RFC 7807 error response for settings endpoints.
// @Description RFC 7807 Problem Details error response.
type SettingsProblemDetail struct {
	Type   string `json:"type" example:"https://subnetree.com/problems/settings-error"`
	Title  string `json:"title" example:"Bad Request"`
	Status int    `json:"status" example:"400"`
	Detail string `json:"detail" example:"interface not found: eth99"`
}

// ThemeLayer identifies a composable layer of a theme.
type ThemeLayer string

const (
	// LayerColors covers backgrounds, text, borders, buttons, inputs, sidebar, status, and charts.
	LayerColors ThemeLayer = "colors"
	// LayerTypography covers font families.
	LayerTypography ThemeLayer = "typography"
	// LayerShape covers border radius values.
	LayerShape ThemeLayer = "shape"
	// LayerEffects covers shadows and transitions.
	LayerEffects ThemeLayer = "effects"
)

// allLayers is the canonical list of all theme layers.
var allLayers = []ThemeLayer{LayerColors, LayerTypography, LayerShape, LayerEffects}

// ThemeTokens represents customizable CSS token overrides organized by category.
// @Description Customizable CSS design token overrides grouped by UI category.
type ThemeTokens struct {
	Backgrounds map[string]string `json:"backgrounds,omitempty"`
	Text        map[string]string `json:"text,omitempty"`
	Borders     map[string]string `json:"borders,omitempty"`
	Buttons     map[string]string `json:"buttons,omitempty"`
	Inputs      map[string]string `json:"inputs,omitempty"`
	Sidebar     map[string]string `json:"sidebar,omitempty"`
	Status      map[string]string `json:"status,omitempty"`
	Charts      map[string]string `json:"charts,omitempty"`
	Typography  map[string]string `json:"typography,omitempty"`
	Spacing     map[string]string `json:"spacing,omitempty"`
	Effects     map[string]string `json:"effects,omitempty"`
}

// ThemeDefinition represents a complete or partial theme configuration.
// @Description A theme with metadata, layer declarations, and CSS token overrides.
type ThemeDefinition struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	BaseMode    string       `json:"base_mode"`
	Version     int          `json:"version"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
	BuiltIn     bool         `json:"built_in"`
	Layers      []ThemeLayer `json:"layers,omitempty"`
	Tokens      ThemeTokens  `json:"tokens"`
}

// ActiveThemeResponse represents the currently active theme reference.
// @Description Response containing the active theme ID.
type ActiveThemeResponse struct {
	ThemeID string `json:"theme_id" example:"builtin-forest-dark"`
}

// ActiveThemeRequest represents a request to set the active theme.
// @Description Request body for setting the active theme.
type ActiveThemeRequest struct {
	ThemeID string `json:"theme_id" example:"builtin-forest-dark"`
}

// Handler provides HTTP handlers for settings endpoints.
type Handler struct {
	interfaces *services.InterfaceService
	settings   services.SettingsRepository
	logger     *zap.Logger
}

// NewHandler creates a settings Handler.
func NewHandler(settings services.SettingsRepository, logger *zap.Logger) *Handler {
	return &Handler{
		interfaces: services.NewInterfaceService(),
		settings:   settings,
		logger:     logger,
	}
}

// RegisterRoutes registers settings-related routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Network interface endpoints (public during setup)
	mux.HandleFunc("GET /api/v1/settings/interfaces", h.handleListInterfaces)
	mux.HandleFunc("GET /api/v1/settings/scan-interface", h.handleGetScanInterface)
	mux.HandleFunc("POST /api/v1/settings/scan-interface", h.handleSetScanInterface)

	// Theme endpoints (literal paths before wildcard)
	mux.HandleFunc("GET /api/v1/settings/themes", h.handleListThemes)
	mux.HandleFunc("GET /api/v1/settings/themes/active", h.handleGetActiveTheme)
	mux.HandleFunc("PUT /api/v1/settings/themes/active", h.handleSetActiveTheme)
	mux.HandleFunc("POST /api/v1/settings/themes", h.handleCreateTheme)
	mux.HandleFunc("GET /api/v1/settings/themes/{id}", h.handleGetTheme)
	mux.HandleFunc("PUT /api/v1/settings/themes/{id}", h.handleUpdateTheme)
	mux.HandleFunc("DELETE /api/v1/settings/themes/{id}", h.handleDeleteTheme)
}

// handleListInterfaces returns all available network interfaces.
//
//	@Summary		List network interfaces
//	@Description	Get a list of all network interfaces available on the server.
//	@Tags			settings
//	@Produce		json
//	@Success		200	{array}		services.NetworkInterface	"List of interfaces"
//	@Failure		500	{object}	SettingsProblemDetail		"Internal server error"
//	@Router			/settings/interfaces [get]
func (h *Handler) handleListInterfaces(w http.ResponseWriter, _ *http.Request) {
	interfaces, err := h.interfaces.ListNetworkInterfaces()
	if err != nil {
		h.logger.Error("failed to list interfaces", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to list network interfaces")
		return
	}

	writeJSON(w, http.StatusOK, interfaces)
}

// handleGetScanInterface returns the currently configured scan interface.
//
//	@Summary		Get scan interface
//	@Description	Get the currently configured network interface for scanning.
//	@Tags			settings
//	@Produce		json
//	@Success		200	{object}	ScanInterfaceResponse	"Current scan interface"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/scan-interface [get]
func (h *Handler) handleGetScanInterface(w http.ResponseWriter, r *http.Request) {
	setting, err := h.settings.Get(r.Context(), "scan_interface")
	if err != nil {
		if err == services.ErrNotFound {
			// No interface configured yet -- return empty response
			writeJSON(w, http.StatusOK, map[string]string{"interface_name": ""})
			return
		}
		h.logger.Error("failed to get scan interface setting", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get scan interface")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"interface_name": setting.Value})
}

// handleSetScanInterface saves the selected scan interface.
//
//	@Summary		Set scan interface
//	@Description	Configure which network interface to use for scanning.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ScanInterfaceRequest	true	"Interface to use"
//	@Success		200		{object}	ScanInterfaceResponse	"Interface configured"
//	@Failure		400		{object}	SettingsProblemDetail	"Invalid request or interface not found"
//	@Failure		500		{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/scan-interface [post]
func (h *Handler) handleSetScanInterface(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InterfaceName string `json:"interface_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSettingsError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate that the interface exists (if non-empty)
	if req.InterfaceName != "" {
		interfaces, err := h.interfaces.ListNetworkInterfaces()
		if err != nil {
			h.logger.Error("failed to list interfaces for validation", zap.Error(err))
			writeSettingsError(w, http.StatusInternalServerError, "failed to validate interface")
			return
		}
		found := false
		for i := range interfaces {
			if interfaces[i].Name == req.InterfaceName {
				found = true
				break
			}
		}
		if !found {
			writeSettingsError(w, http.StatusBadRequest, "interface not found: "+req.InterfaceName)
			return
		}
	}

	if err := h.settings.Set(r.Context(), "scan_interface", req.InterfaceName); err != nil {
		h.logger.Error("failed to set scan interface", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to save scan interface")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"interface_name": req.InterfaceName})
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeSettingsError writes an RFC 7807 problem response.
func writeSettingsError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/settings-error",
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}

// ---------- Theme endpoints ----------

const (
	themeKeyPrefix    = "theme:"
	themeActiveKey    = "theme:active"
	themeSeededKey    = "theme:builtin:seeded"
	defaultThemeID   = "builtin-forest-dark"
)

// handleListThemes returns all stored themes.
//
//	@Summary		List themes
//	@Description	Get all available themes (built-in and custom).
//	@Tags			settings
//	@Produce		json
//	@Success		200	{array}		ThemeDefinition			"List of themes"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes [get]
func (h *Handler) handleListThemes(w http.ResponseWriter, r *http.Request) {
	if err := h.ensureBuiltInThemes(r.Context()); err != nil {
		h.logger.Error("failed to seed built-in themes", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to initialise themes")
		return
	}

	all, err := h.settings.GetAll(r.Context())
	if err != nil {
		h.logger.Error("failed to list settings for themes", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to list themes")
		return
	}

	themes := make([]ThemeDefinition, 0)
	for i := range all {
		key := all[i].Key
		if !strings.HasPrefix(key, themeKeyPrefix) {
			continue
		}
		if key == themeActiveKey || key == themeSeededKey {
			continue
		}
		var td ThemeDefinition
		if err := json.Unmarshal([]byte(all[i].Value), &td); err != nil {
			h.logger.Warn("skipping unparsable theme", zap.String("key", key), zap.Error(err))
			continue
		}
		themes = append(themes, td)
	}

	// Filter by layer if requested.
	if layerFilter := r.URL.Query().Get("layer"); layerFilter != "" {
		filtered := make([]ThemeDefinition, 0, len(themes))
		for i := range themes {
			for _, l := range themes[i].Layers {
				if string(l) == layerFilter {
					filtered = append(filtered, themes[i])
					break
				}
			}
		}
		themes = filtered
	}

	writeJSON(w, http.StatusOK, themes)
}

// handleGetTheme returns a single theme by ID.
//
//	@Summary		Get theme
//	@Description	Get a theme by its ID.
//	@Tags			settings
//	@Produce		json
//	@Param			id	path		string					true	"Theme ID"
//	@Success		200	{object}	ThemeDefinition			"Theme details"
//	@Failure		404	{object}	SettingsProblemDetail	"Theme not found"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/{id} [get]
func (h *Handler) handleGetTheme(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	setting, err := h.settings.Get(r.Context(), themeKeyPrefix+id)
	if err != nil {
		if err == services.ErrNotFound {
			writeSettingsError(w, http.StatusNotFound, "theme not found")
			return
		}
		h.logger.Error("failed to get theme", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get theme")
		return
	}

	var td ThemeDefinition
	if err := json.Unmarshal([]byte(setting.Value), &td); err != nil {
		h.logger.Error("failed to parse theme", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to parse theme")
		return
	}
	writeJSON(w, http.StatusOK, td)
}

// handleCreateTheme creates a new custom theme.
//
//	@Summary		Create theme
//	@Description	Create a new custom theme with CSS token overrides.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ThemeDefinition			true	"Theme definition (id, created_at, updated_at, version, built_in are ignored)"
//	@Success		201		{object}	ThemeDefinition			"Created theme"
//	@Failure		400		{object}	SettingsProblemDetail	"Validation error"
//	@Failure		500		{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes [post]
func (h *Handler) handleCreateTheme(w http.ResponseWriter, r *http.Request) {
	var req ThemeDefinition
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSettingsError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeSettingsError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.BaseMode != "dark" && req.BaseMode != "light" {
		writeSettingsError(w, http.StatusBadRequest, "base_mode must be \"dark\" or \"light\"")
		return
	}

	id, err := generateID()
	if err != nil {
		h.logger.Error("failed to generate theme ID", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to generate theme ID")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	td := ThemeDefinition{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		BaseMode:    req.BaseMode,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		BuiltIn:     false,
		Layers:      req.Layers,
		Tokens:      req.Tokens,
	}

	data, err := json.Marshal(td)
	if err != nil {
		h.logger.Error("failed to marshal theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to store theme")
		return
	}

	if err := h.settings.Set(r.Context(), themeKeyPrefix+td.ID, string(data)); err != nil {
		h.logger.Error("failed to save theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to store theme")
		return
	}

	writeJSON(w, http.StatusCreated, td)
}

// handleUpdateTheme updates an existing custom theme.
//
//	@Summary		Update theme
//	@Description	Update a custom theme. Built-in themes cannot be modified.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Theme ID"
//	@Param			request	body		ThemeDefinition			true	"Fields to update"
//	@Success		200		{object}	ThemeDefinition			"Updated theme"
//	@Failure		403		{object}	SettingsProblemDetail	"Cannot modify built-in theme"
//	@Failure		404		{object}	SettingsProblemDetail	"Theme not found"
//	@Failure		500		{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/{id} [put]
func (h *Handler) handleUpdateTheme(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	setting, err := h.settings.Get(r.Context(), themeKeyPrefix+id)
	if err != nil {
		if err == services.ErrNotFound {
			writeSettingsError(w, http.StatusNotFound, "theme not found")
			return
		}
		h.logger.Error("failed to get theme for update", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get theme")
		return
	}

	var existing ThemeDefinition
	if err := json.Unmarshal([]byte(setting.Value), &existing); err != nil {
		h.logger.Error("failed to parse theme for update", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to parse theme")
		return
	}

	if existing.BuiltIn {
		writeSettingsError(w, http.StatusForbidden, "cannot modify built-in theme")
		return
	}

	var patch ThemeDefinition
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeSettingsError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Merge only provided fields.
	if patch.Name != "" {
		existing.Name = patch.Name
	}
	if patch.Description != "" {
		existing.Description = patch.Description
	}
	if patch.BaseMode != "" {
		if patch.BaseMode != "dark" && patch.BaseMode != "light" {
			writeSettingsError(w, http.StatusBadRequest, "base_mode must be \"dark\" or \"light\"")
			return
		}
		existing.BaseMode = patch.BaseMode
	}
	if patch.Tokens.Backgrounds != nil {
		existing.Tokens.Backgrounds = patch.Tokens.Backgrounds
	}
	if patch.Tokens.Text != nil {
		existing.Tokens.Text = patch.Tokens.Text
	}
	if patch.Tokens.Borders != nil {
		existing.Tokens.Borders = patch.Tokens.Borders
	}
	if patch.Tokens.Buttons != nil {
		existing.Tokens.Buttons = patch.Tokens.Buttons
	}
	if patch.Tokens.Inputs != nil {
		existing.Tokens.Inputs = patch.Tokens.Inputs
	}
	if patch.Tokens.Sidebar != nil {
		existing.Tokens.Sidebar = patch.Tokens.Sidebar
	}
	if patch.Tokens.Status != nil {
		existing.Tokens.Status = patch.Tokens.Status
	}
	if patch.Tokens.Charts != nil {
		existing.Tokens.Charts = patch.Tokens.Charts
	}
	if patch.Tokens.Typography != nil {
		existing.Tokens.Typography = patch.Tokens.Typography
	}
	if patch.Tokens.Spacing != nil {
		existing.Tokens.Spacing = patch.Tokens.Spacing
	}
	if patch.Tokens.Effects != nil {
		existing.Tokens.Effects = patch.Tokens.Effects
	}

	existing.Version++
	existing.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.Marshal(existing)
	if err != nil {
		h.logger.Error("failed to marshal updated theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to store theme")
		return
	}

	if err := h.settings.Set(r.Context(), themeKeyPrefix+id, string(data)); err != nil {
		h.logger.Error("failed to save updated theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to store theme")
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// handleDeleteTheme deletes a custom theme.
//
//	@Summary		Delete theme
//	@Description	Delete a custom theme. Built-in themes cannot be deleted.
//	@Tags			settings
//	@Param			id	path	string	true	"Theme ID"
//	@Success		204	"Theme deleted"
//	@Failure		403	{object}	SettingsProblemDetail	"Cannot delete built-in theme"
//	@Failure		404	{object}	SettingsProblemDetail	"Theme not found"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/{id} [delete]
func (h *Handler) handleDeleteTheme(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	setting, err := h.settings.Get(r.Context(), themeKeyPrefix+id)
	if err != nil {
		if err == services.ErrNotFound {
			writeSettingsError(w, http.StatusNotFound, "theme not found")
			return
		}
		h.logger.Error("failed to get theme for delete", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get theme")
		return
	}

	var existing ThemeDefinition
	if err := json.Unmarshal([]byte(setting.Value), &existing); err != nil {
		h.logger.Error("failed to parse theme for delete", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to parse theme")
		return
	}

	if existing.BuiltIn {
		writeSettingsError(w, http.StatusForbidden, "cannot delete built-in theme")
		return
	}

	if err := h.settings.Delete(r.Context(), themeKeyPrefix+id); err != nil {
		h.logger.Error("failed to delete theme", zap.String("id", id), zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to delete theme")
		return
	}

	// If the deleted theme was the active theme, reset to default.
	active, err := h.settings.Get(r.Context(), themeActiveKey)
	if err == nil && active.Value == id {
		_ = h.settings.Set(r.Context(), themeActiveKey, defaultThemeID)
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetActiveTheme returns the currently active theme ID.
//
//	@Summary		Get active theme
//	@Description	Get the ID of the currently active theme.
//	@Tags			settings
//	@Produce		json
//	@Success		200	{object}	ActiveThemeResponse		"Active theme ID"
//	@Failure		500	{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/active [get]
func (h *Handler) handleGetActiveTheme(w http.ResponseWriter, r *http.Request) {
	setting, err := h.settings.Get(r.Context(), themeActiveKey)
	if err != nil {
		if err == services.ErrNotFound {
			writeJSON(w, http.StatusOK, ActiveThemeResponse{ThemeID: defaultThemeID})
			return
		}
		h.logger.Error("failed to get active theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to get active theme")
		return
	}
	writeJSON(w, http.StatusOK, ActiveThemeResponse{ThemeID: setting.Value})
}

// handleSetActiveTheme sets the active theme.
//
//	@Summary		Set active theme
//	@Description	Set which theme is currently active.
//	@Tags			settings
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ActiveThemeRequest		true	"Theme ID to activate"
//	@Success		200		{object}	ActiveThemeResponse		"Active theme set"
//	@Failure		400		{object}	SettingsProblemDetail	"Validation error"
//	@Failure		404		{object}	SettingsProblemDetail	"Theme not found"
//	@Failure		500		{object}	SettingsProblemDetail	"Internal server error"
//	@Router			/settings/themes/active [put]
func (h *Handler) handleSetActiveTheme(w http.ResponseWriter, r *http.Request) {
	var req ActiveThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSettingsError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.ThemeID) == "" {
		writeSettingsError(w, http.StatusBadRequest, "theme_id is required")
		return
	}

	// Ensure built-in themes are seeded (setup wizard sets theme before listing).
	if err := h.ensureBuiltInThemes(r.Context()); err != nil {
		h.logger.Error("failed to ensure built-in themes", zap.Error(err))
	}

	// Verify the theme exists.
	if _, err := h.settings.Get(r.Context(), themeKeyPrefix+req.ThemeID); err != nil {
		if err == services.ErrNotFound {
			writeSettingsError(w, http.StatusNotFound, "theme not found")
			return
		}
		h.logger.Error("failed to verify theme for activation", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to verify theme")
		return
	}

	if err := h.settings.Set(r.Context(), themeActiveKey, req.ThemeID); err != nil {
		h.logger.Error("failed to set active theme", zap.Error(err))
		writeSettingsError(w, http.StatusInternalServerError, "failed to set active theme")
		return
	}

	writeJSON(w, http.StatusOK, ActiveThemeResponse(req))
}

// ensureBuiltInThemes seeds built-in themes, adding any new ones that don't exist yet.
func (h *Handler) ensureBuiltInThemes(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)
	builtins := []ThemeDefinition{
		// --- Full theme packs (all layers via CSS defaults) ---
		{
			ID:          "builtin-forest-dark",
			Name:        "Forest Dark",
			Description: "Default dark theme with forest green accents.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      allLayers,
			Tokens:      ThemeTokens{},
		},
		{
			ID:          "builtin-forest-light",
			Name:        "Forest Light",
			Description: "Light theme with forest green accents.",
			BaseMode:    "light",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      allLayers,
			Tokens:      ThemeTokens{},
		},
		// --- Color palette themes (colors + effects) ---
		{
			ID:          "builtin-navy-copper",
			Name:        "Navy Copper",
			Description: "Dark navy theme with copper accents and a natural green palette.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerColors, LayerEffects},
			Tokens:      navyCopperTokens(),
		},
		{
			ID:          "builtin-classic-dark",
			Name:        "Classic Dark",
			Description: "Neutral dark theme with slate grays and blue accents.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerColors, LayerEffects},
			Tokens:      classicDarkTokens(),
		},
		{
			ID:          "builtin-classic-light",
			Name:        "Classic Light",
			Description: "Clean light theme with neutral grays and blue accents.",
			BaseMode:    "light",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerColors, LayerEffects},
			Tokens:      classicLightTokens(),
		},
		{
			ID:          "builtin-sunset-teal",
			Name:        "Sunset Teal",
			Description: "Dark teal backgrounds with warm gold and coral accents.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerColors, LayerEffects},
			Tokens:      sunsetTealTokens(),
		},
		{
			ID:          "builtin-navy-gold",
			Name:        "Navy Gold",
			Description: "Deep navy with bright gold accents for high contrast monitoring.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerColors, LayerEffects},
			Tokens:      navyGoldTokens(),
		},
		{
			ID:          "builtin-nordic",
			Name:        "Nordic",
			Description: "Cool slate grays with bold red accents for alert-focused monitoring.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerColors, LayerEffects},
			Tokens:      nordicTokens(),
		},
		{
			ID:          "builtin-ocean-blue",
			Name:        "Ocean Blue",
			Description: "Clean light theme with an ocean-inspired blue gradient palette.",
			BaseMode:    "light",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerColors, LayerEffects},
			Tokens:      oceanBlueTokens(),
		},
		{
			ID:          "builtin-sage-berry",
			Name:        "Sage & Berry",
			Description: "Warm cream and sage with berry red accents for a natural look.",
			BaseMode:    "light",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerColors, LayerEffects},
			Tokens:      sageBerryTokens(),
		},
		// --- Typography presets ---
		{
			ID:          "builtin-type-system",
			Name:        "System Default",
			Description: "System UI font stack with JetBrains Mono for code.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerTypography},
			Tokens: ThemeTokens{Typography: map[string]string{
				"font-sans": "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'Inter', Helvetica, Arial, sans-serif",
				"font-mono": "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace",
			}},
		},
		{
			ID:          "builtin-type-inter",
			Name:        "Inter",
			Description: "Inter for UI text, Fira Code for code blocks.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerTypography},
			Tokens: ThemeTokens{Typography: map[string]string{
				"font-sans": "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif",
				"font-mono": "'Fira Code', 'JetBrains Mono', 'Cascadia Code', monospace",
			}},
		},
		{
			ID:          "builtin-type-mono",
			Name:        "Monospace",
			Description: "Monospace everywhere for a terminal-like experience.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerTypography},
			Tokens: ThemeTokens{Typography: map[string]string{
				"font-sans": "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace",
				"font-mono": "'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace",
			}},
		},
		// --- Shape presets ---
		{
			ID:          "builtin-shape-rounded",
			Name:        "Rounded",
			Description: "Standard rounded corners.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerShape},
			Tokens: ThemeTokens{Spacing: map[string]string{
				"radius-sm": "4px", "radius-md": "8px", "radius-lg": "12px", "radius-xl": "16px",
			}},
		},
		{
			ID:          "builtin-shape-sharp",
			Name:        "Sharp",
			Description: "Minimal rounding for a clean, angular look.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerShape},
			Tokens: ThemeTokens{Spacing: map[string]string{
				"radius-sm": "2px", "radius-md": "4px", "radius-lg": "6px", "radius-xl": "8px",
			}},
		},
		{
			ID:          "builtin-shape-pill",
			Name:        "Pill",
			Description: "Fully rounded elements for a softer interface.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerShape},
			Tokens: ThemeTokens{Spacing: map[string]string{
				"radius-sm": "8px", "radius-md": "12px", "radius-lg": "20px", "radius-xl": "9999px",
			}},
		},
		// --- Effects presets ---
		{
			ID:          "builtin-fx-standard",
			Name:        "Standard",
			Description: "Moderate shadows and smooth transitions.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerEffects},
			Tokens: ThemeTokens{Effects: map[string]string{
				"shadow-sm": "0 1px 2px rgba(0, 0, 0, 0.3)", "shadow-md": "0 4px 6px rgba(0, 0, 0, 0.25)",
				"shadow-lg": "0 10px 25px rgba(0, 0, 0, 0.3)", "shadow-glow": "0 0 20px rgba(74, 222, 128, 0.15)",
				"transition-fast": "150ms ease", "transition-normal": "250ms ease", "transition-slow": "350ms ease",
			}},
		},
		{
			ID:          "builtin-fx-flat",
			Name:        "Flat",
			Description: "No shadows, instant transitions for a flat design.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerEffects},
			Tokens: ThemeTokens{Effects: map[string]string{
				"shadow-sm": "none", "shadow-md": "none",
				"shadow-lg": "none", "shadow-glow": "none",
				"transition-fast": "0ms", "transition-normal": "0ms", "transition-slow": "0ms",
			}},
		},
		{
			ID:          "builtin-fx-dramatic",
			Name:        "Dramatic",
			Description: "Deep shadows and glow effects for a cinematic feel.",
			BaseMode:    "dark",
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
			BuiltIn:     true,
			Layers:      []ThemeLayer{LayerEffects},
			Tokens: ThemeTokens{Effects: map[string]string{
				"shadow-sm": "0 2px 4px rgba(0, 0, 0, 0.4)", "shadow-md": "0 8px 16px rgba(0, 0, 0, 0.4)",
				"shadow-lg": "0 16px 40px rgba(0, 0, 0, 0.5)", "shadow-glow": "0 0 30px rgba(74, 222, 128, 0.25)",
				"transition-fast": "200ms cubic-bezier(0.4, 0, 0.2, 1)", "transition-normal": "350ms cubic-bezier(0.4, 0, 0.2, 1)",
				"transition-slow": "500ms cubic-bezier(0.4, 0, 0.2, 1)",
			}},
		},
	}

	for i := range builtins {
		// Skip themes that already exist.
		if _, err := h.settings.Get(ctx, themeKeyPrefix+builtins[i].ID); err == nil {
			continue
		}
		data, err := json.Marshal(builtins[i])
		if err != nil {
			return fmt.Errorf("marshal built-in theme %s: %w", builtins[i].ID, err)
		}
		if err := h.settings.Set(ctx, themeKeyPrefix+builtins[i].ID, string(data)); err != nil {
			return fmt.Errorf("save built-in theme %s: %w", builtins[i].ID, err)
		}
	}

	// Keep the seeded marker for backward compatibility.
	if _, err := h.settings.Get(ctx, themeSeededKey); err != nil {
		return h.settings.Set(ctx, themeSeededKey, "true")
	}
	return nil
}

// navyCopperTokens returns the complete token overrides for the Navy Copper theme.
func navyCopperTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root":     "#0D2238",
			"bg-surface":  "#122E4E",
			"bg-card":     "#1A4674",
			"bg-elevated": "#1E5080",
			"bg-hover":    "rgba(98, 203, 100, 0.06)",
			"bg-active":   "rgba(98, 203, 100, 0.10)",
			"bg-selected": "rgba(98, 203, 100, 0.08)",
		},
		Text: map[string]string{
			"text-primary":   "#E8EDF2",
			"text-secondary": "#8BA87A",
			"text-muted":     "#658646",
			"text-accent":    "#62CB64",
			"text-warm":      "#D59958",
			"text-inverse":   "#0D2238",
		},
		Borders: map[string]string{
			"border-subtle":  "rgba(98, 203, 100, 0.08)",
			"border-default": "rgba(98, 203, 100, 0.15)",
			"border-strong":  "rgba(98, 203, 100, 0.25)",
			"border-focus":   "#62CB64",
		},
		Buttons: map[string]string{
			"btn-primary-bg":    "#62CB64",
			"btn-primary-hover": "#78D67A",
			"btn-primary-text":  "#0D2238",
			"btn-danger-bg":     "#991b1b",
			"btn-danger-hover":  "#b91c1c",
			"btn-danger-text":   "#fecaca",
		},
		Inputs: map[string]string{
			"input-bg":          "#0D2238",
			"input-border":      "rgba(98, 203, 100, 0.15)",
			"input-focus":       "#62CB64",
			"input-text":        "#E8EDF2",
			"input-placeholder": "#658646",
		},
		Sidebar: map[string]string{
			"sidebar-bg":        "#0D2238",
			"sidebar-item":      "#8BA87A",
			"sidebar-active":    "#62CB64",
			"sidebar-active-bg": "rgba(98, 203, 100, 0.10)",
		},
		Status: map[string]string{
			"status-online":   "#62CB64",
			"status-degraded": "#D59958",
			"status-offline":  "#ef4444",
			"status-unknown":  "#658646",
		},
		Charts: map[string]string{
			"chart-green": "#62CB64",
			"chart-amber": "#D59958",
			"chart-sage":  "#658646",
			"chart-red":   "#ef4444",
			"chart-blue":  "#4A90D9",
			"chart-grid":  "rgba(98, 203, 100, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm":   "0 1px 2px rgba(0, 0, 0, 0.4)",
			"shadow-md":   "0 4px 6px rgba(0, 0, 0, 0.35)",
			"shadow-lg":   "0 10px 25px rgba(0, 0, 0, 0.4)",
			"shadow-glow": "0 0 20px rgba(98, 203, 100, 0.12)",
		},
	}
}

// classicDarkTokens returns the complete token overrides for the Classic Dark theme.
func classicDarkTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root":     "#1a1b26",
			"bg-surface":  "#1e1f2b",
			"bg-card":     "#252736",
			"bg-elevated": "#2a2b3d",
			"bg-hover":    "rgba(91, 156, 246, 0.06)",
			"bg-active":   "rgba(91, 156, 246, 0.10)",
			"bg-selected": "rgba(91, 156, 246, 0.08)",
		},
		Text: map[string]string{
			"text-primary":   "#e1e2e7",
			"text-secondary": "#a0a4b8",
			"text-muted":     "#6b6f85",
			"text-accent":    "#5b9cf6",
			"text-warm":      "#e6a855",
			"text-inverse":   "#1a1b26",
		},
		Borders: map[string]string{
			"border-subtle":  "rgba(91, 156, 246, 0.08)",
			"border-default": "rgba(91, 156, 246, 0.15)",
			"border-strong":  "rgba(91, 156, 246, 0.25)",
			"border-focus":   "#5b9cf6",
		},
		Buttons: map[string]string{
			"btn-primary-bg":    "#5b9cf6",
			"btn-primary-hover": "#7ab3f8",
			"btn-primary-text":  "#ffffff",
			"btn-danger-bg":     "#991b1b",
			"btn-danger-hover":  "#b91c1c",
			"btn-danger-text":   "#fecaca",
		},
		Inputs: map[string]string{
			"input-bg":          "#1a1b26",
			"input-border":      "rgba(91, 156, 246, 0.15)",
			"input-focus":       "#5b9cf6",
			"input-text":        "#e1e2e7",
			"input-placeholder": "#6b6f85",
		},
		Sidebar: map[string]string{
			"sidebar-bg":        "#1a1b26",
			"sidebar-item":      "#a0a4b8",
			"sidebar-active":    "#5b9cf6",
			"sidebar-active-bg": "rgba(91, 156, 246, 0.10)",
		},
		Status: map[string]string{
			"status-online":   "#4ade80",
			"status-degraded": "#e6a855",
			"status-offline":  "#ef4444",
			"status-unknown":  "#a0a4b8",
		},
		Charts: map[string]string{
			"chart-green": "#4ade80",
			"chart-amber": "#e6a855",
			"chart-sage":  "#a0a4b8",
			"chart-red":   "#ef4444",
			"chart-blue":  "#5b9cf6",
			"chart-grid":  "rgba(91, 156, 246, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm":   "0 1px 2px rgba(0, 0, 0, 0.4)",
			"shadow-md":   "0 4px 6px rgba(0, 0, 0, 0.35)",
			"shadow-lg":   "0 10px 25px rgba(0, 0, 0, 0.4)",
			"shadow-glow": "0 0 20px rgba(91, 156, 246, 0.12)",
		},
	}
}

// classicLightTokens returns the complete token overrides for the Classic Light theme.
func classicLightTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root":     "#f6f8fa",
			"bg-surface":  "#eef1f5",
			"bg-card":     "#ffffff",
			"bg-elevated": "#ffffff",
			"bg-hover":    "rgba(9, 105, 218, 0.04)",
			"bg-active":   "rgba(9, 105, 218, 0.08)",
			"bg-selected": "rgba(9, 105, 218, 0.06)",
		},
		Text: map[string]string{
			"text-primary":   "#1f2328",
			"text-secondary": "#656d76",
			"text-muted":     "#8b949e",
			"text-accent":    "#0969da",
			"text-warm":      "#bf8700",
			"text-inverse":   "#f6f8fa",
		},
		Borders: map[string]string{
			"border-subtle":  "rgba(9, 105, 218, 0.06)",
			"border-default": "rgba(9, 105, 218, 0.15)",
			"border-strong":  "rgba(9, 105, 218, 0.25)",
			"border-focus":   "#0969da",
		},
		Buttons: map[string]string{
			"btn-primary-bg":    "#0969da",
			"btn-primary-hover": "#0757b5",
			"btn-primary-text":  "#ffffff",
			"btn-danger-bg":     "#cf222e",
			"btn-danger-hover":  "#a40e26",
			"btn-danger-text":   "#ffffff",
		},
		Inputs: map[string]string{
			"input-bg":          "#ffffff",
			"input-border":      "rgba(9, 105, 218, 0.15)",
			"input-focus":       "#0969da",
			"input-text":        "#1f2328",
			"input-placeholder": "#8b949e",
		},
		Sidebar: map[string]string{
			"sidebar-bg":        "#eef1f5",
			"sidebar-item":      "#656d76",
			"sidebar-active":    "#0969da",
			"sidebar-active-bg": "rgba(9, 105, 218, 0.08)",
		},
		Status: map[string]string{
			"status-online":   "#1a7f37",
			"status-degraded": "#bf8700",
			"status-offline":  "#cf222e",
			"status-unknown":  "#8b949e",
		},
		Charts: map[string]string{
			"chart-green": "#1a7f37",
			"chart-amber": "#bf8700",
			"chart-sage":  "#8b949e",
			"chart-red":   "#cf222e",
			"chart-blue":  "#0969da",
			"chart-grid":  "rgba(9, 105, 218, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm":   "0 1px 2px rgba(0, 0, 0, 0.05)",
			"shadow-md":   "0 4px 6px rgba(0, 0, 0, 0.07)",
			"shadow-lg":   "0 10px 25px rgba(0, 0, 0, 0.1)",
			"shadow-glow": "0 0 20px rgba(9, 105, 218, 0.08)",
		},
	}
}

// sunsetTealTokens returns tokens for the Sunset Teal dark theme.
// Palette: #264653, #2A9D8F, #E9C46A, #F4A261, #E76F51
func sunsetTealTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root": "#1A2F36", "bg-surface": "#1F3840",
			"bg-card": "#264653", "bg-elevated": "#2D5260",
			"bg-hover": "rgba(42, 157, 143, 0.06)", "bg-active": "rgba(42, 157, 143, 0.10)",
			"bg-selected": "rgba(42, 157, 143, 0.08)",
		},
		Text: map[string]string{
			"text-primary": "#F0EDE5", "text-secondary": "#A8C4BE",
			"text-muted": "#5E8A82", "text-accent": "#2A9D8F",
			"text-warm": "#E9C46A", "text-inverse": "#1A2F36",
		},
		Borders: map[string]string{
			"border-subtle": "rgba(42, 157, 143, 0.08)", "border-default": "rgba(42, 157, 143, 0.15)",
			"border-strong": "rgba(42, 157, 143, 0.25)", "border-focus": "#2A9D8F",
		},
		Buttons: map[string]string{
			"btn-primary-bg": "#2A9D8F", "btn-primary-hover": "#35B4A5",
			"btn-primary-text": "#1A2F36", "btn-danger-bg": "#C84B31",
			"btn-danger-hover": "#E76F51", "btn-danger-text": "#FDE8E0",
		},
		Inputs: map[string]string{
			"input-bg": "#1A2F36", "input-border": "rgba(42, 157, 143, 0.15)",
			"input-focus": "#2A9D8F", "input-text": "#F0EDE5", "input-placeholder": "#5E8A82",
		},
		Sidebar: map[string]string{
			"sidebar-bg": "#1A2F36", "sidebar-item": "#A8C4BE",
			"sidebar-active": "#2A9D8F", "sidebar-active-bg": "rgba(42, 157, 143, 0.10)",
		},
		Status: map[string]string{
			"status-online": "#2A9D8F", "status-degraded": "#E9C46A",
			"status-offline": "#E76F51", "status-unknown": "#5E8A82",
		},
		Charts: map[string]string{
			"chart-green": "#2A9D8F", "chart-amber": "#E9C46A",
			"chart-sage": "#A8C4BE", "chart-red": "#E76F51",
			"chart-blue": "#4A90D9", "chart-grid": "rgba(42, 157, 143, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm": "0 1px 2px rgba(0, 0, 0, 0.4)", "shadow-md": "0 4px 6px rgba(0, 0, 0, 0.35)",
			"shadow-lg": "0 10px 25px rgba(0, 0, 0, 0.4)", "shadow-glow": "0 0 20px rgba(42, 157, 143, 0.12)",
		},
	}
}

// navyGoldTokens returns tokens for the Navy Gold dark theme.
// Palette: #000814, #001D3D, #003566, #FFC300, #FFD60A
func navyGoldTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root": "#000814", "bg-surface": "#001126",
			"bg-card": "#001D3D", "bg-elevated": "#002952",
			"bg-hover": "rgba(255, 195, 0, 0.06)", "bg-active": "rgba(255, 195, 0, 0.10)",
			"bg-selected": "rgba(255, 195, 0, 0.08)",
		},
		Text: map[string]string{
			"text-primary": "#F0EDE5", "text-secondary": "#8BA4C4",
			"text-muted": "#4A6A8A", "text-accent": "#FFC300",
			"text-warm": "#FFD60A", "text-inverse": "#000814",
		},
		Borders: map[string]string{
			"border-subtle": "rgba(255, 195, 0, 0.08)", "border-default": "rgba(255, 195, 0, 0.15)",
			"border-strong": "rgba(255, 195, 0, 0.25)", "border-focus": "#FFC300",
		},
		Buttons: map[string]string{
			"btn-primary-bg": "#FFC300", "btn-primary-hover": "#FFD60A",
			"btn-primary-text": "#000814", "btn-danger-bg": "#991b1b",
			"btn-danger-hover": "#b91c1c", "btn-danger-text": "#fecaca",
		},
		Inputs: map[string]string{
			"input-bg": "#000814", "input-border": "rgba(255, 195, 0, 0.15)",
			"input-focus": "#FFC300", "input-text": "#F0EDE5", "input-placeholder": "#4A6A8A",
		},
		Sidebar: map[string]string{
			"sidebar-bg": "#000814", "sidebar-item": "#8BA4C4",
			"sidebar-active": "#FFC300", "sidebar-active-bg": "rgba(255, 195, 0, 0.10)",
		},
		Status: map[string]string{
			"status-online": "#4ade80", "status-degraded": "#FFC300",
			"status-offline": "#ef4444", "status-unknown": "#4A6A8A",
		},
		Charts: map[string]string{
			"chart-green": "#4ade80", "chart-amber": "#FFC300",
			"chart-sage": "#8BA4C4", "chart-red": "#ef4444",
			"chart-blue": "#003566", "chart-grid": "rgba(255, 195, 0, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm": "0 1px 2px rgba(0, 0, 0, 0.5)", "shadow-md": "0 4px 6px rgba(0, 0, 0, 0.45)",
			"shadow-lg": "0 10px 25px rgba(0, 0, 0, 0.5)", "shadow-glow": "0 0 20px rgba(255, 195, 0, 0.12)",
		},
	}
}

// nordicTokens returns tokens for the Nordic dark theme.
// Palette: #2B2D42, #8D99AE, #EDF2F4, #EF233C, #D90429
func nordicTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root": "#1E1F30", "bg-surface": "#242538",
			"bg-card": "#2B2D42", "bg-elevated": "#33354E",
			"bg-hover": "rgba(141, 153, 174, 0.06)", "bg-active": "rgba(141, 153, 174, 0.10)",
			"bg-selected": "rgba(141, 153, 174, 0.08)",
		},
		Text: map[string]string{
			"text-primary": "#EDF2F4", "text-secondary": "#8D99AE",
			"text-muted": "#5A6278", "text-accent": "#EF233C",
			"text-warm": "#F8A825", "text-inverse": "#1E1F30",
		},
		Borders: map[string]string{
			"border-subtle": "rgba(141, 153, 174, 0.08)", "border-default": "rgba(141, 153, 174, 0.15)",
			"border-strong": "rgba(141, 153, 174, 0.25)", "border-focus": "#EF233C",
		},
		Buttons: map[string]string{
			"btn-primary-bg": "#EF233C", "btn-primary-hover": "#F5405A",
			"btn-primary-text": "#FFFFFF", "btn-danger-bg": "#D90429",
			"btn-danger-hover": "#EF233C", "btn-danger-text": "#FDE8EA",
		},
		Inputs: map[string]string{
			"input-bg": "#1E1F30", "input-border": "rgba(141, 153, 174, 0.15)",
			"input-focus": "#EF233C", "input-text": "#EDF2F4", "input-placeholder": "#5A6278",
		},
		Sidebar: map[string]string{
			"sidebar-bg": "#1E1F30", "sidebar-item": "#8D99AE",
			"sidebar-active": "#EF233C", "sidebar-active-bg": "rgba(239, 35, 60, 0.10)",
		},
		Status: map[string]string{
			"status-online": "#4ade80", "status-degraded": "#F8A825",
			"status-offline": "#D90429", "status-unknown": "#8D99AE",
		},
		Charts: map[string]string{
			"chart-green": "#4ade80", "chart-amber": "#F8A825",
			"chart-sage": "#8D99AE", "chart-red": "#D90429",
			"chart-blue": "#60a5fa", "chart-grid": "rgba(141, 153, 174, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm": "0 1px 2px rgba(0, 0, 0, 0.4)", "shadow-md": "0 4px 6px rgba(0, 0, 0, 0.35)",
			"shadow-lg": "0 10px 25px rgba(0, 0, 0, 0.4)", "shadow-glow": "0 0 20px rgba(239, 35, 60, 0.12)",
		},
	}
}

// oceanBlueTokens returns tokens for the Ocean Blue light theme.
// Palette: #03045E, #0077B6, #00B4D8, #90E0EF, #CAF0F8
func oceanBlueTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root": "#F0F9FC", "bg-surface": "#E4F4F9",
			"bg-card": "#FFFFFF", "bg-elevated": "#FFFFFF",
			"bg-hover": "rgba(0, 119, 182, 0.04)", "bg-active": "rgba(0, 119, 182, 0.08)",
			"bg-selected": "rgba(0, 119, 182, 0.06)",
		},
		Text: map[string]string{
			"text-primary": "#03045E", "text-secondary": "#3A6B8C",
			"text-muted": "#7BADC4", "text-accent": "#0077B6",
			"text-warm": "#B8860B", "text-inverse": "#F0F9FC",
		},
		Borders: map[string]string{
			"border-subtle": "rgba(0, 119, 182, 0.06)", "border-default": "rgba(0, 119, 182, 0.15)",
			"border-strong": "rgba(0, 119, 182, 0.25)", "border-focus": "#0077B6",
		},
		Buttons: map[string]string{
			"btn-primary-bg": "#0077B6", "btn-primary-hover": "#005F8F",
			"btn-primary-text": "#FFFFFF", "btn-danger-bg": "#cf222e",
			"btn-danger-hover": "#a40e26", "btn-danger-text": "#FFFFFF",
		},
		Inputs: map[string]string{
			"input-bg": "#FFFFFF", "input-border": "rgba(0, 119, 182, 0.15)",
			"input-focus": "#0077B6", "input-text": "#03045E", "input-placeholder": "#7BADC4",
		},
		Sidebar: map[string]string{
			"sidebar-bg": "#E4F4F9", "sidebar-item": "#3A6B8C",
			"sidebar-active": "#0077B6", "sidebar-active-bg": "rgba(0, 119, 182, 0.08)",
		},
		Status: map[string]string{
			"status-online": "#1a7f37", "status-degraded": "#B8860B",
			"status-offline": "#cf222e", "status-unknown": "#7BADC4",
		},
		Charts: map[string]string{
			"chart-green": "#1a7f37", "chart-amber": "#B8860B",
			"chart-sage": "#7BADC4", "chart-red": "#cf222e",
			"chart-blue": "#0077B6", "chart-grid": "rgba(0, 119, 182, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm": "0 1px 2px rgba(0, 0, 0, 0.05)", "shadow-md": "0 4px 6px rgba(0, 0, 0, 0.07)",
			"shadow-lg": "0 10px 25px rgba(0, 0, 0, 0.1)", "shadow-glow": "0 0 20px rgba(0, 119, 182, 0.08)",
		},
	}
}

// sageBerryTokens returns tokens for the Sage & Berry light theme.
// Palette: #386641, #6A994E, #A7C957, #F2E8CF, #BC4749
func sageBerryTokens() ThemeTokens {
	return ThemeTokens{
		Backgrounds: map[string]string{
			"bg-root": "#F7F2E8", "bg-surface": "#F2E8CF",
			"bg-card": "#FFFFFF", "bg-elevated": "#FFFFFF",
			"bg-hover": "rgba(106, 153, 78, 0.04)", "bg-active": "rgba(106, 153, 78, 0.08)",
			"bg-selected": "rgba(106, 153, 78, 0.06)",
		},
		Text: map[string]string{
			"text-primary": "#2A3B2D", "text-secondary": "#5A7A4A",
			"text-muted": "#8AA87A", "text-accent": "#386641",
			"text-warm": "#8B6914", "text-inverse": "#F7F2E8",
		},
		Borders: map[string]string{
			"border-subtle": "rgba(106, 153, 78, 0.06)", "border-default": "rgba(106, 153, 78, 0.15)",
			"border-strong": "rgba(106, 153, 78, 0.25)", "border-focus": "#6A994E",
		},
		Buttons: map[string]string{
			"btn-primary-bg": "#386641", "btn-primary-hover": "#2D5235",
			"btn-primary-text": "#FFFFFF", "btn-danger-bg": "#BC4749",
			"btn-danger-hover": "#A03638", "btn-danger-text": "#FFFFFF",
		},
		Inputs: map[string]string{
			"input-bg": "#FFFFFF", "input-border": "rgba(106, 153, 78, 0.15)",
			"input-focus": "#6A994E", "input-text": "#2A3B2D", "input-placeholder": "#8AA87A",
		},
		Sidebar: map[string]string{
			"sidebar-bg": "#F2E8CF", "sidebar-item": "#5A7A4A",
			"sidebar-active": "#386641", "sidebar-active-bg": "rgba(56, 102, 65, 0.08)",
		},
		Status: map[string]string{
			"status-online": "#6A994E", "status-degraded": "#D4A017",
			"status-offline": "#BC4749", "status-unknown": "#8AA87A",
		},
		Charts: map[string]string{
			"chart-green": "#6A994E", "chart-amber": "#D4A017",
			"chart-sage": "#8AA87A", "chart-red": "#BC4749",
			"chart-blue": "#4A90D9", "chart-grid": "rgba(106, 153, 78, 0.06)",
		},
		Effects: map[string]string{
			"shadow-sm": "0 1px 2px rgba(0, 0, 0, 0.05)", "shadow-md": "0 4px 6px rgba(0, 0, 0, 0.07)",
			"shadow-lg": "0 10px 25px rgba(0, 0, 0, 0.1)", "shadow-glow": "0 0 20px rgba(106, 153, 78, 0.08)",
		},
	}
}

// generateID returns a random 32-character hex string suitable for use as a theme ID.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
