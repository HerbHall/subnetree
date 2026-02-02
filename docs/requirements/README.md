# NetVantage Requirements

This directory contains the complete project requirements, split into per-section files for easier navigation, editing, and context-efficient AI-assisted development.

## Sections

| # | File | Description |
|---|------|-------------|
| 01 | [Product Vision](01-product-vision.md) | Goals, target users, design philosophy |
| 02 | [Architecture Overview](02-architecture-overview.md) | Components, modules, communication patterns |
| 03 | [Technology Stack](03-technology-stack.md) | Libraries, licenses, versions |
| 04 | [Plugin Architecture](04-plugin-architecture.md) | PluginInfo, lifecycle, registry, API version checks |
| 05 | [Event System](05-event-system.md) | Topics, subscribers, async patterns |
| 06 | [Database Layer](06-database-layer.md) | Schema, migrations, repository pattern |
| 07 | [Authentication](07-authentication.md) | JWT, OIDC, password policy, sessions |
| 08 | [Scout Agent](08-scout-agent.md) | Agent protocol, version negotiation, enrollment |
| 09 | [Tailscale Integration](09-tailscale-integration.md) | Tailscale plugin design |
| 10 | [Data Model](10-data-model.md) | Core entities (Device, Agent, Credential, etc.) |
| 11 | [API Design](11-api-design.md) | REST endpoints, standards, versioning policy |
| 12 | [Brand Identity](12-brand-identity.md) | Colors, typography, design tokens |
| 13 | [Dashboard Architecture](13-dashboard-architecture.md) | React patterns, state, routing |
| 14 | [Topology Visualization](14-topology-visualization.md) | Graph rendering, React Flow |
| 15 | [Credential Vault](15-credential-vault.md) | Encryption, key management |
| 16 | [Observability](16-observability.md) | Logging, metrics, tracing |
| 17 | [AI & Analytics](17-ai-analytics.md) | Three-tier AI, Insight plugin, analytics |
| 18 | [Testing Strategy](18-testing-strategy.md) | Test categories, infrastructure, coverage |
| 19 | [Deployment](19-deployment.md) | Docker, profiles, performance scaling, config |
| 20 | [Project Infrastructure](20-project-infrastructure.md) | Doc split, project tracking, ADRs, tooling |
| 21 | [Phased Roadmap](21-phased-roadmap.md) | Phase 0/1/1b/2/3/4 checklists |
| 22 | [Competitive Positioning](22-competitive-positioning.md) | Market gap, competitor analysis |
| 23 | [Commercialization](23-commercialization.md) | Licensing, pricing, community, metrics |
| 24 | [System Requirements](24-system-requirements.md) | Hardware, platforms, ports |
| 25 | [Operations & Maintenance](25-operations-maintenance.md) | Backup, retention, upgrades |
| 26 | [Release & Distribution](26-release-distribution.md) | CI/CD, versioning, version management |
| 27 | [Non-Functional Requirements](27-non-functional-requirements.md) | Stability, performance, security targets |
| 28 | [Documentation Requirements](28-documentation-requirements.md) | Doc structure, README target, community files |

## Contributing

When editing requirements:
- Edit the individual section file, not the monolithic `requirements.md`
- Cross-reference other sections using relative links: `[Plugin Architecture](04-plugin-architecture.md)`
- Keep each file self-contained with `## Section Title` as the top-level heading
- Update this index if adding or renaming sections
