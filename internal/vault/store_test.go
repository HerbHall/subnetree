package vault

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
)

func testStore(t *testing.T) *VaultStore {
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
	return NewVaultStore(db.DB())
}

// --- Master Key Tests ---

func TestGetMasterKeyRecord_Empty(t *testing.T) {
	s := testStore(t)
	rec, err := s.GetMasterKeyRecord(context.Background())
	if err != nil {
		t.Fatalf("GetMasterKeyRecord() error = %v", err)
	}
	if rec != nil {
		t.Error("expected nil for empty vault_master")
	}
}

func TestUpsertMasterKeyRecord_InsertAndGet(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	salt := []byte("test-salt-16byte")
	verification := []byte("test-verification-blob")

	if err := s.UpsertMasterKeyRecord(ctx, salt, verification); err != nil {
		t.Fatalf("UpsertMasterKeyRecord() error = %v", err)
	}

	rec, err := s.GetMasterKeyRecord(ctx)
	if err != nil {
		t.Fatalf("GetMasterKeyRecord() error = %v", err)
	}
	if rec == nil {
		t.Fatal("expected non-nil record")
	}
	if string(rec.Salt) != string(salt) {
		t.Errorf("Salt = %q, want %q", rec.Salt, salt)
	}
	if string(rec.VerificationBlob) != string(verification) {
		t.Errorf("VerificationBlob = %q, want %q", rec.VerificationBlob, verification)
	}
}

func TestUpsertMasterKeyRecord_Update(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	_ = s.UpsertMasterKeyRecord(ctx, []byte("salt-1"), []byte("blob-1"))
	_ = s.UpsertMasterKeyRecord(ctx, []byte("salt-2"), []byte("blob-2"))

	rec, _ := s.GetMasterKeyRecord(ctx)
	if string(rec.Salt) != "salt-2" {
		t.Errorf("Salt after update = %q, want %q", rec.Salt, "salt-2")
	}
}

// --- Credential Tests ---

func TestInsertCredential_AndGet(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second).UTC()

	cred := &CredentialRecord{
		ID:            "cred-001",
		Name:          "Router SSH",
		Type:          CredTypeSSHPassword,
		DeviceID:      "dev-001",
		Description:   "Main router",
		EncryptedData: []byte("encrypted-data-here"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.InsertCredential(ctx, cred); err != nil {
		t.Fatalf("InsertCredential() error = %v", err)
	}

	got, err := s.GetCredential(ctx, "cred-001")
	if err != nil {
		t.Fatalf("GetCredential() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil credential")
	}
	if got.ID != "cred-001" {
		t.Errorf("ID = %q, want %q", got.ID, "cred-001")
	}
	if got.Name != "Router SSH" {
		t.Errorf("Name = %q, want %q", got.Name, "Router SSH")
	}
	if got.Type != CredTypeSSHPassword {
		t.Errorf("Type = %q, want %q", got.Type, CredTypeSSHPassword)
	}
	if got.DeviceID != "dev-001" {
		t.Errorf("DeviceID = %q, want %q", got.DeviceID, "dev-001")
	}
	if string(got.EncryptedData) != "encrypted-data-here" {
		t.Errorf("EncryptedData = %q, want %q", got.EncryptedData, "encrypted-data-here")
	}
}

func TestGetCredential_NotFound(t *testing.T) {
	s := testStore(t)
	got, err := s.GetCredential(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetCredential() error = %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent credential")
	}
}

func TestListCredentials(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "First", Type: CredTypeSSHPassword,
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-2", Name: "Second", Type: CredTypeAPIKey,
		EncryptedData: []byte("b"), CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second),
	})

	metas, err := s.ListCredentials(ctx)
	if err != nil {
		t.Fatalf("ListCredentials() error = %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("len = %d, want 2", len(metas))
	}
	if metas[0].ID != "cred-1" {
		t.Errorf("metas[0].ID = %q, want %q", metas[0].ID, "cred-1")
	}
	if metas[1].ID != "cred-2" {
		t.Errorf("metas[1].ID = %q, want %q", metas[1].ID, "cred-2")
	}
}

func TestListCredentials_Empty(t *testing.T) {
	s := testStore(t)
	metas, err := s.ListCredentials(context.Background())
	if err != nil {
		t.Fatalf("ListCredentials() error = %v", err)
	}
	if metas != nil {
		t.Errorf("expected nil slice, got %d items", len(metas))
	}
}

func TestListCredentialsByDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "A", Type: CredTypeSSHPassword, DeviceID: "dev-1",
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-2", Name: "B", Type: CredTypeAPIKey, DeviceID: "dev-2",
		EncryptedData: []byte("b"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-3", Name: "C", Type: CredTypeSNMPv2c, DeviceID: "dev-1",
		EncryptedData: []byte("c"), CreatedAt: now, UpdatedAt: now,
	})

	metas, err := s.ListCredentialsByDevice(ctx, "dev-1")
	if err != nil {
		t.Fatalf("ListCredentialsByDevice() error = %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("len = %d, want 2", len(metas))
	}
}

func TestListCredentialsByType(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "A", Type: CredTypeSSHPassword,
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-2", Name: "B", Type: CredTypeSSHPassword,
		EncryptedData: []byte("b"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-3", Name: "C", Type: CredTypeAPIKey,
		EncryptedData: []byte("c"), CreatedAt: now, UpdatedAt: now,
	})

	metas, err := s.ListCredentialsByType(ctx, CredTypeSSHPassword)
	if err != nil {
		t.Fatalf("ListCredentialsByType() error = %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("len = %d, want 2", len(metas))
	}
}

func TestUpdateCredential(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second).UTC()

	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "Old Name", Type: CredTypeSSHPassword,
		EncryptedData: []byte("old"), CreatedAt: now, UpdatedAt: now,
	})

	updated := &CredentialRecord{
		ID: "cred-1", Name: "New Name", Type: CredTypeSSHKey,
		DeviceID: "dev-1", Description: "Updated",
		EncryptedData: []byte("new"), UpdatedAt: now.Add(time.Minute),
	}
	if err := s.UpdateCredential(ctx, updated); err != nil {
		t.Fatalf("UpdateCredential() error = %v", err)
	}

	got, _ := s.GetCredential(ctx, "cred-1")
	if got.Name != "New Name" {
		t.Errorf("Name = %q, want %q", got.Name, "New Name")
	}
	if string(got.EncryptedData) != "new" {
		t.Errorf("EncryptedData = %q, want %q", got.EncryptedData, "new")
	}
}

func TestUpdateCredential_NotFound(t *testing.T) {
	s := testStore(t)
	err := s.UpdateCredential(context.Background(), &CredentialRecord{
		ID:            "nonexistent",
		EncryptedData: []byte("x"),
		UpdatedAt:     time.Now().UTC(),
	})
	if err == nil {
		t.Error("update nonexistent credential should return error")
	}
}

func TestDeleteCredential(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "A", Type: CredTypeSSHPassword,
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})

	if err := s.DeleteCredential(ctx, "cred-1"); err != nil {
		t.Fatalf("DeleteCredential() error = %v", err)
	}

	got, _ := s.GetCredential(ctx, "cred-1")
	if got != nil {
		t.Error("credential should be deleted")
	}
}

func TestDeleteCredential_NotFound(t *testing.T) {
	s := testStore(t)
	err := s.DeleteCredential(context.Background(), "nonexistent")
	if err == nil {
		t.Error("delete nonexistent credential should return error")
	}
}

func TestCredentialCount(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	count, _ := s.CredentialCount(ctx)
	if count != 0 {
		t.Errorf("initial count = %d, want 0", count)
	}

	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "A", Type: CredTypeSSHPassword,
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-2", Name: "B", Type: CredTypeAPIKey,
		EncryptedData: []byte("b"), CreatedAt: now, UpdatedAt: now,
	})

	count, _ = s.CredentialCount(ctx)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

// --- Key Tests ---

func TestInsertKey_AndGet(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Insert parent credential first (foreign key).
	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "A", Type: CredTypeSSHPassword,
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})

	wrappedKey := []byte("wrapped-dek-bytes")
	if err := s.InsertKey(ctx, "cred-1", wrappedKey); err != nil {
		t.Fatalf("InsertKey() error = %v", err)
	}

	got, err := s.GetKey(ctx, "cred-1")
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil key")
	}
	if got.CredentialID != "cred-1" {
		t.Errorf("CredentialID = %q, want %q", got.CredentialID, "cred-1")
	}
	if string(got.WrappedKey) != "wrapped-dek-bytes" {
		t.Errorf("WrappedKey = %q, want %q", got.WrappedKey, "wrapped-dek-bytes")
	}
}

func TestGetKey_NotFound(t *testing.T) {
	s := testStore(t)
	got, err := s.GetKey(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetKey() error = %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent key")
	}
}

func TestUpdateKey(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "A", Type: CredTypeSSHPassword,
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertKey(ctx, "cred-1", []byte("old-wrapped"))

	if err := s.UpdateKey(ctx, "cred-1", []byte("new-wrapped")); err != nil {
		t.Fatalf("UpdateKey() error = %v", err)
	}

	got, _ := s.GetKey(ctx, "cred-1")
	if string(got.WrappedKey) != "new-wrapped" {
		t.Errorf("WrappedKey = %q, want %q", got.WrappedKey, "new-wrapped")
	}
}

func TestUpdateKey_NotFound(t *testing.T) {
	s := testStore(t)
	err := s.UpdateKey(context.Background(), "nonexistent", []byte("key"))
	if err == nil {
		t.Error("update nonexistent key should return error")
	}
}

func TestListAllKeys(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "A", Type: CredTypeSSHPassword,
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-2", Name: "B", Type: CredTypeAPIKey,
		EncryptedData: []byte("b"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertKey(ctx, "cred-1", []byte("key-1"))
	_ = s.InsertKey(ctx, "cred-2", []byte("key-2"))

	keys, err := s.ListAllKeys(ctx)
	if err != nil {
		t.Fatalf("ListAllKeys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len = %d, want 2", len(keys))
	}
}

func TestDeleteKey(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertCredential(ctx, &CredentialRecord{
		ID: "cred-1", Name: "A", Type: CredTypeSSHPassword,
		EncryptedData: []byte("a"), CreatedAt: now, UpdatedAt: now,
	})
	_ = s.InsertKey(ctx, "cred-1", []byte("key-1"))

	if err := s.DeleteKey(ctx, "cred-1"); err != nil {
		t.Fatalf("DeleteKey() error = %v", err)
	}

	got, _ := s.GetKey(ctx, "cred-1")
	if got != nil {
		t.Error("key should be deleted")
	}
}

// --- Audit Tests ---

func TestInsertAuditEntry_AndList(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	entry := &AuditEntry{
		CredentialID: "cred-1",
		UserID:       "user-1",
		Action:       "read",
		Purpose:      "manual_view",
		SourceIP:     "192.168.1.100",
		Timestamp:    now,
	}
	if err := s.InsertAuditEntry(ctx, entry); err != nil {
		t.Fatalf("InsertAuditEntry() error = %v", err)
	}

	entries, err := s.ListAuditEntries(ctx, "cred-1", 100)
	if err != nil {
		t.Fatalf("ListAuditEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1", len(entries))
	}
	if entries[0].Action != "read" {
		t.Errorf("Action = %q, want %q", entries[0].Action, "read")
	}
	if entries[0].Purpose != "manual_view" {
		t.Errorf("Purpose = %q, want %q", entries[0].Purpose, "manual_view")
	}
}

func TestListAuditEntries_AllCredentials(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		CredentialID: "cred-1", UserID: "u1", Action: "create", Timestamp: now,
	})
	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		CredentialID: "cred-2", UserID: "u1", Action: "read", Timestamp: now.Add(time.Second),
	})

	entries, err := s.ListAuditEntries(ctx, "", 100)
	if err != nil {
		t.Fatalf("ListAuditEntries() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
}

func TestListAuditEntries_WithLimit(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	for i := range 5 {
		_ = s.InsertAuditEntry(ctx, &AuditEntry{
			CredentialID: "cred-1", UserID: "u1", Action: "read",
			Timestamp: now.Add(time.Duration(i) * time.Second),
		})
	}

	entries, _ := s.ListAuditEntries(ctx, "cred-1", 3)
	if len(entries) != 3 {
		t.Fatalf("len = %d, want 3", len(entries))
	}
}

func TestDeleteOldAuditEntries(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	old := time.Now().Add(-48 * time.Hour).UTC()
	recent := time.Now().UTC()

	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		CredentialID: "cred-1", UserID: "u1", Action: "read", Timestamp: old,
	})
	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		CredentialID: "cred-1", UserID: "u1", Action: "read", Timestamp: recent,
	})

	cutoff := time.Now().Add(-24 * time.Hour).UTC()
	deleted, err := s.DeleteOldAuditEntries(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOldAuditEntries() error = %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	entries, _ := s.ListAuditEntries(ctx, "", 100)
	if len(entries) != 1 {
		t.Errorf("remaining = %d, want 1", len(entries))
	}
}
