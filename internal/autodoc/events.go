package autodoc

// Event topics consumed by the AutoDoc module from other modules.
const (
	TopicDeviceDiscovered = "recon.device.discovered"
	TopicDeviceUpdated    = "recon.device.updated"
	TopicDeviceLost       = "recon.device.lost"
	TopicScanCompleted    = "recon.scan.completed"
	TopicAlertTriggered   = "pulse.alert.triggered"
	TopicAlertResolved    = "pulse.alert.resolved"
)
