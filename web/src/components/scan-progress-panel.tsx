import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Loader2, CheckCircle2, XCircle, Wifi } from 'lucide-react'
import type { ScanProgress } from '@/stores/scan'
import { cn } from '@/lib/utils'

const statusConfig = {
  starting: {
    icon: Loader2,
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    border: 'border-blue-500/50',
    spin: true,
  },
  scanning: {
    icon: Loader2,
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    border: 'border-blue-500/50',
    spin: true,
  },
  processing: {
    icon: Loader2,
    color: 'text-blue-500',
    bg: 'bg-blue-500/10',
    border: 'border-blue-500/50',
    spin: true,
  },
  completed: {
    icon: CheckCircle2,
    color: 'text-green-500',
    bg: 'bg-green-500/10',
    border: 'border-green-500/50',
    spin: false,
  },
  error: {
    icon: XCircle,
    color: 'text-red-500',
    bg: 'bg-red-500/10',
    border: 'border-red-500/50',
    spin: false,
  },
} as const

interface ScanProgressPanelProps {
  activeScan: ScanProgress | null
  progress: number
}

export function ScanProgressPanel({ activeScan, progress }: ScanProgressPanelProps) {
  if (!activeScan) return null

  const config = statusConfig[activeScan.status]
  const Icon = config.icon
  const isRunning =
    activeScan.status === 'starting' ||
    activeScan.status === 'scanning' ||
    activeScan.status === 'processing'

  return (
    <Card className={cn('transition-all', config.border, config.bg)}>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Icon className={cn('h-4 w-4', config.color, config.spin && 'animate-spin')} />
          {isRunning && `Scanning ${activeScan.targetCidr}...`}
          {activeScan.status === 'completed' && 'Scan Completed'}
          {activeScan.status === 'error' && 'Scan Failed'}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 pt-0">
        {/* Progress Bar */}
        {isRunning && (
          <div className="space-y-1">
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>
                {activeScan.devicesFound} device{activeScan.devicesFound !== 1 ? 's' : ''} found
                {activeScan.status === 'scanning' && ' (ping sweep in progress)'}
                {activeScan.status === 'processing' &&
                  ` of ${activeScan.hostsAlive} alive hosts`}
              </span>
              {activeScan.status === 'processing' && <span>{progress}%</span>}
            </div>
            <Progress
              value={activeScan.status === 'scanning' ? undefined : progress}
              className="h-2"
            />
          </div>
        )}

        {/* Completed summary */}
        {activeScan.status === 'completed' && (
          <p className="text-sm text-green-400">
            Found {activeScan.devicesFound} device{activeScan.devicesFound !== 1 ? 's' : ''} in{' '}
            {activeScan.targetCidr}
          </p>
        )}

        {/* Error message */}
        {activeScan.status === 'error' && (
          <p className="text-sm text-red-400">
            {activeScan.error || 'An error occurred during the scan'}
          </p>
        )}

        {/* Recently discovered devices */}
        {isRunning && activeScan.newDevices.length > 0 && (
          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">Recently found:</p>
            <div className="space-y-1 max-h-32 overflow-y-auto">
              {activeScan.newDevices
                .slice(-5)
                .reverse()
                .map((device) => (
                  <div
                    key={device.id}
                    className="flex items-center gap-2 text-xs p-2 rounded bg-muted/30"
                  >
                    <Wifi className="h-3 w-3 text-green-500 shrink-0" />
                    <span className="font-mono">{device.ip_addresses?.[0]}</span>
                    {device.hostname && (
                      <span className="text-muted-foreground truncate">{device.hostname}</span>
                    )}
                    {device.manufacturer && (
                      <span className="text-muted-foreground truncate ml-auto">
                        {device.manufacturer}
                      </span>
                    )}
                  </div>
                ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
