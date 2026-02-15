# SubNetree Documentation

Modular, source-available network monitoring and management platform for homelabs and small businesses.

**Free for personal/home use forever. Commercial for business use.**

---

## Quick Navigation

<div class="grid cards" markdown>

- :material-rocket-launch: **[Getting Started](guides/getting-started.md)**

    Deploy SubNetree and discover your first devices in minutes.

- :material-server-network: **[Deployment Guides](guides/tailscale-deployment.md)**

    Production deployment with Tailscale, Docker, and platform-specific notes.

- :material-wrench: **[Troubleshooting](guides/troubleshooting.md)**

    Common issues and solutions for installation, scanning, and connectivity.

- :material-file-document: **[Requirements](requirements/README.md)**

    Complete product requirements and architectural specifications.

</div>

## Features

- **Network Discovery** -- Automatic device detection via ARP, mDNS, and SNMP scanning
- **Real-Time Monitoring** -- ICMP, TCP, and HTTP health checks with configurable alerting
- **AI-Powered Analytics** -- Natural language queries and anomaly detection via pluggable LLM providers
- **Credential Vault** -- AES-256-GCM encrypted storage for SSH keys, SNMP credentials, and API tokens
- **Scout Agents** -- Lightweight agents for remote device profiling and metrics collection
- **Network Topology** -- Interactive visualization of device relationships and connectivity
- **Plugin Architecture** -- Every major feature is a plugin; extend or replace any module

## Architecture

SubNetree follows a modular plugin architecture with clear separation of concerns:

| Component | Description |
|-----------|-------------|
| **Server** (`cmd/subnetree/`) | Central application with HTTP API and plugin registry |
| **Scout** (`cmd/scout/`) | Lightweight agent installed on monitored devices |
| **Dashboard** (`web/`) | React + TypeScript SPA served by the server |
| **Modules** (`internal/`) | Recon, Pulse, Dispatch, Vault, Gateway, Insight |
| **Plugin SDK** (`pkg/`) | Public interfaces, Apache 2.0 licensed |

See the [Architecture Overview](requirements/02-architecture-overview.md) and [ADR records](adr/0001-split-licensing-model.md) for detailed design decisions.

## License

- **Core**: BSL 1.1 (converts to Apache 2.0 after 4 years)
- **Plugin SDK**: Apache 2.0
