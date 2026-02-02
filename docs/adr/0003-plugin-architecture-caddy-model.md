# ADR-0003: Plugin Architecture (Caddy Model)

## Status

Accepted

## Date

2025-01-01

## Context

NetVantage's core value proposition is modularity -- every major capability (discovery, monitoring, remote access, credentials, agent management) should be replaceable and extensible. The architecture must:

- Allow built-in modules to be replaced by third-party alternatives
- Enable the community to build new capabilities without modifying core code
- Keep the core server minimal (HTTP, plugin registry, event bus, database, config)
- Support compile-time plugin registration for built-in modules (Phase 1)
- Have a path to runtime plugin loading for third-party plugins (Phase 4)
- Maintain type safety and avoid the complexity of process-level isolation initially

## Decision

Adopt a Caddy-style compile-time plugin architecture with optional interfaces detected via Go type assertions.

**Core plugin interface:**
```go
type Plugin interface {
    Info() PluginInfo
    Init(ctx context.Context, deps Dependencies) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

**Optional capability interfaces:**
- `HTTPProvider` -- registers REST API routes
- `GRPCProvider` -- registers gRPC services
- `HealthChecker` -- reports health status
- `EventSubscriber` -- subscribes to event bus topics
- `Validator` -- validates configuration
- `Reloadable` -- supports hot config reload
- `AnalyticsProvider` -- provides AI/analytics capabilities (Phase 2+)

**Plugin registration:** Compile-time in `cmd/netvantage/main.go`. The registry performs topological sort based on declared dependencies and validates Plugin API version compatibility.

**Phase 4 evolution:** Add HashiCorp go-plugin for process-isolated third-party plugins communicating over gRPC, while keeping built-in plugins in-process for performance.

## Consequences

### Positive

- Simple, idiomatic Go -- no reflection magic, no dynamic loading complexity
- Type-safe: optional interfaces are checked at compile time
- Zero overhead for in-process plugins (direct function calls)
- Caddy has proven this model works at scale with a large plugin ecosystem
- Progressive capability disclosure -- plugins opt into features by implementing interfaces
- Custom binaries via `nvbuild` (like `xcaddy`) for user-selected plugin sets

### Negative

- Adding or removing a plugin requires recompilation (until Phase 4 go-plugin support)
- All built-in plugins share the same process -- a panic in one plugin can crash the server (mitigated by panic recovery in the registry)
- Plugin API changes require version bumps and compatibility checking
- No hot-swapping of plugins at runtime

### Neutral

- Built-in plugins share the server version (they ship in the same binary)
- Third-party plugins will need independent versioning (Phase 4)
- The `Dependencies` struct provides controlled access to shared services (store, event bus, logger, config)

## Alternatives Considered

### Alternative 1: HashiCorp go-plugin (gRPC Process Isolation)

Full process isolation from day one. Each plugin runs as a separate process communicating over gRPC. Provides crash isolation but adds significant latency, complexity, and resource overhead. Deferred to Phase 4 for third-party plugins where isolation is essential.

### Alternative 2: Shared Object / DLL Loading

Go's `plugin` package loads `.so` files at runtime. Only works on Linux, is fragile across Go versions, and has been effectively abandoned by the Go team. Not viable for cross-platform deployment.

### Alternative 3: WASM Plugin Runtime

Running plugins in a WASM sandbox (e.g., Wazero). Provides sandboxing but WASM's Go support is immature, performance overhead is significant, and the developer experience for plugin authors is poor. Could be revisited in future.

### Alternative 4: Grafana-Style (Compile + go-plugin Hybrid)

Start with both in-process and go-plugin support simultaneously. Adds complexity without clear benefit in Phase 1 when all plugins are built-in. Our approach reaches the same end state but defers go-plugin complexity until third-party plugins actually exist.
