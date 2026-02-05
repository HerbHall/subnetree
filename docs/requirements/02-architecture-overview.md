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

### Go Architecture Conventions

The codebase follows idiomatic Go patterns that govern how packages interact. These are not optional style preferences -- they are structural rules that maintain decoupling and testability.

#### Accept Interfaces, Return Structs

Functions and constructors accept interface parameters and return concrete types. This lets callers decide the abstraction level while keeping return types explicit and inspectable.

```go
// Good: returns concrete *ViperConfig, caller assigns to plugin.Config where needed
func New(v *viper.Viper) *ViperConfig { ... }

// Good: accepts interface, caller passes any implementation
func New(addr string, plugins PluginSource, logger *zap.Logger) *Server { ... }
```

#### Consumer-Side Interfaces

Interfaces are defined where they are consumed, not where they are implemented. Go's implicit interface satisfaction means the implementing type never imports the consumer. This prevents import cycles and keeps coupling directional.

```go
// internal/server/server.go defines only what *it* needs from the registry:
type PluginSource interface {
    AllRoutes() map[string][]plugin.Route
    All() []plugin.Plugin
}

// internal/registry.Registry satisfies PluginSource implicitly -- no import of server package.
```

**Exception:** The `pkg/plugin/` SDK defines interfaces (`Plugin`, `HTTPProvider`, `Config`, `EventBus`) because they form a public contract shared across many packages. These are the "ports" in hexagonal architecture terms.

#### Compile-Time Interface Guards

Every concrete type that implements an interface includes a compile-time guard at the top of the file. This catches missing methods at build time rather than at runtime via type assertions.

```go
var (
    _ plugin.Plugin       = (*Module)(nil)
    _ plugin.HTTPProvider = (*Module)(nil)
)
```

Every module in `internal/` and every adapter (`internal/config/`, `internal/event/`) must have these guards.

#### Thin Interfaces and Composition

Interfaces are kept small -- ideally one or two methods. Larger interfaces are composed from smaller ones, following the `io.Reader` / `io.Writer` pattern.

```go
type Publisher interface {
    Publish(ctx context.Context, event Event) error
}

type Subscriber interface {
    Subscribe(topic string, handler EventHandler) (unsubscribe func())
}

// Composed from the above:
type EventBus interface {
    Publisher
    Subscriber
    PublishAsync(ctx context.Context, event Event)
    SubscribeAll(handler EventHandler) (unsubscribe func())
}
```

Consumers that only need to publish accept `Publisher`, not `EventBus`. This minimizes coupling.

#### Contract-Driven Development

Shared contract test suites verify that any implementation of an interface behaves correctly. These live alongside the interface definition (e.g., `pkg/plugin/plugintest/`) and are called from each implementation's test file.

```go
// In internal/recon/recon_test.go:
func TestContract(t *testing.T) {
    plugintest.TestPluginContract(t, func() plugin.Plugin { return recon.New() })
}
```

Every new `plugin.Plugin` implementation must call `plugintest.TestPluginContract` in its tests.

#### Manual Dependency Injection

Dependencies are wired explicitly in `cmd/netvantage/main.go` using constructor injection. No DI frameworks, no service locators outside the plugin registry's `PluginResolver`.

```go
// main.go: explicit construction, all dependencies visible
cfg := config.New(v)
bus := event.NewBus(logger)
reg := registry.New(logger)
srv := server.New(addr, reg, logger)
```

#### Hexagonal Architecture Mapping

The codebase maps to hexagonal (ports and adapters) architecture:

| Hexagonal Concept | NetVantage Location | Examples |
|-------------------|---------------------|----------|
| **Ports** (interfaces) | `pkg/plugin/` | `Plugin`, `Config`, `EventBus`, `HTTPProvider` |
| **Adapters** (implementations) | `internal/` | `config.ViperConfig`, `event.Bus`, `registry.Registry` |
| **Domain** (business logic) | `internal/{module}/` | `recon.Module`, `pulse.Module`, `vault.Module` |
| **Wiring** (composition root) | `cmd/netvantage/main.go` | Constructor calls, dependency injection |

`pkg/` is public and stable (Apache 2.0 licensed). `internal/` is private and free to change. This boundary is enforced by Go's visibility rules.

### Industry Pattern Alignment

NetVantage's architecture deliberately aligns with established industry patterns. See [ADR-0006](../adr/0006-architecture-pattern-adoption.md) for the full decision record.

#### SOLID Principles (Code Level)

Every Go convention above maps to a SOLID principle:

| Principle | Convention | Rationale |
|-----------|------------|-----------|
| **Single Responsibility** | One module = one role | Recon scans, Vault stores credentials, Pulse monitors -- no overlap |
| **Open/Closed** | Plugin interface | Add capabilities by implementing `plugin.Plugin`, not modifying core |
| **Liskov Substitution** | Contract test suites | Any `plugin.Plugin` implementation is drop-in replaceable |
| **Interface Segregation** | Optional capability interfaces | Plugins implement only what they need (`HTTPProvider`, `EventSubscriber`) |
| **Dependency Inversion** | Manual DI + consumer-side interfaces | High-level modules depend on abstractions in `pkg/plugin/`, not concrete implementations |

#### MACH Alignment (System Level)

NetVantage exhibits all four MACH characteristics:

| MACH Pillar | Implementation | Notes |
|-------------|----------------|-------|
| **Microservices** | Plugin architecture | Each plugin is independently lifecycle-managed with its own Init/Start/Stop |
| **API-first** | `HTTPProvider` / `GRPCProvider` | All functionality exposed via versioned APIs; UI is a client |
| **Cloud-native** | Go binary + Docker | Stateless server design, horizontal scaling ready, containerized deployment |
| **Headless** | React SPA in `web/` | Dashboard is completely decoupled; consumes REST API like any other client |

**Deviation from strict MACH:** We deploy as a single binary, not distributed microservices. This is intentional -- NetVantage targets home-lab and single-tenant deployments where operational simplicity outweighs distributed scaling benefits. The plugin boundaries maintain logical separation without physical distribution overhead.

#### MOSA Principles (Strategic)

MOSA (Modular Open Systems Approach) informs our SDK and ecosystem strategy:

| MOSA Concept | NetVantage Application |
|--------------|------------------------|
| **Modular design** | Plugins are severable -- can be replaced without core changes |
| **Open standards** | REST (OpenAPI), gRPC (protobuf), JSON, YAML -- no proprietary formats |
| **Key interfaces** | `plugin.Plugin` is the contract boundary; versioned with `PluginAPIVersion` |
| **Certification** | Contract tests (`plugintest.TestPluginContract`) validate implementations |

See [STANDARDS.md](../STANDARDS.md) for our commitment to standards and documentation of any deviations.
