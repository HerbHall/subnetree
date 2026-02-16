import { useState, useEffect, useCallback, useRef } from 'react'
import { RefreshCw, Radar, CheckCircle2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useScanStore } from '@/stores/scan'

/** Format elapsed seconds as "M:SS". */
function formatElapsed(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

interface ScanButtonProps {
  onScan: () => void
  isPending: boolean
}

export function ScanButton({ onScan, isPending }: ScanButtonProps) {
  const activeScan = useScanStore((s) => s.activeScan)
  const [elapsed, setElapsed] = useState(0)

  const isScanning =
    !!activeScan &&
    (activeScan.status === 'starting' ||
      activeScan.status === 'scanning' ||
      activeScan.status === 'processing')

  const isCompleted = activeScan?.status === 'completed'

  // Store startedAt in a ref via an effect so the interval callback
  // always has the correct value without restarting the interval.
  const startedAtRef = useRef(0)
  const startedAtMs = isScanning && activeScan ? activeScan.startedAt.getTime() : 0

  useEffect(() => {
    startedAtRef.current = startedAtMs
  }, [startedAtMs])

  // Elapsed timer: computes elapsed inside the interval callback (not render).
  useEffect(() => {
    if (!isScanning) return

    const interval = setInterval(() => {
      const now = Date.now()
      setElapsed(Math.floor((now - startedAtRef.current) / 1000))
    }, 1000)

    return () => {
      clearInterval(interval)
    }
  }, [isScanning])

  const handleClick = useCallback(() => {
    onScan()
  }, [onScan])

  const disabled = isPending || isScanning

  // Completion state: the store keeps activeScan as 'completed' for 3 seconds
  // before clearing it, so we render the summary during that window.
  if (isCompleted && activeScan) {
    return (
      <Button disabled className="gap-2">
        <CheckCircle2 className="h-4 w-4 text-green-400" />
        Scan Complete &ndash; {activeScan.devicesFound} device
        {activeScan.devicesFound !== 1 ? 's' : ''}
      </Button>
    )
  }

  // Scanning state: device count + elapsed timer
  if (isScanning && activeScan) {
    const deviceLabel = `${activeScan.devicesFound} device${activeScan.devicesFound !== 1 ? 's' : ''}`
    return (
      <Button disabled className="gap-2">
        <RefreshCw className="h-4 w-4 animate-spin" />
        Scanning... {deviceLabel} ({formatElapsed(elapsed)})
      </Button>
    )
  }

  // Idle state
  return (
    <Button onClick={handleClick} disabled={disabled} className="gap-2">
      {isPending ? (
        <RefreshCw className="h-4 w-4 animate-spin" />
      ) : (
        <Radar className="h-4 w-4" />
      )}
      {isPending ? 'Starting...' : 'Scan Network'}
    </Button>
  )
}
