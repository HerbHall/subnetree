# ADR-0004: Integer-Based Protocol Versioning

## Status

Accepted

## Date

2025-01-01

## Context

NetVantage has multiple versioned interfaces between components:

- **Plugin API:** Server ↔ built-in and third-party plugins
- **Agent protocol:** Server ↔ Scout agents (gRPC)
- **REST API:** Server ↔ Dashboard and external clients
- **Configuration format:** YAML config files across versions

Each interface needs version compatibility checking to prevent silent failures when components are mismatched. The versioning scheme must:

- Be simple to compare programmatically (no string parsing edge cases)
- Support N-1 compatibility windows (current + one prior version)
- Produce clear error messages on mismatch
- Work in both gRPC protobuf messages and Go code

## Decision

Use integer-based versioning for protocol compatibility and SemVer strings for human-facing display.

**Plugin API:**
- `PluginAPIVersionMin` and `PluginAPIVersionCurrent` constants in the server
- Each plugin declares `APIVersion int` in its `PluginInfo`
- Validation: `PluginAPIVersionMin <= plugin.APIVersion <= PluginAPIVersionCurrent`
- Rejected plugins get explicit error messages with version ranges

**Agent-Server protocol:**
- `proto_version uint32` field in gRPC `CheckInRequest`
- `VersionStatus` enum in `CheckInResponse`: `VERSION_OK`, `VERSION_DEPRECATED`, `VERSION_REJECTED`, `VERSION_UPDATE_AVAILABLE`
- Server supports N and N-1 protocol versions
- Rejected agents receive `upgrade_message` with download URL

**REST API:**
- Path-based major version: `/api/v1/`, `/api/v2/`
- Maximum 2 concurrent API versions
- `X-NetVantage-Version` response header on all responses
- `Sunset` + `Deprecation` headers per RFC 8594 during deprecation

**Configuration:**
- `config_version` integer at YAML root
- Server refuses to start if config version is newer than supported
- `netvantage config migrate` CLI command for automatic migration

**Human-facing versions** (server binary, agent binary, SDK) use SemVer 2.0.0 strings injected at build time via ldflags.

## Consequences

### Positive

- Integer comparison is trivial: `>=` and `<=`, no parsing, no edge cases
- Protocol version increments only on breaking changes (not on every release)
- Clear separation: integers for machine compatibility, SemVer for human display
- `VersionStatus` enum gives agents explicit instructions (retry, upgrade, continue)
- N-1 window limits testing burden to 2 versions

### Negative

- Two versioning schemes to maintain (integer protocol + SemVer display)
- Integer versions don't convey "how big" a change is (no MAJOR.MINOR.PATCH semantics)
- N-1 window means agents more than one major protocol version behind are hard-rejected

### Neutral

- Built-in plugins always match the server version (they ship together) so version checks are mainly for future third-party plugins
- `config_version` changes are rare -- most config additions are backwards-compatible
- REST API version bumps (v1 → v2) are expected to be very infrequent

## Alternatives Considered

### Alternative 1: SemVer String Comparison Everywhere

Use SemVer strings for all version checks. Requires string parsing and comparison logic in every component. SemVer comparison has edge cases (pre-release ordering, build metadata). More complex for the same result.

### Alternative 2: No Explicit Version Checking

Let components fail naturally on incompatibility. This leads to cryptic runtime errors, silent data corruption, and difficult debugging. Unacceptable for a product that manages network infrastructure.

### Alternative 3: Capability-Based Negotiation

Instead of version numbers, advertise specific capabilities. More flexible but significantly more complex to implement and test. Overkill for our current component count. Could be added later if the protocol grows complex.
