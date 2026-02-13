//go:build !windows

package profiler

import (
	"context"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

func collectHardware(_ context.Context, logger *zap.Logger) (*scoutpb.HardwareProfile, error) {
	logger.Warn("hardware profiling is only supported on Windows; returning empty profile")
	return &scoutpb.HardwareProfile{}, nil
}
