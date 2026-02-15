# Issue #327 Scoping: Device Discovery and Monitoring

Scoping analysis for [issue #327](https://github.com/HerbHall/subnetree/issues/327) -- "Device discovery and monitoring."

## 1. Capability Audit

The issue requests: ping, traceroute, reverse DNS, MAC address vendor lookup, device type classification, network hierarchy building, and additional device identification methods. The table below maps each request to current codebase status.

| Feature | Requested | Status | File(s) | Notes |
|---------|-----------|--------|---------|-------|
| Ping (ICMP) | Yes | **Exists** | [icmp.go](../../internal/recon/icmp.go), [checker.go](../../internal/pulse/checker.go) | Recon uses `pro-bing` for subnet sweep; Pulse uses it for ongoing health checks. Both support configurable count, timeout, concurrency, and privileged mode on Windows. |
| Reverse DNS lookup | Yes | **Exists** | [scanner.go:203-212](../../internal/recon/scanner.go#L203) | `resolveHostname()` calls `net.DefaultResolver.LookupAddr()` with 500ms timeout during every scan. Hostname is stored on the device record. |
| MAC address discovery (ARP) | Yes | **Exists** | [arp.go](../../internal/recon/arp.go) | Cross-platform ARP table reader (Linux `/proc/net/arp`, Windows `arp -a`, macOS `arp -a`). Enabled by default. |
| MAC vendor lookup (OUI) | Yes | **Exists** | [oui.go](../../internal/recon/oui.go) | Embedded 40k-entry OUI table. `Lookup(mac)` extracts first 3 octets and returns manufacturer name. Used during scan orchestration. |
| SNMP discovery | Yes | **Exists** | [snmp_collector.go](../../internal/recon/snmp_collector.go), [snmp_oids.go](../../internal/recon/snmp_oids.go) | SNMPv2c and SNMPv3 with full auth. Queries sysDescr, sysObjectID, sysUpTime, sysContact, sysName, sysLocation. Walks IF-MIB for interface table. |
| mDNS discovery | Implicit | **Exists** | [mdns.go](../../internal/recon/mdns.go) | Passive mDNS/Bonjour listener querying 14 service types. Infers device type from service name. Background goroutine with configurable interval. |
| UPnP/SSDP discovery | Implicit | **Exists** | [upnp.go](../../internal/recon/upnp.go) | UPnP/SSDP device discovery via `goupnp`. Extracts manufacturer, model, device type from UPnP descriptors. Background goroutine. |
| Device type classification | Yes | **Partial** | [snmp_collector.go:393-414](../../internal/recon/snmp_collector.go#L393), [mdns.go:250-263](../../internal/recon/mdns.go#L250), [upnp.go:220-253](../../internal/recon/upnp.go#L220), [device.go](../../pkg/models/device.go) | 15 device types defined. SNMP infers from sysDescr, mDNS from service name, UPnP from device type URN. ICMP-only scans always produce `unknown` type. |
| Device categorization fields | Partial | **Exists** | [device.go:51-71](../../pkg/models/device.go#L51) | Device model has `Category`, `PrimaryRole`, `Owner`, `Location`, `Tags`, `CustomFields`. These are user-editable via API but not auto-populated. |
| Topology / hierarchy tree | Yes | **Partial** | [scanner.go:214-253](../../internal/recon/scanner.go#L214), [store.go:412-460](../../internal/recon/store.go#L412) | `inferTopologyLinks()` creates edges between devices and subnet gateway (first usable IP). Only link type is `arp`. No switch port mapping, no multi-hop awareness. |
| Traceroute | Yes | **Does NOT exist** | -- | No traceroute implementation anywhere in the codebase. |
| Switch port mapping | Yes | **Does NOT exist** | -- | No SNMP bridge-MIB or forwarding database (FDB) table walking. Cannot determine which device is connected to which switch port. |
| Device auto-classification from fingerprinting | Yes | **Does NOT exist** | -- | No multi-signal fingerprinting. Each discovery method infers type independently (sysDescr keywords, mDNS service, UPnP URN) but there is no combined heuristic engine. ICMP-only devices stay `unknown`. |
| Network hierarchy (WAN -> router -> switch -> devices) | Yes | **Does NOT exist** | -- | Current topology is flat: devices connect to gateway via `arp` link type. No WAN edge detection, no router-switch-host layering, no per-port connectivity. |
| TCP/HTTP health checks | Implicit | **Exists** | [tcp_checker.go](../../internal/pulse/tcp_checker.go), [http_checker.go](../../internal/pulse/http_checker.go) | TCP port connect and HTTP GET checkers with latency measurement. |
| Service mapping | Implicit | **Exists** | [service.go](../../pkg/models/service.go) | Service model supports Docker containers, systemd services, Windows services, applications. Linked to devices. |

### Summary

- **Already fully implemented (6):** Ping/ICMP, reverse DNS, ARP scanning, MAC vendor lookup (OUI), SNMP discovery, mDNS/UPnP discovery
- **Partially implemented (2):** Device type classification (works per-protocol but no combined engine), topology (flat gateway links only)
- **Genuinely new (4):** Traceroute, switch port mapping, multi-signal device fingerprinting, hierarchical network topology

## 2. Proposed Sub-Issues

### Issue A: Traceroute utility endpoint

**Title:** `feat(recon): add traceroute API endpoint for network path discovery`

**Description:** Add a traceroute implementation that traces the network path from the SubNetree server to a target IP. Expose as `POST /api/v1/recon/traceroute` accepting a target IP and returning an ordered list of hops with RTT, hostname, and IP. Use raw ICMP with incrementing TTL (like `pro-bing`) to avoid requiring external `traceroute`/`tracert` binaries. Results should be stored temporarily for topology enrichment.

**Estimated size:** M (medium)

**Key files to modify:**

- `internal/recon/traceroute.go` (new) -- core traceroute logic
- `internal/recon/handlers.go` -- add `/traceroute` route and handler
- `internal/recon/recon.go` -- register new route

**Dependencies:** None. Standalone utility.

---

### Issue B: SNMP bridge-MIB switch port mapping

**Title:** `feat(recon): SNMP bridge-MIB walk for switch port-to-MAC mapping`

**Description:** Walk SNMP bridge-MIB (`.1.3.6.1.2.1.17`) and Q-BRIDGE-MIB (`.1.3.6.1.2.1.17.7`) on managed switches to build a port-to-MAC forwarding table. This reveals which devices are physically connected to which switch ports, enabling accurate Layer 2 topology instead of the current flat gateway-link model. Store results as enriched `TopologyLink` entries with `link_type: "bridge-fdb"` and port metadata.

**Estimated size:** L (large)

**Key files to modify:**

- `internal/recon/snmp_oids.go` -- add bridge-MIB OID constants
- `internal/recon/snmp_collector.go` -- add `GetForwardingTable()` method
- `internal/recon/store.go` -- extend `TopologyLink` with port fields
- `internal/recon/migrations.go` -- migration for new topology link columns
- `internal/recon/handlers.go` -- add `/snmp/fdb/{device_id}` endpoint
- `internal/recon/scanner.go` -- integrate FDB data into topology inference

**Dependencies:** Requires SNMP credentials on managed switches (already supported via Vault). Benefits from Issue D for full hierarchy.

---

### Issue C: Multi-signal device fingerprinting engine

**Title:** `feat(recon): multi-signal device type fingerprinting engine`

**Description:** Create a device fingerprinting engine that combines signals from multiple discovery methods to classify devices more accurately. Inputs include: OUI manufacturer (existing), SNMP sysDescr and sysObjectID (existing), mDNS service types (existing), UPnP device type URN (existing), open TCP ports (new scan), and HTTP server headers (new probe). Implement as a scoring/weighted system where each signal contributes evidence toward a device type. Run automatically after scan completion to upgrade `unknown` devices. Expose classification confidence as a field on the device model.

**Estimated size:** L (large)

**Key files to modify:**

- `internal/recon/fingerprint.go` (new) -- fingerprinting engine with signal aggregation
- `internal/recon/fingerprint_rules.go` (new) -- rule definitions mapping signals to device types
- `internal/recon/scanner.go` -- invoke fingerprinting after scan enrichment
- `pkg/models/device.go` -- add `ClassificationConfidence float64` and `ClassificationSource string` fields
- `internal/recon/store.go` -- persist new fields
- `internal/recon/migrations.go` -- migration for new device columns

**Dependencies:** Standalone, but benefits from port scanning (can be added incrementally). Issue B data (switch port info) would further improve classification.

---

### Issue D: Hierarchical network topology (WAN -> router -> switch -> devices)

**Title:** `feat(recon): hierarchical network topology with device role layering`

**Description:** Extend the topology system to represent network hierarchy: WAN edge -> router -> switches/hubs -> end devices. Use a combination of: (1) gateway detection from default route, (2) SNMP sysObjectID enterprise OIDs to identify network infrastructure, (3) bridge-MIB FDB tables to map switch-to-device connections, and (4) traceroute hop data to infer multi-hop paths. Add `topology_layer` field to `TopologyLink` (values: `wan`, `core`, `distribution`, `access`, `endpoint`). Update the topology API response to include layer information for the React Flow frontend to render a hierarchical layout.

**Estimated size:** L (large)

**Key files to modify:**

- `internal/recon/store.go` -- add layer field to `TopologyLink`, hierarchical query methods
- `internal/recon/migrations.go` -- migration for layer field
- `internal/recon/scanner.go` -- enhanced topology inference with layering
- `internal/recon/handlers.go` -- update `TopologyGraph` response with layer data

**Dependencies:** Heavily depends on Issue B (switch port mapping) for accuracy. Benefits from Issue A (traceroute) for multi-hop path detection and Issue C (fingerprinting) for infrastructure device identification.

---

### Issue E: TCP port scan for service fingerprinting

**Title:** `feat(recon): TCP port scan for common service detection`

**Description:** Add a lightweight TCP port scan phase to the scan orchestrator. Probe a configurable set of common ports (22, 80, 443, 445, 3389, 8080, 8443, etc.) on discovered hosts. Store open ports on the device record and use them as inputs for the fingerprinting engine (Issue C). For example: port 22 open suggests Linux/server, port 3389 suggests Windows/desktop, port 9100 suggests printer. Keep the default port list small (top 20-30) to avoid slow scans.

**Estimated size:** M (medium)

**Key files to modify:**

- `internal/recon/port_scanner.go` (new) -- TCP connect scanner with configurable port list
- `internal/recon/scanner.go` -- add port scan phase after ICMP sweep
- `pkg/models/device.go` -- add `OpenPorts []int` field
- `internal/recon/store.go` -- persist open ports
- `internal/recon/migrations.go` -- migration for ports column
- `internal/recon/config.go` -- add `PortScanEnabled`, `PortScanPorts`, `PortScanTimeout` config

**Dependencies:** None. Feeds into Issue C (fingerprinting).

---

### Issue F: On-demand network diagnostic tools API

**Title:** `feat(recon): on-demand network diagnostic tools (ping, DNS lookup, port check)`

**Description:** Add interactive diagnostic tool endpoints that users can invoke from the dashboard UI for troubleshooting. Endpoints: `POST /api/v1/recon/tools/ping` (ping a target with configurable count), `POST /api/v1/recon/tools/dns` (forward and reverse DNS lookup), `POST /api/v1/recon/tools/port-check` (test TCP connectivity to a specific port). These are one-shot utilities distinct from the automated scan pipeline. Return results immediately (synchronous, with timeout).

**Estimated size:** S (small)

**Key files to modify:**

- `internal/recon/tools.go` (new) -- diagnostic tool handlers
- `internal/recon/recon.go` -- register new routes
- `internal/recon/handlers.go` -- add route entries

**Dependencies:** None. Uses existing `pro-bing`, `net.LookupAddr`, and TCP connect primitives.

## 3. Dependency Graph

```text
Issue F (diagnostic tools) -----> standalone
Issue A (traceroute)       -----> standalone, feeds into D
Issue E (port scan)        -----> standalone, feeds into C
Issue C (fingerprinting)   -----> standalone, enhanced by E and B
Issue B (bridge-MIB FDB)   -----> standalone, feeds into D
Issue D (hierarchy)        -----> depends on B, benefits from A and C
```

## 4. Recommendation

### Current Sprint

1. **Issue F (diagnostic tools) -- S** -- Highest user-facing value for the least effort. Directly addresses the issue's request for "tools to allow user to troubleshoot issues on the network." Can ship independently.
2. **Issue A (traceroute) -- M** -- Directly requested. Standalone implementation. Also produces data that feeds into hierarchy building later.
3. **Issue E (port scan) -- M** -- Enables device fingerprinting. Low risk, well-scoped. Builds on existing TCP connect patterns from `pulse/tcp_checker.go`.

### Next Sprint

4. **Issue C (fingerprinting engine) -- L** -- Combines all existing signals plus port scan data. Biggest impact on reducing `unknown` device types.
5. **Issue B (bridge-MIB FDB) -- L** -- Required foundation for accurate physical topology. Needs real managed switch hardware for testing.

### Defer

6. **Issue D (hierarchical topology) -- L** -- Most complex. Depends on B and benefits from A and C. Should be the capstone after the foundation pieces are in place.

### Issue #327 Disposition

Close #327 and replace with the six child issues above. The original issue is too broad to be actionable -- it mixes features that already exist (ping, reverse DNS, OUI lookup) with genuinely new work (traceroute, port mapping, hierarchy). The child issues are independently shippable and have clear scope.

Update #327's body with a checklist linking to each child issue before closing, so the original reporter can track progress:

```markdown
Replaced by:
- [ ] #NNN feat(recon): on-demand network diagnostic tools
- [ ] #NNN feat(recon): traceroute API endpoint
- [ ] #NNN feat(recon): TCP port scan for service detection
- [ ] #NNN feat(recon): multi-signal device fingerprinting engine
- [ ] #NNN feat(recon): SNMP bridge-MIB switch port mapping
- [ ] #NNN feat(recon): hierarchical network topology
```

## 5. What the Issue Reporter Already Has

The reporter may not realize that SubNetree already supports:

- **ICMP ping** during network scans (configurable count, timeout, concurrency)
- **Reverse DNS** on every discovered host (automatic, 500ms timeout)
- **MAC address collection** from the system ARP table (cross-platform)
- **Vendor identification** from MAC address OUI prefix (40k embedded entries)
- **SNMP device discovery** with sysDescr-based type inference (router, switch, firewall, printer, AP, NAS, server)
- **mDNS/Bonjour** passive discovery (14 service types)
- **UPnP/SSDP** device discovery with manufacturer and model extraction
- **Device type taxonomy** with 15 categories (server, desktop, laptop, mobile, router, switch, printer, IoT, AP, firewall, NAS, phone, tablet, camera, unknown)
- **Basic topology** linking devices to their subnet gateway

Consider updating the issue with this information before creating child issues, so the reporter understands the starting point.
