package settings_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HerbHall/subnetree/internal/services"
	"github.com/HerbHall/subnetree/internal/settings"
	"github.com/HerbHall/subnetree/internal/testutil"
	"go.uber.org/zap"
)

func setupHandlerEnv(t *testing.T) (*settings.Handler, *http.ServeMux) {
	t.Helper()

	store := testutil.NewStore(t)
	repo, err := services.NewSQLiteSettingsRepository(context.Background(), store)
	if err != nil {
		t.Fatalf("NewSQLiteSettingsRepository: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	handler := settings.NewHandler(repo, logger)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return handler, mux
}

func doRequest(mux *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHandleListInterfaces(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "GET", "/api/v1/settings/interfaces", nil)

	if w.Code != http.StatusOK {
		t.Errorf("ListInterfaces status = %d, want %d", w.Code, http.StatusOK)
	}

	var interfaces []services.NetworkInterface
	if err := json.NewDecoder(w.Body).Decode(&interfaces); err != nil {
		t.Fatalf("Decode response: %v", err)
	}

	// We can't assert specific interfaces exist as they depend on the test environment,
	// but we can verify the response structure is valid JSON array
	t.Logf("Found %d interfaces", len(interfaces))
}

func TestHandleGetScanInterface_NotConfigured(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "GET", "/api/v1/settings/scan-interface", nil)

	if w.Code != http.StatusOK {
		t.Errorf("GetScanInterface status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		InterfaceName string `json:"interface_name"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode response: %v", err)
	}

	if resp.InterfaceName != "" {
		t.Errorf("InterfaceName = %q, want empty string", resp.InterfaceName)
	}
}

func TestHandleSetScanInterface_InvalidInterface(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "POST", "/api/v1/settings/scan-interface", map[string]string{
		"interface_name": "nonexistent_interface_xyz",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("SetScanInterface with invalid interface status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSetScanInterface_EmptyInterface(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Setting empty interface should succeed (resets to auto-detect)
	w := doRequest(mux, "POST", "/api/v1/settings/scan-interface", map[string]string{
		"interface_name": "",
	})

	if w.Code != http.StatusOK {
		t.Errorf("SetScanInterface with empty interface status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleSetScanInterface_InvalidBody(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Send invalid JSON
	req := httptest.NewRequest("POST", "/api/v1/settings/scan-interface", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SetScanInterface with invalid body status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------- Theme endpoint tests ----------

func TestHandleListThemes(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "GET", "/api/v1/settings/themes", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("ListThemes status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var themes []settings.ThemeDefinition
	if err := json.NewDecoder(w.Body).Decode(&themes); err != nil {
		t.Fatalf("Decode response: %v", err)
	}

	if len(themes) < 2 {
		t.Fatalf("expected at least 2 built-in themes, got %d", len(themes))
	}

	ids := map[string]bool{}
	for i := range themes {
		ids[themes[i].ID] = true
	}
	if !ids["builtin-forest-dark"] {
		t.Error("missing built-in theme builtin-forest-dark")
	}
	if !ids["builtin-forest-light"] {
		t.Error("missing built-in theme builtin-forest-light")
	}
}

func TestHandleCreateTheme(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	body := map[string]any{
		"name":        "Cyberpunk",
		"description": "Neon lights",
		"base_mode":   "dark",
		"tokens": map[string]any{
			"backgrounds": map[string]string{"primary": "#0a0a2e"},
		},
	}
	w := doRequest(mux, "POST", "/api/v1/settings/themes", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("CreateTheme status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created settings.ThemeDefinition
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("Decode response: %v", err)
	}

	if created.ID == "" {
		t.Error("created theme has empty ID")
	}
	if created.Name != "Cyberpunk" {
		t.Errorf("Name = %q, want %q", created.Name, "Cyberpunk")
	}
	if created.BuiltIn {
		t.Error("created theme should not be built-in")
	}
	if created.Version != 1 {
		t.Errorf("Version = %d, want 1", created.Version)
	}

	// Retrieve the theme by ID.
	w2 := doRequest(mux, "GET", "/api/v1/settings/themes/"+created.ID, nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GetTheme status = %d, want %d", w2.Code, http.StatusOK)
	}

	var fetched settings.ThemeDefinition
	if err := json.NewDecoder(w2.Body).Decode(&fetched); err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("fetched ID = %q, want %q", fetched.ID, created.ID)
	}
	bg, ok := fetched.Tokens.Backgrounds["primary"]
	if !ok || bg != "#0a0a2e" {
		t.Errorf("Tokens.Backgrounds[primary] = %q, want %q", bg, "#0a0a2e")
	}
}

func TestHandleUpdateTheme(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Create a custom theme first.
	createBody := map[string]any{
		"name":      "Original",
		"base_mode": "dark",
	}
	w := doRequest(mux, "POST", "/api/v1/settings/themes", createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateTheme status = %d, want %d", w.Code, http.StatusCreated)
	}
	var created settings.ThemeDefinition
	_ = json.NewDecoder(w.Body).Decode(&created)

	// Update the theme's tokens.
	updateBody := map[string]any{
		"name": "Updated",
		"tokens": map[string]any{
			"text": map[string]string{"primary": "#ffffff"},
		},
	}
	w2 := doRequest(mux, "PUT", "/api/v1/settings/themes/"+created.ID, updateBody)
	if w2.Code != http.StatusOK {
		t.Fatalf("UpdateTheme status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
	}

	var updated settings.ThemeDefinition
	_ = json.NewDecoder(w2.Body).Decode(&updated)

	if updated.Name != "Updated" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated")
	}
	if updated.Version != 2 {
		t.Errorf("Version = %d, want 2", updated.Version)
	}
	tp, ok := updated.Tokens.Text["primary"]
	if !ok || tp != "#ffffff" {
		t.Errorf("Tokens.Text[primary] = %q, want %q", tp, "#ffffff")
	}
}

func TestHandleUpdateTheme_BuiltIn(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Trigger seed by listing themes first.
	doRequest(mux, "GET", "/api/v1/settings/themes", nil)

	w := doRequest(mux, "PUT", "/api/v1/settings/themes/builtin-forest-dark", map[string]any{
		"name": "Hacked",
	})

	if w.Code != http.StatusForbidden {
		t.Errorf("UpdateTheme built-in status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestHandleDeleteTheme(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Create a custom theme.
	createBody := map[string]any{
		"name":      "Disposable",
		"base_mode": "light",
	}
	w := doRequest(mux, "POST", "/api/v1/settings/themes", createBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateTheme status = %d, want %d", w.Code, http.StatusCreated)
	}
	var created settings.ThemeDefinition
	_ = json.NewDecoder(w.Body).Decode(&created)

	// Delete it.
	w2 := doRequest(mux, "DELETE", "/api/v1/settings/themes/"+created.ID, nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("DeleteTheme status = %d, want %d; body: %s", w2.Code, http.StatusNoContent, w2.Body.String())
	}

	// Verify it's gone.
	w3 := doRequest(mux, "GET", "/api/v1/settings/themes/"+created.ID, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("GetTheme after delete status = %d, want %d", w3.Code, http.StatusNotFound)
	}
}

func TestHandleDeleteTheme_BuiltIn(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Trigger seed.
	doRequest(mux, "GET", "/api/v1/settings/themes", nil)

	w := doRequest(mux, "DELETE", "/api/v1/settings/themes/builtin-forest-dark", nil)

	if w.Code != http.StatusForbidden {
		t.Errorf("DeleteTheme built-in status = %d, want %d; body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestHandleGetActiveTheme_Default(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "GET", "/api/v1/settings/themes/active", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("GetActiveTheme status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp settings.ActiveThemeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if resp.ThemeID != "builtin-forest-dark" {
		t.Errorf("ThemeID = %q, want %q", resp.ThemeID, "builtin-forest-dark")
	}
}

func TestHandleSetActiveTheme(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Seed built-ins and create a custom theme.
	doRequest(mux, "GET", "/api/v1/settings/themes", nil)

	w := doRequest(mux, "PUT", "/api/v1/settings/themes/active", map[string]any{
		"theme_id": "builtin-forest-light",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("SetActiveTheme status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify it stuck.
	w2 := doRequest(mux, "GET", "/api/v1/settings/themes/active", nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("GetActiveTheme status = %d, want %d", w2.Code, http.StatusOK)
	}

	var resp settings.ActiveThemeResponse
	_ = json.NewDecoder(w2.Body).Decode(&resp)
	if resp.ThemeID != "builtin-forest-light" {
		t.Errorf("ThemeID = %q, want %q", resp.ThemeID, "builtin-forest-light")
	}
}

func TestHandleCreateTheme_Validation(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	tests := []struct {
		name string
		body map[string]any
		want int
	}{
		{
			name: "empty name",
			body: map[string]any{"name": "", "base_mode": "dark"},
			want: http.StatusBadRequest,
		},
		{
			name: "whitespace-only name",
			body: map[string]any{"name": "   ", "base_mode": "dark"},
			want: http.StatusBadRequest,
		},
		{
			name: "invalid base_mode",
			body: map[string]any{"name": "Test", "base_mode": "neon"},
			want: http.StatusBadRequest,
		},
		{
			name: "missing base_mode",
			body: map[string]any{"name": "Test"},
			want: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(mux, "POST", "/api/v1/settings/themes", tc.body)
			if w.Code != tc.want {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}
