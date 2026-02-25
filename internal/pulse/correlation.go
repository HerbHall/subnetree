package pulse

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// CorrelationEngine checks whether a new alert on a child device should be
// suppressed because its parent device already has active alerts within a
// configurable time window.
type CorrelationEngine struct {
	store  *PulseStore
	window time.Duration
	logger *zap.Logger
}

// NewCorrelationEngine creates a correlation engine with the given window.
func NewCorrelationEngine(store *PulseStore, window time.Duration, logger *zap.Logger) *CorrelationEngine {
	return &CorrelationEngine{
		store:  store,
		window: window,
		logger: logger,
	}
}

// CorrelationResult holds the outcome of a correlation check.
type CorrelationResult struct {
	Suppressed     bool
	ParentDeviceID string
}

// Check determines whether the alert for the given device should be
// suppressed based on parent device alert status. It returns a result
// indicating suppression state and the parent device ID if applicable.
func (c *CorrelationEngine) Check(ctx context.Context, deviceID string) (result CorrelationResult, err error) {
	parentAlerts, parentDeviceID, err := c.store.GetParentActiveAlerts(ctx, deviceID, c.window)
	if err != nil {
		return CorrelationResult{}, err
	}

	if len(parentAlerts) == 0 {
		c.logger.Debug("no parent alerts for correlation",
			zap.String("device_id", deviceID),
		)
		return CorrelationResult{Suppressed: false}, nil
	}

	c.logger.Debug("alert correlated with parent device",
		zap.String("device_id", deviceID),
		zap.String("parent_device_id", parentDeviceID),
		zap.Int("parent_active_alerts", len(parentAlerts)),
		zap.Duration("correlation_window", c.window),
	)

	return CorrelationResult{
		Suppressed:     true,
		ParentDeviceID: parentDeviceID,
	}, nil
}

// CorrelatedAlertGroup represents a parent alert with its suppressed children.
type CorrelatedAlertGroup struct {
	ParentAlert        Alert   `json:"parent_alert"`
	SuppressedChildren []Alert `json:"suppressed_children"`
}
