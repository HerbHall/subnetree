# NetVantage - Claude Code Project Configuration

## Project Overview

NetVantage is a modular, source-available network monitoring and management platform written in Go. It consists of a server with plugin-based modules and a lightweight agent (Scout) for monitored devices.

**Free for personal/home use forever. Commercial for business use.** Licensed under BSL 1.1 (core) and Apache 2.0 (plugin SDK). Built with acquisition readiness in mind.

## Context Conservation Strategy

Requirements are split into `docs/requirements/` with per-section files. Read ONLY the file relevant to your current task.

### Documentation Map (docs/requirements/)

| File | Content | When to Read |
|------|---------|-------------|
| [01-product-vision.md](docs/requirements/01-product-vision.md) | Goals, target users, design philosophy | Starting new features, checking alignment |
| [02-architecture-overview.md](docs/requirements/02-architecture-overview.md) | Components, modules, communication | Understanding system structure |
| [03-technology-stack.md](docs/requirements/03-technology-stack.md) | Libraries, licenses, versions | Adding dependencies |
| [04-plugin-architecture.md](docs/requirements/04-plugin-architecture.md) | PluginInfo, lifecycle, registry, API version checks | Plugin development |
| [05-event-system.md](docs/requirements/05-event-system.md) | Topics, subscribers, async patterns | Adding event-driven features |
| [06-database-layer.md](docs/requirements/06-database-layer.md) | Schema, migrations, repository pattern | Database work |
| [07-authentication.md](docs/requirements/07-authentication.md) | JWT, OIDC, password policy, sessions | Auth features |
| [08-scout-agent.md](docs/requirements/08-scout-agent.md) | Agent protocol, version negotiation, enrollment | Agent development |
| [09-tailscale-integration.md](docs/requirements/09-tailscale-integration.md) | Tailscale plugin design | Tailscale features |
| [10-data-model.md](docs/requirements/10-data-model.md) | Core entities (Device, Agent, Credential, etc.) | Data structure changes |
| [11-api-design.md](docs/requirements/11-api-design.md) | REST endpoints, standards, versioning policy | API development |
| [12-brand-identity.md](docs/requirements/12-brand-identity.md) | Colors, typography, design tokens | UI/styling work |
| [13-dashboard-architecture.md](docs/requirements/13-dashboard-architecture.md) | React patterns, state, routing | Frontend development |
| [14-topology-visualization.md](docs/requirements/14-topology-visualization.md) | Graph rendering, React Flow | Topology features |
| [15-credential-vault.md](docs/requirements/15-credential-vault.md) | Encryption, key management | Vault module |
| [16-observability.md](docs/requirements/16-observability.md) | Logging, metrics, tracing | Operational features |
| [17-ai-analytics.md](docs/requirements/17-ai-analytics.md) | Three-tier AI, Insight plugin, analytics | AI/analytics features |
| [18-testing-strategy.md](docs/requirements/18-testing-strategy.md) | Test categories, infrastructure, coverage | Writing tests |
| [19-deployment.md](docs/requirements/19-deployment.md) | Docker, profiles, performance scaling, config | Deployment/config work |
| [20-project-infrastructure.md](docs/requirements/20-project-infrastructure.md) | Doc split, project tracking, ADRs, tooling research | Infrastructure decisions, Phase 0 |
| [21-phased-roadmap.md](docs/requirements/21-phased-roadmap.md) | Phase 0/1/1b/2/3/4 checklists, tooling research | Planning, checking what's next |
| [22-competitive-positioning.md](docs/requirements/22-competitive-positioning.md) | Market gap, competitor analysis | README, marketing |
| [23-commercialization.md](docs/requirements/23-commercialization.md) | Licensing, pricing, community, metrics | Business decisions |
| [24-system-requirements.md](docs/requirements/24-system-requirements.md) | Hardware, platforms, ports | Deployment requirements |
| [25-operations-maintenance.md](docs/requirements/25-operations-maintenance.md) | Backup, retention, upgrades | Ops features |
| [26-release-distribution.md](docs/requirements/26-release-distribution.md) | CI/CD, versioning, version management | Release process |
| [27-non-functional-requirements.md](docs/requirements/27-non-functional-requirements.md) | Stability, performance, security targets | Quality validation |
| [28-documentation-requirements.md](docs/requirements/28-documentation-requirements.md) | Doc structure, README target, community files | Documentation tasks |

### Context Conservation Rules

1. **Use Explore agents for codebase questions.** Never Glob/Grep the full repo directly from the main context.
2. **Read one requirement file at a time.** Each file in `docs/requirements/` is self-contained. Never read multiple requirement files in a single task.
3. **Delegate research to subagents.** Use Task(subagent_type=Explore) for "where is X?" and Task(subagent_type=Plan) for "how should we build X?".
4. **Use MCP Memory for cross-session knowledge.** Store architectural decisions, user preferences, and recurring patterns in the Memory knowledge graph.
5. **Use the /create-plan skill for multi-step implementations.** It handles context handoffs between phases.
6. **When modifying requirements**, edit the specific section file directly. Don't read the full index.

## Guiding Principles

These principles govern every development decision. When in doubt, refer here:

1. **Ease of use first.** No tech degree required. Intuitive for non-technical users, powerful for experts. If it needs a manual to understand, simplify the UI.
2. **Sensible defaults, deep customization.** Ship preconfigured for instant deployment. Every aspect is user-configurable. Defaults get you running; customization makes it yours.
3. **Stability and security are non-negotiable.** Every release must be stable enough for production infrastructure and secure enough to trust with credentials. If a feature compromises either, it does not ship.
4. **Plugin-powered architecture.** Every major feature is a plugin. The core is minimal. Users and developers can replace, extend, or supplement any module.
5. **Progressive disclosure.** Simple by default, advanced on demand. Never overwhelm a first-time user.

## Architecture

- **Server** (`cmd/netvantage/`): Central application with HTTP API, plugin registry
- **Scout** (`cmd/scout/`): Lightweight agent installed on monitored devices
- **Dashboard** (`web/`): React + TypeScript SPA served by the server
- **Modules** (`internal/`): Recon (scanning), Pulse (monitoring), Dispatch (agent mgmt), Vault (credentials), Gateway (remote access)
- **Plugin SDK** (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`): Public interfaces, Apache 2.0 licensed
- **Proto** (`api/proto/v1/`): gRPC service definitions, Apache 2.0 licensed
- **Design System** (`web/src/styles/design-tokens.css`, `web/tailwind.config.ts`): Forest green + earth tone palette

## Build Commands

```bash
make build          # Build everything
make build-server   # Build server only
make build-scout    # Build agent only
make run-server     # Run server
make test           # Unit tests (-race)
make test-integration  # Integration tests (Docker required)
make test-coverage  # Coverage report
make lint           # golangci-lint (go vet + staticcheck + gosec + more)
make proto          # Generate protobuf code
make license-check  # Check dependency licenses
make clean          # Clean build artifacts
```

## Go Conventions

- Module path: `github.com/HerbHall/netvantage`
- Go 1.25+
- Use `internal/` for private packages, `pkg/` for public
- Standard Go project layout
- Structured logging via `go.uber.org/zap`
- Configuration via `github.com/spf13/viper`
- gRPC for agent-server communication
- Database: `modernc.org/sqlite` (pure Go, no CGo)

## Code Style

- Follow standard Go conventions (gofmt, go vet)
- Error handling: return errors, don't panic
- Use context.Context for cancellation/timeouts
- Table-driven tests
- No ORM -- raw SQL with thin repository layer

## Go Architecture Conventions

These patterns are enforced across the codebase. See [02-architecture-overview.md](docs/requirements/02-architecture-overview.md#go-architecture-conventions) for full rationale.

- **Accept interfaces, return structs:** Functions accept interface params, return concrete types. Never return an interface from a constructor.
- **Consumer-side interfaces:** Define interfaces where consumed, not where implemented. Exception: `pkg/plugin/` defines shared contracts (ports).
- **Compile-time interface guards:** Every type implementing an interface must have `var _ Interface = (*Type)(nil)` at the top of the file.
- **Thin interfaces, composed:** Keep interfaces to 1-2 methods. Compose larger interfaces from small ones (e.g., `EventBus = Publisher + Subscriber`).
- **Contract test suites:** Every `plugin.Plugin` implementation must call `plugintest.TestPluginContract` in its tests.
- **Manual DI in main():** No DI frameworks. All wiring is explicit in `cmd/netvantage/main.go`.
- **Hexagonal mapping:** `pkg/plugin/` = ports, `internal/` = adapters, `cmd/` = composition root.

## Plugin Architecture

Each module implements the `plugin.Plugin` interface:
- `Info() PluginInfo` -- metadata, dependencies, roles, APIVersion
- `Init(ctx context.Context, deps Dependencies) error`
- `Start(ctx context.Context) error`
- `Stop(ctx context.Context) error`

Optional interfaces detected via type assertions:
- `HTTPProvider` -- REST API routes
- `GRPCProvider` -- gRPC services
- `HealthChecker` -- health reporting
- `EventSubscriber` -- event bus subscriptions
- `Validator` -- config validation
- `Reloadable` -- hot config reload
- `AnalyticsProvider` -- AI/analytics capabilities (Phase 2+)

Plugins are registered at compile time in `cmd/netvantage/main.go`. Plugin API version is validated at registration (see [04-plugin-architecture.md](docs/requirements/04-plugin-architecture.md)).

## Version Management

- All components use SemVer 2.0.0 (`MAJOR.MINOR.PATCH`)
- Version injected at build time via ldflags (see `internal/version/version.go`)
- Plugin API uses integer versioning (`PluginAPIVersionMin` / `PluginAPIVersionCurrent`)
- Agent-server uses integer `proto_version` for compatibility negotiation
- REST API: path-based (`/api/v1/`), max 2 concurrent versions
- Config: `config_version` integer at YAML root
- See [26-release-distribution.md](docs/requirements/26-release-distribution.md) for full version compatibility matrix

## Licensing

- **Core (BSL 1.1):** `LICENSE` at repo root. Change Date: 4 years, converts to Apache 2.0.
- **Plugin SDK (Apache 2.0):** `pkg/plugin/`, `pkg/roles/`, `pkg/models/`, `api/proto/`
- **Block:** GPL, AGPL, LGPL, SSPL dependencies. Use `make license-check` to verify.
- **CLA required** for all contributions (GitHub Actions workflow).

## Git Conventions

- Conventional commits: `feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `chore:`
- Branch naming: `feature/`, `fix/`, `refactor/`
- Co-author tag: `Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>`
- Branch protection on `main`: PRs required, CLA check must pass

## Useful Claude Code Skills

These installed skills are particularly relevant for NetVantage development:

| Skill / Command | When to Use |
|----------------|-------------|
| `/create-plan` | Before starting any multi-file feature implementation |
| `/run-plan` | Executing a phase from an existing plan |
| `/check-todos` | Resuming work -- see what's outstanding |
| `/add-to-todos` | Capturing context mid-work for future sessions |
| `/whats-next` | Generating handoff docs when context is running low |
| `/debug` | Systematic debugging with hypothesis testing |
| `/ask-me-questions` | Gathering requirements before implementing |
| `/requirements_generator` | Updating requirements documentation |
| `/eisenhower-matrix` | Prioritizing when there are too many tasks |
| `/first-principles` | Architectural decisions that need rigorous reasoning |

## MCP Tools for This Project

| Tool | Use Case |
|------|----------|
| **Context7** | Fetch current docs for Go libraries before using them |
| **Memory** | Store architecture decisions, user preferences, patterns across sessions |
| **Sequential Thinking** | Complex debugging, multi-step architectural reasoning |
| **SQLite** | Local persistent storage for project tracking data |
