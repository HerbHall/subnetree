import { api } from './client'
import type { Service, UtilizationSummary, FleetSummary, DesiredState } from './types'

/**
 * List all tracked services, optionally filtered.
 */
export async function listServices(params?: {
  device_id?: string
  service_type?: string
  status?: string
}): Promise<Service[]> {
  const query = new URLSearchParams()
  if (params?.device_id) query.set('device_id', params.device_id)
  if (params?.service_type) query.set('service_type', params.service_type)
  if (params?.status) query.set('status', params.status)
  const qs = query.toString()
  return api.get<Service[]>(`/svcmap/services${qs ? `?${qs}` : ''}`)
}

/**
 * Get a single service by ID.
 */
export async function getService(id: string): Promise<Service> {
  return api.get<Service>(`/svcmap/services/${id}`)
}

/**
 * Update the desired state for a service.
 */
export async function updateDesiredState(
  id: string,
  desired_state: DesiredState
): Promise<Service> {
  return api.patch<Service>(`/svcmap/services/${id}`, { desired_state })
}

/**
 * Get all services for a specific device.
 */
export async function getDeviceServices(deviceId: string): Promise<Service[]> {
  return api.get<Service[]>(`/svcmap/devices/${deviceId}/services`)
}

/**
 * Get utilization summary for a specific device.
 */
export async function getDeviceUtilization(
  deviceId: string
): Promise<UtilizationSummary> {
  return api.get<UtilizationSummary>(`/svcmap/devices/${deviceId}/utilization`)
}

/**
 * Get fleet-wide utilization summary.
 */
export async function getFleetSummary(): Promise<FleetSummary> {
  return api.get<FleetSummary>(`/svcmap/utilization/fleet`)
}
