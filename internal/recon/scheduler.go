package recon

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"go.uber.org/zap"
)

// ScanScheduler runs recurring network scans on a configurable interval,
// respecting quiet hours when no scans should be triggered.
type ScanScheduler struct {
	cfg          ScheduleConfig
	orchestrator *ScanOrchestrator
	store        *ReconStore
	activeScans  *sync.Map
	wg           *sync.WaitGroup
	newScanCtx   func() (context.Context, context.CancelFunc)
	logger       *zap.Logger
	nowFunc      func() time.Time

	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewScanScheduler creates a new scheduler. The newScanCtx function should
// return a child context from the module's scan context for cancellation.
func NewScanScheduler(
	cfg ScheduleConfig,
	orchestrator *ScanOrchestrator,
	store *ReconStore,
	activeScans *sync.Map,
	wg *sync.WaitGroup,
	newScanCtx func() (context.Context, context.CancelFunc),
	logger *zap.Logger,
) *ScanScheduler {
	return &ScanScheduler{
		cfg:          cfg,
		orchestrator: orchestrator,
		store:        store,
		activeScans:  activeScans,
		wg:           wg,
		newScanCtx:   newScanCtx,
		logger:       logger,
		nowFunc:      time.Now,
		stopCh:       make(chan struct{}),
	}
}

// Run starts the ticker loop. It blocks until the context is cancelled
// or Stop is called. The caller should run this in a goroutine.
func (s *ScanScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	s.logger.Info("scan scheduler started",
		zap.Duration("interval", s.cfg.Interval),
		zap.String("subnet", s.cfg.Subnet),
		zap.String("quiet_start", s.cfg.QuietStart),
		zap.String("quiet_end", s.cfg.QuietEnd),
	)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scan scheduler stopped (context cancelled)")
			return
		case <-s.stopCh:
			s.logger.Info("scan scheduler stopped")
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

// Stop signals the scheduler to exit its run loop.
func (s *ScanScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

// tick is called on each interval. It checks quiet hours and active scans
// before triggering a new scan.
func (s *ScanScheduler) tick() {
	now := s.nowFunc()

	if isQuietHours(now, s.cfg.QuietStart, s.cfg.QuietEnd) {
		s.logger.Debug("scheduled scan skipped: quiet hours",
			zap.String("quiet_start", s.cfg.QuietStart),
			zap.String("quiet_end", s.cfg.QuietEnd),
		)
		return
	}

	if s.hasActiveScan() {
		s.logger.Debug("scheduled scan skipped: scan already running")
		return
	}

	s.triggerScan()
}

// hasActiveScan returns true if any scan is currently running.
func (s *ScanScheduler) hasActiveScan() bool {
	active := false
	s.activeScans.Range(func(_, _ any) bool {
		active = true
		return false
	})
	return active
}

// triggerScan starts a new scheduled scan, mirroring the pattern from handleScan.
func (s *ScanScheduler) triggerScan() {
	subnet := s.cfg.Subnet

	// Validate CIDR before starting.
	if _, _, err := net.ParseCIDR(subnet); err != nil {
		s.logger.Error("scheduled scan: invalid subnet",
			zap.String("subnet", subnet),
			zap.Error(err),
		)
		return
	}

	scanID := fmt.Sprintf("scheduled-%d", s.nowFunc().UnixMilli())

	scan := &models.ScanResult{
		ID:     scanID,
		Subnet: subnet,
		Status: "running",
	}

	ctx := context.Background()
	if err := s.store.CreateScan(ctx, scan); err != nil {
		s.logger.Error("scheduled scan: failed to create scan record",
			zap.Error(err),
		)
		return
	}

	scanCtx, cancel := s.newScanCtx()
	s.activeScans.Store(scanID, cancel)
	s.wg.Add(1)

	s.logger.Info("scheduled scan started",
		zap.String("scan_id", scanID),
		zap.String("subnet", subnet),
	)

	go func() {
		defer s.wg.Done()
		defer s.activeScans.Delete(scanID)
		s.orchestrator.RunScan(scanCtx, scanID, subnet)
		s.logger.Info("scheduled scan completed",
			zap.String("scan_id", scanID),
		)
	}()
}

// isQuietHours returns true if the given time falls within the quiet window
// defined by startHHMM and endHHMM (format "HH:MM"). Supports overnight
// ranges (e.g., "23:00" to "06:00"). Returns false if either value is empty
// or cannot be parsed.
func isQuietHours(now time.Time, startHHMM, endHHMM string) bool {
	if startHHMM == "" || endHHMM == "" {
		return false
	}

	startMin, ok := parseHHMM(startHHMM)
	if !ok {
		return false
	}
	endMin, ok := parseHHMM(endHHMM)
	if !ok {
		return false
	}

	nowMin := now.Hour()*60 + now.Minute()

	if startMin <= endMin {
		// Same-day range: e.g., 09:00 to 17:00
		return nowMin >= startMin && nowMin < endMin
	}
	// Overnight range: e.g., 23:00 to 06:00
	return nowMin >= startMin || nowMin < endMin
}

// parseHHMM parses a "HH:MM" string into minutes since midnight.
func parseHHMM(s string) (int, bool) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}
