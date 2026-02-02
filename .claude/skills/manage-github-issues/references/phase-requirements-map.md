<phase_requirements_map>

This maps each project phase to the requirements documents that should be read
when generating issues for that phase. Read ONLY the listed files.

**Phase 0: Pre-Development Infrastructure**
- Roadmap: `docs/requirements/21-phased-roadmap.md` (Phase 0 section)
- Requirements: `docs/requirements/20-project-infrastructure.md`
- Milestone: `Phase 0`

**Phase 1: Foundation (Server + Dashboard + Discovery + Topology)**
- Roadmap: `docs/requirements/21-phased-roadmap.md` (Phase 1 section)
- Requirements (read only the one relevant to the specific issue):
  - Server/API: `docs/requirements/11-api-design.md`
  - Dashboard: `docs/requirements/13-dashboard-architecture.md`
  - Topology: `docs/requirements/14-topology-visualization.md`
  - Auth: `docs/requirements/07-authentication.md`
  - Database: `docs/requirements/06-database-layer.md`
  - Plugin system: `docs/requirements/04-plugin-architecture.md`
  - Scanning/Recon: `docs/requirements/10-data-model.md`
  - Testing: `docs/requirements/18-testing-strategy.md`
  - Brand/UI: `docs/requirements/12-brand-identity.md`
- Milestone: `Phase 1: Foundation`
- Sub-milestone: `Phase 1: Testing & Quality`

**Phase 1b: Windows Scout Agent**
- Roadmap: `docs/requirements/21-phased-roadmap.md` (Phase 1b section)
- Requirements:
  - Agent: `docs/requirements/08-scout-agent.md`
  - Dispatch: `docs/requirements/10-data-model.md`
- Milestone: `Phase 1b: Windows Scout Agent`

**Phase 2: Core Monitoring + Multi-Tenancy**
- Roadmap: `docs/requirements/21-phased-roadmap.md` (Phase 2 section)
- Requirements (read only the one relevant to the specific issue):
  - Monitoring: `docs/requirements/10-data-model.md`
  - Tailscale: `docs/requirements/09-tailscale-integration.md`
  - Analytics: `docs/requirements/17-ai-analytics.md`
  - Deployment: `docs/requirements/19-deployment.md`
  - Observability: `docs/requirements/16-observability.md`
- Milestone: `Phase 2: Core Monitoring + Multi-Tenancy`

**Phase 3: Remote Access + Credential Vault**
- Roadmap: `docs/requirements/21-phased-roadmap.md` (Phase 3 section)
- Requirements:
  - Gateway: `docs/requirements/10-data-model.md`
  - Vault: `docs/requirements/15-credential-vault.md`
  - AI: `docs/requirements/17-ai-analytics.md`
- Milestone: `Phase 3: Remote Access + Credential Vault`

**Phase 4: Extended Platform**
- Roadmap: `docs/requirements/21-phased-roadmap.md` (Phase 4 section)
- Requirements:
  - API: `docs/requirements/11-api-design.md`
  - Commercialization: `docs/requirements/23-commercialization.md`
  - AI: `docs/requirements/17-ai-analytics.md`
- Milestone: `Phase 4: Extended Platform`

</phase_requirements_map>

<module_to_label_map>

When creating issues, use this mapping to determine the module label:

| Topic / Keywords | Module Label |
|-----------------|--------------|
| Server, HTTP, middleware, config, plugin registry, event bus | `mod:core` |
| Ping, ARP, OUI, SNMP, mDNS, UPnP, network scan, device discovery | `mod:recon` |
| Uptime, monitoring, alert, threshold, notification | `mod:pulse` |
| Agent management, enrollment, certificate, dispatch | `mod:dispatch` |
| Credential, encryption, key management, vault, secret | `mod:vault` |
| SSH, RDP, VNC, proxy, remote access, tunnel | `mod:gateway` |
| Scout binary, system metrics, agent binary, heartbeat | `mod:scout` |
| React, UI, page, component, dashboard, chart, topology view | `mod:dashboard` |

</module_to_label_map>
