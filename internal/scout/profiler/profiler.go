package profiler

import (
	"context"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

// Profiler collects hardware, software, and service inventory from the host.
type Profiler struct {
	logger *zap.Logger
}

// NewProfiler creates a new system profiler.
func NewProfiler(logger *zap.Logger) *Profiler {
	return &Profiler{logger: logger}
}

// CollectProfile gathers a complete system profile snapshot.
func (p *Profiler) CollectProfile(ctx context.Context) (*scoutpb.SystemProfile, error) {
	profile := &scoutpb.SystemProfile{}

	hw, err := collectHardware(ctx, p.logger)
	if err != nil {
		p.logger.Warn("hardware profile collection failed", zap.Error(err))
	} else {
		profile.Hardware = hw
	}

	sw, err := collectSoftware(ctx, p.logger)
	if err != nil {
		p.logger.Warn("software inventory collection failed", zap.Error(err))
	} else {
		profile.Software = sw
	}

	svcs, err := collectServices(ctx, p.logger)
	if err != nil {
		p.logger.Warn("services collection failed", zap.Error(err))
	} else {
		profile.Services = svcs
	}

	return profile, nil
}
