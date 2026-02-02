## Scout Agent Specification

### Purpose

Lightweight agent installed on monitored devices to report system metrics, accept commands, and facilitate remote access.

### Capabilities

- System metrics: CPU, memory, disk, network usage
- Process listing
- Service status monitoring
- Log forwarding (opt-in)
- Command execution (authorized commands only)
- Auto-update (see Auto-Update Mechanism below)

### Communication

- gRPC with mTLS to server
- Periodic check-in (configurable interval, default 30s)
- Bidirectional streaming for real-time commands
- Exponential backoff reconnection (1s, 2s, 4s, 8s... max 5 minutes)

### Certificate Management

- Server runs an internal CA for mTLS
- Agent enrollment: token-based + certificate signing request
- Per-agent certificates with 90-day validity
- Auto-renewal at day 60
- Certificate revocation list for decommissioned agents

### Resource Constraints

The Scout agent must run unobtrusively on the host system, including resource-constrained devices:

| Resource | Target | Minimum Platform |
|----------|--------|-----------------|
| Binary size | < 15 MB (statically linked) | All |
| Memory (idle) | < 10 MB | Raspberry Pi 3B+ (1 GB RAM) |
| Memory (active collection) | < 20 MB | Raspberry Pi 3B+ (1 GB RAM) |
| CPU (idle) | < 1% | All |
| CPU (active collection) | < 5% | Raspberry Pi 3B+ (quad-core ARM) |
| Disk (binary + data + logs) | < 50 MB | All |
| Network (check-in) | < 1 KB per check-in | All |

### Platforms

| Platform | Priority | Architecture | Method |
|----------|----------|-------------|--------|
| Windows x64 | Phase 1b | x86_64 | Native Go binary, Windows service |
| Linux x64 | Phase 2 | x86_64 | Native Go binary, systemd unit |
| Linux ARM64 | Phase 2 | ARM64 (aarch64) | Cross-compiled Go binary, systemd unit |
| Linux ARM | Phase 2 | ARMv7 (armhf) | Cross-compiled Go binary (Raspberry Pi 3B+, older SBCs) |
| macOS ARM64 | Phase 3 | ARM64 (Apple Silicon) | Native Go binary, launchd plist |
| macOS x64 | Phase 3 | x86_64 | Native Go binary, launchd plist |
| Android | Deferred | ARM64 | Passive monitoring only (ping, ARP, mDNS) |
| IoT/Embedded | Phase 4 | Various | Lightweight Go binary or MQTT-based |

### Auto-Update Mechanism (Phase 2)

Agent auto-update is a security-critical feature. The SolarWinds supply chain attack demonstrated the risk of compromised update channels.

#### Update Flow

1. Agent polls server for available updates during check-in (configurable: enabled/disabled, channel)
2. Server responds with version info + signed manifest if update available
3. Agent downloads binary from server, verifies Cosign signature against pinned public key
4. Agent validates binary integrity (SHA-256 checksum from signed manifest)
5. Agent installs update (platform-specific: replace binary, restart service)
6. Agent reports new version on next check-in; server marks update as successful
7. If agent fails to check in within expected window after update, server marks update as failed

#### Controls

- **Administrator approval:** Updates require explicit approval per version in the server UI before any agent receives them
- **Staged rollout:** Configurable: update N% of agents, wait for health confirmation, then proceed (default: 10% canary, 24h wait)
- **Version pinning:** Administrators can pin individual agents or agent groups to a specific version
- **Update channels:** `stable` (default), `beta`, `pinned` (manual only)
- **Rollback:** Agent retains previous binary. Automatic rollback if health check fails within 5 minutes of update
- **Air-gapped support:** Manual update package (signed binary + manifest) for offline environments

### Security

- Agent authenticates to server via enrollment token + mTLS certificate
- Server issues per-agent certificates during enrollment
- Commands require server-side authorization
- Agent binary is source-available (BSL 1.1) for user trust and auditability
- Per-agent rate limiting in gRPC interceptor
- Update binaries signed with Cosign; agent verifies before applying

### Agent-Server Version Compatibility

The server and agent each carry a SemVer version string (e.g., `1.3.2`). Compatibility is determined by the **gRPC protocol version** (`proto_version` integer in `CheckInRequest`), not by comparing version strings directly. This decouples release cadence from protocol compatibility.

#### Compatibility Table

| Agent Proto Version | Server Proto Version | Result |
|---------------------|---------------------|--------|
| Same | Same | Full compatibility |
| Older (N-1) | Current (N) | Supported -- server handles old message format |
| Older (< N-1) | Current (N) | Rejected -- agent must update |
| Newer than server | Any | Rejected -- agent must not be newer than server |

**Rule:** Always upgrade the server first, then agents. The server supports the current proto version and one version behind (N and N-1). Agents more than one proto version behind are rejected with an explicit upgrade instruction.

#### Version Negotiation Protocol

The `CheckInRequest` message carries version metadata:

```protobuf
message CheckInRequest {
  string agent_id = 1;
  string hostname = 2;
  string platform = 3;
  string agent_version = 4;   // SemVer string, e.g., "1.3.2"
  SystemMetrics metrics = 5;
  uint32 proto_version = 6;   // gRPC protocol version integer
}

message CheckInResponse {
  bool acknowledged = 1;
  int32 check_interval_seconds = 2;
  repeated string pending_commands = 3;
  VersionStatus version_status = 4;     // Compatibility result
  string server_version = 5;            // Server SemVer for diagnostics
  string upgrade_message = 6;           // Human-readable instruction (if rejected/deprecated)
}

enum VersionStatus {
  VERSION_OK = 0;              // Fully compatible
  VERSION_DEPRECATED = 1;     // Works now, will stop working in next server major
  VERSION_REJECTED = 2;       // Incompatible, check-in rejected
  VERSION_UPDATE_AVAILABLE = 3; // Compatible, but newer agent version exists
}
```

**Server behavior on check-in:**

1. Parse `proto_version` from `CheckInRequest`.
2. If `proto_version > server_proto_version`: respond with `VERSION_REJECTED`, message: "Agent proto version %d is newer than server proto version %d. Downgrade the agent or upgrade the server."
3. If `proto_version < server_proto_version - 1`: respond with `VERSION_REJECTED`, message: "Agent proto version %d is too old. Minimum supported: %d. Update the agent to continue."
4. If `proto_version == server_proto_version - 1`: respond with `VERSION_DEPRECATED`, process check-in normally, message: "Agent proto version %d is deprecated. Update before the next server major release."
5. If `proto_version == server_proto_version`: respond with `VERSION_OK`, process check-in normally.
6. In all cases, check if a newer agent binary is available and set `VERSION_UPDATE_AVAILABLE` when applicable (only when the agent is otherwise compatible).

**Rejected agents:** When `VERSION_REJECTED`, the server logs the event, does NOT process metrics or commands, and returns the response with `acknowledged = false`. The agent should log the `upgrade_message` and continue retrying at a reduced interval (5 minutes) in case the server is upgraded.
