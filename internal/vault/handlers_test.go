package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// newTestModule creates a vault Module backed by an in-memory SQLite DB
// with the vault unsealed and ready for handler testing.
func newTestModule(t *testing.T) *Module {
	t.Helper()

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "vault", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	km := NewKeyManager()
	// First-run setup to unseal the vault.
	salt, verification, err := km.FirstRunSetup("test-passphrase")
	if err != nil {
		t.Fatalf("first run setup: %v", err)
	}

	vs := NewVaultStore(db.DB())
	if err := vs.UpsertMasterKeyRecord(ctx, salt, verification); err != nil {
		t.Fatalf("persist master key: %v", err)
	}

	m := &Module{
		logger:         zap.NewNop(),
		store:          vs,
		cfg:            DefaultConfig(),
		km:             km,
		ctx:            ctx,
		readPassphrase: func() (string, error) { return "", fmt.Errorf("no terminal") },
	}
	return m
}

// newSealedTestModule creates a vault Module with the vault sealed.
func newSealedTestModule(t *testing.T) *Module {
	t.Helper()

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "vault", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	km := NewKeyManager()
	// Initialize but don't unseal -- simulate a vault that has been
	// initialized before but is currently sealed.
	salt, verification, err := km.FirstRunSetup("test-passphrase")
	if err != nil {
		t.Fatalf("first run setup: %v", err)
	}

	vs := NewVaultStore(db.DB())
	if err := vs.UpsertMasterKeyRecord(ctx, salt, verification); err != nil {
		t.Fatalf("persist master key: %v", err)
	}

	// Seal after setup.
	km.Seal()

	m := &Module{
		logger:         zap.NewNop(),
		store:          vs,
		cfg:            DefaultConfig(),
		km:             km,
		ctx:            ctx,
		readPassphrase: func() (string, error) { return "", fmt.Errorf("no terminal") },
	}
	return m
}

// testEventBus is a synchronous event bus for testing.
type testEventBus struct {
	mu     sync.Mutex
	events []plugin.Event
}

func (b *testEventBus) Publish(_ context.Context, event plugin.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
	return nil
}

func (b *testEventBus) PublishAsync(_ context.Context, event plugin.Event) {
	_ = b.Publish(context.Background(), event)
}

func (b *testEventBus) Subscribe(_ string, _ plugin.EventHandler) func() {
	return func() {}
}

func (b *testEventBus) SubscribeAll(_ plugin.EventHandler) func() {
	return func() {}
}

func (b *testEventBus) lastEvent() *plugin.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.events) == 0 {
		return nil
	}
	return &b.events[len(b.events)-1]
}

// insertTestCredential creates an encrypted credential in the store via the full
// encryption path (generate DEK, encrypt, wrap, store).
func insertTestCredential(t *testing.T, m *Module, id, name, credType, deviceID string, data map[string]any) {
	t.Helper()

	dataJSON, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal test data: %v", err)
	}

	dek, err := GenerateDEK()
	if err != nil {
		t.Fatalf("generate DEK: %v", err)
	}
	defer ZeroBytes(dek)

	encrypted, err := Encrypt(dek, dataJSON)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	wrappedKey, err := m.km.WrapDEK(dek)
	if err != nil {
		t.Fatalf("wrap DEK: %v", err)
	}

	now := time.Now().UTC()
	if err := m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: id, Name: name, Type: credType, DeviceID: deviceID,
		EncryptedData: encrypted, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("insert credential: %v", err)
	}

	if err := m.store.InsertKey(context.Background(), id, wrappedKey); err != nil {
		t.Fatalf("insert key: %v", err)
	}
}

// --- List Credentials ---

func TestHandleListCredentials_WithData(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "c1", Name: "SSH", Type: CredTypeSSHPassword, DeviceID: "d1",
		EncryptedData: []byte("enc"), CreatedAt: now, UpdatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/credentials", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleListCredentials(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var metas []CredentialMeta
	if err := json.NewDecoder(rr.Body).Decode(&metas); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("len = %d, want 1", len(metas))
	}
	if metas[0].ID != "c1" {
		t.Errorf("ID = %q, want %q", metas[0].ID, "c1")
	}
}

// --- Create Credential ---

func TestHandleCreateCredential_Success(t *testing.T) {
	m := newTestModule(t)
	bus := &testEventBus{}
	m.bus = bus

	body := `{"name":"My SSH","type":"ssh_password","device_id":"dev-1","data":{"username":"admin","password":"secret"}}`
	req := httptest.NewRequest(http.MethodPost, "/credentials", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleCreateCredential(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusCreated, rr.Body.String())
	}

	var meta CredentialMeta
	if err := json.NewDecoder(rr.Body).Decode(&meta); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if meta.Name != "My SSH" {
		t.Errorf("Name = %q, want %q", meta.Name, "My SSH")
	}
	if meta.Type != CredTypeSSHPassword {
		t.Errorf("Type = %q, want %q", meta.Type, CredTypeSSHPassword)
	}

	// Verify event was published.
	if e := bus.lastEvent(); e == nil || e.Topic != TopicCredentialCreated {
		t.Error("expected credential.created event")
	}
}

func TestHandleCreateCredential_Sealed(t *testing.T) {
	m := newSealedTestModule(t)

	body := `{"name":"My SSH","type":"ssh_password","data":{"username":"admin","password":"secret"}}`
	req := httptest.NewRequest(http.MethodPost, "/credentials", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleCreateCredential(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleCreateCredential_InvalidType(t *testing.T) {
	m := newTestModule(t)

	body := `{"name":"Bad","type":"invalid_type","data":{"foo":"bar"}}`
	req := httptest.NewRequest(http.MethodPost, "/credentials", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleCreateCredential(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCredential_MissingName(t *testing.T) {
	m := newTestModule(t)

	body := `{"name":"","type":"ssh_password","data":{"username":"admin","password":"secret"}}`
	req := httptest.NewRequest(http.MethodPost, "/credentials", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleCreateCredential(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCredential_InvalidBody(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodPost, "/credentials", bytes.NewBufferString("not json"))
	rr := httptest.NewRecorder()
	m.handleCreateCredential(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleCreateCredential_MissingRequiredDataField(t *testing.T) {
	m := newTestModule(t)

	// ssh_password requires username and password, only providing username
	body := `{"name":"Incomplete","type":"ssh_password","data":{"username":"admin"}}`
	req := httptest.NewRequest(http.MethodPost, "/credentials", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleCreateCredential(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// --- Get Credential ---

func TestHandleGetCredential_Found(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "c1", Name: "SSH Prod", Type: CredTypeSSHPassword,
		EncryptedData: []byte("enc"), CreatedAt: now, UpdatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/credentials/c1", http.NoBody)
	req.SetPathValue("id", "c1")
	rr := httptest.NewRecorder()
	m.handleGetCredential(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var meta CredentialMeta
	if err := json.NewDecoder(rr.Body).Decode(&meta); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if meta.Name != "SSH Prod" {
		t.Errorf("Name = %q, want %q", meta.Name, "SSH Prod")
	}
}

func TestHandleGetCredential_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/credentials/missing", http.NoBody)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	m.handleGetCredential(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// --- Get Credential Data (decrypt) ---

func TestHandleGetCredentialData_Success(t *testing.T) {
	m := newTestModule(t)
	testData := map[string]any{"username": "admin", "password": "s3cret"}
	insertTestCredential(t, m, "c1", "SSH", CredTypeSSHPassword, "dev-1", testData)

	req := httptest.NewRequest(http.MethodGet, "/credentials/c1/data?purpose=test", http.NoBody)
	req.SetPathValue("id", "c1")
	rr := httptest.NewRecorder()
	m.handleGetCredentialData(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var result CredentialData
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Data["username"] != "admin" {
		t.Errorf("username = %v, want %q", result.Data["username"], "admin")
	}
	if result.Data["password"] != "s3cret" {
		t.Errorf("password = %v, want %q", result.Data["password"], "s3cret")
	}

	// Verify audit entry was created.
	entries, err := m.store.ListAuditEntries(context.Background(), "c1", 10)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(entries))
	}
	if entries[0].Action != "read" {
		t.Errorf("audit action = %q, want %q", entries[0].Action, "read")
	}
	if entries[0].Purpose != "test" {
		t.Errorf("audit purpose = %q, want %q", entries[0].Purpose, "test")
	}
}

func TestHandleGetCredentialData_Sealed(t *testing.T) {
	m := newSealedTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/credentials/c1/data", http.NoBody)
	req.SetPathValue("id", "c1")
	rr := httptest.NewRecorder()
	m.handleGetCredentialData(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleGetCredentialData_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/credentials/missing/data", http.NoBody)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	m.handleGetCredentialData(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// --- Update Credential ---

func TestHandleUpdateCredential_MetadataOnly(t *testing.T) {
	m := newTestModule(t)
	testData := map[string]any{"username": "admin", "password": "s3cret"}
	insertTestCredential(t, m, "c1", "SSH", CredTypeSSHPassword, "dev-1", testData)

	body := `{"name":"SSH Updated","description":"new desc"}`
	req := httptest.NewRequest(http.MethodPut, "/credentials/c1", bytes.NewBufferString(body))
	req.SetPathValue("id", "c1")
	rr := httptest.NewRecorder()
	m.handleUpdateCredential(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var meta CredentialMeta
	if err := json.NewDecoder(rr.Body).Decode(&meta); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if meta.Name != "SSH Updated" {
		t.Errorf("Name = %q, want %q", meta.Name, "SSH Updated")
	}
	if meta.Description != "new desc" {
		t.Errorf("Description = %q, want %q", meta.Description, "new desc")
	}
}

func TestHandleUpdateCredential_WithData(t *testing.T) {
	m := newTestModule(t)
	testData := map[string]any{"username": "admin", "password": "old"}
	insertTestCredential(t, m, "c1", "SSH", CredTypeSSHPassword, "dev-1", testData)

	body := `{"data":{"username":"admin","password":"new-password"}}`
	req := httptest.NewRequest(http.MethodPut, "/credentials/c1", bytes.NewBufferString(body))
	req.SetPathValue("id", "c1")
	rr := httptest.NewRecorder()
	m.handleUpdateCredential(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify the data was re-encrypted by reading it back.
	readReq := httptest.NewRequest(http.MethodGet, "/credentials/c1/data", http.NoBody)
	readReq.SetPathValue("id", "c1")
	readRR := httptest.NewRecorder()
	m.handleGetCredentialData(readRR, readReq)

	if readRR.Code != http.StatusOK {
		t.Fatalf("read status = %d", readRR.Code)
	}
	var result CredentialData
	if err := json.NewDecoder(readRR.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Data["password"] != "new-password" {
		t.Errorf("password = %v, want %q", result.Data["password"], "new-password")
	}
}

func TestHandleUpdateCredential_DataUpdateSealed(t *testing.T) {
	m := newSealedTestModule(t)

	// Insert a credential record directly (no encryption needed for this test).
	now := time.Now().UTC()
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "c1", Name: "SSH", Type: CredTypeSSHPassword,
		EncryptedData: []byte("enc"), CreatedAt: now, UpdatedAt: now,
	})

	body := `{"data":{"username":"admin","password":"new"}}`
	req := httptest.NewRequest(http.MethodPut, "/credentials/c1", bytes.NewBufferString(body))
	req.SetPathValue("id", "c1")
	rr := httptest.NewRecorder()
	m.handleUpdateCredential(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleUpdateCredential_NotFound(t *testing.T) {
	m := newTestModule(t)

	body := `{"name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/credentials/missing", bytes.NewBufferString(body))
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	m.handleUpdateCredential(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// --- Delete Credential ---

func TestHandleDeleteCredential_Success(t *testing.T) {
	m := newTestModule(t)
	bus := &testEventBus{}
	m.bus = bus

	testData := map[string]any{"username": "admin", "password": "s3cret"}
	insertTestCredential(t, m, "c1", "SSH", CredTypeSSHPassword, "", testData)

	req := httptest.NewRequest(http.MethodDelete, "/credentials/c1", http.NoBody)
	req.SetPathValue("id", "c1")
	rr := httptest.NewRecorder()
	m.handleDeleteCredential(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}

	// Verify it's gone.
	rec, _ := m.store.GetCredential(context.Background(), "c1")
	if rec != nil {
		t.Error("credential should be deleted")
	}

	if e := bus.lastEvent(); e == nil || e.Topic != TopicCredentialDeleted {
		t.Error("expected credential.deleted event")
	}
}

func TestHandleDeleteCredential_NotFound(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodDelete, "/credentials/missing", http.NoBody)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	m.handleDeleteCredential(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// --- Device Credentials ---

func TestHandleListDeviceCredentials_Success(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "c1", Name: "SSH", Type: CredTypeSSHPassword, DeviceID: "dev-1",
		EncryptedData: []byte("enc"), CreatedAt: now, UpdatedAt: now,
	})
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "c2", Name: "SNMP", Type: CredTypeSNMPv2c, DeviceID: "dev-1",
		EncryptedData: []byte("enc"), CreatedAt: now, UpdatedAt: now,
	})
	_ = m.store.InsertCredential(context.Background(), &CredentialRecord{
		ID: "c3", Name: "Other", Type: CredTypeAPIKey, DeviceID: "dev-2",
		EncryptedData: []byte("enc"), CreatedAt: now, UpdatedAt: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/credentials/device/dev-1", http.NoBody)
	req.SetPathValue("device_id", "dev-1")
	rr := httptest.NewRecorder()
	m.handleListDeviceCredentials(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var metas []CredentialMeta
	if err := json.NewDecoder(rr.Body).Decode(&metas); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(metas) != 2 {
		t.Errorf("len = %d, want 2", len(metas))
	}
}

// --- Seal / Unseal ---

func TestHandleSeal(t *testing.T) {
	m := newTestModule(t)
	bus := &testEventBus{}
	m.bus = bus

	if m.km.IsSealed() {
		t.Fatal("precondition: vault should be unsealed")
	}

	req := httptest.NewRequest(http.MethodPost, "/seal", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleSeal(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !m.km.IsSealed() {
		t.Error("vault should be sealed after handleSeal")
	}

	if e := bus.lastEvent(); e == nil || e.Topic != TopicVaultStatusChanged {
		t.Error("expected vault.status.changed event")
	}
}

func TestHandleUnseal_FirstRun(t *testing.T) {
	// Create a module with an uninitialized KeyManager.
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "vault", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	m := &Module{
		logger:         zap.NewNop(),
		store:          NewVaultStore(db.DB()),
		cfg:            DefaultConfig(),
		km:             NewKeyManager(), // Not initialized yet.
		ctx:            ctx,
		readPassphrase: func() (string, error) { return "", fmt.Errorf("no terminal") },
	}
	bus := &testEventBus{}
	m.bus = bus

	body := `{"passphrase":"my-secure-pass"}`
	req := httptest.NewRequest(http.MethodPost, "/unseal", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleUnseal(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if m.km.IsSealed() {
		t.Error("vault should be unsealed after first-run setup")
	}
	if !m.km.IsInitialized() {
		t.Error("vault should be initialized after first-run setup")
	}

	if e := bus.lastEvent(); e == nil || e.Topic != TopicVaultStatusChanged {
		t.Error("expected vault.status.changed event")
	}
}

func TestHandleUnseal_ExistingVault(t *testing.T) {
	m := newSealedTestModule(t)

	body := `{"passphrase":"test-passphrase"}`
	req := httptest.NewRequest(http.MethodPost, "/unseal", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleUnseal(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if m.km.IsSealed() {
		t.Error("vault should be unsealed after correct passphrase")
	}
}

func TestHandleUnseal_WrongPassphrase(t *testing.T) {
	m := newSealedTestModule(t)

	body := `{"passphrase":"wrong-password"}`
	req := httptest.NewRequest(http.MethodPost, "/unseal", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleUnseal(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
	if !m.km.IsSealed() {
		t.Error("vault should remain sealed after wrong passphrase")
	}
}

func TestHandleUnseal_EmptyPassphrase(t *testing.T) {
	m := newSealedTestModule(t)

	body := `{"passphrase":""}`
	req := httptest.NewRequest(http.MethodPost, "/unseal", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleUnseal(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// --- Vault Status ---

func TestHandleVaultStatus_Unsealed(t *testing.T) {
	m := newTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/status", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleVaultStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["sealed"] != false {
		t.Errorf("sealed = %v, want false", result["sealed"])
	}
	if result["initialized"] != true {
		t.Errorf("initialized = %v, want true", result["initialized"])
	}
}

func TestHandleVaultStatus_Sealed(t *testing.T) {
	m := newSealedTestModule(t)

	req := httptest.NewRequest(http.MethodGet, "/status", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleVaultStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["sealed"] != true {
		t.Errorf("sealed = %v, want true", result["sealed"])
	}
}

// --- Rotate Keys ---

func TestHandleRotateKeys_Success(t *testing.T) {
	m := newTestModule(t)
	bus := &testEventBus{}
	m.bus = bus

	// Create two credentials with proper encryption.
	insertTestCredential(t, m, "c1", "SSH", CredTypeSSHPassword, "", map[string]any{
		"username": "admin", "password": "pass1",
	})
	insertTestCredential(t, m, "c2", "API", CredTypeAPIKey, "", map[string]any{
		"key": "api-key-123",
	})

	body := `{"new_passphrase":"new-secure-pass"}`
	req := httptest.NewRequest(http.MethodPost, "/rotate-keys", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleRotateKeys(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["rotated"] != float64(2) {
		t.Errorf("rotated = %v, want 2", result["rotated"])
	}

	// Verify we can still decrypt with the new KEK.
	readReq := httptest.NewRequest(http.MethodGet, "/credentials/c1/data", http.NoBody)
	readReq.SetPathValue("id", "c1")
	readRR := httptest.NewRecorder()
	m.handleGetCredentialData(readRR, readReq)

	if readRR.Code != http.StatusOK {
		t.Fatalf("decrypt after rotation: status = %d; body = %s", readRR.Code, readRR.Body.String())
	}

	var data CredentialData
	if err := json.NewDecoder(readRR.Body).Decode(&data); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if data.Data["password"] != "pass1" {
		t.Errorf("password after rotation = %v, want %q", data.Data["password"], "pass1")
	}

	if e := bus.lastEvent(); e == nil || e.Topic != TopicKeysRotated {
		t.Error("expected vault.keys.rotated event")
	}
}

func TestHandleRotateKeys_Sealed(t *testing.T) {
	m := newSealedTestModule(t)

	body := `{"new_passphrase":"new-pass"}`
	req := httptest.NewRequest(http.MethodPost, "/rotate-keys", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleRotateKeys(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleRotateKeys_EmptyPassphrase(t *testing.T) {
	m := newTestModule(t)

	body := `{"new_passphrase":""}`
	req := httptest.NewRequest(http.MethodPost, "/rotate-keys", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleRotateKeys(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// --- Audit ---

func TestHandleListAudit_Success(t *testing.T) {
	m := newTestModule(t)

	// Insert some audit entries directly.
	now := time.Now().UTC()
	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		CredentialID: "c1", UserID: "user1", Action: "read",
		Purpose: "test", SourceIP: "127.0.0.1", Timestamp: now,
	})
	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		CredentialID: "c2", UserID: "user1", Action: "create",
		Timestamp: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/audit", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleListAudit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var entries []AuditEntry
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("len = %d, want 2", len(entries))
	}
}

func TestHandleListAudit_WithCredentialFilter(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		CredentialID: "c1", Action: "read", Timestamp: now,
	})
	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		CredentialID: "c2", Action: "read", Timestamp: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/audit?credential_id=c1", http.NoBody)
	rr := httptest.NewRecorder()
	m.handleListAudit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var entries []AuditEntry
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("len = %d, want 1", len(entries))
	}
}

func TestHandleCredentialAudit_Success(t *testing.T) {
	m := newTestModule(t)

	now := time.Now().UTC()
	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		CredentialID: "c1", Action: "read", Timestamp: now,
	})
	_ = m.store.InsertAuditEntry(context.Background(), &AuditEntry{
		CredentialID: "c1", Action: "update", Timestamp: now,
	})

	req := httptest.NewRequest(http.MethodGet, "/audit/c1", http.NoBody)
	req.SetPathValue("credential_id", "c1")
	rr := httptest.NewRecorder()
	m.handleCredentialAudit(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var entries []AuditEntry
	if err := json.NewDecoder(rr.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("len = %d, want 2", len(entries))
	}
}

// --- Nil Store Edge Cases ---

func TestHandleCreateCredential_NilStore(t *testing.T) {
	m := &Module{
		logger:         zap.NewNop(),
		km:             NewKeyManager(),
		readPassphrase: func() (string, error) { return "", fmt.Errorf("no terminal") },
	}

	body := `{"name":"SSH","type":"ssh_password","data":{"username":"a","password":"b"}}`
	req := httptest.NewRequest(http.MethodPost, "/credentials", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	m.handleCreateCredential(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleDeleteCredential_NilStore(t *testing.T) {
	m := &Module{
		logger:         zap.NewNop(),
		km:             NewKeyManager(),
		readPassphrase: func() (string, error) { return "", fmt.Errorf("no terminal") },
	}

	req := httptest.NewRequest(http.MethodDelete, "/credentials/c1", http.NoBody)
	req.SetPathValue("id", "c1")
	rr := httptest.NewRecorder()
	m.handleDeleteCredential(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

// --- Helper Tests ---

func TestVaultParseLimit(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		defLimit int
		want     int
	}{
		{"default", "", 100, 100},
		{"valid", "?limit=50", 100, 50},
		{"too_large", "?limit=5000", 100, 100},
		{"negative", "?limit=-1", 100, 100},
		{"non_numeric", "?limit=abc", 100, 100},
		{"zero", "?limit=0", 100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/audit"+tt.query, http.NoBody)
			got := vaultParseLimit(req, tt.defLimit)
			if got != tt.want {
				t.Errorf("vaultParseLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}
