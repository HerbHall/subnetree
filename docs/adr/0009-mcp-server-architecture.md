# 9. MCP server architecture for AI tool integration

Date: 2026-02-21

## Status

Accepted

## Context

AI tools such as Claude Desktop, Cursor, and other MCP-compatible clients need
structured access to SubNetree device inventory, hardware profiles, and fleet
summaries. Without a standard integration protocol, each tool would require a
custom API adapter or plugin.

The Model Context Protocol (MCP) is an open standard created by Anthropic for
connecting AI assistants to external data sources and tools. It supports two
transport modes: HTTP (for web/server integrations) and stdio (for desktop
applications like Claude Desktop). An official Go SDK
(`github.com/modelcontextprotocol/go-sdk`) is available.

SubNetree already exposes device data through REST endpoints, but MCP provides
a tool-calling interface specifically designed for AI consumption -- structured
input schemas, tool descriptions, and discovery -- which REST alone does not
offer.

## Decision

Implement an MCP server as a SubNetree plugin module (`internal/mcp/`) with
the following design:

1. **Plugin architecture:** The MCP module implements `plugin.Plugin` and
   `plugin.HTTPProvider`, following the same lifecycle as all other modules.

2. **Consumer-side interface:** A `DeviceQuerier` interface defined in the MCP
   package abstracts device data access. The recon module's store satisfies this
   interface. Wiring happens in `cmd/subnetree/main.go` via `SetQuerier()`,
   maintaining decoupling between internal packages.

3. **Dual transport:**
   - **HTTP** at `/api/v1/mcp/` using `StreamableHTTPHandler` from the Go SDK.
     Supports optional API key authentication via Bearer token.
   - **stdio** via `subnetree mcp` CLI subcommand for Claude Desktop integration.
     The server runs as a child process, communicating over stdin/stdout.

4. **Five tools exposed:**
   - `get_device` -- retrieve a single device by ID
   - `list_devices` -- paginated device listing
   - `get_hardware_profile` -- hardware details for a device
   - `get_fleet_summary` -- aggregate fleet statistics
   - `query_devices` -- hardware-based device queries with filters

5. **Security:** Vault credentials are never exposed through MCP tools. An
   optional API key protects the HTTP transport. Tool invocations emit
   `mcp.tool.called` events on the event bus for observability.

6. **Dependency:** Uses `github.com/modelcontextprotocol/go-sdk` (the official
   Go MCP SDK maintained by the MCP project).

## Consequences

### Positive

- AI tools gain structured, discoverable access to SubNetree device data
- Dual transport covers both server (HTTP) and desktop (stdio) use cases
- Consumer-side `DeviceQuerier` interface maintains package decoupling
- Plugin architecture means MCP can be disabled without affecting other modules
- Event bus integration provides audit trail for tool calls

### Negative

- New external dependency on `github.com/modelcontextprotocol/go-sdk`
- MCP protocol is relatively new; SDK may have breaking changes
- HTTP transport adds another authenticated endpoint to maintain

### Neutral

- The module depends on `recon` for device data, matching the existing
  dependency pattern (Pulse also depends on Recon)
- Future tools (service inventory, stale devices, audit log) can be added
  incrementally without architectural changes

## Alternatives Considered

### Alternative 1: Custom REST endpoints for AI tools

Expose dedicated REST endpoints with AI-friendly response formats. Rejected
because each AI tool would need a custom integration adapter. MCP provides a
standard protocol that works across all MCP-compatible clients without
per-tool customization.

### Alternative 2: GraphQL endpoint

Add a GraphQL API alongside REST for flexible querying. Rejected because
GraphQL solves a different problem (client-specified queries) and does not
provide tool discovery, input schemas, or the stdio transport that desktop
AI tools require. The implementation complexity is also significantly higher.

### Alternative 3: Third-party MCP proxy

Run a separate MCP proxy process that wraps SubNetree's REST API. Rejected
because it adds operational complexity (another process to manage), increases
latency (extra network hop), and loses access to internal data that the
REST API may not expose.
