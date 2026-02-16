import { useCallback, useRef, useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useWebSocket } from './use-websocket'
import { useScanStore } from '@/stores/scan'
import type { DeviceListResponse } from '@/api/devices'
import type {
  ScanWSMessage,
  ScanStartedData,
  ScanProgressData,
  ScanDeviceFoundData,
  ScanCompletedData,
  ScanErrorData,
} from '@/api/types'

/** Debounce interval for device list invalidation during streaming discovery (ms). */
const DEVICE_INVALIDATE_DEBOUNCE_MS = 800

export function useScanProgress() {
  const queryClient = useQueryClient()
  const { activeScan, startScan, loadKnownDevices, setScanProgress, addDevice, completeScan, setError, clearScan } =
    useScanStore()

  // Debounce timer for device query invalidation during scan streaming.
  // Avoids firing hundreds of re-fetches for large subnets while still
  // keeping the device list visibly up to date every ~800ms.
  const invalidateTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Clean up timer on unmount.
  useEffect(() => {
    return () => {
      if (invalidateTimerRef.current) {
        clearTimeout(invalidateTimerRef.current)
      }
    }
  }, [])

  const debouncedInvalidateDevices = useCallback(() => {
    if (invalidateTimerRef.current) return // already scheduled
    invalidateTimerRef.current = setTimeout(() => {
      invalidateTimerRef.current = null
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      queryClient.invalidateQueries({ queryKey: ['inventorySummary'] })
    }, DEVICE_INVALIDATE_DEBOUNCE_MS)
  }, [queryClient])

  const handleMessage = useCallback(
    (raw: unknown) => {
      const message = raw as ScanWSMessage
      if (!message.type || !message.scan_id) return

      switch (message.type) {
        case 'scan.started': {
          const data = message.data as ScanStartedData
          startScan(message.scan_id, data.target_cidr)

          // Load known device IPs from the React Query cache so we can show
          // them immediately with a "checking" state while the scan runs.
          const cachedQueries = queryClient.getQueriesData<DeviceListResponse>({
            queryKey: ['devices'],
          })
          const knownIps: string[] = []
          for (const [, cached] of cachedQueries) {
            if (cached?.devices) {
              for (const device of cached.devices) {
                for (const ip of device.ip_addresses ?? []) {
                  knownIps.push(ip)
                }
              }
            }
          }
          if (knownIps.length > 0) {
            loadKnownDevices(knownIps)
          }

          toast.info(`Scanning ${data.target_cidr}...`)
          break
        }

        case 'scan.progress': {
          const data = message.data as ScanProgressData
          setScanProgress(message.scan_id, data.hosts_alive, data.subnet_size)
          break
        }

        case 'scan.device_found': {
          const data = message.data as ScanDeviceFoundData
          addDevice(message.scan_id, data.device)
          // Debounce device list refresh so the table updates during the scan
          // without hammering the API on every single host response.
          debouncedInvalidateDevices()
          break
        }

        case 'scan.completed': {
          const data = message.data as ScanCompletedData
          completeScan(message.scan_id, data.total, data.online)
          toast.success(`Scan complete: ${data.total} devices found, ${data.online} online`)

          // Cancel any pending debounced invalidation; we do a final one now.
          if (invalidateTimerRef.current) {
            clearTimeout(invalidateTimerRef.current)
            invalidateTimerRef.current = null
          }

          // Invalidate cached data so lists refresh with final state.
          queryClient.invalidateQueries({ queryKey: ['devices'] })
          queryClient.invalidateQueries({ queryKey: ['topology'] })
          queryClient.invalidateQueries({ queryKey: ['scans'] })
          queryClient.invalidateQueries({ queryKey: ['inventorySummary'] })

          // Clear the progress panel after a delay.
          setTimeout(() => clearScan(), 3000)
          break
        }

        case 'scan.error': {
          const data = message.data as ScanErrorData
          setError(message.scan_id, data.error)
          toast.error(`Scan failed: ${data.error}`)

          // Cancel any pending debounced invalidation.
          if (invalidateTimerRef.current) {
            clearTimeout(invalidateTimerRef.current)
            invalidateTimerRef.current = null
          }

          setTimeout(() => clearScan(), 5000)
          break
        }
      }
    },
    [startScan, loadKnownDevices, setScanProgress, addDevice, completeScan, setError, clearScan, queryClient, debouncedInvalidateDevices],
  )

  const { isConnected, reconnectCount } = useWebSocket({
    url: '/api/v1/ws/scan',
    onMessage: handleMessage,
  })

  // Compute progress percentage.
  const progress = activeScan
    ? Math.min(100, Math.round((activeScan.devicesFound / activeScan.subnetSize) * 100))
    : 0

  return {
    activeScan,
    progress,
    isConnected,
    reconnectCount,
  }
}
