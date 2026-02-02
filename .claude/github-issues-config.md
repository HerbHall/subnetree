# GitHub Issues Configuration -- NetVantage

This file is read by the `/manage-github-issues` skill to provide project-specific context.

## Project

- **Name**: NetVantage
- **Repo**: HerbHall/netvantage
- **Description**: Modular network monitoring and management platform (Go)

## Roadmap

- **Roadmap file**: `docs/requirements/21-phased-roadmap.md`
- **Requirements directory**: `docs/requirements/`

## Phases

| Phase | Label | Milestone | Description |
|-------|-------|-----------|-------------|
| Phase 0 | `phase:0` | `Phase 0` | Pre-development infrastructure |
| Phase 1 | `phase:1` | `Phase 1: Foundation` | Server + dashboard + discovery + topology |
| Phase 1 (Testing) | `phase:1` | `Phase 1: Testing & Quality` | Test infrastructure and quality gates |
| Phase 1b | `phase:1b` | `Phase 1b: Windows Scout Agent` | Windows Scout agent |
| Phase 2 | `phase:2` | `Phase 2: Core Monitoring + Multi-Tenancy` | Core monitoring + multi-tenancy |
| Phase 3 | `phase:3` | `Phase 3: Remote Access + Credential Vault` | Remote access + credential vault |
| Phase 4 | `phase:4` | `Phase 4: Extended Platform` | Extended platform (IoT, marketplace, RBAC) |

## Module Labels

| Label | Scope / Keywords |
|-------|-----------------|
| `mod:core` | Server, HTTP, middleware, config, plugin registry, event bus |
| `mod:recon` | Ping, ARP, OUI, SNMP, mDNS, UPnP, network scan, device discovery |
| `mod:pulse` | Uptime, monitoring, alert, threshold, notification |
| `mod:dispatch` | Agent management, enrollment, certificate, dispatch |
| `mod:vault` | Credential, encryption, key management, vault, secret |
| `mod:gateway` | SSH, RDP, VNC, proxy, remote access, tunnel |
| `mod:scout` | Scout binary, system metrics, agent binary, heartbeat |
| `mod:dashboard` | React, UI, page, component, dashboard, chart, topology view |

## Phase-to-Requirements Mapping

Read ONLY the listed files when generating issues for a phase. Read ONE file at a time.

### Phase 0: Pre-Development Infrastructure
- Requirements: `docs/requirements/20-project-infrastructure.md`

### Phase 1: Foundation
- Server/API: `docs/requirements/11-api-design.md`
- Dashboard: `docs/requirements/13-dashboard-architecture.md`
- Topology: `docs/requirements/14-topology-visualization.md`
- Auth: `docs/requirements/07-authentication.md`
- Database: `docs/requirements/06-database-layer.md`
- Plugin system: `docs/requirements/04-plugin-architecture.md`
- Scanning/Recon: `docs/requirements/10-data-model.md`
- Testing: `docs/requirements/18-testing-strategy.md`
- Brand/UI: `docs/requirements/12-brand-identity.md`

### Phase 1b: Windows Scout Agent
- Agent: `docs/requirements/08-scout-agent.md`
- Dispatch: `docs/requirements/10-data-model.md`

### Phase 2: Core Monitoring + Multi-Tenancy
- Monitoring: `docs/requirements/10-data-model.md`
- Tailscale: `docs/requirements/09-tailscale-integration.md`
- Analytics: `docs/requirements/17-ai-analytics.md`
- Deployment: `docs/requirements/19-deployment.md`
- Observability: `docs/requirements/16-observability.md`

### Phase 3: Remote Access + Credential Vault
- Gateway: `docs/requirements/10-data-model.md`
- Vault: `docs/requirements/15-credential-vault.md`
- AI: `docs/requirements/17-ai-analytics.md`

### Phase 4: Extended Platform
- API: `docs/requirements/11-api-design.md`
- Commercialization: `docs/requirements/23-commercialization.md`
- AI: `docs/requirements/17-ai-analytics.md`
