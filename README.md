# NetVantage

[![Go Report Card](https://goreportcard.com/badge/github.com/HerbHall/netvantage)](https://goreportcard.com/report/github.com/HerbHall/netvantage)
[![codecov](https://codecov.io/gh/HerbHall/netvantage/branch/main/graph/badge.svg)](https://codecov.io/gh/HerbHall/netvantage)
[![License](https://img.shields.io/badge/license-BSL%201.1-blue)](LICENSE)

> **Alpha Status**: NetVantage is in active development. Core scanning and dashboard work, but expect rough edges. Contributions and feedback welcome!

**Your homelab command center.** NetVantage discovers devices on your network, monitors their status, and gives you one-click access to everything -- without typing passwords a thousand times a day.

## Why NetVantage?

Homelabbers juggle dozens of tools: UnRAID for storage, Proxmox for VMs, Home Assistant for automation, plus routers, NAS boxes, and random IoT devices. NetVantage doesn't replace any of them -- it's your **dashboard and aggregator** that:

- **Discovers everything** on your LAN automatically (ARP, ICMP, mDNS, SNMP)
- **Shows status at a glance** from multiple platforms in one place
- **Launches anything** with one click -- web UIs, SSH, RDP -- credentials handled
- **Extends via plugins** to monitor whatever you need

## Quick Start (Docker)

```bash
docker run -d --name netvantage \
  -p 8080:8080 \
  -v netvantage-data:/data \
  --cap-add NET_RAW --cap-add NET_ADMIN \
  ghcr.io/herbhall/netvantage:latest
```

Open <http://localhost:8080> -- first-time setup will prompt you to create an admin account.

For full network scanning capability on home networks:

```bash
docker run -d --name netvantage \
  --network host \
  -v netvantage-data:/data \
  ghcr.io/herbhall/netvantage:latest
```

### Docker Compose

```yaml
# docker-compose.yml
services:
  netvantage:
    image: ghcr.io/herbhall/netvantage:latest
    container_name: netvantage
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - netvantage-data:/data
    cap_add:
      - NET_RAW
      - NET_ADMIN

volumes:
  netvantage-data:
```

```bash
docker-compose up -d
```

## Features

### Discovery & Mapping

- LAN scanning with ARP, ICMP, mDNS, SNMP, UPnP
- Automatic device identification (OS, manufacturer, type)
- Network topology visualization

### Monitoring

- Device health and status tracking
- Optional Scout agents for detailed host metrics
- Plugin-extensible -- monitor anything

### Quick Access

- One-click launch to web UIs, SSH, RDP, VNC
- Credential vault so you don't re-type passwords
- Secure browser-based remote access (coming soon)

## Architecture

```
                    +------------------+
                    |    Dashboard     |
                    | (React + TS)     |
                    +--------+---------+
                             |
                    REST / WebSocket
                             |
+----------+       +---------+---------+       +----------+
|  Scout   | gRPC  |   NetVantage      |       | Network  |
|  Agent   +------>+   Server          +------>+ Devices  |
|          |       |                   | ICMP/  | (SNMP,   |
+----------+       | +------+ +------+ | SNMP/  |  mDNS,   |
                   | |Recon | |Pulse | | ARP/   |  UPnP)   |
                   | +------+ +------+ | mDNS   +----------+
                   | |Dispatch|Vault | |
                   | +------+ +------+ |
                   | |Gateway|        |
                   | +------+         |
                   +-------------------+
```

### Modules

| Module | Description |
| --- | --- |
| **Recon** | Network scanning and device discovery |
| **Pulse** | Health monitoring, metrics, alerting |
| **Dispatch** | Scout agent enrollment and management |
| **Vault** | Encrypted credential storage |
| **Gateway** | Browser-based remote access (SSH, RDP, HTTP proxy) |

## Building from Source

### Prerequisites

- Go 1.25+
- Node.js 22+ (for frontend)
- Make (optional)

### Build

```bash
# Build everything (server + frontend)
make build

# Or build separately
make build-server
cd web && npm ci && npm run build
```

### Run

```bash
# Start server (serves dashboard at :8080)
./bin/netvantage serve

# With config file
./bin/netvantage serve -config configs/netvantage.example.yaml
```

### Development

```bash
# Run tests
make test

# Run linter
make lint

# Generate protobuf
make proto
```

## Project Structure

```text
cmd/
  netvantage/     Server entry point
  scout/          Agent entry point
internal/
  recon/          Network discovery module
  pulse/          Monitoring module
  dispatch/       Agent management module
  vault/          Credential management module
  gateway/        Remote access module
web/              React dashboard (Vite + shadcn/ui)
pkg/
  plugin/         Public plugin SDK (Apache 2.0)
  models/         Shared data types
api/
  proto/v1/       gRPC service definitions
```

## Roadmap

- **Phase 1** (Current): Server + dashboard + agentless scanning
- **Phase 1b**: Windows Scout agent
- **Phase 2**: Enhanced discovery (SNMP, mDNS, UPnP) + monitoring + Linux agent
- **Phase 3**: Remote access (SSH, RDP) + credential vault
- **Phase 4**: Homelab integrations (Home Assistant, UnRAID, Proxmox)

## Support the Project

NetVantage is **free for personal, homelab, and non-competing production use**. If you find it useful:

- [GitHub Sponsors](https://github.com/sponsors/HerbHall)
- [Ko-fi](https://ko-fi.com/herbhall)
- [Buy Me a Coffee](https://buymeacoffee.com/herbhall)

You can also contribute by [reporting bugs](https://github.com/HerbHall/netvantage/issues), [requesting features](https://github.com/HerbHall/netvantage/discussions), testing alpha releases, or building plugins.

## License

NetVantage uses a split licensing model:

- **Core** (server, agent, built-in modules): [Business Source License 1.1](LICENSE) -- free for personal, homelab, educational, and non-competing production use. Converts to Apache 2.0 after 4 years.
- **Plugin SDK** (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`, `api/proto/`): [Apache License 2.0](pkg/plugin/LICENSE) -- build plugins and integrations with no restrictions.

See [LICENSING.md](LICENSING.md) for full details.
