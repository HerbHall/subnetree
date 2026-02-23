//go:build !linux && !windows

package recon

import (
	"context"

	"go.uber.org/zap"
)

type stubAPClientEnumerator struct{}

// NewAPClientEnumerator returns a stub APClientEnumerator for unsupported platforms.
func NewAPClientEnumerator(_ *zap.Logger) APClientEnumerator {
	return &stubAPClientEnumerator{}
}

func (s *stubAPClientEnumerator) Available() bool { return false }

func (s *stubAPClientEnumerator) Enumerate(_ context.Context) ([]APClientInfo, error) {
	return nil, nil
}
