package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
)

func testStore(t *testing.T) *GatewayStore {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "gateway", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewGatewayStore(db.DB())
}

func TestInsertAuditEntry_AndList(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	entry := &AuditEntry{
		SessionID:   "sess-1",
		DeviceID:    "dev-1",
		UserID:      "user-1",
		SessionType: string(SessionTypeProxy),
		Target:      "192.168.1.1:80",
		Action:      "created",
		BytesIn:     1024,
		BytesOut:    2048,
		SourceIP:    "10.0.0.1",
		Timestamp:   now,
	}
	if err := s.InsertAuditEntry(ctx, entry); err != nil {
		t.Fatalf("InsertAuditEntry() error = %v", err)
	}

	entries, err := s.ListAuditEntries(ctx, "", 100)
	if err != nil {
		t.Fatalf("ListAuditEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1", len(entries))
	}
	if entries[0].SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", entries[0].SessionID, "sess-1")
	}
	if entries[0].Action != "created" {
		t.Errorf("Action = %q, want %q", entries[0].Action, "created")
	}
	if entries[0].BytesIn != 1024 {
		t.Errorf("BytesIn = %d, want 1024", entries[0].BytesIn)
	}
	if entries[0].BytesOut != 2048 {
		t.Errorf("BytesOut = %d, want 2048", entries[0].BytesOut)
	}
}

func TestListAuditEntries_FilterByDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		SessionID: "s1", DeviceID: "dev-1", SessionType: "http_proxy",
		Target: "192.168.1.1:80", Action: "created", Timestamp: now,
	})
	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		SessionID: "s2", DeviceID: "dev-2", SessionType: "ssh",
		Target: "192.168.1.2:22", Action: "created", Timestamp: now,
	})
	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		SessionID: "s3", DeviceID: "dev-1", SessionType: "http_proxy",
		Target: "192.168.1.1:443", Action: "closed", Timestamp: now,
	})

	entries, err := s.ListAuditEntries(ctx, "dev-1", 100)
	if err != nil {
		t.Fatalf("ListAuditEntries() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len = %d, want 2", len(entries))
	}
}

func TestListAuditEntries_Empty(t *testing.T) {
	s := testStore(t)
	entries, err := s.ListAuditEntries(context.Background(), "", 100)
	if err != nil {
		t.Fatalf("ListAuditEntries() error = %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil slice, got %d items", len(entries))
	}
}

func TestListAuditEntries_WithLimit(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	for i := range 5 {
		_ = s.InsertAuditEntry(ctx, &AuditEntry{
			SessionID: "s1", DeviceID: "dev-1", SessionType: "http_proxy",
			Target: "192.168.1.1:80", Action: "ping",
			Timestamp: now.Add(time.Duration(i) * time.Second),
		})
	}

	entries, _ := s.ListAuditEntries(ctx, "", 3)
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
		SessionID: "s1", DeviceID: "dev-1", SessionType: "http_proxy",
		Target: "192.168.1.1:80", Action: "created", Timestamp: old,
	})
	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		SessionID: "s2", DeviceID: "dev-1", SessionType: "http_proxy",
		Target: "192.168.1.1:80", Action: "created", Timestamp: recent,
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

func TestListAuditEntriesByDevice(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		SessionID: "s1", DeviceID: "dev-1", SessionType: "http_proxy",
		Target: "192.168.1.1:80", Action: "created", Timestamp: now,
	})
	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		SessionID: "s2", DeviceID: "dev-2", SessionType: "ssh",
		Target: "192.168.1.2:22", Action: "created", Timestamp: now,
	})

	entries, err := s.ListAuditEntriesByDevice(ctx, "dev-1", 100)
	if err != nil {
		t.Fatalf("ListAuditEntriesByDevice() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len = %d, want 1", len(entries))
	}
	if entries[0].DeviceID != "dev-1" {
		t.Errorf("DeviceID = %q, want %q", entries[0].DeviceID, "dev-1")
	}
}

func TestDeleteOldAuditEntries_NoneToDelete(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	recent := time.Now().UTC()
	_ = s.InsertAuditEntry(ctx, &AuditEntry{
		SessionID: "s1", DeviceID: "dev-1", SessionType: "http_proxy",
		Target: "192.168.1.1:80", Action: "created", Timestamp: recent,
	})

	cutoff := time.Now().Add(-24 * time.Hour).UTC()
	deleted, err := s.DeleteOldAuditEntries(ctx, cutoff)
	if err != nil {
		t.Fatalf("DeleteOldAuditEntries() error = %v", err)
	}
	if deleted != 0 {
		t.Errorf("deleted = %d, want 0", deleted)
	}
}
