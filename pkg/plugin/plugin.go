// Package plugin provides the public SDK types for NetVantage plugins.
// All NetVantage modules (built-in and third-party) implement these interfaces.
// This package is Apache 2.0 licensed, separate from the BSL 1.1 core.
package plugin

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// API version constants for plugin compatibility checking.
// The registry rejects plugins outside the supported range.
const (
	APIVersionMin     = 1 // Oldest Plugin API version this server supports
	APIVersionCurrent = 1 // Current Plugin API version
)

// Plugin defines the interface that all NetVantage modules must implement.
type Plugin interface {
	// Info returns the plugin's metadata and dependency declarations.
	Info() PluginInfo

	// Init initializes the plugin with its dependencies.
	Init(ctx context.Context, deps Dependencies) error

	// Start begins the plugin's background operations.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the plugin.
	Stop(ctx context.Context) error
}

// PluginInfo contains plugin metadata and dependency declarations.
type PluginInfo struct {
	Name         string   // Unique identifier: "recon", "pulse", "vault", etc.
	Version      string   // Semantic version string
	Description  string   // Human-readable summary
	Dependencies []string // Plugin names that must initialize first
	Required     bool     // If true, server refuses to start without this plugin
	Roles        []string // Roles this plugin fills: "discovery", "credential_store"
	APIVersion   int      // Plugin API version targeted (currently 1)
}

// Dependencies provides controlled access to shared services.
// Injected by the registry during Init.
type Dependencies struct {
	Config  Config      // Scoped to this plugin's config section
	Logger  *zap.Logger // Named logger for this plugin
	Bus     EventBus    // Event publish/subscribe for inter-plugin communication
	Plugins PluginResolver
}

// Route represents an HTTP route exposed by a plugin.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// HealthStatus represents a plugin's health report.
type HealthStatus struct {
	Status  string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// Config abstracts configuration access. Wraps Viper today, replaceable later.
type Config interface {
	Unmarshal(target any) error
	Get(key string) any
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetDuration(key string) time.Duration
	IsSet(key string) bool
	Sub(key string) Config
}

// Publisher sends events to the bus. Use this thin interface in code
// that only needs to emit events (follows io.Writer pattern).
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// Subscriber receives events from the bus. Use this thin interface in
// code that only needs to listen for events (follows io.Reader pattern).
type Subscriber interface {
	Subscribe(topic string, handler EventHandler) (unsubscribe func())
}

// EventBus provides typed publish/subscribe for inter-plugin communication.
// Composes Publisher and Subscriber with async and wildcard extensions.
type EventBus interface {
	Publisher
	Subscriber
	PublishAsync(ctx context.Context, event Event)
	SubscribeAll(handler EventHandler) (unsubscribe func())
}

// Event represents a typed message on the event bus.
type Event struct {
	Topic     string
	Source    string // Plugin name that emitted the event
	Timestamp time.Time
	Payload   any // Type depends on topic
}

// EventHandler processes events from the bus.
type EventHandler func(ctx context.Context, event Event)

// Subscription declares a topic subscription for EventSubscriber plugins.
type Subscription struct {
	Topic   string
	Handler EventHandler
}

// PluginResolver allows plugins to locate other plugins by name or role.
type PluginResolver interface {
	Resolve(name string) (Plugin, bool)
	ResolveByRole(role string) []Plugin
}
