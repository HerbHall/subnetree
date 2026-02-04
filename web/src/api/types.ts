/** JWT access + refresh token pair returned by login and refresh endpoints. */
export interface TokenPair {
  access_token: string
  refresh_token: string
  expires_in: number
}

/** User account as returned by the server. */
export interface User {
  id: string
  username: string
  email: string
  role: 'admin' | 'operator' | 'viewer'
  auth_provider: string
  oidc_subject?: string
  created_at: string
  last_login?: string
  disabled: boolean
  locked_until?: string
}

/** RFC 7807 Problem Detail error response. */
export interface ProblemDetail {
  type: string
  title: string
  status: number
  detail?: string
  instance?: string
}

// ============================================================================
// Device Types
// ============================================================================

/** Device status as returned by the server. */
export type DeviceStatus = 'online' | 'offline' | 'degraded' | 'unknown'

/** Device type classification. */
export type DeviceType =
  | 'server'
  | 'desktop'
  | 'laptop'
  | 'router'
  | 'switch'
  | 'access_point'
  | 'firewall'
  | 'printer'
  | 'nas'
  | 'iot'
  | 'phone'
  | 'tablet'
  | 'camera'
  | 'unknown'

/** How the device was discovered. */
export type DiscoveryMethod = 'agent' | 'icmp' | 'arp' | 'snmp' | 'mdns' | 'upnp'

/** Device as returned by topology/device endpoints. */
export interface Device {
  id: string
  hostname: string
  ip_addresses: string[]
  mac_address: string
  manufacturer: string
  device_type: DeviceType
  os: string
  status: DeviceStatus
  discovery_method: DiscoveryMethod
  agent_id?: string
  last_seen: string
  first_seen: string
  notes?: string
  tags?: string[]
  custom_fields?: Record<string, string>
}

/** Topology node (simplified device for graph display). */
export interface TopologyNode {
  id: string
  label: string
  device_type: DeviceType
  status: DeviceStatus
  ip_addresses: string[]
  mac_address: string
  manufacturer: string
}

/** Topology edge (connection between devices). */
export interface TopologyEdge {
  id: string
  source: string
  target: string
  link_type: string
  speed?: number
}

/** Network topology graph response. */
export interface TopologyGraph {
  nodes: TopologyNode[]
  edges: TopologyEdge[]
}

/** Scan status. */
export type ScanStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

/** Network scan record. */
export interface Scan {
  id: string
  status: ScanStatus
  target_cidr: string
  started_at: string
  completed_at?: string
  devices_found: number
  error?: string
}
