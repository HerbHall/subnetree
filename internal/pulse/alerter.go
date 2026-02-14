package pulse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// Alerter tracks consecutive check failures and manages alert lifecycle.
type Alerter struct {
	store     *PulseStore
	bus       plugin.EventBus
	threshold int
	logger    *zap.Logger

	mu       sync.Mutex
	failures map[string]int // check_id -> consecutive failure count
}

// NewAlerter creates an alerter with the given consecutive failure threshold.
func NewAlerter(store *PulseStore, bus plugin.EventBus, threshold int, logger *zap.Logger) *Alerter {
	return &Alerter{
		store:     store,
		bus:       bus,
		threshold: threshold,
		logger:    logger,
		failures:  make(map[string]int),
	}
}

// ProcessResult evaluates a check result and triggers or resolves alerts.
func (a *Alerter) ProcessResult(ctx context.Context, check Check, result *CheckResult) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if result.Success {
		a.handleSuccess(ctx, check)
	} else {
		a.handleFailure(ctx, check, result)
	}
}

// handleSuccess resets the failure counter and resolves any active alert.
func (a *Alerter) handleSuccess(ctx context.Context, check Check) {
	delete(a.failures, check.ID)

	alert, err := a.store.GetActiveAlert(ctx, check.ID)
	if err != nil {
		a.logger.Warn("failed to get active alert", zap.String("check_id", check.ID), zap.Error(err))
		return
	}
	if alert == nil {
		return
	}

	now := time.Now().UTC()
	if err := a.store.ResolveAlert(ctx, alert.ID, now); err != nil {
		a.logger.Warn("failed to resolve alert", zap.String("alert_id", alert.ID), zap.Error(err))
		return
	}

	alert.ResolvedAt = &now
	a.logger.Info("alert resolved",
		zap.String("alert_id", alert.ID),
		zap.String("check_id", check.ID),
		zap.String("device_id", check.DeviceID),
	)

	if a.bus != nil {
		a.bus.PublishAsync(ctx, plugin.Event{
			Topic:     TopicAlertResolved,
			Source:    "pulse",
			Timestamp: now,
			Payload:   alert,
		})
	}
}

// handleFailure increments the failure counter and triggers an alert if threshold reached.
func (a *Alerter) handleFailure(ctx context.Context, check Check, result *CheckResult) {
	a.failures[check.ID]++
	count := a.failures[check.ID]

	if count < a.threshold {
		return
	}

	// Check if an alert already exists for this check.
	existing, err := a.store.GetActiveAlert(ctx, check.ID)
	if err != nil {
		a.logger.Warn("failed to check existing alert", zap.String("check_id", check.ID), zap.Error(err))
		return
	}

	now := time.Now().UTC()

	if existing != nil {
		// Update severity if escalation threshold reached.
		if count >= a.threshold*2 && existing.Severity != "critical" {
			a.logger.Info("alert escalated to critical",
				zap.String("alert_id", existing.ID),
				zap.String("check_id", check.ID),
				zap.Int("consecutive_failures", count),
			)
		}
		return
	}

	// Determine severity.
	severity := "warning"
	if count >= a.threshold*2 {
		severity = "critical"
	}

	message := fmt.Sprintf("check %s failed %d consecutive times", check.ID, count)
	if result.ErrorMessage != "" {
		message = result.ErrorMessage
	}

	alert := &Alert{
		ID:                  fmt.Sprintf("alert-%s-%d", check.ID, now.UnixMilli()),
		CheckID:             check.ID,
		DeviceID:            check.DeviceID,
		Severity:            severity,
		Message:             message,
		TriggeredAt:         now,
		ConsecutiveFailures: count,
	}

	// Check if this alert should be suppressed due to an upstream device failure.
	suppressed, byDevice, suppErr := a.store.IsSuppressed(ctx, check.ID)
	if suppErr != nil {
		a.logger.Warn("suppression check failed, proceeding with alert",
			zap.String("check_id", check.ID),
			zap.Error(suppErr),
		)
	}
	if suppressed {
		alert.Suppressed = true
		alert.SuppressedBy = byDevice
	}

	if err := a.store.InsertAlert(ctx, alert); err != nil {
		a.logger.Warn("failed to insert alert", zap.String("check_id", check.ID), zap.Error(err))
		return
	}

	if alert.Suppressed {
		a.logger.Info("alert suppressed",
			zap.String("alert_id", alert.ID),
			zap.String("check_id", check.ID),
			zap.String("device_id", check.DeviceID),
			zap.String("suppressed_by", alert.SuppressedBy),
		)

		if a.bus != nil {
			a.bus.PublishAsync(ctx, plugin.Event{
				Topic:     TopicAlertSuppressed,
				Source:    "pulse",
				Timestamp: now,
				Payload:   alert,
			})
		}
		return
	}

	a.logger.Warn("alert triggered",
		zap.String("alert_id", alert.ID),
		zap.String("check_id", check.ID),
		zap.String("device_id", check.DeviceID),
		zap.String("severity", severity),
		zap.Int("consecutive_failures", count),
	)

	if a.bus != nil {
		a.bus.PublishAsync(ctx, plugin.Event{
			Topic:     TopicAlertTriggered,
			Source:    "pulse",
			Timestamp: now,
			Payload:   alert,
		})
	}
}
