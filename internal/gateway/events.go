package gateway

// Event topics published by the Gateway module.
const (
	TopicSessionCreated = "gateway.session.created" //nolint:gosec // G101: event topic name, not a secret
	TopicSessionClosed  = "gateway.session.closed"  //nolint:gosec // G101: event topic name, not a secret
)
