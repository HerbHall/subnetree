import { api } from './client'
import type { TopologyGraph, Scan, Device, DeviceType } from './types'

/**
 * Fetch the network topology (devices + connections).
 */
export async function getTopology(): Promise<TopologyGraph> {
  return api.get<TopologyGraph>('/recon/topology')
}

/**
 * Get a single device by ID.
 */
export async function getDevice(id: string): Promise<Device> {
  return api.get<Device>(`/recon/devices/${id}`)
}

/**
 * Update device fields (notes, tags, custom_fields, device_type).
 */
export async function updateDevice(
  id: string,
  data: { notes?: string; tags?: string[]; custom_fields?: Record<string, string>; device_type?: DeviceType }
): Promise<Device> {
  return api.put<Device>(`/recon/devices/${id}`, data)
}

/**
 * Request body for creating a new device.
 */
export interface CreateDeviceRequest {
  hostname: string
  ip_addresses: string[]
  mac_address?: string
  device_type?: DeviceType
  notes?: string
  tags?: string[]
}

/**
 * Paginated device list response.
 */
export interface DeviceListResponse {
  devices: Device[]
  total: number
  limit: number
  offset: number
}

/**
 * Parameters for listing devices.
 */
export interface ListDevicesParams {
  limit?: number
  offset?: number
  status?: string
  type?: string
}

/**
 * List devices with optional filters and pagination.
 */
export async function listDevices(params: ListDevicesParams = {}): Promise<DeviceListResponse> {
  const searchParams = new URLSearchParams()
  if (params.limit !== undefined) searchParams.set('limit', String(params.limit))
  if (params.offset !== undefined) searchParams.set('offset', String(params.offset))
  if (params.status && params.status !== 'all') searchParams.set('status', params.status)
  if (params.type && params.type !== 'all') searchParams.set('type', params.type)
  const query = searchParams.toString()
  return api.get<DeviceListResponse>(`/recon/devices${query ? `?${query}` : ''}`)
}

/**
 * Create a new device.
 */
export async function createDevice(data: CreateDeviceRequest): Promise<Device> {
  return api.post<Device>('/recon/devices', data)
}

/**
 * Delete a device by ID.
 */
export async function deleteDevice(id: string): Promise<void> {
  return api.delete<void>(`/recon/devices/${id}`)
}

/**
 * Get status history for a device.
 */
export interface DeviceStatusEvent {
  id: string
  device_id: string
  status: 'online' | 'offline' | 'degraded' | 'unknown'
  timestamp: string
}

export async function getDeviceStatusHistory(
  id: string,
  limit = 50
): Promise<DeviceStatusEvent[]> {
  return api.get<DeviceStatusEvent[]>(`/recon/devices/${id}/history?limit=${limit}`)
}

/**
 * Get scan history for a device (scans that discovered/updated this device).
 */
export async function getDeviceScanHistory(id: string): Promise<Scan[]> {
  return api.get<Scan[]>(`/recon/devices/${id}/scans`)
}

/**
 * Trigger a new network scan.
 * @param subnet CIDR range to scan (defaults to 192.168.1.0/24)
 */
export async function triggerScan(subnet = '192.168.1.0/24'): Promise<Scan> {
  return api.post<Scan>('/recon/scan', { subnet })
}

/**
 * List recent scans.
 * @param limit Number of scans to return (default 20)
 * @param offset Pagination offset
 */
export async function listScans(limit = 20, offset = 0): Promise<Scan[]> {
  return api.get<Scan[]>(`/recon/scans?limit=${limit}&offset=${offset}`)
}

/**
 * Get a specific scan by ID.
 */
export async function getScan(id: string): Promise<Scan> {
  return api.get<Scan>(`/recon/scans/${id}`)
}
