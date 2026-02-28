# Homelab Validation Test Plan

Reusable test plan for validating SubNetree across three homelab platforms:
Docker Desktop, UNRAID, and Proxmox VE. Each phase references these
checklists. See epic issue #486 for the full dependency graph.

## Deployment Phases

| Phase | Platform | Server Method | Issues |
|-------|----------|---------------|--------|
| Phase 1 | Docker Desktop (Windows) | docker-compose | #493, #494 |
| Phase 2 | UNRAID | Docker container | #495, #496 |
| Phase 3 | Proxmox VE | LXC container or VM | #497, #498 |

Each phase: clean install, network scanning, Scout deployment, full
feature validation, multi-platform UI testing, bug triage, fix round.

## Checklist 1: Server Deployment

Reused each phase. Copy into phase A issues.

- [ ] Container/VM starts without errors
- [ ] Web UI accessible from local network (not just localhost)
- [ ] Setup wizard completes (admin account creation)
- [ ] Scan subnet(s) configured matching local network ranges
- [ ] Initial scan completes without errors
- [ ] Dashboard populates with discovered devices
- [ ] Topology view renders
- [ ] Persistence: stop container, restart, verify data survives
- [ ] Logs clean (no panics, no repeated errors)
- [ ] Resource baseline: record memory/CPU after 10 minutes idle

## Checklist 2: Network Discovery Validation

- [ ] ARP scan discovers devices on primary subnet
- [ ] mDNS discovery finds Apple/Chromecast/IoT devices
- [ ] UPnP/SSDP discovery finds media devices and routers
- [ ] SNMP scan discovers managed switches (if SNMP-enabled devices exist)
- [ ] LLDP neighbors detected (if supported switches exist)
- [ ] Device classification assigns correct types (router, switch, NAS, etc.)
- [ ] Multi-subnet scanning (if VLANs available -- document which subnets reachable)
- [ ] WiFi AP client enumeration (if APs are discoverable)

## Checklist 3: Scout Agent Deployment Playbook

Same 3-5 hosts enrolled fresh each phase:

| Host | OS | Install Method | Validates |
|------|----|---------------|-----------|
| Proxmox host | Linux (Debian) | curl one-liner | Hardware profiling, VM/LXC inventory |
| UNRAID server | Linux (Slackware) | curl one-liner | NAS profiling, Docker container stats |
| Windows workstation | Windows 11 | PowerShell one-liner / Inno Setup | Service mode, GPU collection |
| Raspberry Pi | Linux (Debian ARM) | curl one-liner | Tier 0 resource validation |
| Additional device | varies | varies | Coverage breadth |

Per Scout host:

- [ ] One-click install script succeeds
- [ ] mTLS enrollment completes
- [ ] Hardware profile populates in server UI (CPU, RAM, storage)
- [ ] GPU data collected (if applicable)
- [ ] Docker container stats flowing (if Docker installed)
- [ ] Agent appears in Dispatch module agent list
- [ ] Agent auto-update mechanism works (stop/start cycle)

## Checklist 4: Feature Validation Matrix

Run against each server deployment:

| Module | Test | Pass Criteria |
|--------|------|---------------|
| **Dashboard** | All widgets populated | Real device counts, health scores, scan status |
| **Devices** | Device list correct | All discovered devices listed, correct types, sortable |
| **Topology** | Network map accurate | Hierarchy reflects real infrastructure, links correct |
| **Alerts** | Health checks active | ICMP/TCP/HTTP checks running, alerts fire on real conditions |
| **Auto-docs** | Per-device Markdown | Renders with real hardware/services/alerts/changelog |
| **Vault** | Credential storage | Store SNMP community string, use in scan, verify encrypted at rest |
| **MCP** | Claude Desktop query | Query devices via stdio transport |
| **Metrics** | Historical data | Data accumulates over 24-48 hours, graphs render |
| **CSV** | Export/import | Export inventory, reimport on fresh instance, data matches |
| **Themes** | Theme switching | All 19 themes render correctly |
| **Settings** | Configuration | Scan intervals, notification settings persist across restart |
| **Diagnostic tools** | Ping, DNS, traceroute | Execute from UI against real targets |
| **Demo mode** | Toggle on/off | Demo mode activates/deactivates cleanly |
| **Scan analytics** | Health scoring | Scan phases show timing, health score calculated |
| **SNMP FDB** | Switch port mapping | FDB table walks return real data (requires managed switch) |
| **Tailscale** | Overlay plugin | Tailscale devices appear if Tailscale is running |
| **MFA/TOTP** | Two-factor auth | Enable TOTP, verify login requires code, disable works |
| **NetBox export** | DCIM sync | Dry-run produces valid payload, live sync creates devices |
| **HA MQTT** | Auto-discovery | Devices and alerts appear in Home Assistant |

## Checklist 5: Multi-Platform UI Test Matrix

| Client | OS | Browsers | Tests |
|--------|----|----------|-------|
| Workstation 1 | Windows 11 | Chrome, Firefox, Edge | Full feature matrix |
| Workstation 2 | Linux | Chrome, Firefox | Full feature matrix |
| Workstation 3 | macOS (if available) | Chrome, Safari | Full feature matrix |
| Mobile/Tablet | iOS/Android | Safari/Chrome mobile | Responsive layout, read-only nav |

Per browser:

- [ ] Login/setup wizard renders correctly
- [ ] Dashboard layout correct, no overflow/clipping
- [ ] Topology zoom/pan/drag works
- [ ] Device detail page renders all sections
- [ ] Theme switching applies correctly
- [ ] Tables sort/filter/paginate
- [ ] Modals/dialogs open and close cleanly
- [ ] Forms submit successfully (settings, vault, scan config)
- [ ] No console errors (check DevTools)

## Checklist 6: Network Infrastructure Tests

Setup-dependent -- document what's available and skip what isn't:

- [ ] **Multi-subnet/VLAN scanning**: List available VLANs, verify scanner reaches each
- [ ] **SNMP-enabled devices**: Document which devices have SNMP, test community string in vault
- [ ] **WiFi AP client enumeration**: Identify any APs, test nl80211/netsh discovery
- [ ] **Reverse proxy**: Test behind UNRAID Nginx proxy or Caddy (path rewriting, WebSocket)
- [ ] **Browser matrix edge cases**: Test from devices on different subnets/VLANs

## Checklist 7: Stability Soak (24-48 Hours)

Run between Phase B completion and next Phase A start:

- [ ] Server running continuously for 24+ hours
- [ ] Memory usage stable (no leak -- compare to 10-minute baseline)
- [ ] CPU usage at idle < 5%
- [ ] Scheduled scans executing on time
- [ ] No log errors accumulating
- [ ] Scout agents remain connected, heartbeats flowing
- [ ] Alerts fire and resolve correctly over time

## Pass/Fail Criteria

### Phase passes when

- All Server Deployment Checklist items green
- 80%+ of Network Discovery items green (some depend on available hardware)
- All Scout agents enrolled and reporting
- All Feature Validation "core" items green (Dashboard, Devices, Topology, Alerts, Auto-docs)
- No critical or high-severity bugs open from this phase
- UI renders correctly on at least 2 platforms x 2 browsers

### Phase fails when

- Server crashes or data corruption
- Scout enrollment fails on any host
- Core features non-functional
- Memory leak (>2x baseline after 24 hours)

## Bug Severity Definitions

| Severity | Definition | Example | Phase Gate |
|----------|-----------|---------|------------|
| **Critical** | Data loss, crash, security flaw | DB corruption, panic on scan | Blocks phase completion |
| **High** | Core feature broken | Topology doesn't render, Scout can't enroll | Blocks phase completion |
| **Medium** | Feature degraded but usable | Wrong device type, UI alignment issue | Fix before next phase |
| **Low** | Cosmetic or minor UX | Tooltip text wrong, theme color off | Backlog |

## Bug Tracking Convention

File bugs as GitHub issues with labels:

- `bug` + `severity:critical` / `severity:high` / `severity:medium` / `severity:low`
- `validation:phase-1` / `validation:phase-2` / `validation:phase-3`
- Title prefix: `[Validation] <description>`

## Content Capture Reminders

During every phase, capture for #499:

- Screenshots of dashboard with real device data
- Topology view of actual network
- Scout agent list with real hardware profiles
- Before/after scan results
- Any impressive auto-generated device documentation
