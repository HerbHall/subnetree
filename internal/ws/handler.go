package ws

import (
	"context"
	"net/http"
	"time"

	"github.com/HerbHall/subnetree/internal/auth"
	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
	"github.com/coder/websocket"
)

// Handler provides WebSocket endpoints for real-time scan updates.
type Handler struct {
	hub    *Hub
	tokens *auth.TokenService
	bus    plugin.EventBus
	logger *zap.Logger
}

// Compile-time check that Handler implements the server interface.
var _ interface {
	RegisterRoutes(mux *http.ServeMux)
} = (*Handler)(nil)

// NewHandler creates a WebSocket handler and subscribes to scan events.
func NewHandler(tokens *auth.TokenService, bus plugin.EventBus, logger *zap.Logger) *Handler {
	h := &Handler{
		hub:    NewHub(logger),
		tokens: tokens,
		bus:    bus,
		logger: logger,
	}
	h.subscribeToEvents()
	return h
}

// RegisterRoutes registers WebSocket routes on the server mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/ws/scan", h.handleScanStream)
}

// handleScanStream upgrades the connection to WebSocket and streams scan events.
func (h *Handler) handleScanStream(w http.ResponseWriter, r *http.Request) {
	// Validate JWT from query parameter (browser WS API doesn't support headers).
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token parameter", http.StatusUnauthorized)
		return
	}

	claims, err := h.tokens.ValidateAccessToken(token)
	if err != nil {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	// Accept WebSocket upgrade.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Allow any origin since we validate via JWT token.
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.logger.Error("websocket accept failed", zap.Error(err))
		return
	}

	client := &Client{
		conn:   conn,
		userID: claims.UserID,
		send:   make(chan Message, 256),
		logger: h.logger,
	}

	h.hub.Register(client)

	// Run read and write pumps. When either exits, clean up.
	ctx := r.Context()
	done := make(chan struct{})
	go func() {
		client.writePump(ctx)
		close(done)
	}()

	// readPump blocks until client disconnects.
	client.readPump(ctx)

	// Client disconnected -- stop write pump and unregister.
	h.hub.Unregister(client)
	conn.Close(websocket.StatusNormalClosure, "")
	<-done
}

// subscribeToEvents subscribes to recon scan events and forwards them to all
// connected WebSocket clients.
func (h *Handler) subscribeToEvents() {
	if h.bus == nil {
		return
	}

	h.bus.Subscribe(recon.TopicScanStarted, func(_ context.Context, event plugin.Event) {
		scan, ok := event.Payload.(*models.ScanResult)
		if !ok {
			return
		}
		h.hub.Broadcast(Message{
			Type:      MessageScanStarted,
			ScanID:    scan.ID,
			Timestamp: event.Timestamp,
			Data: ScanStartedData{
				TargetCIDR: scan.Subnet,
				Status:     scan.Status,
			},
		})
	})

	h.bus.Subscribe(recon.TopicScanProgress, func(_ context.Context, event plugin.Event) {
		progress, ok := event.Payload.(*recon.ScanProgressEvent)
		if !ok {
			return
		}
		h.hub.Broadcast(Message{
			Type:      MessageScanProgress,
			ScanID:    progress.ScanID,
			Timestamp: event.Timestamp,
			Data: ScanProgressData{
				HostsAlive: progress.HostsAlive,
				SubnetSize: progress.SubnetSize,
			},
		})
	})

	h.bus.Subscribe(recon.TopicDeviceDiscovered, func(_ context.Context, event plugin.Event) {
		devEvent, ok := event.Payload.(*recon.DeviceEvent)
		if !ok {
			return
		}
		h.hub.Broadcast(Message{
			Type:      MessageScanDeviceFound,
			ScanID:    devEvent.ScanID,
			Timestamp: event.Timestamp,
			Data: ScanDeviceFoundData{
				Device: devEvent.Device,
			},
		})
	})

	h.bus.Subscribe(recon.TopicDeviceUpdated, func(_ context.Context, event plugin.Event) {
		devEvent, ok := event.Payload.(*recon.DeviceEvent)
		if !ok {
			return
		}
		h.hub.Broadcast(Message{
			Type:      MessageScanDeviceFound,
			ScanID:    devEvent.ScanID,
			Timestamp: event.Timestamp,
			Data: ScanDeviceFoundData{
				Device: devEvent.Device,
			},
		})
	})

	h.bus.Subscribe(recon.TopicScanCompleted, func(_ context.Context, event plugin.Event) {
		scan, ok := event.Payload.(*models.ScanResult)
		if !ok {
			return
		}
		h.hub.Broadcast(Message{
			Type:      MessageScanCompleted,
			ScanID:    scan.ID,
			Timestamp: event.Timestamp,
			Data: ScanCompletedData{
				Total:   scan.Total,
				Online:  scan.Online,
				EndedAt: scan.EndedAt,
			},
		})
	})

	h.logger.Info("subscribed to recon scan events for WebSocket broadcasting")
}

// BroadcastError sends an error message to all connected clients.
func (h *Handler) BroadcastError(scanID, errMsg string) {
	h.hub.Broadcast(Message{
		Type:      MessageScanError,
		ScanID:    scanID,
		Timestamp: time.Now(),
		Data: ScanErrorData{
			Error: errMsg,
		},
	})
}
