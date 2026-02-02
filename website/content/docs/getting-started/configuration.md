---
title: Configuration
weight: 2
---

NetVantage uses a YAML configuration file with sensible defaults. The server runs out of the box with no configuration required.

## Configuration File

By default, NetVantage looks for configuration in:

1. `./netvantage.yaml` (current directory)
2. `$HOME/.config/netvantage/netvantage.yaml`
3. `/etc/netvantage/netvantage.yaml`

Or specify a path explicitly:

```bash
./bin/netvantage -config /path/to/netvantage.yaml
```

## Example Configuration

An example configuration file is included in the repository:

```bash
cp configs/netvantage.example.yaml netvantage.yaml
```

## Environment Variables

All configuration values can be overridden via environment variables using the `NETVANTAGE_` prefix with underscore-separated paths.

| YAML Path | Environment Variable | Default |
|-----------|---------------------|---------|
| `server.http.port` | `NETVANTAGE_SERVER_HTTP_PORT` | `8080` |
| `server.grpc.port` | `NETVANTAGE_SERVER_GRPC_PORT` | `9090` |
| `database.path` | `NETVANTAGE_DATABASE_PATH` | `./data/netvantage.db` |
| `log.level` | `NETVANTAGE_LOG_LEVEL` | `info` |
| `log.format` | `NETVANTAGE_LOG_FORMAT` | `json` |

## API Endpoints

Once running, the following endpoints are available:

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/health` | Aggregated health status from all plugins |
| `GET /api/v1/plugins` | List of registered plugins and their status |

All API responses include the `X-NetVantage-Version` header.

{{< callout type="info" >}}
Configuration options will expand as new modules are implemented. This page reflects the current Phase 1 configuration surface. See the [full requirements](https://github.com/HerbHall/netvantage/tree/main/docs/requirements) for planned configuration options.
{{< /callout >}}
