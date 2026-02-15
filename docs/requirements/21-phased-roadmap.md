## Phased Roadmap

**Target Audience:** HomeLabbers and small business IT administrators. The roadmap prioritizes features that serve single-subnet home networks (15–200 devices) while maintaining a scalable, acquisition-ready architecture.

**Key Integration Targets:** Home Assistant, UnRAID, Proxmox VE -- the HomeLab community staples that differentiate this project from enterprise-focused tools.

### Phase 0: Pre-Development Infrastructure

**Goal:** Establish project infrastructure, tooling, and processes before writing product code. Everything here is a prerequisite for efficient Phase 1 development.

#### Documentation Split

- [x] Create `docs/requirements/` directory structure
- [x] Split `requirements.md` into per-section files (28 files)
- [x] Create `docs/requirements/README.md` index with section descriptions
- [x] Update `.claude/CLAUDE.md` Documentation Map to reference individual files
- [x] Replace `requirements.md` with redirect to `docs/requirements/README.md`
- [x] Verify all cross-references resolve correctly

#### Architecture Decision Records

- [x] Create `docs/adr/` directory with MADR template
- [x] Write ADR-0001: Split licensing model (BSL 1.1 + Apache 2.0)
- [x] Write ADR-0002: SQLite-first database strategy
- [x] Write ADR-0003: Plugin architecture (Caddy model with optional interfaces)
- [x] Write ADR-0004: Integer-based protocol versioning

#### GitHub Project Setup

- [x] Create GitHub Projects v2 board with Kanban, Roadmap, and Table views
- [x] Define custom fields: Phase, Module, Priority, Effort
- [x] Create milestone for each phase (Phase 0 through Phase 4)
- [x] Apply label taxonomy: type, priority, module, phase, contributor labels
- [x] Seed initial issues from Phase 1 checklist items
- [x] Configure issue templates: bug report, feature request, plugin idea

#### CI/CD Pipeline Scaffolding

- [x] GitHub Actions: Go build matrix (Linux amd64/arm64, Windows amd64, macOS amd64/arm64)
- [x] GitHub Actions: test workflow (unit tests with race detector)
- [x] GitHub Actions: lint workflow (golangci-lint with project config)
- [x] GitHub Actions: license check workflow
- [x] GitHub Actions: CLA check workflow (CLA Assistant or custom)
- [x] Dependabot: configure for Go modules and GitHub Actions
- [x] Pre-commit hooks: gofmt, go vet, license header check (lefthook)

#### Development Environment

- [x] Document development setup in `docs/guides/development-setup.md`
- [x] Makefile: verify all targets work on Windows (MSYS2/Git Bash), Linux, macOS
- [x] `.editorconfig` for consistent formatting across editors
- [x] VS Code recommended extensions list (`.vscode/extensions.json`)
- [x] Go workspace configuration -- not needed (single module)

#### Community Health Files

- [x] CONTRIBUTING.md: fork-and-PR workflow, commit conventions, code review process
- [x] CODE_OF_CONDUCT.md: Contributor Covenant v2.1
- [x] SECURITY.md: vulnerability reporting process
- [x] Pull request template (`.github/pull_request_template.md`)
- [x] Issue templates: bug, feature, plugin idea (`.github/ISSUE_TEMPLATE/`)

#### Metrics Baseline

- [x] Register repository on Go Report Card
- [x] Configure Codecov for coverage tracking
- [x] Document badge URLs for README (CI, coverage, Go Report Card, license, release)

### Phase 1: Foundation (Server + Dashboard + Discovery + Topology)

**Status:** v0.2.0 shipped 2026-02-08. All core modules implemented: Recon, Pulse, Insight, LLM, Vault, Gateway. Dashboard polish and Tailscale guides complete. Post-release: Docs module (#132), Device CRUD + inventory (#162, #163), modular themes (#158), code splitting (#174), E2E tests (#175).

**Goal:** Functional web-based network scanner with topology visualization. Validate architecture. Time to First Value under 10 minutes.

#### Pre-Phase Tooling Research

- [x] Evaluate and configure golangci-lint (15+ linters, project-specific `.golangci.yml`)
- [x] Establish test framework patterns: table-driven tests, testify assertions, testcontainers for integration
- [x] Set up Codecov integration for coverage tracking in CI
- [x] Register repository on Go Report Card
- [x] Configure GitHub Actions workflows: build, test, lint, license-check
- [x] Configure Dependabot for Go modules and GitHub Actions
- [x] Set up pre-commit hooks: gofmt, go vet, license header check (lefthook)
- [x] Evaluate and document React + TypeScript toolchain for dashboard (Vite, ESLint, Prettier)

#### Architecture & Infrastructure

- [x] Redesigned plugin system: `PluginInfo`, `Dependencies`, optional interfaces
- [x] Config abstraction wrapping Viper
- [x] Event bus (synchronous default with PublishAsync for slow consumers like analytics)
- [x] Role interfaces in `pkg/roles/` (including `AnalyticsProvider` interface -- definition only, no implementation)
- [x] Plugin registry with topological sort, graceful degradation
- [x] Store interface + SQLite implementation (modernc.org/sqlite, pure Go)
- [x] Per-plugin database migrations (reserve `analytics_` table prefix for Phase 2 Insight plugin)
- [x] Repository interfaces in `internal/services/`
- [x] Metrics collection format: uniform `(timestamp, device_id, metric_name, value, tags)` for analytics consumption (Pulse publishes MetricPoints consumed by Insight)

#### Server & API

- [x] HTTP server with core routes
- [x] RFC 7807 error responses
- [x] Request ID middleware
- [x] Structured request logging middleware
- [x] Prometheus metrics at `/metrics`
- [x] Liveness (`/healthz`) and readiness (`/readyz`) endpoints
- [x] Per-IP rate limiting
- [x] Configuration via YAML + environment variables
- [x] Configurable Zap logger factory

#### Authentication

- [x] Local auth with bcrypt password hashing
- [x] JWT access/refresh token flow
- [x] First-run setup endpoint (create admin when no users exist)
- [ ] OIDC/OAuth2 optional configuration (schema ready -- defer provider to Phase 2)

#### Recon Module

- [x] ICMP ping sweep
- [x] ARP scanning
- [x] OUI manufacturer lookup (embedded database)
- [ ] LLDP/CDP neighbor discovery for topology (deferred to Phase 2)
- [x] Device persistence in SQLite
- [x] Publishes `recon.device.discovered` events

#### Dashboard

- [x] React + Vite + TypeScript + shadcn/ui + TanStack Query + Zustand
- [x] First-run setup wizard
- [x] Dashboard overview page (device counts, status summary)
- [x] Device list with search, filter, sort, pagination
- [x] Device detail page
- [x] Network topology visualization (auto-generated from LLDP/CDP/ARP)
- [x] Scan trigger with real-time progress (WebSocket)
- [x] Dark mode support
- [x] Settings page (server config, user profile)
- [x] About page with version info, license, and Community Supporters section
- [x] Route-level code splitting with React.lazy (747KB -> 409KB main bundle)
- [x] Modular theme layer system with 19 built-in themes (#158)

#### Documentation

- [x] Tailscale deployment guide: running SubNetree + Scout over Tailscale
- [x] Tailscale Funnel/Serve guide: exposing dashboard without port forwarding

#### Operations

- [x] Backup/restore CLI commands (`subnetree backup`, `subnetree restore`)
- [x] Data retention configuration with automated purge job (Pulse and Gateway maintenance loops)
- [x] Security headers middleware (CSP, X-Frame-Options, HSTS, etc.)
- [x] Account lockout after failed login attempts
- [x] SECURITY.md with vulnerability disclosure process

#### Testing & Quality

- [x] Test infrastructure: `internal/testutil/` with mocks, fixtures, helpers, mock clock
- [ ] Test infrastructure: `testdata/` directory with SNMP fixtures, test configs, migration snapshots
- [x] Plugin contract tests: table-driven tests for `Plugin` interface and all optional interfaces
- [x] Plugin isolation tests: panic recovery in Init, Start, Stop, and HTTP handlers
- [x] Plugin lifecycle tests: full Init → Start → Stop cycle, dependency ordering, cascade disable
- [x] Plugin API version validation tests: too old, too new, exact match, backward-compatible range
- [x] API endpoint tests: `httptest.NewRecorder()` for all routes (status codes, content types, RFC 7807 errors)
- [x] Security middleware tests: auth enforcement, security headers, CORS, CSRF, rate limiting (429)
- [x] Input validation tests: malformed JSON, oversized payloads, SQL injection, XSS, path traversal
- [x] Secrets hygiene tests: verify credentials never appear in log output or error responses
- [x] Repository tests: in-memory SQLite CRUD, edge cases, transactions, constraint violations
- [x] Database migration tests: fresh install, sequential upgrade, per-plugin isolation, idempotent check
- [x] Configuration tests: defaults, env overrides, YAML overrides, invalid values, `config_version` validation
- [ ] Version compatibility tests: Plugin API, agent proto, config version, database schema version
- [x] Graceful shutdown tests: SIGTERM/SIGINT handling, per-plugin timeout, connection draining
- [x] E2E browser tests: Playwright infrastructure with 17 smoke tests (#95)
- [x] Health endpoint tests: `/healthz`, `/readyz`, per-plugin health status
- [ ] Fuzz tests: API input fuzzing, configuration fuzzing (Go `testing.F`)
- [ ] Performance baselines: benchmark key operations, memory profile at 0/50 devices, startup time
- [ ] E2E browser tests: first-run wizard, device list, scan trigger, login/logout (Playwright, headless)
- [x] CI pipeline: GitHub Actions `ci.yml` with golangci-lint, `go test -race`, build, coverage report, license check
- [ ] CI coverage enforcement: fail PR if any package drops below minimum coverage target
- [x] `.golangci-lint.yml`: errcheck, gosec, gocritic, staticcheck, bodyclose, noctx, sqlclosecheck
- [x] GoReleaser configuration for cross-platform binary builds
- [x] Cross-platform CI: build verification for `linux/amd64`, `linux/arm64`, `windows/amd64`, `darwin/arm64`
- [x] OpenAPI spec generation (swaggo/swag)

#### Metrics & Measurement Infrastructure

- [x] Codecov integration: GitHub Action uploads coverage report, badge in README, PR comments with coverage diff
- [x] Go Report Card: register project at goreportcard.com, add badge to README
- [x] GitHub Dependabot: enable automated dependency vulnerability alerts
- [ ] GitHub Insights: establish baseline tracking cadence (weekly traffic review)
- [ ] Release download tracking: GoReleaser generates checksums, GitHub Releases API provides download counts
- [x] Docker image pull count tracking: publish to GitHub Container Registry (GHCR) or Docker Hub
- [x] README badges: CI build, coverage, Go Report Card, Go version, license, latest release (Docker pulls deferred until image published)

#### Community & Launch Readiness

- [x] CONTRIBUTING.md: development setup, code style, PR process, testing expectations, CLA explanation
- [x] Pull request template (`.github/pull_request_template.md`) with checklist (tests, lint, description)
- [x] First tagged release: `v0.1.0-alpha` with pre-built binaries (GoReleaser) and GitHub Release notes
- [x] Dockerfile: multi-stage build (builder + distroless/alpine runtime), multi-arch (`linux/amd64`, `linux/arm64`)
- [x] docker-compose.yml: one-command deployment matching the spec in Deployment section
- [x] README: "Why SubNetree?" section -- value proposition, feature comparison table (discovery + monitoring + remote access + vault + IoT in one tool), clear differentiation from Zabbix/LibreNMS/Uptime Kuma
- [x] README: status badges (CI build, Go version, license, latest release, coverage, Go Report Card)
- [x] README: Docker quickstart section (`docker run` one-liner + docker-compose snippet)
- [x] README: screenshots/GIF of dashboard (added in PR #156)
- [x] README: "Current Status" section -- honest about what works today vs. what's planned, links to roadmap
- [x] README: clarify licensing wording to "free for personal, home-lab, and non-competing production use"
- [x] Seed GitHub Issues: 5–10 issues labeled `good first issue` and `help wanted` (e.g., "add device type icon mapping", "write Prometheus exporter example plugin", "add ARM64 CI build target")
- [x] Seed GitHub Discussions: introductory post, roadmap discussion thread, "plugin ideas" thread, "show your setup" thread
- [ ] Community channel: create Discord server (or Matrix space) for real-time contributor discussion, linked from README and CONTRIBUTING.md
- [ ] Blog post / announcement: publish initial announcement on personal blog, r/homelab, r/selfhosted, Hacker News (after v0.1.0-alpha has working dashboard + discovery)
- [x] CODE_OF_CONDUCT.md: Contributor Covenant (standard, expected by contributors and evaluators)

### Phase 1b: Windows Scout Agent

**Status:** Complete. Core agent shipped 2026-02-13 (PRs #179-182, #190-192). mTLS + CA shipped 2026-02-14 (PRs #207-212). Scout reports metrics and system profiles to Dispatch via gRPC with mTLS.

**Goal:** First agent reporting metrics to server.

#### Pre-Phase Tooling Research

- [ ] Evaluate gRPC tooling: buf vs protoc, connect-go vs grpc-go
- [ ] Research Windows cross-compilation CI (GitHub Actions Windows runners, MSYS2 in CI)
- [ ] Evaluate agent packaging: MSI (WiX Toolset), NSIS, or Go-native installer
- [x] Research certificate management libraries for mTLS (Go stdlib crypto/x509 patterns -- internal/ca/ package, PRs #207-208)
- [ ] Evaluate Windows service management (golang.org/x/sys/windows/svc)

#### Scout Agent Implementation

- [x] Scout agent binary for Windows (functional: metrics, profiling, enrollment -- PRs #179-182)
- [x] Internal CA for mTLS certificate management (PRs #207-208)
- [x] Token-based enrollment (enrollment tokens with max uses and expiry -- PRs #180, #192; mTLS cert issued on enrollment -- PR #209)
- [x] gRPC communication (mTLS -- PRs #209-210)
- [x] System metrics: CPU, memory, disk, network (PR #181)
- [x] System profiling: hardware specs, installed software, running services (#164, PR #182)
- [ ] Exponential backoff reconnection
- [x] Certificate auto-renewal (90-day certs, renew at day 60 -- PR #211)
- [x] Dispatch module: agent list, status, check-in tracking (full implementation -- PR #179)
- [x] Dashboard: agent status view, enrollment flow (PRs #190, #191, #192)
- [ ] Proto management via buf (replace protoc)

#### Device Management API (#162)

- [x] Device CRUD endpoints: GET/PUT/DELETE `/devices/{id}`, POST `/devices`
- [x] Manual device creation (`discovery_method = "manual"`)
- [x] Device status history table and endpoint
- [x] Wire frontend device pages to backend (list, detail, edit, delete)
- [x] Device inventory management: categorization, bulk updates, inventory summary (#163)

#### Infrastructure Documentation (#132)

- [x] Docs plugin module with application + snapshot CRUD (`internal/docs/`)
- [x] Docker collector: container discovery and config capture (cross-platform)
- [x] Snapshot history browsing and LCS-based config diffing
- [x] Background retention maintenance worker
- [x] Dashboard Documentation tab with timeline, diff viewer, collector controls
- [ ] Additional collectors: systemd, Home Assistant, Plex (future)

### Documentation and UX (Cross-Cutting)

**Status:** Three-tier model adopted 2026-02-14. P0/P1 items shipped in v0.3.0 (PRs #225-232). MkDocs site not yet scaffolded.

**Goal:** Follow the three-tier documentation model (README landing page, MkDocs docs site, in-repo contributor docs). Remove barriers for first-time homelab users while keeping experienced users efficient.

**Source:** Novice UX Review (2026-02-14), competitive research of 9 high-adoption OSS projects.

**Reference:** [28-documentation-requirements.md](28-documentation-requirements.md), `.claude/rules/novice-ux-principles.md`

#### P0 - Infrastructure

- [ ] Set up MkDocs Material scaffolding (`mkdocs.yml`, `docs-site/` directory)
- [ ] Deploy docs site to GitHub Pages (`herbhall.github.io/subnetree`)
- [x] Verify Docker image is pullable on GHCR (#215, PR #225)
- [x] Simplify Quick Start to single recommended Docker path (#214, PR #231)

#### P1 - README and First Experience

- [x] Restructure README with user-first information hierarchy (#219, PR #232)
- [x] Add "What You'll Need" prerequisites section to README (#213, PR #230)
- [x] Replace jargon with user-benefit language in README (#220, PR #232)
- [x] Separate user vs dev docker-compose files (#217, PR #226)

#### P2 - Docs Site Content

- [ ] Getting Started: Installation page (tabbed Docker/Binary/Source) (#216)
- [ ] Getting Started: First Scan walkthrough
- [ ] Getting Started: Dashboard Tour
- [ ] Getting Started: FAQ
- [ ] Operations: Troubleshooting with common novice issues (#218)
- [ ] Operations: Platform-specific notes (#223)
- [ ] User Guide: Common tasks for day-2 operations (#221)
- [ ] Expand example config with novice-friendly comments (#222)
- [x] Add .env.example for Docker Compose users (#224, PR #227)

#### P3 - Polish

- [ ] Update comparison table setup time to be realistic (#225)
- [ ] Migrate existing `docs/guides/` content to docs site
- [ ] Auto-generated API reference from OpenAPI spec

### Phase 2: Core Monitoring + Multi-Tenancy

**Status:** Core monitoring shipped in v0.3.0. v0.4.0 features shipped: mDNS discovery (PR #248), metrics history (PR #243), dependency-aware alert suppression (PR #261), Linux Scout (PR #262). v0.5.0 features shipped: MkDocs site (PR #270), LLM BYOK providers (PR #271), NL query bar (PR #272), LLM settings UI (PR #273), AI recommendations (PR #274), logo integration (PR #275), power monitoring research (PR #276). Remaining: UPnP discovery, Tailscale plugin, multi-tenancy, maintenance windows, device inventory, analytics dashboard enhancements.

**Goal:** Comprehensive monitoring with alerting. MSP-ready multi-tenancy.

#### Pre-Phase Tooling Research

- [ ] Evaluate PostgreSQL + TimescaleDB: migration tooling (golang-migrate), hypertable performance, connection pooling
- [x] Research Docker multi-arch build pipeline (buildx, QEMU, manifest lists) (GoReleaser + buildx in v0.1.0-alpha)
- [x] Scaffold MkDocs Material documentation site, configure GitHub Pages deployment (PR #270)
- [ ] Evaluate Plausible Analytics: self-hosted vs cloud, deployment requirements
- [ ] Research OpenTelemetry Go SDK integration patterns for tracing
- [x] Evaluate SBOM generation tooling (Syft) and signing (Cosign) for release pipeline (Syft in GoReleaser since v0.1.0-alpha)
- [x] Research SNMP Go libraries (gosnmp) and MIB parsing (gosnmp adopted -- PR #204)
- [x] Evaluate mDNS/UPnP discovery libraries (hashicorp/mdns, huin/goupnp) (hashicorp/mdns adopted -- PR #248)

#### Discovery Enhancements

- [x] SNMP v2c/v3 discovery (gosnmp, credential-based -- PRs #204, #205)
- [x] mDNS/Bonjour discovery (PR #248, issue #234)
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

- [x] Uptime monitoring (ICMP, TCP port, HTTP/HTTPS) (ICMP in v0.2.0; TCP/HTTP in PRs #196, #202)
- [x] Sensible default thresholds (avoid alert fatigue)
- [x] Dependency-aware alerting (router down suppresses downstream alerts) (PR #261, issue #236)
- [x] Alert notifications: webhook with HMAC-SHA256 signing (PR #203; email, Slack, PagerDuty TODO)
- [x] Metrics history and time-series graphs (PR #243, issue #235)
- [ ] Maintenance windows (suppress alerts during scheduled work)

#### Multi-Tenancy

- [ ] TenantID on all core entities (Device, Agent, Credential)
- [ ] Tenant isolation in all queries (row-level filtering)
- [ ] Tenant management API
- [ ] Per-tenant configuration overrides
- [ ] Tenant-scoped API authentication
- [ ] Dashboard: tenant selector for MSP operators

#### Analytics (Insight Module -- Tier 1)

- [x] Insight plugin implementing `AnalyticsProvider` role
- [x] EWMA adaptive baselines for all monitored metrics
- [x] Z-score anomaly detection with configurable sensitivity (default: 3σ)
- [ ] Seasonal baselines (time-of-day, day-of-week patterns via Holt-Winters)
- [x] Trend detection and capacity forecasting (linear regression on sliding windows)
- [ ] Topology-aware alert correlation (suppress downstream alerts on parent failure)
- [x] Cross-metric correlation detection (e.g., CPU spike + packet loss on same device)
- [ ] Alert pattern learning (reduce sensitivity for chronic false positives)
- [x] Change-point detection (CUSUM algorithm for permanent shifts in metric behavior)
- [ ] Dashboard: anomaly indicators on metric charts (highlight deviations from baseline)
- [ ] Dashboard: capacity forecast warnings on device detail pages
- [ ] Dashboard: correlated alert grouping in alert list view
- [x] API: `/api/v1/analytics/anomalies` and `/api/v1/analytics/forecasts/{device_id}` endpoints
- [ ] Performance-profile-aware: disabled on micro, basic on small, full on medium+

#### Device Inventory Management (#163)

- [ ] Structured inventory fields on Device model: location, category, primary_role, justification, device_policy, owner
- [ ] Stale device detection (configurable threshold, default 30 days inactive)
- [ ] Dashboard: inventory view with category filter, sort by last seen
- [ ] Dashboard: inventory summary widget (counts by category, stale count)
- [ ] Bulk categorization endpoint (PATCH multiple devices)
- [ ] Policy recommendations: thin-client for portables, full-workstation for desktops

#### Service-to-Device Mapping (#165)

- [x] Service entity: maps discovered services (Docker, systemd, Windows) to host devices (PR #193)
- [x] Auto-populate from Docs module collectors + Scout system profiling (PR #193)
- [x] Correlate service resource usage with Pulse device metrics (PRs #194, #195)
- [x] Utilization grading per device (A-F rating based on efficiency) (PR #195)
- [x] Dashboard: service map view (device -> services -> utilization) (PR #195)
- [x] Dashboard: underutilized/overloaded device lists (PR #195)
- [ ] Service movement detection (service appears on new device)

#### Infrastructure

- [ ] PostgreSQL + TimescaleDB support (with hypertables for metrics and continuous aggregates for analytics feature engineering)
- [x] Scout: Linux agent (x64, ARM64) (PR #262, issue #240)
- [ ] Agent auto-update with binary signing (Cosign) and staged rollout
- [ ] `nvbuild` tool for custom binaries with third-party modules
- [ ] OpenTelemetry tracing
- [ ] Plugin developer SDK and documentation
- [ ] Interface Catalog: document all plugin interface types (API, Event, Config, Data) with versioning policy
- [x] Dashboard: monitoring views, alert management (PR #206; metric graphs TODO)
- [ ] MFA/TOTP authentication support
- [x] SBOM generation (Syft) and SLSA provenance for releases (Syft in GoReleaser since v0.1.0-alpha; Cosign signing TODO)
- [ ] Cosign signing for Docker images
- [x] govulncheck in CI pipeline (Trivy TODO)
- [ ] IPv6 scanning and agent communication support
- [ ] Per-tenant rate limiting
- [ ] Public demo instance: read-only demo on free-tier cloud (Oracle Cloud ARM64 or similar) with synthetic data, linked from README and website
- [ ] Project website (GitHub Pages or similar): documentation hub, blog, supporter showcase, demo link
- [ ] Opt-in telemetry: anonymous usage ping (weekly, disabled by default, payload documented and viewable in UI)
- [ ] Telemetry endpoint: simple HTTPS collector for installation count, MAU, feature usage tracking
- [ ] Google Search Console: register project website for organic search traffic tracking
- [ ] Plausible Analytics (self-hosted or cloud): privacy-friendly website analytics for project site
- [ ] Architecture Decision Records (ADRs): establish `docs/adr/` directory with template, document key decisions retroactively
- [ ] SonarQube Community (optional): technical debt tracking if Go Report Card + golangci-lint prove insufficient
- [ ] Modularity Metrics: establish baseline measures for plugin efficiency (shared components), flexibility (valid configurations), and agility (changes required for new features)

### Phase 3: Remote Access + Credential Vault

**Goal:** Browser-based remote access to any device with secure credential management.

#### Pre-Phase Tooling Research

- [x] Research WebSocket + xterm.js integration patterns for SSH-in-browser
- [ ] Evaluate Apache Guacamole Docker deployment for RDP/VNC proxying
- [x] Benchmark AES-256-GCM envelope encryption libraries in Go
- [x] Benchmark Argon2id key derivation across target platforms (cost parameter tuning)
- [x] Evaluate memguard for in-memory secret protection (decided: pure Go crypto, no CGo dependency)
- [x] Research LLM provider SDKs: OpenAI Go client, Anthropic SDK, Ollama local API (Ollama provider shipped in v0.1.0-alpha; OpenAI/Anthropic deferred)
- [ ] Evaluate data anonymization approaches for LLM context (PII stripping, metric-only summaries)

#### Remote Access & Vault Implementation

- [x] Gateway: SSH-in-browser via xterm.js (WebSocket backend shipped; frontend xterm.js deferred)
- [x] Gateway: HTTP/HTTPS reverse proxy via Go stdlib
- [ ] Gateway: RDP/VNC via Apache Guacamole (Docker)
- [x] Vault: AES-256-GCM envelope encryption
- [x] Vault: Argon2id master key derivation
- [ ] Vault: memguard for in-memory key protection (deferred -- pure Go crypto used instead)
- [ ] Vault: Per-device credential assignment
- [ ] Vault: Auto-fill credentials for remote sessions
- [x] Vault: Credential access audit logging
- [x] Vault: Master key rotation
- [ ] Dashboard: remote access launcher, session management, credential manager
- [ ] Tailscale plugin: prefer Tailscale IPs for Gateway remote access when device is on tailnet
- [ ] Scout: macOS agent
- [x] LLM integration: natural language query interface (OpenAI, Anthropic, Ollama providers) (Ollama v0.1.0-alpha; OpenAI/Anthropic PR #271)
- [ ] LLM integration: incident summarization on alert groups
- [x] LLM integration: "bring your own API key" configuration in settings (PR #271 backend, PR #273 frontend)
- [ ] LLM integration: privacy controls (data anonymization levels, local-only mode)
- [x] Dashboard: natural language query bar (optional, appears when LLM configured) (API endpoint v0.2.0; dashboard UI PR #272)
- [ ] Dashboard: AI-generated incident summaries on alert detail pages
- [ ] Vault: anomalous credential access detection (analytics-powered, from audit log events)

#### AI Infrastructure Optimization (#166)

- [x] Tier 1 rule-based recommendations (underutilized, overloaded, idle, upgrade needed) (PR #274)
- [ ] User-configurable optimization goals (utilization, responsiveness, power, balance)
- [ ] Recommendations API with accept/dismiss/snooze workflow
- [ ] Infrastructure Health Score (0-100) with category breakdown
- [x] Dashboard: recommendations panel with severity and suggested actions (PR #274)
- [ ] Tier 2 statistical recommendations (growth forecast, seasonal patterns, anomaly attribution)
- [ ] Tier 3 AI-assisted: migration planning, hardware advisor, what-if simulator (requires LLM)

### Phase 4: Extended Platform

**Goal:** IoT awareness, ecosystem growth, acquisition readiness.

#### Pre-Phase Tooling Research

- [ ] Evaluate MQTT Go libraries: Eclipse Paho vs alternatives (mochi-co/server for embedded broker)
- [ ] Research ONNX Runtime Go bindings (onnxruntime_go): platform support, model loading, inference performance
- [ ] Evaluate HashiCorp go-plugin for process-isolated third-party plugins (gRPC transport, versioning)
- [ ] Research plugin marketplace hosting: static index vs registry service, discovery UX
- [ ] Evaluate Home Assistant API integration patterns and authentication
- [ ] Research RBAC frameworks for Go (Casbin vs custom implementation)

#### HomeLab Platform Integrations

SubNetree is a dashboard and aggregator, not a replacement for HomeLab tools. These integrations provide status-at-a-glance and quick-launch access to other platforms:

- [ ] MQTT integration (Eclipse Paho) -- subscribe to status updates from IoT devices
- [ ] Home Assistant integration -- pull entity states, display status tiles, quick-launch to HA dashboard
- [ ] UnRAID integration -- pull Docker/VM status, disk health, array status; quick-launch to UnRAID UI
- [ ] Proxmox VE integration -- pull VM/LXC status, node health; quick-launch to Proxmox UI
- [ ] Generic service tiles -- configurable quick-launch links with optional health check (HTTP 200)
- [ ] Scout: Lightweight agent for devices without native integrations
- [ ] API: Public REST API with API key authentication
- [ ] RBAC: Custom roles with granular permissions
- [ ] Audit logging (all state-changing operations)
- [ ] Configuration backup for network devices (Oxidized-style)
- [ ] Plugin marketplace: curated index, `nvbuild` integration
- [ ] Plugin marketplace: AI/analytics plugin category
- [ ] Plugin Compatibility Matrix: define and publish tested/supported plugin combinations with configuration rules
- [ ] HashiCorp go-plugin support for process-isolated third-party plugins
- [ ] On-device inference: ONNX Runtime integration via onnxruntime_go
- [ ] On-device inference: device fingerprinting model (Python training pipeline + ONNX export)
- [ ] On-device inference: traffic classification model
- [ ] LLM integration: weekly/monthly report generation (scheduled, non-interactive)
- [ ] LLM integration: configuration assistance chatbot
- [ ] Comprehensive documentation: user guide, admin guide, plugin developer guide
- [ ] Performance benchmarks and optimization
