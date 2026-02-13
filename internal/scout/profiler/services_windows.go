//go:build windows

package profiler

import (
	"context"

	scoutpb "github.com/HerbHall/subnetree/api/proto/v1"
	"go.uber.org/zap"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func collectServices(_ context.Context, logger *zap.Logger) ([]*scoutpb.ServiceInfo, error) {
	manager, err := mgr.Connect()
	if err != nil {
		return nil, err
	}
	defer func() { _ = manager.Disconnect() }()

	names, err := manager.ListServices()
	if err != nil {
		return nil, err
	}

	services := make([]*scoutpb.ServiceInfo, 0, len(names))
	for _, name := range names {
		svcHandle, openErr := manager.OpenService(name)
		if openErr != nil {
			continue
		}

		status, queryErr := svcHandle.Query()
		if queryErr != nil {
			svcHandle.Close()
			continue
		}

		config, cfgErr := svcHandle.Config()
		svcHandle.Close()

		si := &scoutpb.ServiceInfo{
			Name:   name,
			Status: svcStateToString(status.State),
		}

		if cfgErr == nil {
			si.DisplayName = config.DisplayName
			si.StartType = startTypeToString(config.StartType)
		}

		services = append(services, si)
	}

	logger.Debug("collected services", zap.Int("count", len(services)))
	return services, nil
}

func svcStateToString(state svc.State) string {
	switch state {
	case svc.Running:
		return "running"
	case svc.Stopped:
		return "stopped"
	case svc.StartPending:
		return "start_pending"
	case svc.StopPending:
		return "stop_pending"
	case svc.Paused:
		return "paused"
	default:
		return "unknown"
	}
}

func startTypeToString(startType uint32) string {
	switch startType {
	case mgr.StartAutomatic:
		return "auto"
	case mgr.StartManual:
		return "manual"
	case mgr.StartDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}
