//go:build !windows

package profiler

import (
	"context"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

func collectServices(_ context.Context, logger *zap.Logger) ([]*scoutpb.ServiceInfo, error) {
	logger.Warn("service enumeration is only supported on Windows; returning empty list")
	return nil, nil
}
