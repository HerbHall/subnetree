# API Design

## Standards

- **Error responses:** RFC 7807 Problem Details (`application/problem+json`)
- **Pagination:** Cursor-based with `PaginatedResponse<T>` wrapper
- **Versioning:** URL path versioning (`/api/v1/`). Version is the first path segment after `/api/`. The versioning policy is:

  **REST API Versioning Rules:**
  - Only the **major** component appears in the URL path (`/api/v1/`, `/api/v2/`).
  - Minor and patch releases add fields, endpoints, or fix bugs without changing the path.
  - **Additive changes** (new fields in responses, new optional query parameters, new endpoints) are NOT breaking and do NOT require a new API version.
  - **Breaking changes** (removing/renaming fields, changing response structure, removing endpoints, changing authentication) require a new API version.
  - **Deprecation timeline:** When `/api/v2/` is introduced, `/api/v1/` continues to work for a minimum of **6 months** (two minor release cycles). Deprecated API versions return a `Sunset` header (RFC 8594) and a `Deprecation` header on every response.
  - **Version response header:** All API responses include `X-SubNetree-Version: {server_version}` (e.g., `X-SubNetree-Version: 1.3.2`). This enables clients to detect server version without a dedicated endpoint.
  - **Maximum concurrent API versions:** 2 (current + one prior). No more than two URL path versions served simultaneously.
  - **Health and metrics endpoints** (`/healthz`, `/readyz`, `/metrics`) are unversioned -- they are not part of the API contract.

- **Rate limiting:** Per-IP using `golang.org/x/time/rate`; per-tenant rate limiting in Phase 2
- **Documentation:** OpenAPI 3.0 via `swaggo/swag` annotations
- **Request tracing:** `X-Request-ID` header (generated if not provided)
- **Idempotency:** `Idempotency-Key` header supported on POST endpoints (device creation, credential storage) for safe retries. Server stores key-to-response mapping for 24 hours.
- **Conditional requests:** `ETag` + `If-None-Match` on GET endpoints for client-side cache validation. Reduces bandwidth for polling clients.

## Error Response Format

```json
{
  "type": "https://subnetree.io/problems/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "Device with ID 'abc-123' does not exist",
  "instance": "/api/v1/devices/abc-123"
}
```

## Pagination Format

```json
{
  "data": [...],
  "pagination": {
    "total": 142,
    "limit": 50,
    "next_cursor": "base64encoded",
    "has_more": true
  }
}
```

## REST API

Base path: `/api/v1/`

### Core Endpoints

| Endpoint | Method | Description |
| -------- | ------ | ----------- |
| `/healthz` | GET | Liveness probe (always 200 if process is alive) |
| `/readyz` | GET | Readiness probe (checks DB, plugin health) |
| `/metrics` | GET | Prometheus metrics |
| `/api/v1/health` | GET | Readiness (alias for backward compat) |
| `/api/v1/plugins` | GET | List loaded plugins with status |
| `/api/v1/plugins/{name}/enable` | POST | Enable a plugin at runtime |
| `/api/v1/plugins/{name}/disable` | POST | Disable a plugin at runtime |

### Auth Endpoints

| Endpoint | Method | Description |
| -------- | ------ | ----------- |
| `/api/v1/auth/login` | POST | Authenticate, returns JWT pair |
| `/api/v1/auth/refresh` | POST | Refresh access token |
| `/api/v1/auth/logout` | POST | Revoke refresh token |
| `/api/v1/auth/setup` | POST | First-run: create admin account |
| `/api/v1/auth/oidc/callback` | GET | OIDC callback handler |
| `/api/v1/users` | GET | List users (admin only) |
| `/api/v1/users/{id}` | GET/PUT/DELETE | User management (admin only) |

### Device Endpoints

| Endpoint | Method | Description |
| -------- | ------ | ----------- |
| `/api/v1/devices` | GET | List devices (paginated, filterable) |
| `/api/v1/devices/{id}` | GET | Device details with related data |
| `/api/v1/devices` | POST | Create device manually |
| `/api/v1/devices/{id}` | PUT | Update device |
| `/api/v1/devices/{id}` | DELETE | Remove device |
| `/api/v1/devices/{id}/topology` | GET | Device's topology connections |

### Plugin Endpoints (mounted under `/api/v1/{plugin-name}/`)

| Endpoint | Method | Plugin | Description |
| -------- | ------ | ------ | ----------- |
| `/recon/scan` | POST | Recon | Trigger network scan |
| `/recon/scans` | GET | Recon | List scan history |
| `/recon/topology` | GET | Recon | Full topology graph |
| `/pulse/status` | GET | Pulse | Overall monitoring status |
| `/pulse/alerts` | GET | Pulse | List active/recent alerts |
| `/pulse/alerts/{id}/ack` | POST | Pulse | Acknowledge an alert |
| `/pulse/metrics/{device_id}` | GET | Pulse | Device metrics with time range |
| `/dispatch/agents` | GET | Dispatch | List connected agents |
| `/dispatch/agents/{id}` | GET | Dispatch | Agent details |
| `/dispatch/enroll` | POST | Dispatch | Generate enrollment token |
| `/vault/credentials` | GET | Vault | List credentials (metadata only) |
| `/vault/credentials` | POST | Vault | Store new credential |
| `/vault/credentials/{id}` | GET | Vault | Credential metadata |
| `/vault/credentials/{id}` | DELETE | Vault | Delete credential |
| `/gateway/sessions` | GET | Gateway | List active remote sessions |
| `/gateway/ssh/{device_id}` | WebSocket | Gateway | SSH terminal session |
| `/gateway/rdp/{device_id}` | WebSocket | Gateway | RDP session (via Guacamole) |
| `/gateway/proxy/{device_id}` | ANY | Gateway | HTTP reverse proxy to device |

## WebSocket Connection

- **Endpoint:** `GET /ws/` (upgrades to WebSocket)
- **Authentication:** JWT token sent in the first message after connection (not in URL query params, which leak in server logs and browser history)
- **Protocol:** JSON messages with `{ "type": "...", "payload": { ... } }` envelope
- **Reconnection:** Client implements exponential backoff (1s, 2s, 4s... max 30s) with jitter
- **Heartbeat:** Server sends `ping` every 30s; client responds with `pong`. Connection closed after 3 missed pongs.

## WebSocket Events (Dashboard Real-Time)

| Event | Direction | Description |
| ----- | --------- | ----------- |
| `device.discovered` | Server -> Client | New device found during scan |
| `device.status_changed` | Server -> Client | Device status update |
| `scan.progress` | Server -> Client | Scan completion percentage |
| `scan.completed` | Server -> Client | Scan finished |
| `alert.triggered` | Server -> Client | New alert |
| `alert.resolved` | Server -> Client | Alert cleared |
| `agent.connected` | Server -> Client | Agent came online |
| `agent.disconnected` | Server -> Client | Agent went offline |

## gRPC Services (Agent Communication)

```protobuf
service ScoutService {
  rpc Enroll(EnrollRequest) returns (EnrollResponse);
  rpc CheckIn(CheckInRequest) returns (CheckInResponse);
  rpc ReportMetrics(stream MetricsReport) returns (Ack);
  rpc CommandStream(stream CommandResponse) returns (stream Command);
  rpc RenewCertificate(CertRenewalRequest) returns (CertRenewalResponse);
}
```

**gRPC API Versioning Policy:**

- **Proto package versioning:** Proto definitions live in `api/proto/v1/` with package `subnetree.v1`. A breaking change creates `api/proto/v2/` with package `subnetree.v2`.
- **Breaking changes in gRPC** include: removing or renaming fields, changing field numbers, changing field types, removing RPC methods, changing streaming semantics (unary to streaming or vice versa).
- **Non-breaking changes** include: adding new fields (proto3 handles unknown fields gracefully), adding new RPC methods, adding new enum values.
- **Proto version integer:** Each proto package version has a corresponding integer (`proto_version`) sent in `CheckInRequest`. This enables version negotiation without parsing proto package names at runtime (see Agent-Server Version Compatibility).
- **`buf breaking` enforcement:** The `buf` toolchain runs breaking-change detection in CI against the previous tagged release. Any breaking change fails the build unless the proto package version is incremented.
- **Backward compatibility guarantee:** The server supports the current proto version and one version behind (N and N-1). This matches the agent-server compatibility rule.
- **gRPC metadata:** The server sets `x-subnetree-version` in gRPC response metadata (trailing headers) for diagnostic purposes.

## Rate Limits

| Endpoint Pattern | Rate | Burst | Reason |
| ---------------- | ---- | ----- | ------ |
| General API | 100/s | 200 | Dashboard makes parallel requests |
| `POST /recon/scan` | 1/min | 2 | Scans are expensive network operations |
| `POST /vault/credentials` | 10/s | 20 | Security-sensitive |
| `POST /auth/login` | 5/min | 10 | Brute force protection |
| `/healthz`, `/readyz`, `/metrics` | Unlimited | -- | Orchestrator/monitoring probes |
