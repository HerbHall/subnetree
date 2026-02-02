## Plugin Architecture

### Design Principles

The plugin system is not a bolt-on feature -- it **is** the architecture. Every major capability (discovery, monitoring, remote access, credentials, agent management) is implemented as a plugin. The core server is intentionally minimal: HTTP server, plugin registry, event bus, database, and configuration. Everything else is a plugin.

This design serves two goals: **user customizability** (swap, disable, or extend any module without rebuilding) and **ecosystem growth** (third-party developers can build plugins using the Apache 2.0-licensed SDK with zero friction).

The system follows the **Caddy/Grafana model**: a minimal core interface with optional interfaces detected via Go type assertions. Plugins declare their roles, dependencies, and capabilities in a manifest. The registry resolves dependencies via topological sort and provides a service locator for inter-plugin communication.

### Core Plugin Interface

```go
// pkg/plugin/plugin.go

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

type PluginInfo struct {
    Name         string         // Unique identifier: "recon", "pulse", "vault", etc.
    Version      string         // Semantic version string
    Description  string         // Human-readable summary
    Dependencies []string       // Plugin names that must initialize first
    Required     bool           // If true, server refuses to start without this plugin
    Roles        []string       // Roles this plugin fills: "discovery", "credential_store"
    APIVersion   int            // Plugin API version targeted (currently 1)
    Prerequisites Prerequisites // Resource and capability requirements for this plugin
}

type Prerequisites struct {
    MinRAMBytes   uint64   // Minimum additional RAM this module needs (0 = no requirement)
    MinDiskBytes  uint64   // Minimum additional disk space needed (0 = no requirement)
    RequiredPorts []int    // Ports this module needs to bind
    ExternalDeps  []string // External services needed (e.g., "guacamole" for Gateway)
    Capabilities  []string // OS capabilities needed (e.g., "CAP_NET_RAW" for ARP scanning)
}
```

### Dependencies Struct

Replaces raw Viper injection, decoupling plugins from infrastructure:

```go
type Dependencies struct {
    Config   Config         // Scoped to this plugin's config section
    Logger   *zap.Logger    // Named logger for this plugin
    Store    Store          // Database access with per-plugin migrations
    Bus      EventBus       // Event publish/subscribe for inter-plugin communication
    Plugins  PluginResolver // Resolve other plugins by name or service interface
}
```

### Config Abstraction

```go
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
```

Wraps Viper today. Replaceable without touching any plugin code.

### Optional Interfaces

Plugins implement only what they need. The registry and server detect capabilities via type assertions.

```go
// HTTPProvider -- plugins with REST API routes
type HTTPProvider interface {
    Routes() []Route
}

// GRPCProvider -- plugins with gRPC services
type GRPCProvider interface {
    RegisterGRPC(registrar grpc.ServiceRegistrar)
}

// HealthChecker -- plugins that report their health
type HealthChecker interface {
    Health(ctx context.Context) HealthStatus
}

// EventSubscriber -- plugins that declare event subscriptions at init
type EventSubscriber interface {
    Subscriptions() []Subscription
}

// Validator -- plugins that validate their config post-init
type Validator interface {
    ValidateConfig() error
}

// Reloadable -- plugins that support config hot-reload
type Reloadable interface {
    Reload(ctx context.Context, config Config) error
}

// AnalyticsProvider -- plugins that analyze metrics, events, or device data
type AnalyticsProvider interface {
    // AnalyzeMetrics processes a batch of metrics and returns insights (anomalies, trends, forecasts).
    AnalyzeMetrics(ctx context.Context, batch MetricsBatch) ([]Insight, error)

    // ClassifyDevice returns a confidence-scored device classification from available metadata.
    ClassifyDevice(ctx context.Context, device *models.Device) (*Classification, error)

    // CorrelateAlerts groups related alerts and identifies likely root causes.
    CorrelateAlerts(ctx context.Context, alerts []Alert) ([]AlertGroup, error)
}
```

### Plugin Prerequisite Verification

Before a module is initialized, the registry checks its `Prerequisites` to verify the host environment can support it. This ensures users get clear, actionable feedback instead of opaque runtime failures.

**Check flow:**
1. User enables a module (via config, CLI flag, or web UI toggle)
2. Registry reads the plugin's `Prerequisites` from `PluginInfo`
3. Each requirement is checked against the current environment
4. **Hard failures** (missing external dependency, insufficient resources) prevent the plugin from starting with a clear error message
5. **Soft warnings** (missing optional capability) allow startup with degraded functionality and a logged warning

**Prerequisite checks by type:**

| Check | Method | Failure Behavior |
|-------|--------|-----------------|
| `MinRAMBytes` | Query available system memory at startup | Warn if below threshold; block only if critically low (<50% of requirement) |
| `MinDiskBytes` | Check free space in data directory | Block if insufficient |
| `RequiredPorts` | Attempt bind or check for conflicts | Block if port is already in use; suggest alternative |
| `ExternalDeps` | Health-check external service (e.g., TCP connect to Guacamole on port 4822) | Block with setup instructions |
| `Capabilities` | Check OS capabilities (e.g., `CAP_NET_RAW` on Linux, admin rights on Windows) | Warn and fall back to reduced functionality |

**Example messages:**

- *Gateway:* "Gateway module requires Apache Guacamole running on port 4822. See https://netvantage.dev/docs/gateway-setup for Docker instructions."
- *Recon (ARP):* "ARP scanning requires root privileges or CAP_NET_RAW capability. Falling back to ICMP-only discovery. Run with `sudo` or add capability: `setcap cap_net_raw+ep ./netvantage`"
- *Pulse (large deployment):* "Pulse module monitoring 500+ devices recommends 512MB+ available RAM. Current available: 280MB. Consider reducing scan frequency or disabling unused discovery protocols."

### Role System

Roles define abstract capabilities that alternative implementations can fill. Role interfaces live in `pkg/roles/` (public) so external modules can import and implement them.

| Role | Cardinality | Default Provider | Replaceable? |
|------|-------------|-----------------|--------------|
| `credential_store` | Single | Vault | Yes (e.g., HashiCorp Vault adapter) |
| `discovery` | Multiple (supplementary) | Recon | Yes, can add supplementary engines |
| `monitoring` | Single | Pulse | Yes |
| `agent_management` | Single | Dispatch | Yes |
| `remote_access` | Single | Gateway | Yes |
| `overlay_network` | Multiple (supplementary) | Tailscale | Yes, can add other overlay providers (ZeroTier, Nebula, etc.) |
| `notifier` | Multiple | None (add-on) | N/A (extensible slot) |
| `data_export` | Multiple | None (add-on) | N/A (extensible slot) |
| `analytics` | Multiple (supplementary) | Insight (built-in, Phase 2) | Yes, can add LLM-based or custom analyzers |
| `device_store` | Single (core) | Server | No (always provided by server) |

### Plugin Composition Strategy

| Phase | Approach | Scope |
|-------|----------|-------|
| **Phase 1** | Compile-time composition with build tags | Core 5 modules |
| **Phase 2** | `nvbuild` tool (like Caddy's xcaddy) | Third-party module inclusion |
| **Phase 3** | HashiCorp go-plugin (gRPC process isolation) | Untrusted community plugins |

Build tags allow custom binaries without unused modules:
```bash
go build -tags "nogateway,novault" -o netvantage-monitor ./cmd/netvantage
```

### Plugin Lifecycle

1. **Register** -- Plugins are registered (compile-time in main.go)
2. **Validate** -- Registry validates dependency graph, role cardinality, cycles
3. **Init** -- Topological sort order. Each plugin receives `Dependencies`
4. **ValidateConfig** -- Post-init validation for plugins implementing `Validator`
5. **Start** -- Background operations begin, in dependency order
6. **Health Check Loop** -- Periodic health checks for plugins implementing `HealthChecker`
7. **Stop** -- Reverse dependency order, with context timeout per plugin

#### Plugin API Version Compatibility Check (Step 2: Validate)

During the Validate step, the registry checks each plugin's `APIVersion` against the server's supported range. This prevents silent incompatibilities between plugins compiled against different Plugin API versions.

**Server-side state:**

```go
const (
    PluginAPIVersionMin     = 1  // Oldest Plugin API version this server supports
    PluginAPIVersionCurrent = 1  // Current Plugin API version
)
```

**Validation rules:**

| Plugin's `APIVersion` | Server Action | Rationale |
|------------------------|---------------|-----------|
| `== PluginAPIVersionCurrent` | Accept | Exact match, full feature support |
| `>= PluginAPIVersionMin` and `< PluginAPIVersionCurrent` | Accept with warning | Backward compatible; server may expose features the plugin cannot use |
| `< PluginAPIVersionMin` | Reject with error | API contract too old; server cannot guarantee correct behavior |
| `> PluginAPIVersionCurrent` | Reject with error | Plugin expects features this server does not have |

**Error messages:**

- Reject (too old): `"plugin %q targets Plugin API v%d, but this server requires v%d or newer (current: v%d). Upgrade the plugin or use an older server."`
- Reject (too new): `"plugin %q targets Plugin API v%d, but this server only supports up to v%d. Upgrade the server to use this plugin."`
- Accept with warning: `"plugin %q targets Plugin API v%d (server current: v%d). The plugin will work but may not support newer features."`

**When does `PluginAPIVersionMin` increment?** When a new Plugin API version introduces changes that require server-side behavior the old API contract cannot express (e.g., a new required lifecycle method, a changed `Dependencies` struct layout). The old `PluginAPIVersionMin` is bumped one major release after deprecation notice. Example timeline:

1. **v1.2.0:** Plugin API v2 introduced. v1 still supported. `Min=1, Current=2`
2. **v1.3.0:** Plugin API v1 deprecated with log warning at startup.
3. **v2.0.0:** Plugin API v1 dropped. `Min=2, Current=2`

**Third-party plugin targeting:** Third-party plugins declare a single `APIVersion` integer in their `PluginInfo`. They should target the lowest API version that provides the features they need, to maximize compatibility across server versions.

### Go Interface Conventions for Plugins

The plugin system follows idiomatic Go patterns documented in [02-architecture-overview.md](02-architecture-overview.md#go-architecture-conventions). Key rules specific to plugins:

**Compile-time interface guards:** Every module file must include guards at the top:

```go
var (
    _ plugin.Plugin       = (*Module)(nil)
    _ plugin.HTTPProvider = (*Module)(nil)  // only if Routes() is implemented
)
```

**Contract test suite:** Every `plugin.Plugin` implementation must call the shared contract test in its `_test.go`:

```go
func TestContract(t *testing.T) {
    plugintest.TestPluginContract(t, func() plugin.Plugin { return mymodule.New() })
}
```

The contract test suite lives in `pkg/plugin/plugintest/` and verifies: valid metadata, successful init, start-after-init, stop-without-start safety, and info idempotency.

**Return concrete types:** Module constructors return `*Module`, not `plugin.Plugin`. Callers assign to the interface where needed:

```go
func New() *Module { return &Module{} }  // returns concrete
```

### Registry Features

- Topological sort of startup order from dependency declarations
- Graceful degradation: optional plugins that fail to init are disabled, not fatal
- Cascade disable: if a plugin fails, its dependents are also disabled
- Runtime enable/disable via API (with dependency checking)
- Config hot-reload via Viper's fsnotify watcher
