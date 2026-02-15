## Product Vision

SubNetree is a modular, source-available network monitoring and management platform that provides unified device discovery, monitoring, remote access, credential management, and IoT awareness in a single self-hosted application.

**Strategic Position: "Start Here, Grow Anywhere."** SubNetree is the gateway product for infrastructure management. It gives users a single entry point that discovers, monitors, and documents their network -- then connects them to the broader ecosystem of specialized tools as they grow. Every homelab is unique; SubNetree supports that by letting users pick their own components while providing sensible defaults for everything.

**Target Users:** HomeLab enthusiasts, prosumers, and small business IT administrators running 10-500 network devices on hardware ranging from a Raspberry Pi to a small business server.

**Market Scope:** SubNetree targets single-subnet home and small-office networks. The primary market is 10-75 devices (covering ~80% of homelabbers); the secondary market is 75-250 devices (advanced homelabs and small businesses). The focus is building a product that delights HomeLabbers and small business users, not competing with enterprise monitoring platforms. The backend architecture is modular, well-documented, and acquisition-ready -- the same standard protocols that connect homelab tools are what enterprise acquirers need.

**Core Value Proposition:** No existing source-available tool combines device discovery, monitoring, remote access, credential management, and IoT awareness in a single product. SubNetree fills the gap that every other tool leaves open: automatic infrastructure discovery and documentation. NetBox has no auto-discovery. Uptime Kuma has no device inventory. Ansible inventory is static by default. SubNetree is the automation layer that makes all of them useful.

**Licensing:** Free for personal, HomeLab, and non-competing production use. BSL 1.1 licensed core with Apache 2.0 plugin SDK for ecosystem growth.

### What It Does

**Discovery and Mapping:**

- LAN scanning and device detection (ARP, mDNS, SNMP, Nmap)
- Device identification (OS, manufacturer, type, hardware profile)
- Network topology mapping and visualization
- Automatic inventory that stays current without manual entry

**Monitoring:**

- Health and status monitoring via active scanning or lightweight agents (Scout)
- Plugins can monitor anything -- the platform provides the framework, plugins provide the data
- Adaptive thresholds and alert suppression for dependency-aware monitoring

**Quick Access:**

- One-click access to systems and services
- Credential vault so you don't type passwords hundreds of times a day
- Launch RDP, SSH, web UIs directly from the dashboard

**Rich, Customizable UI:**

- List views, tree views, charts, graphs, gauges
- Various status indicators (up/down, health, alerts)
- Highly customizable -- users display what matters to them

**Plugin Extensibility:**

- Open plugin architecture for anything users want to add
- Data can come from active scanning/detection or from helper agents on hosts
- Community and third-party plugins welcome

**Ecosystem Integration:**

- Export to tools users already run (NetBox, Grafana, Ansible, Home Assistant)
- Import from tools users are migrating away from (Zabbix, Nagios, spreadsheets)
- Prometheus-compatible `/metrics` endpoint for universal monitoring interoperability
- MQTT publisher for Home Assistant integration

### Gateway Philosophy

SubNetree is a jack of all trades, master of none -- by design. Each module provides 80% of what a dedicated tool offers, enough for most users. When users outgrow a module, SubNetree helps them graduate to specialized tools and stays in the loop as the discovery and inventory engine.

**Growth Model:**

1. **Detect limits.** SubNetree recognizes when a user's needs exceed a module's capabilities (e.g., monitoring 200+ endpoints, needing HA failover, requiring compliance reporting).
2. **Assess capability.** SubNetree knows the user's hardware profile from Scout data. It compares available resources against what each growth option requires, filtering recommendations to what the user can actually run.
3. **Suggest alternatives.** Present two tiers of options: (a) what's possible on current hardware, and (b) what becomes possible with hardware upgrades. Recommendations include both additional SubNetree modules and ecosystem tools, ordered by cost (free first) and hardware fit.
4. **Operate side-by-side.** SubNetree feeds data to the specialized tool via standard protocols (Prometheus, REST, MQTT, YAML). Both tools run simultaneously during transition.
5. **Remain the inventory.** Even after a user adopts Grafana for dashboards or Uptime Kuma for uptime monitoring, SubNetree stays as the central device inventory and discovery engine.

**Recommendation Engine:**

The growth model is powered by a curated **ecosystem catalog** -- a maintained database of SubNetree modules and ecosystem tools with feature comparisons, hardware requirements, and integration status. The catalog has two tiers:

- **Embedded catalog** -- Ships with the binary (~50 KB YAML). Covers SubNetree modules and the top 15 ecosystem tools. Updated each release. Works fully offline.
- **Remote catalog** -- Optional weekly fetch from a static JSON file on GitHub Pages (free hosting). Covers 50+ tools with current version info, community ratings, and detailed comparisons. Cached locally in SQLite. Degrades gracefully if unreachable.

Each catalog entry includes hardware requirements (min/recommended RAM, CPU, storage), feature lists, advantages vs SubNetree, cost, license, and growth triggers (usage-pattern conditions that suggest this tool). SubNetree's recommendation engine cross-references the user's hardware profile and usage patterns against the catalog to generate personalized, contextual suggestions.

Example recommendation flow: "You're monitoring 85 endpoints on a Tier 1 mini PC (32 GB RAM, 22 GB free). Your hardware supports: enabling the Insight analytics module (+50 MB), adding Uptime Kuma for dedicated uptime monitoring (+80 MB), or adding Grafana for advanced dashboards (+200 MB). With a Tier 3 upgrade, you could also run Prometheus for long-term metrics storage."

**Module Growth Pathways:**

| Module | SubNetree Provides | Free/Low-Cost Growth Path | Enterprise Growth Path |
| ------ | ----------------- | ------------------------ | -------------------- |
| **Recon** (Discovery) | ARP/mDNS/SNMP/Nmap scanning, device profiling | NetAlertX, WatchYourLAN | NetBox Discovery, runZero |
| **Pulse** (Monitoring) | ICMP, HTTP, TCP, DNS checks with alerting | Uptime Kuma, Beszel | Prometheus + Grafana, Zabbix |
| **Vault** (Credentials) | Encrypted credential storage, device linking | Vaultwarden | IT Glue, CyberArk |
| **Gateway** (Remote Access) | SSH proxy, future RDP/VNC | Guacamole | BeyondTrust, Teleport |
| **Insight** (Analytics) | Rule-based anomaly detection, LLM integration | Grafana dashboards | Datadog, Splunk |
| **Dispatch** (Agents) | Scout agent for device profiling and metrics | Beszel agent (complementary) | Salt, Ansible Tower |
| **Documentation** | Auto-generated device docs, change logs | BookStack, Obsidian | Hudu, IT Glue |

### Hardware Tier System

SubNetree targets five hardware tiers based on real community survey data (selfh.st 4,081 respondents, deployn 850+, Home Assistant Analytics 596,779 installations). Resource targets ensure SubNetree runs comfortably on any tier.

| Tier | Hardware | Market Share | Available RAM | SubNetree Target | Network Size |
| ---- | -------- | ----------- | ------------- | --------------- | ----------- |
| **0** | SBC (RPi 4/5) | 40.6% (declining) | 3-7 GB | Server lite: 50-100 MB | 5-25 devices |
| **1** | Mini PC (N100/Ryzen) | 48.4% (rising) | 14-30 GB | Server standard: 100-200 MB | 10-75 devices |
| **2** | NAS (Synology/QNAP) | 22.5% | 2-6 GB | Scout only: 5-15 MB | Agent only |
| **3** | Cluster (multi-node) | ~15% | 60-186 GB | Server full: 150-300 MB | 50-250 devices |
| **4** | Small Business server | Varies | 28-60 GB | Server full: 150-300 MB | 50-500 devices |

**Design Constraint:** The Tier 0 experience (Raspberry Pi 4, 4 GB RAM) is the minimum viable target. If a feature cannot run within 100 MB on a Pi 4, it must be optional and disabled by default. The Scout agent must stay under 15 MB to run on NAS devices and low-end VPS instances.

**Tier-Aware Defaults:**

| Tier | Scan Interval | Data Retention | AI Features | Modules Enabled |
| ---- | ------------ | -------------- | ----------- | -------------- |
| **0** (SBC) | 5 min | 7 days | Off | Recon, Pulse |
| **1** (Mini PC) | 2 min | 30 days | Basic (rule-based) | All |
| **2** (NAS) | 1 min (agent) | N/A (server-side) | N/A | Scout only |
| **3** (Cluster) | 1 min | 90 days | Full (LLM integration) | All + Analytics |
| **4** (SMB) | 1 min | 180 days | Full | All + Analytics |

### Integration Strategy

SubNetree is the "data pump" that feeds the entire self-hosted ecosystem. Integration priority targets free and low-cost tools first, matching the homelab audience's budget reality.

**Priority 0 -- Foundation (ship with v0.5.0+):**

- Prometheus `/metrics` endpoint (instant Grafana/Prometheus compatibility)
- MQTT publisher for Home Assistant auto-discovery
- Generic webhook notifications (Alertmanager-compatible format)
- CSV import/export (universal compatibility)

**Priority 1 -- Tool Bridges:**

- NetBox JSON export (push devices to NetBox CMDB)
- Ansible YAML inventory import/export
- Uptime Kuma monitor sync (auto-create monitors from discovered services)
- Nmap XML import (seed discovery from existing scans)

**Priority 2 -- Ecosystem Depth:**

- Grafana dashboard bundle (pre-built JSON dashboards)
- NetAlertX device import
- Markdown documentation generation
- Nagios plugin executor (run existing check scripts)

**Priority 3 -- Migration and Enterprise:**

- Zabbix host/template import (enterprise migration path)
- Nagios config import (legacy migration path)
- Beszel metric display (complementary tool bridge)
- Config versioning (Oxidized-style git-based snapshots)

### Design Philosophy

1. **Ease of use first.** You should not need a tech degree to operate SubNetree. The interface should be intuitive enough that a non-technical small business owner can understand their network health at a glance, while an experienced sysadmin can drill into the detail they need.

2. **Sensible defaults, deep customization.** SubNetree ships fully preconfigured for rapid deployment -- install and go. Defaults are tuned per hardware tier so a Raspberry Pi user gets a lightweight experience and a cluster user gets full features. The defaults get you running; customization makes it yours.

3. **Plugin-powered extensibility.** The plugin architecture is not an afterthought -- it is the architecture. Every major feature is a plugin. Users and third-party developers can replace, extend, or supplement any module. The system is designed to be shaped by its users, not constrained by its authors.

4. **Stability and security are non-negotiable.** These are not features that ship "later." Every release must be stable enough to trust with production infrastructure and secure enough to trust with network credentials. If a feature compromises stability or security, it does not ship.

5. **Time to First Value under 10 minutes.** Users will forgive missing features but will not forgive a bad first experience. Download, install, see your network -- in minutes, not hours.

6. **Gateway, not gatekeeper.** SubNetree helps users grow into the tools that best fit their needs. Each module provides enough capability to start, then offers clear pathways to specialized alternatives. Never lock users in; always make it easy to export data and integrate with other tools.

7. **Lightweight by default, powerful on demand.** The server fits in 50-200 MB of RAM. The Scout agent fits in 5-15 MB. Heavy features (AI analytics, long retention, deep scanning) are opt-in and disabled by default on constrained hardware. The system adapts to the user's hardware, not the other way around.

8. **Free-tier ecosystem first.** Growth recommendations prioritize free and low-cost self-hosted tools (Uptime Kuma, Beszel, Grafana, Vaultwarden, BookStack) over enterprise solutions. Enterprise tools are documented but presented as the upgrade path, not the default.

9. **Standard protocols over custom APIs.** Prefer Prometheus metrics, MQTT, Alertmanager webhooks, Ansible YAML, and NetBox JSON over proprietary formats. Every data exchange should use the format that the receiving tool already speaks.

10. **Infrastructure that documents itself.** The number one unmet need in the homelab community is documentation that stays current. SubNetree fills this gap by automatically tracking device inventory, configuration changes, service status, and network topology -- the documentation that no one writes manually but everyone wishes existed.

11. **Support every homelab.** Every homelab is unique. Users run different hardware (Pi to rack servers), different hypervisors (Proxmox, ESXi, bare metal), different orchestrators (Docker, K3s, LXC), and different monitoring stacks. SubNetree works with all of them through standard protocols and a flexible plugin system.

### Exit Strategy Alignment

The gateway positioning creates a clear path for commercial acquisition:

- **Clean IP chain.** BSL 1.1 core + Apache 2.0 SDK. CLA on all contributions. No GPL dependencies.
- **Integration surface.** The same protocols that connect homelab tools (Prometheus, MQTT, REST) are the same ones enterprise platforms need. An acquirer gets a product that already speaks the enterprise language.
- **Community and ecosystem.** A passionate community of homelabbers and small business users provides a built-in customer base and plugin ecosystem.
- **Upmarket expansion.** The modular architecture supports multi-tenancy (#280), managed deployment, and commercial integrations without rewriting the core. An acquirer with enterprise sales can layer MSP features, compliance reporting, and SLA management on top of the existing platform.
