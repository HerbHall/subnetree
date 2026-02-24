import { api } from './client'

export interface TailscaleStatus {
  enabled: boolean
  last_sync_time?: string
  last_sync_result?: TailscaleSyncResult
  error?: string
}

export interface TailscaleSyncResult {
  devices_found: number
  created: number
  updated: number
  unchanged: number
}

export async function getTailscaleStatus(): Promise<TailscaleStatus> {
  return api.get<TailscaleStatus>('/tailscale/status')
}

export async function triggerTailscaleSync(): Promise<TailscaleSyncResult> {
  return api.post<TailscaleSyncResult>('/tailscale/sync')
}
