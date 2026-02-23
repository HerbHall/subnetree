import { api } from './client'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ProxmoxResource {
  device_id: string
  device_name: string
  device_type: 'virtual_machine' | 'container'
  status: string
  cpu_percent: number
  mem_used_mb: number
  mem_total_mb: number
  disk_used_gb: number
  disk_total_gb: number
  uptime_sec: number
  netin_bytes: number
  netout_bytes: number
  collected_at: string
}

export interface ProxmoxSyncResult {
  nodes_scanned: number
  vms_found: number
  lxcs_found: number
  created: number
  updated: number
}

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

/**
 * Fetch all Proxmox VMs/containers, optionally filtered by parent host.
 */
export async function getProxmoxVMs(parentId?: string): Promise<ProxmoxResource[]> {
  const params = parentId ? `?parent_id=${parentId}` : ''
  return api.get<ProxmoxResource[]>(`/recon/proxmox/vms${params}`)
}

/**
 * Fetch the resource snapshot for a specific VM/container.
 */
export async function getProxmoxResources(deviceId: string): Promise<ProxmoxResource> {
  return api.get<ProxmoxResource>(`/recon/proxmox/vms/${deviceId}/resources`)
}
