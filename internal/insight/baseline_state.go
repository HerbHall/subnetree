package insight

import (
	"sync"

	"github.com/HerbHall/subnetree/internal/insight/anomaly"
	"github.com/HerbHall/subnetree/internal/insight/baseline"
)

// baselineState holds the in-memory statistical state for a single device+metric pair.
type baselineState struct {
	EWMA *baseline.EWMA
	CUSUM *anomaly.CUSUM
}

// stateKey returns a map key for a device+metric pair.
func stateKey(deviceID, metric string) string {
	return deviceID + ":" + metric
}

// stateManager provides thread-safe access to per-device/metric baseline state.
type stateManager struct {
	mu     sync.RWMutex
	states map[string]*baselineState
	alpha  float64
	drift  float64
	threshold float64
}

// newStateManager creates a new state manager with the given EWMA alpha and CUSUM parameters.
func newStateManager(alpha, cusumDrift, cusumThreshold float64) *stateManager {
	return &stateManager{
		states:    make(map[string]*baselineState),
		alpha:     alpha,
		drift:     cusumDrift,
		threshold: cusumThreshold,
	}
}

// getOrCreate returns the state for a device+metric pair, creating it if needed.
func (sm *stateManager) getOrCreate(deviceID, metric string) *baselineState {
	key := stateKey(deviceID, metric)
	sm.mu.RLock()
	s, ok := sm.states[key]
	sm.mu.RUnlock()
	if ok {
		return s
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Double-check after acquiring write lock
	if s, ok = sm.states[key]; ok {
		return s
	}
	s = &baselineState{
		EWMA:  baseline.NewEWMA(sm.alpha),
		CUSUM: anomaly.NewCUSUM(sm.drift, sm.threshold),
	}
	sm.states[key] = s
	return s
}

// count returns the number of tracked device+metric pairs.
func (sm *stateManager) count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.states)
}

// snapshot returns a copy of all keys for iteration (avoids holding lock during DB writes).
func (sm *stateManager) snapshot() map[string]*baselineState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	cp := make(map[string]*baselineState, len(sm.states))
	for k, v := range sm.states {
		cp[k] = v
	}
	return cp
}
