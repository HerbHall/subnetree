package gateway

import (
	"testing"
	"time"
)

func newTestSession(id, deviceID string, expiresAt time.Time) *Session {
	return &Session{
		ID:          id,
		DeviceID:    deviceID,
		UserID:      "user-1",
		SessionType: SessionTypeProxy,
		Target:      ProxyTarget{Host: "192.168.1.1", Port: 80},
		SourceIP:    "10.0.0.1",
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   expiresAt,
	}
}

func TestSessionManager_Create(t *testing.T) {
	sm := NewSessionManager(10)
	session := newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute))

	if err := sm.Create(session); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if sm.Count() != 1 {
		t.Errorf("Count() = %d, want 1", sm.Count())
	}
}

func TestSessionManager_Create_MaxSessions(t *testing.T) {
	sm := NewSessionManager(2)

	s1 := newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute))
	s2 := newTestSession("s2", "dev-2", time.Now().Add(30*time.Minute))
	s3 := newTestSession("s3", "dev-3", time.Now().Add(30*time.Minute))

	if err := sm.Create(s1); err != nil {
		t.Fatalf("Create(s1) error = %v", err)
	}
	if err := sm.Create(s2); err != nil {
		t.Fatalf("Create(s2) error = %v", err)
	}
	if err := sm.Create(s3); err == nil {
		t.Error("Create(s3) should fail when max sessions reached")
	}
}

func TestSessionManager_Get_Found(t *testing.T) {
	sm := NewSessionManager(10)
	session := newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute))
	_ = sm.Create(session)

	got, ok := sm.Get("s1")
	if !ok {
		t.Fatal("Get() returned false for existing session")
	}
	if got.ID != "s1" {
		t.Errorf("ID = %q, want %q", got.ID, "s1")
	}
	if got.DeviceID != "dev-1" {
		t.Errorf("DeviceID = %q, want %q", got.DeviceID, "dev-1")
	}
}

func TestSessionManager_Get_NotFound(t *testing.T) {
	sm := NewSessionManager(10)

	_, ok := sm.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for nonexistent session")
	}
}

func TestSessionManager_List(t *testing.T) {
	sm := NewSessionManager(10)
	_ = sm.Create(newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute)))
	_ = sm.Create(newTestSession("s2", "dev-2", time.Now().Add(30*time.Minute)))
	_ = sm.Create(newTestSession("s3", "dev-3", time.Now().Add(30*time.Minute)))

	sessions := sm.List()
	if len(sessions) != 3 {
		t.Errorf("List() returned %d sessions, want 3", len(sessions))
	}
}

func TestSessionManager_List_Empty(t *testing.T) {
	sm := NewSessionManager(10)

	sessions := sm.List()
	if len(sessions) != 0 {
		t.Errorf("List() returned %d sessions for empty manager, want 0", len(sessions))
	}
}

func TestSessionManager_Delete(t *testing.T) {
	sm := NewSessionManager(10)
	_ = sm.Create(newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute)))

	if !sm.Delete("s1") {
		t.Error("Delete() returned false for existing session")
	}
	if sm.Count() != 0 {
		t.Errorf("Count() = %d after delete, want 0", sm.Count())
	}
}

func TestSessionManager_Delete_NotFound(t *testing.T) {
	sm := NewSessionManager(10)

	if sm.Delete("nonexistent") {
		t.Error("Delete() returned true for nonexistent session")
	}
}

func TestSessionManager_Count(t *testing.T) {
	sm := NewSessionManager(10)

	if sm.Count() != 0 {
		t.Errorf("initial Count() = %d, want 0", sm.Count())
	}

	_ = sm.Create(newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute)))
	_ = sm.Create(newTestSession("s2", "dev-2", time.Now().Add(30*time.Minute)))

	if sm.Count() != 2 {
		t.Errorf("Count() = %d, want 2", sm.Count())
	}

	sm.Delete("s1")

	if sm.Count() != 1 {
		t.Errorf("Count() after delete = %d, want 1", sm.Count())
	}
}

func TestSessionManager_CloseExpired(t *testing.T) {
	sm := NewSessionManager(10)

	// Create one expired and one active session.
	expired := newTestSession("s-expired", "dev-1", time.Now().Add(-1*time.Minute))
	active := newTestSession("s-active", "dev-2", time.Now().Add(30*time.Minute))

	_ = sm.Create(expired)
	_ = sm.Create(active)

	closedSessions := sm.CloseExpired()
	if len(closedSessions) != 1 {
		t.Fatalf("CloseExpired() returned %d sessions, want 1", len(closedSessions))
	}
	if closedSessions[0].ID != "s-expired" {
		t.Errorf("expired session ID = %q, want %q", closedSessions[0].ID, "s-expired")
	}

	// Verify the active session is still present.
	if sm.Count() != 1 {
		t.Errorf("Count() after CloseExpired = %d, want 1", sm.Count())
	}
	_, ok := sm.Get("s-active")
	if !ok {
		t.Error("active session should still exist after CloseExpired")
	}
}

func TestSessionManager_CloseExpired_None(t *testing.T) {
	sm := NewSessionManager(10)
	_ = sm.Create(newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute)))

	closed := sm.CloseExpired()
	if len(closed) != 0 {
		t.Errorf("CloseExpired() returned %d sessions, want 0", len(closed))
	}
}

func TestSession_ByteCounters(t *testing.T) {
	s := &Session{}

	s.BytesIn.Store(1024)
	s.BytesOut.Store(2048)

	if s.BytesInCount() != 1024 {
		t.Errorf("BytesInCount() = %d, want 1024", s.BytesInCount())
	}
	if s.BytesOutCount() != 2048 {
		t.Errorf("BytesOutCount() = %d, want 2048", s.BytesOutCount())
	}
}

func TestSession_ToView(t *testing.T) {
	s := newTestSession("s1", "dev-1", time.Now().Add(30*time.Minute))
	s.BytesIn.Store(100)
	s.BytesOut.Store(200)

	v := s.toView()
	if v.ID != "s1" {
		t.Errorf("ID = %q, want %q", v.ID, "s1")
	}
	if v.BytesIn != 100 {
		t.Errorf("BytesIn = %d, want 100", v.BytesIn)
	}
	if v.BytesOut != 200 {
		t.Errorf("BytesOut = %d, want 200", v.BytesOut)
	}
}
