## Architecture Overview

### Components

| Component | Name | Description |
|-----------|------|-------------|
| Server | **NetVantage** | Central application: HTTP API, plugin registry, data storage, web dashboard |
| Agent | **Scout** | Lightweight Go agent installed on monitored devices |
| Dashboard | *web/* | React + TypeScript SPA served by the server |

### Server Modules (Plugins)

Each module fills one or more **roles** (abstract capabilities). Alternative implementations can replace any built-in module by implementing the same role interface.

| Module | Name | Role | Purpose |
|--------|------|------|---------|
| Discovery | **Recon** | `discovery` | Network scanning, device discovery (ICMP, ARP, SNMP, mDNS, UPnP, SSDP) |
| Monitoring | **Pulse** | `monitoring` | Health checks, uptime monitoring, metrics collection, alerting |
| Agent Management | **Dispatch** | `agent_management` | Scout agent enrollment, check-in, command dispatch, status tracking |
| Credentials | **Vault** | `credential_store` | Encrypted credential storage, per-device credential assignment |
| Remote Access | **Gateway** | `remote_access` | Browser-based SSH, RDP (via Guacamole), HTTP/HTTPS reverse proxy, VNC |
| Overlay Network | **Tailscale** | `overlay_network` | Tailscale tailnet device discovery, overlay IP enrichment, subnet route awareness |

### Communication

- **Server <-> Dashboard:** REST API + WebSocket (real-time updates)
- **Server <-> Scout:** gRPC with mTLS (bidirectional streaming)
- **Server <-> Network Devices:** ICMP, ARP, SNMP v2c/v3, mDNS, UPnP/SSDP, MQTT
- **Server <-> Tailscale API:** HTTPS REST (device enumeration, subnet routes, DNS)

### Module Dependency Graph

```
Vault (no deps, provides credential_store)
  |
  +---> Recon (optional: credential_store for authenticated scanning)
  |       |
  |       +---> Pulse (requires: discovery for device list)
  |       +---> Gateway (requires: discovery + optional credential_store)
  |
Dispatch (no deps, provides agent_management)
  |
  +---> Pulse (optional: agent_management for agent metrics)
  +---> Recon (optional: agent_management for agent-assisted scans)

Tailscale (requires: credential_store for API key/OAuth storage)
  |
  +---> Recon (optional: overlay_network for Tailscale-discovered devices)
  +---> Gateway (optional: overlay_network for Tailscale IP connectivity)
```

**Topological Startup Order:** Vault -> Dispatch -> Tailscale -> Recon -> Pulse -> Gateway
**Shutdown Order (reverse):** Gateway -> Pulse -> Recon -> Tailscale -> Dispatch -> Vault
