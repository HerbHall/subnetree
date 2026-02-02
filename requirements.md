# NetVantage Requirements

## Product Vision

NetVantage is a modular, source-available network monitoring and management platform that provides unified device discovery, monitoring, remote access, credential management, and IoT awareness in a single self-hosted application.

**Strategic Intent:** Free for personal and home use forever. Built to become a commercial product for business, MSP, and enterprise use. The codebase, documentation, community, and clean IP chain are the product -- designed from day one for acquisition readiness.

**Target Users:** Home lab enthusiasts, prosumers, small business IT administrators, managed service providers (MSPs).

**Core Value Proposition:** No existing source-available tool combines device discovery, monitoring, remote access, credential management, and IoT awareness in a single product. Free for personal, home-lab, and non-competing production use. BSL 1.1 licensed core with Apache 2.0 plugin SDK for ecosystem growth.

### Design Philosophy

1. **Ease of use first.** You should not need a tech degree to operate NetVantage. The interface should be intuitive enough that a non-technical small business owner can understand their network health at a glance, while an experienced sysadmin can drill into the detail they need.

2. **Sensible defaults, deep customization.** NetVantage ships fully preconfigured for rapid deployment -- install and go. But the true power lies in the ability to configure and customize every aspect of the system: dashboards, alerts, scan schedules, notification channels, plugins, and themes. The defaults get you running; customization makes it yours.

3. **Plugin-powered extensibility.** The plugin architecture is not an afterthought -- it is the architecture. Every major feature is a plugin. Users and third-party developers can replace, extend, or supplement any module. The system is designed to be shaped by its users, not constrained by its authors.

4. **Stability and security are non-negotiable.** These are not features that ship "later." Every release must be stable enough to trust with production infrastructure and secure enough to trust with network credentials. If a feature compromises stability or security, it does not ship.

5. **Time to First Value under 10 minutes.** Users will forgive missing features but will not forgive a bad first experience. Download, install, see your network -- in minutes, not hours.

---

## Architecture Overview

### Components

| Component | Name | Description |
|-----------|------|-------------|
| Server | **NetVantage** | Central application: HTTP API, plugin registry, data storage, web dashboard |
| Agent | **Scout** | Lightweight Go agent installed on monitored devices |
| Dashboard | *web/* | React + TypeScript SPA served by the server |

### Server Modules (Plugins)

Each module fills one or more **roles** (abstract capabilities). Alternative implementations can replace any built-in module by implementing the same role interface.

| Module | Name | Role | Purpose |
|--------|------|------|---------|
| Discovery | **Recon** | `discovery` | Network scanning, device discovery (ICMP, ARP, SNMP, mDNS, UPnP, SSDP) |
| Monitoring | **Pulse** | `monitoring` | Health checks, uptime monitoring, metrics collection, alerting |
| Agent Management | **Dispatch** | `agent_management` | Scout agent enrollment, check-in, command dispatch, status tracking |
| Credentials | **Vault** | `credential_store` | Encrypted credential storage, per-device credential assignment |
| Remote Access | **Gateway** | `remote_access` | Browser-based SSH, RDP (via Guacamole), HTTP/HTTPS reverse proxy, VNC |
| Overlay Network | **Tailscale** | `overlay_network` | Tailscale tailnet device discovery, overlay IP enrichment, subnet route awareness |

### Communication

- **Server <-> Dashboard:** REST API + WebSocket (real-time updates)
- **Server <-> Scout:** gRPC with mTLS (bidirectional streaming)
- **Server <-> Network Devices:** ICMP, ARP, SNMP v2c/v3, mDNS, UPnP/SSDP, MQTT
- **Server <-> Tailscale API:** HTTPS REST (device enumeration, subnet routes, DNS)

### Module Dependency Graph

```
Vault (no deps, provides credential_store)
  |
  +---> Recon (optional: credential_store for authenticated scanning)
  |       |
  |       +---> Pulse (requires: discovery for device list)
  |       +---> Gateway (requires: discovery + optional credential_store)
  |
Dispatch (no deps, provides agent_management)
  |
  +---> Pulse (optional: agent_management for agent metrics)
  +---> Recon (optional: agent_management for agent-assisted scans)

Tailscale (requires: credential_store for API key/OAuth storage)
  |
  +---> Recon (optional: overlay_network for Tailscale-discovered devices)
  +---> Gateway (optional: overlay_network for Tailscale IP connectivity)
```

**Topological Startup Order:** Vault -> Dispatch -> Tailscale -> Recon -> Pulse -> Gateway
**Shutdown Order (reverse):** Gateway -> Pulse -> Recon -> Tailscale -> Dispatch -> Vault

---

## Technology Stack

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Server | Go (1.25+) | Performance, single binary deployment, strong networking stdlib |
| Agent | Go | Same language as server, cross-compiles to all targets |
| Dashboard | React + TypeScript (Vite) | Largest ecosystem, rich component libraries |
| UI Components | shadcn/ui + Tailwind CSS | Customizable, not a dependency, modern styling |
| UI State | TanStack Query + Zustand | TanStack for server state, Zustand for client state |
| UI Charts | Recharts | Composable React charting library |
| Agent Communication | gRPC + Protobuf (buf) | Efficient binary protocol, bidirectional streaming, code generation |
| Real-time UI | WebSocket | Push updates to dashboard without polling |
| Configuration | Viper (YAML) | Standard Go config library, env var support |
| Logging | Zap | High-performance structured logging |
| Database (Phase 1) | SQLite via modernc.org/sqlite | Pure Go (no CGo), zero-config, cross-compilation friendly |
| Database (Phase 2+) | PostgreSQL + TimescaleDB | Time-series metrics at scale, multi-tenant support |
| HTTP Routing | net/http (stdlib) | No unnecessary dependencies for Phase 1 |
| Authentication | Local (bcrypt) + JWT | Local auth default, OIDC optional |
| Remote Desktop | Apache Guacamole (Docker) | Apache 2.0 licensed, proven RDP/VNC gateway |
| SSH Terminal | xterm.js + Go SSH library | Browser-based SSH terminal |
| HTTP Proxy | Go reverse proxy (stdlib) | Access device web interfaces through server |
| SNMP | gosnmp | Pure Go SNMP library |
| MQTT | Eclipse Paho Go | MQTT client for IoT device communication |
| Metrics Exposition | Prometheus client_golang | Industry standard metrics format |
| Tailscale API | tailscale-client-go-v2 | MIT licensed Tailscale API client for tailnet device discovery |
| Graph Operations | dominikbraun/graph | Apache 2.0 licensed generic graph library for dependency resolution, topology computation, cycle detection |
| Statistical Analysis | gonum.org/v1/gonum | BSD-3 licensed numerical computing: statistics, linear algebra, FFT for analytics engine |
| LLM Client (OpenAI) | sashabaranov/go-openai | Apache 2.0 licensed OpenAI API client with streaming, function calling, embeddings (Phase 3) |
| LLM Client (Anthropic) | liushuangls/go-anthropic | Apache 2.0 licensed Anthropic Claude API client with tool use, streaming (Phase 3) |
| Model Inference | yalue/onnxruntime_go | MIT licensed ONNX Runtime bindings for Go, supports x64 + ARM64 (Phase 4) |
| Proto Management | buf | Modern protobuf toolchain, linting, breaking change detection |

---

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

### Registry Features

- Topological sort of startup order from dependency declarations
- Graceful degradation: optional plugins that fail to init are disabled, not fatal
- Cascade disable: if a plugin fails, its dependents are also disabled
- Runtime enable/disable via API (with dependency checking)
- Config hot-reload via Viper's fsnotify watcher

---

## Event System

### Event Bus

Inter-plugin communication via typed publish/subscribe. Synchronous by default (handlers run in publisher's goroutine) with `PublishAsync` available for slow handlers.

```go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    PublishAsync(ctx context.Context, event Event)
    Subscribe(topic string, handler EventHandler) (unsubscribe func())
    SubscribeAll(handler EventHandler) (unsubscribe func())
}

type Event struct {
    Topic     string    // "{plugin}.{entity}.{action}" e.g., "recon.device.discovered"
    Source    string    // Plugin name that emitted the event
    Timestamp time.Time
    Payload   any       // Type depends on topic (documented per constant)
}
```

### Core Event Topics

| Topic | Payload Type | Emitter | Subscribers |
|-------|-------------|---------|-------------|
| `recon.device.discovered` | `*models.Device` | Recon | Pulse, Gateway, Topology |
| `recon.device.updated` | `*models.Device` | Recon | Pulse, Dashboard |
| `recon.device.lost` | `DeviceLostEvent` | Recon | Pulse, Dashboard |
| `recon.scan.started` | `*models.ScanResult` | Recon | Dashboard |
| `recon.scan.completed` | `*models.ScanResult` | Recon | Dashboard |
| `pulse.alert.triggered` | `Alert` | Pulse | Notifiers, Dashboard |
| `pulse.alert.resolved` | `Alert` | Pulse | Notifiers, Dashboard |
| `pulse.metrics.collected` | `MetricsBatch` | Pulse | Data Exporters, Analytics |
| `dispatch.agent.connected` | `*models.AgentInfo` | Dispatch | Dashboard |
| `dispatch.agent.disconnected` | `*models.AgentInfo` | Dispatch | Dashboard |
| `dispatch.agent.enrolled` | `*models.AgentInfo` | Dispatch | Recon, Dashboard |
| `vault.credential.created` | `CredentialEvent` | Vault | Audit Log |
| `vault.credential.accessed` | `CredentialEvent` | Vault | Audit Log |
| `system.plugin.unhealthy` | `PluginHealthEvent` | Registry | Dashboard, Notifiers |

---

## Database Layer

### Architecture

Shared connection pool with per-plugin schema ownership. Each plugin owns its own tables (prefixed with plugin name) but shares a single database connection.

### Store Interface

```go
type Store interface {
    DB() *sql.DB
    Tx(ctx context.Context, fn func(tx *sql.Tx) error) error
    Migrate(ctx context.Context, pluginName string, migrations []Migration) error
}

type Migration struct {
    Version     int
    Description string
    Up          func(tx *sql.Tx) error
}
```

### SQLite Configuration (Phase 1)

Driver: `modernc.org/sqlite` (pure Go, no CGo dependency)

Connection pragmas:
- `_journal_mode=WAL` -- Concurrent reads during writes
- `_busy_timeout=5000` -- Wait up to 5s for locks instead of failing immediately
- `_synchronous=NORMAL` -- Safe with WAL mode, better write performance
- `_foreign_keys=ON` -- Enforce referential integrity
- `_cache_size=-20000` -- 20MB page cache

`MaxOpenConns(1)` -- SQLite performs best with a single write connection. WAL enables concurrent readers.

### Migration Tracking

A shared `_migrations` table tracks applied migrations per plugin:

```sql
CREATE TABLE _migrations (
    plugin_name TEXT NOT NULL,
    version     INTEGER NOT NULL,
    description TEXT NOT NULL,
    applied_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (plugin_name, version)
);
```

### Repository Pattern

- **Shared interfaces** in `internal/services/` -- `DeviceRepository`, `CredentialProvider`, `AgentManager`
- **Private implementations** in each plugin package -- SQLite-specific query code
- **No ORM** -- Raw SQL with thin repository layer. Queries are straightforward CRUD, and raw SQL provides performance transparency and debugging clarity.

### PostgreSQL Migration Path (Phase 2+)

- Repository interfaces remain the same; only implementations change
- TimescaleDB hypertables for time-series metrics (Pulse module)
- Continuous aggregates for dashboard rollup queries
- Retention policies for automatic data lifecycle
- Connection pooling via pgxpool

---

## Authentication and Authorization

### Phase 1: Local Authentication

- User accounts stored in SQLite with bcrypt-hashed passwords
- JWT access tokens (short-lived, 15 minutes)
- JWT refresh tokens (long-lived, 7 days, stored server-side, rotated on use)
- First-run setup wizard creates the initial admin account
- API key support for automation/scripting

### Phase 1 (Optional): OIDC/OAuth2

- Optional external identity provider support (Google, Keycloak, Authentik, Azure AD)
- Configured via YAML; disabled by default
- Auto-create local user on first OIDC login
- Map OIDC claims to NetVantage roles

### Data Model: User

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| Username | string | Login identifier |
| Email | string | Email address |
| PasswordHash | string | bcrypt hash (null for OIDC-only users) |
| Role | enum | admin, operator, viewer |
| AuthProvider | enum | local, oidc |
| OIDCSubject | string? | OIDC subject identifier |
| CreatedAt | timestamp | Account creation |
| LastLogin | timestamp | Last successful authentication |
| Disabled | bool | Account disabled flag |

### Authorization Model (Phase 1)

Three roles with fixed permissions:

| Role | Permissions |
|------|------------|
| **admin** | Full access: user management, plugin management, all CRUD |
| **operator** | Device management, scan triggers, credential use, remote sessions |
| **viewer** | Read-only access to dashboards, device list, monitoring status |

### Phase 2: RBAC

- Custom roles with granular permissions
- Per-tenant role assignments for MSP multi-tenancy
- Permission inheritance

---

## Scout Agent Specification

### Purpose

Lightweight agent installed on monitored devices to report system metrics, accept commands, and facilitate remote access.

### Capabilities

- System metrics: CPU, memory, disk, network usage
- Process listing
- Service status monitoring
- Log forwarding (opt-in)
- Command execution (authorized commands only)
- Auto-update (see Auto-Update Mechanism below)

### Communication

- gRPC with mTLS to server
- Periodic check-in (configurable interval, default 30s)
- Bidirectional streaming for real-time commands
- Exponential backoff reconnection (1s, 2s, 4s, 8s... max 5 minutes)

### Certificate Management

- Server runs an internal CA for mTLS
- Agent enrollment: token-based + certificate signing request
- Per-agent certificates with 90-day validity
- Auto-renewal at day 60
- Certificate revocation list for decommissioned agents

### Resource Constraints

The Scout agent must run unobtrusively on the host system, including resource-constrained devices:

| Resource | Target | Minimum Platform |
|----------|--------|-----------------|
| Binary size | < 15 MB (statically linked) | All |
| Memory (idle) | < 10 MB | Raspberry Pi 3B+ (1 GB RAM) |
| Memory (active collection) | < 20 MB | Raspberry Pi 3B+ (1 GB RAM) |
| CPU (idle) | < 1% | All |
| CPU (active collection) | < 5% | Raspberry Pi 3B+ (quad-core ARM) |
| Disk (binary + data + logs) | < 50 MB | All |
| Network (check-in) | < 1 KB per check-in | All |

### Platforms

| Platform | Priority | Architecture | Method |
|----------|----------|-------------|--------|
| Windows x64 | Phase 1b | x86_64 | Native Go binary, Windows service |
| Linux x64 | Phase 2 | x86_64 | Native Go binary, systemd unit |
| Linux ARM64 | Phase 2 | ARM64 (aarch64) | Cross-compiled Go binary, systemd unit |
| Linux ARM | Phase 2 | ARMv7 (armhf) | Cross-compiled Go binary (Raspberry Pi 3B+, older SBCs) |
| macOS ARM64 | Phase 3 | ARM64 (Apple Silicon) | Native Go binary, launchd plist |
| macOS x64 | Phase 3 | x86_64 | Native Go binary, launchd plist |
| Android | Deferred | ARM64 | Passive monitoring only (ping, ARP, mDNS) |
| IoT/Embedded | Phase 4 | Various | Lightweight Go binary or MQTT-based |

### Auto-Update Mechanism (Phase 2)

Agent auto-update is a security-critical feature. The SolarWinds supply chain attack demonstrated the risk of compromised update channels.

#### Update Flow

1. Agent polls server for available updates during check-in (configurable: enabled/disabled, channel)
2. Server responds with version info + signed manifest if update available
3. Agent downloads binary from server, verifies Cosign signature against pinned public key
4. Agent validates binary integrity (SHA-256 checksum from signed manifest)
5. Agent installs update (platform-specific: replace binary, restart service)
6. Agent reports new version on next check-in; server marks update as successful
7. If agent fails to check in within expected window after update, server marks update as failed

#### Controls

- **Administrator approval:** Updates require explicit approval per version in the server UI before any agent receives them
- **Staged rollout:** Configurable: update N% of agents, wait for health confirmation, then proceed (default: 10% canary, 24h wait)
- **Version pinning:** Administrators can pin individual agents or agent groups to a specific version
- **Update channels:** `stable` (default), `beta`, `pinned` (manual only)
- **Rollback:** Agent retains previous binary. Automatic rollback if health check fails within 5 minutes of update
- **Air-gapped support:** Manual update package (signed binary + manifest) for offline environments

### Security

- Agent authenticates to server via enrollment token + mTLS certificate
- Server issues per-agent certificates during enrollment
- Commands require server-side authorization
- Agent binary is source-available (BSL 1.1) for user trust and auditability
- Per-agent rate limiting in gRPC interceptor
- Update binaries signed with Cosign; agent verifies before applying

### Agent-Server Version Compatibility

The server and agent each carry a SemVer version string (e.g., `1.3.2`). Compatibility is determined by the **gRPC protocol version** (`proto_version` integer in `CheckInRequest`), not by comparing version strings directly. This decouples release cadence from protocol compatibility.

#### Compatibility Table

| Agent Proto Version | Server Proto Version | Result |
|---------------------|---------------------|--------|
| Same | Same | Full compatibility |
| Older (N-1) | Current (N) | Supported -- server handles old message format |
| Older (< N-1) | Current (N) | Rejected -- agent must update |
| Newer than server | Any | Rejected -- agent must not be newer than server |

**Rule:** Always upgrade the server first, then agents. The server supports the current proto version and one version behind (N and N-1). Agents more than one proto version behind are rejected with an explicit upgrade instruction.

#### Version Negotiation Protocol

The `CheckInRequest` message carries version metadata:

```protobuf
message CheckInRequest {
  string agent_id = 1;
  string hostname = 2;
  string platform = 3;
  string agent_version = 4;   // SemVer string, e.g., "1.3.2"
  SystemMetrics metrics = 5;
  uint32 proto_version = 6;   // gRPC protocol version integer
}

message CheckInResponse {
  bool acknowledged = 1;
  int32 check_interval_seconds = 2;
  repeated string pending_commands = 3;
  VersionStatus version_status = 4;     // Compatibility result
  string server_version = 5;            // Server SemVer for diagnostics
  string upgrade_message = 6;           // Human-readable instruction (if rejected/deprecated)
}

enum VersionStatus {
  VERSION_OK = 0;              // Fully compatible
  VERSION_DEPRECATED = 1;     // Works now, will stop working in next server major
  VERSION_REJECTED = 2;       // Incompatible, check-in rejected
  VERSION_UPDATE_AVAILABLE = 3; // Compatible, but newer agent version exists
}
```

**Server behavior on check-in:**

1. Parse `proto_version` from `CheckInRequest`.
2. If `proto_version > server_proto_version`: respond with `VERSION_REJECTED`, message: "Agent proto version %d is newer than server proto version %d. Downgrade the agent or upgrade the server."
3. If `proto_version < server_proto_version - 1`: respond with `VERSION_REJECTED`, message: "Agent proto version %d is too old. Minimum supported: %d. Update the agent to continue."
4. If `proto_version == server_proto_version - 1`: respond with `VERSION_DEPRECATED`, process check-in normally, message: "Agent proto version %d is deprecated. Update before the next server major release."
5. If `proto_version == server_proto_version`: respond with `VERSION_OK`, process check-in normally.
6. In all cases, check if a newer agent binary is available and set `VERSION_UPDATE_AVAILABLE` when applicable (only when the agent is otherwise compatible).

**Rejected agents:** When `VERSION_REJECTED`, the server logs the event, does NOT process metrics or commands, and returns the response with `acknowledged = false`. The agent should log the `upgrade_message` and continue retrying at a reduced interval (5 minutes) in case the server is upgraded.

---

## Tailscale Integration (Plugin)

### Overview

The Tailscale plugin provides automatic device discovery and overlay network connectivity for users running Tailscale. It queries the Tailscale API to enumerate devices on the user's tailnet, enriching NetVantage's device inventory with Tailscale metadata (IP addresses, hostnames, tags, OS, online status). For distributed home labs and multi-site networks, Tailscale eliminates NAT traversal complexity -- devices are reachable by their stable 100.x.y.z addresses regardless of physical location.

**Licensing:** The Tailscale API client library (`tailscale-client-go-v2`) is MIT licensed. No copyleft or licensing conflicts with BSL 1.1.

**Base distribution candidate:** This plugin adds minimal binary size and zero runtime cost when disabled. Flagged for inclusion in the default build if testing confirms no impact on startup time or first-run experience for users without Tailscale.

### Capabilities

| Capability | Description | Phase |
|-----------|-------------|-------|
| Device discovery | Enumerate tailnet devices via Tailscale API (hostname, IPs, OS, tags, last seen, online status) | 2 |
| Tailscale IP enrichment | Add Tailscale IPs (100.x.y.z) to existing device records, enabling monitoring across NAT | 2 |
| Subnet route awareness | Detect Tailscale subnet routers and offer to scan advertised subnets | 2 |
| MagicDNS hostname resolution | Use Tailscale DNS names (e.g., `device.tailnet.ts.net`) for device identification | 2 |
| Scout over Tailscale | Support Scout agent communication via Tailscale IPs (no port forwarding required) | 2 |
| Connectivity preference | Prefer Tailscale IPs for monitoring/remote access when devices are on the tailnet | 3 |
| Tailscale Funnel/Serve guidance | Documentation for exposing NetVantage dashboard via Tailscale Funnel | 1 (docs only) |

### Authentication

| Method | Use Case | Storage |
|--------|----------|---------|
| API key | Personal tailnets, simple setup | Vault-encrypted |
| OAuth client | Organizational tailnets, token refresh | Vault-encrypted (client ID + secret) |

API credentials are stored in the Vault module (encrypted at rest). The plugin never stores credentials outside Vault.

### Device Merging

Tailscale-discovered devices are merged with existing NetVantage device records using a match priority:

1. **MAC address** -- exact match (most reliable)
2. **Hostname** -- case-insensitive match
3. **IP overlap** -- any shared IP between Tailscale device and existing record

If no match is found, a new device record is created with `discovery_method: tailscale`. Merged devices retain their original discovery method and gain Tailscale metadata as supplemental data.

### Dependencies

- **Required:** Vault (for API key / OAuth credential storage)
- **Optional:** Recon (merges Tailscale-discovered devices with scan results)
- **Optional:** Gateway (uses Tailscale IPs for remote access)

### Implementation Notes

- Uses `tailscale-client-go-v2` (MIT) -- lightweight client, no heavy dependency tree
- Respects Tailscale API rate limits (`Retry-After` headers); default sync interval of 5 minutes
- Graceful degradation: if Tailscale API is unreachable, the plugin logs a warning and retries on next sync interval; does not block other discovery methods
- Dashboard shows a "Tailscale" badge on devices discovered or enriched via the tailnet

---

## Data Model (Core Entities)

### Device

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| TenantID | UUID? | Tenant (null for single-tenant, populated in MSP mode) |
| Hostname | string | Device hostname |
| IPAddresses | []string | All known IP addresses |
| MACAddress | string | Primary MAC address |
| Manufacturer | string | Derived from OUI database |
| DeviceType | enum | server, desktop, laptop, mobile, router, switch, printer, ap, firewall, iot, camera, nas, unknown |
| OS | string | Operating system (if known) |
| Status | enum | online, offline, degraded, unknown |
| DiscoveryMethod | enum | agent, icmp, arp, snmp, mdns, upnp, mqtt, tailscale, manual |
| AgentID | UUID? | Linked Scout agent (if any) |
| ParentDeviceID | UUID? | Upstream device for topology (switch port, router) |
| LastSeen | timestamp | Last successful contact |
| FirstSeen | timestamp | Initial discovery |
| Notes | string | User-provided notes |
| Tags | []string | User-defined tags |
| CustomFields | map | User-defined key-value pairs |

### Agent (Scout)

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| TenantID | UUID? | Tenant |
| DeviceID | UUID | Linked device |
| Version | string | Agent software version |
| Status | enum | connected, disconnected, stale |
| LastCheckIn | timestamp | Last successful check-in |
| EnrolledAt | timestamp | Enrollment timestamp |
| CertSerialNo | string | mTLS certificate serial number |
| CertExpiresAt | timestamp | Certificate expiration |
| Platform | string | OS/architecture |

### Credential (Vault)

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| TenantID | UUID? | Tenant |
| Name | string | Display name |
| Type | enum | ssh_password, ssh_key, rdp, http_basic, snmp_community, snmp_v3, api_key |
| Data | encrypted blob | Encrypted credential data (AES-256-GCM envelope encryption) |
| DeviceIDs | []UUID | Associated devices |
| CreatedBy | UUID | User who created |
| CreatedAt | timestamp | Creation timestamp |
| UpdatedAt | timestamp | Last modification |
| LastAccessedAt | timestamp | Last time credential was used |

### Topology Link

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| SourceDeviceID | UUID | Upstream device |
| TargetDeviceID | UUID | Downstream device |
| SourcePort | string | Port/interface name on source |
| TargetPort | string | Port/interface name on target |
| LinkType | enum | lldp, cdp, arp, manual |
| Speed | int | Link speed in Mbps |
| DiscoveredAt | timestamp | When this link was detected |
| LastConfirmed | timestamp | Last time this link was confirmed active |

### Tenant (Phase 2)

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| Name | string | Tenant/client name |
| Slug | string | URL-safe identifier |
| Status | enum | active, suspended, archived |
| MaxDevices | int | Device limit for this tenant |
| CreatedAt | timestamp | Tenant creation |

---

## API Design

### Standards

- **Error responses:** RFC 7807 Problem Details (`application/problem+json`)
- **Pagination:** Cursor-based with `PaginatedResponse<T>` wrapper
- **Versioning:** URL path versioning (`/api/v1/`). Version is the first path segment after `/api/`. The versioning policy is:

  **REST API Versioning Rules:**
  - Only the **major** component appears in the URL path (`/api/v1/`, `/api/v2/`).
  - Minor and patch releases add fields, endpoints, or fix bugs without changing the path.
  - **Additive changes** (new fields in responses, new optional query parameters, new endpoints) are NOT breaking and do NOT require a new API version.
  - **Breaking changes** (removing/renaming fields, changing response structure, removing endpoints, changing authentication) require a new API version.
  - **Deprecation timeline:** When `/api/v2/` is introduced, `/api/v1/` continues to work for a minimum of **6 months** (two minor release cycles). Deprecated API versions return a `Sunset` header (RFC 8594) and a `Deprecation` header on every response.
  - **Version response header:** All API responses include `X-NetVantage-Version: {server_version}` (e.g., `X-NetVantage-Version: 1.3.2`). This enables clients to detect server version without a dedicated endpoint.
  - **Maximum concurrent API versions:** 2 (current + one prior). No more than two URL path versions served simultaneously.
  - **Health and metrics endpoints** (`/healthz`, `/readyz`, `/metrics`) are unversioned -- they are not part of the API contract.

- **Rate limiting:** Per-IP using `golang.org/x/time/rate`; per-tenant rate limiting in Phase 2
- **Documentation:** OpenAPI 3.0 via `swaggo/swag` annotations
- **Request tracing:** `X-Request-ID` header (generated if not provided)
- **Idempotency:** `Idempotency-Key` header supported on POST endpoints (device creation, credential storage) for safe retries. Server stores key-to-response mapping for 24 hours.
- **Conditional requests:** `ETag` + `If-None-Match` on GET endpoints for client-side cache validation. Reduces bandwidth for polling clients.

### Error Response Format

```json
{
  "type": "https://netvantage.io/problems/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "Device with ID 'abc-123' does not exist",
  "instance": "/api/v1/devices/abc-123"
}
```

### Pagination Format

```json
{
  "data": [...],
  "pagination": {
    "total": 142,
    "limit": 50,
    "next_cursor": "base64encoded",
    "has_more": true
  }
}
```

### REST API

Base path: `/api/v1/`

#### Core Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/healthz` | GET | Liveness probe (always 200 if process is alive) |
| `/readyz` | GET | Readiness probe (checks DB, plugin health) |
| `/metrics` | GET | Prometheus metrics |
| `/api/v1/health` | GET | Readiness (alias for backward compat) |
| `/api/v1/plugins` | GET | List loaded plugins with status |
| `/api/v1/plugins/{name}/enable` | POST | Enable a plugin at runtime |
| `/api/v1/plugins/{name}/disable` | POST | Disable a plugin at runtime |

#### Auth Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/login` | POST | Authenticate, returns JWT pair |
| `/api/v1/auth/refresh` | POST | Refresh access token |
| `/api/v1/auth/logout` | POST | Revoke refresh token |
| `/api/v1/auth/setup` | POST | First-run: create admin account |
| `/api/v1/auth/oidc/callback` | GET | OIDC callback handler |
| `/api/v1/users` | GET | List users (admin only) |
| `/api/v1/users/{id}` | GET/PUT/DELETE | User management (admin only) |

#### Device Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/devices` | GET | List devices (paginated, filterable) |
| `/api/v1/devices/{id}` | GET | Device details with related data |
| `/api/v1/devices` | POST | Create device manually |
| `/api/v1/devices/{id}` | PUT | Update device |
| `/api/v1/devices/{id}` | DELETE | Remove device |
| `/api/v1/devices/{id}/topology` | GET | Device's topology connections |

#### Plugin Endpoints (mounted under `/api/v1/{plugin-name}/`)

| Endpoint | Method | Plugin | Description |
|----------|--------|--------|-------------|
| `/recon/scan` | POST | Recon | Trigger network scan |
| `/recon/scans` | GET | Recon | List scan history |
| `/recon/topology` | GET | Recon | Full topology graph |
| `/pulse/status` | GET | Pulse | Overall monitoring status |
| `/pulse/alerts` | GET | Pulse | List active/recent alerts |
| `/pulse/alerts/{id}/ack` | POST | Pulse | Acknowledge an alert |
| `/pulse/metrics/{device_id}` | GET | Pulse | Device metrics with time range |
| `/dispatch/agents` | GET | Dispatch | List connected agents |
| `/dispatch/agents/{id}` | GET | Dispatch | Agent details |
| `/dispatch/enroll` | POST | Dispatch | Generate enrollment token |
| `/vault/credentials` | GET | Vault | List credentials (metadata only) |
| `/vault/credentials` | POST | Vault | Store new credential |
| `/vault/credentials/{id}` | GET | Vault | Credential metadata |
| `/vault/credentials/{id}` | DELETE | Vault | Delete credential |
| `/gateway/sessions` | GET | Gateway | List active remote sessions |
| `/gateway/ssh/{device_id}` | WebSocket | Gateway | SSH terminal session |
| `/gateway/rdp/{device_id}` | WebSocket | Gateway | RDP session (via Guacamole) |
| `/gateway/proxy/{device_id}` | ANY | Gateway | HTTP reverse proxy to device |

### WebSocket Connection

- **Endpoint:** `GET /ws/` (upgrades to WebSocket)
- **Authentication:** JWT token sent in the first message after connection (not in URL query params, which leak in server logs and browser history)
- **Protocol:** JSON messages with `{ "type": "...", "payload": { ... } }` envelope
- **Reconnection:** Client implements exponential backoff (1s, 2s, 4s... max 30s) with jitter
- **Heartbeat:** Server sends `ping` every 30s; client responds with `pong`. Connection closed after 3 missed pongs.

### WebSocket Events (Dashboard Real-Time)

| Event | Direction | Description |
|-------|-----------|-------------|
| `device.discovered` | Server -> Client | New device found during scan |
| `device.status_changed` | Server -> Client | Device status update |
| `scan.progress` | Server -> Client | Scan completion percentage |
| `scan.completed` | Server -> Client | Scan finished |
| `alert.triggered` | Server -> Client | New alert |
| `alert.resolved` | Server -> Client | Alert cleared |
| `agent.connected` | Server -> Client | Agent came online |
| `agent.disconnected` | Server -> Client | Agent went offline |

### gRPC Services (Agent Communication)

```protobuf
service ScoutService {
  rpc Enroll(EnrollRequest) returns (EnrollResponse);
  rpc CheckIn(CheckInRequest) returns (CheckInResponse);
  rpc ReportMetrics(stream MetricsReport) returns (Ack);
  rpc CommandStream(stream CommandResponse) returns (stream Command);
  rpc RenewCertificate(CertRenewalRequest) returns (CertRenewalResponse);
}
```

**gRPC API Versioning Policy:**

- **Proto package versioning:** Proto definitions live in `api/proto/v1/` with package `netvantage.v1`. A breaking change creates `api/proto/v2/` with package `netvantage.v2`.
- **Breaking changes in gRPC** include: removing or renaming fields, changing field numbers, changing field types, removing RPC methods, changing streaming semantics (unary to streaming or vice versa).
- **Non-breaking changes** include: adding new fields (proto3 handles unknown fields gracefully), adding new RPC methods, adding new enum values.
- **Proto version integer:** Each proto package version has a corresponding integer (`proto_version`) sent in `CheckInRequest`. This enables version negotiation without parsing proto package names at runtime (see Agent-Server Version Compatibility).
- **`buf breaking` enforcement:** The `buf` toolchain runs breaking-change detection in CI against the previous tagged release. Any breaking change fails the build unless the proto package version is incremented.
- **Backward compatibility guarantee:** The server supports the current proto version and one version behind (N and N-1). This matches the agent-server compatibility rule.
- **gRPC metadata:** The server sets `x-netvantage-version` in gRPC response metadata (trailing headers) for diagnostic purposes.

### Rate Limits

| Endpoint Pattern | Rate | Burst | Reason |
|-----------------|------|-------|--------|
| General API | 100/s | 200 | Dashboard makes parallel requests |
| `POST /recon/scan` | 1/min | 2 | Scans are expensive network operations |
| `POST /vault/credentials` | 10/s | 20 | Security-sensitive |
| `POST /auth/login` | 5/min | 10 | Brute force protection |
| `/healthz`, `/readyz`, `/metrics` | Unlimited | -- | Orchestrator/monitoring probes |

---

## Brand Identity & Design System

### Logo

The NetVantage logo is an "N" constructed from network topology elements:
- **4 primary nodes** at the letter's corners (green) -- network endpoints
- **3 midpoint nodes** (amber/sage) -- monitored devices along connections
- **2 satellite nodes** (sage) -- discovered peripheral devices
- **Connection lines** forming the N shape -- network links and topology
- **Outer pulse ring** (dashed) -- monitoring/discovery radar sweep
- **Center node with glow** -- the vantage point (the server)

Logo files: `assets/brand/logo.svg` (dark background), `assets/brand/logo-light.svg` (light background)
Favicon: `web/public/favicon.svg`

### Color Palette

Dark mode is the default. The palette uses forest greens and earth tones.

| Role | Token | Hex | Usage |
|------|-------|-----|-------|
| **Primary accent** | `green-400` | `#4ade80` | Healthy status, primary actions, links, "online" |
| **Primary dark** | `green-600` | `#16a34a` | Buttons, active states |
| **Secondary accent** | `earth-400` | `#c4a77d` | Warm highlights, degraded status, secondary elements |
| **Tertiary** | `sage-400` | `#9ca389` | Muted text, unknown status, subtle elements |
| **Background** | `forest-950` | `#0c1a0e` | Root dark background |
| **Surface** | `forest-900` | `#0f1a10` | Page background |
| **Card** | `forest-700` | `#1a2e1c` | Card/elevated surfaces |
| **Text primary** | -- | `#f5f0e8` | Warm cream white |
| **Text secondary** | `sage-400` | `#9ca389` | Subdued content |
| **Danger** | -- | `#ef4444` | Offline status, errors, destructive actions |

### Status Color Mapping

| Status | Color | Token |
|--------|-------|-------|
| Online / Healthy | Green | `status-online` (#4ade80) |
| Degraded / Warning | Amber | `status-degraded` (#c4a77d) |
| Offline / Error | Red | `status-offline` (#ef4444) |
| Unknown | Sage | `status-unknown` (#9ca389) |

### Design Token Files

- **CSS custom properties:** `web/src/styles/design-tokens.css` (includes dark + light mode)
- **Tailwind config:** `web/tailwind.config.ts` (maps palette to Tailwind classes)

### Typography

- **Sans-serif:** System font stack (-apple-system, BlinkMacSystemFont, Segoe UI, Inter)
- **Monospace:** JetBrains Mono, Fira Code, Cascadia Code (terminal output, code, IPs)

---

## Dashboard Architecture

The dashboard is the primary interface for most users. It must be approachable enough for someone with no networking background to understand "is my network healthy?" while powerful enough for an experienced administrator to customize every aspect of their monitoring experience.

### Technology

- **Framework:** React 18+ with TypeScript
- **Build Tool:** Vite
- **Components:** shadcn/ui (Tailwind-based, copy-paste components, not a npm dependency)
- **Server State:** TanStack Query (React Query) for API data, caching, and real-time invalidation
- **Client State:** Zustand for UI state (sidebar collapsed, selected filters, theme)
- **Charts:** Recharts for time-series graphs and monitoring visualizations
- **Topology:** React Flow for interactive network topology map (zoom, pan, custom nodes, auto-layout)
- **Real-time:** WebSocket connection managed by a custom hook, invalidates TanStack Query caches
- **Routing:** React Router v6+
- **Dark Mode:** First-class support from day one (Tailwind dark: variant)

### Browser Support

| Browser | Version | Support Level |
|---------|---------|---------------|
| Chrome / Edge | Last 2 major versions | Full support |
| Firefox | Last 2 major versions | Full support |
| Safari | Last 2 major versions | Full support |
| Mobile Chrome/Safari | Last 2 major versions | Responsive support (triage-focused) |
| Internet Explorer | Any | Not supported |

### Accessibility

Target: **WCAG 2.1 AA** compliance for all dashboard pages.

- Semantic HTML elements (`nav`, `main`, `article`, `table`, etc.)
- ARIA labels for interactive elements and icon-only buttons
- Full keyboard navigation (tab order, focus indicators, skip links)
- Color contrast: minimum 4.5:1 for normal text, 3:1 for large text
- Status information conveyed by more than color alone (icons + labels + color)
- Screen reader support for data tables and alert notifications
- Reduced motion support (`prefers-reduced-motion` media query)

### Error & Empty State Patterns

Defined UX patterns for non-happy-path states:

| State | Pattern | Example |
|-------|---------|---------|
| Empty (no data yet) | Illustration + explanation + CTA | "No devices discovered. Run your first scan." |
| Loading | Skeleton placeholders (not spinners) | Shimmer cards matching final layout |
| Error (API failure) | Inline error with retry button | "Failed to load devices. Retry" |
| Connection lost | Toast notification + auto-reconnect | "Connection lost. Reconnecting..." |
| Permission denied | Explanation + redirect or contact admin | "You need operator access to view credentials." |
| No results (filtered) | Clear message + clear-filters action | "No devices match your filters. Clear filters" |

### Key UX Principles

Design for the non-technical user first, then layer in power-user capabilities. A small business owner should understand their network health at a glance. A sysadmin should be able to customize everything.

1. **Wall of Green:** When everything is healthy, the dashboard is calm (forest green background, green-400 status dots). Problems (red/amber) visually pop against the positive baseline.
2. **Information Density Gradient:** High-level status at top, progressive detail as you drill down. The default view is simple; complexity is opt-in.
3. **Search as Primary Navigation:** Fast, always-visible search bar for devices, alerts, agents. Users shouldn't need to learn a menu hierarchy.
4. **Contextual Actions:** When a device is in alert, offer immediate actions: acknowledge, connect, view history. Reduce clicks to resolution.
5. **Time Range Controls:** Every graph has "1h / 6h / 24h / 7d / 30d / custom" selectors.
6. **Customizable Everything:** Dashboard layouts, widget arrangement, alert thresholds, notification preferences, theme, and sidebar organization should all be user-configurable. Defaults are opinionated; users can override anything.
7. **Progressive Disclosure:** Show simple controls by default, reveal advanced options behind "Advanced" toggles or settings pages. Never overwhelm a first-time user.

### Dashboard Pages

| Page | Route | Description |
|------|-------|-------------|
| Setup Wizard | `/setup` | First-run: create admin, configure network, first scan |
| Dashboard | `/` | Overview: device counts by status, recent alerts, scan activity |
| Devices | `/devices` | Device list with filtering, sorting, search |
| Device Detail | `/devices/:id` | Device info, metrics, topology links, credentials, remote access |
| Topology | `/topology` | Auto-generated network topology map |
| Monitoring | `/monitoring` | Alert list, monitoring status, metric graphs |
| Agents | `/agents` | Scout agent list, enrollment, status |
| Credentials | `/credentials` | Credential management (admin/operator only) |
| Remote Sessions | `/sessions` | Active remote sessions, launch SSH/RDP |
| Settings | `/settings` | Server config, user management, plugin management |
| About | `/about` | Version info, license, Community Supporters, system diagnostics |

### First-Run Setup Wizard

Guided flow triggered when no admin account exists. This is the single most important UX moment in the product -- it determines whether a user continues or abandons. Every step should feel obvious, with no jargon and no dead ends.

1. **Welcome** -- Product overview, what you're about to set up. Friendly tone, not technical.
2. **Create Admin Account** -- Username, email, password. Clear password requirements shown inline.
3. **Network Configuration** -- Auto-detect local subnets, show them with plain-language descriptions ("Home network: 192.168.1.0/24 -- 254 possible devices"). Allow editing for power users, but defaults should just work.
4. **First Scan** -- Trigger initial network scan with live progress. Show devices appearing in real-time as they're discovered. This is the "wow" moment.
5. **Results** -- Show discovered devices with auto-classification (router, desktop, phone, IoT, etc.). Invite user to explore. Offer guided next steps ("Set up monitoring", "Add credentials for remote access").

Goal: Under 5 minutes from first launch to seeing your network. Zero configuration required for the default experience.

### Mobile Responsiveness

Optimized for the "2 AM on-call" workflow:
- Push-capable notification support
- Summary dashboard: critical / warning / ok counts
- Device search and status view
- Acknowledge alerts and schedule downtime
- NOT a full replica of desktop -- focused on triage

---

## Topology Visualization

### Data Sources for Topology

| Protocol | Data Provided | Phase |
|----------|--------------|-------|
| LLDP (Link Layer Discovery Protocol) | Direct neighbor connections, port names | 1 |
| CDP (Cisco Discovery Protocol) | Cisco device neighbors | 1 |
| ARP Tables | IP-to-MAC mappings, indicate shared L2 segments | 1 |
| SNMP Interface Tables | Port descriptions, speeds, status | 2 |
| Traceroute | L3 path between devices | 2 |
| Agent-reported interfaces | Network connections from agent perspective | 1b |

### Topology Map Features (Phase 1)

- Auto-generated from discovery data (LLDP/CDP/ARP)
- Devices as nodes, connections as edges
- Color-coded by status (green=online, red=offline, yellow=degraded)
- Click device to see detail panel
- Click connection to see link speed, utilization
- Zoom, pan, auto-layout with manual override
- Export as PNG/SVG

### Topology Map Features (Phase 2)

- Real-time traffic utilization on links (color gradient: green -> yellow -> red)
- Overlay views: by device type, by subnet, by status
- Custom backgrounds (floor plans, rack diagrams)
- Saved layout persistence

---

## Credential Vault Security

### Encryption Architecture

- **Envelope Encryption:** Each credential encrypted with a unique Data Encryption Key (DEK)
- **DEK wrapping:** Each DEK encrypted with the Master Key (KEK)
- **Master Key Derivation:** Argon2id from admin passphrase (set during first-run)
- **At Rest:** AES-256-GCM for all encrypted data
- **In Memory:** Master key protected via `memguard` (mlock'd memory pages)

### Key Hierarchy

```
Admin Passphrase
    |
    v (Argon2id)
Master Key (KEK) -- stored in memguard, never written to disk
    |
    v (AES-256-GCM wrap)
Data Encryption Key (per credential)
    |
    v (AES-256-GCM encrypt)
Credential Data
```

### Key Management

- Master key derived at server startup from passphrase (interactive or env var)
- Key rotation: new master key re-wraps all DEKs without re-encrypting data
- Passphrase change: re-derive master key, re-wrap all DEKs
- Emergency access: sealed key file encrypted to recovery key (optional)

### Credential Access Audit

Every credential access is logged:

| Field | Description |
|-------|-------------|
| Timestamp | When accessed |
| CredentialID | Which credential |
| UserID | Who accessed |
| Action | read, create, update, delete |
| Purpose | "ssh_session", "snmp_scan", "manual_view" |
| SourceIP | Requester's IP address |

---

## Observability

### Structured Logging

Configurable Zap logger factory supporting:
- **Level:** debug, info, warn, error (configurable, default: info)
- **Format:** json (production), console (development with color)
- **Per-plugin named loggers:** `logger.Named("recon")` for filtering

#### Logging Conventions

| Context | Required Fields |
|---------|----------------|
| HTTP requests | request_id, method, path, status, duration, remote_addr |
| Plugin operations | plugin name (via Named logger) |
| Agent communication | agent_id |
| Database operations | query name, duration |
| Credential access | credential_id, action, user_id |

### Prometheus Metrics

Exposed at `GET /metrics` from day one.

#### Metric Naming Convention

`netvantage_{subsystem}_{metric}_{unit}` (e.g., `netvantage_http_request_duration_seconds`)

#### Core Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `netvantage_http_requests_total` | Counter | method, path, status_code | Total HTTP requests |
| `netvantage_http_request_duration_seconds` | Histogram | method, path | Request latency |
| `netvantage_recon_devices_total` | Gauge | status | Discovered devices by status |
| `netvantage_recon_scans_total` | Counter | status | Network scans by outcome |
| `netvantage_recon_scan_duration_seconds` | Histogram | -- | Scan duration |
| `netvantage_dispatch_agents_connected` | Gauge | -- | Connected Scout agents |
| `netvantage_dispatch_agent_checkins_total` | Counter | -- | Agent check-in RPCs |
| `netvantage_vault_access_total` | Counter | action, success | Credential vault accesses |
| `netvantage_db_query_duration_seconds` | Histogram | query | Database query latency |

### Health Endpoints

| Endpoint | Purpose | Checks |
|----------|---------|--------|
| `GET /healthz` | **Liveness** -- Is the process alive? | Always 200 unless deadlocked. Never call DB. |
| `GET /readyz` | **Readiness** -- Can we handle requests? | DB connectivity, plugin health status. 503 if not ready. |

### OpenTelemetry Tracing (Phase 2)

- OTLP exporter for distributed tracing
- Trace scan operations: ICMP sweep -> ARP scan -> SNMP enrichment -> OUI lookup
- Trace agent check-in pipeline
- 10% sampling rate by default

---

## AI & Analytics Strategy

### Design Philosophy

AI in NetVantage follows the same principles as the rest of the platform: **optional, progressive, and practical**. Every AI feature is a plugin that can be disabled entirely. The core monitoring system works perfectly without any AI capabilities. AI enhances the experience -- it never gates it.

Research into how "AI" works in production monitoring tools (Datadog, Dynatrace, Auvik, Moogsoft) reveals that the most valuable features are fundamentally **statistical algorithms and graph traversal**, not deep learning. Dynamic baselining, anomaly detection, and topology-aware alert correlation deliver 80%+ of the value at a fraction of the computational cost of neural networks.

### Three-Tier Architecture

AI capabilities are organized into three tiers that scale with available hardware. Each tier is independently optional.

| Tier | Name | Compute | Additional RAM | Phase | Default State |
|------|------|---------|---------------|-------|---------------|
| 1 | Statistical Analytics | Pure Go (gonum), in-process | 20–200 MB | 2 | Enabled on `medium`+ profiles |
| 2 | On-Device Inference | ONNX Runtime, in-process | 100–500 MB | 4 | Disabled (opt-in) |
| 3 | LLM Integration | External API or local Ollama | Negligible (API) or 4–16 GB (local) | 3 | Disabled (opt-in, BYOK) |

### Tier 1: Statistical Analytics Engine (Phase 2)

The built-in **Insight** plugin provides always-on statistical analysis using pure Go and gonum. No external dependencies, no GPU, no API keys. This is the highest-value tier.

**Adaptive Baselines:**
- EWMA (Exponentially Weighted Moving Average) for all monitored metrics -- learns each device's "normal" automatically
- Time-of-day and day-of-week seasonal baselines (Holt-Winters family) -- understands that Monday 9 AM traffic differs from Sunday 3 AM
- Per-device, per-metric baselines that adapt to gradual drift (growing traffic over months) while detecting sudden anomalies
- Learning period: 7 days minimum before baselines are considered stable; flags metrics as "learning" in the dashboard

**Anomaly Detection:**
- Z-score detection with configurable sensitivity (default: 3σ) -- flags metrics that deviate significantly from their baseline
- Multivariate correlation: CPU spike + memory increase + network drop occurring simultaneously on one device is a single anomaly, not three
- Change-point detection (CUSUM algorithm) -- identifies when a metric's behavior permanently shifts ("the network got slower starting Tuesday")

**Trend Detection & Capacity Forecasting:**
- Linear regression on sliding windows -- "this disk will be full in 3 days at current growth rate"
- Capacity forecast with confidence intervals displayed on device detail pages
- Proactive warnings before resources reach critical thresholds (not just when they cross a line)

**Topology-Aware Alert Correlation:**
- When a switch goes down, suppress alerts for all devices behind that switch
- Group correlated alerts into a single incident with identified root cause
- Learn device dependency relationships from alert timing patterns (if Device A always alerts 2 seconds before Device B, they are likely dependent)
- Reduce alert volume by 50–90% during cascading failures

**Alert Pattern Learning:**
- Track which alerts are acknowledged vs investigated vs result in action
- Reduce sensitivity for metrics that consistently produce false positives
- Flag "flapping" devices (rapid up/down cycling) for special handling
- Maintenance window prediction: "this device has been rebooted at 2 AM every Sunday for the last 4 weeks"

**Resource requirements (Tier 1):**

| Deployment Size | Additional RAM | Additional CPU | Notes |
|----------------|---------------|----------------|-------|
| 50 devices | 20–50 MB | < 1% of 1 core | Runs on any hardware |
| 200 devices | 80–200 MB | 2–5% of 1 core | Comfortable on RPi 5 |
| 500 devices | 200–500 MB | 5–10% of 1 core | Needs 2 GB+ free RAM |

**Implementation:** gonum for statistical functions + hand-rolled algorithms (~50–300 lines each for EWMA, CUSUM, Holt-Winters, Z-score). No external library dependencies beyond gonum.

### Tier 2: On-Device Model Inference (Phase 4)

Optional machine learning models that run locally via ONNX Runtime. Models are trained offline (in Python) and shipped as ONNX files. Go loads and runs inference without Python.

**Device Fingerprinting:**
- Input features: MAC vendor (OUI), ICMP response characteristics (TTL, timing), open ports, SNMP sysDescr, hostname patterns, mDNS service types
- Output: device type classification with confidence score (e.g., "Raspberry Pi running Home Assistant, 94% confidence")
- Model: Random Forest or Gradient Boosted Trees, 5–20 MB ONNX file
- Inference time: < 10 ms per device on CPU
- Training data: crowd-sourced from opt-in telemetry (anonymized) or user-corrected classifications

**Traffic Classification:**
- Input: flow features (bytes/packets per flow, duration, port, protocol)
- Output: traffic category (web browsing, video streaming, VoIP, backup, IoT telemetry)
- Model: Gradient Boosted Trees (XGBoost/LightGBM), 10–30 MB ONNX file
- Inference time: < 5 ms per flow batch on CPU

**Prerequisites:**
- Minimum 8 GB RAM (model + ONNX Runtime overhead ~100–200 MB on top of base server)
- x64 or ARM64 architecture (ONNX Runtime supports both via onnxruntime_go)
- 500 MB disk for model weight files
- `large` performance profile or explicit opt-in

**Why Phase 4:** Training data for device fingerprinting and traffic classification will not exist at meaningful scale until the platform is deployed to hundreds of users. Tier 1 statistical methods handle 80% of the use cases without training data.

### Tier 3: LLM Integration (Phase 3)

Optional integration with large language models for natural language interaction. Follows a "bring your own API key" model -- NetVantage never requires a paid AI subscription.

**Natural Language Querying:**
- "Show me all devices that had CPU above 90% last Tuesday"
- "Which switches had the most packet loss this month?"
- "What changed on the network in the last 24 hours?"
- Implementation: LLM translates natural language to a structured API query via function calling / tool use, executes it, returns formatted results
- Latency: 1–5 seconds for API round-trips (acceptable for interactive use)

**Incident Summarization:**
- Feed alert timeline + metric data to LLM, receive human-readable incident summary
- Example: "At 3:42 AM, switch SW-CORE-01 began experiencing elevated packet loss (12%) on interface Gi0/1. This correlated with a 40% bandwidth spike from subnet 10.1.5.0/24. The issue resolved at 4:15 AM when the traffic subsided."
- Displayed on alert detail pages and in email/webhook notifications

**Report Generation:**
- Weekly/monthly network health reports in natural language
- Aggregated metrics summarized with highlights and recommendations
- Scheduled (not interactive), so latency is irrelevant

**Configuration Assistance:**
- "Help me configure SNMP v3 for this Cisco switch"
- Context-aware suggestions based on device inventory and existing configuration

**Supported Providers:**

| Provider | Library | Auth | Notes |
|----------|---------|------|-------|
| OpenAI (GPT-4o, GPT-4o-mini) | sashabaranov/go-openai | API key | Lowest latency, function calling support |
| Anthropic (Claude Sonnet, Haiku) | liushuangls/go-anthropic | API key | Tool use support, strong reasoning |
| Ollama (local) | net/http (REST API) | None | Fully offline, requires 8–16 GB RAM for 7B model |

**Privacy Controls:**
- **Data anonymization:** Before sending data to external APIs, replace real IPs with opaque device IDs, strip hostnames, aggregate metrics. Configurable level: none / partial / full.
- **Local-only mode:** Restrict to Ollama only -- no data leaves the premises.
- **Audit logging:** Every LLM API call is logged with the query (sanitized) and the response.
- **No training data contribution:** API providers (OpenAI, Anthropic) do not train on API data per their data processing agreements.

**Cost estimates (external API):**

| Usage Level | Queries/Day | Monthly Cost (GPT-4o-mini) | Monthly Cost (Claude Haiku) |
|-------------|------------|---------------------------|----------------------------|
| Light | 10 | $3–15 | $5–20 |
| Medium | 50 + daily reports | $15–75 | $25–100 |
| Heavy | Interactive + reports | $50–200 | $75–300 |

### Analytics by Performance Profile

| Profile | Tier 1 (Statistical) | Tier 2 (Inference) | Tier 3 (LLM API) | Tier 3 (Local Ollama) |
|---------|---------------------|-------------------|-------------------|----------------------|
| **micro** | Disabled | Disabled | Disabled | Disabled |
| **small** | Basic (EWMA, Z-score only) | Disabled | Available (if API key set) | Disabled |
| **medium** | Full Tier 1 (all algorithms) | Disabled | Available | Disabled |
| **large** | Full Tier 1 | Available (opt-in) | Available | Available (7B model, background tasks) |

### AI Plugin Architecture

The Insight plugin (Tier 1) and optional Tier 2/3 plugins follow the standard NetVantage plugin pattern:

- Implements `Plugin` interface (core lifecycle)
- Implements `EventSubscriber` to receive metric/alert/device events via `PublishAsync` (never blocks metric collection)
- Implements `AnalyticsProvider` for structured analysis queries from other plugins and the API
- Implements `HTTPProvider` to expose `/api/v1/analytics/*` REST endpoints
- Implements `HealthChecker` to report model load status, inference latency, learning progress
- Implements `Validator` to check model files, RAM availability, API key validity at startup
- Declares `Prerequisites` for RAM, disk, and optional external dependencies (ONNX Runtime, Ollama)

**Event subscriptions:**

| Event | AI Use |
|-------|--------|
| `pulse.metrics.collected` | Anomaly detection, baseline updates, trend analysis |
| `recon.device.discovered` | Device classification, risk scoring |
| `recon.device.updated` | Reclassification with new metadata |
| `pulse.alert.triggered` | Alert correlation, pattern learning |
| `pulse.alert.resolved` | Recovery time distribution learning |
| `vault.credential.accessed` | Anomalous access pattern detection |

**New API endpoints (exposed by Insight plugin):**

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/analytics/anomalies` | List detected anomalies with severity, device, metric |
| `GET` | `/api/v1/analytics/anomalies/{device_id}` | Anomalies for a specific device |
| `GET` | `/api/v1/analytics/forecasts/{device_id}` | Capacity forecasts for device metrics |
| `GET` | `/api/v1/analytics/correlations` | Active alert correlation groups |
| `GET` | `/api/v1/analytics/baselines/{device_id}` | Current learned baselines for a device |
| `POST` | `/api/v1/analytics/query` | Natural language query (Tier 3, if configured) |

### Foundation Requirements (Phase 1)

To enable AI features in later phases without retrofit, Phase 1 must establish:

1. **Metrics format:** Every collected metric includes `timestamp`, `device_id`, `metric_name`, `value`, and optional `tags` map. This uniform format is what the analytics engine consumes.
2. **Event bus async support:** `PublishAsync` handlers for slow consumers (already planned). Analytics plugins are the primary consumer of this capability.
3. **Analytics role interface:** Define `AnalyticsProvider` in `pkg/roles/analytics.go` (interface only, no implementation in Phase 1). This establishes the contract that Tier 1/2/3 plugins implement.
4. **Baseline storage schema:** Reserve a `analytics_baselines` table prefix in the database migration plan. The Insight plugin will create its tables during Phase 2 initialization.

These are interface definitions and data format conventions -- zero implementation overhead in Phase 1.

---

## Testing Strategy

Testing is not a phase -- it is a continuous requirement. The test suite is the primary mechanism for ensuring the stability and security commitments in the Non-Functional Requirements. Every feature ships with tests. Every bug fix ships with a regression test. PRs that reduce coverage are rejected by CI.

### Testing Principles

1. **Tests are the stability guarantee.** The Non-Functional Requirements promise months of unattended operation. Only automated tests can verify this at scale.
2. **Fast feedback first.** Unit tests run in < 30 seconds. Integration tests run in < 5 minutes. Developers should never wait to run the fast suite.
3. **Deterministic, not flaky.** Tests that pass sometimes and fail sometimes are worse than no tests. All tests must be deterministic. Time-dependent tests use a mock clock. Network-dependent tests use recorded fixtures or local containers.
4. **Test the contract, not the implementation.** Plugin contract tests verify interface compliance, not internal state. API tests verify request/response pairs, not handler internals. This allows refactoring without rewriting tests.
5. **Coverage targets are minimums, not goals.** Meeting 70% coverage does not mean the code is well-tested. Coverage prevents large untested gaps but does not guarantee quality. Critical paths (auth, encryption, plugin lifecycle) require explicit test cases regardless of coverage numbers.

### Test Categories

#### Unit Tests

- **Plugin contract tests:** Table-driven tests verifying every plugin against the `Plugin` interface and optional interfaces (`HTTPProvider`, `GRPCProvider`, `HealthChecker`, `EventSubscriber`, `Validator`, `Reloadable`, `AnalyticsProvider`). Each plugin is tested in isolation with mocked dependencies.
- **Handler tests:** `httptest.NewRecorder()` for all API endpoints. Every route returns the correct status code, content type, response body structure, and error format (RFC 7807). Every authenticated endpoint rejects unauthenticated requests.
- **Repository tests:** In-memory SQLite (`:memory:`) for database logic. Every repository method tested for CRUD operations, edge cases (empty results, duplicate keys, constraint violations), and transaction behavior.
- **Mock strategy:** Interface-based mocking for external dependencies (PingScanner, ARPScanner, SNMPClient, DNSResolver, CredentialStore, EventBus). Mocks live in `internal/testutil/mocks.go` and are generated from interfaces.
- **SNMP fixtures:** Recorded SNMP responses stored as JSON in `testdata/snmp/`. Tests replay these fixtures instead of querying live devices.
- **Configuration tests:** Every config key has a test for default value, environment variable override, YAML override, and invalid value rejection. `config_version` validation tested for missing, current, old, and future versions.
- **Version validation tests:** Plugin API version checking (too old, too new, exact match, backward-compatible). Config version validation. Database schema version checking.

#### Integration Tests

- **Build tag:** `//go:build integration` -- excluded from `make test`, included in `make test-integration`
- **Database:** `testcontainers-go` for PostgreSQL + TimescaleDB. Integration tests exercise the full database layer including migrations, queries, and TimescaleDB-specific features (hypertables, continuous aggregates).
- **Full server wire-up:** `httptest.Server` wrapping the real HTTP handler (exposed via `Server.Handler()` method). Tests exercise the full request pipeline: middleware, routing, auth, handler, database, response.
- **Plugin lifecycle:** Full Init → Start → Stop cycle with real dependencies (in-memory SQLite for unit, testcontainers for integration).
- **gRPC agent communication:** Full Enroll → CheckIn → ReportMetrics cycle using `bufconn` (in-memory gRPC transport). Tests verify mTLS handshake, version negotiation, metric delivery, and command dispatch.
- **Event bus:** End-to-end event flow: Recon discovers device → event published → Pulse picks up monitoring → metrics collected → analytics baseline updated.

#### Security Tests

Security tests verify every requirement in the Non-Functional Requirements > Security section. These are not optional and run on every PR.

**Authentication & Authorization:**
- JWT token issuance, validation, expiration, and refresh flow
- First-run setup wizard creates admin, rejects if admin exists
- Authenticated endpoints reject missing/expired/malformed tokens
- Password policy enforcement (minimum length, breached password check)
- Account lockout after 5 failed attempts, progressive delay
- Session limit enforcement (max 5 concurrent)
- OIDC/OAuth2 flow when configured (Phase 2)
- MFA/TOTP verification when enabled (Phase 2)

**Transport & Encryption:**
- TLS 1.2+ enforcement (reject TLS 1.0/1.1 connections)
- mTLS certificate validation for agent communication
- Vault credential encryption roundtrip (encrypt → store → retrieve → decrypt)
- Vault master key derivation (Argon2id parameters)
- memguard key protection verification

**Web Security:**
- Security headers present on every response: `Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Strict-Transport-Security`, `Referrer-Policy`, `Permissions-Policy`
- CORS rejects cross-origin requests in production mode
- CSRF protection: `SameSite=Strict` cookies + `X-Requested-With` header validation
- Rate limiting enforced: verify 429 responses after threshold exceeded
- Input validation: reject malformed JSON, oversized payloads, SQL injection attempts, XSS payloads, path traversal attempts in every endpoint that accepts user input

**Secrets Hygiene:**
- Credential values never appear in structured log output (test by capturing zap output)
- Credential values never appear in API error responses
- Credential values never appear in Prometheus metrics labels
- Stack traces redact sensitive fields

#### Stability & Reliability Tests

These tests verify the system can run unattended and degrade gracefully under failure conditions.

**Plugin Isolation:**
- A plugin that panics during `Init()` is caught via `recover()`, logged, and disabled -- server continues starting
- A plugin that panics during `Start()` is caught, logged, and disabled -- other plugins continue
- A plugin that panics in an HTTP handler is caught via middleware recovery -- returns 500, other routes unaffected
- A plugin that blocks forever in `Stop()` is killed after the per-plugin timeout -- shutdown completes
- A plugin that exceeds its memory budget triggers backpressure, not a crash
- Cascade disable: if plugin A fails and plugin B depends on A, plugin B is also disabled with a clear log message

**Graceful Shutdown:**
- SIGTERM triggers orderly shutdown: stop accepting new connections → drain in-flight requests → stop plugins in reverse dependency order → checkpoint database → exit 0
- SIGINT triggers same orderly shutdown
- Second SIGTERM/SIGINT during shutdown forces immediate exit
- Per-plugin stop timeout: plugins that don't stop within timeout are force-killed, remaining plugins still stop cleanly
- Active WebSocket connections receive close frame before server exits

**Database Resilience:**
- SQLite WAL mode: concurrent readers do not block writer
- WAL checkpoint runs correctly (verify WAL file size bounded)
- Database locked error during write retries with backoff (not crash)
- Corrupted WAL file detected at startup with actionable error message
- Migration version too new (from newer server) detected and rejected at startup
- In-flight transactions committed or rolled back cleanly on shutdown

**Circuit Breakers & Timeouts:**
- Every outbound operation (ICMP scan, SNMP query, DNS lookup, HTTP check, Tailscale API call) has a configurable timeout
- Timed-out operations return errors, do not hang goroutines
- Circuit breaker opens after N consecutive failures to an external service, returns fast-fail, closes after cooldown period

**Resource Limits:**
- Event bus queue depth: sustained depth > 1,000 triggers backpressure alert
- Database query queue depth: bounded, excess queries rejected with 503
- Goroutine count: bounded by semaphores for scan workers (not `go func()` in a loop)

#### Performance Tests

Performance tests verify the targets in Non-Functional Requirements > Performance. These run in CI on a schedule (nightly) rather than on every PR, because they require consistent hardware for meaningful results.

**Benchmarks (Go `testing.B`):**
- API response time: `/api/v1/health` < 5ms, `/api/v1/devices` (100 devices) < 50ms, `/api/v1/devices` (1,000 devices) < 100ms
- Database query time: single device lookup < 1ms, device list with pagination < 10ms, metric insertion batch (100 metrics) < 20ms
- Plugin lifecycle: full Init → Start → Stop cycle < 500ms per plugin
- Event bus throughput: 10,000 events/second with 5 subscribers
- JSON serialization: device list (100 devices) < 1ms

**Memory Profiling:**
- Startup memory with zero devices < 50 MB (verify via `runtime.MemStats`)
- Memory at 50 devices < 200 MB (load simulated devices, measure RSS)
- Memory at 200 devices < 500 MB
- Memory at 500 devices < 1.2 GB
- No memory leaks: run 1-hour soak test with continuous scan cycles, verify heap does not grow unbounded (heap in-use at end within 110% of heap in-use at minute 5)

**Startup Time:**
- Server starts and serves `/healthz` within 5 seconds (micro profile)
- Server starts and serves `/healthz` within 10 seconds (large profile with all plugins)

**Scan Performance:**
- /24 subnet ICMP sweep completes in < 30 seconds
- /24 subnet ARP scan completes in < 10 seconds

**Agent Performance:**
- Scout binary size < 15 MB (verified in CI via `ls -la`)
- Scout idle CPU < 1% (measured over 60-second window)
- Scout collection CPU < 5% (measured during metric collection burst)
- Scout memory < 20 MB on x64, < 25 MB on ARM64

#### Version Compatibility Tests

These tests ensure components built for different versions work correctly together, directly validating the Version Management strategy.

**Plugin API Compatibility:**
- Plugin with `APIVersion == PluginAPIVersionCurrent` loads successfully
- Plugin with `APIVersion` in `[PluginAPIVersionMin, PluginAPIVersionCurrent)` loads with deprecation warning
- Plugin with `APIVersion < PluginAPIVersionMin` rejected with clear error
- Plugin with `APIVersion > PluginAPIVersionCurrent` rejected with clear error

**Agent-Server Protocol:**
- Agent with `proto_version == server_proto_version` gets `VERSION_OK`
- Agent with `proto_version == server_proto_version - 1` gets `VERSION_DEPRECATED`, check-in succeeds
- Agent with `proto_version < server_proto_version - 1` gets `VERSION_REJECTED`, check-in fails
- Agent with `proto_version > server_proto_version` gets `VERSION_REJECTED`

**Configuration Compatibility:**
- Config with missing `config_version` treated as version 1, server starts
- Config with `config_version == current` loads normally
- Config with `config_version < current` loads with warning, suggests migration
- Config with `config_version > current` rejects with error

**Database Migration:**
- Fresh database: all migrations run in order, schema matches expected state
- Existing database: only new migrations run, existing data preserved
- Migration from newer server version: detected and rejected at startup
- Per-plugin migrations: each plugin's migrations isolated (recon tables don't affect pulse tables)

#### Database Migration Tests

Migrations are irreversible. Testing them is critical because a broken migration in production requires manual intervention.

- **Fresh install:** Every migration runs successfully on an empty database. Final schema matches expected state (verified by comparing table definitions).
- **Sequential upgrade:** Simulate upgrade path from v0.1.0 through each intermediate version to current. Verify data survives each migration.
- **Per-plugin isolation:** Plugin A's migration does not modify Plugin B's tables. Verified by running each plugin's migrations independently.
- **Idempotent check:** Running migrations twice does not error (migration tracker prevents re-execution).
- **Rollback detection:** If a migration panics, the transaction is rolled back and the migration is not marked as applied. Server logs the failure and refuses to start until the issue is resolved.
- **Reserved prefixes:** `analytics_` table prefix reserved for Phase 2 Insight plugin. Other plugins cannot create tables with this prefix.

#### Fuzz Testing

Go's built-in fuzz testing (`testing.F`) for inputs that cross trust boundaries:

- **API input fuzzing:** Fuzz JSON request bodies for all POST/PUT endpoints. Verify the server never panics, always returns valid HTTP responses (even if 400/500).
- **Configuration fuzzing:** Fuzz YAML configuration input. Verify the server either starts correctly or refuses to start with a clear error (never panics, never corrupts state).
- **SNMP response fuzzing:** Fuzz raw SNMP response bytes. Verify the parser never panics, returns errors for malformed input.
- **gRPC message fuzzing:** Fuzz protobuf message bytes sent to the gRPC server. Verify the server never panics.

Fuzz tests run in CI nightly (not on every PR due to runtime).

#### Cross-Platform Tests

CI verifies the following build targets on every PR:

| Target | Build | Unit Tests | Integration Tests |
|--------|-------|------------|-------------------|
| `linux/amd64` | Yes | Yes | Yes (primary) |
| `linux/arm64` | Yes | Yes (via QEMU or native runner) | No (Phase 2) |
| `windows/amd64` | Yes | Yes | No (Phase 2) |
| `darwin/arm64` | Yes | No (no macOS CI runner) | No |

**Phase 2:** Add ARM64 integration tests (self-hosted ARM64 runner or cross-compilation + QEMU), Windows integration tests.

#### End-to-End Tests (Dashboard)

When the React dashboard is implemented (Phase 1), E2E tests verify the full user journey:

- **Framework:** Playwright (headless Chromium)
- **Scope:** Critical user paths only (not exhaustive UI testing)
- **Test cases:**
  - First-run setup wizard: create admin account, configure network range
  - Dashboard loads and displays device count
  - Device list: search, filter, sort, pagination
  - Trigger manual scan, observe real-time progress via WebSocket
  - Device detail page loads with correct data
  - Topology map renders (visual snapshot test)
  - Settings page: change config, verify saved
  - Login/logout flow
  - Dark mode toggle persists across reload

E2E tests run in CI on every PR (headless, < 2 minutes).

#### Upgrade Tests

Verify that upgrading from one version to another works without data loss or service disruption.

- **Binary upgrade:** Install version N, populate with test data (devices, metrics, alerts, credentials), replace binary with version N+1, restart, verify all data intact and accessible.
- **Database migration upgrade:** Snapshot database at version N, run version N+1 migrations, verify schema correct and data preserved.
- **Config migration:** Load version N config with version N+1 server, verify warning and `netvantage config migrate` produces valid config.
- **Agent compatibility across upgrade:** Server at version N+1 accepts agents still at version N (within N-1 proto window).
- **Rollback detection:** After upgrading server to N+1, starting the old version N binary detects the newer database and refuses to start (does not corrupt data).

### Test Infrastructure

#### Test Commands

```bash
make test              # Unit tests only, -race flag, < 30 seconds
make test-integration  # Integration tests (requires Docker), < 5 minutes
make test-coverage     # Unit tests + coverage report (HTML + text)
make test-fuzz         # Fuzz tests, 30-second budget per fuzz target
make test-bench        # Performance benchmarks
make test-e2e          # End-to-end browser tests (requires built dashboard)
make test-all          # All of the above except fuzz
make lint              # golangci-lint (includes go vet, staticcheck, errcheck, gosec, gocritic)
```

#### Test Directory Layout

```
internal/
  testutil/
    mocks.go          # Generated mocks for all external interfaces
    fixtures.go       # Shared test fixtures (sample devices, metrics, configs)
    helpers.go        # Test helper functions (setup server, create test DB, etc.)
    clock.go          # Mock clock for time-dependent tests
  plugin/
    plugin_test.go    # Plugin interface contract tests
    registry_test.go  # Registry lifecycle, dependency ordering, version checking
  server/
    server_test.go    # HTTP handler tests
    config_test.go    # Configuration loading, validation, version checking
    middleware_test.go # Auth, rate limiting, security headers, request ID
  recon/
    recon_test.go     # Discovery logic with mocked scanners
  pulse/
    pulse_test.go     # Monitoring logic with mocked checks
  (etc. for each module)
pkg/
  plugin/
    plugin_test.go    # SDK contract tests (exported interfaces)
  models/
    models_test.go    # Model validation, serialization
testdata/
  snmp/               # Recorded SNMP responses (JSON fixtures)
  configs/            # Test configuration files (valid, invalid, edge cases)
  migrations/         # Database snapshots for migration testing
test/
  e2e/                # Playwright end-to-end tests
  bench/              # Performance benchmark scenarios
  fuzz/               # Fuzz corpus and seed inputs
```

#### CI Configuration (`.golangci-lint.yml`)

```yaml
linters:
  enable:
    - errcheck         # Unchecked errors
    - gosec            # Security issues (SQL injection, hardcoded creds, weak crypto)
    - gocritic         # Code style and common mistakes
    - govet            # Go vet checks
    - staticcheck      # Comprehensive static analysis
    - ineffassign      # Ineffectual assignments
    - unused           # Unused code
    - misspell         # Spelling mistakes in comments/strings
    - bodyclose        # HTTP response body not closed
    - noctx            # HTTP request without context
    - sqlclosecheck    # SQL rows/stmt not closed
    - exportloopref    # Loop variable capture
    - durationcheck    # Suspicious duration math
    - exhaustive       # Missing enum cases in switch
    - nilerr           # Returning nil error after error check
    - prealloc         # Suggest slice preallocation
```

#### Coverage Targets

| Package | Minimum Coverage | Rationale |
|---------|-----------------|-----------|
| `pkg/plugin/` | 90%+ | Core contract -- bugs here affect every plugin |
| `pkg/roles/` | 90%+ | Role interfaces -- bugs here break module system |
| `internal/server/` | 80%+ | HTTP handling -- user-facing, security-critical |
| `internal/server/middleware/` | 90%+ | Auth, rate limiting, security headers -- security-critical |
| `internal/plugin/` | 85%+ | Registry, lifecycle, version checking -- stability-critical |
| `internal/recon/` | 70%+ | Discovery business logic |
| `internal/pulse/` | 70%+ | Monitoring business logic |
| `internal/dispatch/` | 70%+ | Agent management |
| `internal/vault/` | 85%+ | Credential handling -- security-critical |
| `internal/gateway/` | 70%+ | Remote access |
| `internal/scout/` | 70%+ | Agent core logic |
| `cmd/` | 50%+ | CLI wiring (lower target due to `main()` difficulty) |

**CI enforcement:** Coverage is measured on every PR. If a PR reduces coverage of any package below its minimum, the CI check fails. Coverage reports are uploaded as PR comments for visibility.

### Test Phasing

Not all test categories are needed in Phase 1. The test plan phases with the feature roadmap:

| Test Category | Phase 1 | Phase 1b | Phase 2 | Phase 3 | Phase 4 |
|---------------|---------|----------|---------|---------|---------|
| Plugin contract tests | Yes | Yes | Yes | Yes | Yes |
| HTTP handler tests | Yes | -- | Yes | Yes | Yes |
| Repository tests (SQLite) | Yes | -- | Yes | -- | -- |
| Configuration tests | Yes | -- | Yes | -- | -- |
| Security header tests | Yes | -- | -- | -- | -- |
| Auth/JWT tests | Yes | -- | Yes (OIDC, MFA) | -- | Yes (RBAC) |
| Rate limiting tests | Yes | -- | Yes (per-tenant) | -- | -- |
| Input validation tests | Yes | -- | Yes | Yes | Yes |
| Plugin isolation tests | Yes | -- | -- | -- | -- |
| Graceful shutdown tests | Yes | -- | -- | -- | -- |
| Version compatibility tests | Yes | Yes | Yes | Yes | Yes |
| Database migration tests | Yes | -- | Yes (PostgreSQL) | -- | -- |
| gRPC agent tests | -- | Yes | Yes | -- | -- |
| mTLS certificate tests | -- | Yes | -- | -- | -- |
| E2E browser tests | Yes (basic) | -- | Yes | Yes | -- |
| Performance benchmarks | Yes (baseline) | -- | Yes | -- | Yes |
| Memory profiling | Yes (baseline) | -- | Yes (soak) | -- | Yes |
| Cross-platform builds | Yes | Yes | Yes | Yes | Yes |
| Integration tests (containers) | -- | -- | Yes | Yes | Yes |
| Fuzz testing | Yes (API inputs) | -- | Yes | Yes | -- |
| Upgrade tests | -- | -- | Yes | Yes | Yes |
| Multi-tenancy tests | -- | -- | Yes | -- | -- |
| Analytics algorithm tests | -- | -- | Yes | -- | -- |
| LLM integration tests | -- | -- | -- | Yes | -- |
| ONNX inference tests | -- | -- | -- | -- | Yes |
| MQTT integration tests | -- | -- | -- | -- | Yes |

---

## Deployment

### Single Binary

The Go server embeds:
- Static web assets (`web/dist/` via `embed.FS`)
- Database migrations (via `embed.FS`)
- Default configuration
- OUI database for manufacturer lookup

### Docker Compose (Full Stack)

```yaml
services:
  netvantage:
    image: netvantage/server:latest
    ports:
      - "8080:8080"   # Web UI + API
      - "9090:9090"   # gRPC (Scout agents)
    volumes:
      - netvantage-data:/data
    environment:
      - NV_DATABASE_DSN=/data/netvantage.db
      - NV_VAULT_PASSPHRASE_FILE=/run/secrets/vault_passphrase

  guacamole:  # Optional: only if Gateway module is enabled
    image: guacamole/guacd
    ports:
      - "4822:4822"
```

### Deployment Profiles

Pre-configured module sets for common use cases:

| Profile | Modules Enabled | Use Case |
|---------|----------------|----------|
| **full** | All | Home lab with everything |
| **monitoring-only** | Recon + Pulse | Network awareness without remote access |
| **remote-access** | Vault + Gateway + Recon | Remote access tool without monitoring |
| **msp** | All + multi-tenancy | Managed service provider |

Usage: `netvantage --profile monitoring-only` or copy profile as starting config.

### Performance Profiles (Adaptive Scaling)

The server adapts its resource usage to the available hardware. At startup, it detects available CPU cores and RAM and selects a performance profile automatically. Users can override with `--performance-profile <name>` or in `config.yaml`.

| Profile | Auto-Selected When | Scan Workers | Default Scan Interval | In-Memory Metrics Buffer | Disk Retention | Target Memory | Analytics |
|---------|-------------------|-------------|----------------------|------------------------|---------------|---------------|-----------|
| **micro** | RAM < 1 GB | 1 (sequential) | 10 minutes | 24 hours | 7 days | < 200 MB | Disabled |
| **small** | RAM 1–2 GB | 2 | 5 minutes | 48 hours | 30 days | < 500 MB | Basic (EWMA, Z-score) |
| **medium** | RAM 2–4 GB | 4 | 5 minutes | 72 hours | 90 days | < 1 GB | Full Tier 1 |
| **large** | RAM > 4 GB | 8+ (scales with cores) | 5 minutes (configurable to 1m) | Configurable | Configurable | Configurable | Full Tier 1 + Tier 2/3 opt-in |

**Micro profile constraints:**
- Discovery protocols run sequentially (ARP, then ICMP, then mDNS, then SSDP) to limit peak memory
- Metrics aggregated to 5-minute granularity after 24 hours (reduces storage 5x)
- Topology map limited to 100 nodes before requiring manual pagination
- Dashboard WebSocket updates throttled to 5-second intervals

**Scaling behaviors:**
- Module lazy initialization: subsystems allocate resources on first use, not at startup
- Per-module memory budgets with backpressure: if a module exceeds its budget, it reduces polling frequency and drops oldest buffered metrics
- Discovery concurrency scales with available CPU cores: `min(cores, configured_max_workers)`
- SQLite WAL checkpoint frequency adapts: more aggressive on constrained storage, less frequent on large disks

**Detection override:** Users on constrained hardware who know their workload can override auto-detection. A Raspberry Pi 5 with 8 GB RAM monitoring only 10 devices can safely use the `medium` profile despite the auto-selected `small` profile.

### Configuration

Every setting has a sensible default. A zero-configuration deployment (just run the binary) works out of the box with all modules enabled, SQLite storage, and automatic network detection. Advanced users can customize every aspect via YAML config, environment variables, CLI flags, or the web UI settings page.

**Configuration priority (highest wins):** CLI flags > environment variables > config file > built-in defaults.

**Configuration Format Versioning:**

The configuration file includes a `config_version` field at the root level. This enables the server to detect and handle config files from older or newer versions.

**Rules:**
- `config_version` is an integer, incremented only when a config change is **breaking** (renamed keys, removed keys, changed semantics of existing keys).
- **Additive changes** (new keys with defaults) do NOT increment `config_version`. An old config file works without modification.
- If `config_version` is missing, the server assumes version 1 (backward compatible with pre-versioning configs).
- If `config_version` is newer than the server supports, the server refuses to start with: `"Configuration file version %d is newer than this server supports (max: %d). Upgrade the server or downgrade the config."`
- If `config_version` is older than current, the server starts normally but logs a warning: `"Configuration file version %d is outdated (current: %d). Run 'netvantage config migrate' to update. See release notes for changes."`
- The `netvantage config migrate` CLI command transforms an old config file to the current version, writing a backup of the original.
- **Breaking config changes are rare.** Most config evolution is additive (new keys with sensible defaults). A `config_version` bump is expected roughly once per major server version, if at all.

```yaml
config_version: 1    # Configuration schema version (integer, incremented only on breaking changes)

server:
  host: "0.0.0.0"
  port: 8080
  data_dir: "./data"

logging:
  level: "info"      # debug, info, warn, error
  format: "json"     # json, console

database:
  driver: "sqlite"
  dsn: "./data/netvantage.db"

auth:
  jwt_secret: ""                    # Auto-generated on first run
  access_token_ttl: "15m"
  refresh_token_ttl: "168h"         # 7 days
  oidc:
    enabled: false
    issuer: ""
    client_id: ""
    client_secret: ""
    redirect_url: "http://localhost:8080/api/v1/auth/oidc/callback"

modules:
  recon:
    enabled: true
    scan_interval: "5m"
    methods:
      icmp: true
      arp: true
      snmp: false
  pulse:
    enabled: true
    check_interval: "30s"
  dispatch:
    enabled: true
    grpc_port: 9090
  vault:
    enabled: true
    passphrase_file: ""             # Path to file containing vault passphrase
  gateway:
    enabled: true
    guacamole_address: "guacamole:4822"
  tailscale:
    enabled: false                          # Disabled by default (not all users have Tailscale)
    tailnet: ""                             # Tailnet name (e.g., "example.com" or "user@gmail.com")
    auth_method: "api_key"                  # "api_key" or "oauth"
    api_key_credential_id: ""              # Vault credential ID for API key
    oauth_credential_id: ""                # Vault credential ID for OAuth client
    sync_interval: "5m"                     # How often to poll Tailscale API for device changes
    import_tags: true                       # Import Tailscale ACL tags as NetVantage device tags
    prefer_tailscale_ip: true              # Use Tailscale IPs when device is on tailnet
    discover_subnet_routes: true            # Detect and offer to scan advertised subnet routes
```

Environment variable override prefix: `NV_` (e.g., `NV_SERVER_PORT=9090`, `NV_MODULES_GATEWAY_ENABLED=false`)

---

## Project Infrastructure

This section defines tooling, processes, and documentation infrastructure that supports development across all phases. These are not product features -- they are the systems that enable contributors and AI-assisted development to work efficiently at scale.

### Documentation Architecture

#### Current State: Monolithic `requirements.md`

The project requirements live in a single `requirements.md` file (~3,200+ lines). This works for early planning but creates problems:

- **Context window pressure:** AI assistants must page through large files, consuming context budget on navigation
- **Merge conflicts:** Multiple contributors editing the same file creates frequent conflicts
- **Discoverability:** New contributors can't find relevant sections quickly
- **Partial reads:** Small edits require loading large amounts of unrelated content

#### Target State: Split Documentation (Phase 0)

Migrate to a `docs/requirements/` directory with per-section files:

```
docs/
  requirements/
    README.md                    # Index with brief description of each file
    01-product-vision.md
    02-architecture-overview.md
    03-technology-stack.md
    04-plugin-architecture.md
    05-event-system.md
    06-database-layer.md
    07-authentication.md
    08-scout-agent.md
    09-tailscale-integration.md
    10-data-model.md
    11-api-design.md
    12-brand-identity.md
    13-dashboard-architecture.md
    14-topology-visualization.md
    15-credential-vault.md
    16-observability.md
    17-ai-analytics.md
    18-testing-strategy.md
    19-deployment.md
    20-project-infrastructure.md
    21-phased-roadmap.md
    22-competitive-positioning.md
    23-commercialization.md
    24-system-requirements.md
    25-operations-maintenance.md
    26-release-distribution.md
    27-non-functional-requirements.md
    28-documentation-requirements.md
  adr/
    README.md                    # ADR index and template reference
    template.md                  # MADR template
    0001-split-licensing-model.md
    0002-sqlite-first-database.md
    0003-plugin-architecture-caddy-model.md
    0004-integer-protocol-versioning.md
  guides/
    contributing.md              # Detailed contributor guide (supplements CONTRIBUTING.md)
    plugin-development.md        # Plugin SDK guide (Phase 2+)
    deployment.md                # Deployment guide
```

**Migration rules:**
- Each file is self-contained with its own `## Section Title` as an H2 heading
- Cross-references use relative links: `[Plugin Architecture](04-plugin-architecture.md)`
- CLAUDE.md Documentation Map is updated to reference individual files instead of line ranges
- The original `requirements.md` is replaced with a short redirect pointing to `docs/requirements/README.md`

#### Documentation Site (Phase 2)

- **Generator:** Hugo with Docsy theme (Go-native, used by Kubernetes, Prometheus, gRPC)
- **Hosting:** GitHub Pages (free, auto-deploy via GitHub Actions)
- **Content source:** `docs/` directory -- Hugo reads markdown directly
- **Features:** Versioned docs, search, API reference section, blog
- **Decision rationale:** Hugo is Go-native (aligns with project language), Docsy is the standard for infrastructure projects, GitHub Pages is free and integrated

### Project Tracking & Management

#### GitHub Projects v2

Primary project tracking tool. Free, integrated with GitHub Issues and PRs.

| View | Purpose |
|------|---------|
| **Board** (Kanban) | Sprint-style task tracking: Backlog → In Progress → Review → Done |
| **Roadmap** (Timeline) | Phase milestones on a timeline with dependencies |
| **Table** | Filterable list of all issues with custom fields |

**Custom fields:**
- `Phase`: Phase 0, 1, 1b, 2, 3, 4
- `Module`: Core, Recon, Pulse, Dispatch, Vault, Gateway, Scout, Dashboard, Docs
- `Priority`: Critical, High, Medium, Low
- `Effort`: XS, S, M, L, XL

**Milestone structure:**
- One milestone per phase (e.g., "Phase 1: Foundation")
- Sub-milestones for large feature groups (e.g., "Phase 1: Testing & Quality")

**Label taxonomy:**

| Category | Labels |
|----------|--------|
| Type | `feature`, `bug`, `enhancement`, `refactor`, `docs`, `test`, `chore` |
| Priority | `P0-critical`, `P1-high`, `P2-medium`, `P3-low` |
| Module | `mod:core`, `mod:recon`, `mod:pulse`, `mod:dispatch`, `mod:vault`, `mod:gateway`, `mod:scout`, `mod:dashboard` |
| Phase | `phase:0`, `phase:1`, `phase:1b`, `phase:2`, `phase:3`, `phase:4` |
| Contributor | `good first issue`, `help wanted`, `mentor available` |
| Status | `blocked`, `needs-design`, `needs-review`, `wontfix` |

#### Architecture Decision Records (ADRs)

Format: MADR (Markdown Any Decision Records) -- lightweight, GitHub-rendered.

**When to write an ADR:**
- Technology choice affecting multiple modules
- Architectural pattern adoption
- Security-sensitive design decisions
- Breaking changes to APIs or protocols
- Any decision a future contributor would ask "why?"

**ADR lifecycle:** Proposed → Accepted → Deprecated/Superseded

#### Diagrams

- **Format:** Mermaid (renders natively in GitHub markdown)
- **Location:** Inline in requirement docs or `docs/diagrams/` for complex standalone diagrams
- **Types:** Architecture (C4), sequence diagrams, state machines, entity relationships
- **Tooling:** Mermaid CLI for CI validation, VS Code Mermaid extension for authoring

### Phase-Gated Tooling Research

Before beginning each development phase, research and deploy the tools and processes needed for that phase. This prevents mid-phase disruptions and ensures infrastructure is ready before coding begins.

| Phase | Tooling Research Required |
|-------|--------------------------|
| **Phase 0** | Documentation split tooling, GitHub Projects setup, ADR template, CI/CD pipeline scaffolding, label/milestone structure, development environment documentation |
| **Phase 1** | golangci-lint config, test framework patterns (table-driven, testcontainers), coverage tooling (Codecov), Go Report Card registration, GitHub Actions workflows, Dependabot config, pre-commit hooks |
| **Phase 1b** | gRPC tooling (buf, connect-go evaluation), Windows cross-compilation CI, agent packaging (MSI/WiX evaluation), certificate management libraries |
| **Phase 2** | PostgreSQL/TimescaleDB migration tooling, Docker multi-arch build pipeline, Hugo + Docsy scaffolding, Plausible Analytics deployment, OpenTelemetry SDK integration, SBOM generation (Syft), Cosign signing |
| **Phase 3** | WebSocket/xterm.js integration patterns, Guacamole Docker deployment, AES-256-GCM envelope encryption libraries, Argon2id benchmarks per platform, memguard integration |
| **Phase 4** | MQTT broker evaluation (Eclipse Paho vs alternatives), ONNX Runtime Go bindings, HashiCorp go-plugin patterns, plugin marketplace hosting design |

### AI-Assisted Development Infrastructure

#### Context Conservation Strategy

The project uses Claude Code for AI-assisted development. Context window management is critical for productivity.

**Principles:**
1. **Never read large files in full.** Use section-targeted reads with offset+limit.
2. **Delegate exploration to subagents.** Use Task(subagent_type=Explore) for codebase questions.
3. **Store cross-session knowledge in MCP Memory.** Architecture decisions, user preferences, recurring patterns.
4. **Use plans for multi-step implementations.** The `/create-plan` skill handles context handoffs between phases.
5. **Keep CLAUDE.md current.** The Documentation Map in `.claude/CLAUDE.md` is the primary navigation aid.

**Tooling:**
- MCP Memory server for persistent knowledge graph
- MCP Sequential Thinking for complex reasoning
- Context7 for up-to-date library documentation
- Custom skills for project workflows (plan creation, debugging, requirements generation)

#### Contributor Onboarding for AI-Assisted Workflow

Contributors using Claude Code (or similar AI tools) benefit from:
- A well-maintained `CLAUDE.md` with build commands, conventions, and section map
- Small, focused requirement files (post-split) that fit within context windows
- ADRs explaining past decisions (so AI doesn't re-debate settled questions)
- Consistent code patterns that AI can learn from examples

---

## Phased Roadmap

### Phase 0: Pre-Development Infrastructure

**Goal:** Establish project infrastructure, tooling, and processes before writing product code. Everything here is a prerequisite for efficient Phase 1 development.

#### Documentation Split
- [ ] Create `docs/requirements/` directory structure
- [ ] Split `requirements.md` into per-section files (28 files)
- [ ] Create `docs/requirements/README.md` index with section descriptions
- [ ] Update `.claude/CLAUDE.md` Documentation Map to reference individual files
- [ ] Replace `requirements.md` with redirect to `docs/requirements/README.md`
- [ ] Verify all cross-references resolve correctly

#### Architecture Decision Records
- [ ] Create `docs/adr/` directory with MADR template
- [ ] Write ADR-0001: Split licensing model (BSL 1.1 + Apache 2.0)
- [ ] Write ADR-0002: SQLite-first database strategy
- [ ] Write ADR-0003: Plugin architecture (Caddy model with optional interfaces)
- [ ] Write ADR-0004: Integer-based protocol versioning

#### GitHub Project Setup
- [ ] Create GitHub Projects v2 board with Kanban, Roadmap, and Table views
- [ ] Define custom fields: Phase, Module, Priority, Effort
- [ ] Create milestone for each phase (Phase 0 through Phase 4)
- [ ] Apply label taxonomy: type, priority, module, phase, contributor labels
- [ ] Seed initial issues from Phase 1 checklist items
- [ ] Configure issue templates: bug report, feature request, plugin idea

#### CI/CD Pipeline Scaffolding
- [ ] GitHub Actions: Go build matrix (Linux amd64/arm64, Windows amd64, macOS amd64/arm64)
- [ ] GitHub Actions: test workflow (unit tests with race detector)
- [ ] GitHub Actions: lint workflow (golangci-lint with project config)
- [ ] GitHub Actions: license check workflow
- [ ] GitHub Actions: CLA check workflow (CLA Assistant or custom)
- [ ] Dependabot: configure for Go modules and GitHub Actions
- [ ] Pre-commit hooks: gofmt, go vet, license header check

#### Development Environment
- [ ] Document development setup in `docs/guides/contributing.md`
- [ ] Makefile: verify all targets work on Windows (MSYS2/Git Bash), Linux, macOS
- [ ] `.editorconfig` for consistent formatting across editors
- [ ] VS Code recommended extensions list (`.vscode/extensions.json`)
- [ ] Go workspace configuration (`go.work` if needed for multi-module)

#### Community Health Files
- [ ] CONTRIBUTING.md: fork-and-PR workflow, commit conventions, code review process
- [ ] CODE_OF_CONDUCT.md: Contributor Covenant v2.1
- [ ] SECURITY.md: vulnerability reporting process
- [ ] Pull request template (`.github/pull_request_template.md`)
- [ ] Issue templates: bug, feature, plugin idea (`.github/ISSUE_TEMPLATE/`)

#### Metrics Baseline
- [ ] Register repository on Go Report Card
- [ ] Configure Codecov for coverage tracking
- [ ] Document badge URLs for README (CI, coverage, Go Report Card, license, release)

### Phase 1: Foundation (Server + Dashboard + Discovery + Topology)

**Goal:** Functional web-based network scanner with topology visualization. Validate architecture. Time to First Value under 10 minutes.

#### Pre-Phase Tooling Research
- [ ] Evaluate and configure golangci-lint (15+ linters, project-specific `.golangci-lint.yml`)
- [ ] Establish test framework patterns: table-driven tests, testify assertions, testcontainers for integration
- [ ] Set up Codecov integration for coverage tracking in CI
- [ ] Register repository on Go Report Card
- [ ] Configure GitHub Actions workflows: build, test, lint, license-check
- [ ] Configure Dependabot for Go modules and GitHub Actions
- [ ] Set up pre-commit hooks: gofmt, go vet, license header check
- [ ] Evaluate and document React + TypeScript toolchain for dashboard (Vite, ESLint, Prettier)

#### Architecture & Infrastructure
- [ ] Redesigned plugin system: `PluginInfo`, `Dependencies`, optional interfaces
- [ ] Config abstraction wrapping Viper
- [ ] Event bus (synchronous default with PublishAsync for slow consumers like analytics)
- [ ] Role interfaces in `pkg/roles/` (including `AnalyticsProvider` interface -- definition only, no implementation)
- [ ] Plugin registry with topological sort, graceful degradation
- [ ] Store interface + SQLite implementation (modernc.org/sqlite, pure Go)
- [ ] Per-plugin database migrations (reserve `analytics_` table prefix for Phase 2 Insight plugin)
- [ ] Repository interfaces in `internal/services/`
- [ ] Metrics collection format: uniform `(timestamp, device_id, metric_name, value, tags)` for analytics consumption

#### Server & API
- [ ] HTTP server with core routes
- [ ] RFC 7807 error responses
- [ ] Request ID middleware
- [ ] Structured request logging middleware
- [ ] Prometheus metrics at `/metrics`
- [ ] Liveness (`/healthz`) and readiness (`/readyz`) endpoints
- [ ] Per-IP rate limiting
- [ ] Configuration via YAML + environment variables
- [ ] Configurable Zap logger factory

#### Authentication
- [ ] Local auth with bcrypt password hashing
- [ ] JWT access/refresh token flow
- [ ] First-run setup endpoint (create admin when no users exist)
- [ ] OIDC/OAuth2 optional configuration

#### Recon Module
- [ ] ICMP ping sweep
- [ ] ARP scanning
- [ ] OUI manufacturer lookup (embedded database)
- [ ] LLDP/CDP neighbor discovery for topology
- [ ] Device persistence in SQLite
- [ ] Publishes `recon.device.discovered` events

#### Dashboard
- [ ] React + Vite + TypeScript + shadcn/ui + TanStack Query + Zustand
- [ ] First-run setup wizard
- [ ] Dashboard overview page (device counts, status summary)
- [ ] Device list with search, filter, sort, pagination
- [ ] Device detail page
- [ ] Network topology visualization (auto-generated from LLDP/CDP/ARP)
- [ ] Scan trigger with real-time progress (WebSocket)
- [ ] Dark mode support
- [ ] Settings page (server config, user profile)
- [ ] About page with version info, license, and Community Supporters section

#### Documentation
- [ ] Tailscale deployment guide: running NetVantage + Scout over Tailscale
- [ ] Tailscale Funnel/Serve guide: exposing dashboard without port forwarding

#### Operations
- [ ] Backup/restore CLI commands (`netvantage backup`, `netvantage restore`)
- [ ] Data retention configuration with automated purge job
- [ ] Security headers middleware (CSP, X-Frame-Options, HSTS, etc.)
- [ ] Account lockout after failed login attempts
- [ ] SECURITY.md with vulnerability disclosure process

#### Testing & Quality
- [ ] Test infrastructure: `internal/testutil/` with mocks, fixtures, helpers, mock clock
- [ ] Test infrastructure: `testdata/` directory with SNMP fixtures, test configs, migration snapshots
- [ ] Plugin contract tests: table-driven tests for `Plugin` interface and all optional interfaces
- [ ] Plugin isolation tests: panic recovery in Init, Start, Stop, and HTTP handlers
- [ ] Plugin lifecycle tests: full Init → Start → Stop cycle, dependency ordering, cascade disable
- [ ] Plugin API version validation tests: too old, too new, exact match, backward-compatible range
- [ ] API endpoint tests: `httptest.NewRecorder()` for all routes (status codes, content types, RFC 7807 errors)
- [ ] Security middleware tests: auth enforcement, security headers, CORS, CSRF, rate limiting (429)
- [ ] Input validation tests: malformed JSON, oversized payloads, SQL injection, XSS, path traversal
- [ ] Secrets hygiene tests: verify credentials never appear in log output or error responses
- [ ] Repository tests: in-memory SQLite CRUD, edge cases, transactions, constraint violations
- [ ] Database migration tests: fresh install, sequential upgrade, per-plugin isolation, idempotent check
- [ ] Configuration tests: defaults, env overrides, YAML overrides, invalid values, `config_version` validation
- [ ] Version compatibility tests: Plugin API, agent proto, config version, database schema version
- [ ] Graceful shutdown tests: SIGTERM/SIGINT handling, per-plugin timeout, connection draining
- [ ] Health endpoint tests: `/healthz`, `/readyz`, per-plugin health status
- [ ] Fuzz tests: API input fuzzing, configuration fuzzing (Go `testing.F`)
- [ ] Performance baselines: benchmark key operations, memory profile at 0/50 devices, startup time
- [ ] E2E browser tests: first-run wizard, device list, scan trigger, login/logout (Playwright, headless)
- [ ] CI pipeline: GitHub Actions `ci.yml` with golangci-lint, `go test -race`, build, coverage report, license check
- [ ] CI coverage enforcement: fail PR if any package drops below minimum coverage target
- [ ] `.golangci-lint.yml`: errcheck, gosec, gocritic, staticcheck, bodyclose, noctx, sqlclosecheck
- [ ] GoReleaser configuration for cross-platform binary builds
- [ ] Cross-platform CI: build verification for `linux/amd64`, `linux/arm64`, `windows/amd64`, `darwin/arm64`
- [ ] OpenAPI spec generation (swaggo/swag)

#### Metrics & Measurement Infrastructure
- [ ] Codecov integration: GitHub Action uploads coverage report, badge in README, PR comments with coverage diff
- [ ] Go Report Card: register project at goreportcard.com, add badge to README
- [ ] GitHub Dependabot: enable automated dependency vulnerability alerts
- [ ] GitHub Insights: establish baseline tracking cadence (weekly traffic review)
- [ ] Release download tracking: GoReleaser generates checksums, GitHub Releases API provides download counts
- [ ] Docker image pull count tracking: publish to GitHub Container Registry (GHCR) or Docker Hub
- [ ] README badges: CI build, coverage, Go Report Card, Go version, license, latest release, Docker pulls (see Success Metrics)

#### Community & Launch Readiness
- [ ] CONTRIBUTING.md: development setup, code style, PR process, testing expectations, CLA explanation
- [ ] Pull request template (`.github/pull_request_template.md`) with checklist (tests, lint, description)
- [ ] First tagged release: `v0.1.0-alpha` with pre-built binaries (GoReleaser) and GitHub Release notes
- [ ] Dockerfile: multi-stage build (builder + distroless/alpine runtime), multi-arch (`linux/amd64`, `linux/arm64`)
- [ ] docker-compose.yml: one-command deployment matching the spec in Deployment section
- [ ] README: "Why NetVantage?" section -- value proposition, feature comparison table (discovery + monitoring + remote access + vault + IoT in one tool), clear differentiation from Zabbix/LibreNMS/Uptime Kuma
- [ ] README: status badges (CI build, Go version, license, latest release, Docker pulls)
- [ ] README: Docker quickstart section (`docker run` one-liner + docker-compose snippet)
- [ ] README: screenshots/GIF of dashboard (blocked on dashboard implementation -- placeholder with architecture diagram until then)
- [ ] README: "Current Status" section -- honest about what works today vs. what's planned, links to roadmap
- [ ] README: clarify licensing wording to "free for personal, home-lab, and non-competing production use"
- [ ] Seed GitHub Issues: 5–10 issues labeled `good first issue` and `help wanted` (e.g., "add device type icon mapping", "write Prometheus exporter example plugin", "add ARM64 CI build target")
- [ ] Seed GitHub Discussions: introductory post, roadmap discussion thread, "plugin ideas" thread, "show your setup" thread
- [ ] Community channel: create Discord server (or Matrix space) for real-time contributor discussion, linked from README and CONTRIBUTING.md
- [ ] Blog post / announcement: publish initial announcement on personal blog, r/homelab, r/selfhosted, Hacker News (after v0.1.0-alpha has working dashboard + discovery)
- [ ] CODE_OF_CONDUCT.md: Contributor Covenant (standard, expected by contributors and evaluators)

### Phase 1b: Windows Scout Agent

**Goal:** First agent reporting metrics to server.

#### Pre-Phase Tooling Research
- [ ] Evaluate gRPC tooling: buf vs protoc, connect-go vs grpc-go
- [ ] Research Windows cross-compilation CI (GitHub Actions Windows runners, MSYS2 in CI)
- [ ] Evaluate agent packaging: MSI (WiX Toolset), NSIS, or Go-native installer
- [ ] Research certificate management libraries for mTLS (Go stdlib crypto/x509 patterns)
- [ ] Evaluate Windows service management (golang.org/x/sys/windows/svc)

- [ ] Scout agent binary for Windows
- [ ] Internal CA for mTLS certificate management
- [ ] Token-based enrollment with certificate signing
- [ ] gRPC communication with mTLS
- [ ] System metrics: CPU, memory, disk, network
- [ ] Exponential backoff reconnection
- [ ] Certificate auto-renewal (90-day certs, renew at day 60)
- [ ] Dispatch module: agent list, status, check-in tracking
- [ ] Dashboard: agent status view, enrollment flow
- [ ] Proto management via buf (replace protoc)

### Phase 2: Core Monitoring + Multi-Tenancy

**Goal:** Comprehensive monitoring with alerting. MSP-ready multi-tenancy.

#### Pre-Phase Tooling Research
- [ ] Evaluate PostgreSQL + TimescaleDB: migration tooling (golang-migrate), hypertable performance, connection pooling
- [ ] Research Docker multi-arch build pipeline (buildx, QEMU, manifest lists)
- [ ] Scaffold Hugo + Docsy documentation site, configure GitHub Pages deployment
- [ ] Evaluate Plausible Analytics: self-hosted vs cloud, deployment requirements
- [ ] Research OpenTelemetry Go SDK integration patterns for tracing
- [ ] Evaluate SBOM generation tooling (Syft) and signing (Cosign) for release pipeline
- [ ] Research SNMP Go libraries (gosnmp) and MIB parsing
- [ ] Evaluate mDNS/UPnP discovery libraries (hashicorp/mdns, huin/goupnp)

#### Discovery Enhancements
- [ ] SNMP v2c/v3 discovery
- [ ] mDNS/Bonjour discovery
- [ ] UPnP/SSDP discovery
- [ ] Tailscale plugin: tailnet device discovery via Tailscale API
- [ ] Tailscale plugin: device merging (match by MAC, hostname, or IP overlap)
- [ ] Tailscale plugin: Tailscale IP enrichment on existing device records
- [ ] Tailscale plugin: subnet route detection and scan integration
- [ ] Tailscale plugin: MagicDNS hostname resolution
- [ ] Tailscale plugin: dashboard "Tailscale" badge on tailnet devices
- [ ] Scout over Tailscale: document and support agent communication via Tailscale IPs
- [ ] Topology: real-time link utilization overlay
- [ ] Topology: custom backgrounds, saved layouts

#### Monitoring (Pulse)
- [ ] Uptime monitoring (ICMP, TCP port, HTTP/HTTPS)
- [ ] Sensible default thresholds (avoid alert fatigue)
- [ ] Dependency-aware alerting (router down suppresses downstream alerts)
- [ ] Alert notifications: email, webhook, Slack, PagerDuty (as notifier plugins)
- [ ] Metrics history and time-series graphs
- [ ] Maintenance windows (suppress alerts during scheduled work)

#### Multi-Tenancy
- [ ] TenantID on all core entities (Device, Agent, Credential)
- [ ] Tenant isolation in all queries (row-level filtering)
- [ ] Tenant management API
- [ ] Per-tenant configuration overrides
- [ ] Tenant-scoped API authentication
- [ ] Dashboard: tenant selector for MSP operators

#### Analytics (Insight Module -- Tier 1)
- [ ] Insight plugin implementing `AnalyticsProvider` role
- [ ] EWMA adaptive baselines for all monitored metrics
- [ ] Z-score anomaly detection with configurable sensitivity (default: 3σ)
- [ ] Seasonal baselines (time-of-day, day-of-week patterns via Holt-Winters)
- [ ] Trend detection and capacity forecasting (linear regression on sliding windows)
- [ ] Topology-aware alert correlation (suppress downstream alerts on parent failure)
- [ ] Cross-metric correlation detection (e.g., CPU spike + packet loss on same device)
- [ ] Alert pattern learning (reduce sensitivity for chronic false positives)
- [ ] Change-point detection (CUSUM algorithm for permanent shifts in metric behavior)
- [ ] Dashboard: anomaly indicators on metric charts (highlight deviations from baseline)
- [ ] Dashboard: capacity forecast warnings on device detail pages
- [ ] Dashboard: correlated alert grouping in alert list view
- [ ] API: `/api/v1/analytics/anomalies` and `/api/v1/analytics/forecasts/{device_id}` endpoints
- [ ] Performance-profile-aware: disabled on micro, basic on small, full on medium+

#### Infrastructure
- [ ] PostgreSQL + TimescaleDB support (with hypertables for metrics and continuous aggregates for analytics feature engineering)
- [ ] Scout: Linux agent (x64, ARM64)
- [ ] Agent auto-update with binary signing (Cosign) and staged rollout
- [ ] `nvbuild` tool for custom binaries with third-party modules
- [ ] OpenTelemetry tracing
- [ ] Plugin developer SDK and documentation
- [ ] Dashboard: monitoring views, alert management, metric graphs
- [ ] MFA/TOTP authentication support
- [ ] SBOM generation (Syft) and SLSA provenance for releases
- [ ] Cosign signing for Docker images
- [ ] govulncheck + Trivy in CI pipeline
- [ ] IPv6 scanning and agent communication support
- [ ] Per-tenant rate limiting
- [ ] Public demo instance: read-only demo on free-tier cloud (Oracle Cloud ARM64 or similar) with synthetic data, linked from README and website
- [ ] Project website (GitHub Pages or similar): documentation hub, blog, supporter showcase, demo link
- [ ] Opt-in telemetry: anonymous usage ping (weekly, disabled by default, payload documented and viewable in UI)
- [ ] Telemetry endpoint: simple HTTPS collector for installation count, MAU, feature usage tracking
- [ ] Google Search Console: register project website for organic search traffic tracking
- [ ] Plausible Analytics (self-hosted or cloud): privacy-friendly website analytics for project site
- [ ] Architecture Decision Records (ADRs): establish `docs/adr/` directory with template, document key decisions retroactively
- [ ] SonarQube Community (optional): technical debt tracking if Go Report Card + golangci-lint prove insufficient

### Phase 3: Remote Access + Credential Vault

**Goal:** Browser-based remote access to any device with secure credential management.

#### Pre-Phase Tooling Research
- [ ] Research WebSocket + xterm.js integration patterns for SSH-in-browser
- [ ] Evaluate Apache Guacamole Docker deployment for RDP/VNC proxying
- [ ] Benchmark AES-256-GCM envelope encryption libraries in Go
- [ ] Benchmark Argon2id key derivation across target platforms (cost parameter tuning)
- [ ] Evaluate memguard for in-memory secret protection (Go compatibility, platform support)
- [ ] Research LLM provider SDKs: OpenAI Go client, Anthropic SDK, Ollama local API
- [ ] Evaluate data anonymization approaches for LLM context (PII stripping, metric-only summaries)

- [ ] Gateway: SSH-in-browser via xterm.js
- [ ] Gateway: HTTP/HTTPS reverse proxy via Go stdlib
- [ ] Gateway: RDP/VNC via Apache Guacamole (Docker)
- [ ] Vault: AES-256-GCM envelope encryption
- [ ] Vault: Argon2id master key derivation
- [ ] Vault: memguard for in-memory key protection
- [ ] Vault: Per-device credential assignment
- [ ] Vault: Auto-fill credentials for remote sessions
- [ ] Vault: Credential access audit logging
- [ ] Vault: Master key rotation
- [ ] Dashboard: remote access launcher, session management, credential manager
- [ ] Tailscale plugin: prefer Tailscale IPs for Gateway remote access when device is on tailnet
- [ ] Scout: macOS agent
- [ ] LLM integration: natural language query interface (OpenAI, Anthropic, Ollama providers)
- [ ] LLM integration: incident summarization on alert groups
- [ ] LLM integration: "bring your own API key" configuration in settings
- [ ] LLM integration: privacy controls (data anonymization levels, local-only mode)
- [ ] Dashboard: natural language query bar (optional, appears when LLM configured)
- [ ] Dashboard: AI-generated incident summaries on alert detail pages
- [ ] Vault: anomalous credential access detection (analytics-powered, from audit log events)

### Phase 4: Extended Platform

**Goal:** IoT awareness, ecosystem growth, acquisition readiness.

#### Pre-Phase Tooling Research
- [ ] Evaluate MQTT Go libraries: Eclipse Paho vs alternatives (mochi-co/server for embedded broker)
- [ ] Research ONNX Runtime Go bindings (onnxruntime_go): platform support, model loading, inference performance
- [ ] Evaluate HashiCorp go-plugin for process-isolated third-party plugins (gRPC transport, versioning)
- [ ] Research plugin marketplace hosting: static index vs registry service, discovery UX
- [ ] Evaluate Home Assistant API integration patterns and authentication
- [ ] Research RBAC frameworks for Go (Casbin vs custom implementation)

- [ ] MQTT broker integration (Eclipse Paho)
- [ ] Home Assistant API integration
- [ ] Scout: Lightweight IoT agent
- [ ] API: Public REST API with API key authentication
- [ ] RBAC: Custom roles with granular permissions
- [ ] Audit logging (all state-changing operations)
- [ ] Configuration backup for network devices (Oxidized-style)
- [ ] Plugin marketplace: curated index, `nvbuild` integration
- [ ] Plugin marketplace: AI/analytics plugin category
- [ ] HashiCorp go-plugin support for process-isolated third-party plugins
- [ ] On-device inference: ONNX Runtime integration via onnxruntime_go
- [ ] On-device inference: device fingerprinting model (Python training pipeline + ONNX export)
- [ ] On-device inference: traffic classification model
- [ ] LLM integration: weekly/monthly report generation (scheduled, non-interactive)
- [ ] LLM integration: configuration assistance chatbot
- [ ] Comprehensive documentation: user guide, admin guide, plugin developer guide
- [ ] Performance benchmarks and optimization

---

## Competitive Positioning

### Market Gap

No existing source-available tool combines all five capabilities in a single self-hosted application:

1. Device discovery (network scanning, SNMP, mDNS, auto-topology)
2. Monitoring (uptime, metrics, dependency-aware alerting)
3. Remote access (RDP, SSH, HTTP proxy, no VPN required)
4. Credential management (encrypted vault, per-device, audit logged)
5. IoT/home automation awareness (MQTT, smart devices)

### Detailed Competitor Analysis

| Tool | Strengths | Gaps vs NetVantage | AI/Analytics |
|------|-----------|-------------------|-------------|
| **Zabbix** | Powerful templates, distributed monitoring, huge community | Steep learning curve (6+ months), no remote access, no credentials, GPL license, users add Grafana for visualization | Static thresholds only; users bolt on external ML tools |
| **LibreNMS** | Excellent auto-discovery, SNMP-focused, welcoming community | PHP/LAMP stack feels dated, no remote access, no credentials, slow with 800+ devices | Basic heuristic discovery; no anomaly detection |
| **Checkmk** | Best auto-discovery agent, rule-based config | Edition confusion (free features disappear after trial), learning curve | Rule-based discovery; no ML or dynamic baselines |
| **PRTG** | Best setup experience (<1hr), beautiful maps | Windows-only server, sensor-based pricing shock, no Linux server | Map visualization; basic correlation; no ML |
| **MeshCentral** | Free RMM replacement, Intel AMT support | UI looks dated, weak discovery, no monitoring depth, no dashboards | None |
| **Uptime Kuma** | Best UX in monitoring, beautiful, 50K+ GitHub stars | Monitoring only, no SNMP, no agents, no discovery, SQLite scale limits | None |
| **Domotz** | Best MSP remote access, TCP tunneling | Proprietary, cloud-dependent, $21/site/month, shallow monitoring | Basic device fingerprinting; no anomaly detection |
| **Netbox** | Gold standard IPAM/DCIM, excellent API | Documentation only, no monitoring, no remote access | None |

### Adoption Formula (From Research)

```
Time to First Value < 10 minutes     (Uptime Kuma, PRTG model)
+ Beautiful by Default               (Uptime Kuma model)
+ Auto-Discovery that Reduces Work   (LibreNMS, Checkmk model)
+ Depth Available When Needed        (Zabbix model, progressive disclosure)
+ Intelligent Analytics Built-in     (No competitor offers this in a self-hosted tool)
+ Fair Pricing / Truly Free          (Zabbix, LibreNMS model)
+ Active Community                   (all successful tools)
+ Proof It Works                     (release, CI badge, demo, screenshots)
= Mass Adoption
```

**Critical adoption insight:** A project with no releases, no CI badge, no screenshots, and empty issues/discussions reads as abandoned or not-yet-started -- regardless of code quality. The pre-launch checklist in Community Engagement & Launch Strategy addresses this directly.

**Analytics Differentiation:** No self-hosted / source-available monitoring tool offers built-in adaptive baselines, anomaly detection, or LLM integration. Enterprise SaaS tools (Datadog, Dynatrace) charge $15-30+/host/month for these capabilities. NetVantage delivers the same core algorithms (EWMA, Holt-Winters, topology-aware correlation) at zero additional cost in the free tier.

### User Segment Priorities

| Segment | Top Need | NetVantage Differentiator | Typical Hardware | Typical Devices |
|---------|----------|--------------------------|-----------------|----------------|
| **Home Lab (beginner)** | Simple visibility into all home devices | Auto-discovery + topology in 10 minutes | RPi 4/5, Docker on NAS | 15–30 (router, switch, AP, IoT, personal devices) |
| **Home Lab (enthusiast)** | Single pane of glass replacing 3–5 separate tools | Discovery + monitoring + topology + remote access + credential vault | N100 mini PC, Proxmox VM, refurb enterprise micro | 50–200 (managed switches, VLANs, 20–50 containers, NAS, cameras, IoT) |
| **Prosumer / Remote Worker** | Network reliability for income-dependent connectivity | Latency monitoring, ISP vs VPN diagnostics, Tailscale integration | N100 mini PC, cloud VPS | 20–75 (work + personal + IoT, Tailscale mesh) |
| **Small Biz IT (5–25 employees)** | Minimal maintenance + zero-config monitoring | Setup wizard + sensible defaults + auto-discovery | Existing server VM, NAS Docker, or $200 mini PC | 20–90 (router, switches, APs, endpoints, printers, VoIP phones) |
| **Small Biz IT (25–50 employees)** | Proactive monitoring to prevent outages | SNMP monitoring + dependency-aware alerting + reports | VM on existing Hyper-V/Proxmox host | 50–170 (firewall, managed switches, APs, servers, endpoints, UPS) |
| **MSP** | Multi-tenant + remote access without VPN | Tenant isolation + Gateway module + low per-site cost | Cloud VPS or on-prem appliance per client site | 50–500 per client, 500–5,000 across portfolio |

---

## Commercialization Strategy

### Strategic Intent

**Free for personal and home use forever.** This is a firm commitment, not a marketing tactic. Home lab enthusiasts, students, and hobbyists will always have full access to every feature at no cost. This community is the foundation of adoption, feedback, and evangelism.

**Commercial for business use.** Organizations using NetVantage in a professional capacity (MSPs, enterprises, commercial IT operations) are the revenue target. Commercial tiers add multi-user, multi-tenant, SSO, RBAC, audit logging, and priority support -- features businesses need that home users typically don't.

**Built for acquisition.** The codebase, documentation, community, and clean IP chain are the product -- not just the software. Every architectural decision, license choice, and documentation effort is made with the awareness that this project is designed to be attractive for acquisition by a larger platform company.

### Licensing & Intellectual Property

#### Split Licensing Model

| Component | License | Rationale |
|-----------|---------|-----------|
| **Core Server + Scout Agent** | BSL 1.1 (Business Source License) | Protects commercial rights; prevents competing hosted offerings; acquirer-friendly (HashiCorp/IBM precedent) |
| **Plugin SDK** (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`) | Apache 2.0 | Maximizes plugin ecosystem adoption; no friction for community or commercial plugin authors |
| **Protobuf Definitions** (`api/proto/`) | Apache 2.0 | Allows third-party agents and integrations |
| **Community Plugins** (`plugins/community/`) | Apache 2.0 (recommended default) | Contributors choose; Apache 2.0 template provided |

#### BSL 1.1 Terms (Core)

- **Change Date:** 4 years from each release date
- **Change License:** Apache 2.0 (code auto-converts after Change Date)
- **Additional Use Grant:** Non-competing production use permitted. Personal, home-lab, and educational use always permitted regardless of this grant.
- **Commercial Use:** Requires a paid license from the copyright holder for:
  - Offering NetVantage as a hosted/managed service
  - Embedding NetVantage in a commercial product that competes with NetVantage offerings
  - Reselling or white-labeling NetVantage

#### Contributor License Agreement (CLA)

- **Required** for all contributions via CLA Assistant (GitHub App)
- Contributors sign once via GitHub comment on their first PR
- Grants the project owner:
  - Copyright assignment or broad license grant to contributions
  - Right to relicense contributions under any terms
  - Patent license for contributions
- **Essential for acquisition:** Clean IP ownership chain required by acquirers

#### Trademark

- Use **NetVantage™** (common-law TM symbol) immediately to establish rights
- Defer USPTO registration until closer to commercialization
- Trademark policy: forks may not use the "NetVantage" name
- Trademark guidelines documented in TRADEMARK.md

#### Dependency Compliance

- `go-licenses` integrated into CI pipeline
- Block any dependency with GPL, AGPL, LGPL, or SSPL license (incompatible with BSL 1.1)
- Allowed: MIT, BSD-2, BSD-3, Apache 2.0, ISC, MPL-2.0 (file-level copyleft only)
- License audit report generated on every build
- **Dual-licensed packages:** `eclipse/paho.mqtt.golang` -- elect EDL-1.0 (BSD-3-Clause) option
- **Weak copyleft:** `hashicorp/go-plugin` (MPL-2.0) -- use as unmodified library only
- **Docker images:** Use only official `guacamole/guacd` (Apache 2.0); avoid `flcontainers/guacamole` (GPL v3)
- Full dependency audit completed: **zero incompatible dependencies** found across all Go and npm packages

#### Repository Licensing Structure

```
d:\NetVantage\
  LICENSE                    # BSL 1.1 (covers everything by default)
  LICENSING.md              # Human-readable explanation of the licensing model
  pkg/
    plugin/
      LICENSE               # Apache 2.0
    roles/
      LICENSE               # Apache 2.0
    models/
      LICENSE               # Apache 2.0
  api/
    proto/
      LICENSE               # Apache 2.0
  plugins/
    community/
      LICENSE               # Apache 2.0 (template)
```

### Pricing Model: Hybrid (No Device Limits)

All tiers have **unlimited devices and unlimited customization**. Pricing based on team/business features, not scale or functionality. A home user with 500 devices gets the same core capabilities as an enterprise with 5,000.

| Tier | Price | Target | Features |
|------|-------|--------|----------|
| **Community** | Free forever | Home, personal, educational | All modules, all plugins, unlimited devices, single user, full customization, community support |
| **Team** | $9/month | Small business, teams | + Multi-user (up to 5), OIDC/SSO, scheduled reports, email support |
| **Professional** | $29/month | MSPs, mid-size orgs | + Multi-tenant (up to 10 sites), RBAC, audit logging, API access, priority support |
| **Enterprise** | $99/month | Large organizations | + Unlimited tenants, custom branding, dedicated support, SLA |

**Principle:** The free tier is never crippled. It includes every module, every plugin, and every customization option. Paid tiers add collaboration and business operations features (multi-user, multi-tenant, SSO, audit trails) that are genuinely unnecessary for a solo home user.

### Community Contributions

Free and home users are the foundation of the project. Their contributions are valued and recognized.

#### Non-Financial Contributions

| Contribution | Channel | Recognition |
|-------------|---------|-------------|
| Bug reports | GitHub Issues (templated) | Contributor credit in release notes |
| Feature requests | GitHub Discussions | Acknowledgment if implemented |
| Beta testing | Opt-in beta channel | Early access + tester recognition |
| Documentation | Pull requests | Contributor credit + CLA on file |
| Plugin development | Apache 2.0 SDK | Listed in plugin directory |
| Community support | GitHub Discussions | Community helper recognition |

#### Voluntary Financial Support

Three platforms, zero obligation. All support is voluntary and does not unlock additional features -- the free tier is always complete.

| Platform | Type | Link |
|----------|------|------|
| **GitHub Sponsors** | Recurring or one-time | github.com/sponsors/HerbHall |
| **Ko-fi** | Recurring or one-time | ko-fi.com/herbhall |
| **Buy Me a Coffee** | One-time or membership | buymeacoffee.com/herbhall |

Configured via `.github/FUNDING.yml` for GitHub's native "Sponsor" button integration.

#### Supporter Recognition

Financial supporters are recognized in the product and repository:

| Tier | Threshold | Recognition |
|------|-----------|-------------|
| **Supporter** | $5+/mo or $25+ one-time | Name in `SUPPORTERS.md` + in-app About page "Community Supporters" section |
| **Backer** | $25+/mo or $100+ one-time | Above + name on project website |
| **Champion** | $100+/mo or $500+ one-time | Above + logo/link on README and website |

**In-app recognition:** The dashboard About/Settings page includes a "Community Supporters" tab displaying supporter names. This is a visible, permanent acknowledgment of community investment. Supporters list is maintained in `SUPPORTERS.md` and bundled with each release.

**Signals to acquirers:** A named list of financial supporters demonstrates genuine community investment beyond GitHub stars and download counts.

### Community Engagement & Launch Strategy

A technically excellent project with zero community engagement is invisible. The GitHub evaluation identified that all the foundational code work is meaningless if no one can find the project, understand what it does, or feel confident it actually works. These items are not optional polish -- they are adoption prerequisites.

#### Pre-Launch Checklist (Before First Public Announcement)

Every item must be complete before any public promotion (blog posts, Reddit, Hacker News):

| Item | Why It Matters |
|------|---------------|
| `v0.1.0-alpha` tagged release with binaries | "No releases" = "this doesn't work yet" in visitor perception |
| CI pipeline with passing badge in README | Proves the code compiles and tests pass -- basic credibility signal |
| Docker one-liner in README | Lowest friction path to "Time to First Value < 10 minutes" |
| At least one screenshot or GIF in README | Visitors decide in 10 seconds; a wall of text loses them |
| "Why NetVantage?" section in README | Answers the immediate question every visitor has |
| "Current Status" section in README | Honesty about what works prevents disappointment and builds trust |
| CONTRIBUTING.md | Contributors need a clear path; without it, PRs don't happen |
| 5+ labeled issues (`good first issue`, `help wanted`) | Contributors scan issues first; an empty issue tracker signals a dead project |

#### Contributor Onboarding Funnel

The goal is to convert passive visitors into active contributors. Each step reduces friction:

```
1. GitHub visitor reads README → understands value proposition (Why NetVantage?)
2. Visitor tries Docker quickstart → sees it work in < 10 minutes
3. User files a bug or feature request → issue templates guide quality reports
4. Interested dev reads CONTRIBUTING.md → knows how to set up dev environment
5. Dev picks a "good first issue" → scoped, achievable, well-described
6. Dev submits PR → PR template ensures quality, CLA bot handles IP
7. Maintainer reviews promptly → contributor feels valued, contributes again
```

**Key insight from evaluation:** An empty GitHub Issues tab and empty Discussions tab signal an inactive or abandoned project. Seeding these with genuine items (real bugs found during development, real architectural questions, real feature ideas) is not artificial -- it's making internal knowledge public.

#### Launch Announcement Strategy

Sequence matters. Announce only after the pre-launch checklist is complete.

| Channel | Timing | Content |
|---------|--------|---------|
| GitHub Release (`v0.1.0-alpha`) | Day 0 | Release notes, binary downloads, Docker image |
| Personal blog post | Day 0 | "Why I built NetVantage" -- problem statement, architecture choices, what works, what's planned |
| r/selfhosted | Day 0–1 | Show & Tell post, link to blog, Docker quickstart |
| r/homelab | Day 0–1 | Focus on homelab use case, hardware requirements, screenshots |
| Hacker News (Show HN) | Day 1–2 | Technical focus, architecture, BSL licensing rationale |
| Discord/Matrix | Ongoing | Real-time Q&A, feedback collection, community building |
| GitHub Discussions | Ongoing | Roadmap feedback, plugin ideas, deployment guides |

#### Community Channels

| Channel | Purpose | Phase |
|---------|---------|-------|
| GitHub Issues | Bug reports, tracked feature requests | 1 (exists) |
| GitHub Discussions | General Q&A, roadmap discussion, plugin ideas, show-your-setup | 1 |
| Discord server (or Matrix space) | Real-time chat, contributor coordination, support | 1 |
| Project website | Documentation, blog, supporter recognition | 2 |

**Discord vs. Matrix:** Discord has lower friction for most users (no account needed to browse, familiar UI). Matrix is better for open-source purists and bridge compatibility. Starting with Discord is recommended; bridge to Matrix later if demand exists.

### Acquisition Readiness Checklist

| Attribute | Requirement | Measurable Target |
|-----------|------------|-------------------|
| **Clean architecture** | Modular plugin system, clear separation of concerns, documented interfaces | Go Report Card A+ grade, architecture decision records documented |
| **Test coverage** | 70%+ across core packages, CI/CD pipeline | Codecov badge showing 70%+, green CI on main branch, < 5% flaky test rate |
| **Security posture** | Zero critical vulnerabilities, documented scan history | govulncheck + Trivy clean, Dependabot enabled, SECURITY.md published |
| **Documentation** | User guide, admin guide, plugin developer guide, API reference (OpenAPI) | 100% API endpoints documented, installation guide tested on 3+ platforms |
| **Community** | Active GitHub discussions, contributor guidelines, plugin ecosystem | 10+ contributors, < 7 day median issue response, Discord active |
| **Legal** | BSL 1.1 core, Apache 2.0 SDK, CLA, trademark, dependency audit | go-licenses clean in CI, CLA on 100% of contributions, trademark filed |
| **Adoption metrics** | Tracked and growing user base | 1,000+ stars, 1,000+ Docker pulls, 100+ verified installs (telemetry) |
| **Revenue** | Demonstrable paid tier adoption | First paid user, documented conversion path, pricing page live |
| **Operational maturity** | Bus factor > 1, reproducible builds, release process | 10+ contributors, GoReleaser automated, SBOM on every release |

### Success Metrics & Measurement

Metrics are only useful if they are tracked consistently from early in the project lifecycle. This section defines what to measure, how to measure it, which tools to use, and what targets to aim for. Implementation is phased -- lightweight metrics start in Phase 1, deeper analytics in Phase 2+.

#### Metric Categories

##### 1. User & Community Metrics

| Metric | Source / Tool | Phase | Target (12 months) | Why It Matters |
|--------|--------------|-------|---------------------|----------------|
| GitHub Stars | GitHub Insights (built-in) | 1 | 1,000+ | Social proof and discoverability threshold |
| GitHub Forks | GitHub Insights | 1 | 100+ | Shows people building on or experimenting with the project |
| Contributors | GitHub Insights | 1 | 10+ (beyond maintainer) | Bus factor, sustainability signal |
| Issues (open/closed ratio) | GitHub Insights | 1 | Closed > open, < 7 day median response time | Responsiveness signal -- stale issues repel contributors |
| Release downloads | GitHub Releases API | 1 | 500+ cumulative across first 3 releases | Actual adoption beyond stars |
| Docker pulls | Docker Hub / GitHub Container Registry | 1 | 1,000+ | The primary installation metric for self-hosted tools |
| Discord/Matrix members | Discord server analytics | 1 | 100+ | Engaged community beyond GitHub |
| GitHub Discussions activity | GitHub Insights | 1 | 5+ threads/month with responses | Shows an active, helpful community |

##### 2. Code Quality Metrics

| Metric | Tool | Phase | Target | Why It Matters |
|--------|------|-------|--------|----------------|
| Test coverage | Codecov (free for open source) | 1 | 70%+ overall, 90%+ on core contracts | Verifiable quality signal for users and acquirers |
| Code quality grade | Go Report Card (goreportcard.com) | 1 | A+ grade | Zero-effort badge that signals Go best practices |
| Security vulnerabilities | Snyk or GitHub Dependabot + govulncheck + Trivy | 1 | Zero critical/high, < 5 medium | Clean security posture is table stakes |
| License compliance | go-licenses in CI (already planned) | 1 | Zero incompatible dependencies | Required for BSL 1.1 integrity and acquisition |
| Linter issues | golangci-lint in CI (already planned) | 1 | Zero warnings on main branch | Prevents quality erosion |
| Documentation coverage | Manual review + OpenAPI completeness | 1 | 100% of API endpoints documented | Users and integrators need complete references |
| CI pipeline health | GitHub Actions badge | 1 | Green main branch, < 5% flaky test rate | Confidence signal for contributors |

##### 3. Adoption & Growth Metrics

| Metric | Source / Tool | Phase | Target (12 months) | Why It Matters |
|--------|--------------|-------|---------------------|----------------|
| Active installations | Opt-in telemetry (anonymous ping) | 2 | 100+ verified | The only metric that proves real-world use |
| Monthly active users (MAU) | Opt-in telemetry | 2 | 50+ | Actual engagement, not just downloads |
| Retention (30-day) | Opt-in telemetry | 2 | 60%+ | Are users sticking around after first try? |
| Time to first value | User testing / dogfooding | 1 | < 10 minutes (design goal, already stated) | First experience determines adoption |
| Feature usage distribution | Opt-in telemetry | 2 | Identify top 5 features by usage | Guides development priority |
| Organic search traffic | Google Search Console (project website) | 2 | 500+ monthly impressions | Discoverability without paid promotion |
| Mentions / backlinks | GitHub search, Reddit, blog mentions | 1 | 10+ organic mentions | Word of mouth is the strongest growth signal |

##### 4. Enterprise & Acquisition Appeal Metrics

| Metric | How to Demonstrate | Phase | Why Acquirers Care |
|--------|--------------------|-------|-------------------|
| Addressable market (TAM/SAM/SOM) | Market research document | 2 | Validates revenue potential |
| Conversion path (free → paid) | Pricing page + documented funnel | 2 | Revenue predictability |
| Competitive differentiation | Feature comparison + unique capabilities | 1 (already done) | Why buy this vs. build or buy competitor |
| Switching costs | Plugin ecosystem + integrations + data lock-in | 3 | Defensibility after adoption |
| Bus factor | Contributors, documentation quality, architecture | 1 | Can it survive without the founder? |
| Technical debt | SonarQube or CodeClimate report | 2 | Integration complexity for acquirer |
| Security audit history | govulncheck + Trivy scan archive in CI | 1 | Due diligence requirement |
| CLA coverage | CLA Assistant tracking (already implemented) | 1 | Clean IP chain for acquisition |
| Architecture decision records | ADRs in `docs/adr/` | 2 | Shows deliberate, documented decisions |

#### Measurement Tools & Implementation

| Tool | Purpose | Cost | Phase | Integration Method |
|------|---------|------|-------|-------------------|
| **GitHub Insights** | Stars, forks, traffic, clones, referrers | Free (built-in) | 1 | Check Settings → Insights → Traffic weekly |
| **Codecov** | Test coverage tracking + PR comments | Free for open source | 1 | GitHub Action uploads coverage after `make test-coverage` |
| **Go Report Card** | Code quality grade + badge | Free | 1 | Register at goreportcard.com, add badge to README |
| **GitHub Dependabot** | Dependency vulnerability alerts | Free (built-in) | 1 | Enable in repository settings |
| **govulncheck** | Go-specific vulnerability scanning | Free (Go toolchain) | 1 | Already planned in CI pipeline |
| **Trivy** | Container image vulnerability scanning | Free | 1 | Already planned in CI pipeline |
| **go-licenses** | License compliance checking | Free | 1 | Already planned in CI pipeline |
| **golangci-lint** | Static analysis + linting | Free | 1 | Already planned in CI pipeline |
| **Google Search Console** | Organic search traffic for project website | Free | 2 | Register when project website launches |
| **Plausible Analytics** | Privacy-friendly website analytics | Free (self-hosted) or $9/mo | 2 | Embed in project website (NOT in the NetVantage product) |
| **Opt-in telemetry** (custom) | Active installations, MAU, feature usage | Free (custom implementation) | 2 | See Telemetry Design below |
| **SonarQube Community** | Technical debt + code smell tracking | Free (self-hosted) | 2 | Optional -- Go Report Card + golangci-lint may suffice |

#### Opt-In Telemetry Design (Phase 2)

Telemetry is the only way to measure actual adoption vs. downloads. It must be designed with privacy as the primary constraint.

**Principles:**
1. **Opt-in only.** Telemetry is disabled by default. Users explicitly enable it in settings or during the first-run wizard ("Help improve NetVantage by sharing anonymous usage data").
2. **Anonymous.** No IP addresses, hostnames, device names, credentials, or network topology sent. Ever.
3. **Transparent.** Users can see exactly what is sent before enabling. The telemetry payload is documented and viewable in the UI.
4. **Minimal.** Only aggregate data needed to answer specific questions (see payload below).
5. **No third-party services.** Telemetry data sent to a NetVantage-operated endpoint, not Google Analytics, Mixpanel, or similar.

**Telemetry payload (sent weekly):**

```json
{
  "v": 1,
  "installation_id": "random-uuid-generated-on-first-enable",
  "server_version": "1.3.2",
  "os": "linux",
  "arch": "amd64",
  "performance_profile": "medium",
  "device_count": 47,
  "agent_count": 3,
  "enabled_modules": ["recon", "pulse", "dispatch"],
  "database_driver": "sqlite",
  "uptime_hours": 168,
  "features_used": ["topology_map", "dark_mode", "scan_schedule"]
}
```

- `installation_id` is a random UUID generated when telemetry is first enabled. It is not tied to any user identity. It enables deduplication (count unique installations, not unique pings).
- No fields contain user data, device data, network data, or credentials.
- The endpoint is a simple HTTPS POST to `telemetry.netvantage.io` (or similar). The server responds with 200 OK and no body.

**Configuration:**

```yaml
telemetry:
  enabled: false                    # Opt-in only, default off
  endpoint: "https://telemetry.netvantage.io/v1/ping"
  interval: "168h"                  # Weekly
```

**Dashboard indicator:** When telemetry is enabled, a small indicator in the About page shows "Anonymous usage data sharing: enabled" with a link to the exact payload that was last sent.

#### README Badges

Badges provide at-a-glance confidence signals. Add these to the README as each becomes available:

| Badge | Source | When to Add |
|-------|--------|-------------|
| CI Build (passing/failing) | GitHub Actions | Phase 1 (when `ci.yml` is implemented) |
| Test Coverage (%) | Codecov | Phase 1 (when coverage reporting is set up) |
| Go Report Card (A+) | goreportcard.com | Phase 1 (when code quality is stable) |
| Go Version | shields.io | Phase 1 (immediately) |
| License (BSL 1.1) | shields.io | Phase 1 (immediately) |
| Latest Release | GitHub Releases | Phase 1 (when v0.1.0-alpha is tagged) |
| Docker Pulls | Docker Hub / GHCR | Phase 1 (when Docker images are published) |
| Vulnerabilities | Snyk or GitHub | Phase 2 (when security scanning is mature) |

**Badge markdown (target state):**

```markdown
[![Build](https://github.com/HerbHall/netvantage/actions/workflows/ci.yml/badge.svg)](https://github.com/HerbHall/netvantage/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/HerbHall/netvantage/branch/main/graph/badge.svg)](https://codecov.io/gh/HerbHall/netvantage)
[![Go Report Card](https://goreportcard.com/badge/github.com/HerbHall/netvantage)](https://goreportcard.com/report/github.com/HerbHall/netvantage)
[![Go Version](https://img.shields.io/github/go-mod/go-version/HerbHall/netvantage)](go.mod)
[![License](https://img.shields.io/badge/license-BSL%201.1-blue)](LICENSE)
[![Release](https://img.shields.io/github/v/release/HerbHall/netvantage)](https://github.com/HerbHall/netvantage/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/herbhall/netvantage)](https://hub.docker.com/r/herbhall/netvantage)
```

#### Milestone Targets

Concrete milestones that signal project health at each stage:

| Milestone | Target | Timeframe | Significance |
|-----------|--------|-----------|-------------|
| **v0.1.0-alpha** | First release with working discovery + dashboard | Phase 1 | "It exists and works" |
| **100 GitHub stars** | Organic growth from announcement posts | 1–3 months post-launch | Initial visibility |
| **1,000 GitHub stars** | Credibility threshold | 6–12 months | Social proof for new visitors |
| **10 contributors** | Community beyond maintainer | 6 months | Sustainability signal |
| **100 Docker pulls** | Early adoption | 1–2 months post-launch | People are trying it |
| **1,000 Docker pulls** | Real traction | 6–12 months | Consistent adoption |
| **100 active installs** | Verified via opt-in telemetry | Phase 2 | Real-world usage proof |
| **First paid user** | Team or Professional tier | Phase 2+ | Revenue validation |
| **5 case studies** | User testimonials or "show your setup" posts | 12 months | Social proof for acquirers |
| **Clean security audit** | Zero critical vulnerabilities, documented scan history | Ongoing | Due diligence readiness |

#### Metrics Review Cadence

| Frequency | Review | Action |
|-----------|--------|--------|
| Weekly | GitHub traffic (Settings → Insights → Traffic), Discord activity | Note trends, respond to spikes |
| Monthly | Stars, forks, contributors, Docker pulls, issue response time, coverage % | Update internal tracking, adjust priorities |
| Quarterly | Adoption metrics (telemetry), search traffic, competitive landscape | Strategic review, roadmap adjustments |
| Per release | Download counts, bug reports, user feedback | Release retrospective, quality assessment |

#### Competitive Benchmarks

For context, here is where established players in the network monitoring space sit. NetVantage will not compete on raw numbers with mature projects, but a focused niche with strong metrics can be very attractive for acquisition.

| Project | Stars | Contributors | Docker Pulls | Key Insight |
|---------|-------|-------------|-------------|-------------|
| Prometheus | 56k+ | 900+ | 1B+ | De facto standard, massive ecosystem |
| Grafana | 65k+ | 2,000+ | 1B+ | Visualization layer, not monitoring -- complementary |
| Netdata | 70k+ | 400+ | 500M+ | Agent-first, zero-config ethos (closest to NetVantage philosophy) |
| Zabbix | 5k+ | 200+ | 100M+ | Enterprise-focused, steep learning curve |
| LibreNMS | 4k+ | 400+ | 10M+ | SNMP-focused, welcoming community, PHP stack |
| Uptime Kuma | 60k+ | 300+ | 100M+ | Beautiful UX, monitoring-only, SQLite (closest UX target) |

**NetVantage positioning:** Not competing on scale but on breadth (5-in-1: discovery + monitoring + remote access + credentials + IoT) and ease of use. A project with 1,000 stars, 10 contributors, 100 active installs, and 5 paying customers is a credible acquisition candidate if the code quality, documentation, and IP chain are clean.

---

## System & Network Requirements

### Minimum Hardware

| Tier | Devices | CPU | RAM | Disk | Notes |
|------|---------|-----|-----|------|-------|
| **Micro** | < 25 | 1 core (ARM64 or x64) | 512 MB | 4 GB | Raspberry Pi 4 (4 GB+), Pi 5, Docker on NAS. Uses `micro` performance profile. |
| **Small** | 25–100 | 1 vCPU | 1 GB | 10 GB | Intel N100 mini PC, refurb enterprise micro (Dell OptiPlex, HP EliteDesk), small Proxmox VM |
| **Medium** | 100–500 | 2 vCPU | 2 GB | 25 GB | Dedicated VM or container, small business single site |
| **Large** | 500–1,000 | 4 vCPU | 4 GB | 50 GB | MSP, multi-site |
| **Enterprise** | 1,000+ | 4+ vCPU | 8+ GB | 100+ GB | Requires PostgreSQL + TimescaleDB (Phase 2) |

Disk estimates assume default data retention policies. SNMP polling and high-frequency metrics increase storage requirements. The server auto-selects a performance profile based on detected hardware (see Adaptive Performance Profiles).

### Server Resource Footprint

Target memory consumption for the Go server process (excluding OS and other services):

| Component | Memory |
|-----------|--------|
| Go runtime + core application | 15–25 MB |
| HTTP server + embedded web assets | 10–20 MB |
| SQLite engine (WAL mode) | 5–10 MB |
| Discovery engines (ARP, mDNS, SSDP, SNMP) | 10–30 MB |
| MQTT client + message buffer | 5–15 MB |
| Per monitored device (state + metrics buffer) | 0.5–2 MB each |
| **Estimated total: 50 devices** | **~100–200 MB** |
| **Estimated total: 200 devices** | **~200–500 MB** |
| **Estimated total: 500 devices** | **~500 MB–1.2 GB** |

These estimates guide the performance profile auto-selection and prerequisite checks. Actual usage depends on enabled modules, scan frequency, and metrics retention settings.

### Typical Deployment Scenarios

| Scenario | Hardware | Performance Profile | Expected Device Count |
|----------|----------|--------------------|-----------------------|
| Raspberry Pi 5 (8 GB) dedicated | ARM64, 4 cores, 8 GB RAM, NVMe HAT | `small` (auto) or `medium` (override) | 20–75 |
| Docker on Synology DS920+ | x64, 4 cores, 4–8 GB shared | `micro` or `small` (depends on container limits) | 15–50 |
| Intel N100 mini PC (bare metal) | x64, 4 cores, 16 GB RAM | `large` (auto) | 50–300 |
| Proxmox VM (2 vCPU, 2 GB) | x64, 2 cores, 2 GB RAM | `medium` (auto) | 50–200 |
| Refurb Dell OptiPlex Micro (16 GB) | x64, 6 cores, 16 GB RAM | `large` (auto) | 100–500 |
| Cloud VPS (2 vCPU, 2 GB) + Tailscale | x64, 2 cores, 2 GB RAM | `medium` (auto) | 50–200 (via Tailscale + Scout agents) |
| Rack server VM (4 vCPU, 8 GB) | x64, 4 cores, 8 GB RAM | `large` (auto) | 500–1,000 |

### Supported Server Platforms

| Platform | Architecture | Phase | Notes |
|----------|-------------|-------|-------|
| Linux (Debian/Ubuntu, RHEL/Fedora, Alpine) | x64, ARM64 | 1 | Primary target |
| Windows Server 2019+ / Windows 10+ | x64 | 1 | Native binary |
| Docker | x64, ARM64 | 1 | Official images, multi-arch manifest |
| macOS | ARM64 (Apple Silicon) | 2 | Development/testing use |

**Validated deployment targets** (tested in CI or community-verified):

| Target | Example Hardware | Docker? | Native? | Notes |
|--------|-----------------|---------|---------|-------|
| Raspberry Pi 4/5 | ARM64, 4–16 GB | Yes | Yes | Micro/small profile. NVMe via HAT recommended for Pi 5. |
| Intel N100 mini PCs | x64, 8–16 GB, 6W TDP | Yes | Yes | Ideal homelab platform. Beelink, MinisForum, CWWK, etc. |
| Refurb enterprise micro PCs | x64, 8–64 GB | Yes | Yes | Dell OptiPlex Micro, HP EliteDesk Mini, Lenovo ThinkCentre Tiny |
| Synology NAS (DS920+, DS1522+) | x64, 4–32 GB | Yes (Container Manager) | No | Docker container on existing NAS. Zero additional hardware. |
| QNAP NAS (TS-464, TS-873A) | x64, 8–16 GB | Yes (Container Station) | No | Docker container on existing NAS. |
| Proxmox VE (VM or LXC) | x64 or ARM64 | Yes (inside VM/LXC) | Yes (inside VM/LXC) | Common homelab hypervisor. LXC is more resource-efficient than full VM. |
| Unraid | x64 | Yes (Community Apps) | No | Docker container alongside media/NAS workloads. |
| TrueNAS SCALE | x64 | Yes | No | Docker container on ZFS-backed storage. |
| Cloud VPS (DigitalOcean, Linode, Hetzner, Oracle Cloud) | x64 or ARM64 | Yes | Yes | Remote monitoring via Tailscale or Scout agents. Oracle Cloud free tier (ARM64, 24 GB) is popular. |

### Port & Protocol Matrix

| Port | Protocol | Direction | Component | Purpose | Required |
|------|----------|-----------|-----------|---------|----------|
| 8080 | TCP/HTTP(S) | Inbound | Server | Web UI + REST API | Yes |
| 9090 | TCP/gRPC | Inbound | Server | Scout agent communication (mTLS) | If agents used |
| 4822 | TCP | Internal | Guacamole | RDP/VNC gateway | If Gateway module enabled |
| 161 | UDP/SNMP | Outbound | Server | SNMP polling | If SNMP scanning enabled |
| 162 | UDP/SNMP | Inbound | Server | SNMP traps | If SNMP traps enabled |
| -- | ICMP | Outbound | Server | Ping sweep | If ICMP scanning enabled |
| 5353 | UDP/mDNS | Outbound | Server | mDNS discovery | If mDNS scanning enabled |
| 1900 | UDP/SSDP | Outbound | Server | UPnP/SSDP discovery | If UPnP scanning enabled |
| 1883/8883 | TCP/MQTT | Outbound | Server | MQTT broker communication | If MQTT enabled |

### Reverse Proxy Deployment

NetVantage supports operation behind a reverse proxy (nginx, Traefik, Caddy). Requirements:
- WebSocket upgrade support for real-time dashboard updates (`/ws/` path)
- gRPC passthrough or gRPC-Web translation for Scout agent communication on port 9090
- `X-Forwarded-For`, `X-Forwarded-Proto` headers for accurate client IP logging
- Configurable `--base-path` flag for non-root deployments (e.g., `/netvantage/`)

### Network Considerations

- **IPv6:** Phase 1 is IPv4-only. IPv6 scanning and agent communication targeted for Phase 2.
- **Time synchronization:** NTP is strongly recommended. mTLS certificate validation and metric accuracy depend on synchronized clocks. Server logs a warning at startup if clock skew > 5 seconds from an NTP check.
- **DNS:** Server needs DNS resolution for hostname lookups during discovery. Configurable DNS server override for environments with split DNS.
- **Tailscale:** When the Tailscale plugin is enabled, the server uses the Tailscale REST API (outbound HTTPS to `api.tailscale.com`) for device discovery. No additional ports required. Scout agents on Tailscale-connected devices can reach the server via Tailscale IPs (100.x.y.z), eliminating the need for port forwarding or public IP exposure.

---

## Operations & Maintenance

### Backup & Restore

#### What to Back Up

| Component | Location | Method |
|-----------|----------|--------|
| Database | `data/netvantage.db` | SQLite online backup API (safe during operation) |
| Configuration | `config.yaml` + env vars | File copy |
| TLS certificates | `data/certs/` | File copy (CA key, server cert, agent certs) |
| OUI database | Embedded in binary | Not needed (re-embedded on upgrade) |
| Vault master key | Not on disk (derived from passphrase) | User must retain passphrase |

#### Backup Commands

```bash
netvantage backup --output /path/to/backup.tar.gz    # Full backup (DB + config + certs)
netvantage restore --input /path/to/backup.tar.gz     # Restore to current data dir
netvantage backup --db-only --output /path/to/db.bak  # Database-only backup
```

- Online backup: safe to run while server is operating (uses SQLite backup API)
- Restore to different host: supported (for disaster recovery / migration)
- Automated backups: configurable schedule in `config.yaml` with retention count

#### Backup Configuration

```yaml
backup:
  enabled: false
  schedule: "0 2 * * *"      # Cron expression (daily at 2 AM)
  retention_count: 7          # Keep last N backups
  output_dir: "./data/backups"
```

### Data Retention

Configurable per data type with automated purge. Defaults balance storage with useful history.

| Data Type | Default Retention | Configurable | Purge Method |
|-----------|------------------|--------------|--------------|
| Raw device metrics | 7 days | Yes | Automated daily purge |
| Scan results | 30 days | Yes | Automated daily purge |
| Alerts / events | 180 days | Yes | Automated daily purge |
| Audit logs | 1 year | Yes | Automated daily purge |
| Agent check-in records | 7 days | Yes | Automated daily purge |
| Aggregated metrics (Phase 2) | 1 year | Yes | TimescaleDB retention policy |
| Device records | Never (manual delete) | No | User-initiated |

Configuration:

```yaml
retention:
  metrics_days: 7
  scans_days: 30
  alerts_days: 180
  audit_days: 365
  checkins_days: 7
  purge_schedule: "0 3 * * *"  # Daily at 3 AM
```

### Database Maintenance

- **SQLite WAL checkpointing:** Automatic on server shutdown; configurable periodic checkpoint during operation
- **SQLite VACUUM:** Manual via CLI command `netvantage db vacuum`; not automatic (can be slow on large databases)
- **Database size monitoring:** Exposed as Prometheus metric `netvantage_db_size_bytes`

### Upgrade Strategy

#### Server Upgrades

- Replace binary + restart. Database schema migrations run automatically on startup.
- Migrations are forward-only (no automatic rollback). Take a backup before upgrading.
- Server logs applied migrations at startup for auditability.
- Upgrade path: any version within the same major version can upgrade directly to the latest. Major version upgrades may require intermediate steps (documented in release notes).

#### Agent-Server Version Compatibility

See **Scout Agent Specification > Agent-Server Version Compatibility** for the full compatibility table and version negotiation protocol.

**Summary rule:** Agents must be the same or older proto version as the server (server supports current and N-1). Always upgrade the server first, then agents. Incompatible agents are rejected at check-in with an explicit upgrade message.

### Self-Monitoring

The server monitors its own health and exposes metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `netvantage_db_size_bytes` | Gauge | Database file size |
| `netvantage_db_query_queue_depth` | Gauge | Pending database queries |
| `netvantage_event_bus_queue_depth` | Gauge | Pending async events |
| `netvantage_goroutine_count` | Gauge | Active goroutines |
| `netvantage_disk_free_bytes` | Gauge | Free disk space on data directory |
| `netvantage_uptime_seconds` | Gauge | Server uptime |

Self-monitoring alerts (built-in, always active):
- Disk space < 10% free on data directory
- Database size approaching configured limit
- Event bus queue depth sustained > 1,000

---

## Release & Distribution

### CI/CD Pipeline (GitHub Actions)

| Trigger | Workflow | Steps |
|---------|----------|-------|
| Every PR | `ci.yml` | Lint (golangci-lint), test (race detector), build snapshot, license check |
| Push to `main` | `ci.yml` | Same as PR + integration tests |
| Tag `v*` | `release.yml` | Full release: GoReleaser, Docker build+push, SBOM, signing, changelog |

### Build Tooling

- **GoReleaser:** Cross-platform binary builds from tagged releases
- **Targets:** `linux/amd64`, `linux/arm64`, `windows/amd64`, `darwin/arm64`
- **Snapshot builds:** GoReleaser `--snapshot` on PRs for build verification

### Release Artifacts

| Artifact | Format | Signed | Description |
|----------|--------|--------|-------------|
| Server binaries | tar.gz / zip | Cosign | Per-platform server binaries |
| Agent binaries | tar.gz / zip | Cosign | Per-platform Scout binaries |
| Docker images | OCI | Cosign | Multi-arch manifest, GitHub Container Registry |
| Checksums | SHA256 | Cosign | `checksums.txt` with detached signature |
| SBOM | SPDX JSON | Cosign | Syft-generated software bill of materials |
| Changelog | Markdown | -- | Auto-generated from conventional commits |
| SLSA Provenance | JSON (intoto) | -- | Build provenance attestation (Phase 2) |

### Supply Chain Security

- **Binary signing:** Cosign keyless signing (Sigstore) for all release binaries and Docker images
- **SBOM:** Generated by Syft at build time, attached to GitHub Release and Docker image
- **Vulnerability scanning:** `govulncheck` for Go dependencies, Trivy for Docker images, run in CI on every PR
- **Dependency audit:** `go-licenses` checks for incompatible licenses on every PR
- **Reproducible builds:** Go's deterministic compilation with pinned toolchain version

### Version Management

All version strings follow **Semantic Versioning 2.0.0** (`MAJOR.MINOR.PATCH`) with optional pre-release suffixes. This section is the single source of truth for how versioning works across all NetVantage components. Component-specific enforcement rules are documented inline in the relevant sections (Plugin Lifecycle, Agent-Server Compatibility, REST API Standards, gRPC Services, Configuration).

#### Component Version Inventory

| Component | Version Format | Where Defined | How Consumed |
|-----------|---------------|---------------|--------------|
| Server binary | SemVer (`1.3.2`) | Build-time ldflags | `--version` flag, `/api/v1/health`, `X-NetVantage-Version` header, logs |
| Scout agent binary | SemVer (`1.3.2`) | Build-time ldflags | `--version` flag, `CheckInRequest.agent_version`, logs |
| Plugin API | Integer (`1`, `2`, ...) | `PluginAPIVersionCurrent` constant | `PluginInfo.APIVersion`, registry validation at startup |
| Individual plugins | SemVer (`0.4.1`) | Build-time ldflags (per module) | `PluginInfo.Version`, `/api/v1/plugins` response |
| REST API | Integer in URL path (`v1`) | URL prefix `/api/v1/` | Client requests, `Sunset`/`Deprecation` headers on deprecated versions |
| gRPC API | Integer (`1`, `2`, ...) | Proto package (`netvantage.v1`) | `CheckInRequest.proto_version`, `buf breaking` CI check |
| Database schema | Integer per plugin (`1`, `2`, ...) | `_migrations` table | Automatic forward-only migration at startup |
| Configuration format | Integer (`1`, `2`, ...) | `config_version` YAML field | Startup validation, `netvantage config migrate` CLI |
| Plugin SDK packages | SemVer via Go module | `pkg/plugin/`, `pkg/roles/`, `pkg/models/` | Go module versioning (same repo, tagged independently when needed) |

#### SemVer Rules for Server and Agent

- **MAJOR** bump: Breaking changes to REST API contract, gRPC protocol, Plugin API (dropping an old `APIVersion`), or configuration format (incrementing `config_version`). Users must take action.
- **MINOR** bump: New features, new API endpoints, new optional config keys, new optional plugin interfaces. Backward compatible. No user action required.
- **PATCH** bump: Bug fixes, security patches, performance improvements. No new features. No user action required.
- **Pre-release:** `v1.0.0-alpha.1`, `v1.0.0-beta.1`, `v1.0.0-rc.1`. Pre-release versions have no stability guarantees and are used for testing only.
- **Changelog:** Auto-generated from conventional commit messages (`feat:`, `fix:`, `refactor:`, etc.) by GoReleaser.

#### Build-Time Version Injection

Version information is injected at compile time via Go `ldflags`. No version string is hardcoded in source code.

**Version variables** (in `internal/version/version.go`):

```go
package version

// Set at build time via ldflags. Do not edit.
var (
    Version   = "dev"       // SemVer string, e.g., "1.3.2"
    GitCommit = "unknown"   // Short SHA of the build commit
    BuildDate = "unknown"   // RFC 3339 build timestamp
    GoVersion = "unknown"   // Go toolchain version
)
```

**Makefile pattern:**

```makefile
VERSION   ?= $(shell git describe --tags --always --dirty)
COMMIT    ?= $(shell git rev-parse --short HEAD)
DATE      ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOVERSION ?= $(shell go version | awk '{print $$3}')

VERSION_PKG = github.com/HerbHall/netvantage/internal/version
LDFLAGS = -ldflags "-s -w \
  -X $(VERSION_PKG).Version=$(VERSION) \
  -X $(VERSION_PKG).GitCommit=$(COMMIT) \
  -X $(VERSION_PKG).BuildDate=$(DATE) \
  -X $(VERSION_PKG).GoVersion=$(GOVERSION)"
```

**`--version` flag output** (both server and agent):

```
netvantage version 1.3.2 (commit: a1b2c3d, built: 2025-07-15T14:30:00Z, go: go1.25.6)
```

**Plugin versions:** Each built-in plugin's `Version()` method returns `version.Version` (the server version), since built-in plugins ship with the server binary. Third-party plugins compiled separately inject their own version via their own build's ldflags.

#### Plugin SDK Versioning

The Plugin SDK (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`) is Apache 2.0 licensed and lives in the same Git repository as the BSL-licensed core. It versions as follows:

- **Go module path:** `github.com/HerbHall/netvantage` (same module). SDK packages are importable as `github.com/HerbHall/netvantage/pkg/plugin`, etc.
- **Version coupling (Phase 1-2):** During early development, SDK packages version with the server. A server release `v1.3.2` means `pkg/plugin/` is also at `v1.3.2`. This is the standard Go monorepo pattern.
- **Independent versioning (Phase 3+):** If the SDK needs to release independently (e.g., SDK bugfix without server release), split to a Go sub-module (`pkg/plugin/go.mod`). This is deferred until there is actual demand from third-party plugin developers.
- **API stability promise:** Within a major version, the SDK's exported types and interfaces are backward compatible. New optional interfaces are additive. Removing or changing existing interfaces is a major version bump.

#### Version Compatibility Matrix

Single reference table for what must be compatible with what.

| When this changes... | ...check compatibility with | Rule |
|----------------------|-----------------------------|------|
| Server major version | REST API clients | REST API version may increment. Deprecated API removed after 6 months / 2 minors. |
| Server major version | Scout agents | `proto_version` may increment. Agents at N-1 still accepted. Agents at N-2 or older rejected. |
| Server major version | Plugins | `PluginAPIVersionMin` may increment (drop old API). Plugins must target `>= Min`. |
| Server major version | Config files | `config_version` may increment. Old configs get migration warning + CLI migration tool. |
| Server major version | Database schema | Forward-only migrations run automatically. Backup before upgrade. No rollback. |
| Server minor version | Everything above | Backward compatible. No breaking changes. New features are additive. |
| Plugin API version | Third-party plugins | Plugin targets a single `APIVersion` integer. Server accepts `[Min, Current]` range. |
| gRPC proto version | Scout agents | Integer-based negotiation at check-in. Server supports N and N-1. |
| REST API version | Dashboard, external clients | URL path changes (`/api/v2/`). Old path returns `Sunset` header for 6 months. |
| Plugin's own version | Nothing external | Plugin version is informational (displayed in UI, logged). No compatibility enforcement between plugins. |
| SDK package changes | Third-party plugin source code | Standard Go module compatibility. Additive within major. |

#### Version Mismatch Error Strategy

All version mismatches produce **explicit, actionable error messages**. No silent failures.

| Mismatch | Detection Point | Behavior |
|----------|----------------|----------|
| Plugin API too old | Server startup (registry validation) | Server refuses to load plugin. Error logged with versions and upgrade instructions. |
| Plugin API too new | Server startup (registry validation) | Server refuses to load plugin. Error logged with "upgrade server" message. |
| Agent proto too old | Agent check-in (gRPC handler) | Check-in rejected (`VERSION_REJECTED`). Agent logs upgrade message. Server logs event. |
| Agent proto too new | Agent check-in (gRPC handler) | Check-in rejected (`VERSION_REJECTED`). Agent logs downgrade/wait message. Server logs event. |
| Config version too new | Server startup (config load) | Server refuses to start. Error with "upgrade server or downgrade config" message. |
| Config version too old | Server startup (config load) | Server starts with warning. Suggests `netvantage config migrate`. |
| REST API version removed | Client HTTP request | HTTP 410 Gone with `Sunset` header and message pointing to new API version. |
| Database schema too new | Server startup (migration check) | Server refuses to start. "Database was created by a newer server version." |

---

## Non-Functional Requirements

The ordering below is intentional. **Stability and security come first** -- before performance, before features, before convenience. A monitoring tool that is itself unstable or insecure is worse than no monitoring tool at all.

### Stability

- The server must run unattended for months without intervention, memory leaks, or degradation.
- Plugin failures must be isolated -- a crashing plugin must never take down the server or other plugins.
- Database corruption must be prevented through proper WAL mode, checkpointing, and backup capabilities.
- All background operations (scan jobs, metrics collection, event processing) must have timeouts and circuit breakers.
- Graceful degradation over hard failure: if a subsystem is unhealthy, the rest of the system continues operating.

### Performance

**Server:**
- Handles 1,000+ devices with < 100ms API response times (large/enterprise profile)
- Base memory footprint < 50 MB with zero devices monitored
- Memory usage scales linearly: < 200 MB at 50 devices, < 500 MB at 200 devices, < 1.2 GB at 500 devices
- Startup time < 5 seconds on micro profile, < 10 seconds on large profile
- Network scan of /24 subnet completes in < 30 seconds
- Dashboard loads in < 2 seconds
- Topology map renders smoothly with 500+ devices (progressive rendering for larger networks)
- On micro profile (RPi 4/5): < 200 MB total memory, < 25% CPU during active scan of 25 devices

**Agent (Scout):**
- Binary size < 15 MB (statically linked)
- CPU usage < 1% idle, < 5% during metric collection
- Memory usage < 20 MB on x64, < 25 MB on ARM64
- Runs on Raspberry Pi 3B+ (1 GB RAM, ARMv8) as the minimum target platform
- Disk usage < 50 MB including binary + data + logs

### Security

#### Transport & Encryption
- All agent communication encrypted (mTLS)
- Credentials encrypted at rest (AES-256-GCM envelope encryption)
- TLS 1.2+ enforced for all external connections (HTTPS, gRPC)

#### Authentication & Access Control
- No default credentials (first-run wizard enforces account creation)
- API authentication required (JWT tokens)
- Password policy: minimum 12 characters, checked against breached password list (HaveIBeenPwned k-anonymity API, optional)
- Account lockout: progressive delay after failed login attempts (5 failures = 15 minute lockout)
- Session management: concurrent session limit per user (configurable, default: 5)
- MFA/TOTP: planned for Phase 2 (TOTP at minimum, WebAuthn stretch goal)

#### Web Security
- CORS properly configured (same-origin in production, configurable for dev)
- CSRF protection: SameSite=Strict cookies + custom `X-Requested-With` header validation
- Security headers served by Go HTTP server:
  - `Content-Security-Policy` (restrictive CSP for the SPA)
  - `X-Frame-Options: DENY`
  - `X-Content-Type-Options: nosniff`
  - `Strict-Transport-Security` (when TLS enabled)
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Permissions-Policy` (disable unnecessary browser APIs)
- Input validation at all API boundaries
- Rate limiting on all endpoints

#### Audit & Compliance
- Credential access audit logging
- Secrets hygiene: credentials must never appear in logs, error messages, or API responses
- OWASP Top 10 awareness in all development
- Vulnerability disclosure process documented in SECURITY.md
- **Compliance alignment:** Designed with SOC 2 Type II control categories in mind (access control, encryption, audit logging, change management). Not claiming certification, but signaling security maturity to evaluators and acquirers.

### Deployment

- Single binary server (Go, embeds web assets and migrations)
- Single binary agent (Go, cross-compiled)
- Docker Compose for full stack (server + Guacamole)
- Configuration via YAML file + environment variables
- Deployment profiles for common use cases

### Reliability

See also: **Stability** (above) for the foundational stability requirements.

- Graceful shutdown on SIGTERM/SIGINT with per-plugin timeout
- Automatic agent reconnection with exponential backoff
- Database migrations via embedded SQL (per-plugin, tracked, forward-only)
- Liveness and readiness health check endpoints
- Plugin graceful degradation (optional plugin failure doesn't crash server)
- SQLite WAL mode for concurrent read/write access
- Automatic WAL checkpointing to prevent unbounded WAL growth

### Observability

- Structured logging via Zap (configurable level and format)
- Prometheus metrics at `/metrics`
- Request tracing via `X-Request-ID` headers
- Per-plugin health status in readiness endpoint
- OpenTelemetry tracing support (Phase 2)

---

## Documentation Requirements

### User-Facing Documentation

| Document | Description | Phase |
|----------|-------------|-------|
| README.md | Quick start, feature overview, screenshots | 1 |
| Installation Guide | Single binary, Docker, Docker Compose | 1 |
| Configuration Reference | All YAML keys, env vars, defaults | 1 |
| User Guide | Dashboard walkthrough, common workflows | 1 |
| Admin Guide | User management, backup/restore, upgrades | 2 |
| API Reference | OpenAPI 3.0 spec, auto-generated | 1 |
| Agent Deployment Guide | Windows, Linux, macOS installation | 1b/2/3 |

### Developer Documentation

| Document | Description | Phase |
|----------|-------------|-------|
| Architecture Overview | System design, plugin system, data flow | 1 |
| Plugin Developer Guide | Creating custom modules, role interfaces, SDK | 2 |
| Contributing Guide (CONTRIBUTING.md) | Development setup, code style, PR process, testing, CLA | 1 |
| Plugin API Changelog | Breaking changes by API version | 2 |
| Example Plugins | Webhook notifier, Prometheus exporter, alternative credential store | 2 |

### Community Health Files (GitHub)

Standard files that GitHub recognizes and surfaces in the repository UI. These establish project professionalism and reduce friction for contributors and evaluators.

| File | Description | Phase |
|------|-------------|-------|
| `CONTRIBUTING.md` | Development setup, PR process, code style, testing expectations, CLA | 1 |
| `SECURITY.md` | Vulnerability disclosure process, supported versions, security contacts | 1 |
| `.github/pull_request_template.md` | PR checklist: description, tests, lint, breaking changes | 1 |
| `.github/ISSUE_TEMPLATE/bug_report.md` | Bug report template (exists) | 1 |
| `.github/ISSUE_TEMPLATE/feature_request.md` | Feature request template (exists) | 1 |
| `.github/FUNDING.yml` | Sponsor button configuration (exists) | 1 |
| `.github/workflows/ci.yml` | CI pipeline: lint, test, build, license check | 1 |
| `.github/workflows/release.yml` | Release pipeline: GoReleaser, Docker, SBOM, signing | 1 |
| `CODE_OF_CONDUCT.md` | Contributor Covenant or similar code of conduct | 1 |
| `SUPPORTERS.md` | Financial supporter recognition (exists) | 1 |
| `LICENSING.md` | Human-readable licensing explanation (exists) | 1 |

### README Structure (Target State)

The README is the project's front door. It must convert a skeptical visitor into someone who tries the software. Target structure for Phase 1:

```
# NetVantage
[badges: CI | Go version | License | Latest Release | Docker Pulls]

One-sentence description + key screenshot/GIF

## Why NetVantage?
- Feature comparison table (vs Zabbix, LibreNMS, Uptime Kuma, Domotz)
- "One tool instead of five" value proposition

## Current Status
- What works today (honest)
- What's in progress
- Link to roadmap

## Quick Start
### Binary
### Docker (one-liner)
### Docker Compose

## Screenshots
Dashboard, topology map, device detail, scan in progress

## Architecture
[existing architecture diagram]

## Modules
[existing module table]

## Development
Build, test, lint commands

## Contributing
Link to CONTRIBUTING.md

## Support the Project
Sponsor links

## License
Clear BSL 1.1 explanation with "free for personal, home-lab, and non-competing production use"
```
