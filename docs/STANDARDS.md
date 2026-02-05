# Standards and Best Practices Commitment

NetVantage is committed to following industry-standard best practices in architecture, coding, security, and operations. This document tracks our adopted standards and any intentional deviations.

## Guiding Principle

> **Follow standards by default. Document deviations explicitly.**

When we deviate from a standard, we document:
1. What standard we're deviating from
2. Why we chose a different path
3. What we do instead
4. When (if ever) we might align with the standard

## Adopted Standards

### Architecture Patterns

| Standard | Status | Reference |
|----------|--------|-----------|
| SOLID Principles | Adopted | [ADR-0006](adr/0006-architecture-pattern-adoption.md) |
| MACH Architecture | Adopted (adapted) | [ADR-0006](adr/0006-architecture-pattern-adoption.md) |
| MOSA Principles | Adopted (strategic) | [ADR-0006](adr/0006-architecture-pattern-adoption.md) |
| Modular Management | Adopted (metrics) | [Modular Management](https://www.modularmanagement.com/blog/all-you-need-to-know-about-modularization) |
| Hexagonal Architecture | Adopted | [02-architecture-overview.md](requirements/02-architecture-overview.md#hexagonal-architecture-mapping) |
| 12-Factor App | Partial | See [Deviations](#12-factor-app) |

### Coding Standards

| Standard | Status | Reference |
|----------|--------|-----------|
| Go Code Review Comments | Adopted | [Go Wiki](https://go.dev/wiki/CodeReviewComments) |
| Effective Go | Adopted | [Effective Go](https://go.dev/doc/effective_go) |
| Go Project Layout | Adopted | [golang-standards/project-layout](https://github.com/golang-standards/project-layout) |
| Uber Go Style Guide | Partial | See [Deviations](#uber-go-style-guide) |

### API Standards

| Standard | Status | Reference |
|----------|--------|-----------|
| REST API Design | Adopted | [11-api-design.md](requirements/11-api-design.md) |
| OpenAPI 3.x | Planned (Phase 2) | API documentation |
| gRPC Best Practices | Adopted | [gRPC docs](https://grpc.io/docs/guides/) |
| Semantic Versioning 2.0.0 | Adopted | [26-release-distribution.md](requirements/26-release-distribution.md) |
| JSON:API | Not adopted | See [Deviations](#jsonapi) |

### Security Standards

| Standard | Status | Reference |
|----------|--------|-----------|
| OWASP Top 10 | Adopted | Security review checklist |
| OWASP ASVS | Partial | Authentication/authorization |
| CWE/SANS Top 25 | Adopted | Code review checklist |
| NIST Password Guidelines | Adopted | [07-authentication.md](requirements/07-authentication.md) |

### Protocol Standards

| Standard | Status | Reference |
|----------|--------|-----------|
| HTTP/1.1, HTTP/2 | Adopted | Server communication |
| gRPC over HTTP/2 | Adopted | Agent-server protocol |
| mTLS | Adopted | Agent authentication |
| JWT (RFC 7519) | Adopted | API authentication |
| SNMP v2c/v3 | Adopted | Device discovery |

### Documentation Standards

| Standard | Status | Reference |
|----------|--------|-----------|
| ADR (Architecture Decision Records) | Adopted | [docs/adr/](adr/) |
| Conventional Commits | Adopted | Git workflow |
| Keep a Changelog | Planned | CHANGELOG.md |
| SemVer | Adopted | Version management |

### Modularity Framework

We adopt concepts from Modular Management methodology to measure and improve our plugin architecture:

**Interface Categories** (Phase 2)

| Interface Type | Description | Examples |
|----------------|-------------|----------|
| **API Interfaces** | REST/gRPC endpoints exposed by plugins | `HTTPProvider`, `GRPCProvider` |
| **Event Interfaces** | Event bus topics and payload schemas | `recon.device.discovered`, `pulse.alert.triggered` |
| **Config Interfaces** | Shared configuration keys and schemas | Plugin-specific YAML sections |
| **Data Interfaces** | Database schemas and migration contracts | Per-plugin table prefixes |

**Modularity Health Metrics** (Phase 2)

| Metric | Definition | Target |
|--------|------------|--------|
| **Efficiency** | Shared components / total components | Higher is better (less duplication) |
| **Flexibility** | Valid plugin configurations / total plugins | Higher is better (more composability) |
| **Agility** | Plugins changed / new features added | Lower is better (less coupling) |

**Configuration Rules Architecture** (Phase 4)

Plugin compatibility matrix defining which combinations are tested and supported. See [plugin-certification.md](plugin-certification.md) for certification tiers.

---

## Documented Deviations

### 12-Factor App

**Standard:** [12factor.net](https://12factor.net/) - Methodology for building SaaS applications.

**Deviation:** We partially adopt 12-Factor principles.

| Factor | Status | Notes |
|--------|--------|-------|
| I. Codebase | Adopted | Single repo, multiple deploys |
| II. Dependencies | Adopted | Go modules, explicit declaration |
| III. Config | Adopted | Environment + config files via Viper |
| IV. Backing services | Adopted | Database as attached resource |
| V. Build, release, run | Adopted | CI/CD pipeline |
| VI. Processes | **Partial** | Stateless server, but single-process (not horizontally scaled by default) |
| VII. Port binding | Adopted | Self-contained HTTP server |
| VIII. Concurrency | **Partial** | Vertical scaling preferred for home-lab; horizontal optional |
| IX. Disposability | Adopted | Fast startup, graceful shutdown |
| X. Dev/prod parity | Adopted | Same SQLite in dev and prod |
| XI. Logs | Adopted | Structured logging to stdout |
| XII. Admin processes | **Partial** | CLI commands, not one-off dynos |

**Reason:** NetVantage targets single-tenant, home-lab deployments where operational simplicity matters more than horizontal scalability. Full 12-Factor compliance adds complexity without proportional benefit.

---

### Uber Go Style Guide

**Standard:** [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

**Deviation:** We follow most recommendations but deviate on:

| Recommendation | Our Approach | Reason |
|----------------|--------------|--------|
| Use `uber-go/zap` for logging | Adopted | Matches our choice |
| Avoid `init()` | **Partial** | We use `init()` for compile-time interface guards only |
| Prefer `errors.New` over `fmt.Errorf` for static strings | Adopted | Performance |
| Use `go.uber.org/atomic` | Not adopted | Standard `sync/atomic` sufficient |
| Embed for interface satisfaction | Not adopted | Explicit implementation preferred |

**Reason:** Uber's guide is designed for their scale (thousands of engineers). Some recommendations add ceremony without benefit at our scale.

---

### JSON:API

**Standard:** [jsonapi.org](https://jsonapi.org/) - Specification for building JSON APIs.

**Not Adopted.**

**Reason:** JSON:API's envelope structure (`{"data": {...}, "meta": {...}}`) adds verbosity without clear benefit for our use case. Our API uses simple JSON objects with consistent patterns documented in [11-api-design.md](requirements/11-api-design.md).

**Alternative:** We follow REST best practices with:
- Consistent resource naming (`/api/v1/devices`, `/api/v1/devices/{id}`)
- Standard HTTP methods and status codes
- Consistent error response format
- Pagination via `Link` headers and `?limit=&offset=` params

---

### Strict MACH (Distributed Microservices)

**Standard:** MACH architecture typically implies physically distributed microservices.

**Deviation:** NetVantage deploys as a single binary with plugin-based logical separation.

**Reason:**
- Target audience (home-lab, single-tenant) values operational simplicity
- Plugin boundaries maintain SOLID/MACH benefits without network overhead
- Distributed deployment adds latency, complexity, and failure modes
- Can evolve toward physical distribution in Phase 4 if market demands it

**Mitigation:**
- Plugins are lifecycle-independent (separate Init/Start/Stop)
- Plugins communicate via event bus (async) or defined interfaces (sync)
- No shared mutable state between plugins
- Plugin API versioning supports future process isolation

---

### ORM Usage

**Standard:** Many Go projects use ORMs (GORM, Ent, sqlc) for database access.

**Not Adopted.**

**Reason:**
- Raw SQL with thin repository layer provides full control
- Avoids ORM magic that obscures query behavior
- SQLite-specific optimizations are explicit
- Repository pattern provides testability without ORM overhead

**Alternative:**
- `database/sql` with `modernc.org/sqlite` driver
- Repository interfaces in `pkg/plugin/` (ports)
- Concrete implementations in `internal/store/` (adapters)
- Table-driven tests with `testutil.NewStore(t)` for in-memory SQLite

---

### Dependency Injection Frameworks

**Standard:** Many Go projects use DI frameworks (Wire, Fx, dig).

**Not Adopted.**

**Reason:**
- Manual DI in `main()` makes dependencies explicit and visible
- No magic, no code generation, no reflection
- Easy to understand for new contributors
- Plugin registry provides service location for runtime discovery

**Alternative:** Constructor injection with explicit wiring. See [02-architecture-overview.md](requirements/02-architecture-overview.md#manual-dependency-injection).

---

## Adding New Standards

When adopting a new standard or methodology:

1. **Evaluate fit:** Does it solve a real problem for NetVantage?
2. **Document adoption:** Add to the "Adopted Standards" table above
3. **Note deviations:** If we can't fully adopt, add a "Documented Deviations" section
4. **Create ADR:** For significant architectural decisions, create an ADR in `docs/adr/`

## Adding New Deviations

When deviating from an established standard:

1. **Justify clearly:** Explain why the standard doesn't fit
2. **Document alternative:** Describe what we do instead
3. **Consider future alignment:** Note if/when we might adopt the standard
4. **Link references:** Point to relevant ADRs, requirements docs, or code
