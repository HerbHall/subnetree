//go:build !windows

package service

import (
	"fmt"

	"github.com/HerbHall/subnetree/internal/scout"
	"go.uber.org/zap"
)

// ServiceName is the Windows service name. Defined on all platforms for cross-platform code.
const ServiceName = "SubNetreeScout"

// RunAsService is not supported on non-Windows platforms.
func RunAsService(_ *scout.Config, _ *zap.Logger) error {
	return fmt.Errorf("windows service mode is not supported on this platform")
}

// InstallService is not supported on non-Windows platforms.
func InstallService(_ string, _ *scout.Config) error {
	return fmt.Errorf("windows service installation is not supported on this platform")
}

// UninstallService is not supported on non-Windows platforms.
func UninstallService() error {
	return fmt.Errorf("windows service removal is not supported on this platform")
}

// IsService always returns false on non-Windows platforms.
func IsService() (bool, error) {
	return false, nil
}
