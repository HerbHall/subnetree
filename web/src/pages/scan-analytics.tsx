import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  type TooltipContentProps,
} from 'recharts'
import {
  Activity,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Clock,
  Monitor,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { HelpIcon, HelpPopover } from '@/components/contextual-help'
import {
  getHealthScore,
  getRawMetrics,
  getAggregateMetrics,
  type RawScanMetric,
  type ScanMetricsAggregate,
  type HealthScoreResponse,
} from '@/api/scan-metrics'

type TimeRange = '7d' | '30d' | '90d' | '1y'

const TIME_RANGES: { label: string; value: TimeRange }[] = [
  { label: '7 Days', value: '7d' },
  { label: '30 Days', value: '30d' },
  { label: '90 Days', value: '90d' },
  { label: '1 Year', value: '1y' },
]

// ============================================================================
// Chart data helpers
// ============================================================================

interface DurationChartPoint {
  date: string
  total: number
  ping: number
  enrich: number
}

interface DeviceChartPoint {
  date: string
  hostsAlive: number
  devicesCreated: number
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

function formatMs(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function rawToDurationPoints(metrics: RawScanMetric[]): DurationChartPoint[] {
  // Metrics come in DESC order; reverse for chronological chart.
  const sorted = [...metrics].reverse()
  return sorted.map((m) => ({
    date: formatDate(m.created_at),
    total: Math.round(m.duration_ms),
    ping: Math.round(m.ping_phase_ms),
    enrich: Math.round(m.enrich_phase_ms),
  }))
}

function aggToDurationPoints(aggs: ScanMetricsAggregate[]): DurationChartPoint[] {
  const sorted = [...aggs].reverse()
  return sorted.map((a) => ({
    date: formatDate(a.period_start),
    total: Math.round(a.avg_duration_ms),
    ping: Math.round(a.avg_ping_phase_ms),
    enrich: Math.round(a.avg_enrich_ms),
  }))
}

function rawToDevicePoints(metrics: RawScanMetric[]): DeviceChartPoint[] {
  const sorted = [...metrics].reverse()
  return sorted.map((m) => ({
    date: formatDate(m.created_at),
    hostsAlive: m.hosts_alive,
    devicesCreated: m.devices_created,
  }))
}

function aggToDevicePoints(aggs: ScanMetricsAggregate[]): DeviceChartPoint[] {
  const sorted = [...aggs].reverse()
  return sorted.map((a) => ({
    date: formatDate(a.period_start),
    hostsAlive: Math.round(a.avg_hosts_alive),
    devicesCreated: a.total_new_devices,
  }))
}

// ============================================================================
// Tooltips
// ============================================================================

function DurationTooltip({ active, payload }: Partial<TooltipContentProps<number, string>>) {
  if (!active || !payload || payload.length === 0) return null
  const data = payload[0].payload as DurationChartPoint
  return (
    <div className="rounded-md border bg-popover px-3 py-2 text-xs shadow-md">
      <p className="font-medium">{data.date}</p>
      <p className="text-muted-foreground">Total: {formatMs(data.total)}</p>
      <p className="text-muted-foreground">Ping: {formatMs(data.ping)}</p>
      <p className="text-muted-foreground">Enrich: {formatMs(data.enrich)}</p>
    </div>
  )
}

function DeviceTooltip({ active, payload }: Partial<TooltipContentProps<number, string>>) {
  if (!active || !payload || payload.length === 0) return null
  const data = payload[0].payload as DeviceChartPoint
  return (
    <div className="rounded-md border bg-popover px-3 py-2 text-xs shadow-md">
      <p className="font-medium">{data.date}</p>
      <p className="text-muted-foreground">Hosts alive: {data.hostsAlive}</p>
      <p className="text-muted-foreground">New devices: {data.devicesCreated}</p>
    </div>
  )
}

// ============================================================================
// Health Score Card
// ============================================================================

function HealthScoreCard({ data, isLoading }: { data?: HealthScoreResponse; isLoading: boolean }) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Activity className="h-4 w-4 text-muted-foreground" />
            Network Health Score
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-6">
            <Skeleton className="h-24 w-24 rounded-full" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-1/2" />
              <Skeleton className="h-4 w-2/3" />
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!data) {
    return (
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Activity className="h-4 w-4 text-muted-foreground" />
            Network Health Score
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground text-center py-4">
            No scan data available to compute health score.
          </p>
        </CardContent>
      </Card>
    )
  }

  const gradeColors = {
    green: {
      ring: 'border-green-500',
      text: 'text-green-600 dark:text-green-400',
      bg: 'bg-green-500/10',
      label: 'Healthy',
    },
    yellow: {
      ring: 'border-amber-500',
      text: 'text-amber-600 dark:text-amber-400',
      bg: 'bg-amber-500/10',
      label: 'Needs Attention',
    },
    red: {
      ring: 'border-red-500',
      text: 'text-red-600 dark:text-red-400',
      bg: 'bg-red-500/10',
      label: 'Unhealthy',
    },
  }
  const gc = gradeColors[data.grade]

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Activity className="h-4 w-4 text-muted-foreground" />
          Network Health Score
          <HelpPopover title="Scan Health Score">
            <p className="text-xs text-muted-foreground">
              A composite score (0-100) based on scan success rate, device discovery rate, and response times.
              Above 80 is healthy, 50-80 needs attention, below 50 indicates problems.
            </p>
          </HelpPopover>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-6">
          {/* Score circle */}
          <div
            className={cn(
              'flex items-center justify-center h-24 w-24 rounded-full border-4 shrink-0',
              gc.ring,
              gc.bg,
            )}
          >
            <div className="text-center">
              <span className={cn('text-3xl font-bold', gc.text)}>{data.score}</span>
              <p className={cn('text-xs font-medium', gc.text)}>{gc.label}</p>
            </div>
          </div>

          {/* Factor breakdown */}
          <div className="flex-1 space-y-1.5">
            {data.factors.map((f) => (
              <div key={f.name} className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground capitalize">
                  {f.name.replace(/_/g, ' ')}
                </span>
                <span className="font-medium">
                  {Math.round(f.score)}/{100}
                  <span className="text-muted-foreground ml-1">({Math.round(f.weight * 100)}%)</span>
                </span>
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

// ============================================================================
// Scan History Table
// ============================================================================

interface ScanHistoryEntry {
  date: string
  durationMs: number
  devicesFound: number
  status: 'completed' | 'failed'
}

function rawToHistory(metrics: RawScanMetric[]): ScanHistoryEntry[] {
  return metrics.map((m) => ({
    date: new Date(m.created_at).toLocaleString(),
    durationMs: m.duration_ms,
    devicesFound: m.hosts_alive,
    status: m.duration_ms > 0 ? 'completed' as const : 'failed' as const,
  }))
}

function ScanHistoryTable({ data, isLoading }: { data: ScanHistoryEntry[]; isLoading: boolean }) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            Recent Scans
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {[...Array(5)].map((_, i) => (
              <Skeleton key={i} className="h-10" />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Clock className="h-4 w-4 text-muted-foreground" />
          Recent Scans
        </CardTitle>
      </CardHeader>
      <CardContent>
        {data.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            No scan history available.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-xs text-muted-foreground">
                  <th className="pb-2 pr-4">Date</th>
                  <th className="pb-2 pr-4">Duration</th>
                  <th className="pb-2 pr-4">Devices</th>
                  <th className="pb-2">Status</th>
                </tr>
              </thead>
              <tbody>
                {data.map((entry, idx) => (
                  <tr key={idx} className="border-b last:border-0 hover:bg-muted/50">
                    <td className="py-2 pr-4 text-xs">{entry.date}</td>
                    <td className="py-2 pr-4 font-medium">{formatMs(entry.durationMs)}</td>
                    <td className="py-2 pr-4">
                      <span className="flex items-center gap-1">
                        <Monitor className="h-3 w-3 text-muted-foreground" />
                        {entry.devicesFound}
                      </span>
                    </td>
                    <td className="py-2">
                      {entry.status === 'completed' ? (
                        <span className="flex items-center gap-1 text-green-600 dark:text-green-400">
                          <CheckCircle2 className="h-3 w-3" />
                          OK
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-red-600 dark:text-red-400">
                          <XCircle className="h-3 w-3" />
                          Failed
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ============================================================================
// Main Page
// ============================================================================

export function ScanAnalyticsPage() {
  const [timeRange, setTimeRange] = useState<TimeRange>('7d')

  // Health score
  const { data: healthScore, isLoading: healthLoading } = useQuery({
    queryKey: ['scan-health-score'],
    queryFn: getHealthScore,
    staleTime: 60 * 1000,
  })

  // Raw metrics for 7d/30d
  const { data: rawMetrics, isLoading: rawLoading } = useQuery({
    queryKey: ['scan-raw-metrics', timeRange],
    queryFn: () => getRawMetrics(timeRange === '7d' ? 50 : 200),
    enabled: timeRange === '7d' || timeRange === '30d',
  })

  // Weekly aggregates for 90d
  const { data: weeklyAggs, isLoading: weeklyLoading } = useQuery({
    queryKey: ['scan-weekly-aggs'],
    queryFn: () => getAggregateMetrics('weekly', 13),
    enabled: timeRange === '90d',
  })

  // Monthly aggregates for 1y
  const { data: monthlyAggs, isLoading: monthlyLoading } = useQuery({
    queryKey: ['scan-monthly-aggs'],
    queryFn: () => getAggregateMetrics('monthly', 12),
    enabled: timeRange === '1y',
  })

  // Derive chart data based on time range
  const durationPoints = useMemo(() => {
    if (timeRange === '7d' || timeRange === '30d') {
      return rawMetrics ? rawToDurationPoints(rawMetrics) : []
    }
    if (timeRange === '90d') {
      return weeklyAggs ? aggToDurationPoints(weeklyAggs) : []
    }
    return monthlyAggs ? aggToDurationPoints(monthlyAggs) : []
  }, [timeRange, rawMetrics, weeklyAggs, monthlyAggs])

  const devicePoints = useMemo(() => {
    if (timeRange === '7d' || timeRange === '30d') {
      return rawMetrics ? rawToDevicePoints(rawMetrics) : []
    }
    if (timeRange === '90d') {
      return weeklyAggs ? aggToDevicePoints(weeklyAggs) : []
    }
    return monthlyAggs ? aggToDevicePoints(monthlyAggs) : []
  }, [timeRange, rawMetrics, weeklyAggs, monthlyAggs])

  const historyEntries = useMemo(() => {
    if (!rawMetrics) return []
    return rawToHistory(rawMetrics.slice(0, 20))
  }, [rawMetrics])

  const chartsLoading =
    (timeRange === '7d' || timeRange === '30d') ? rawLoading :
    timeRange === '90d' ? weeklyLoading :
    monthlyLoading

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Scan Analytics</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Historical scan trends and network health scoring
          </p>
        </div>

        {/* Time range toggle */}
        <div className="flex items-center gap-1 rounded-md border bg-muted/30 p-1">
          {TIME_RANGES.map((tr) => (
            <Button
              key={tr.value}
              variant={timeRange === tr.value ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => setTimeRange(tr.value)}
              className="h-7 px-3 text-xs"
            >
              {tr.label}
            </Button>
          ))}
        </div>
      </div>

      {/* Health Score Card */}
      <HealthScoreCard data={healthScore} isLoading={healthLoading} />

      {/* Scan Duration Chart */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Activity className="h-4 w-4 text-muted-foreground" />
            Scan Duration
            <HelpIcon content="Time taken for each network scan, broken down by ping (host discovery) and enrich (device detail collection) phases." />
          </CardTitle>
        </CardHeader>
        <CardContent>
          {chartsLoading ? (
            <Skeleton className="h-64 w-full" />
          ) : durationPoints.length < 2 ? (
            <div className="flex flex-col items-center py-12 text-center">
              <AlertTriangle className="h-8 w-8 text-muted-foreground mb-2" />
              <p className="text-sm text-muted-foreground">
                Not enough data points for this time range.
              </p>
            </div>
          ) : (
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={durationPoints} margin={{ top: 5, right: 10, bottom: 5, left: 10 }}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                  <XAxis
                    dataKey="date"
                    tick={{ fontSize: 11 }}
                    className="text-muted-foreground"
                  />
                  <YAxis
                    tick={{ fontSize: 11 }}
                    tickFormatter={(v: number) => formatMs(v)}
                    className="text-muted-foreground"
                  />
                  <Tooltip content={<DurationTooltip />} />
                  <defs>
                    <linearGradient id="totalFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--nv-green-400)" stopOpacity={0.3} />
                      <stop offset="100%" stopColor="var(--nv-green-400)" stopOpacity={0.05} />
                    </linearGradient>
                    <linearGradient id="pingFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.2} />
                      <stop offset="100%" stopColor="#3b82f6" stopOpacity={0.02} />
                    </linearGradient>
                    <linearGradient id="enrichFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="#f59e0b" stopOpacity={0.2} />
                      <stop offset="100%" stopColor="#f59e0b" stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <Area
                    type="monotone"
                    dataKey="total"
                    name="Total"
                    stroke="var(--nv-green-400)"
                    fill="url(#totalFill)"
                    strokeWidth={2}
                    dot={false}
                    isAnimationActive={false}
                  />
                  <Area
                    type="monotone"
                    dataKey="ping"
                    name="Ping"
                    stroke="#3b82f6"
                    fill="url(#pingFill)"
                    strokeWidth={1.5}
                    dot={false}
                    isAnimationActive={false}
                  />
                  <Area
                    type="monotone"
                    dataKey="enrich"
                    name="Enrich"
                    stroke="#f59e0b"
                    fill="url(#enrichFill)"
                    strokeWidth={1.5}
                    dot={false}
                    isAnimationActive={false}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}

          {/* Legend */}
          {durationPoints.length >= 2 && (
            <div className="flex items-center gap-4 mt-3 text-xs text-muted-foreground">
              <span className="flex items-center gap-1">
                <span className="inline-block h-2 w-2 rounded-full bg-[var(--nv-green-400)]" />
                Total
              </span>
              <span className="flex items-center gap-1">
                <span className="inline-block h-2 w-2 rounded-full bg-blue-500" />
                Ping Phase
              </span>
              <span className="flex items-center gap-1">
                <span className="inline-block h-2 w-2 rounded-full bg-amber-500" />
                Enrich Phase
              </span>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Device Count Trend */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Monitor className="h-4 w-4 text-muted-foreground" />
            Device Discovery Trend
            <HelpIcon content="Number of hosts responding to ping (Hosts Alive) and newly discovered devices over time." />
          </CardTitle>
        </CardHeader>
        <CardContent>
          {chartsLoading ? (
            <Skeleton className="h-64 w-full" />
          ) : devicePoints.length < 2 ? (
            <div className="flex flex-col items-center py-12 text-center">
              <AlertTriangle className="h-8 w-8 text-muted-foreground mb-2" />
              <p className="text-sm text-muted-foreground">
                Not enough data points for this time range.
              </p>
            </div>
          ) : (
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={devicePoints} margin={{ top: 5, right: 10, bottom: 5, left: 10 }}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                  <XAxis
                    dataKey="date"
                    tick={{ fontSize: 11 }}
                    className="text-muted-foreground"
                  />
                  <YAxis
                    tick={{ fontSize: 11 }}
                    className="text-muted-foreground"
                  />
                  <Tooltip content={<DeviceTooltip />} />
                  <defs>
                    <linearGradient id="aliveFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--nv-green-400)" stopOpacity={0.3} />
                      <stop offset="100%" stopColor="var(--nv-green-400)" stopOpacity={0.05} />
                    </linearGradient>
                    <linearGradient id="createdFill" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="#8b5cf6" stopOpacity={0.2} />
                      <stop offset="100%" stopColor="#8b5cf6" stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <Area
                    type="monotone"
                    dataKey="hostsAlive"
                    name="Hosts Alive"
                    stroke="var(--nv-green-400)"
                    fill="url(#aliveFill)"
                    strokeWidth={2}
                    dot={false}
                    isAnimationActive={false}
                  />
                  <Area
                    type="monotone"
                    dataKey="devicesCreated"
                    name="New Devices"
                    stroke="#8b5cf6"
                    fill="url(#createdFill)"
                    strokeWidth={1.5}
                    dot={false}
                    isAnimationActive={false}
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}

          {/* Legend */}
          {devicePoints.length >= 2 && (
            <div className="flex items-center gap-4 mt-3 text-xs text-muted-foreground">
              <span className="flex items-center gap-1">
                <span className="inline-block h-2 w-2 rounded-full bg-[var(--nv-green-400)]" />
                Hosts Alive
              </span>
              <span className="flex items-center gap-1">
                <span className="inline-block h-2 w-2 rounded-full bg-violet-500" />
                New Devices
              </span>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Scan History Table */}
      <ScanHistoryTable
        data={historyEntries}
        isLoading={rawLoading}
      />
    </div>
  )
}
