## Phased Roadmap

### Phase 0: Pre-Development Infrastructure

**Goal:** Establish project infrastructure, tooling, and processes before writing product code. Everything here is a prerequisite for efficient Phase 1 development.

#### Documentation Split
- [x] Create `docs/requirements/` directory structure
- [x] Split `requirements.md` into per-section files (28 files)
- [x] Create `docs/requirements/README.md` index with section descriptions
- [x] Update `.claude/CLAUDE.md` Documentation Map to reference individual files
- [x] Replace `requirements.md` with redirect to `docs/requirements/README.md`
- [ ] Verify all cross-references resolve correctly

#### Architecture Decision Records
- [x] Create `docs/adr/` directory with MADR template
- [x] Write ADR-0001: Split licensing model (BSL 1.1 + Apache 2.0)
- [x] Write ADR-0002: SQLite-first database strategy
- [x] Write ADR-0003: Plugin architecture (Caddy model with optional interfaces)
- [x] Write ADR-0004: Integer-based protocol versioning

#### GitHub Project Setup
- [ ] Create GitHub Projects v2 board with Kanban, Roadmap, and Table views
- [ ] Define custom fields: Phase, Module, Priority, Effort
- [ ] Create milestone for each phase (Phase 0 through Phase 4)
- [ ] Apply label taxonomy: type, priority, module, phase, contributor labels
- [ ] Seed initial issues from Phase 1 checklist items
- [x] Configure issue templates: bug report, feature request, plugin idea

#### CI/CD Pipeline Scaffolding
- [x] GitHub Actions: Go build matrix (Linux amd64/arm64, Windows amd64, macOS amd64/arm64)
- [x] GitHub Actions: test workflow (unit tests with race detector)
- [x] GitHub Actions: lint workflow (golangci-lint with project config)
- [x] GitHub Actions: license check workflow
- [x] GitHub Actions: CLA check workflow (CLA Assistant or custom)
- [x] Dependabot: configure for Go modules and GitHub Actions
- [ ] Pre-commit hooks: gofmt, go vet, license header check

#### Development Environment
- [ ] Document development setup in `docs/guides/contributing.md`
- [ ] Makefile: verify all targets work on Windows (MSYS2/Git Bash), Linux, macOS
- [x] `.editorconfig` for consistent formatting across editors
- [x] VS Code recommended extensions list (`.vscode/extensions.json`)
- [ ] Go workspace configuration (`go.work` if needed for multi-module)

#### Community Health Files
- [x] CONTRIBUTING.md: fork-and-PR workflow, commit conventions, code review process
- [x] CODE_OF_CONDUCT.md: Contributor Covenant v2.1
- [x] SECURITY.md: vulnerability reporting process
- [x] Pull request template (`.github/pull_request_template.md`)
- [x] Issue templates: bug, feature, plugin idea (`.github/ISSUE_TEMPLATE/`)

#### Metrics Baseline
- [ ] Register repository on Go Report Card
- [ ] Configure Codecov for coverage tracking
- [ ] Document badge URLs for README (CI, coverage, Go Report Card, license, release)

### Phase 1: Foundation (Server + Dashboard + Discovery + Topology)

**Goal:** Functional web-based network scanner with topology visualization. Validate architecture. Time to First Value under 10 minutes.

#### Pre-Phase Tooling Research
- [ ] Evaluate and configure golangci-lint (15+ linters, project-specific `.golangci-lint.yml`)
- [ ] Establish test framework patterns: table-driven tests, testify assertions, testcontainers for integration
- [ ] Set up Codecov integration for coverage tracking in CI
- [ ] Register repository on Go Report Card
- [ ] Configure GitHub Actions workflows: build, test, lint, license-check
- [ ] Configure Dependabot for Go modules and GitHub Actions
- [ ] Set up pre-commit hooks: gofmt, go vet, license header check
- [ ] Evaluate and document React + TypeScript toolchain for dashboard (Vite, ESLint, Prettier)

#### Architecture & Infrastructure
- [ ] Redesigned plugin system: `PluginInfo`, `Dependencies`, optional interfaces
- [ ] Config abstraction wrapping Viper
- [ ] Event bus (synchronous default with PublishAsync for slow consumers like analytics)
- [ ] Role interfaces in `pkg/roles/` (including `AnalyticsProvider` interface -- definition only, no implementation)
- [ ] Plugin registry with topological sort, graceful degradation
- [ ] Store interface + SQLite implementation (modernc.org/sqlite, pure Go)
- [ ] Per-plugin database migrations (reserve `analytics_` table prefix for Phase 2 Insight plugin)
- [ ] Repository interfaces in `internal/services/`
- [ ] Metrics collection format: uniform `(timestamp, device_id, metric_name, value, tags)` for analytics consumption

#### Server & API
- [ ] HTTP server with core routes
- [ ] RFC 7807 error responses
- [ ] Request ID middleware
- [ ] Structured request logging middleware
- [ ] Prometheus metrics at `/metrics`
- [ ] Liveness (`/healthz`) and readiness (`/readyz`) endpoints
- [ ] Per-IP rate limiting
- [ ] Configuration via YAML + environment variables
- [ ] Configurable Zap logger factory

#### Authentication
- [ ] Local auth with bcrypt password hashing
- [ ] JWT access/refresh token flow
- [ ] First-run setup endpoint (create admin when no users exist)
- [ ] OIDC/OAuth2 optional configuration

#### Recon Module
- [ ] ICMP ping sweep
- [ ] ARP scanning
- [ ] OUI manufacturer lookup (embedded database)
- [ ] LLDP/CDP neighbor discovery for topology
- [ ] Device persistence in SQLite
- [ ] Publishes `recon.device.discovered` events

#### Dashboard
- [ ] React + Vite + TypeScript + shadcn/ui + TanStack Query + Zustand
- [ ] First-run setup wizard
- [ ] Dashboard overview page (device counts, status summary)
- [ ] Device list with search, filter, sort, pagination
- [ ] Device detail page
- [ ] Network topology visualization (auto-generated from LLDP/CDP/ARP)
- [ ] Scan trigger with real-time progress (WebSocket)
- [ ] Dark mode support
- [ ] Settings page (server config, user profile)
- [ ] About page with version info, license, and Community Supporters section

#### Documentation
- [ ] Tailscale deployment guide: running NetVantage + Scout over Tailscale
- [ ] Tailscale Funnel/Serve guide: exposing dashboard without port forwarding

#### Operations
- [ ] Backup/restore CLI commands (`netvantage backup`, `netvantage restore`)
- [ ] Data retention configuration with automated purge job
- [ ] Security headers middleware (CSP, X-Frame-Options, HSTS, etc.)
- [ ] Account lockout after failed login attempts
- [ ] SECURITY.md with vulnerability disclosure process

#### Testing & Quality
- [ ] Test infrastructure: `internal/testutil/` with mocks, fixtures, helpers, mock clock
- [ ] Test infrastructure: `testdata/` directory with SNMP fixtures, test configs, migration snapshots
- [ ] Plugin contract tests: table-driven tests for `Plugin` interface and all optional interfaces
- [ ] Plugin isolation tests: panic recovery in Init, Start, Stop, and HTTP handlers
- [ ] Plugin lifecycle tests: full Init → Start → Stop cycle, dependency ordering, cascade disable
- [ ] Plugin API version validation tests: too old, too new, exact match, backward-compatible range
- [ ] API endpoint tests: `httptest.NewRecorder()` for all routes (status codes, content types, RFC 7807 errors)
- [ ] Security middleware tests: auth enforcement, security headers, CORS, CSRF, rate limiting (429)
- [ ] Input validation tests: malformed JSON, oversized payloads, SQL injection, XSS, path traversal
- [ ] Secrets hygiene tests: verify credentials never appear in log output or error responses
- [ ] Repository tests: in-memory SQLite CRUD, edge cases, transactions, constraint violations
- [ ] Database migration tests: fresh install, sequential upgrade, per-plugin isolation, idempotent check
- [ ] Configuration tests: defaults, env overrides, YAML overrides, invalid values, `config_version` validation
- [ ] Version compatibility tests: Plugin API, agent proto, config version, database schema version
- [ ] Graceful shutdown tests: SIGTERM/SIGINT handling, per-plugin timeout, connection draining
- [ ] Health endpoint tests: `/healthz`, `/readyz`, per-plugin health status
- [ ] Fuzz tests: API input fuzzing, configuration fuzzing (Go `testing.F`)
- [ ] Performance baselines: benchmark key operations, memory profile at 0/50 devices, startup time
- [ ] E2E browser tests: first-run wizard, device list, scan trigger, login/logout (Playwright, headless)
- [ ] CI pipeline: GitHub Actions `ci.yml` with golangci-lint, `go test -race`, build, coverage report, license check
- [ ] CI coverage enforcement: fail PR if any package drops below minimum coverage target
- [ ] `.golangci-lint.yml`: errcheck, gosec, gocritic, staticcheck, bodyclose, noctx, sqlclosecheck
- [ ] GoReleaser configuration for cross-platform binary builds
- [ ] Cross-platform CI: build verification for `linux/amd64`, `linux/arm64`, `windows/amd64`, `darwin/arm64`
- [ ] OpenAPI spec generation (swaggo/swag)

#### Metrics & Measurement Infrastructure
- [ ] Codecov integration: GitHub Action uploads coverage report, badge in README, PR comments with coverage diff
- [ ] Go Report Card: register project at goreportcard.com, add badge to README
- [ ] GitHub Dependabot: enable automated dependency vulnerability alerts
- [ ] GitHub Insights: establish baseline tracking cadence (weekly traffic review)
- [ ] Release download tracking: GoReleaser generates checksums, GitHub Releases API provides download counts
- [ ] Docker image pull count tracking: publish to GitHub Container Registry (GHCR) or Docker Hub
- [ ] README badges: CI build, coverage, Go Report Card, Go version, license, latest release, Docker pulls (see Success Metrics)

#### Community & Launch Readiness
- [ ] CONTRIBUTING.md: development setup, code style, PR process, testing expectations, CLA explanation
- [ ] Pull request template (`.github/pull_request_template.md`) with checklist (tests, lint, description)
- [ ] First tagged release: `v0.1.0-alpha` with pre-built binaries (GoReleaser) and GitHub Release notes
- [ ] Dockerfile: multi-stage build (builder + distroless/alpine runtime), multi-arch (`linux/amd64`, `linux/arm64`)
- [ ] docker-compose.yml: one-command deployment matching the spec in Deployment section
- [ ] README: "Why NetVantage?" section -- value proposition, feature comparison table (discovery + monitoring + remote access + vault + IoT in one tool), clear differentiation from Zabbix/LibreNMS/Uptime Kuma
- [ ] README: status badges (CI build, Go version, license, latest release, Docker pulls)
- [ ] README: Docker quickstart section (`docker run` one-liner + docker-compose snippet)
- [ ] README: screenshots/GIF of dashboard (blocked on dashboard implementation -- placeholder with architecture diagram until then)
- [ ] README: "Current Status" section -- honest about what works today vs. what's planned, links to roadmap
- [ ] README: clarify licensing wording to "free for personal, home-lab, and non-competing production use"
- [ ] Seed GitHub Issues: 5–10 issues labeled `good first issue` and `help wanted` (e.g., "add device type icon mapping", "write Prometheus exporter example plugin", "add ARM64 CI build target")
- [ ] Seed GitHub Discussions: introductory post, roadmap discussion thread, "plugin ideas" thread, "show your setup" thread
- [ ] Community channel: create Discord server (or Matrix space) for real-time contributor discussion, linked from README and CONTRIBUTING.md
- [ ] Blog post / announcement: publish initial announcement on personal blog, r/homelab, r/selfhosted, Hacker News (after v0.1.0-alpha has working dashboard + discovery)
- [ ] CODE_OF_CONDUCT.md: Contributor Covenant (standard, expected by contributors and evaluators)

### Phase 1b: Windows Scout Agent

**Goal:** First agent reporting metrics to server.

#### Pre-Phase Tooling Research
- [ ] Evaluate gRPC tooling: buf vs protoc, connect-go vs grpc-go
- [ ] Research Windows cross-compilation CI (GitHub Actions Windows runners, MSYS2 in CI)
- [ ] Evaluate agent packaging: MSI (WiX Toolset), NSIS, or Go-native installer
- [ ] Research certificate management libraries for mTLS (Go stdlib crypto/x509 patterns)
- [ ] Evaluate Windows service management (golang.org/x/sys/windows/svc)

- [ ] Scout agent binary for Windows
- [ ] Internal CA for mTLS certificate management
- [ ] Token-based enrollment with certificate signing
- [ ] gRPC communication with mTLS
- [ ] System metrics: CPU, memory, disk, network
- [ ] Exponential backoff reconnection
- [ ] Certificate auto-renewal (90-day certs, renew at day 60)
- [ ] Dispatch module: agent list, status, check-in tracking
- [ ] Dashboard: agent status view, enrollment flow
- [ ] Proto management via buf (replace protoc)

### Phase 2: Core Monitoring + Multi-Tenancy

**Goal:** Comprehensive monitoring with alerting. MSP-ready multi-tenancy.

#### Pre-Phase Tooling Research
- [ ] Evaluate PostgreSQL + TimescaleDB: migration tooling (golang-migrate), hypertable performance, connection pooling
- [ ] Research Docker multi-arch build pipeline (buildx, QEMU, manifest lists)
- [ ] Scaffold Hugo + Docsy documentation site, configure GitHub Pages deployment
- [ ] Evaluate Plausible Analytics: self-hosted vs cloud, deployment requirements
- [ ] Research OpenTelemetry Go SDK integration patterns for tracing
- [ ] Evaluate SBOM generation tooling (Syft) and signing (Cosign) for release pipeline
- [ ] Research SNMP Go libraries (gosnmp) and MIB parsing
- [ ] Evaluate mDNS/UPnP discovery libraries (hashicorp/mdns, huin/goupnp)

#### Discovery Enhancements
- [ ] SNMP v2c/v3 discovery
- [ ] mDNS/Bonjour discovery
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
- [ ] Uptime monitoring (ICMP, TCP port, HTTP/HTTPS)
- [ ] Sensible default thresholds (avoid alert fatigue)
- [ ] Dependency-aware alerting (router down suppresses downstream alerts)
- [ ] Alert notifications: email, webhook, Slack, PagerDuty (as notifier plugins)
- [ ] Metrics history and time-series graphs
- [ ] Maintenance windows (suppress alerts during scheduled work)

#### Multi-Tenancy
- [ ] TenantID on all core entities (Device, Agent, Credential)
- [ ] Tenant isolation in all queries (row-level filtering)
- [ ] Tenant management API
- [ ] Per-tenant configuration overrides
- [ ] Tenant-scoped API authentication
- [ ] Dashboard: tenant selector for MSP operators

#### Analytics (Insight Module -- Tier 1)
- [ ] Insight plugin implementing `AnalyticsProvider` role
- [ ] EWMA adaptive baselines for all monitored metrics
- [ ] Z-score anomaly detection with configurable sensitivity (default: 3σ)
- [ ] Seasonal baselines (time-of-day, day-of-week patterns via Holt-Winters)
- [ ] Trend detection and capacity forecasting (linear regression on sliding windows)
- [ ] Topology-aware alert correlation (suppress downstream alerts on parent failure)
- [ ] Cross-metric correlation detection (e.g., CPU spike + packet loss on same device)
- [ ] Alert pattern learning (reduce sensitivity for chronic false positives)
- [ ] Change-point detection (CUSUM algorithm for permanent shifts in metric behavior)
- [ ] Dashboard: anomaly indicators on metric charts (highlight deviations from baseline)
- [ ] Dashboard: capacity forecast warnings on device detail pages
- [ ] Dashboard: correlated alert grouping in alert list view
- [ ] API: `/api/v1/analytics/anomalies` and `/api/v1/analytics/forecasts/{device_id}` endpoints
- [ ] Performance-profile-aware: disabled on micro, basic on small, full on medium+

#### Infrastructure
- [ ] PostgreSQL + TimescaleDB support (with hypertables for metrics and continuous aggregates for analytics feature engineering)
- [ ] Scout: Linux agent (x64, ARM64)
- [ ] Agent auto-update with binary signing (Cosign) and staged rollout
- [ ] `nvbuild` tool for custom binaries with third-party modules
- [ ] OpenTelemetry tracing
- [ ] Plugin developer SDK and documentation
- [ ] Dashboard: monitoring views, alert management, metric graphs
- [ ] MFA/TOTP authentication support
- [ ] SBOM generation (Syft) and SLSA provenance for releases
- [ ] Cosign signing for Docker images
- [ ] govulncheck + Trivy in CI pipeline
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

### Phase 3: Remote Access + Credential Vault

**Goal:** Browser-based remote access to any device with secure credential management.

#### Pre-Phase Tooling Research
- [ ] Research WebSocket + xterm.js integration patterns for SSH-in-browser
- [ ] Evaluate Apache Guacamole Docker deployment for RDP/VNC proxying
- [ ] Benchmark AES-256-GCM envelope encryption libraries in Go
- [ ] Benchmark Argon2id key derivation across target platforms (cost parameter tuning)
- [ ] Evaluate memguard for in-memory secret protection (Go compatibility, platform support)
- [ ] Research LLM provider SDKs: OpenAI Go client, Anthropic SDK, Ollama local API
- [ ] Evaluate data anonymization approaches for LLM context (PII stripping, metric-only summaries)

- [ ] Gateway: SSH-in-browser via xterm.js
- [ ] Gateway: HTTP/HTTPS reverse proxy via Go stdlib
- [ ] Gateway: RDP/VNC via Apache Guacamole (Docker)
- [ ] Vault: AES-256-GCM envelope encryption
- [ ] Vault: Argon2id master key derivation
- [ ] Vault: memguard for in-memory key protection
- [ ] Vault: Per-device credential assignment
- [ ] Vault: Auto-fill credentials for remote sessions
- [ ] Vault: Credential access audit logging
- [ ] Vault: Master key rotation
- [ ] Dashboard: remote access launcher, session management, credential manager
- [ ] Tailscale plugin: prefer Tailscale IPs for Gateway remote access when device is on tailnet
- [ ] Scout: macOS agent
- [ ] LLM integration: natural language query interface (OpenAI, Anthropic, Ollama providers)
- [ ] LLM integration: incident summarization on alert groups
- [ ] LLM integration: "bring your own API key" configuration in settings
- [ ] LLM integration: privacy controls (data anonymization levels, local-only mode)
- [ ] Dashboard: natural language query bar (optional, appears when LLM configured)
- [ ] Dashboard: AI-generated incident summaries on alert detail pages
- [ ] Vault: anomalous credential access detection (analytics-powered, from audit log events)

### Phase 4: Extended Platform

**Goal:** IoT awareness, ecosystem growth, acquisition readiness.

#### Pre-Phase Tooling Research
- [ ] Evaluate MQTT Go libraries: Eclipse Paho vs alternatives (mochi-co/server for embedded broker)
- [ ] Research ONNX Runtime Go bindings (onnxruntime_go): platform support, model loading, inference performance
- [ ] Evaluate HashiCorp go-plugin for process-isolated third-party plugins (gRPC transport, versioning)
- [ ] Research plugin marketplace hosting: static index vs registry service, discovery UX
- [ ] Evaluate Home Assistant API integration patterns and authentication
- [ ] Research RBAC frameworks for Go (Casbin vs custom implementation)

- [ ] MQTT broker integration (Eclipse Paho)
- [ ] Home Assistant API integration
- [ ] Scout: Lightweight IoT agent
- [ ] API: Public REST API with API key authentication
- [ ] RBAC: Custom roles with granular permissions
- [ ] Audit logging (all state-changing operations)
- [ ] Configuration backup for network devices (Oxidized-style)
- [ ] Plugin marketplace: curated index, `nvbuild` integration
- [ ] Plugin marketplace: AI/analytics plugin category
- [ ] HashiCorp go-plugin support for process-isolated third-party plugins
- [ ] On-device inference: ONNX Runtime integration via onnxruntime_go
- [ ] On-device inference: device fingerprinting model (Python training pipeline + ONNX export)
- [ ] On-device inference: traffic classification model
- [ ] LLM integration: weekly/monthly report generation (scheduled, non-interactive)
- [ ] LLM integration: configuration assistance chatbot
- [ ] Comprehensive documentation: user guide, admin guide, plugin developer guide
- [ ] Performance benchmarks and optimization
