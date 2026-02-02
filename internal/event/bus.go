// Package event provides an in-memory implementation of the plugin.EventBus interface.
package event

import (
	"context"
	"sync"

	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// Compile-time interface guard.
var _ plugin.EventBus = (*Bus)(nil)

// Bus is an in-memory event bus implementing plugin.EventBus.
// Publish is synchronous (handlers run in the caller's goroutine).
// PublishAsync dispatches handlers in separate goroutines.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]handlerEntry // topic -> handlers
	allSubs  []handlerEntry            // handlers subscribed to all topics
	nextID   uint64
	logger   *zap.Logger
}

type handlerEntry struct {
	id      uint64
	handler plugin.EventHandler
}

// NewBus creates a new in-memory event bus.
func NewBus(logger *zap.Logger) *Bus {
	return &Bus{
		handlers: make(map[string][]handlerEntry),
		logger:   logger,
	}
}

// Publish dispatches an event synchronously to all matching handlers.
func (b *Bus) Publish(ctx context.Context, event plugin.Event) error {
	b.mu.RLock()
	topicHandlers := make([]handlerEntry, len(b.handlers[event.Topic]))
	copy(topicHandlers, b.handlers[event.Topic])
	allHandlers := make([]handlerEntry, len(b.allSubs))
	copy(allHandlers, b.allSubs)
	b.mu.RUnlock()

	for _, h := range topicHandlers {
		b.safeCall(ctx, h.handler, event)
	}
	for _, h := range allHandlers {
		b.safeCall(ctx, h.handler, event)
	}
	return nil
}

// PublishAsync dispatches an event asynchronously to all matching handlers.
func (b *Bus) PublishAsync(ctx context.Context, event plugin.Event) {
	b.mu.RLock()
	topicHandlers := make([]handlerEntry, len(b.handlers[event.Topic]))
	copy(topicHandlers, b.handlers[event.Topic])
	allHandlers := make([]handlerEntry, len(b.allSubs))
	copy(allHandlers, b.allSubs)
	b.mu.RUnlock()

	for _, h := range topicHandlers {
		go b.safeCall(ctx, h.handler, event)
	}
	for _, h := range allHandlers {
		go b.safeCall(ctx, h.handler, event)
	}
}

// Subscribe registers a handler for a specific topic. Returns an unsubscribe function.
func (b *Bus) Subscribe(topic string, handler plugin.EventHandler) (unsubscribe func()) {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.handlers[topic] = append(b.handlers[topic], handlerEntry{id: id, handler: handler})
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		entries := b.handlers[topic]
		for i, e := range entries {
			if e.id == id {
				b.handlers[topic] = append(entries[:i], entries[i+1:]...)
				return
			}
		}
	}
}

// SubscribeAll registers a handler for all topics. Returns an unsubscribe function.
func (b *Bus) SubscribeAll(handler plugin.EventHandler) (unsubscribe func()) {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.allSubs = append(b.allSubs, handlerEntry{id: id, handler: handler})
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, e := range b.allSubs {
			if e.id == id {
				b.allSubs = append(b.allSubs[:i], b.allSubs[i+1:]...)
				return
			}
		}
	}
}

func (b *Bus) safeCall(ctx context.Context, handler plugin.EventHandler, event plugin.Event) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("event handler panicked",
				zap.String("topic", event.Topic),
				zap.String("source", event.Source),
				zap.Any("panic", r),
			)
		}
	}()
	handler(ctx, event)
}
