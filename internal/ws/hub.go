package ws

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Client represents a connected WebSocket client.
type Client struct {
	conn   *websocket.Conn
	userID string
	send   chan Message
	logger *zap.Logger
}

// Hub manages active WebSocket connections and broadcasts messages.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]struct{}
	logger     *zap.Logger
}

// NewHub creates a new WebSocket hub.
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		logger:  logger,
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	h.logger.Debug("websocket client connected", zap.String("user_id", c.userID))
}

// Unregister removes a client from the hub and closes its send channel.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
	h.logger.Debug("websocket client disconnected", zap.String("user_id", c.userID))
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
			h.logger.Warn("client send buffer full, dropping message",
				zap.String("user_id", c.userID))
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// writePump sends messages from the client's send channel to the WebSocket.
func (c *Client) writePump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				// Channel closed by hub (unregister).
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if err := wsjson.Write(writeCtx, c.conn, msg); err != nil {
				cancel()
				c.logger.Debug("websocket write error", zap.Error(err))
				return
			}
			cancel()
		}
	}
}

// readPump reads from the WebSocket to detect client disconnect.
// We don't expect client-to-server messages, so we just drain.
func (c *Client) readPump(ctx context.Context) {
	for {
		_, _, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
	}
}
