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
| Location | string? | Physical location (Phase 2, #163) |
| Category | enum? | keep, sell, repurpose, undecided (Phase 2, #163) |
| PrimaryRole | string? | Intended purpose (Phase 2, #163) |
| Justification | string? | Why this device is needed (Phase 2, #163) |
| DevicePolicy | enum? | full-workstation, thin-client, server, appliance (Phase 2, #163) |
| Owner | string? | Responsible person (Phase 2, #163) |
| AcquiredDate | timestamp? | When acquired (Phase 2, #163) |

### Service (Phase 2, #165)

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| Name | string | Human-readable service name |
| ServiceType | enum | docker-container, systemd-service, windows-service, application |
| DeviceID | UUID | Host device |
| ApplicationID | UUID? | Link to Docs module application (if matched) |
| Status | enum | running, stopped, failed, unknown |
| Ports | []int | Listening ports |
| ResourceUsage | object | Latest CPU%, memory, disk I/O |
| DesiredState | enum? | should-run, should-stop, monitoring-only (user-annotated) |
| FirstSeen | timestamp | |
| LastSeen | timestamp | |

### DeviceHardware (Phase 2, #437)

Core hardware profile for a device. Auto-collected values never overwrite user-entered (manual) data.

| Field | Type | Description |
|-------|------|-------------|
| DeviceID | UUID | Parent device |
| Hostname | string? | Hostname from hardware collection |
| FQDN | string? | Fully qualified domain name |
| OSName | string? | Operating system name (e.g., "Ubuntu 24.04") |
| OSVersion | string? | OS version string |
| OSArch | string? | OS architecture (e.g., "amd64") |
| Kernel | string? | Kernel version |
| CPUModel | string? | Processor model |
| CPUCores | int? | Physical cores |
| CPUThreads | int? | Logical threads |
| CPUArch | string? | CPU architecture (e.g., "x86_64") |
| RAMTotalMB | int? | Total RAM in megabytes |
| RAMType | string? | Memory type (e.g., "DDR4") |
| RAMSlotsUsed | int? | Used memory slots |
| RAMSlotsTotal | int? | Total memory slots |
| PlatformType | string? | Platform: baremetal, vm, lxc, docker |
| Hypervisor | string? | Hypervisor name (e.g., "proxmox") |
| VMHostID | string? | Parent VM host device ID |
| SystemManufacturer | string? | Hardware manufacturer |
| SystemModel | string? | Hardware model |
| SerialNumber | string? | Serial number |
| BIOSVersion | string? | Firmware version |
| CollectionSource | string | Source: scout-wmi, scout-linux, proxmox-api, unraid-api, homeassistant-api, manual |
| CollectedAt | timestamp? | Last collection timestamp |
| UpdatedAt | timestamp? | Last update timestamp |

### DeviceStorage (Phase 2, #437)

Storage devices attached to a device. Multiple records per device.

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| DeviceID | UUID | Parent device |
| Name | string? | Display name (e.g., "Samsung 990 Pro 4TB") |
| DiskType | string? | Type: nvme, ssd, hdd, emmc |
| Interface | string? | Interface: pcie4, sata, usb |
| CapacityGB | int? | Capacity in gigabytes |
| Model | string? | Hardware model string |
| Role | string? | Role: boot, data, cache, swap |
| CollectionSource | string | Source identifier |
| CollectedAt | timestamp? | Collection timestamp |

### DeviceGPU (Phase 2, #437)

GPU devices in a device. Multiple records per device.

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| DeviceID | UUID | Parent device |
| Model | string? | GPU model (e.g., "NVIDIA RTX 3090 Ti") |
| Vendor | string? | Vendor: nvidia, amd, intel |
| VRAMMB | int? | Video RAM in megabytes |
| DriverVersion | string? | Driver version string |
| CollectionSource | string | Source identifier |
| CollectedAt | timestamp? | Collection timestamp |

### DeviceService (Phase 2, #437)

Running services on a device. Multiple records per device.

| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique identifier |
| DeviceID | UUID | Parent device |
| Name | string? | Service name (e.g., "plex") |
| ServiceType | string? | Type: docker, systemd, windows-service, application |
| Port | int? | Listening port |
| URL | string? | Service URL |
| Version | string? | Service version |
| Status | string? | Status: running, stopped, failed, unknown |
| CollectionSource | string | Source identifier |
| CollectedAt | timestamp? | Collection timestamp |

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
