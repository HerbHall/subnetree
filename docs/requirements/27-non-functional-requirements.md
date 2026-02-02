## Non-Functional Requirements

The ordering below is intentional. **Stability and security come first** -- before performance, before features, before convenience. A monitoring tool that is itself unstable or insecure is worse than no monitoring tool at all.

### Stability

- The server must run unattended for months without intervention, memory leaks, or degradation.
- Plugin failures must be isolated -- a crashing plugin must never take down the server or other plugins.
- Database corruption must be prevented through proper WAL mode, checkpointing, and backup capabilities.
- All background operations (scan jobs, metrics collection, event processing) must have timeouts and circuit breakers.
- Graceful degradation over hard failure: if a subsystem is unhealthy, the rest of the system continues operating.

### Performance

**Server:**
- Handles 1,000+ devices with < 100ms API response times (large/enterprise profile)
- Base memory footprint < 50 MB with zero devices monitored
- Memory usage scales linearly: < 200 MB at 50 devices, < 500 MB at 200 devices, < 1.2 GB at 500 devices
- Startup time < 5 seconds on micro profile, < 10 seconds on large profile
- Network scan of /24 subnet completes in < 30 seconds
- Dashboard loads in < 2 seconds
- Topology map renders smoothly with 500+ devices (progressive rendering for larger networks)
- On micro profile (RPi 4/5): < 200 MB total memory, < 25% CPU during active scan of 25 devices

**Agent (Scout):**
- Binary size < 15 MB (statically linked)
- CPU usage < 1% idle, < 5% during metric collection
- Memory usage < 20 MB on x64, < 25 MB on ARM64
- Runs on Raspberry Pi 3B+ (1 GB RAM, ARMv8) as the minimum target platform
- Disk usage < 50 MB including binary + data + logs

### Security

#### Transport & Encryption
- All agent communication encrypted (mTLS)
- Credentials encrypted at rest (AES-256-GCM envelope encryption)
- TLS 1.2+ enforced for all external connections (HTTPS, gRPC)

#### Authentication & Access Control
- No default credentials (first-run wizard enforces account creation)
- API authentication required (JWT tokens)
- Password policy: minimum 12 characters, checked against breached password list (HaveIBeenPwned k-anonymity API, optional)
- Account lockout: progressive delay after failed login attempts (5 failures = 15 minute lockout)
- Session management: concurrent session limit per user (configurable, default: 5)
- MFA/TOTP: planned for Phase 2 (TOTP at minimum, WebAuthn stretch goal)

#### Web Security
- CORS properly configured (same-origin in production, configurable for dev)
- CSRF protection: SameSite=Strict cookies + custom `X-Requested-With` header validation
- Security headers served by Go HTTP server:
  - `Content-Security-Policy` (restrictive CSP for the SPA)
  - `X-Frame-Options: DENY`
  - `X-Content-Type-Options: nosniff`
  - `Strict-Transport-Security` (when TLS enabled)
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Permissions-Policy` (disable unnecessary browser APIs)
- Input validation at all API boundaries
- Rate limiting on all endpoints

#### Audit & Compliance
- Credential access audit logging
- Secrets hygiene: credentials must never appear in logs, error messages, or API responses
- OWASP Top 10 awareness in all development
- Vulnerability disclosure process documented in SECURITY.md
- **Compliance alignment:** Designed with SOC 2 Type II control categories in mind (access control, encryption, audit logging, change management). Not claiming certification, but signaling security maturity to evaluators and acquirers.

### Deployment

- Single binary server (Go, embeds web assets and migrations)
- Single binary agent (Go, cross-compiled)
- Docker Compose for full stack (server + Guacamole)
- Configuration via YAML file + environment variables
- Deployment profiles for common use cases

### Reliability

See also: **Stability** (above) for the foundational stability requirements.

- Graceful shutdown on SIGTERM/SIGINT with per-plugin timeout
- Automatic agent reconnection with exponential backoff
- Database migrations via embedded SQL (per-plugin, tracked, forward-only)
- Liveness and readiness health check endpoints
- Plugin graceful degradation (optional plugin failure doesn't crash server)
- SQLite WAL mode for concurrent read/write access
- Automatic WAL checkpointing to prevent unbounded WAL growth

### Observability

- Structured logging via Zap (configurable level and format)
- Prometheus metrics at `/metrics`
- Request tracing via `X-Request-ID` headers
- Per-plugin health status in readiness endpoint
- OpenTelemetry tracing support (Phase 2)
