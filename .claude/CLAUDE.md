# NetVantage - Claude Code Project Configuration

## Project Overview

NetVantage is a modular, source-available network monitoring and management platform written in Go. It consists of a server with plugin-based modules and a lightweight agent (Scout) for monitored devices.

**Free for personal/home use forever. Commercial for business use.** Licensed under BSL 1.1 (core) and Apache 2.0 (plugin SDK). Built with acquisition readiness in mind.

## Context Conservation Strategy

**CRITICAL: The requirements document (`requirements.md`) is 3,000+ lines.** Never read it in full. Use targeted reads of specific sections.

### Documentation Map (requirements.md sections by line range)

Use this map to read ONLY the sections relevant to your current task. Line numbers are approximate and shift as edits are made -- use `grep -n "^## Section Name"` to find current positions.

| Section | Content | When to Read |
|---------|---------|-------------|
| Product Vision | Goals, target users, design philosophy | Starting new features, checking alignment |
| Architecture Overview | Components, modules, communication | Understanding system structure |
| Technology Stack | Libraries, licenses, versions | Adding dependencies |
| Plugin Architecture | PluginInfo, lifecycle, registry, API version checks | Plugin development |
| Event System | Topics, subscribers, async patterns | Adding event-driven features |
| Database Layer | Schema, migrations, repository pattern | Database work |
| Authentication | JWT, OIDC, password policy, sessions | Auth features |
| Scout Agent Spec | Agent protocol, version negotiation, enrollment | Agent development |
| Tailscale Integration | Tailscale plugin design | Tailscale features |
| Data Model | Core entities (Device, Agent, Credential, etc.) | Data structure changes |
| API Design | REST endpoints, standards, versioning policy | API development |
| Brand Identity | Colors, typography, design tokens | UI/styling work |
| Dashboard Architecture | React patterns, state, routing | Frontend development |
| Topology Visualization | Graph rendering, React Flow | Topology features |
| Credential Vault Security | Encryption, key management | Vault module |
| Observability | Logging, metrics, tracing | Operational features |
| AI & Analytics Strategy | Three-tier AI, Insight plugin, analytics | AI/analytics features |
| Testing Strategy | Test categories, infrastructure, coverage | Writing tests |
| Deployment | Docker, profiles, performance scaling, config | Deployment/config work |
| Project Infrastructure | Doc split strategy, project tracking, ADRs, tooling research, AI-assisted dev | Infrastructure decisions, tooling setup, Phase 0 work |
| Phased Roadmap | Phase 0/1/1b/2/3/4 checklists, pre-phase tooling research | Planning, checking what's next |
| Competitive Positioning | Market gap, competitor analysis | README, marketing |
| Commercialization | Licensing, pricing, community, metrics | Business decisions |
| System & Network Requirements | Hardware, platforms, ports | Deployment requirements |
| Operations & Maintenance | Backup, retention, upgrades | Ops features |
| Release & Distribution | CI/CD, versioning, version management | Release process |
| Non-Functional Requirements | Stability, performance, security targets | Quality validation |
| Documentation Requirements | Doc structure, README target, community files | Documentation tasks |

### Context Conservation Rules

1. **Use Explore agents for codebase questions.** Never Glob/Grep the full repo directly from the main context.
2. **Read requirements.md by section, not in full.** Use `grep -n "^## "` to find section boundaries, then `Read` with offset+limit.
3. **Delegate research to subagents.** Use Task(subagent_type=Explore) for "where is X?" and Task(subagent_type=Plan) for "how should we build X?".
4. **Use MCP Memory for cross-session knowledge.** Store architectural decisions, user preferences, and recurring patterns in the Memory knowledge graph.
5. **Use the /create-plan skill for multi-step implementations.** It handles context handoffs between phases.
6. **When modifying requirements.md**, read only the target section (offset+limit), make the edit, and verify. Don't re-read the entire file.

### Future: Split Documentation (Phase 0)

Requirements will be split into `docs/requirements/` with per-section files. Until then, use the Documentation Map above. When the split happens, this map will be updated to reference individual files instead of line ranges.

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
- Interfaces in the consumer package
- Table-driven tests
- No ORM -- raw SQL with thin repository layer

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

Plugins are registered at compile time in `cmd/netvantage/main.go`. Plugin API version is validated at registration (see Plugin Architecture section of requirements.md).

## Version Management

- All components use SemVer 2.0.0 (`MAJOR.MINOR.PATCH`)
- Version injected at build time via ldflags (see `internal/version/version.go`)
- Plugin API uses integer versioning (`PluginAPIVersionMin` / `PluginAPIVersionCurrent`)
- Agent-server uses integer `proto_version` for compatibility negotiation
- REST API: path-based (`/api/v1/`), max 2 concurrent versions
- Config: `config_version` integer at YAML root
- See Version Management section of requirements.md for full compatibility matrix

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
