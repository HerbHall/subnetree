import { api } from './client'
import type { TopologyGraph, Scan } from './types'

/**
 * Fetch the network topology (devices + connections).
 */
export async function getTopology(): Promise<TopologyGraph> {
  return api.get<TopologyGraph>('/recon/topology')
}

/**
 * Trigger a new network scan.
 * @param targetCidr Optional CIDR range to scan (defaults to local subnet)
 */
export async function triggerScan(targetCidr?: string): Promise<Scan> {
  return api.post<Scan>('/recon/scan', targetCidr ? { target_cidr: targetCidr } : undefined)
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
