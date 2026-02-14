# SubNetree Quick Reference

## Docker Commands

```bash
# Start SubNetree
docker run -d --name subnetree \
  -p 8080:8080 \
  --cap-add NET_RAW --cap-add NET_ADMIN \
  -v subnetree-data:/data \
  ghcr.io/herbhall/subnetree:latest

# Start with host networking (Linux -- recommended for full discovery)
docker run -d --name subnetree \
  --network host \
  --cap-add NET_RAW --cap-add NET_ADMIN \
  -v subnetree-data:/data \
  ghcr.io/herbhall/subnetree:latest

# View logs
docker logs -f subnetree

# Stop
docker stop subnetree

# Start again
docker start subnetree

# Update to latest version
docker pull ghcr.io/herbhall/subnetree:latest
docker stop subnetree && docker rm subnetree
docker run -d --name subnetree \
  -p 8080:8080 \
  --cap-add NET_RAW --cap-add NET_ADMIN \
  -v subnetree-data:/data \
  ghcr.io/herbhall/subnetree:latest

# Backup data volume
docker run --rm \
  -v subnetree-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/subnetree-backup.tar.gz /data

# Docker Compose
docker compose up -d
docker compose down
docker compose logs -f
```

## Key API Endpoints

Replace `$TOKEN` with a valid JWT access token from the login endpoint.

### Health and Status

```bash
# Liveness probe (no auth required)
curl http://localhost:8080/healthz

# Readiness probe (no auth required)
curl http://localhost:8080/readyz

# Detailed health with plugin status
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/health

# List registered plugins
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/plugins
```

### Authentication

```bash
# Login (returns access_token and refresh_token)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "your-password"}'

# Refresh access token
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "your-refresh-token"}'

# Logout
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN"

# First-time setup (create admin account)
curl -X POST http://localhost:8080/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "your-secure-password"}'
```

### Recon (Network Discovery)

```bash
# List discovered devices
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/recon/devices

# Get a specific device
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/recon/devices/{id}

# Start a network scan
curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/recon/scan \
  -H "Content-Type: application/json" \
  -d '{"targets": ["192.168.1.0/24"]}'

# List past scans
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/recon/scans

# Get network topology
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/recon/topology
```

### Pulse (Monitoring)

```bash
# List all health checks
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/pulse/checks

# Create a new check
curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/pulse/checks \
  -H "Content-Type: application/json" \
  -d '{"device_id": "device-uuid", "type": "ping", "interval": "30s"}'

# List active alerts
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/pulse/alerts

# Acknowledge an alert
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/pulse/alerts/{id}/acknowledge

# Get device monitoring status
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/pulse/status/{device_id}
```

### Vault (Credential Storage)

```bash
# Check vault status (sealed/unsealed)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/vault/status

# Unseal the vault
curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/vault/unseal \
  -H "Content-Type: application/json" \
  -d '{"passphrase": "your-vault-passphrase"}'

# Seal the vault
curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/vault/seal

# List stored credentials (metadata only)
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/vault/credentials
```

## Configuration Quick Reference

### Environment Variables

Environment variables use the `NV_` prefix and override config file values. Nested keys use underscores (e.g., `plugins.recon.concurrency` becomes `NV_PLUGINS_RECON_CONCURRENCY`).

| Variable | Default | Description |
|----------|---------|-------------|
| `NV_SERVER_PORT` | `8080` | HTTP listen port |
| `NV_SERVER_HOST` | `0.0.0.0` | HTTP listen address |
| `NV_LOGGING_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `NV_LOGGING_FORMAT` | `json` | Log format: `json` or `console` |
| `NV_DATABASE_DSN` | `./data/subnetree.db` | SQLite database file path |
| `NV_AUTH_JWT_SECRET` | (auto-generated) | JWT signing secret (set for persistent sessions) |
| `SUBNETREE_VAULT_PASSPHRASE` | (none) | Vault encryption passphrase (auto-unseal on start) |
| `NV_PLUGINS_RECON_CONCURRENCY` | `64` | Max concurrent scan probes |
| `NV_PLUGINS_RECON_SCAN_TIMEOUT` | `5m` | Maximum scan duration |
| `NV_PLUGINS_PULSE_CHECK_INTERVAL` | `30s` | Default monitoring check interval |
| `NV_PLUGINS_PULSE_MAX_WORKERS` | `10` | Max concurrent check workers |
| `NV_PLUGINS_PULSE_CONSECUTIVE_FAILURES` | `3` | Failures before alerting |
| `NV_PLUGINS_GATEWAY_MAX_SESSIONS` | `100` | Max concurrent remote sessions |
| `NV_PLUGINS_LLM_URL` | `http://localhost:11434` | Ollama API endpoint |
| `NV_PLUGINS_LLM_MODEL` | `qwen2.5:32b` | Ollama model name |

### Config File Paths (searched in order)

1. `./subnetree.yaml`
2. `./configs/subnetree.yaml`
3. `/etc/subnetree/subnetree.yaml`

Override with `--config /path/to/config.yaml`.

### Key YAML Config Structure

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  data_dir: "./data"

logging:
  level: "info"        # debug | info | warn | error
  format: "json"       # json | console

database:
  dsn: "./data/subnetree.db"

plugins:
  recon:
    concurrency: 64    # reduce to 16-32 on Raspberry Pi
    scan_timeout: "5m"
  pulse:
    check_interval: "30s"
    max_workers: 10
  vault:
    enabled: true
  gateway:
    max_sessions: 100
```

## Troubleshooting Checklist

| Problem | Likely Cause | Fix |
|---------|-------------|-----|
| Cannot access web UI | Port not mapped or firewall blocking | Verify `-p 8080:8080` in docker run; check host firewall rules |
| Scan finds no devices | Container networking is isolated | Use `--network host` (Linux) or add `--cap-add NET_RAW --cap-add NET_ADMIN` |
| Vault is sealed | No passphrase provided at startup | `POST /api/v1/vault/unseal` with passphrase, or set `SUBNETREE_VAULT_PASSPHRASE` env var |
| High memory usage | Too many concurrent probes | Reduce `NV_PLUGINS_RECON_CONCURRENCY` (try `16` or `32`) |
| Connection refused | Server not running or wrong port | Check `docker ps`, then `docker logs subnetree` for errors |
| Login fails | Wrong credentials or no account exists | Use `/api/v1/auth/setup` on first run to create admin account |
| Tokens expire after restart | JWT secret is auto-generated | Set `NV_AUTH_JWT_SECRET` to a stable 64+ character hex string |
| Agent cannot connect | gRPC port not exposed | Expose port `9090` with `-p 9090:9090` in docker run |
| Scan timeout | Large subnet or slow network | Increase `NV_PLUGINS_RECON_SCAN_TIMEOUT` (e.g., `10m`) |

## Keyboard Shortcuts

Shortcuts are active when not focused on a text input field.

| Key | Action |
|-----|--------|
| `1` | Go to Dashboard |
| `2` | Go to Devices |
| `3` | Go to Topology |
| `4` | Go to Agents |
| `5` | Go to Monitoring |
| `6` | Go to Services |
| `7` | Go to Vault |
| `8` | Go to Docs |
| `9` | Go to Settings |
| `Shift` + `?` | Show all keyboard shortcuts |

## Default Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| `8080` | HTTP | Web UI and REST API |
| `9090` | gRPC | Scout agent communication |
