import { useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useWebSocket } from './use-websocket'
import { useScanStore } from '@/stores/scan'
import type {
  ScanWSMessage,
  ScanStartedData,
  ScanProgressData,
  ScanDeviceFoundData,
  ScanCompletedData,
  ScanErrorData,
} from '@/api/types'

export function useScanProgress() {
  const queryClient = useQueryClient()
  const { activeScan, startScan, setScanProgress, addDevice, completeScan, setError, clearScan } =
    useScanStore()

  const handleMessage = useCallback(
    (raw: unknown) => {
      const message = raw as ScanWSMessage
      if (!message.type || !message.scan_id) return

      switch (message.type) {
        case 'scan.started': {
          const data = message.data as ScanStartedData
          startScan(message.scan_id, data.target_cidr)
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
          break
        }

        case 'scan.completed': {
          const data = message.data as ScanCompletedData
          completeScan(message.scan_id, data.total, data.online)
          toast.success(`Scan complete: ${data.total} devices found, ${data.online} online`)

          // Invalidate cached data so lists refresh.
          queryClient.invalidateQueries({ queryKey: ['devices'] })
          queryClient.invalidateQueries({ queryKey: ['topology'] })
          queryClient.invalidateQueries({ queryKey: ['scans'] })

          // Clear the progress panel after a delay.
          setTimeout(() => clearScan(), 3000)
          break
        }

        case 'scan.error': {
          const data = message.data as ScanErrorData
          setError(message.scan_id, data.error)
          toast.error(`Scan failed: ${data.error}`)

          setTimeout(() => clearScan(), 5000)
          break
        }
      }
    },
    [startScan, setScanProgress, addDevice, completeScan, setError, clearScan, queryClient],
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
