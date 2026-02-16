package recon

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/internal/store"
	"go.uber.org/zap"
)

func TestIsQuietHours(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		start     string
		end       string
		wantQuiet bool
	}{
		{
			name:      "empty strings returns false",
			now:       time.Date(2026, 2, 16, 14, 0, 0, 0, time.UTC),
			start:     "",
			end:       "",
			wantQuiet: false,
		},
		{
			name:      "only start set returns false",
			now:       time.Date(2026, 2, 16, 14, 0, 0, 0, time.UTC),
			start:     "23:00",
			end:       "",
			wantQuiet: false,
		},
		{
			name:      "only end set returns false",
			now:       time.Date(2026, 2, 16, 14, 0, 0, 0, time.UTC),
			start:     "",
			end:       "06:00",
			wantQuiet: false,
		},
		{
			name:      "same-day range: inside quiet hours",
			now:       time.Date(2026, 2, 16, 10, 30, 0, 0, time.UTC),
			start:     "09:00",
			end:       "17:00",
			wantQuiet: true,
		},
		{
			name:      "same-day range: before quiet hours",
			now:       time.Date(2026, 2, 16, 8, 0, 0, 0, time.UTC),
			start:     "09:00",
			end:       "17:00",
			wantQuiet: false,
		},
		{
			name:      "same-day range: after quiet hours",
			now:       time.Date(2026, 2, 16, 18, 0, 0, 0, time.UTC),
			start:     "09:00",
			end:       "17:00",
			wantQuiet: false,
		},
		{
			name:      "same-day range: at start boundary",
			now:       time.Date(2026, 2, 16, 9, 0, 0, 0, time.UTC),
			start:     "09:00",
			end:       "17:00",
			wantQuiet: true,
		},
		{
			name:      "same-day range: at end boundary (exclusive)",
			now:       time.Date(2026, 2, 16, 17, 0, 0, 0, time.UTC),
			start:     "09:00",
			end:       "17:00",
			wantQuiet: false,
		},
		{
			name:      "overnight range: late at night (inside)",
			now:       time.Date(2026, 2, 16, 23, 30, 0, 0, time.UTC),
			start:     "23:00",
			end:       "06:00",
			wantQuiet: true,
		},
		{
			name:      "overnight range: early morning (inside)",
			now:       time.Date(2026, 2, 16, 3, 0, 0, 0, time.UTC),
			start:     "23:00",
			end:       "06:00",
			wantQuiet: true,
		},
		{
			name:      "overnight range: afternoon (outside)",
			now:       time.Date(2026, 2, 16, 14, 0, 0, 0, time.UTC),
			start:     "23:00",
			end:       "06:00",
			wantQuiet: false,
		},
		{
			name:      "overnight range: at start boundary",
			now:       time.Date(2026, 2, 16, 23, 0, 0, 0, time.UTC),
			start:     "23:00",
			end:       "06:00",
			wantQuiet: true,
		},
		{
			name:      "overnight range: at end boundary (exclusive)",
			now:       time.Date(2026, 2, 16, 6, 0, 0, 0, time.UTC),
			start:     "23:00",
			end:       "06:00",
			wantQuiet: false,
		},
		{
			name:      "invalid start format returns false",
			now:       time.Date(2026, 2, 16, 14, 0, 0, 0, time.UTC),
			start:     "not-a-time",
			end:       "06:00",
			wantQuiet: false,
		},
		{
			name:      "invalid end format returns false",
			now:       time.Date(2026, 2, 16, 14, 0, 0, 0, time.UTC),
			start:     "23:00",
			end:       "bad",
			wantQuiet: false,
		},
		{
			name:      "midnight to midnight is always quiet",
			now:       time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC),
			start:     "00:00",
			end:       "00:00",
			wantQuiet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isQuietHours(tt.now, tt.start, tt.end)
			if got != tt.wantQuiet {
				t.Errorf("isQuietHours(%s, %q, %q) = %v, want %v",
					tt.now.Format("15:04"), tt.start, tt.end, got, tt.wantQuiet)
			}
		})
	}
}

func TestParseHHMM(t *testing.T) {
	tests := []struct {
		input   string
		wantMin int
		wantOK  bool
	}{
		{"00:00", 0, true},
		{"23:59", 1439, true},
		{"12:30", 750, true},
		{"06:00", 360, true},
		{"bad", 0, false},
		{"25:00", 0, false},
		{"12:60", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotMin, gotOK := parseHHMM(tt.input)
			if gotOK != tt.wantOK {
				t.Errorf("parseHHMM(%q) ok = %v, want %v", tt.input, gotOK, tt.wantOK)
			}
			if gotOK && gotMin != tt.wantMin {
				t.Errorf("parseHHMM(%q) = %d, want %d", tt.input, gotMin, tt.wantMin)
			}
		})
	}
}

func setupTestScheduler(t *testing.T, cfg ScheduleConfig) (*ScanScheduler, *ReconStore) {
	t.Helper()

	db, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	if err := db.Migrate(ctx, "recon", migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	s := NewReconStore(db.DB())
	bus := &mockEventBus{}
	logger := zap.NewNop()

	pinger := &noopPinger{}
	orchestrator := NewScanOrchestrator(s, bus, NewOUITable(), pinger, nil, logger)

	var activeScans sync.Map
	var wg sync.WaitGroup

	parentCtx, parentCancel := context.WithCancel(context.Background())
	t.Cleanup(parentCancel)

	newScanCtx := func() (context.Context, context.CancelFunc) {
		return context.WithCancel(parentCtx)
	}

	sched := NewScanScheduler(cfg, orchestrator, s, &activeScans, &wg, newScanCtx, logger)
	return sched, s
}

// noopPinger is a PingScanner that immediately returns with no results.
type noopPinger struct{}

func (p *noopPinger) Scan(_ context.Context, _ *net.IPNet, _ chan<- HostResult) error {
	return nil
}

func TestScheduler_StartsAndStops(t *testing.T) {
	cfg := ScheduleConfig{
		Enabled:  true,
		Interval: 50 * time.Millisecond,
		Subnet:   "192.168.1.0/24",
	}
	sched, _ := setupTestScheduler(t, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	// Let it run at least one tick.
	time.Sleep(100 * time.Millisecond)

	cancel()

	select {
	case <-done:
		// Scheduler stopped cleanly.
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop within 2 seconds after context cancellation")
	}
}

func TestScheduler_StopMethodTerminatesRun(t *testing.T) {
	cfg := ScheduleConfig{
		Enabled:  true,
		Interval: 50 * time.Millisecond,
		Subnet:   "192.168.1.0/24",
	}
	sched, _ := setupTestScheduler(t, cfg)

	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	sched.Stop()

	select {
	case <-done:
		// Scheduler stopped cleanly via Stop().
	case <-time.After(2 * time.Second):
		t.Fatal("scheduler did not stop within 2 seconds after Stop()")
	}
}

func TestScheduler_SkipsQuietHours(t *testing.T) {
	cfg := ScheduleConfig{
		Enabled:    true,
		Interval:   50 * time.Millisecond,
		Subnet:     "192.168.1.0/24",
		QuietStart: "00:00",
		QuietEnd:   "23:59",
	}
	sched, s := setupTestScheduler(t, cfg)

	// Override nowFunc to always return a time within quiet hours.
	sched.nowFunc = func() time.Time {
		return time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	// Wait several ticks.
	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	// No scans should have been created during quiet hours.
	scans, err := s.ListScans(context.Background(), 100, 0)
	if err != nil {
		t.Fatalf("ListScans: %v", err)
	}
	if len(scans) != 0 {
		t.Errorf("expected 0 scans during quiet hours, got %d", len(scans))
	}
}

func TestScheduler_TriggersScans(t *testing.T) {
	cfg := ScheduleConfig{
		Enabled:  true,
		Interval: 50 * time.Millisecond,
		Subnet:   "10.0.0.0/24",
	}
	sched, s := setupTestScheduler(t, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	// Wait enough time for at least one tick to fire and the scan goroutine to create a record.
	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	// At least one scan should have been created.
	scans, err := s.ListScans(context.Background(), 100, 0)
	if err != nil {
		t.Fatalf("ListScans: %v", err)
	}
	if len(scans) == 0 {
		t.Error("expected at least 1 scheduled scan, got 0")
	}

	// Verify scan ID prefix.
	for _, scan := range scans {
		if len(scan.ID) < 10 || scan.ID[:10] != "scheduled-" {
			t.Errorf("scan ID %q does not start with 'scheduled-'", scan.ID)
		}
	}
}

func TestScheduler_SkipsWhenScanAlreadyActive(t *testing.T) {
	cfg := ScheduleConfig{
		Enabled:  true,
		Interval: 50 * time.Millisecond,
		Subnet:   "10.0.0.0/24",
	}
	sched, s := setupTestScheduler(t, cfg)

	// Pre-populate an active scan in the map to simulate a running scan.
	sched.activeScans.Store("existing-scan", context.CancelFunc(func() {}))

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sched.Run(ctx)
		close(done)
	}()

	// Let several ticks fire.
	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	// No new scans should have been created since one was already active.
	scans, err := s.ListScans(context.Background(), 100, 0)
	if err != nil {
		t.Fatalf("ListScans: %v", err)
	}
	if len(scans) != 0 {
		t.Errorf("expected 0 scans when scan already active, got %d", len(scans))
	}
}
