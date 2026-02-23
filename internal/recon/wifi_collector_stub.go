//go:build !linux && !windows

package recon

import (
	"context"

	"go.uber.org/zap"
)

type stubWifiScanner struct{}

// NewWifiScanner returns a no-op scanner on unsupported platforms.
func NewWifiScanner(_ *zap.Logger) WifiScanner { return &stubWifiScanner{} }

// Available always returns false on unsupported platforms.
func (s *stubWifiScanner) Available() bool { return false }

// Scan returns nil on unsupported platforms.
func (s *stubWifiScanner) Scan(_ context.Context) ([]AccessPointInfo, error) { return nil, nil }
