package vault

import (
	"context"
	"encoding/json"
	"net/http"
)

// handleListCredentials returns credential metadata. Works when sealed.
func (m *Module) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}
	metas, err := m.store.ListCredentials(r.Context())
	if err != nil {
		m.logger.Warn("failed to list credentials")
		vaultWriteError(w, http.StatusInternalServerError, "failed to list credentials")
		return
	}
	if metas == nil {
		metas = []CredentialMeta{}
	}
	vaultWriteJSON(w, http.StatusOK, metas)
}

// handleVaultStatus returns the vault's sealed/initialized state.
func (m *Module) handleVaultStatus(w http.ResponseWriter, _ *http.Request) {
	initialized := m.km != nil && m.km.IsInitialized()
	sealed := m.km == nil || m.km.IsSealed()

	var count int
	if m.store != nil {
		c, err := m.store.CredentialCount(context.Background())
		if err == nil {
			count = c
		}
	}

	vaultWriteJSON(w, http.StatusOK, map[string]any{
		"sealed":           sealed,
		"initialized":      initialized,
		"credential_count": count,
	})
}

// vaultWriteJSON writes a JSON response with the given status code.
func vaultWriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// vaultWriteError writes a problem+json error response.
func vaultWriteError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://subnetree.com/problems/" + http.StatusText(status),
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
