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
// System Types
// ============================================================================

/** Version information map returned by the health endpoint. */
export interface VersionInfo {
  version: string
  git_commit: string
  build_date: string
  go_version: string
  os: string
  arch: string
}

/** Health check response from GET /api/v1/health. */
export interface HealthResponse {
  status: string
  service: string
  version: VersionInfo
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
  | 'mobile'
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
  location?: string
  category?: string
  primary_role?: string
  owner?: string
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

// ============================================================================
// WebSocket Message Types
// ============================================================================

/** WebSocket message types for scan progress. */
export type ScanWSMessageType =
  | 'scan.started'
  | 'scan.progress'
  | 'scan.device_found'
  | 'scan.completed'
  | 'scan.error'

/** WebSocket message envelope. */
export interface ScanWSMessage {
  type: ScanWSMessageType
  scan_id: string
  timestamp: string
  data: ScanStartedData | ScanProgressData | ScanDeviceFoundData | ScanCompletedData | ScanErrorData
}

export interface ScanStartedData {
  target_cidr: string
  status: string
}

export interface ScanProgressData {
  hosts_alive: number
  subnet_size: number
}

export interface ScanDeviceFoundData {
  device: Device
}

export interface ScanCompletedData {
  total: number
  online: number
  ended_at: string
}

export interface ScanErrorData {
  error: string
}

// ============================================================================
// Agent Types
// ============================================================================

/** Agent connection status. */
export type AgentStatus = 'pending' | 'connected' | 'disconnected'

/** Registered Scout agent as returned by the Dispatch module. */
export interface AgentInfo {
  id: string
  hostname: string
  platform: string
  agent_version: string
  proto_version: number
  device_id: string
  status: AgentStatus
  last_check_in?: string
  enrolled_at: string
  cert_serial: string
  cert_expires_at?: string
  config_json: string
}

/** Request body for creating an enrollment token. */
export interface CreateEnrollmentTokenRequest {
  description: string
  max_uses?: number
  expires_in?: string
}

/** Response from creating an enrollment token (raw token only returned once). */
export interface EnrollmentTokenResponse {
  id: string
  token: string
  expires_at?: string
  max_uses: number
}

/** Hardware profile reported by Scout agent. */
export interface HardwareProfile {
  cpu_model: string
  cpu_cores: number
  cpu_threads: number
  ram_bytes: number
  disks: DiskInfo[]
  gpus: GPUInfo[]
  nics: NICInfo[]
  bios_version: string
  system_manufacturer: string
  system_model: string
  serial_number: string
}

export interface DiskInfo {
  name: string
  size_bytes: number
  disk_type: string
  model: string
  serial: string
}

export interface GPUInfo {
  model: string
  vram_bytes: number
  driver_version: string
}

export interface NICInfo {
  name: string
  speed_mbps: number
  mac_address: string
  nic_type: string
}

/** Software inventory reported by Scout agent. */
export interface SoftwareInventory {
  os_name: string
  os_version: string
  os_build: string
  packages: InstalledPackage[]
  docker_containers: DockerContainer[]
}

export interface InstalledPackage {
  name: string
  version: string
  publisher: string
  install_date: string
}

export interface DockerContainer {
  container_id: string
  name: string
  image: string
  status: string
}

/** Service information reported by Scout agent. */
export interface ServiceInfo {
  name: string
  display_name: string
  status: string
  start_type: string
  cpu_percent: number
  memory_bytes: number
  ports: number[]
}

// ============================================================================
// Service Mapping Types
// ============================================================================

/** Service type classification. */
export type ServiceType = 'docker-container' | 'systemd-service' | 'windows-service' | 'application'

/** Service operational status. */
export type ServiceStatus = 'running' | 'stopped' | 'failed' | 'unknown'

/** Desired operational state for a service. */
export type DesiredState = 'should-run' | 'should-stop' | 'monitoring-only'

/** Tracked service on a device. */
export interface Service {
  id: string
  name: string
  display_name: string
  service_type: ServiceType
  device_id: string
  application_id?: string
  status: ServiceStatus
  desired_state: DesiredState
  ports?: string[]
  cpu_percent: number
  memory_bytes: number
  first_seen: string
  last_seen: string
}

/** Resource utilization summary for a single device. */
export interface UtilizationSummary {
  device_id: string
  hostname: string
  cpu_percent: number
  memory_percent: number
  disk_percent: number
  service_count: number
  grade: string
  headroom: number
}

/** Fleet-wide utilization aggregation. */
export interface FleetSummary {
  total_devices: number
  total_services: number
  avg_cpu: number
  avg_memory: number
  by_grade: Record<string, number>
  underutilized?: string[]
  overloaded?: string[]
}

// ============================================================================
// Monitoring Types (Pulse)
// ============================================================================

/** Check type classification. */
export type CheckType = 'icmp' | 'tcp' | 'http'

/** Monitoring check for a device. */
export interface Check {
  id: string
  device_id: string
  check_type: CheckType
  target: string
  interval_seconds: number
  enabled: boolean
  created_at: string
  updated_at: string
}

/** Result from a single health check execution. */
export interface CheckResult {
  id: number
  check_id: string
  device_id: string
  success: boolean
  latency_ms: number
  packet_loss: number
  error_message?: string
  checked_at: string
}

/** Monitoring alert triggered by consecutive check failures. */
export interface Alert {
  id: string
  check_id: string
  device_id: string
  severity: string
  message: string
  triggered_at: string
  resolved_at?: string
  acknowledged_at?: string
  consecutive_failures: number
}

/** Request body for creating a new check. */
export interface CreateCheckRequest {
  device_id: string
  check_type: CheckType
  target: string
  interval_seconds?: number
}

/** Request body for updating a check. */
export interface UpdateCheckRequest {
  target?: string
  check_type?: CheckType
  interval_seconds?: number
  enabled?: boolean
}

/** Composite monitoring status for a device. */
export interface MonitoringStatus {
  device_id: string
  healthy: boolean
  message: string
  checked_at?: string
}

/** Notification delivery channel configuration. */
export interface NotificationChannel {
  id: string
  name: string
  type: string // "webhook" | "email"
  config: string // JSON blob
  enabled: boolean
  created_at: string
  updated_at: string
}

/** Request body for creating a notification channel. */
export interface CreateNotificationRequest {
  name: string
  type: string
  config: string
}

/** Request body for updating a notification channel. */
export interface UpdateNotificationRequest {
  name?: string
  config?: string
  enabled?: boolean
}

// ============================================================================
// SNMP Types
// ============================================================================

/** SNMP system information from device query. */
export interface SNMPSystemInfo {
  description: string
  object_id: string
  up_time_ms: number
  contact: string
  name: string
  location: string
}

/** SNMP network interface from device query. */
export interface SNMPInterface {
  index: number
  description: string
  type: number
  mtu: number
  speed: number
  phys_address: string
  admin_status: number
  oper_status: number
}

/** Request body for SNMP discover endpoint. */
export interface SNMPDiscoverRequest {
  target: string
  credential_id: string
}
