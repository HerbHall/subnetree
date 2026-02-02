## Testing Strategy

Testing is not a phase -- it is a continuous requirement. The test suite is the primary mechanism for ensuring the stability and security commitments in the Non-Functional Requirements. Every feature ships with tests. Every bug fix ships with a regression test. PRs that reduce coverage are rejected by CI.

### Testing Principles

1. **Tests are the stability guarantee.** The Non-Functional Requirements promise months of unattended operation. Only automated tests can verify this at scale.
2. **Fast feedback first.** Unit tests run in < 30 seconds. Integration tests run in < 5 minutes. Developers should never wait to run the fast suite.
3. **Deterministic, not flaky.** Tests that pass sometimes and fail sometimes are worse than no tests. All tests must be deterministic. Time-dependent tests use a mock clock. Network-dependent tests use recorded fixtures or local containers.
4. **Test the contract, not the implementation.** Plugin contract tests verify interface compliance, not internal state. API tests verify request/response pairs, not handler internals. This allows refactoring without rewriting tests.
5. **Coverage targets are minimums, not goals.** Meeting 70% coverage does not mean the code is well-tested. Coverage prevents large untested gaps but does not guarantee quality. Critical paths (auth, encryption, plugin lifecycle) require explicit test cases regardless of coverage numbers.

### Test Categories

#### Unit Tests

- **Plugin contract tests:** Shared contract test suite in `pkg/plugin/plugintest/` verifying every plugin against the `Plugin` interface. Each module's `_test.go` calls `plugintest.TestPluginContract(t, factory)` to verify: valid metadata, successful init, start-after-init, stop-without-start safety, and info idempotency. Optional interfaces (`HTTPProvider`, `GRPCProvider`, `HealthChecker`, `EventSubscriber`, `Validator`, `Reloadable`, `AnalyticsProvider`) are tested per-module with mocked dependencies.
- **Handler tests:** `httptest.NewRecorder()` for all API endpoints. Every route returns the correct status code, content type, response body structure, and error format (RFC 7807). Every authenticated endpoint rejects unauthenticated requests.
- **Repository tests:** In-memory SQLite (`:memory:`) for database logic. Every repository method tested for CRUD operations, edge cases (empty results, duplicate keys, constraint violations), and transaction behavior.
- **Mock strategy:** Interface-based mocking for external dependencies (PingScanner, ARPScanner, SNMPClient, DNSResolver, CredentialStore, EventBus). Mocks live in `internal/testutil/mocks.go` and are generated from interfaces.
- **SNMP fixtures:** Recorded SNMP responses stored as JSON in `testdata/snmp/`. Tests replay these fixtures instead of querying live devices.
- **Configuration tests:** Every config key has a test for default value, environment variable override, YAML override, and invalid value rejection. `config_version` validation tested for missing, current, old, and future versions.
- **Version validation tests:** Plugin API version checking (too old, too new, exact match, backward-compatible). Config version validation. Database schema version checking.

#### Integration Tests

- **Build tag:** `//go:build integration` -- excluded from `make test`, included in `make test-integration`
- **Database:** `testcontainers-go` for PostgreSQL + TimescaleDB. Integration tests exercise the full database layer including migrations, queries, and TimescaleDB-specific features (hypertables, continuous aggregates).
- **Full server wire-up:** `httptest.Server` wrapping the real HTTP handler (exposed via `Server.Handler()` method). Tests exercise the full request pipeline: middleware, routing, auth, handler, database, response.
- **Plugin lifecycle:** Full Init → Start → Stop cycle with real dependencies (in-memory SQLite for unit, testcontainers for integration).
- **gRPC agent communication:** Full Enroll → CheckIn → ReportMetrics cycle using `bufconn` (in-memory gRPC transport). Tests verify mTLS handshake, version negotiation, metric delivery, and command dispatch.
- **Event bus:** End-to-end event flow: Recon discovers device → event published → Pulse picks up monitoring → metrics collected → analytics baseline updated.

#### Security Tests

Security tests verify every requirement in the Non-Functional Requirements > Security section. These are not optional and run on every PR.

**Authentication & Authorization:**
- JWT token issuance, validation, expiration, and refresh flow
- First-run setup wizard creates admin, rejects if admin exists
- Authenticated endpoints reject missing/expired/malformed tokens
- Password policy enforcement (minimum length, breached password check)
- Account lockout after 5 failed attempts, progressive delay
- Session limit enforcement (max 5 concurrent)
- OIDC/OAuth2 flow when configured (Phase 2)
- MFA/TOTP verification when enabled (Phase 2)

**Transport & Encryption:**
- TLS 1.2+ enforcement (reject TLS 1.0/1.1 connections)
- mTLS certificate validation for agent communication
- Vault credential encryption roundtrip (encrypt → store → retrieve → decrypt)
- Vault master key derivation (Argon2id parameters)
- memguard key protection verification

**Web Security:**
- Security headers present on every response: `Content-Security-Policy`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Strict-Transport-Security`, `Referrer-Policy`, `Permissions-Policy`
- CORS rejects cross-origin requests in production mode
- CSRF protection: `SameSite=Strict` cookies + `X-Requested-With` header validation
- Rate limiting enforced: verify 429 responses after threshold exceeded
- Input validation: reject malformed JSON, oversized payloads, SQL injection attempts, XSS payloads, path traversal attempts in every endpoint that accepts user input

**Secrets Hygiene:**
- Credential values never appear in structured log output (test by capturing zap output)
- Credential values never appear in API error responses
- Credential values never appear in Prometheus metrics labels
- Stack traces redact sensitive fields

#### Stability & Reliability Tests

These tests verify the system can run unattended and degrade gracefully under failure conditions.

**Plugin Isolation:**
- A plugin that panics during `Init()` is caught via `recover()`, logged, and disabled -- server continues starting
- A plugin that panics during `Start()` is caught, logged, and disabled -- other plugins continue
- A plugin that panics in an HTTP handler is caught via middleware recovery -- returns 500, other routes unaffected
- A plugin that blocks forever in `Stop()` is killed after the per-plugin timeout -- shutdown completes
- A plugin that exceeds its memory budget triggers backpressure, not a crash
- Cascade disable: if plugin A fails and plugin B depends on A, plugin B is also disabled with a clear log message

**Graceful Shutdown:**
- SIGTERM triggers orderly shutdown: stop accepting new connections → drain in-flight requests → stop plugins in reverse dependency order → checkpoint database → exit 0
- SIGINT triggers same orderly shutdown
- Second SIGTERM/SIGINT during shutdown forces immediate exit
- Per-plugin stop timeout: plugins that don't stop within timeout are force-killed, remaining plugins still stop cleanly
- Active WebSocket connections receive close frame before server exits

**Database Resilience:**
- SQLite WAL mode: concurrent readers do not block writer
- WAL checkpoint runs correctly (verify WAL file size bounded)
- Database locked error during write retries with backoff (not crash)
- Corrupted WAL file detected at startup with actionable error message
- Migration version too new (from newer server) detected and rejected at startup
- In-flight transactions committed or rolled back cleanly on shutdown

**Circuit Breakers & Timeouts:**
- Every outbound operation (ICMP scan, SNMP query, DNS lookup, HTTP check, Tailscale API call) has a configurable timeout
- Timed-out operations return errors, do not hang goroutines
- Circuit breaker opens after N consecutive failures to an external service, returns fast-fail, closes after cooldown period

**Resource Limits:**
- Event bus queue depth: sustained depth > 1,000 triggers backpressure alert
- Database query queue depth: bounded, excess queries rejected with 503
- Goroutine count: bounded by semaphores for scan workers (not `go func()` in a loop)

#### Performance Tests

Performance tests verify the targets in Non-Functional Requirements > Performance. These run in CI on a schedule (nightly) rather than on every PR, because they require consistent hardware for meaningful results.

**Benchmarks (Go `testing.B`):**
- API response time: `/api/v1/health` < 5ms, `/api/v1/devices` (100 devices) < 50ms, `/api/v1/devices` (1,000 devices) < 100ms
- Database query time: single device lookup < 1ms, device list with pagination < 10ms, metric insertion batch (100 metrics) < 20ms
- Plugin lifecycle: full Init → Start → Stop cycle < 500ms per plugin
- Event bus throughput: 10,000 events/second with 5 subscribers
- JSON serialization: device list (100 devices) < 1ms

**Memory Profiling:**
- Startup memory with zero devices < 50 MB (verify via `runtime.MemStats`)
- Memory at 50 devices < 200 MB (load simulated devices, measure RSS)
- Memory at 200 devices < 500 MB
- Memory at 500 devices < 1.2 GB
- No memory leaks: run 1-hour soak test with continuous scan cycles, verify heap does not grow unbounded (heap in-use at end within 110% of heap in-use at minute 5)

**Startup Time:**
- Server starts and serves `/healthz` within 5 seconds (micro profile)
- Server starts and serves `/healthz` within 10 seconds (large profile with all plugins)

**Scan Performance:**
- /24 subnet ICMP sweep completes in < 30 seconds
- /24 subnet ARP scan completes in < 10 seconds

**Agent Performance:**
- Scout binary size < 15 MB (verified in CI via `ls -la`)
- Scout idle CPU < 1% (measured over 60-second window)
- Scout collection CPU < 5% (measured during metric collection burst)
- Scout memory < 20 MB on x64, < 25 MB on ARM64

#### Version Compatibility Tests

These tests ensure components built for different versions work correctly together, directly validating the Version Management strategy.

**Plugin API Compatibility:**
- Plugin with `APIVersion == PluginAPIVersionCurrent` loads successfully
- Plugin with `APIVersion` in `[PluginAPIVersionMin, PluginAPIVersionCurrent)` loads with deprecation warning
- Plugin with `APIVersion < PluginAPIVersionMin` rejected with clear error
- Plugin with `APIVersion > PluginAPIVersionCurrent` rejected with clear error

**Agent-Server Protocol:**
- Agent with `proto_version == server_proto_version` gets `VERSION_OK`
- Agent with `proto_version == server_proto_version - 1` gets `VERSION_DEPRECATED`, check-in succeeds
- Agent with `proto_version < server_proto_version - 1` gets `VERSION_REJECTED`, check-in fails
- Agent with `proto_version > server_proto_version` gets `VERSION_REJECTED`

**Configuration Compatibility:**
- Config with missing `config_version` treated as version 1, server starts
- Config with `config_version == current` loads normally
- Config with `config_version < current` loads with warning, suggests migration
- Config with `config_version > current` rejects with error

**Database Migration:**
- Fresh database: all migrations run in order, schema matches expected state
- Existing database: only new migrations run, existing data preserved
- Migration from newer server version: detected and rejected at startup
- Per-plugin migrations: each plugin's migrations isolated (recon tables don't affect pulse tables)

#### Database Migration Tests

Migrations are irreversible. Testing them is critical because a broken migration in production requires manual intervention.

- **Fresh install:** Every migration runs successfully on an empty database. Final schema matches expected state (verified by comparing table definitions).
- **Sequential upgrade:** Simulate upgrade path from v0.1.0 through each intermediate version to current. Verify data survives each migration.
- **Per-plugin isolation:** Plugin A's migration does not modify Plugin B's tables. Verified by running each plugin's migrations independently.
- **Idempotent check:** Running migrations twice does not error (migration tracker prevents re-execution).
- **Rollback detection:** If a migration panics, the transaction is rolled back and the migration is not marked as applied. Server logs the failure and refuses to start until the issue is resolved.
- **Reserved prefixes:** `analytics_` table prefix reserved for Phase 2 Insight plugin. Other plugins cannot create tables with this prefix.

#### Fuzz Testing

Go's built-in fuzz testing (`testing.F`) for inputs that cross trust boundaries:

- **API input fuzzing:** Fuzz JSON request bodies for all POST/PUT endpoints. Verify the server never panics, always returns valid HTTP responses (even if 400/500).
- **Configuration fuzzing:** Fuzz YAML configuration input. Verify the server either starts correctly or refuses to start with a clear error (never panics, never corrupts state).
- **SNMP response fuzzing:** Fuzz raw SNMP response bytes. Verify the parser never panics, returns errors for malformed input.
- **gRPC message fuzzing:** Fuzz protobuf message bytes sent to the gRPC server. Verify the server never panics.

Fuzz tests run in CI nightly (not on every PR due to runtime).

#### Cross-Platform Tests

CI verifies the following build targets on every PR:

| Target | Build | Unit Tests | Integration Tests |
|--------|-------|------------|-------------------|
| `linux/amd64` | Yes | Yes | Yes (primary) |
| `linux/arm64` | Yes | Yes (via QEMU or native runner) | No (Phase 2) |
| `windows/amd64` | Yes | Yes | No (Phase 2) |
| `darwin/arm64` | Yes | No (no macOS CI runner) | No |

**Phase 2:** Add ARM64 integration tests (self-hosted ARM64 runner or cross-compilation + QEMU), Windows integration tests.

#### End-to-End Tests (Dashboard)

When the React dashboard is implemented (Phase 1), E2E tests verify the full user journey:

- **Framework:** Playwright (headless Chromium)
- **Scope:** Critical user paths only (not exhaustive UI testing)
- **Test cases:**
  - First-run setup wizard: create admin account, configure network range
  - Dashboard loads and displays device count
  - Device list: search, filter, sort, pagination
  - Trigger manual scan, observe real-time progress via WebSocket
  - Device detail page loads with correct data
  - Topology map renders (visual snapshot test)
  - Settings page: change config, verify saved
  - Login/logout flow
  - Dark mode toggle persists across reload

E2E tests run in CI on every PR (headless, < 2 minutes).

#### Upgrade Tests

Verify that upgrading from one version to another works without data loss or service disruption.

- **Binary upgrade:** Install version N, populate with test data (devices, metrics, alerts, credentials), replace binary with version N+1, restart, verify all data intact and accessible.
- **Database migration upgrade:** Snapshot database at version N, run version N+1 migrations, verify schema correct and data preserved.
- **Config migration:** Load version N config with version N+1 server, verify warning and `netvantage config migrate` produces valid config.
- **Agent compatibility across upgrade:** Server at version N+1 accepts agents still at version N (within N-1 proto window).
- **Rollback detection:** After upgrading server to N+1, starting the old version N binary detects the newer database and refuses to start (does not corrupt data).

### Test Infrastructure

#### Test Commands

```bash
make test              # Unit tests only, -race flag, < 30 seconds
make test-integration  # Integration tests (requires Docker), < 5 minutes
make test-coverage     # Unit tests + coverage report (HTML + text)
make test-fuzz         # Fuzz tests, 30-second budget per fuzz target
make test-bench        # Performance benchmarks
make test-e2e          # End-to-end browser tests (requires built dashboard)
make test-all          # All of the above except fuzz
make lint              # golangci-lint (includes go vet, staticcheck, errcheck, gosec, gocritic)
```

#### Test Directory Layout

```
internal/
  testutil/
    mocks.go          # Generated mocks for all external interfaces
    fixtures.go       # Shared test fixtures (sample devices, metrics, configs)
    helpers.go        # Test helper functions (setup server, create test DB, etc.)
    clock.go          # Mock clock for time-dependent tests
  plugin/
    plugin_test.go    # Plugin interface contract tests
    registry_test.go  # Registry lifecycle, dependency ordering, version checking
  server/
    server_test.go    # HTTP handler tests
    config_test.go    # Configuration loading, validation, version checking
    middleware_test.go # Auth, rate limiting, security headers, request ID
  recon/
    recon_test.go     # Discovery logic with mocked scanners
  pulse/
    pulse_test.go     # Monitoring logic with mocked checks
  (etc. for each module)
pkg/
  plugin/
    plugin_test.go    # SDK contract tests (exported interfaces)
  models/
    models_test.go    # Model validation, serialization
testdata/
  snmp/               # Recorded SNMP responses (JSON fixtures)
  configs/            # Test configuration files (valid, invalid, edge cases)
  migrations/         # Database snapshots for migration testing
test/
  e2e/                # Playwright end-to-end tests
  bench/              # Performance benchmark scenarios
  fuzz/               # Fuzz corpus and seed inputs
```

#### CI Configuration (`.golangci-lint.yml`)

```yaml
linters:
  enable:
    - errcheck         # Unchecked errors
    - gosec            # Security issues (SQL injection, hardcoded creds, weak crypto)
    - gocritic         # Code style and common mistakes
    - govet            # Go vet checks
    - staticcheck      # Comprehensive static analysis
    - ineffassign      # Ineffectual assignments
    - unused           # Unused code
    - misspell         # Spelling mistakes in comments/strings
    - bodyclose        # HTTP response body not closed
    - noctx            # HTTP request without context
    - sqlclosecheck    # SQL rows/stmt not closed
    - exportloopref    # Loop variable capture
    - durationcheck    # Suspicious duration math
    - exhaustive       # Missing enum cases in switch
    - nilerr           # Returning nil error after error check
    - prealloc         # Suggest slice preallocation
```

#### Coverage Targets

| Package | Minimum Coverage | Rationale |
|---------|-----------------|-----------|
| `pkg/plugin/` | 90%+ | Core contract -- bugs here affect every plugin |
| `pkg/roles/` | 90%+ | Role interfaces -- bugs here break module system |
| `internal/server/` | 80%+ | HTTP handling -- user-facing, security-critical |
| `internal/server/middleware/` | 90%+ | Auth, rate limiting, security headers -- security-critical |
| `internal/plugin/` | 85%+ | Registry, lifecycle, version checking -- stability-critical |
| `internal/recon/` | 70%+ | Discovery business logic |
| `internal/pulse/` | 70%+ | Monitoring business logic |
| `internal/dispatch/` | 70%+ | Agent management |
| `internal/vault/` | 85%+ | Credential handling -- security-critical |
| `internal/gateway/` | 70%+ | Remote access |
| `internal/scout/` | 70%+ | Agent core logic |
| `cmd/` | 50%+ | CLI wiring (lower target due to `main()` difficulty) |

**CI enforcement:** Coverage is measured on every PR. If a PR reduces coverage of any package below its minimum, the CI check fails. Coverage reports are uploaded as PR comments for visibility.

### Test Phasing

Not all test categories are needed in Phase 1. The test plan phases with the feature roadmap:

| Test Category | Phase 1 | Phase 1b | Phase 2 | Phase 3 | Phase 4 |
|---------------|---------|----------|---------|---------|---------|
| Plugin contract tests | Yes | Yes | Yes | Yes | Yes |
| HTTP handler tests | Yes | -- | Yes | Yes | Yes |
| Repository tests (SQLite) | Yes | -- | Yes | -- | -- |
| Configuration tests | Yes | -- | Yes | -- | -- |
| Security header tests | Yes | -- | -- | -- | -- |
| Auth/JWT tests | Yes | -- | Yes (OIDC, MFA) | -- | Yes (RBAC) |
| Rate limiting tests | Yes | -- | Yes (per-tenant) | -- | -- |
| Input validation tests | Yes | -- | Yes | Yes | Yes |
| Plugin isolation tests | Yes | -- | -- | -- | -- |
| Graceful shutdown tests | Yes | -- | -- | -- | -- |
| Version compatibility tests | Yes | Yes | Yes | Yes | Yes |
| Database migration tests | Yes | -- | Yes (PostgreSQL) | -- | -- |
| gRPC agent tests | -- | Yes | Yes | -- | -- |
| mTLS certificate tests | -- | Yes | -- | -- | -- |
| E2E browser tests | Yes (basic) | -- | Yes | Yes | -- |
| Performance benchmarks | Yes (baseline) | -- | Yes | -- | Yes |
| Memory profiling | Yes (baseline) | -- | Yes (soak) | -- | Yes |
| Cross-platform builds | Yes | Yes | Yes | Yes | Yes |
| Integration tests (containers) | -- | -- | Yes | Yes | Yes |
| Fuzz testing | Yes (API inputs) | -- | Yes | Yes | -- |
| Upgrade tests | -- | -- | Yes | Yes | Yes |
| Multi-tenancy tests | -- | -- | Yes | -- | -- |
| Analytics algorithm tests | -- | -- | Yes | -- | -- |
| LLM integration tests | -- | -- | -- | Yes | -- |
| ONNX inference tests | -- | -- | -- | -- | Yes |
| MQTT integration tests | -- | -- | -- | -- | Yes |
