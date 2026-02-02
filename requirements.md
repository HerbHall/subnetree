# NetVantage Requirements

## Product Vision

NetVantage is a modular, source-available network monitoring and management platform that provides unified device discovery, monitoring, remote access, credential management, and IoT awareness in a single self-hosted application.

**Strategic Intent:** Free for personal and home use forever. Built to become a commercial product for business, MSP, and enterprise use. The codebase, documentation, community, and clean IP chain are the product -- designed from day one for acquisition readiness.

**Target Users:** Home lab enthusiasts, prosumers, small business IT administrators, managed service providers (MSPs).

**Core Value Proposition:** No existing source-available tool combines device discovery, monitoring, remote access, credential management, and IoT awareness in a single product. Free for home users, BSL 1.1 licensed core with Apache 2.0 plugin SDK for ecosystem growth.

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
    Name         string   // Unique identifier: "recon", "pulse", "vault", etc.
    Version      string   // Semantic version string
    Description  string   // Human-readable summary
    Dependencies []string // Plugin names that must initialize first
    Required     bool     // If true, server refuses to start without this plugin
    Roles        []string // Roles this plugin fills: "discovery", "credential_store"
    APIVersion   int      // Plugin API version targeted (currently 1)
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
```

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
| `pulse.metrics.collected` | `MetricsBatch` | Pulse | Data Exporters |
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

### Platforms

| Platform | Priority | Method |
|----------|----------|--------|
| Windows x64 | Phase 1b | Native Go binary, Windows service |
| Linux x64 | Phase 2 | Native Go binary, systemd unit |
| Linux ARM64 | Phase 2 | Cross-compiled Go binary |
| macOS | Phase 3 | Native Go binary, launchd plist |
| Android | Deferred | Passive monitoring only (ping, ARP, mDNS) |
| IoT/Embedded | Phase 4 | Lightweight Go binary or MQTT-based |

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

| Agent Version | Server Version | Support Level |
|---------------|---------------|---------------|
| Same major+minor | Same major+minor | Full support |
| Same major, older minor | Current | Supported (server backward-compatible within major) |
| Older major (N-1) | Current | Degraded (basic check-in only, no new features) |
| Newer than server | Any | Not supported (agent must not be newer than server) |

**Rule:** Always upgrade the server first, then agents.

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
- **Versioning:** URL path versioning (`/api/v1/`)
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

## Testing Strategy

### Unit Tests

- **Plugin contract tests:** Table-driven tests verifying every plugin against the interface
- **Handler tests:** `httptest.NewRecorder()` for API endpoint testing
- **Repository tests:** In-memory SQLite (`:memory:`) for database logic
- **Mock strategy:** Interface-based mocking for external dependencies (PingScanner, ARPScanner, SNMPClient, DNSResolver)
- **SNMP fixtures:** Recorded responses stored as JSON in `testdata/`

### Integration Tests

- Build tag: `//go:build integration`
- `testcontainers-go` for PostgreSQL + TimescaleDB
- Full server wire-up via `httptest.Server` for API integration tests
- Expose `Handler()` on Server struct for test injection

### Test Commands

```bash
make test              # Unit tests only (fast)
make test-integration  # Full integration suite with containers
make test-coverage     # Generate coverage report
make lint              # go vet + staticcheck
```

### Coverage Targets

| Package | Target |
|---------|--------|
| `pkg/plugin/` | 90%+ (core contracts) |
| `internal/server/` | 80%+ (HTTP handling) |
| `internal/*/` (modules) | 70%+ (business logic) |
| `cmd/` | 50%+ (wiring) |

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

### Configuration

Every setting has a sensible default. A zero-configuration deployment (just run the binary) works out of the box with all modules enabled, SQLite storage, and automatic network detection. Advanced users can customize every aspect via YAML config, environment variables, CLI flags, or the web UI settings page.

**Configuration priority (highest wins):** CLI flags > environment variables > config file > built-in defaults.

```yaml
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

## Phased Roadmap

### Phase 1: Foundation (Server + Dashboard + Discovery + Topology)

**Goal:** Functional web-based network scanner with topology visualization. Validate architecture. Time to First Value under 10 minutes.

#### Architecture & Infrastructure
- [ ] Redesigned plugin system: `PluginInfo`, `Dependencies`, optional interfaces
- [ ] Config abstraction wrapping Viper
- [ ] Event bus (synchronous default with PublishAsync)
- [ ] Role interfaces in `pkg/roles/`
- [ ] Plugin registry with topological sort, graceful degradation
- [ ] Store interface + SQLite implementation (modernc.org/sqlite, pure Go)
- [ ] Per-plugin database migrations
- [ ] Repository interfaces in `internal/services/`

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
- [ ] Plugin contract tests
- [ ] API endpoint tests (httptest)
- [ ] Repository tests (in-memory SQLite)
- [ ] CI pipeline: GitHub Actions with lint (golangci-lint), test (race detector), build
- [ ] GoReleaser configuration for cross-platform binary builds
- [ ] OpenAPI spec generation (swaggo/swag)

### Phase 1b: Windows Scout Agent

**Goal:** First agent reporting metrics to server.

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

#### Infrastructure
- [ ] PostgreSQL + TimescaleDB support (with hypertables for metrics)
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

### Phase 3: Remote Access + Credential Vault

**Goal:** Browser-based remote access to any device with secure credential management.

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

### Phase 4: Extended Platform

**Goal:** IoT awareness, ecosystem growth, acquisition readiness.

- [ ] MQTT broker integration (Eclipse Paho)
- [ ] Home Assistant API integration
- [ ] Scout: Lightweight IoT agent
- [ ] API: Public REST API with API key authentication
- [ ] RBAC: Custom roles with granular permissions
- [ ] Audit logging (all state-changing operations)
- [ ] Configuration backup for network devices (Oxidized-style)
- [ ] Plugin marketplace: curated index, `nvbuild` integration
- [ ] HashiCorp go-plugin support for process-isolated third-party plugins
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

| Tool | Strengths | Gaps vs NetVantage |
|------|-----------|-------------------|
| **Zabbix** | Powerful templates, distributed monitoring, huge community | Steep learning curve (6+ months), no remote access, no credentials, GPL license, users add Grafana for visualization |
| **LibreNMS** | Excellent auto-discovery, SNMP-focused, welcoming community | PHP/LAMP stack feels dated, no remote access, no credentials, slow with 800+ devices |
| **Checkmk** | Best auto-discovery agent, rule-based config | Edition confusion (free features disappear after trial), learning curve |
| **PRTG** | Best setup experience (<1hr), beautiful maps | Windows-only server, sensor-based pricing shock, no Linux server |
| **MeshCentral** | Free RMM replacement, Intel AMT support | UI looks dated, weak discovery, no monitoring depth, no dashboards |
| **Uptime Kuma** | Best UX in monitoring, beautiful, 50K+ GitHub stars | Monitoring only, no SNMP, no agents, no discovery, SQLite scale limits |
| **Domotz** | Best MSP remote access, TCP tunneling | Proprietary, cloud-dependent, $21/site/month, shallow monitoring |
| **Netbox** | Gold standard IPAM/DCIM, excellent API | Documentation only, no monitoring, no remote access |

### Adoption Formula (From Research)

```
Time to First Value < 10 minutes     (Uptime Kuma, PRTG model)
+ Beautiful by Default               (Uptime Kuma model)
+ Auto-Discovery that Reduces Work   (LibreNMS, Checkmk model)
+ Depth Available When Needed        (Zabbix model, progressive disclosure)
+ Fair Pricing / Truly Free          (Zabbix, LibreNMS model)
+ Active Community                   (all successful tools)
= Mass Adoption
```

### User Segment Priorities

| Segment | Top Need | NetVantage Differentiator |
|---------|----------|--------------------------|
| **Home Lab** | Single pane of glass for all devices + IoT | Discovery + monitoring + topology in one tool |
| **MSP** | Multi-tenant + remote access without VPN | Tenant isolation + Gateway module + low per-site cost |
| **Small Biz IT** | Minimal maintenance + management reports | Setup wizard + sensible defaults + scheduled reports |

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

### Acquisition Readiness Checklist

| Attribute | Requirement |
|-----------|------------|
| **Clean architecture** | Modular plugin system, clear separation of concerns, documented interfaces |
| **Test coverage** | 70%+ across core packages, CI/CD pipeline |
| **Documentation** | User guide, admin guide, plugin developer guide, API reference (OpenAPI) |
| **Community** | Active GitHub discussions, contributor guidelines, plugin ecosystem |
| **Legal** | BSL 1.1 core license, Apache 2.0 SDK, CLA via CLA Assistant, NetVantage™ trademark, clean dependency audit (go-licenses in CI) |
| **Metrics** | GitHub stars, Docker pulls, active installations (opt-in telemetry) |
| **Revenue** | Demonstrable paid tier adoption, even at small scale |

---

## System & Network Requirements

### Minimum Hardware

| Tier | Devices | CPU | RAM | Disk | Notes |
|------|---------|-----|-----|------|-------|
| **Small** | < 100 | 1 vCPU | 1 GB | 10 GB | Home lab, small office |
| **Medium** | 100–500 | 2 vCPU | 2 GB | 25 GB | Small business, single site |
| **Large** | 500–1,000 | 4 vCPU | 4 GB | 50 GB | MSP, multi-site |
| **Enterprise** | 1,000+ | 4+ vCPU | 8+ GB | 100+ GB | Requires PostgreSQL + TimescaleDB (Phase 2) |

Disk estimates assume default data retention policies. SNMP polling and high-frequency metrics increase storage requirements.

### Supported Server Platforms

| Platform | Architecture | Phase | Notes |
|----------|-------------|-------|-------|
| Linux (Debian/Ubuntu, RHEL/Fedora, Alpine) | x64, ARM64 | 1 | Primary target |
| Windows Server 2019+ | x64 | 1 | Native binary |
| Docker | x64, ARM64 | 1 | Official images, multi-arch manifest |
| macOS | ARM64 (Apple Silicon) | 2 | Development/testing use |

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

| Agent Version | Server Version | Compatibility |
|---------------|---------------|---------------|
| v1.x | v1.x | Full compatibility within major version |
| v1.x | v2.x | Backward compatible (server supports old agents for 1 major version) |
| v2.x | v1.x | Not supported (agent cannot be newer than server) |

Rule: **agents must be the same or older major version as the server.** Upgrade the server first, then agents.

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

### Versioning

- **Semantic Versioning** (semver): `MAJOR.MINOR.PATCH`
- **Breaking changes:** Major version bump (plugin API changes, config format changes, database schema requiring data migration)
- **Features:** Minor version bump
- **Bug fixes:** Patch version bump
- **Pre-release:** `v1.0.0-rc.1`, `v1.0.0-beta.1` for testing releases
- **Changelog:** Auto-generated from conventional commit messages (`feat:`, `fix:`, `refactor:`, etc.)

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

- Server handles 1,000+ devices with < 100ms API response times
- Agent CPU usage < 1% idle, < 5% during metric collection
- Agent memory usage < 20MB
- Dashboard loads in < 2 seconds
- Network scan of /24 subnet completes in < 30 seconds
- Topology map renders smoothly with 500+ devices

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
| Contributing Guide | Code style, PR process, CLA | 1 |
| Plugin API Changelog | Breaking changes by API version | 2 |
| Example Plugins | Webhook notifier, Prometheus exporter, alternative credential store | 2 |
