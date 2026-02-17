## Phased Roadmap

**Target Audience:** HomeLabbers and small business IT administrators running 10-500 devices on hardware from Raspberry Pi 4 to small business servers. See [01-product-vision.md](01-product-vision.md) for the five-tier hardware target system.

**Strategic Position:** "Start Here, Grow Anywhere." SubNetree is the gateway product -- a jack of all trades that provides 80% of what dedicated tools offer, then helps users graduate to specialized tools via standard protocols. The roadmap prioritizes integration and interoperability alongside core features.

**Key Integration Targets:** Prometheus/Grafana (monitoring), Home Assistant/MQTT (IoT), NetBox (CMDB), Ansible (IaC), Uptime Kuma/Beszel (complementary monitoring) -- the ecosystem tools that homelab and small business users actually run. See [01-product-vision.md Integration Strategy](01-product-vision.md) for the full priority matrix.

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
- [x] LLDP/CDP neighbor discovery for topology (LLDP via SNMP -- PR #370, v0.6.0)
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
- [x] E2E browser tests: first-run wizard, device list, scan trigger, login/logout (Playwright, headless) (Sprint 3, PR #409)
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
- [ ] Community channel: create Discord server (or Matrix space) for real-time contributor discussion, linked from README and CONTRIBUTING.md *(deferred: build stable feature-rich product first)*
- [ ] Blog post / announcement: publish initial announcement on personal blog, r/homelab, r/selfhosted, Hacker News *(deferred: announce when product is stable and feature-rich)*
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

**Status:** Core monitoring shipped in v0.3.0. v0.4.0: mDNS discovery, metrics history, alert suppression, Linux Scout. v0.4.1: MkDocs, LLM BYOK, NL query, AI recommendations, UPnP, topology enhancements, maintenance windows, inventory widget, analytics dashboard. v0.5.0: MQTT publisher, Alertmanager webhooks, CSV import/export, tier-aware defaults, recommendation engine catalog. v0.6.0: OUI classification, SNMP BRIDGE-MIB, TTL capture, LLDP discovery, port fingerprinting, composite classifier, unmanaged switch detection, service movement detection. v0.6.1: streaming scan pipeline with per-phase metrics, scan analytics page, scan health widget, agent download page, version display, scheduled scans, metrics consolidation. Sprint 1: CI smoke test in release pipeline, classification confidence on Device model, ICMP traceroute. Sprint 2: SNMP FDB table walks, seed data, interactive diagnostic tools. Sprint 3: network hierarchy inference, Playwright E2E tests. Post-QC: one-click Scout deployment with install scripts. Remaining: Tailscale plugin, multi-tenancy, seasonal baselines, alert pattern learning.

**Goal:** Comprehensive monitoring with alerting. MSP-ready multi-tenancy.

#### Pre-Phase Tooling Research

- [ ] Evaluate PostgreSQL + TimescaleDB: migration tooling (golang-migrate), hypertable performance, connection pooling
- [x] Research Docker multi-arch build pipeline (buildx, QEMU, manifest lists) (GoReleaser + buildx in v0.1.0-alpha)
- [x] Scaffold MkDocs Material documentation site, configure GitHub Pages deployment (PR #270)
- [ ] Evaluate Plausible Analytics: self-hosted vs cloud, deployment requirements *(deferred: needs community/website traffic first)*
- [ ] Research OpenTelemetry Go SDK integration patterns for tracing
- [x] Evaluate SBOM generation tooling (Syft) and signing (Cosign) for release pipeline (Syft in GoReleaser since v0.1.0-alpha)
- [x] Research SNMP Go libraries (gosnmp) and MIB parsing (gosnmp adopted -- PR #204)
- [x] Evaluate mDNS/UPnP discovery libraries (hashicorp/mdns, huin/goupnp) (hashicorp/mdns adopted -- PR #248)

#### Discovery Enhancements

- [x] SNMP v2c/v3 discovery (gosnmp, credential-based -- PRs #204, #205)
- [x] mDNS/Bonjour discovery (PR #248, issue #234)
- [x] UPnP/SSDP discovery (PR #292)
- [x] OUI device classifier with weighted scoring (v0.6.0)
- [x] Enhanced SNMP: BRIDGE-MIB and sysServices-based classification (v0.6.0)
- [x] TTL capture from ICMP for device fingerprinting (v0.6.0)
- [x] Port fingerprinting for device classification (v0.6.0)
- [x] Composite weighted classifier with confidence scoring (v0.6.0)
- [x] Unmanaged switch heuristic detection (v0.6.0)
- [x] Scan pipeline refactor for modular phase execution (v0.6.0)
- [x] SNMP FDB table walks for switch port mapping (Sprint 2, PR #403)
- [x] Network hierarchy inference from scan data -- NetworkLayer field on Device (Sprint 3, PR #408)
- [x] ICMP traceroute: `POST /api/v1/recon/traceroute` (Sprint 1, PR #402)
- [x] Classification confidence persisted on Device model: ClassificationConfidence, ClassificationSource, ClassificationSignals fields (Sprint 1, PR #401)
- [ ] Tailscale plugin: tailnet device discovery via Tailscale API
- [ ] Tailscale plugin: device merging (match by MAC, hostname, or IP overlap)
- [ ] Tailscale plugin: Tailscale IP enrichment on existing device records
- [ ] Tailscale plugin: subnet route detection and scan integration
- [ ] Tailscale plugin: MagicDNS hostname resolution
- [ ] Tailscale plugin: dashboard "Tailscale" badge on tailnet devices
- [ ] Scout over Tailscale: document and support agent communication via Tailscale IPs
- [x] Topology: real-time link utilization overlay (PR #293)
- [x] Topology: saved layouts with localStorage persistence (PR #293)
- [ ] Topology: custom backgrounds

#### Monitoring (Pulse)

- [x] Uptime monitoring (ICMP, TCP port, HTTP/HTTPS) (ICMP in v0.2.0; TCP/HTTP in PRs #196, #202)
- [x] Sensible default thresholds (avoid alert fatigue)
- [x] Dependency-aware alerting (router down suppresses downstream alerts) (PR #261, issue #236)
- [x] Alert notifications: webhook with HMAC-SHA256 signing (PR #203; email, Slack, PagerDuty TODO)
- [x] Metrics history and time-series graphs (PR #243, issue #235)
- [x] Maintenance windows (suppress alerts during scheduled work) (PR #294)
- [x] Streaming scan pipeline with per-phase metrics (v0.6.1)
- [x] Scan analytics page with health scoring (v0.6.1)
- [x] Scan health dashboard widget (v0.6.1)
- [x] Scheduled recurring scans (v0.6.1)
- [x] Metrics consolidation (v0.6.1)
- [x] Interactive diagnostic tools: ping, DNS lookup, port check (Sprint 2, PR #405)

#### Integration Foundation (Gateway P0)

- [x] Prometheus `/metrics` endpoint (shipped in Phase 1 -- `/metrics` at server root)
- [x] MQTT publisher for Home Assistant auto-discovery (device status, alerts, metrics as HA sensors) (PR #316, v0.5.0)
- [x] Alertmanager-compatible webhook format for alert notifications (PR #314, v0.5.0)
- [x] CSV import/export (universal device inventory interchange) (PR #315, v0.5.0)
- [x] Tier-aware default configuration (auto-detect hardware tier, set scan interval/retention/modules) (PR #317, v0.5.0)

#### Recommendation Engine Framework

Framework for hardware-aware growth recommendations. SubNetree uses Scout's hardware profiles to suggest modules and ecosystem tools the user's hardware can support. Full implementation in Phase 3-4; framework here enables the data model and basic recommendations.

- [x] `pkg/catalog/` data model: Go structs for catalog entries (tool name, category, hardware requirements, features, growth triggers, integration status) (PR #313, v0.5.0)
- [x] Embedded catalog: `catalog.yaml` compiled into binary via `//go:embed` with SubNetree modules + top 15 ecosystem tools (~50 KB) (PR #313, v0.5.0)
- [x] Hardware capability assessment: compare Scout's hardware profile (CPU, RAM, disk) against catalog entry requirements (PR #313, v0.5.0)
- [x] `GET /api/v1/recommendations` endpoint: returns personalized suggestions based on hardware tier and current module usage (PR #313, v0.5.0)
- [ ] Dashboard: basic recommendation card on overview page ("Your hardware supports enabling Analytics" or "Consider Uptime Kuma for dedicated uptime monitoring") *(API only via `GET /api/v1/recommendations` -- UI widget deferred to Phase 3)*

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
- [x] Dashboard: anomaly indicators on device detail page (PR #296)
- [x] Dashboard: capacity forecast warnings on device detail pages (PR #296)
- [x] Dashboard: correlated alert grouping on device detail page (PR #296)
- [x] API: `/api/v1/analytics/anomalies` and `/api/v1/analytics/forecasts/{device_id}` endpoints
- [ ] Performance-profile-aware: disabled on micro, basic on small, full on medium+

#### Device Inventory Management (#163)

- [x] Structured inventory fields on Device model: location, category, primary_role, justification, device_policy, owner (API/DB schema done)
- [ ] Dashboard: inventory field editing UI for structured fields (location, category, primary_role, etc.)
- [x] Stale device detection (configurable threshold, default 30 days inactive) (PR #295)
- [ ] Dashboard: inventory view with category filter, sort by last seen
- [x] Dashboard: inventory summary widget (counts by category, stale count) (PR #295)
- [ ] Bulk categorization endpoint (PATCH multiple devices)
- [ ] Policy recommendations: thin-client for portables, full-workstation for desktops

#### Service-to-Device Mapping (#165)

- [x] Service entity: maps discovered services (Docker, systemd, Windows) to host devices (PR #193)
- [x] Auto-populate from Docs module collectors + Scout system profiling (PR #193)
- [x] Correlate service resource usage with Pulse device metrics (PRs #194, #195)
- [x] Utilization grading per device (A-F rating based on efficiency) (PR #195)
- [x] Dashboard: service map view (device -> services -> utilization) (PR #195)
- [x] Dashboard: underutilized/overloaded device lists (PR #195)
- [x] Service movement detection (service appears on new device) (PR #288, v0.6.0)

#### Infrastructure

- [ ] PostgreSQL + TimescaleDB support (with hypertables for metrics and continuous aggregates for analytics feature engineering)
- [x] Scout: Linux agent (x64, ARM64) (PR #262, issue #240)
- [ ] Agent auto-update with binary signing (Cosign) and staged rollout
- [ ] `nvbuild` tool for custom binaries with third-party modules
- [ ] OpenTelemetry tracing
- [ ] Plugin developer SDK and documentation
- [ ] Interface Catalog: document all plugin interface types (API, Event, Config, Data) with versioning policy
- [x] Dashboard: monitoring views, alert management (PR #206; metric graphs TODO)
- [x] Dashboard: agent download page with platform-specific instructions (v0.6.1)
- [x] Dashboard: version display on setup page (v0.6.1)
- [x] One-click Scout agent deployment with install scripts and download redirects (PR #432)
- [x] GoReleaser bare binary archive for Scout (PR #432)
- [x] Dashboard: light theme color overrides for topology and charts (QC, PR #420)
- [x] Dashboard: compact device table rows, default sort by IP (QC, PR #422)
- [x] Dashboard: increased default page size to 256 for full Class C (QC, PR #424)
- [x] Dashboard: agent pages UX fixes -- setup link in empty state, shell labels (QC, PR #425)
- [ ] MFA/TOTP authentication support
- [x] SBOM generation (Syft) and SLSA provenance for releases (Syft in GoReleaser since v0.1.0-alpha; Cosign signing TODO)
- [ ] Cosign signing for Docker images
- [x] govulncheck in CI pipeline (Trivy TODO)
- [x] CI smoke test in release pipeline (Sprint 1, PR #400)
- [x] Seed data for staging/demo environments (Sprint 2, PR #404)
- [ ] IPv6 scanning and agent communication support
- [ ] Per-tenant rate limiting
- [ ] Public demo instance: read-only demo on free-tier cloud (Oracle Cloud ARM64 or similar) with synthetic data, linked from README and website *(deferred: launch when product is stable and feature-rich)*
- [ ] Project website (GitHub Pages or similar): documentation hub, blog, supporter showcase, demo link *(deferred: build after product maturity)*
- [ ] Opt-in telemetry: anonymous usage ping (weekly, disabled by default, payload documented and viewable in UI) *(deferred: needs community adoption first)*
- [ ] Telemetry endpoint: simple HTTPS collector for installation count, MAU, feature usage tracking *(deferred: needs community adoption first)*
- [ ] Google Search Console: register project website for organic search traffic tracking *(deferred: needs website and community)*
- [ ] Plausible Analytics (self-hosted or cloud): privacy-friendly website analytics for project site *(deferred: needs website and community)*
- [x] Architecture Decision Records (ADRs): establish `docs/adr/` directory with template, document key decisions retroactively (done in Phase 0; ADR-0001 through ADR-0008)
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

#### Integration Bridges (Gateway P1)

- [ ] NetBox JSON export (push discovered devices to NetBox CMDB via REST API)
- [ ] Ansible YAML inventory import (parse INI and YAML inventory as scan seed data)
- [ ] Ansible YAML inventory export (generate inventory from discovered devices with host_vars)
- [ ] Uptime Kuma monitor sync (auto-create monitors from discovered HTTP/TCP/DNS services via Socket.IO API)
- [ ] Nmap XML import (seed discovery from existing scan results)
- [ ] Grafana dashboard bundle (pre-built JSON dashboards + data source provisioning YAML)

#### Recommendation Engine (Full)

Builds on Phase 2 framework. Adds remote catalog, usage-pattern triggers, and rich dashboard UX.

- [ ] Remote catalog: weekly fetch from GitHub Pages static JSON, cached in SQLite, graceful offline degradation
- [ ] Usage-pattern growth triggers (e.g., monitor count > 50, needs status page, needs notification diversity)
- [ ] Two-tier recommendations: (a) possible on current hardware, (b) possible with hardware upgrade
- [ ] Feature comparison view: side-by-side SubNetree module vs ecosystem tool with hardware requirements
- [ ] Dashboard: recommendation cards with "Enable module" / "Learn about alternative" / "Dismiss" actions
- [ ] Dashboard: "Growth Path" page showing current tier, enabled modules, and available upgrades

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

**Goal:** Full ecosystem integration, IoT awareness, acquisition readiness. SubNetree becomes the "data pump" that feeds all other tools in the homelab/SMB stack.

#### Pre-Phase Tooling Research

- [x] Evaluate MQTT Go libraries: Eclipse Paho vs alternatives (Eclipse Paho adopted -- PR #316, v0.5.0)
- [ ] Research ONNX Runtime Go bindings (onnxruntime_go): platform support, model loading, inference performance
- [ ] Evaluate HashiCorp go-plugin for process-isolated third-party plugins (gRPC transport, versioning)
- [ ] Research plugin marketplace hosting: static index vs registry service, discovery UX
- [ ] Evaluate Home Assistant API integration patterns and authentication
- [ ] Research RBAC frameworks for Go (Casbin vs custom implementation)

#### Integration Ecosystem (Gateway P2/P3)

- [ ] NetAlertX device import (migrate existing discovery data via REST/GraphQL API)
- [ ] Markdown documentation generation (auto-generated per-device docs with hardware, services, change history)
- [ ] Nagios plugin executor (run existing check scripts, parse exit code + perfdata)
- [ ] Zabbix host/template import (parse `zabbix_export` JSON, map items to Pulse monitors)
- [ ] Nagios config import (parse `hosts.cfg`/`services.cfg`, extract check definitions)
- [ ] Beszel metric display (pull system metrics via PocketBase REST API into device detail view)
- [ ] Config versioning (git-based config snapshots, Oxidized-style, for Scout-collected configs)
- [ ] SubNetree Ansible dynamic inventory plugin (Python script querying `/api/v1/recon/devices`)
- [ ] Hudu API integration (push assets for MSP documentation workflows)
- [ ] PDF infrastructure report generation (inventory + topology + health + change history)
- [ ] NetBox bidirectional sync (webhook-driven reconciliation with conflict resolution)

#### Recommendation Catalog Maintenance

- [ ] Community-contributed catalog entries (PRs to catalog repo, reviewed before merge) *(deferred: needs community adoption)*
- [ ] Catalog versioning with changelog (users see "3 new tools added since last update")
- [ ] Integration status badges in catalog (none/planned/basic/full with links to setup docs)
- [ ] Anonymous opt-in usage telemetry to inform catalog priorities (which tools are users actually pairing with SubNetree) *(deferred: needs community adoption)*

#### HomeLab Platform Integrations

SubNetree is the gateway product that connects users to their existing tools. These integrations provide status-at-a-glance, quick-launch access, and data exchange with the broader homelab ecosystem:

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
