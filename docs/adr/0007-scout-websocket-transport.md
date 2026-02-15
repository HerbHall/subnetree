# ADR-0007: Scout WebSocket Transport for NAT Traversal

## Status

Proposed

## Date

2026-02-15

## Context

SubNetree's Scout agent communicates with the server via gRPC on a dedicated TCP port (`:9090`) with optional mTLS. The transport layer is implemented in two places:

- **Agent side:** [`internal/scout/agent.go`](../../internal/scout/agent.go) -- `dialGRPC()` establishes a `grpc.ClientConn` to `config.ServerAddr` (default `localhost:9090`), with exponential backoff reconnection and TLS/mTLS certificate management.
- **Server side:** [`internal/dispatch/dispatch.go`](../../internal/dispatch/dispatch.go) -- `Start()` opens a raw TCP listener on `GRPCAddr`, registers a `scoutServer` implementing the `ScoutService` protobuf service, and optionally configures mTLS with `VerifyClientCertIfGiven`.

The protobuf contract ([`api/proto/v1/scout.proto`](../../api/proto/v1/scout.proto)) defines four RPCs: `CheckIn` (unary), `ReportMetrics` (client-streaming), `CommandStream` (bidirectional streaming), and `ReportProfile` (unary). Of these, `CheckIn` and `ReportProfile` are implemented; the two streaming RPCs are defined but not yet wired.

This architecture works well for single-LAN homelab deployments (SubNetree's primary target: 10-75 devices on one network). However, it fails in three scenarios:

1. **Agents behind NAT** -- A Scout at a remote site cannot initiate a connection to the server's gRPC port if the server is also behind NAT or if the user cannot configure port forwarding.
2. **Restrictive firewalls** -- Corporate or ISP firewalls may block non-HTTP traffic on arbitrary ports. Port 9090 is not universally open.
3. **Docker/container networks** -- Agents inside Docker bridge networks cannot reach the host's gRPC port without explicit port mapping.

GitHub issue [#312](https://github.com/HerbHall/subnetree/issues/312) proposes WebSocket as an alternative transport to address these gaps. This ADR evaluates WebSocket against the current gRPC transport and against the planned Tailscale integration ([#278](https://github.com/HerbHall/subnetree/issues/278)).

### Existing WebSocket Usage

SubNetree already uses `coder/websocket` (v1.8.14, a direct dependency in `go.mod`) in two places:

- [`internal/ws/handler.go`](../../internal/ws/handler.go) -- Real-time scan event streaming to the dashboard via `GET /api/v1/ws/scan` over the existing HTTP server (`:8080`).
- [`internal/gateway/ssh.go`](../../internal/gateway/ssh.go) -- WebSocket-to-SSH bridge for browser-based terminal access via `GET /api/v1/ws/gateway/ssh/{device_id}`.

Both use JWT-based authentication via query parameter (`?token=`) since the browser WebSocket API does not support custom headers. This pattern is proven and already deployed.

## Decision

**Defer** WebSocket as a Scout agent transport. Do not implement it now. Revisit when either (a) remote-site monitoring becomes a documented user need, or (b) the Tailscale integration (#278) proves insufficient for NAT traversal.

### Rationale

**1. The problem is rare for the target audience.**

SubNetree targets homelab users with 10-75 devices, predominantly on a single LAN. In this deployment model, the server and all agents share a network segment. gRPC on `:9090` works without configuration. NAT traversal is needed only for multi-site monitoring, which is an advanced use case that no current user has requested.

**2. gRPC streaming is architecturally superior for agent communication.**

The `scout.proto` contract already defines `ReportMetrics` (client-streaming) and `CommandStream` (bidirectional streaming). These leverage gRPC's native streaming with automatic flow control, backpressure, and multiplexing over a single HTTP/2 connection. Reimplementing equivalent semantics over WebSocket would require:

- Custom message framing (protobuf over WebSocket binary frames)
- Manual flow control and backpressure
- Reconnection state management (which stream offsets were acknowledged)
- A second authentication path (JWT query params instead of mTLS client certificates)

This is substantial engineering effort to replicate what gRPC provides natively.

**3. Tailscale integration (#278) is a better NAT traversal solution.**

Tailscale creates a flat overlay network where every device gets a stable IP address regardless of NAT topology. This solves the NAT problem at the network layer rather than the application layer:

- No code changes to the Scout agent or Dispatch module
- mTLS continues to work unchanged (Tailscale adds WireGuard encryption, mTLS adds identity verification)
- Works for all protocols, not just agent check-ins (SSH bridge, future gRPC streams, SNMP polling)
- Already documented in [`docs/guides/tailscale-deployment.md`](../../docs/guides/tailscale-deployment.md) and [`docs/guides/tailscale-funnel.md`](../../docs/guides/tailscale-funnel.md)
- Tailscale is free for up to 100 devices (covers SubNetree's target range)

**4. The implementation cost is disproportionate to the benefit.**

Adding WebSocket transport would require:

- A transport abstraction layer in the Scout agent (currently tightly coupled to `grpc.ClientConn` in [`agent.go:33-34`](../../internal/scout/agent.go#L33-L34))
- A WebSocket-to-gRPC bridge or a parallel WebSocket handler in Dispatch
- Dual authentication paths (mTLS for gRPC, JWT for WebSocket)
- Configuration complexity (users choose between `grpc://`, `ws://`, `wss://` schemes)
- Test matrix doubling (every agent behavior tested on both transports)

For a problem that affects a small fraction of deployments, this cost is not justified.

**5. A simpler escape hatch already exists.**

Users needing NAT traversal today can use any of these approaches without code changes:

- **Reverse SSH tunnel:** `ssh -R 9090:localhost:9090 remote-server`
- **Tailscale/WireGuard VPN:** Flat network, no port forwarding needed
- **Cloudflare Tunnel / ngrok:** Expose `:9090` via HTTPS tunnel
- **Docker host networking:** `--network host` eliminates container NAT

## Consequences

### Positive

- **No additional complexity** -- The agent transport remains a single code path (gRPC), reducing test surface and debugging scenarios.
- **mTLS stays unified** -- Client certificate identity verification ([`grpc.go:65-76`](../../internal/dispatch/grpc.go#L65-L76)) does not need a parallel JWT-based identity path.
- **Focus on higher-value work** -- Engineering effort stays on features that benefit all users (monitoring, analytics, vault), not edge-case transport issues.
- **Tailscale path remains clean** -- When #278 ships, it solves NAT traversal universally without protocol-level workarounds.

### Negative

- **No built-in NAT traversal** -- Users with multi-site deployments must configure external tunneling (VPN, SSH tunnel, or Tailscale) to connect remote agents. This adds setup friction for that use case.
- **Port 9090 dependency** -- Restrictive firewalls that only allow HTTP/HTTPS traffic will block gRPC. Users in those environments have no agent transport option without external tunneling.
- **Perceived feature gap** -- Competitors that offer agent-initiated HTTP/WebSocket connections may appear more accessible for remote monitoring use cases.

### Neutral

- **WebSocket expertise is in-house** -- The existing `internal/ws/` and `internal/gateway/ssh.go` implementations demonstrate that the team can implement WebSocket transport if the need materializes. The decision to defer is based on prioritization, not capability.
- **Proto contract is stable** -- The `scout.proto` RPCs are transport-agnostic at the message level. If WebSocket transport is needed later, the same protobuf messages can be serialized over WebSocket frames without changing the service contract.
- **`gorilla/websocket` is also available** -- Present as an indirect dependency (via `paho.mqtt.golang`). If WebSocket transport is eventually built, either `coder/websocket` or `gorilla/websocket` could be used, though `coder/websocket` is preferred for consistency with existing code.

## Alternatives Considered

### Alternative 1: WebSocket Transport (Full Implementation)

Add a WebSocket endpoint (e.g., `GET /api/v1/ws/scout`) to the HTTP server that accepts agent connections, authenticates via enrollment token or JWT, and bridges messages to the existing Dispatch store logic. The agent would gain a `--transport ws` flag.

**Strengths:**

- Works through any HTTP-capable proxy or firewall
- Reuses the existing HTTP server port (`:8080`), eliminating the need for port `:9090`
- Agent-initiated connections naturally traverse NAT (outbound TCP from agent to server)
- `coder/websocket` already in the dependency tree

**Rejected because:**

- Duplicates gRPC functionality (streaming, reconnection, backpressure) in application code
- Creates a second authentication path that must be kept in sync with mTLS
- Doubles the transport test matrix
- Solves a problem that affects a small minority of the target audience
- Tailscale (#278) provides a cleaner network-layer solution

### Alternative 2: gRPC-Web Proxy

Run a gRPC-Web proxy (e.g., `improbable-eng/grpc-web` or Envoy) in front of the gRPC server. gRPC-Web encodes gRPC frames over HTTP/1.1 or HTTP/2, making them firewall-friendly.

**Strengths:**

- No agent code changes (gRPC-Web clients exist for Go)
- Maintains protobuf contract and streaming semantics
- Industry-standard approach for browser-to-gRPC communication

**Rejected because:**

- Adds an operational dependency (proxy sidecar) that conflicts with SubNetree's single-binary design philosophy
- gRPC-Web Go client support is immature compared to browser (JavaScript) clients
- Bidirectional streaming is not supported in gRPC-Web (only unary and server-streaming)
- `CommandStream` RPC requires full bidirectional streaming

### Alternative 3: Tailscale-Only (Issue #278)

Rely entirely on Tailscale integration for NAT traversal. Agents and server join the same Tailnet; Tailscale handles routing, NAT traversal, and encryption.

**Strengths:**

- Zero application-layer changes
- Adds WireGuard encryption on top of mTLS
- Free for up to 100 devices
- Already documented in deployment guides
- Solves NAT traversal for all protocols (gRPC, SSH bridge, SNMP, HTTP API)

**This is the recommended approach** when NAT traversal is needed. It is not "rejected" but rather the preferred path for users who need cross-network agent connectivity. The reason this ADR recommends deferring rather than adopting Alternative 3 right now is that Tailscale integration (#278) is its own separate implementation effort, and the current gRPC transport works for the primary LAN deployment scenario.

### Alternative 4: QUIC Transport

Replace gRPC-over-TCP with gRPC-over-QUIC (UDP-based, with built-in multiplexing and connection migration). QUIC handles NAT traversal via UDP hole punching and survives network changes (e.g., Wi-Fi to Ethernet) without reconnection.

**Strengths:**

- Connection migration eliminates reconnection churn
- UDP hole punching provides some NAT traversal
- Lower latency on lossy networks (no head-of-line blocking)

**Rejected because:**

- gRPC QUIC support in Go is experimental
- UDP may be blocked by firewalls that allow TCP
- Adds significant complexity to TLS configuration
- Benefits are marginal for LAN deployments with stable connections

## Revisit Triggers

This decision should be revisited if any of the following occur:

- **User demand:** Three or more users request remote-site agent connectivity without VPN
- **Tailscale limitations:** The Tailscale integration (#278) ships but proves insufficient (e.g., users cannot or will not install Tailscale on monitored devices)
- **Protocol consolidation:** A decision is made to eliminate the separate gRPC port (`:9090`) and run all communication through the HTTP server (`:8080`)
- **Commercial deployment:** Enterprise customers require agent connectivity through corporate proxies that only allow HTTPS
