package gateway

import (
	"fmt"
	"sync"
	"time"
)

// SessionManager manages active remote access sessions using a sync.Map
// for concurrent access from HTTP handlers and maintenance goroutines.
type SessionManager struct {
	sessions    sync.Map
	maxSessions int
}

// NewSessionManager creates a new SessionManager with the given maximum session limit.
func NewSessionManager(limit int) *SessionManager {
	return &SessionManager{
		maxSessions: limit,
	}
}

// Create adds a new session. Returns an error if the maximum session limit is reached.
func (sm *SessionManager) Create(session *Session) error {
	if sm.Count() >= sm.maxSessions {
		return fmt.Errorf("maximum sessions reached (%d)", sm.maxSessions)
	}
	sm.sessions.Store(session.ID, session)
	return nil
}

// Get returns a session by ID, or nil and false if not found.
func (sm *SessionManager) Get(id string) (*Session, bool) {
	val, ok := sm.sessions.Load(id)
	if !ok {
		return nil, false
	}
	return val.(*Session), true
}

// List returns all active sessions.
func (sm *SessionManager) List() []*Session {
	var result []*Session
	sm.sessions.Range(func(_, value any) bool {
		result = append(result, value.(*Session))
		return true
	})
	return result
}

// Delete removes a session by ID. Returns true if the session was found and removed.
func (sm *SessionManager) Delete(id string) bool {
	_, loaded := sm.sessions.LoadAndDelete(id)
	return loaded
}

// Count returns the number of active sessions.
func (sm *SessionManager) Count() int {
	count := 0
	sm.sessions.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// CloseExpired removes and returns all sessions that have exceeded their expiration time.
func (sm *SessionManager) CloseExpired() []*Session {
	now := time.Now()
	var expired []*Session

	sm.sessions.Range(func(key, value any) bool {
		s := value.(*Session)
		if now.After(s.ExpiresAt) {
			sm.sessions.Delete(key)
			expired = append(expired, s)
		}
		return true
	})

	return expired
}
