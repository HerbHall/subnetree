package pulse

// Event topics consumed by the Pulse module.
const (
	TopicDeviceDiscovered = "recon.device.discovered"
)

// Event topics published by the Pulse module.
const (
	TopicMetricsCollected = "pulse.metrics.collected"
	TopicAlertTriggered   = "pulse.alert.triggered"
	TopicAlertResolved    = "pulse.alert.resolved"
)
