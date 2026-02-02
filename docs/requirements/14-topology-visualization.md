## Topology Visualization

### Data Sources for Topology

| Protocol | Data Provided | Phase |
|----------|--------------|-------|
| LLDP (Link Layer Discovery Protocol) | Direct neighbor connections, port names | 1 |
| CDP (Cisco Discovery Protocol) | Cisco device neighbors | 1 |
| ARP Tables | IP-to-MAC mappings, indicate shared L2 segments | 1 |
| SNMP Interface Tables | Port descriptions, speeds, status | 2 |
| Traceroute | L3 path between devices | 2 |
| Agent-reported interfaces | Network connections from agent perspective | 1b |

### Topology Map Features (Phase 1)

- Auto-generated from discovery data (LLDP/CDP/ARP)
- Devices as nodes, connections as edges
- Color-coded by status (green=online, red=offline, yellow=degraded)
- Click device to see detail panel
- Click connection to see link speed, utilization
- Zoom, pan, auto-layout with manual override
- Export as PNG/SVG

### Topology Map Features (Phase 2)

- Real-time traffic utilization on links (color gradient: green -> yellow -> red)
- Overlay views: by device type, by subnet, by status
- Custom backgrounds (floor plans, rack diagrams)
- Saved layout persistence
