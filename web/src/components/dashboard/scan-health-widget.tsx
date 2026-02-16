import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  Radar,
  ArrowRight,
  TrendingUp,
  TrendingDown,
  Minus,
} from 'lucide-react'
import {
  AreaChart,
  Area,
  ResponsiveContainer,
  YAxis,
  Tooltip,
  type TooltipContentProps,
} from 'recharts'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { listScanMetrics } from '@/api/devices'
import type { Scan } from '@/api/types'
import { cn } from '@/lib/utils'

interface DurationPoint {
  date: string
  durationSec: number
  devicesFound: number
}

function computeScanDurationSec(scan: Scan): number | null {
  if (!scan.completed_at) return null
  const start = new Date(scan.started_at).getTime()
  const end = new Date(scan.completed_at).getTime()
  if (Number.isNaN(start) || Number.isNaN(end)) return null
  const diff = (end - start) / 1000
  return diff > 0 ? diff : null
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  const mins = Math.floor(seconds / 60)
  const secs = Math.round(seconds % 60)
  return secs > 0 ? `${mins}m ${secs}s` : `${mins}m`
}

function computeStats(scans: Scan[]) {
  const completed = scans
    .filter((s) => s.status === 'completed' && s.completed_at)
    .sort(
      (a, b) =>
        new Date(a.started_at).getTime() - new Date(b.started_at).getTime()
    )

  const points: DurationPoint[] = []
  const durations: number[] = []

  for (const scan of completed) {
    const dur = computeScanDurationSec(scan)
    if (dur === null) continue
    durations.push(dur)
    points.push({
      date: new Date(scan.started_at).toLocaleDateString(),
      durationSec: Math.round(dur * 10) / 10,
      devicesFound: scan.devices_found,
    })
  }

  if (durations.length === 0) {
    return { points: [], avgDuration: 0, lastDuration: 0, trend: 'flat' as const, totalDevices: 0, onlineDevices: 0, deviceDelta: 0 }
  }

  const avgDuration =
    durations.reduce((sum, d) => sum + d, 0) / durations.length
  const lastDuration = durations[durations.length - 1]

  // Trend: compare last 3 scans average to previous 3
  let trend: 'up' | 'down' | 'flat' = 'flat'
  if (durations.length >= 4) {
    const recentThree = durations.slice(-3)
    const previousThree = durations.slice(-6, -3)
    if (previousThree.length > 0) {
      const recentAvg =
        recentThree.reduce((s, d) => s + d, 0) / recentThree.length
      const prevAvg =
        previousThree.reduce((s, d) => s + d, 0) / previousThree.length
      const changePercent = ((recentAvg - prevAvg) / prevAvg) * 100
      if (changePercent > 10) trend = 'up'
      else if (changePercent < -10) trend = 'down'
    }
  }

  // Device counts from most recent completed scan
  const latestScan = completed[completed.length - 1]
  const totalDevices = latestScan.devices_found

  // Delta: compare latest scan devices_found to the one before
  let deviceDelta = 0
  if (completed.length >= 2) {
    const previousScan = completed[completed.length - 2]
    deviceDelta = latestScan.devices_found - previousScan.devices_found
  }

  return {
    points,
    avgDuration,
    lastDuration,
    trend,
    totalDevices,
    onlineDevices: totalDevices,
    deviceDelta,
  }
}

function CustomTooltip({ active, payload }: TooltipContentProps<number, string>) {
  if (!active || !payload || payload.length === 0) return null
  const data = payload[0].payload as DurationPoint
  return (
    <div className="rounded-md border bg-popover px-3 py-2 text-xs shadow-md">
      <p className="font-medium">{data.date}</p>
      <p className="text-muted-foreground">
        Duration: {formatDuration(data.durationSec)}
      </p>
      <p className="text-muted-foreground">
        Devices: {data.devicesFound}
      </p>
    </div>
  )
}

export function ScanHealthWidget() {
  const { data: scans, isLoading } = useQuery({
    queryKey: ['scanMetrics', '7d'],
    queryFn: () => listScanMetrics('7d'),
    refetchInterval: 60 * 1000,
  })

  const stats = useMemo(() => {
    if (!scans || scans.length === 0) return null
    return computeStats(scans)
  }, [scans])

  const trendConfig = {
    up: {
      icon: TrendingUp,
      label: 'Slower',
      color: 'text-red-500',
    },
    down: {
      icon: TrendingDown,
      label: 'Faster',
      color: 'text-green-600 dark:text-green-400',
    },
    flat: {
      icon: Minus,
      label: 'Stable',
      color: 'text-muted-foreground',
    },
  }

  return (
    <Card>
      <CardHeader className="pb-3 flex flex-row items-center justify-between">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Radar className="h-4 w-4 text-muted-foreground" />
          Network Health
        </CardTitle>
        <Button variant="ghost" size="sm" asChild className="gap-1 text-xs">
          <Link to="/scan-analytics">
            View All
            <ArrowRight className="h-3 w-3" />
          </Link>
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-16 w-full" />
            <div className="grid grid-cols-3 gap-4">
              {[...Array(3)].map((_, i) => (
                <div key={i} className="text-center">
                  <Skeleton className="h-6 w-12 mx-auto" />
                  <Skeleton className="h-3 w-16 mx-auto mt-1" />
                </div>
              ))}
            </div>
          </div>
        ) : !stats || stats.points.length === 0 ? (
          <div className="flex flex-col items-center py-6 text-center">
            <Radar className="h-8 w-8 text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">
              No scan history yet
            </p>
            <p className="text-xs text-muted-foreground mt-1">
              Run a network scan to start tracking performance trends
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Sparkline */}
            {stats.points.length >= 2 && (
              <div className="h-16">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart
                    data={stats.points}
                    margin={{ top: 2, right: 2, bottom: 2, left: 2 }}
                  >
                    <YAxis hide domain={['dataMin', 'dataMax']} />
                    <Tooltip content={<CustomTooltip />} />
                    <defs>
                      <linearGradient id="scanDurationFill" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="var(--nv-green-400)" stopOpacity={0.3} />
                        <stop offset="100%" stopColor="var(--nv-green-400)" stopOpacity={0.05} />
                      </linearGradient>
                    </defs>
                    <Area
                      type="monotone"
                      dataKey="durationSec"
                      stroke="var(--nv-green-400)"
                      fill="url(#scanDurationFill)"
                      strokeWidth={1.5}
                      dot={false}
                      isAnimationActive={false}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            )}

            {/* Stats row */}
            <div className="grid grid-cols-3 gap-4">
              <div className="text-center">
                <p className="text-lg font-bold">
                  {formatDuration(stats.avgDuration)}
                </p>
                <p className="text-xs text-muted-foreground">Avg Duration</p>
              </div>
              <div className="text-center">
                <p className="text-lg font-bold">
                  {formatDuration(stats.lastDuration)}
                </p>
                <p className="text-xs text-muted-foreground">Last Scan</p>
              </div>
              <div className="text-center">
                {(() => {
                  const tc = trendConfig[stats.trend]
                  const TrendIcon = tc.icon
                  return (
                    <>
                      <p className={cn('text-lg font-bold flex items-center justify-center gap-1', tc.color)}>
                        <TrendIcon className="h-4 w-4" />
                        {tc.label}
                      </p>
                      <p className="text-xs text-muted-foreground">Trend</p>
                    </>
                  )
                })()}
              </div>
            </div>

            {/* Device summary */}
            <div className="flex items-center justify-between border-t pt-3">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium">
                  {stats.totalDevices} devices found
                </span>
                {stats.deviceDelta !== 0 && (
                  <span
                    className={cn(
                      'text-xs px-1.5 py-0.5 rounded',
                      stats.deviceDelta > 0
                        ? 'bg-green-500/10 text-green-600 dark:text-green-400'
                        : 'bg-red-500/10 text-red-600 dark:text-red-400'
                    )}
                  >
                    {stats.deviceDelta > 0 ? '+' : ''}
                    {stats.deviceDelta}
                  </span>
                )}
              </div>
              <span className="text-xs text-muted-foreground">
                {stats.points.length} scans
              </span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
