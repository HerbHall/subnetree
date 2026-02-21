import { api } from './client'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface DeviceHardware {
  device_id: string
  hostname?: string
  fqdn?: string
  os_name?: string
  os_version?: string
  os_arch?: string
  kernel?: string
  cpu_model?: string
  cpu_cores?: number
  cpu_threads?: number
  cpu_arch?: string
  ram_total_mb?: number
  ram_type?: string
  ram_slots_used?: number
  ram_slots_total?: number
  platform_type?: string
  hypervisor?: string
  vm_host_id?: string
  system_manufacturer?: string
  system_model?: string
  serial_number?: string
  bios_version?: string
  collection_source?: string
  collected_at?: string
  updated_at?: string
}

export interface DeviceStorage {
  id: string
  device_id: string
  name?: string
  disk_type?: string
  interface?: string
  capacity_gb?: number
  model?: string
  role?: string
  collection_source?: string
  collected_at?: string
}

export interface DeviceGPU {
  id: string
  device_id: string
  model?: string
  vendor?: string
  vram_mb?: number
  driver_version?: string
  collection_source?: string
  collected_at?: string
}

export interface DeviceService {
  id: string
  device_id: string
  name?: string
  service_type?: string
  port?: number
  url?: string
  version?: string
  status?: string
  collection_source?: string
  collected_at?: string
}

export interface DeviceHardwareResponse {
  hardware: DeviceHardware | null
  storage: DeviceStorage[]
  gpus: DeviceGPU[]
  services: DeviceService[]
}

export interface HardwareSummary {
  total_with_hardware: number
  total_ram_mb: number
  total_storage_gb: number
  total_gpus: number
  by_os: Record<string, number>
  by_cpu_model: Record<string, number>
  by_platform_type: Record<string, number>
  by_gpu_vendor: Record<string, number>
}

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

/**
 * Fetch the full hardware profile for a device (hardware, storage, GPUs, services).
 */
export async function getDeviceHardware(id: string): Promise<DeviceHardwareResponse> {
  return api.get<DeviceHardwareResponse>(`/recon/devices/${id}/hardware`)
}

/**
 * Update (or create) a device's hardware profile manually.
 */
export async function updateDeviceHardware(
  id: string,
  data: Partial<DeviceHardware>,
): Promise<DeviceHardware> {
  return api.put<DeviceHardware>(`/recon/devices/${id}/hardware`, data)
}

/**
 * Fetch fleet-wide aggregate hardware statistics.
 */
export async function getHardwareSummary(): Promise<HardwareSummary> {
  return api.get<HardwareSummary>('/recon/inventory/hardware-summary')
}
