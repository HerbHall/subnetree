# Platform-Specific Installation Notes

This guide covers platform-specific considerations for running SubNetree on different operating systems and hardware. For general setup, see [Getting Started](getting-started.md).

## Ports Reference

SubNetree uses two ports by default:

| Port | Protocol | Purpose |
|------|----------|---------|
| 8080 | TCP/HTTP | Web UI and REST API |
| 9090 | TCP/gRPC | Scout agent communication (only if using Scout agents) |

## Linux

### Systemd Service

Create a systemd unit file to run SubNetree as a background service that starts on boot.

```ini
# /etc/systemd/system/subnetree.service
[Unit]
Description=SubNetree Network Monitoring
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=subnetree
Group=subnetree
WorkingDirectory=/opt/subnetree
ExecStart=/opt/subnetree/subnetree --config /etc/subnetree/subnetree.yaml
Restart=on-failure
RestartSec=5

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/subnetree/data

# Required for network scanning (ICMP ping, ARP table)
AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN
CapabilityBoundingSet=CAP_NET_RAW CAP_NET_ADMIN

# Vault passphrase (uncomment and set for auto-unseal)
# Environment=SUBNETREE_VAULT_PASSPHRASE=your-secure-passphrase

# Alternatively, load from a protected file
# EnvironmentFile=/etc/subnetree/env

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable subnetree
sudo systemctl start subnetree
sudo systemctl status subnetree

# View logs
sudo journalctl -u subnetree -f
```

### File Permissions

```bash
# Create a dedicated service user
sudo useradd --system --no-create-home --shell /usr/sbin/nologin subnetree

# Set ownership on the installation and data directories
sudo chown -R subnetree:subnetree /opt/subnetree
sudo chmod 750 /opt/subnetree
sudo chmod 700 /opt/subnetree/data

# Protect the config file (may contain JWT secret)
sudo chown root:subnetree /etc/subnetree/subnetree.yaml
sudo chmod 640 /etc/subnetree/subnetree.yaml

# Protect the environment file if using one
sudo chmod 600 /etc/subnetree/env
```

### Firewall (iptables / nftables)

```bash
# Allow HTTP (web UI and API)
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# Allow gRPC (Scout agents) -- only if using agents
sudo iptables -A INPUT -p tcp --dport 9090 -j ACCEPT

# For firewalld (RHEL, Fedora, CentOS):
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=9090/tcp   # only if using Scout agents
sudo firewall-cmd --reload
```

### Linux Capabilities (Binary Install)

When running SubNetree as a non-root binary (not Docker), it needs `NET_RAW` and `NET_ADMIN` capabilities for network scanning:

```bash
sudo setcap 'cap_net_raw,cap_net_admin=+ep' /opt/subnetree/subnetree
```

This is not needed for Docker deployments (handled by `cap_add` in compose) or when running as root (not recommended).

## Windows

### Running as a Windows Service

SubNetree does not include a built-in Windows service wrapper. Use [NSSM](https://nssm.cc/) (Non-Sucking Service Manager) to run it as a service.

**Install NSSM:**

Download NSSM from [nssm.cc](https://nssm.cc/download) and place `nssm.exe` in your PATH.

**Create the service:**

```powershell
# Install SubNetree as a Windows service
nssm install SubNetree "C:\SubNetree\subnetree.exe"
nssm set SubNetree AppParameters "--config C:\SubNetree\subnetree.yaml"
nssm set SubNetree AppDirectory "C:\SubNetree"
nssm set SubNetree AppStdout "C:\SubNetree\data\logs\stdout.log"
nssm set SubNetree AppStderr "C:\SubNetree\data\logs\stderr.log"

# Set environment variables
nssm set SubNetree AppEnvironmentExtra "SUBNETREE_VAULT_PASSPHRASE=your-secure-passphrase"

# Configure automatic restart
nssm set SubNetree AppExit Default Restart
nssm set SubNetree AppRestartDelay 5000

# Start the service
nssm start SubNetree
```

**Manage the service:**

```powershell
nssm status SubNetree
nssm restart SubNetree
nssm stop SubNetree
nssm remove SubNetree confirm   # uninstall
```

### Windows Defender Exclusions

SQLite performs frequent small writes. Windows Defender real-time scanning can cause significant I/O overhead on the database file. Exclude the data directory:

```powershell
# Run as Administrator
Add-MpExclusion -Path "C:\SubNetree\data"
```

This is especially important on systems with spinning disks or when running many concurrent monitoring checks.

### Windows Firewall

```powershell
# Allow HTTP (web UI and API)
New-NetFirewallRule -DisplayName "SubNetree HTTP" `
    -Direction Inbound -Protocol TCP -LocalPort 8080 -Action Allow

# Allow gRPC (Scout agents) -- only if using agents
New-NetFirewallRule -DisplayName "SubNetree gRPC" `
    -Direction Inbound -Protocol TCP -LocalPort 9090 -Action Allow
```

### Network Scanning on Windows

Network scanning on Windows has some limitations compared to Linux:

- **ICMP ping** works natively (Windows includes raw socket support for ICMP)
- **ARP table** reading works via system commands
- **mDNS discovery** works but may require Windows Firewall rules for UDP port 5353

## macOS

### Docker Desktop Limitations

Docker Desktop for macOS runs containers inside a Linux VM, which affects networking:

- **No host networking mode.** `network_mode: host` is not supported on macOS Docker Desktop. You must use published ports (`-p 8080:8080`).
- **Network scanning from Docker is limited.** The container sees the Docker bridge network, not your LAN. For full network discovery, run SubNetree natively or use a Linux host.
- **File sharing for data volumes.** Ensure the volume mount path is in Docker Desktop's allowed file sharing list (Settings > Resources > File Sharing).

### Running Natively on macOS

```bash
# Download the binary for your architecture (amd64 or arm64)
# Place it in a convenient location
mkdir -p ~/subnetree/data
mv subnetree ~/subnetree/

# Grant network capabilities (required for ICMP scanning)
# macOS requires running as root for raw socket access
sudo ~/subnetree/subnetree --config ~/subnetree/subnetree.yaml
```

### macOS Firewall

If the macOS firewall is enabled (System Settings > Network > Firewall), you will be prompted to allow incoming connections when SubNetree starts. Click "Allow" to permit access from other devices on your network.

Alternatively, add SubNetree to the firewall allow list manually in System Settings.

### Homebrew Installation

SubNetree is not currently published to Homebrew. Install from the GitHub release:

```bash
# Download the latest release for macOS
# Replace VERSION and ARCH with your values (e.g., v0.3.0, arm64)
curl -LO "https://github.com/HerbHall/subnetree/releases/download/VERSION/subnetree_darwin_ARCH.tar.gz"
tar xzf subnetree_darwin_ARCH.tar.gz
```

## Docker

### Bridge Networking (Default)

The default Docker networking mode. Works on all platforms but limits network scanning to devices visible from the Docker bridge network.

```bash
docker run -d \
  --name subnetree \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 9090:9090 \
  -v subnetree-data:/data \
  --cap-add NET_RAW \
  --cap-add NET_ADMIN \
  -e SUBNETREE_VAULT_PASSPHRASE=your-secure-passphrase \
  ghcr.io/herbhall/subnetree:latest
```

### Host Networking (Linux Only)

For full network scanning on home networks, host networking lets the container access the LAN directly. This is the recommended mode for Linux deployments that need complete device discovery.

```bash
docker run -d \
  --name subnetree \
  --restart unless-stopped \
  --network host \
  -v subnetree-data:/data \
  --cap-add NET_RAW \
  --cap-add NET_ADMIN \
  -e SUBNETREE_VAULT_PASSPHRASE=your-secure-passphrase \
  ghcr.io/herbhall/subnetree:latest
```

With host networking, ports are bound directly on the host -- no `-p` flags needed. The web UI is available at `http://<host-ip>:8080`.

**Note:** Host networking is not supported on Docker Desktop for macOS or Windows. It only works on native Linux Docker.

### Docker Compose

The repository includes a ready-to-use `docker-compose.yml` at the project root. Quick start:

```bash
docker compose up -d
```

See [docker-compose.yml](../../docker-compose.yml) for the full configuration with comments explaining each option.

### Volume Mounts

```yaml
volumes:
  # Persist all data (database, internal state)
  - subnetree-data:/data

  # Optionally mount a custom config file (instead of using env vars)
  # - ./subnetree.yaml:/etc/subnetree/subnetree.yaml:ro
```

The `/data` volume contains the SQLite database, vault encryption keys, and all persistent state. **Back up this volume regularly.**

### Environment Variables

Key environment variables for Docker deployments:

| Variable | Purpose | Example |
|----------|---------|---------|
| `SUBNETREE_VAULT_PASSPHRASE` | Auto-unseal vault on startup | `my-secure-passphrase` |
| `NV_SERVER_PORT` | Override HTTP port | `9080` |
| `NV_LOGGING_LEVEL` | Override log level | `debug` |
| `NV_LOGGING_FORMAT` | Override log format | `console` |
| `NV_PLUGINS_RECON_CONCURRENCY` | Override scan concurrency | `32` |

Environment variables use the `NV_` prefix and replace dots with underscores. For example, `plugins.recon.concurrency` becomes `NV_PLUGINS_RECON_CONCURRENCY`.

Exception: `SUBNETREE_VAULT_PASSPHRASE` uses its own prefix for security clarity.

## Raspberry Pi and ARM64

SubNetree provides ARM64 binaries and Docker images, making it a great fit for Raspberry Pi 4/5 and other ARM64 single-board computers.

### Performance Expectations

| Component | Pi 4 (4 GB) | Pi 5 (8 GB) | Notes |
|-----------|-------------|-------------|-------|
| Startup time | 3-5 seconds | 1-2 seconds | First run is slower (schema migration) |
| Memory usage | ~50-80 MB | ~50-80 MB | Increases with monitored device count |
| SQLite on SD card | Slow writes | Slow writes | Use USB SSD for production |
| 100-device scan | ~30 seconds | ~15 seconds | With default concurrency |

**Storage recommendation:** Run SubNetree's data directory on a USB SSD rather than the SD card. SD cards have limited write endurance and slow random I/O, both of which impact SQLite performance. Mount a USB drive and point `data_dir` or the Docker volume to it.

### Recommended Configuration

For Raspberry Pi 4 (4 GB RAM) or similar constrained devices, reduce concurrency to avoid memory pressure:

```yaml
plugins:
  recon:
    concurrency: 16          # Reduced from default 64
    ping_timeout: "3s"       # Slightly longer timeout for slower networks
  pulse:
    max_workers: 4           # Reduced from default 10
    check_interval: "60s"    # Less frequent checks to reduce CPU load
  insight:
    maintenance_interval: "6h"  # Less frequent cleanup
```

For Raspberry Pi 5 (8 GB RAM), the default settings work well for networks up to ~200 devices. You may want to reduce concurrency to 32 if monitoring more than 100 devices simultaneously.

### Docker on Raspberry Pi

```bash
# Use the ARM64 image (auto-selected on ARM hardware)
docker run -d \
  --name subnetree \
  --restart unless-stopped \
  --network host \
  -v /mnt/ssd/subnetree:/data \
  --cap-add NET_RAW \
  --cap-add NET_ADMIN \
  -e SUBNETREE_VAULT_PASSPHRASE=your-secure-passphrase \
  ghcr.io/herbhall/subnetree:latest
```

Host networking is strongly recommended on Raspberry Pi to enable full LAN device discovery.
