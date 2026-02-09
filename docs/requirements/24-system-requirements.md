## System & Network Requirements

### Minimum Hardware

| Tier | Devices | CPU | RAM | Disk | Notes |
|------|---------|-----|-----|------|-------|
| **Micro** | < 25 | 1 core (ARM64 or x64) | 512 MB | 4 GB | Raspberry Pi 4 (4 GB+), Pi 5, Docker on NAS. Uses `micro` performance profile. |
| **Small** | 25–100 | 1 vCPU | 1 GB | 10 GB | Intel N100 mini PC, refurb enterprise micro (Dell OptiPlex, HP EliteDesk), small Proxmox VM |
| **Medium** | 100–500 | 2 vCPU | 2 GB | 25 GB | Dedicated VM or container, small business single site |
| **Large** | 500–1,000 | 4 vCPU | 4 GB | 50 GB | MSP, multi-site |
| **Enterprise** | 1,000+ | 4+ vCPU | 8+ GB | 100+ GB | Requires PostgreSQL + TimescaleDB (Phase 2) |

Disk estimates assume default data retention policies. SNMP polling and high-frequency metrics increase storage requirements. The server auto-selects a performance profile based on detected hardware (see Adaptive Performance Profiles).

### Server Resource Footprint

Target memory consumption for the Go server process (excluding OS and other services):

| Component | Memory |
|-----------|--------|
| Go runtime + core application | 15–25 MB |
| HTTP server + embedded web assets | 10–20 MB |
| SQLite engine (WAL mode) | 5–10 MB |
| Discovery engines (ARP, mDNS, SSDP, SNMP) | 10–30 MB |
| MQTT client + message buffer | 5–15 MB |
| Per monitored device (state + metrics buffer) | 0.5–2 MB each |
| **Estimated total: 50 devices** | **~100–200 MB** |
| **Estimated total: 200 devices** | **~200–500 MB** |
| **Estimated total: 500 devices** | **~500 MB–1.2 GB** |

These estimates guide the performance profile auto-selection and prerequisite checks. Actual usage depends on enabled modules, scan frequency, and metrics retention settings.

### Typical Deployment Scenarios

| Scenario | Hardware | Performance Profile | Expected Device Count |
|----------|----------|--------------------|-----------------------|
| Raspberry Pi 5 (8 GB) dedicated | ARM64, 4 cores, 8 GB RAM, NVMe HAT | `small` (auto) or `medium` (override) | 20–75 |
| Docker on Synology DS920+ | x64, 4 cores, 4–8 GB shared | `micro` or `small` (depends on container limits) | 15–50 |
| Intel N100 mini PC (bare metal) | x64, 4 cores, 16 GB RAM | `large` (auto) | 50–300 |
| Proxmox VM (2 vCPU, 2 GB) | x64, 2 cores, 2 GB RAM | `medium` (auto) | 50–200 |
| Refurb Dell OptiPlex Micro (16 GB) | x64, 6 cores, 16 GB RAM | `large` (auto) | 100–500 |
| Cloud VPS (2 vCPU, 2 GB) + Tailscale | x64, 2 cores, 2 GB RAM | `medium` (auto) | 50–200 (via Tailscale + Scout agents) |
| Rack server VM (4 vCPU, 8 GB) | x64, 4 cores, 8 GB RAM | `large` (auto) | 500–1,000 |

### Supported Server Platforms

| Platform | Architecture | Phase | Notes |
|----------|-------------|-------|-------|
| Linux (Debian/Ubuntu, RHEL/Fedora, Alpine) | x64, ARM64 | 1 | Primary target |
| Windows Server 2019+ / Windows 10+ | x64 | 1 | Native binary |
| Docker | x64, ARM64 | 1 | Official images, multi-arch manifest |
| macOS | ARM64 (Apple Silicon) | 2 | Development/testing use |

**Validated deployment targets** (tested in CI or community-verified):

| Target | Example Hardware | Docker? | Native? | Notes |
|--------|-----------------|---------|---------|-------|
| Raspberry Pi 4/5 | ARM64, 4–16 GB | Yes | Yes | Micro/small profile. NVMe via HAT recommended for Pi 5. |
| Intel N100 mini PCs | x64, 8–16 GB, 6W TDP | Yes | Yes | Ideal HomeLab platform. Beelink, MinisForum, CWWK, etc. |
| Refurb enterprise micro PCs | x64, 8–64 GB | Yes | Yes | Dell OptiPlex Micro, HP EliteDesk Mini, Lenovo ThinkCentre Tiny |
| Synology NAS (DS920+, DS1522+) | x64, 4–32 GB | Yes (Container Manager) | No | Docker container on existing NAS. Zero additional hardware. |
| QNAP NAS (TS-464, TS-873A) | x64, 8–16 GB | Yes (Container Station) | No | Docker container on existing NAS. |
| Proxmox VE (VM or LXC) | x64 or ARM64 | Yes (inside VM/LXC) | Yes (inside VM/LXC) | Common HomeLab hypervisor. LXC is more resource-efficient than full VM. |
| Unraid | x64 | Yes (Community Apps) | No | Docker container alongside media/NAS workloads. |
| TrueNAS SCALE | x64 | Yes | No | Docker container on ZFS-backed storage. |
| Cloud VPS (DigitalOcean, Linode, Hetzner, Oracle Cloud) | x64 or ARM64 | Yes | Yes | Remote monitoring via Tailscale or Scout agents. Oracle Cloud free tier (ARM64, 24 GB) is popular. |

### Port & Protocol Matrix

| Port | Protocol | Direction | Component | Purpose | Required |
|------|----------|-----------|-----------|---------|----------|
| 8080 | TCP/HTTP(S) | Inbound | Server | Web UI + REST API | Yes |
| 9090 | TCP/gRPC | Inbound | Server | Scout agent communication (mTLS) | If agents used |
| 4822 | TCP | Internal | Guacamole | RDP/VNC gateway | If Gateway module enabled |
| 161 | UDP/SNMP | Outbound | Server | SNMP polling | If SNMP scanning enabled |
| 162 | UDP/SNMP | Inbound | Server | SNMP traps | If SNMP traps enabled |
| -- | ICMP | Outbound | Server | Ping sweep | If ICMP scanning enabled |
| 5353 | UDP/mDNS | Outbound | Server | mDNS discovery | If mDNS scanning enabled |
| 1900 | UDP/SSDP | Outbound | Server | UPnP/SSDP discovery | If UPnP scanning enabled |
| 1883/8883 | TCP/MQTT | Outbound | Server | MQTT broker communication | If MQTT enabled |

### Reverse Proxy Deployment

SubNetree supports operation behind a reverse proxy (nginx, Traefik, Caddy). Requirements:
- WebSocket upgrade support for real-time dashboard updates (`/ws/` path)
- gRPC passthrough or gRPC-Web translation for Scout agent communication on port 9090
- `X-Forwarded-For`, `X-Forwarded-Proto` headers for accurate client IP logging
- Configurable `--base-path` flag for non-root deployments (e.g., `/subnetree/`)

### Network Considerations

- **IPv6:** Phase 1 is IPv4-only. IPv6 scanning and agent communication targeted for Phase 2.
- **Time synchronization:** NTP is strongly recommended. mTLS certificate validation and metric accuracy depend on synchronized clocks. Server logs a warning at startup if clock skew > 5 seconds from an NTP check.
- **DNS:** Server needs DNS resolution for hostname lookups during discovery. Configurable DNS server override for environments with split DNS.
- **Tailscale:** When the Tailscale plugin is enabled, the server uses the Tailscale REST API (outbound HTTPS to `api.tailscale.com`) for device discovery. No additional ports required. Scout agents on Tailscale-connected devices can reach the server via Tailscale IPs (100.x.y.z), eliminating the need for port forwarding or public IP exposure.
