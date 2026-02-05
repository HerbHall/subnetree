# Plugin Certification Checklist

This checklist defines the requirements for certifying a NetVantage plugin as conformant with our architecture standards. Based on MOSA (Modular Open Systems Approach) principles, certification ensures plugins are interoperable, maintainable, and safe.

## Overview

Plugin certification has three tiers:

| Tier | Name | Requirements | Use Case |
|------|------|--------------|----------|
| **1** | Core Compliant | Contract tests pass | Built-in plugins, internal development |
| **2** | SDK Compliant | Tier 1 + documentation, versioning | Community plugins, open-source contributions |
| **3** | Marketplace Certified | Tier 2 + security review, support commitment | Commercial plugins, official marketplace |

## Tier 1: Core Compliant

Minimum requirements for any plugin to be loaded by NetVantage.

### 1.1 Interface Contract

- [ ] Implements `plugin.Plugin` interface
- [ ] `Info()` returns valid `PluginInfo` with:
  - [ ] Non-empty `Name` (lowercase, alphanumeric + hyphens)
  - [ ] Non-empty `Version` (SemVer 2.0.0 format)
  - [ ] Valid `APIVersion` within supported range (`PluginAPIVersionMin` to `PluginAPIVersionCurrent`)
  - [ ] Declared `Roles` (at least one)
  - [ ] Declared `Dependencies` (may be empty)
- [ ] `Init(ctx, deps)` completes without error when dependencies are satisfied
- [ ] `Start(ctx)` completes without error after successful `Init`
- [ ] `Stop(ctx)` completes gracefully within timeout (default: 30s)

### 1.2 Contract Tests

- [ ] Plugin test file calls `plugintest.TestPluginContract(t, factory)`
- [ ] All contract tests pass: `go test -race ./...`

```go
// Example: internal/myplugin/myplugin_test.go
func TestContract(t *testing.T) {
    plugintest.TestPluginContract(t, func() plugin.Plugin {
        return New()
    })
}
```

### 1.3 Compile-Time Guards

- [ ] File includes compile-time interface guards for all implemented interfaces

```go
var (
    _ plugin.Plugin       = (*Module)(nil)
    _ plugin.HTTPProvider = (*Module)(nil)  // if applicable
)
```

### 1.4 Lifecycle Correctness

- [ ] `Init` is idempotent (safe to call multiple times)
- [ ] `Start` only succeeds after `Init`
- [ ] `Stop` releases all resources (goroutines, file handles, connections)
- [ ] `Stop` is safe to call even if `Start` was never called
- [ ] No goroutine leaks (verified via `goleak` or manual inspection)

### 1.5 Context Handling

- [ ] All long-running operations respect `ctx.Done()`
- [ ] Cancellation propagates correctly through the plugin
- [ ] No blocking operations without timeout

---

## Tier 2: SDK Compliant

Additional requirements for plugins distributed outside the core repository.

### 2.1 Tier 1 Complete

- [ ] All Tier 1 requirements satisfied

### 2.2 Documentation

- [ ] README.md with:
  - [ ] Plugin description and purpose
  - [ ] Installation instructions
  - [ ] Configuration options (all config keys documented)
  - [ ] API endpoints (if `HTTPProvider`)
  - [ ] Events published/subscribed (if `EventSubscriber`)
  - [ ] Example usage
- [ ] CHANGELOG.md following [Keep a Changelog](https://keepachangelog.com/)
- [ ] LICENSE file (must be compatible with BSL 1.1 or Apache 2.0)

### 2.3 Versioning

- [ ] Version follows SemVer 2.0.0
- [ ] Breaking changes increment MAJOR version
- [ ] Plugin API version compatibility documented
- [ ] Minimum NetVantage version documented

### 2.4 Configuration

- [ ] All configuration keys use consistent naming (snake_case)
- [ ] Defaults are sensible (plugin works with zero configuration where possible)
- [ ] Invalid configuration fails fast at `Init` with clear error messages
- [ ] Configuration is validated (implements `plugin.Validator` if complex)

### 2.5 Logging

- [ ] Uses injected `*zap.Logger` from `Dependencies`
- [ ] No direct writes to stdout/stderr
- [ ] Log levels used appropriately:
  - `Debug`: Verbose troubleshooting
  - `Info`: Normal operational events
  - `Warn`: Recoverable issues
  - `Error`: Failures requiring attention
- [ ] Structured fields used (not string interpolation)

```go
// Good
logger.Info("device discovered", zap.String("ip", ip), zap.String("mac", mac))

// Bad
logger.Info(fmt.Sprintf("device discovered: ip=%s mac=%s", ip, mac))
```

### 2.6 Error Handling

- [ ] Errors are wrapped with context: `fmt.Errorf("operation failed: %w", err)`
- [ ] Errors are returned, not logged-and-swallowed
- [ ] No panics in normal operation (only for programmer errors)
- [ ] Sentinel errors defined for expected failure modes

### 2.7 Testing

- [ ] Unit test coverage >= 70%
- [ ] Integration tests for external dependencies (mocked or containerized)
- [ ] Tests pass with `-race` flag
- [ ] No test pollution (tests are independent)

---

## Tier 3: Marketplace Certified

Additional requirements for plugins in the official NetVantage marketplace.

### 3.1 Tier 2 Complete

- [ ] All Tier 2 requirements satisfied

### 3.2 Security Review

- [ ] No known vulnerabilities in dependencies (`govulncheck ./...`)
- [ ] No hardcoded credentials or secrets
- [ ] Input validation on all external data (API inputs, config, network data)
- [ ] SQL injection prevention (parameterized queries only)
- [ ] Command injection prevention (no shell execution with user input)
- [ ] Path traversal prevention (validated file paths)
- [ ] Credentials handled via Vault plugin, not stored internally

### 3.3 API Standards (if HTTPProvider)

- [ ] REST endpoints follow [11-api-design.md](requirements/11-api-design.md)
- [ ] Endpoints are versioned (`/api/v1/plugin-name/...`)
- [ ] OpenAPI spec provided for all endpoints
- [ ] Authentication required for sensitive operations
- [ ] Rate limiting considered for expensive operations

### 3.4 Event Standards (if EventSubscriber)

- [ ] Events use standard topic naming: `plugin.{name}.{action}`
- [ ] Event payloads are documented
- [ ] Events are idempotent (handlers tolerate duplicates)
- [ ] No blocking in event handlers

### 3.5 Performance

- [ ] No unbounded memory growth
- [ ] Benchmarks provided for critical paths
- [ ] Resource usage documented (CPU, memory, network expectations)
- [ ] Graceful degradation under load

### 3.6 Support Commitment

- [ ] Maintainer contact information provided
- [ ] Issue tracker available (GitHub Issues or equivalent)
- [ ] Response time SLA documented (e.g., "critical bugs within 72 hours")
- [ ] Compatibility commitment (supported NetVantage versions)

### 3.7 Legal

- [ ] License compatible with NetVantage distribution
- [ ] CLA signed (if contributing to core repository)
- [ ] No GPL/AGPL/LGPL/SSPL dependencies
- [ ] Third-party licenses documented

---

## Certification Process

### For Built-in Plugins (Tier 1)

1. Implement plugin following checklist
2. Add contract tests
3. Run `make test` and `make lint`
4. Submit PR for review

### For Community Plugins (Tier 2)

1. Complete Tier 1 requirements
2. Create public repository with documentation
3. Submit plugin for listing in community index
4. Maintainer self-certifies compliance

### For Marketplace Plugins (Tier 3)

1. Complete Tier 2 requirements
2. Submit plugin for security review
3. Sign support commitment agreement
4. Plugin added to official marketplace after approval

---

## Automated Verification

The following can be verified automatically:

```bash
# Tier 1 checks
go build ./...                          # Compiles
go test -race ./...                     # Tests pass
golangci-lint run                       # Linting passes

# Tier 2 checks
go test -coverprofile=coverage.out ./... # Coverage check
go tool cover -func=coverage.out | grep total  # >= 70%

# Tier 3 checks
govulncheck ./...                       # No known vulnerabilities
go-licenses check ./...                 # License compliance
```

Future: CI workflow for automated certification badge.

---

## References

- [ADR-0003: Plugin Architecture](adr/0003-plugin-architecture-caddy-model.md)
- [ADR-0006: Architecture Pattern Adoption](adr/0006-architecture-pattern-adoption.md)
- [04-plugin-architecture.md](requirements/04-plugin-architecture.md)
- [STANDARDS.md](STANDARDS.md)
- [MOSA Implementation Guidebook](https://www.cto.mil/wp-content/uploads/2025/03/MOSA-Implementation-Guidebook-27Feb2025-Cleared.pdf)
