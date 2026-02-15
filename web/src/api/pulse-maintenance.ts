import { api } from './client'

// MaintWindow represents a scheduled maintenance window.
export interface MaintWindow {
  id: string
  name: string
  description: string
  start_time: string
  end_time: string
  recurrence: 'once' | 'daily' | 'weekly' | 'monthly'
  device_ids: string[]
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateMaintWindowRequest {
  name: string
  description: string
  start_time: string
  end_time: string
  recurrence: string
  device_ids: string[]
}

export interface UpdateMaintWindowRequest {
  name?: string
  description?: string
  start_time?: string
  end_time?: string
  recurrence?: string
  device_ids?: string[]
  enabled?: boolean
}

export async function listMaintWindows(): Promise<MaintWindow[]> {
  return api.get<MaintWindow[]>('/pulse/maintenance-windows')
}

export async function createMaintWindow(req: CreateMaintWindowRequest): Promise<MaintWindow> {
  return api.post<MaintWindow>('/pulse/maintenance-windows', req)
}

export async function getMaintWindow(id: string): Promise<MaintWindow> {
  return api.get<MaintWindow>(`/pulse/maintenance-windows/${id}`)
}

export async function updateMaintWindow(id: string, req: UpdateMaintWindowRequest): Promise<MaintWindow> {
  return api.put<MaintWindow>(`/pulse/maintenance-windows/${id}`, req)
}

export async function deleteMaintWindow(id: string): Promise<void> {
  return api.delete(`/pulse/maintenance-windows/${id}`)
}
