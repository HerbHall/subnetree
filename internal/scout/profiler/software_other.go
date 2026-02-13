//go:build !windows

package profiler

import (
	"context"
	"runtime"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
)

func collectSoftware(_ context.Context, logger *zap.Logger) (*scoutpb.SoftwareInventory, error) {
	logger.Warn("software inventory is only fully supported on Windows; returning basic OS info")
	return &scoutpb.SoftwareInventory{
		OsName:    runtime.GOOS,
		OsVersion: runtime.GOARCH,
	}, nil
}
