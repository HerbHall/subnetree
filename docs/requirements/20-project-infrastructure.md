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
