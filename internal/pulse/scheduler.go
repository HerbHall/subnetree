package pulse

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CheckExecutor is called by the scheduler for each enabled check.
type CheckExecutor func(ctx context.Context, check Check)

// Scheduler runs monitoring checks on a periodic interval using a worker pool.
type Scheduler struct {
	store    *PulseStore
	executor CheckExecutor
	interval time.Duration
	workers  int
	logger   *zap.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewScheduler creates a scheduler that dispatches checks to the executor.
func NewScheduler(store *PulseStore, executor CheckExecutor, interval time.Duration, workers int, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		store:    store,
		executor: executor,
		interval: interval,
		workers:  workers,
		logger:   logger,
	}
}

// Start begins the scheduling loop. Blocks until Stop is called.
func (s *Scheduler) Start(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		// Run immediately on start, then on each tick.
		s.tick()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.tick()
			}
		}
	}()
}

// Stop signals the scheduler to stop and waits for completion.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// Running reports whether the scheduler loop is active.
func (s *Scheduler) Running() bool {
	return s.ctx != nil && s.ctx.Err() == nil
}

// tick loads all enabled checks and dispatches them to the worker pool.
func (s *Scheduler) tick() {
	if s.store == nil {
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, s.interval)
	defer cancel()

	checks, err := s.store.ListEnabledChecks(ctx)
	if err != nil {
		s.logger.Warn("scheduler: failed to load checks", zap.Error(err))
		return
	}

	if len(checks) == 0 {
		return
	}

	// Semaphore-based worker pool.
	sem := make(chan struct{}, s.workers)
	var wg sync.WaitGroup

dispatch:
	for i := range checks {
		select {
		case <-s.ctx.Done():
			break dispatch
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(c Check) {
			defer wg.Done()
			defer func() { <-sem }()
			s.executor(ctx, c)
		}(checks[i])
	}

	wg.Wait()
}
