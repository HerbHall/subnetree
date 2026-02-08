package pulse

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"go.uber.org/zap"
)

// newTestStore creates an in-memory PulseStore for scheduler tests.
// Named differently from testStore (if any exists in store_test.go) to avoid
// redeclaration within the same test binary.
func newTestStore(t *testing.T) *PulseStore {
	t.Helper()
	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()
	if err := db.Migrate(ctx, "pulse", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewPulseStore(db.DB())
}

func TestScheduler_StartStop(t *testing.T) {
	ps := newTestStore(t)
	executor := func(_ context.Context, _ Check) {}

	s := NewScheduler(ps, executor, 50*time.Millisecond, 2, zap.NewNop())
	s.Start(context.Background())
	time.Sleep(100 * time.Millisecond)
	s.Stop()
	// If we reach here without panic, the test passes.
}

func TestScheduler_Running(t *testing.T) {
	ps := newTestStore(t)
	executor := func(_ context.Context, _ Check) {}

	s := NewScheduler(ps, executor, 50*time.Millisecond, 2, zap.NewNop())

	if s.Running() {
		t.Error("Running() = true before Start, want false")
	}

	s.Start(context.Background())
	if !s.Running() {
		t.Error("Running() = false after Start, want true")
	}

	s.Stop()
	if s.Running() {
		t.Error("Running() = true after Stop, want false")
	}
}

func TestScheduler_ExecutesChecks(t *testing.T) {
	ps := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Insert 2 enabled checks.
	for _, c := range []Check{
		{ID: "chk-1", DeviceID: "dev-1", CheckType: "icmp", Target: "10.0.0.1", IntervalSeconds: 30, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: "chk-2", DeviceID: "dev-2", CheckType: "icmp", Target: "10.0.0.2", IntervalSeconds: 30, Enabled: true, CreatedAt: now, UpdatedAt: now},
	} {
		if err := ps.InsertCheck(ctx, &c); err != nil {
			t.Fatalf("InsertCheck(%s): %v", c.ID, err)
		}
	}

	var counter atomic.Int64
	executor := func(_ context.Context, _ Check) {
		counter.Add(1)
	}

	s := NewScheduler(ps, executor, 50*time.Millisecond, 4, zap.NewNop())
	s.Start(context.Background())
	// Wait for the initial tick plus one interval tick to be safe.
	time.Sleep(100 * time.Millisecond)
	s.Stop()

	got := counter.Load()
	if got < 2 {
		t.Errorf("executor called %d times, want >= 2", got)
	}
}

func TestScheduler_WorkerConcurrencyLimit(t *testing.T) {
	ps := newTestStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Insert 5 checks using fmt-based IDs for clarity.
	for i := range 5 {
		id := fmt.Sprintf("chk-%d", i)
		c := Check{
			ID:              id,
			DeviceID:        fmt.Sprintf("dev-%d", i),
			CheckType:       "icmp",
			Target:          fmt.Sprintf("10.0.0.%d", i+1),
			IntervalSeconds: 30,
			Enabled:         true,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := ps.InsertCheck(ctx, &c); err != nil {
			t.Fatalf("InsertCheck(%s): %v", c.ID, err)
		}
	}

	var current atomic.Int64
	var peak atomic.Int64

	executor := func(_ context.Context, _ Check) {
		cur := current.Add(1)
		// Update peak if current concurrency exceeds it.
		for {
			p := peak.Load()
			if cur <= p || peak.CompareAndSwap(p, cur) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		current.Add(-1)
	}

	// Use a long interval so the tick's internal context.WithTimeout does
	// not expire before all workers finish. The scheduler fires immediately
	// on Start, so we only need to wait long enough for the first tick.
	maxWorkers := 2
	s := NewScheduler(ps, executor, 5*time.Second, maxWorkers, zap.NewNop())
	s.Start(context.Background())
	// 5 checks / 2 workers * 50ms sleep = ~150ms; wait 500ms for margin.
	time.Sleep(500 * time.Millisecond)
	s.Stop()

	peakVal := peak.Load()
	if peakVal > int64(maxWorkers) {
		t.Errorf("peak concurrency = %d, want <= %d", peakVal, maxWorkers)
	}
	if peakVal == 0 {
		t.Error("peak concurrency = 0, executor was never called")
	}
}
