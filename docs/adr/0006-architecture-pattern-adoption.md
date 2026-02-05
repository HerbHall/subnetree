# ADR-0006: Architecture Pattern Adoption (MACH + SOLID Foundation)

## Status

Accepted

## Date

2026-02-04

## Context

NetVantage needs a principled architectural foundation that:

- Guides consistent decision-making across the codebase
- Aligns with industry standards and best practices
- Supports long-term maintainability and ecosystem growth
- Enables plugin marketplace and third-party contributions (Phase 2+)
- Provides clear documentation for contributors and acquirers

We evaluated four established architectural patterns:

| Pattern | Focus | Origin |
|---------|-------|--------|
| **SOA** (Service-Oriented Architecture) | Enterprise service integration via ESB | 2000s enterprise IT |
| **MACH** (Microservices, API-first, Cloud-native, Headless) | Composable, cloud-native applications | MACH Alliance (2020) |
| **MOSA** (Modular Open Systems Approach) | Modular acquisition with open standards | U.S. DoD (statutory requirement) |
| **SOLID** | Object-oriented design principles | Robert C. Martin (~2000) |

## Decision

Adopt **MACH + SOLID** as the primary architectural foundation, with **MOSA** principles informing long-term commercialization strategy.

### SOLID as Code-Level Foundation

SOLID principles govern all Go code:

| Principle | NetVantage Implementation |
|-----------|---------------------------|
| **Single Responsibility** | Each module has one purpose (Recon scans, Vault stores credentials) |
| **Open/Closed** | `plugin.Plugin` interface allows extension without core modification |
| **Liskov Substitution** | Contract tests (`plugintest.TestPluginContract`) ensure interchangeability |
| **Interface Segregation** | Optional interfaces (`HTTPProvider`, `EventSubscriber`) not forced on all plugins |
| **Dependency Inversion** | Manual DI in `main()`, abstractions in `pkg/plugin/` |

### MACH as System-Level Architecture

MACH characteristics guide system design:

| Pillar | NetVantage Implementation |
|--------|---------------------------|
| **Microservices** | Plugins as independently lifecycle-managed modules |
| **API-first** | All functionality exposed via REST/gRPC; `HTTPProvider`/`GRPCProvider` interfaces |
| **Cloud-native** | Go binary, Docker support, stateless design, horizontal scaling ready |
| **Headless** | React SPA completely decoupled from server, consumes REST API |

### MOSA for Strategic Governance

MOSA principles apply to SDK governance and plugin ecosystem:

| MOSA Concept | Application |
|--------------|-------------|
| **Modular design** | Plugins are severable -- can be replaced independently |
| **Open standards** | Plugin SDK (Apache 2.0), REST, gRPC, JSON, YAML |
| **Key interfaces** | `plugin.Plugin` as the contract boundary |
| **Certification** | Contract test suites for plugin validation |

### SOA: Limited Adoption

Traditional SOA with ESB is not adopted. NetVantage's plugin pattern is simpler and more appropriate for single-tenant deployment. We retain SOA's principle of stable service contracts through API versioning.

## Consequences

### Positive

- **Clear guidance:** Developers can reference SOLID/MACH when making design decisions
- **Industry alignment:** Established patterns with extensive documentation and community support
- **Acquisition readiness:** MACH and MOSA language is familiar to enterprise evaluators
- **Ecosystem foundation:** MOSA-informed certification enables plugin marketplace (Phase 2+)
- **Documentation coherence:** Architecture overview can reference standard pattern names

### Negative

- **Learning curve:** Contributors unfamiliar with these patterns need orientation
- **Pattern tension:** Strict MACH implies distributed deployment; NetVantage is single-binary
- **Over-engineering risk:** Applying patterns dogmatically can add unnecessary complexity

### Neutral

- **Hybrid model:** We take what fits from each pattern rather than strict adherence to one
- **Documentation overhead:** Deviations from standard patterns must be documented (see STANDARDS.md)

## Alternatives Considered

### Alternative 1: Pure Microservices (Strict MACH)

Deploy each module as a separate service with independent scaling. Rejected because:
- Adds operational complexity inappropriate for single-tenant/home-lab deployments
- Plugin architecture achieves modular benefits without distributed systems overhead
- Can evolve toward this model in future phases if needed

### Alternative 2: Traditional SOA with ESB

Use message broker (RabbitMQ, NATS) as central integration hub. Rejected because:
- No legacy systems to integrate
- Event bus already handles async communication
- ESB adds latency and single point of failure
- Overkill for single-tenant deployment

### Alternative 3: Domain-Driven Design (DDD) Strict

Full DDD with bounded contexts, aggregates, domain events. Rejected because:
- Plugin boundaries already provide context separation
- Full DDD adds vocabulary overhead without proportional benefit
- DDD concepts (like event sourcing) can be adopted selectively later

### Alternative 4: No Explicit Pattern

Let architecture emerge organically. Rejected because:
- Inconsistent decisions across the codebase
- Harder to onboard contributors
- Acquisition evaluators expect documented architectural rationale

## References

### MACH

- [MACH Alliance](https://machalliance.org/)
- [Sitecore: What is MACH Architecture](https://www.sitecore.com/resources/insights/development/what-is-mach-architecture)

### MOSA

- [DoD CTO: MOSA](https://www.cto.mil/sea/mosa/)
- [MOSA Implementation Guidebook (Feb 2025)](https://www.cto.mil/wp-content/uploads/2025/03/MOSA-Implementation-Guidebook-27Feb2025-Cleared.pdf)

### SOLID

- [DigitalOcean: SOLID Principles](https://www.digitalocean.com/community/conceptual-articles/s-o-l-i-d-the-first-five-principles-of-object-oriented-design)
- [Robert C. Martin: Principles of OOD](http://butunclebob.com/ArticleS.UncleBob.PrinciplesOfOod)
