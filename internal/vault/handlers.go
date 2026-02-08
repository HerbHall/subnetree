package vault

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		// Credential CRUD
		{Method: "GET", Path: "/credentials", Handler: m.handleListCredentials},
		{Method: "POST", Path: "/credentials", Handler: m.handleCreateCredential},
		{Method: "GET", Path: "/credentials/{id}", Handler: m.handleGetCredential},
		{Method: "PUT", Path: "/credentials/{id}", Handler: m.handleUpdateCredential},
		{Method: "DELETE", Path: "/credentials/{id}", Handler: m.handleDeleteCredential},
		// Decrypted data retrieval
		{Method: "GET", Path: "/credentials/{id}/data", Handler: m.handleGetCredentialData},
		// Device-scoped listing
		{Method: "GET", Path: "/credentials/device/{device_id}", Handler: m.handleListDeviceCredentials},
		// Key management
		{Method: "POST", Path: "/rotate-keys", Handler: m.handleRotateKeys},
		{Method: "POST", Path: "/seal", Handler: m.handleSeal},
		{Method: "POST", Path: "/unseal", Handler: m.handleUnseal},
		{Method: "GET", Path: "/status", Handler: m.handleVaultStatus},
		// Audit
		{Method: "GET", Path: "/audit", Handler: m.handleListAudit},
		{Method: "GET", Path: "/audit/{credential_id}", Handler: m.handleCredentialAudit},
	}
}

// --- Credential CRUD ---

// handleListCredentials returns credential metadata. Works when sealed.
func (m *Module) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}
	metas, err := m.store.ListCredentials(r.Context())
	if err != nil {
		m.logger.Warn("failed to list credentials", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to list credentials")
		return
	}
	if metas == nil {
		metas = []CredentialMeta{}
	}
	vaultWriteJSON(w, http.StatusOK, metas)
}

// createCredentialRequest is the expected JSON body for POST /credentials.
type createCredentialRequest struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	DeviceID    string         `json:"device_id,omitempty"`
	Description string         `json:"description,omitempty"`
	Data        map[string]any `json:"data"`
}

// handleCreateCredential creates a new encrypted credential. Requires unsealed vault.
func (m *Module) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}
	if m.km.IsSealed() {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault is sealed")
		return
	}

	var req createCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		vaultWriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := ValidateCredentialName(req.Name); err != nil {
		vaultWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ValidateCredentialType(req.Type); err != nil {
		vaultWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ValidateCredentialData(req.Type, req.Data); err != nil {
		vaultWriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Serialize credential data to JSON.
	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		vaultWriteError(w, http.StatusInternalServerError, "failed to serialize credential data")
		return
	}

	// Generate DEK and encrypt data.
	dek, err := GenerateDEK()
	if err != nil {
		m.logger.Error("failed to generate DEK", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "encryption error")
		return
	}
	defer ZeroBytes(dek)

	encryptedData, err := Encrypt(dek, dataJSON)
	if err != nil {
		m.logger.Error("failed to encrypt credential data", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "encryption error")
		return
	}

	// Wrap DEK with KEK.
	wrappedKey, err := m.km.WrapDEK(dek)
	if err != nil {
		m.logger.Error("failed to wrap DEK", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "encryption error")
		return
	}

	now := time.Now().UTC()
	credID := fmt.Sprintf("vault-%d", now.UnixNano())

	cred := &CredentialRecord{
		ID:            credID,
		Name:          req.Name,
		Type:          req.Type,
		DeviceID:      req.DeviceID,
		Description:   req.Description,
		EncryptedData: encryptedData,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := m.store.InsertCredential(r.Context(), cred); err != nil {
		m.logger.Error("failed to insert credential", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to store credential")
		return
	}

	if err := m.store.InsertKey(r.Context(), credID, wrappedKey); err != nil {
		m.logger.Error("failed to insert key", zap.Error(err))
		// Best-effort cleanup.
		_ = m.store.DeleteCredential(r.Context(), credID)
		vaultWriteError(w, http.StatusInternalServerError, "failed to store encryption key")
		return
	}

	// Audit log.
	m.auditLog(r, credID, "create", "")

	// Publish event.
	m.publishEvent(TopicCredentialCreated, map[string]string{
		"credential_id": credID,
		"type":          req.Type,
	})

	meta := CredentialMeta{
		ID: credID, Name: req.Name, Type: req.Type,
		DeviceID: req.DeviceID, Description: req.Description,
		CreatedAt: now, UpdatedAt: now,
	}
	vaultWriteJSON(w, http.StatusCreated, meta)
}

// handleGetCredential returns credential metadata. Works when sealed.
func (m *Module) handleGetCredential(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		vaultWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	rec, err := m.store.GetCredential(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get credential", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to get credential")
		return
	}
	if rec == nil {
		vaultWriteError(w, http.StatusNotFound, "credential not found")
		return
	}

	meta := CredentialMeta{
		ID: rec.ID, Name: rec.Name, Type: rec.Type,
		DeviceID: rec.DeviceID, Description: rec.Description,
		CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
	}
	vaultWriteJSON(w, http.StatusOK, meta)
}

// handleGetCredentialData decrypts and returns credential data. Requires unsealed vault.
func (m *Module) handleGetCredentialData(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}
	if m.km.IsSealed() {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault is sealed")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		vaultWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	rec, err := m.store.GetCredential(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get credential", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to get credential")
		return
	}
	if rec == nil {
		vaultWriteError(w, http.StatusNotFound, "credential not found")
		return
	}

	// Get wrapped DEK and unwrap it.
	key, err := m.store.GetKey(r.Context(), id)
	if err != nil || key == nil {
		m.logger.Error("failed to get encryption key", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "encryption key not found")
		return
	}

	dek, err := m.km.UnwrapDEK(key.WrappedKey)
	if err != nil {
		m.logger.Error("failed to unwrap DEK", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "decryption error")
		return
	}
	defer ZeroBytes(dek)

	plaintext, err := Decrypt(dek, rec.EncryptedData)
	if err != nil {
		m.logger.Error("failed to decrypt credential", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "decryption error")
		return
	}

	var data map[string]any
	if err := json.Unmarshal(plaintext, &data); err != nil {
		m.logger.Error("failed to unmarshal decrypted data", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "data corruption")
		return
	}

	// Audit log with purpose.
	purpose := r.URL.Query().Get("purpose")
	m.auditLog(r, id, "read", purpose)

	result := CredentialData{
		CredentialMeta: CredentialMeta{
			ID: rec.ID, Name: rec.Name, Type: rec.Type,
			DeviceID: rec.DeviceID, Description: rec.Description,
			CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
		},
		Data: data,
	}
	vaultWriteJSON(w, http.StatusOK, result)
}

// updateCredentialRequest is the expected JSON body for PUT /credentials/{id}.
type updateCredentialRequest struct {
	Name        *string        `json:"name,omitempty"`
	DeviceID    *string        `json:"device_id,omitempty"`
	Description *string        `json:"description,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
}

// handleUpdateCredential updates credential metadata and/or data.
// Metadata-only updates work when sealed. Data updates require unsealed vault.
func (m *Module) handleUpdateCredential(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		vaultWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req updateCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		vaultWriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// If data is being updated, vault must be unsealed.
	if req.Data != nil && m.km.IsSealed() {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault is sealed; cannot update credential data")
		return
	}

	rec, err := m.store.GetCredential(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get credential", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to get credential")
		return
	}
	if rec == nil {
		vaultWriteError(w, http.StatusNotFound, "credential not found")
		return
	}

	// Apply metadata updates.
	if req.Name != nil {
		if err := ValidateCredentialName(*req.Name); err != nil {
			vaultWriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		rec.Name = *req.Name
	}
	if req.DeviceID != nil {
		rec.DeviceID = *req.DeviceID
	}
	if req.Description != nil {
		rec.Description = *req.Description
	}

	// Apply data update (re-encrypt with same DEK).
	if req.Data != nil {
		if err := ValidateCredentialData(rec.Type, req.Data); err != nil {
			vaultWriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		key, err := m.store.GetKey(r.Context(), id)
		if err != nil || key == nil {
			m.logger.Error("failed to get encryption key", zap.Error(err))
			vaultWriteError(w, http.StatusInternalServerError, "encryption key not found")
			return
		}

		dek, err := m.km.UnwrapDEK(key.WrappedKey)
		if err != nil {
			m.logger.Error("failed to unwrap DEK", zap.Error(err))
			vaultWriteError(w, http.StatusInternalServerError, "decryption error")
			return
		}
		defer ZeroBytes(dek)

		dataJSON, err := json.Marshal(req.Data)
		if err != nil {
			vaultWriteError(w, http.StatusInternalServerError, "failed to serialize credential data")
			return
		}

		encrypted, err := Encrypt(dek, dataJSON)
		if err != nil {
			m.logger.Error("failed to encrypt credential data", zap.Error(err))
			vaultWriteError(w, http.StatusInternalServerError, "encryption error")
			return
		}
		rec.EncryptedData = encrypted
	}

	rec.UpdatedAt = time.Now().UTC()
	if err := m.store.UpdateCredential(r.Context(), rec); err != nil {
		m.logger.Error("failed to update credential", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to update credential")
		return
	}

	m.auditLog(r, id, "update", "")
	m.publishEvent(TopicCredentialUpdated, map[string]string{"credential_id": id})

	meta := CredentialMeta{
		ID: rec.ID, Name: rec.Name, Type: rec.Type,
		DeviceID: rec.DeviceID, Description: rec.Description,
		CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
	}
	vaultWriteJSON(w, http.StatusOK, meta)
}

// handleDeleteCredential deletes a credential and its key. Works when sealed.
func (m *Module) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		vaultWriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Verify credential exists.
	rec, err := m.store.GetCredential(r.Context(), id)
	if err != nil {
		m.logger.Warn("failed to get credential", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to get credential")
		return
	}
	if rec == nil {
		vaultWriteError(w, http.StatusNotFound, "credential not found")
		return
	}

	// Delete key first (foreign key), then credential.
	_ = m.store.DeleteKey(r.Context(), id)
	if err := m.store.DeleteCredential(r.Context(), id); err != nil {
		m.logger.Error("failed to delete credential", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to delete credential")
		return
	}

	m.auditLog(r, id, "delete", "")
	m.publishEvent(TopicCredentialDeleted, map[string]string{"credential_id": id})

	w.WriteHeader(http.StatusNoContent)
}

// handleListDeviceCredentials returns credentials for a specific device.
func (m *Module) handleListDeviceCredentials(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}

	deviceID := r.PathValue("device_id")
	if deviceID == "" {
		vaultWriteError(w, http.StatusBadRequest, "device_id is required")
		return
	}

	metas, err := m.store.ListCredentialsByDevice(r.Context(), deviceID)
	if err != nil {
		m.logger.Warn("failed to list credentials", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to list credentials")
		return
	}
	if metas == nil {
		metas = []CredentialMeta{}
	}
	vaultWriteJSON(w, http.StatusOK, metas)
}

// --- Key Management ---

// rotateKeysRequest is the expected JSON body for POST /rotate-keys.
type rotateKeysRequest struct {
	NewPassphrase string `json:"new_passphrase"`
}

// handleRotateKeys re-wraps all DEKs with a new passphrase. Requires unsealed vault.
func (m *Module) handleRotateKeys(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}
	if m.km.IsSealed() {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault is sealed")
		return
	}

	var req rotateKeysRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		vaultWriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.NewPassphrase == "" {
		vaultWriteError(w, http.StatusBadRequest, "new_passphrase is required")
		return
	}

	// Rotate KEK.
	newSalt, newVerification, rewrap, err := m.km.RotateKEK(req.NewPassphrase)
	if err != nil {
		m.logger.Error("failed to rotate KEK", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "key rotation failed")
		return
	}

	// Re-wrap all DEKs.
	keys, err := m.store.ListAllKeys(r.Context())
	if err != nil {
		m.logger.Error("failed to list keys for rotation", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "key rotation failed")
		return
	}

	for i := range keys {
		newWrapped, err := rewrap(keys[i].WrappedKey)
		if err != nil {
			m.logger.Error("failed to re-wrap DEK",
				zap.String("credential_id", keys[i].CredentialID),
				zap.Error(err))
			vaultWriteError(w, http.StatusInternalServerError, "key rotation failed during re-wrap")
			return
		}
		if err := m.store.UpdateKey(r.Context(), keys[i].CredentialID, newWrapped); err != nil {
			m.logger.Error("failed to update wrapped key",
				zap.String("credential_id", keys[i].CredentialID),
				zap.Error(err))
			vaultWriteError(w, http.StatusInternalServerError, "key rotation failed during update")
			return
		}
	}

	// Persist new master key record.
	if err := m.store.UpsertMasterKeyRecord(r.Context(), newSalt, newVerification); err != nil {
		m.logger.Error("failed to update master key record", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "key rotation failed")
		return
	}

	m.auditLog(r, "", "rotate_keys", "")
	m.publishEvent(TopicKeysRotated, map[string]string{"keys_rotated": fmt.Sprintf("%d", len(keys))})

	vaultWriteJSON(w, http.StatusOK, map[string]any{
		"rotated": len(keys),
		"message": "key rotation complete",
	})
}

// handleSeal seals the vault by zeroing the KEK.
func (m *Module) handleSeal(w http.ResponseWriter, _ *http.Request) {
	if m.km == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault not initialized")
		return
	}

	m.km.Seal()
	m.publishEvent(TopicVaultStatusChanged, map[string]string{"status": "sealed"})

	vaultWriteJSON(w, http.StatusOK, map[string]string{"status": "sealed"})
}

// unsealRequest is the expected JSON body for POST /unseal.
type unsealRequest struct {
	Passphrase string `json:"passphrase"`
}

// handleUnseal unseals the vault with the given passphrase.
func (m *Module) handleUnseal(w http.ResponseWriter, r *http.Request) {
	if m.km == nil || m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault not initialized")
		return
	}

	var req unsealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		vaultWriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Passphrase == "" {
		vaultWriteError(w, http.StatusBadRequest, "passphrase is required")
		return
	}

	if !m.km.IsInitialized() {
		// First-run setup.
		salt, verification, err := m.km.FirstRunSetup(req.Passphrase)
		if err != nil {
			m.logger.Error("failed to initialize vault", zap.Error(err))
			vaultWriteError(w, http.StatusInternalServerError, "vault initialization failed")
			return
		}
		if err := m.store.UpsertMasterKeyRecord(r.Context(), salt, verification); err != nil {
			m.logger.Error("failed to persist master key", zap.Error(err))
			m.km.Seal()
			vaultWriteError(w, http.StatusInternalServerError, "vault initialization failed")
			return
		}
		m.publishEvent(TopicVaultStatusChanged, map[string]string{"status": "unsealed", "first_run": "true"})
		vaultWriteJSON(w, http.StatusOK, map[string]string{"status": "unsealed", "message": "vault initialized"})
		return
	}

	if err := m.km.Unseal(req.Passphrase); err != nil {
		if err == ErrWrongPassphrase {
			vaultWriteError(w, http.StatusUnauthorized, "wrong passphrase")
			return
		}
		m.logger.Error("failed to unseal vault", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "unseal failed")
		return
	}

	m.publishEvent(TopicVaultStatusChanged, map[string]string{"status": "unsealed"})
	vaultWriteJSON(w, http.StatusOK, map[string]string{"status": "unsealed"})
}

// handleVaultStatus returns the vault's sealed/initialized state.
func (m *Module) handleVaultStatus(w http.ResponseWriter, r *http.Request) {
	initialized := m.km != nil && m.km.IsInitialized()
	sealed := m.km == nil || m.km.IsSealed()

	var count int
	if m.store != nil {
		c, err := m.store.CredentialCount(r.Context())
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

// --- Audit ---

// handleListAudit returns audit log entries with optional filtering.
func (m *Module) handleListAudit(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}

	credentialID := r.URL.Query().Get("credential_id")
	limit := vaultParseLimit(r, 100)

	entries, err := m.store.ListAuditEntries(r.Context(), credentialID, limit)
	if err != nil {
		m.logger.Warn("failed to list audit entries", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	vaultWriteJSON(w, http.StatusOK, entries)
}

// handleCredentialAudit returns audit entries for a specific credential.
func (m *Module) handleCredentialAudit(w http.ResponseWriter, r *http.Request) {
	if m.store == nil {
		vaultWriteError(w, http.StatusServiceUnavailable, "vault store not available")
		return
	}

	credentialID := r.PathValue("credential_id")
	if credentialID == "" {
		vaultWriteError(w, http.StatusBadRequest, "credential_id is required")
		return
	}

	limit := vaultParseLimit(r, 100)
	entries, err := m.store.ListAuditEntries(r.Context(), credentialID, limit)
	if err != nil {
		m.logger.Warn("failed to list audit entries", zap.Error(err))
		vaultWriteError(w, http.StatusInternalServerError, "failed to list audit entries")
		return
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	vaultWriteJSON(w, http.StatusOK, entries)
}

// --- Helpers ---

// auditLog records a credential access event. Non-blocking -- errors are logged.
func (m *Module) auditLog(r *http.Request, credentialID, action, purpose string) {
	if m.store == nil {
		return
	}
	entry := &AuditEntry{
		CredentialID: credentialID,
		UserID:       extractUserID(r),
		Action:       action,
		Purpose:      purpose,
		SourceIP:     r.RemoteAddr,
		Timestamp:    time.Now().UTC(),
	}
	if err := m.store.InsertAuditEntry(r.Context(), entry); err != nil {
		m.logger.Warn("failed to write audit entry", zap.Error(err))
	}
}

// publishEvent publishes an event to the bus if available.
func (m *Module) publishEvent(topic string, payload any) {
	if m.bus == nil {
		return
	}
	m.bus.PublishAsync(m.ctx, plugin.Event{
		Topic:     topic,
		Source:    "vault",
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
}

// extractUserID attempts to extract the user ID from the request context.
// Returns empty string if not available.
func extractUserID(r *http.Request) string {
	// The auth middleware stores user ID in context.
	if uid, ok := r.Context().Value("user_id").(string); ok {
		return uid
	}
	return ""
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

// vaultParseLimit extracts a limit query parameter with a default value.
func vaultParseLimit(r *http.Request, defaultLimit int) int {
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 1000 {
			return n
		}
	}
	return defaultLimit
}
