import { api } from './client'
import type { TopologyGraph, Scan, Device } from './types'

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
 * Update device notes and tags.
 */
export async function updateDevice(
  id: string,
  data: { notes?: string; tags?: string[] }
): Promise<Device> {
  return api.put<Device>(`/recon/devices/${id}`, data)
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
