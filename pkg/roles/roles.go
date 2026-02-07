// Package roles defines typed contracts for plugin roles.
// Plugins that fill a role (declared via PluginInfo.Roles) should implement
// the corresponding interface so callers can use type-safe access via
// PluginResolver.ResolveByRole followed by a type assertion.
//
// This package is Apache 2.0 licensed, part of the public plugin SDK.
package roles

import (
	"context"

	"github.com/HerbHall/subnetree/pkg/analytics"
	"github.com/HerbHall/subnetree/pkg/llm"
	"github.com/HerbHall/subnetree/pkg/models"
)

// Role name constants match the strings used in PluginInfo.Roles.
const (
	RoleDiscovery       = "discovery"
	RoleMonitoring      = "monitoring"
	RoleCredentialStore = "credential_store" //nolint:gosec // G101: role name, not a credential
	RoleAgentManagement = "agent_management"
	RoleNotification    = "notification"
	RoleRemoteAccess    = "remote_access"
	RoleLLM             = "llm"
	RoleAnalytics       = "analytics"
)

// DiscoveryProvider is implemented by plugins that discover network devices.
type DiscoveryProvider interface {
	// Devices returns all discovered devices.
	Devices(ctx context.Context) ([]models.Device, error)

	// DeviceByID returns a single device by its ID.
	DeviceByID(ctx context.Context, id string) (*models.Device, error)
}

// MonitoringProvider is implemented by plugins that monitor device health.
type MonitoringProvider interface {
	// Status returns the current monitoring status for a device.
	Status(ctx context.Context, deviceID string) (*MonitorStatus, error)
}

// CredentialProvider is implemented by plugins that store and retrieve
// credentials for managed devices.
type CredentialProvider interface {
	// Credential retrieves a credential by ID.
	Credential(ctx context.Context, id string) (*Credential, error)

	// CredentialsForDevice returns all credentials associated with a device.
	CredentialsForDevice(ctx context.Context, deviceID string) ([]Credential, error)
}

// AgentManager is implemented by plugins that manage Scout agents.
type AgentManager interface {
	// Agents returns all registered agents.
	Agents(ctx context.Context) ([]models.AgentInfo, error)

	// AgentByID returns a single agent by its ID.
	AgentByID(ctx context.Context, id string) (*models.AgentInfo, error)
}

// Notifier is implemented by plugins that send notifications (webhooks,
// email, Slack, etc.).
type Notifier interface {
	// Notify sends a notification with the given payload.
	Notify(ctx context.Context, notification Notification) error
}

// RemoteAccessProvider is implemented by plugins that provide remote access
// to managed devices (SSH tunnels, Tailscale, etc.).
type RemoteAccessProvider interface {
	// Available reports whether remote access is available for a device.
	Available(ctx context.Context, deviceID string) (bool, error)
}

// LLMProvider is implemented by plugins that provide LLM capabilities.
// Resolve via PluginResolver.ResolveByRole(RoleLLM) then type-assert.
type LLMProvider interface {
	// Provider returns the underlying LLM provider interface.
	Provider() llm.Provider
}

// AnalyticsProvider is implemented by plugins that analyze metrics and devices.
// Resolve via PluginResolver.ResolveByRole(RoleAnalytics) then type-assert.
type AnalyticsProvider interface {
	// Anomalies returns detected anomalies, optionally filtered by device.
	// Pass empty deviceID to list all anomalies.
	Anomalies(ctx context.Context, deviceID string) ([]analytics.Anomaly, error)

	// Baselines returns learned baselines for a device.
	Baselines(ctx context.Context, deviceID string) ([]analytics.Baseline, error)

	// Forecasts returns capacity forecasts for a device.
	Forecasts(ctx context.Context, deviceID string) ([]analytics.Forecast, error)
}
