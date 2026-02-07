package insight

// Event topics consumed by the Insight module.
const (
	TopicMetricsCollected = "pulse.metrics.collected"
	TopicDeviceDiscovered = "recon.device.discovered"
	TopicDeviceUpdated    = "recon.device.updated"
	TopicAlertTriggered   = "pulse.alert.triggered"
	TopicAlertResolved    = "pulse.alert.resolved"
)

// Event topics published by the Insight module.
const (
	TopicAnomalyDetected = "insight.anomaly.detected"
	TopicAnomalyResolved = "insight.anomaly.resolved"
	TopicForecastWarning = "insight.forecast.warning"
	TopicBaselineStable  = "insight.baseline.stable"
)
