import { create } from 'zustand'
import type { Device } from '@/api/types'

export interface ScanProgress {
  scanId: string
  targetCidr: string
  status: 'starting' | 'scanning' | 'processing' | 'completed' | 'error'
  devicesFound: number
  subnetSize: number
  hostsAlive: number
  newDevices: Device[]
  error?: string
  startedAt: Date
  completedAt?: Date
}

interface ScanStore {
  activeScan: ScanProgress | null

  startScan: (scanId: string, targetCidr: string) => void
  setScanProgress: (scanId: string, hostsAlive: number, subnetSize: number) => void
  addDevice: (scanId: string, device: Device) => void
  completeScan: (scanId: string, total: number, online: number) => void
  setError: (scanId: string, error: string) => void
  clearScan: () => void
}

export const useScanStore = create<ScanStore>((set, get) => ({
  activeScan: null,

  startScan: (scanId, targetCidr) =>
    set({
      activeScan: {
        scanId,
        targetCidr,
        status: 'scanning',
        devicesFound: 0,
        subnetSize: calculateSubnetSize(targetCidr),
        hostsAlive: 0,
        newDevices: [],
        startedAt: new Date(),
      },
    }),

  setScanProgress: (scanId, hostsAlive, subnetSize) => {
    const { activeScan } = get()
    if (activeScan?.scanId !== scanId) return
    set({
      activeScan: {
        ...activeScan,
        status: 'processing',
        hostsAlive,
        subnetSize,
      },
    })
  },

  addDevice: (scanId, device) => {
    const { activeScan } = get()
    if (activeScan?.scanId !== scanId) return
    set({
      activeScan: {
        ...activeScan,
        devicesFound: activeScan.devicesFound + 1,
        newDevices: [...activeScan.newDevices, device],
      },
    })
  },

  completeScan: (scanId, _total, _online) => {
    const { activeScan } = get()
    if (activeScan?.scanId !== scanId) return
    set({
      activeScan: {
        ...activeScan,
        status: 'completed',
        completedAt: new Date(),
      },
    })
  },

  setError: (scanId, error) => {
    const { activeScan } = get()
    if (activeScan?.scanId !== scanId) return
    set({
      activeScan: {
        ...activeScan,
        status: 'error',
        error,
      },
    })
  },

  clearScan: () => set({ activeScan: null }),
}))

/** Calculate the number of usable hosts from a CIDR notation. */
function calculateSubnetSize(cidr: string): number {
  const parts = cidr.split('/')
  if (parts.length !== 2) return 254
  const bits = parseInt(parts[1], 10)
  if (isNaN(bits) || bits < 0 || bits > 32) return 254
  const hostBits = 32 - bits
  return Math.max(1, Math.pow(2, hostBits) - 2)
}
