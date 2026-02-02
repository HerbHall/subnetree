## Data Model (Core Entities)

### Device

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| TenantID | UUID? | Tenant (null for single-tenant, populated in MSP mode) |
| Hostname | string | Device hostname |
| IPAddresses | []string | All known IP addresses |
| MACAddress | string | Primary MAC address |
| Manufacturer | string | Derived from OUI database |
| DeviceType | enum | server, desktop, laptop, mobile, router, switch, printer, ap, firewall, iot, camera, nas, unknown |
| OS | string | Operating system (if known) |
| Status | enum | online, offline, degraded, unknown |
| DiscoveryMethod | enum | agent, icmp, arp, snmp, mdns, upnp, mqtt, tailscale, manual |
| AgentID | UUID? | Linked Scout agent (if any) |
| ParentDeviceID | UUID? | Upstream device for topology (switch port, router) |
| LastSeen | timestamp | Last successful contact |
| FirstSeen | timestamp | Initial discovery |
| Notes | string | User-provided notes |
| Tags | []string | User-defined tags |
| CustomFields | map | User-defined key-value pairs |

### Agent (Scout)

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| TenantID | UUID? | Tenant |
| DeviceID | UUID | Linked device |
| Version | string | Agent software version |
| Status | enum | connected, disconnected, stale |
| LastCheckIn | timestamp | Last successful check-in |
| EnrolledAt | timestamp | Enrollment timestamp |
| CertSerialNo | string | mTLS certificate serial number |
| CertExpiresAt | timestamp | Certificate expiration |
| Platform | string | OS/architecture |

### Credential (Vault)

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| TenantID | UUID? | Tenant |
| Name | string | Display name |
| Type | enum | ssh_password, ssh_key, rdp, http_basic, snmp_community, snmp_v3, api_key |
| Data | encrypted blob | Encrypted credential data (AES-256-GCM envelope encryption) |
| DeviceIDs | []UUID | Associated devices |
| CreatedBy | UUID | User who created |
| CreatedAt | timestamp | Creation timestamp |
| UpdatedAt | timestamp | Last modification |
| LastAccessedAt | timestamp | Last time credential was used |

### Topology Link

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| SourceDeviceID | UUID | Upstream device |
| TargetDeviceID | UUID | Downstream device |
| SourcePort | string | Port/interface name on source |
| TargetPort | string | Port/interface name on target |
| LinkType | enum | lldp, cdp, arp, manual |
| Speed | int | Link speed in Mbps |
| DiscoveredAt | timestamp | When this link was detected |
| LastConfirmed | timestamp | Last time this link was confirmed active |

### Tenant (Phase 2)

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| Name | string | Tenant/client name |
| Slug | string | URL-safe identifier |
| Status | enum | active, suspended, archived |
| MaxDevices | int | Device limit for this tenant |
| CreatedAt | timestamp | Tenant creation |
