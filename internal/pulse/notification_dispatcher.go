package pulse

import (
	"context"
	"strings"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// NotificationDispatcher handles alert events and dispatches notifications
// to all enabled notification channels.
type NotificationDispatcher struct {
	store  *PulseStore
	logger *zap.Logger
}

// NewNotificationDispatcher creates a new dispatcher.
func NewNotificationDispatcher(store *PulseStore, logger *zap.Logger) *NotificationDispatcher {
	return &NotificationDispatcher{
		store:  store,
		logger: logger,
	}
}

// HandleAlertEvent processes an alert event from the event bus and delivers
// notifications to all enabled channels.
func (d *NotificationDispatcher) HandleAlertEvent(ctx context.Context, event plugin.Event) {
	alert, ok := event.Payload.(*Alert)
	if !ok {
		d.logger.Warn("unexpected payload type for alert event",
			zap.String("topic", event.Topic),
		)
		return
	}

	// Determine event type from topic.
	eventType := "triggered"
	if strings.HasSuffix(event.Topic, ".resolved") {
		eventType = "resolved"
	}

	channels, err := d.store.ListEnabledChannels(ctx)
	if err != nil {
		d.logger.Warn("failed to load notification channels", zap.Error(err))
		return
	}

	if len(channels) == 0 {
		return
	}

	for i := range channels {
		notifier, buildErr := buildNotifier(channels[i])
		if buildErr != nil {
			d.logger.Warn("failed to build notifier",
				zap.String("channel_id", channels[i].ID),
				zap.String("channel_type", channels[i].Type),
				zap.Error(buildErr),
			)
			continue
		}
		if notifier == nil {
			// Stubbed notifier type (e.g., email).
			d.logger.Debug("skipping stubbed notifier type",
				zap.String("channel_id", channels[i].ID),
				zap.String("channel_type", channels[i].Type),
			)
			continue
		}

		if notifyErr := notifier.Notify(ctx, alert, eventType); notifyErr != nil {
			d.logger.Warn("notification delivery failed",
				zap.String("channel_id", channels[i].ID),
				zap.String("channel_type", channels[i].Type),
				zap.String("alert_id", alert.ID),
				zap.Error(notifyErr),
			)
			continue
		}

		d.logger.Debug("notification delivered",
			zap.String("channel_id", channels[i].ID),
			zap.String("channel_type", channels[i].Type),
			zap.String("alert_id", alert.ID),
			zap.String("event_type", eventType),
		)
	}
}
