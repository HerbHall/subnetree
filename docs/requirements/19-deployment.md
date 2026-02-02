## Deployment

### Single Binary

The Go server embeds:
- Static web assets (`web/dist/` via `embed.FS`)
- Database migrations (via `embed.FS`)
- Default configuration
- OUI database for manufacturer lookup

### Docker Compose (Full Stack)

```yaml
services:
  netvantage:
    image: netvantage/server:latest
    ports:
      - "8080:8080"   # Web UI + API
      - "9090:9090"   # gRPC (Scout agents)
    volumes:
      - netvantage-data:/data
    environment:
      - NV_DATABASE_DSN=/data/netvantage.db
      - NV_VAULT_PASSPHRASE_FILE=/run/secrets/vault_passphrase

  guacamole:  # Optional: only if Gateway module is enabled
    image: guacamole/guacd
    ports:
      - "4822:4822"
```

### Deployment Profiles

Pre-configured module sets for common use cases:

| Profile | Modules Enabled | Use Case |
|---------|----------------|----------|
| **full** | All | Home lab with everything |
| **monitoring-only** | Recon + Pulse | Network awareness without remote access |
| **remote-access** | Vault + Gateway + Recon | Remote access tool without monitoring |
| **msp** | All + multi-tenancy | Managed service provider |

Usage: `netvantage --profile monitoring-only` or copy profile as starting config.

### Performance Profiles (Adaptive Scaling)

The server adapts its resource usage to the available hardware. At startup, it detects available CPU cores and RAM and selects a performance profile automatically. Users can override with `--performance-profile <name>` or in `config.yaml`.

| Profile | Auto-Selected When | Scan Workers | Default Scan Interval | In-Memory Metrics Buffer | Disk Retention | Target Memory | Analytics |
|---------|-------------------|-------------|----------------------|------------------------|---------------|---------------|-----------|
| **micro** | RAM < 1 GB | 1 (sequential) | 10 minutes | 24 hours | 7 days | < 200 MB | Disabled |
| **small** | RAM 1–2 GB | 2 | 5 minutes | 48 hours | 30 days | < 500 MB | Basic (EWMA, Z-score) |
| **medium** | RAM 2–4 GB | 4 | 5 minutes | 72 hours | 90 days | < 1 GB | Full Tier 1 |
| **large** | RAM > 4 GB | 8+ (scales with cores) | 5 minutes (configurable to 1m) | Configurable | Configurable | Configurable | Full Tier 1 + Tier 2/3 opt-in |

**Micro profile constraints:**
- Discovery protocols run sequentially (ARP, then ICMP, then mDNS, then SSDP) to limit peak memory
- Metrics aggregated to 5-minute granularity after 24 hours (reduces storage 5x)
- Topology map limited to 100 nodes before requiring manual pagination
- Dashboard WebSocket updates throttled to 5-second intervals

**Scaling behaviors:**
- Module lazy initialization: subsystems allocate resources on first use, not at startup
- Per-module memory budgets with backpressure: if a module exceeds its budget, it reduces polling frequency and drops oldest buffered metrics
- Discovery concurrency scales with available CPU cores: `min(cores, configured_max_workers)`
- SQLite WAL checkpoint frequency adapts: more aggressive on constrained storage, less frequent on large disks

**Detection override:** Users on constrained hardware who know their workload can override auto-detection. A Raspberry Pi 5 with 8 GB RAM monitoring only 10 devices can safely use the `medium` profile despite the auto-selected `small` profile.

### Configuration

Every setting has a sensible default. A zero-configuration deployment (just run the binary) works out of the box with all modules enabled, SQLite storage, and automatic network detection. Advanced users can customize every aspect via YAML config, environment variables, CLI flags, or the web UI settings page.

**Configuration priority (highest wins):** CLI flags > environment variables > config file > built-in defaults.

**Configuration Format Versioning:**

The configuration file includes a `config_version` field at the root level. This enables the server to detect and handle config files from older or newer versions.

**Rules:**
- `config_version` is an integer, incremented only when a config change is **breaking** (renamed keys, removed keys, changed semantics of existing keys).
- **Additive changes** (new keys with defaults) do NOT increment `config_version`. An old config file works without modification.
- If `config_version` is missing, the server assumes version 1 (backward compatible with pre-versioning configs).
- If `config_version` is newer than the server supports, the server refuses to start with: `"Configuration file version %d is newer than this server supports (max: %d). Upgrade the server or downgrade the config."`
- If `config_version` is older than current, the server starts normally but logs a warning: `"Configuration file version %d is outdated (current: %d). Run 'netvantage config migrate' to update. See release notes for changes."`
- The `netvantage config migrate` CLI command transforms an old config file to the current version, writing a backup of the original.
- **Breaking config changes are rare.** Most config evolution is additive (new keys with sensible defaults). A `config_version` bump is expected roughly once per major server version, if at all.

```yaml
config_version: 1    # Configuration schema version (integer, incremented only on breaking changes)

server:
  host: "0.0.0.0"
  port: 8080
  data_dir: "./data"

logging:
  level: "info"      # debug, info, warn, error
  format: "json"     # json, console

database:
  driver: "sqlite"
  dsn: "./data/netvantage.db"

auth:
  jwt_secret: ""                    # Auto-generated on first run
  access_token_ttl: "15m"
  refresh_token_ttl: "168h"         # 7 days
  oidc:
    enabled: false
    issuer: ""
    client_id: ""
    client_secret: ""
    redirect_url: "http://localhost:8080/api/v1/auth/oidc/callback"

modules:
  recon:
    enabled: true
    scan_interval: "5m"
    methods:
      icmp: true
      arp: true
      snmp: false
  pulse:
    enabled: true
    check_interval: "30s"
  dispatch:
    enabled: true
    grpc_port: 9090
  vault:
    enabled: true
    passphrase_file: ""             # Path to file containing vault passphrase
  gateway:
    enabled: true
    guacamole_address: "guacamole:4822"
  tailscale:
    enabled: false                          # Disabled by default (not all users have Tailscale)
    tailnet: ""                             # Tailnet name (e.g., "example.com" or "user@gmail.com")
    auth_method: "api_key"                  # "api_key" or "oauth"
    api_key_credential_id: ""              # Vault credential ID for API key
    oauth_credential_id: ""                # Vault credential ID for OAuth client
    sync_interval: "5m"                     # How often to poll Tailscale API for device changes
    import_tags: true                       # Import Tailscale ACL tags as NetVantage device tags
    prefer_tailscale_ip: true              # Use Tailscale IPs when device is on tailnet
    discover_subnet_routes: true            # Detect and offer to scan advertised subnet routes
```

Environment variable override prefix: `NV_` (e.g., `NV_SERVER_PORT=9090`, `NV_MODULES_GATEWAY_ENABLED=false`)
