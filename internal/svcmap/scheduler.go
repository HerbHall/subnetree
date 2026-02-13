package svcmap

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// AgentLister provides a list of agents with their device mappings.
type AgentLister interface {
	ListAgentsWithDevice(ctx context.Context) ([]AgentRef, error)
}

// AgentRef is a lightweight reference to an agent and its device.
type AgentRef struct {
	AgentID  string
	DeviceID string
	Hostname string
}

// Scheduler runs periodic service correlation across all agents.
type Scheduler struct {
	correlator *Correlator
	svcSource  ServiceSource
	appSource  AppSource
	agents     AgentLister
	interval   time.Duration
	logger     *zap.Logger
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewScheduler creates a new correlation scheduler.
func NewScheduler(
	correlator *Correlator,
	svcSource ServiceSource,
	appSource AppSource,
	agents AgentLister,
	interval time.Duration,
	logger *zap.Logger,
) *Scheduler {
	return &Scheduler{
		correlator: correlator,
		svcSource:  svcSource,
		appSource:  appSource,
		agents:     agents,
		interval:   interval,
		logger:     logger,
		done:       make(chan struct{}),
	}
}

// Start begins periodic correlation in a background goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)
	go s.run(ctx)
	s.logger.Info("svcmap scheduler started", zap.Duration("interval", s.interval))
}

// Stop cancels the scheduler and waits for the goroutine to finish.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	<-s.done
	s.logger.Info("svcmap scheduler stopped")
}

func (s *Scheduler) run(ctx context.Context) {
	defer close(s.done)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.correlateAll(ctx)
		}
	}
}

func (s *Scheduler) correlateAll(ctx context.Context) {
	agents, err := s.agents.ListAgentsWithDevice(ctx)
	if err != nil {
		s.logger.Warn("failed to list agents for correlation", zap.Error(err))
		return
	}

	if len(agents) == 0 {
		return
	}

	var correlated int
	for i := range agents {
		if agents[i].DeviceID == "" {
			continue
		}
		err := s.correlator.CorrelateDevice(ctx, agents[i].DeviceID, agents[i].AgentID, s.svcSource, s.appSource)
		if err != nil {
			s.logger.Warn("correlation failed",
				zap.String("device_id", agents[i].DeviceID),
				zap.String("agent_id", agents[i].AgentID),
				zap.Error(err))
			continue
		}
		correlated++
	}

	s.logger.Debug("correlation cycle complete",
		zap.Int("agents", len(agents)),
		zap.Int("correlated", correlated))
}
