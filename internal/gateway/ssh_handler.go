package gateway

import (
	"net/http"

	"go.uber.org/zap"
)

// SSHWebSocketHandler registers Gateway SSH WebSocket routes.
// It implements server.SimpleRouteRegistrar.
type SSHWebSocketHandler struct {
	bridge *SSHBridge
}

// Compile-time interface guard.
var _ interface {
	RegisterRoutes(mux *http.ServeMux)
} = (*SSHWebSocketHandler)(nil)

// NewSSHWebSocketHandler creates a new SSHWebSocketHandler.
func NewSSHWebSocketHandler(module *Module, tokens TokenValidator, logger *zap.Logger) *SSHWebSocketHandler {
	return &SSHWebSocketHandler{
		bridge: &SSHBridge{
			module: module,
			tokens: tokens,
			logger: logger,
		},
	}
}

// RegisterRoutes registers the SSH WebSocket route on the server mux.
func (h *SSHWebSocketHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/ws/gateway/ssh/{device_id}", h.bridge.HandleSSH)
}
