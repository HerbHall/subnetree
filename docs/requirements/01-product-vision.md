## Product Vision

SubNetree is a modular, source-available network monitoring and management platform that provides unified device discovery, monitoring, remote access, credential management, and IoT awareness in a single self-hosted application.

**Strategic Intent:** Free for personal and home use forever. Built to become a commercial product for business, MSP, and enterprise use. The codebase, documentation, community, and clean IP chain are the product -- designed from day one for acquisition readiness.

**Target Users:** HomeLab enthusiasts, prosumers, and small business IT administrators.

**Strategic Position: "Start Here, Grow Anywhere."** SubNetree is a gateway product -- jack of all trades, master of none, by design. Each module provides 80% of what a dedicated tool offers. When users outgrow a module, SubNetree helps them graduate to specialized free/low-cost tools via standard protocols (Prometheus, MQTT, REST, Ansible YAML) while remaining the central discovery and inventory engine. No competing tool auto-discovers infrastructure; SubNetree eliminates the manual inventory that every other tool requires.

**Market Scope:** SubNetree targets single-subnet home and small-office networks (typically 15–200 devices). The current focus is building a product that delights HomeLabbers and small business users, not competing head-to-head with enterprise monitoring platforms. The backend architecture is modular, well-documented, and acquisition-ready -- the same standard protocols that connect homelab tools are what enterprise acquirers need.

**Core Value Proposition:** No existing source-available tool combines device discovery, monitoring, remote access, credential management, and IoT awareness in a single product. Free for personal, HomeLab, and non-competing production use. BSL 1.1 licensed core with Apache 2.0 plugin SDK for ecosystem growth.

### What It Does

**Discovery & Mapping:**

- LAN scanning and device detection
- Device identification (OS, manufacturer, type)
- Network topology mapping and visualization

**Monitoring:**

- Health and status monitoring via active scanning or lightweight agents (Scout)
- Plugins can monitor anything - the platform provides the framework, plugins provide the data

**Quick Access:**

- One-click access to systems and services
- Credential vault so you don't type passwords hundreds of times a day
- Launch RDP, SSH, web UIs directly from the dashboard

**Rich, Customizable UI:**

- List views, tree views, charts, graphs, gauges
- Various status indicators (up/down, health, alerts)
- Highly customizable - users display what matters to them

**Plugin Extensibility:**

- Open plugin architecture for anything users want to add
- Data can come from active scanning/detection or from helper agents on hosts
- Community and third-party plugins welcome

### Design Philosophy

1. **Ease of use first.** You should not need a tech degree to operate SubNetree. The interface should be intuitive enough that a non-technical small business owner can understand their network health at a glance, while an experienced sysadmin can drill into the detail they need.

2. **Sensible defaults, deep customization.** SubNetree ships fully preconfigured for rapid deployment -- install and go. But the true power lies in the ability to configure and customize every aspect of the system: dashboards, alerts, scan schedules, notification channels, plugins, and themes. The defaults get you running; customization makes it yours.

3. **Plugin-powered extensibility.** The plugin architecture is not an afterthought -- it is the architecture. Every major feature is a plugin. Users and third-party developers can replace, extend, or supplement any module. The system is designed to be shaped by its users, not constrained by its authors.

4. **Stability and security are non-negotiable.** These are not features that ship "later." Every release must be stable enough to trust with production infrastructure and secure enough to trust with network credentials. If a feature compromises stability or security, it does not ship.

5. **Time to First Value under 10 minutes.** Users will forgive missing features but will not forgive a bad first experience. Download, install, see your network -- in minutes, not hours.

6. **Gateway depth: 80% of dedicated tools.** Each module should handle the common cases well. When users need the remaining 20%, SubNetree provides growth pathways to specialized tools rather than trying to replicate enterprise features.

7. **Standard protocols for integration.** Export Prometheus metrics, publish MQTT events, expose REST APIs, generate Ansible-compatible YAML. Users should never feel locked in.

8. **Free/low-cost growth paths first.** When recommending tools users can graduate to, prioritize free and open-source options (Uptime Kuma, Grafana, NetBox) before commercial alternatives.
