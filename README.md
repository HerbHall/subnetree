# NetVantage

Modular, self-hosted network monitoring and management platform.

NetVantage provides unified device discovery, monitoring, remote access, credential management, and IoT awareness in a single application.

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

## Modules

| Module | Description |
|--------|-------------|
| **Recon** | Network scanning and device discovery |
| **Pulse** | Health monitoring, metrics, alerting |
| **Dispatch** | Scout agent enrollment and management |
| **Vault** | Encrypted credential storage |
| **Gateway** | Browser-based remote access (SSH, RDP, HTTP proxy) |

## Quick Start

### Prerequisites

- Go 1.25+
- Make (optional)

### Build

```bash
make build
```

### Run Server

```bash
# With defaults
./bin/netvantage

# With config file
./bin/netvantage -config configs/netvantage.example.yaml
```

The server starts on `http://localhost:8080` by default.

### Run Scout Agent

```bash
./bin/scout -server localhost:9090 -interval 30
```

### API

```bash
# Health check
curl http://localhost:8080/api/v1/health

# List plugins
curl http://localhost:8080/api/v1/plugins
```

## Development

```bash
# Run tests
make test

# Run linter
make lint

# Clean build artifacts
make clean
```

## Project Structure

```
cmd/
  netvantage/     Server entry point
  scout/          Agent entry point
internal/
  server/         HTTP server and configuration
  plugin/         Plugin interface and registry
  recon/          Network discovery module
  pulse/          Monitoring module
  dispatch/       Agent management module
  vault/          Credential management module
  gateway/        Remote access module
  scout/          Agent core logic
pkg/
  models/         Shared data types
api/
  proto/v1/       gRPC service definitions
web/              React dashboard (Phase 2)
configs/          Example configuration files
```

## Roadmap

- **Phase 1**: Server + dashboard + agentless scanning
- **Phase 1b**: Windows Scout agent
- **Phase 2**: SNMP, mDNS, UPnP discovery + monitoring + Linux agent
- **Phase 3**: Remote access (SSH, RDP, HTTP proxy) + credential vault
- **Phase 4**: MQTT/IoT + cross-platform agents + API integrations

## Support the Project

NetVantage is **free for personal and home use forever**. If you find it useful and want to support continued development:

- [GitHub Sponsors](https://github.com/sponsors/HerbHall)
- [Ko-fi](https://ko-fi.com/herbhall)
- [Buy Me a Coffee](https://buymeacoffee.com/herbhall)

Financial supporters are recognized in [SUPPORTERS.md](SUPPORTERS.md) and the in-app About page. You can also contribute by [reporting bugs](https://github.com/HerbHall/netvantage/issues), [requesting features](https://github.com/HerbHall/netvantage/discussions), testing beta releases, or building plugins.

## License

NetVantage uses a split licensing model:

- **Core** (server, agent, built-in modules): [Business Source License 1.1](LICENSE) -- free for personal, home-lab, and non-competing production use. Converts to Apache 2.0 after 4 years.
- **Plugin SDK** (`pkg/plugin/`, `pkg/roles/`, `pkg/models/`, `api/proto/`): [Apache License 2.0](pkg/plugin/LICENSE) -- build plugins and integrations with no restrictions.

See [LICENSING.md](LICENSING.md) for full details.
