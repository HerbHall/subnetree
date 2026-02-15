# Competitive Tracking

Lightweight competitor analysis for SubNetree. Updated monthly. Data sourced from GitHub API and prior research (see `D:/DevSpace/research/HomeLab/analysis/`).

**Last updated**: 2026-02-15
**Next review**: 2026-03-15

---

## Competitor Snapshots

### NetAlertX

| Attribute | Value |
|-----------|-------|
| GitHub | [netalertx/NetAlertX](https://github.com/netalertx/NetAlertX) |
| Stars | 5,784 |
| License | GPL-3.0 |
| Language | Python |
| Created | 2021-12-23 |
| Activity | Active (regular releases) |

**Positioning**: ARP/DHCP/Nmap network scanner with device change alerts and Home Assistant MQTT integration. Closest to SubNetree's Recon module scope.

**Core capabilities**:

- ARP, DHCP, ping/ICMP, Nmap scanning
- Device change detection (IP, MAC, hostname, open ports)
- 80+ notification channels via Apprise
- Home Assistant MQTT integration (native)
- Multi-VLAN/subnet support
- Pi-hole / AdGuard Home / UNIFI controller import
- SNMP device import (FortiGate)

**Key weaknesses**:

- No health monitoring or uptime checks
- No agent-based collection
- No credential vault or secrets management
- No configuration documentation generation
- No plugin architecture (monolithic codebase)
- GPL-3.0 limits commercial adoption
- Python runtime heavier than Go binary

**Source**: RF-006, `community-validation-reddit-2026-02-14.md`

---

### Beszel

| Attribute | Value |
|-----------|-------|
| GitHub | [henrygd/beszel](https://github.com/henrygd/beszel) |
| Stars | 19,338 |
| License | MIT |
| Language | Go |
| Created | 2024-07-07 |
| Activity | Active (v0.18.3, rapid growth -- 0 to 19k stars in 18 months) |

**Positioning**: Lightweight server monitoring hub. Positioned as "the perfect middle ground between Uptime Kuma and Grafana+Prometheus."

**Core capabilities**:

- Go-based agent with SSH+WebSocket dual transport (CBOR encoding)
- CPU, RAM, disk, temperature, GPU monitoring
- Docker container stats
- S.M.A.R.T. disk health
- Historical data with charts
- Alerts and notifications
- Auto-update mechanism (systemd, OpenRC, procd, FreeBSD)

**Key weaknesses**:

- No network discovery (manual agent install per host)
- No credential vault
- No configuration documentation
- No topology visualization
- No plugin architecture
- SSH-based agent communication (vs SubNetree's gRPC + mTLS)
- 210 open issues (growing backlog)
- Manual enrollment (key/token copy)

**Source**: RF-010, RF-006, `community-validation-reddit-2026-02-14.md`

---

### Scanopy

| Attribute | Value |
|-----------|-------|
| GitHub | [scanopy/scanopy](https://github.com/scanopy/scanopy) |
| Stars | 4,060 |
| License | AGPL-3.0 / Commercial dual-license |
| Language | Rust (backend), SvelteKit (frontend) |
| Created | 2025-09-29 (as NetVisor, renamed Dec 2025) |
| Activity | Active (aggressive release cadence, ~1 every 2 days, bus factor = 1) |

**Positioning**: Network topology discovery and visualization. "Clean network diagrams. One-time setup, zero upkeep."

**Core capabilities**:

- ARP host discovery with dynamic concurrency
- TCP/UDP port scanning (800ms timeout, batch=200)
- 230+ service identification definitions
- SNMP enrichment (v2c only -- system MIB, LLDP/CDP)
- Docker container discovery
- Interactive topology (elkjs auto-layout)
- Multi-user RBAC (Owner/Admin/Member/Viewer)
- OIDC/SSO support
- Distributed scanning (multi-daemon)
- REST API with OpenAPI docs

**Key weaknesses**:

- No change history (each scan overwrites state)
- No alerting or notifications (webhooks marked "coming soon")
- No configuration drift detection
- No health monitoring
- No agent-based profiling
- No credential vault (users request `_FILE` env vars, issue #165)
- Multi-NIC hosts appear as duplicates (issue #477, 10 comments)
- Scanning can freeze host VMs (discussion #423)
- Docker-only deployment (requires `privileged: true` + `network_mode: host`)
- Bus factor = 1 (93%+ commits from sole maintainer)
- AGPL-3.0 limits commercial ecosystem
- Rust raises contributor barrier

**Source**: RF-001, `scanopy-competitive-analysis.md`, `scanopy-gap-exploitation-2026-02-14.md`

---

### WatchYourLAN

| Attribute | Value |
|-----------|-------|
| GitHub | [aceberg/WatchYourLAN](https://github.com/aceberg/WatchYourLAN) |
| Stars | 6,763 |
| License | MIT |
| Language | Go |
| Created | 2022-08-15 |
| Activity | Maintained (slower cadence) |

**Positioning**: Simple Go-based ARP LAN scanner with device history.

**Core capabilities**:

- ARP scanning with device history
- Notifications via Shoutrrr
- Grafana export
- PostgreSQL support
- Simple web UI

**Key weaknesses**:

- ARP-only scanning (no mDNS, SNMP, UPnP)
- No health monitoring
- No agent-based collection
- No credential vault
- No topology visualization
- No VLAN awareness
- No plugin architecture
- No configuration documentation
- Narrow scope (LAN inventory only)

**Source**: RF-006, `community-validation-reddit-2026-02-14.md`

---

## Feature Comparison Matrix

| Capability | SubNetree | NetAlertX | Beszel | Scanopy | WatchYourLAN |
|------------|-----------|-----------|--------|---------|--------------|
| **Discovery** | | | | | |
| ARP scanning | Yes | Yes | -- | Yes | Yes |
| mDNS discovery | Yes | -- | -- | -- | -- |
| UPnP/SSDP discovery | Yes | -- | -- | -- | -- |
| SNMP enrichment | Yes | Partial | -- | Yes (v2c) | -- |
| Docker container discovery | -- | -- | Yes (stats) | Yes | -- |
| **Inventory** | | | | | |
| Device inventory | Yes | Yes | -- | Yes | Yes |
| Hardware profiling | Yes (Scout) | -- | Yes (agent) | -- | -- |
| Stale device detection | Yes | Yes | -- | -- | -- |
| **Monitoring** | | | | | |
| Health checks (ICMP/TCP/HTTP) | Yes | Partial (ping) | Yes | -- | -- |
| Dependency-aware suppression | Yes | -- | -- | -- | -- |
| Anomaly detection (EWMA) | Yes | -- | -- | -- | -- |
| Metrics history | Yes (30d) | -- | Yes | -- | -- |
| Docker container stats | Yes (Scout) | -- | Yes | -- | -- |
| **Alerts** | | | | | |
| Alert management | Yes | Yes | Yes | -- | Partial |
| Webhook notifications | Yes | Yes | Yes | -- | -- |
| Alertmanager-compatible | Yes | -- | -- | -- | -- |
| Notification channels | Webhooks | 80+ (Apprise) | Built-in | -- | Shoutrrr |
| **Visualization** | | | | | |
| Topology visualization | Yes | -- | -- | Yes | -- |
| Service mapping | Yes | -- | -- | Yes (230+) | -- |
| **Security** | | | | | |
| Credential vault | Yes | -- | -- | -- | -- |
| mTLS agent communication | Yes | -- | -- | -- | -- |
| **Agent** | | | | | |
| Agent-based profiling | Yes (Scout) | -- | Yes | -- | -- |
| Auto-update mechanism | Yes | -- | Yes | -- | -- |
| Multi-platform agent | Yes (Win+Linux) | -- | Yes | -- | -- |
| **Analytics** | | | | | |
| AI/NL queries | Yes | -- | -- | -- | -- |
| Recommendation engine | Yes (48 tools) | -- | -- | -- | -- |
| **Integration** | | | | | |
| MQTT / Home Assistant | Yes | Yes | Partial | -- | -- |
| Prometheus /metrics | Yes | -- | -- | -- | Grafana export |
| CSV import/export | Yes | -- | -- | -- | -- |
| **Architecture** | | | | | |
| Plugin architecture | Yes | -- | -- | -- | -- |
| REST API | Yes | Partial | Yes (PocketBase) | Yes (OpenAPI) | -- |
| Tier-aware defaults | Yes | -- | -- | -- | -- |

**Legend**: Yes = shipped, Partial = limited implementation, -- = not available

---

## Monthly Check Template

Use this checklist on the 15th of each month. Takes about 5 minutes with `gh` CLI.

### Quick commands

```bash
# Star counts (run all four)
gh api repos/netalertx/NetAlertX --jq '.stargazers_count'
gh api repos/henrygd/beszel --jq '.stargazers_count'
gh api repos/scanopy/scanopy --jq '.stargazers_count'
gh api repos/aceberg/WatchYourLAN --jq '.stargazers_count'

# Latest releases
gh release list -R netalertx/NetAlertX --limit 3
gh release list -R henrygd/beszel --limit 3
gh release list -R scanopy/scanopy --limit 3
gh release list -R aceberg/WatchYourLAN --limit 3

# Open issue counts
gh api repos/netalertx/NetAlertX --jq '.open_issues_count'
gh api repos/henrygd/beszel --jq '.open_issues_count'
gh api repos/scanopy/scanopy --jq '.open_issues_count'
gh api repos/aceberg/WatchYourLAN --jq '.open_issues_count'
```

### Checklist

- [ ] **Star counts**: Record in tracking log below. Look for sudden jumps (viral moment) or plateaus (stagnation).
- [ ] **New releases**: Check changelogs for features that close gaps against SubNetree (alerting, state history, agent support, credential management, documentation).
- [ ] **Open issue trends**: Growing backlog = maintainer burnout risk. Declining = healthy project or abandoned triage.
- [ ] **New competitors**: Search GitHub for `homelab network discovery self-hosted` and `homelab infrastructure monitoring`. Check r/selfhosted and r/homelab mentions via blog aggregation.
- [ ] **Community sentiment**: Skim recent issues/discussions on each repo for recurring complaints or feature requests that SubNetree already solves.
- [ ] **Update this file**: Bump star counts, note new releases, flag any strategic changes.

### Tracking log

| Date | NetAlertX | Beszel | Scanopy | WatchYourLAN | Notes |
|------|-----------|--------|---------|--------------|-------|
| 2026-02-15 | 5,784 | 19,338 | 4,060 | 6,763 | Baseline. Beszel growing fastest. |
| 2026-03-15 | | | | | |
| 2026-04-15 | | | | | |
| 2026-05-15 | | | | | |
| 2026-06-15 | | | | | |

---

## Strategic Positioning Summary

SubNetree's competitive moat is **breadth combined with depth in areas competitors ignore**. The homelab monitoring space has fragmented into single-purpose tools: NetAlertX for device scanning, Beszel for server metrics, Uptime Kuma for service uptime, Scanopy for topology visualization. Users run 3-5 tools and complain about "tool fatigue." SubNetree is the only platform that unifies discovery, monitoring, alerting, credential management, and agent-based profiling in a single binary. The gateway philosophy (D-004) -- providing 80% of each tool's core value in one integrated platform -- directly addresses the community's most vocal pain point. Three capabilities have zero competition: encrypted credential vault, AI-powered analytics, and plugin-extensible architecture. The planned auto-documentation engine (issue #299) targets the single largest unmet need identified across all community research: no tool automatically documents infrastructure changes. Competitors would need fundamental architectural changes to match these capabilities, and the solo-maintainer projects (Scanopy bus factor = 1, WatchYourLAN low velocity) are unlikely to make them.
