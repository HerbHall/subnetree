import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
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
  Monitor,
  Search,
  ArrowRightLeft,
  Bell,
  BellOff,
  Activity,
  FileText,
  Clock,
  Download,
  RefreshCw,
  AlertCircle,
  History,
  Radio,
  Wifi,
  WifiOff,
  ChevronDown,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { getChangelog, getStats, exportChangelog } from '@/api/autodoc'
import type { ChangelogEntry, ChangelogFilter } from '@/api/autodoc'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

type TimeRange = '7d' | '30d' | '90d' | 'all'

const TIME_RANGES: { value: TimeRange; label: string }[] = [
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: '90d', label: '90 days' },
  { value: 'all', label: 'All time' },
]

const EVENT_TYPE_CONFIG: Record<string, { label: string; color: string; chartColor: string }> = {
  'recon.device.discovered': { label: 'Discovered', color: 'text-green-500 bg-green-500/10', chartColor: 'var(--nv-chart-green)' },
  'recon.device.updated': { label: 'Updated', color: 'text-blue-500 bg-blue-500/10', chartColor: 'var(--nv-chart-blue)' },
  'recon.device.lost': { label: 'Lost', color: 'text-orange-500 bg-orange-500/10', chartColor: 'var(--nv-chart-amber)' },
  'recon.scan.completed': { label: 'Scan', color: 'text-purple-500 bg-purple-500/10', chartColor: 'var(--nv-chart-sage)' },
  'recon.service.moved': { label: 'Service Moved', color: 'text-blue-400 bg-blue-400/10', chartColor: 'var(--nv-chart-blue)' },
  'pulse.alert.fired': { label: 'Alert Fired', color: 'text-red-500 bg-red-500/10', chartColor: 'var(--nv-chart-red)' },
  'pulse.alert.triggered': { label: 'Alert', color: 'text-red-500 bg-red-500/10', chartColor: 'var(--nv-chart-red)' },
  'pulse.alert.resolved': { label: 'Resolved', color: 'text-green-500 bg-green-500/10', chartColor: 'var(--nv-chart-green)' },
  'pulse.device.status_changed': { label: 'Status Changed', color: 'text-amber-500 bg-amber-500/10', chartColor: 'var(--nv-chart-amber)' },
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getTimeRangeSince(range: TimeRange): string | undefined {
  if (range === 'all') return undefined
  const days = parseInt(range, 10)
  const since = new Date()
  since.setDate(since.getDate() - days)
  return since.toISOString()
}

function eventIcon(eventType: string) {
  const iconClass = 'h-5 w-5'
  switch (eventType) {
    case 'recon.device.discovered':
      return <Monitor className={cn(iconClass, 'text-green-500')} />
    case 'recon.scan.completed':
      return <Search className={cn(iconClass, 'text-purple-500')} />
    case 'recon.service.moved':
      return <ArrowRightLeft className={cn(iconClass, 'text-blue-400')} />
    case 'pulse.alert.fired':
    case 'pulse.alert.triggered':
      return <Bell className={cn(iconClass, 'text-red-500')} />
    case 'pulse.alert.resolved':
      return <BellOff className={cn(iconClass, 'text-green-500')} />
    case 'pulse.device.status_changed':
      return <Activity className={cn(iconClass, 'text-amber-500')} />
    case 'recon.device.updated':
      return <Radio className={cn(iconClass, 'text-blue-500')} />
    case 'recon.device.lost':
      return <WifiOff className={cn(iconClass, 'text-orange-500')} />
    default:
      return <FileText className={cn(iconClass, 'text-muted-foreground')} />
  }
}

function formatRelativeTime(dateStr: string): string {
  try {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)

    if (diffMins < 1) return 'Just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    return `${diffDays}d ago`
  } catch {
    return dateStr
  }
}

function formatDate(dateStr: string): string {
  try {
    return new Intl.DateTimeFormat(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(dateStr))
  } catch {
    return dateStr
  }
}

function formatChartDate(dateStr: string): string {
  try {
    return new Intl.DateTimeFormat(undefined, {
      month: 'short',
      day: 'numeric',
    }).format(new Date(dateStr))
  } catch {
    return dateStr
  }
}

interface DailyBucket {
  date: string
  total: number
  [key: string]: string | number
}

/** Group changelog entries by day and count events per type. */
function bucketByDay(entries: ChangelogEntry[]): DailyBucket[] {
  const buckets = new Map<string, DailyBucket>()

  for (const entry of entries) {
    const date = entry.created_at.slice(0, 10) // YYYY-MM-DD
    let bucket = buckets.get(date)
    if (!bucket) {
      bucket = { date, total: 0 }
      buckets.set(date, bucket)
    }
    bucket.total += 1
    const typeKey = entry.event_type.replace(/\./g, '_')
    bucket[typeKey] = ((bucket[typeKey] as number) || 0) + 1
  }

  const sorted = Array.from(buckets.values()).sort((a, b) => a.date.localeCompare(b.date))
  return sorted
}

// ---------------------------------------------------------------------------
// Page component
// ---------------------------------------------------------------------------

export function TimelinePage() {
  const [timeRange, setTimeRange] = useState<TimeRange>('30d')
  const [eventTypeFilter, setEventTypeFilter] = useState('')
  const [showFilter, setShowFilter] = useState(false)
  const [page, setPage] = useState(1)
  const perPage = 25

  const since = getTimeRangeSince(timeRange)

  // Fetch stats
  const { data: stats } = useQuery({
    queryKey: ['timeline-stats'],
    queryFn: () => getStats(),
  })

  // Fetch all entries for chart (large page to get chart data)
  const { data: chartData, isLoading: chartLoading } = useQuery({
    queryKey: ['timeline-chart', timeRange],
    queryFn: () => getChangelog({
      per_page: 1000,
      since,
    }),
  })

  // Fetch paginated entries for event list
  const filter: ChangelogFilter = {
    page,
    per_page: perPage,
    event_type: eventTypeFilter || undefined,
    since,
  }

  const { data: listData, isLoading: listLoading, error: listError, refetch } = useQuery({
    queryKey: ['timeline-list', page, perPage, eventTypeFilter, timeRange],
    queryFn: () => getChangelog(filter),
  })

  const chartEntries = useMemo(() => chartData?.entries ?? [], [chartData?.entries])
  const dailyBuckets = useMemo(() => bucketByDay(chartEntries), [chartEntries])

  const entries = listData?.entries ?? []
  const total = listData?.total ?? 0
  const totalPages = Math.ceil(total / perPage)

  // Collect unique event types from chart data for area rendering
  const eventTypes = useMemo(() => {
    const types = new Set<string>()
    for (const entry of chartEntries) {
      types.add(entry.event_type)
    }
    return Array.from(types)
  }, [chartEntries])

  async function handleExport() {
    try {
      const md = await exportChangelog(timeRange === 'all' ? '9999d' : timeRange)
      const blob = new Blob([md], { type: 'text/markdown' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `timeline-${timeRange}.md`
      a.click()
      URL.revokeObjectURL(url)
      toast.success('Timeline exported')
    } catch {
      toast.error('Failed to export timeline')
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Timeline</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Network change history over time
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={handleExport}>
            <Download className="h-4 w-4 mr-1.5" />
            Export
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={() => refetch()}
            disabled={listLoading}
          >
            <RefreshCw className={cn('h-4 w-4', listLoading && 'animate-spin')} />
          </Button>
        </div>
      </div>

      {/* Time range selector */}
      <div className="flex gap-1 p-1 bg-muted/50 rounded-lg w-fit">
        {TIME_RANGES.map(({ value, label }) => (
          <button
            key={value}
            className={cn(
              'px-4 py-1.5 rounded-md text-sm font-medium transition-colors',
              timeRange === value
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground',
            )}
            onClick={() => { setTimeRange(value); setPage(1) }}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Stats bar */}
      {stats && stats.total_entries > 0 && (
        <div className="grid gap-4 sm:grid-cols-4">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs text-muted-foreground">Total Events</p>
                  <p className="text-2xl font-bold mt-1">{stats.total_entries}</p>
                </div>
                <div className="p-2.5 rounded-lg bg-muted/50">
                  <History className="h-5 w-5 text-muted-foreground" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs text-muted-foreground">Event Types</p>
                  <p className="text-2xl font-bold mt-1">
                    {Object.keys(stats.entries_by_type).length}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-blue-500/10">
                  <Activity className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs text-muted-foreground">Discoveries</p>
                  <p className="text-2xl font-bold mt-1">
                    {stats.entries_by_type['recon.device.discovered'] ?? 0}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-green-500/10">
                  <Wifi className="h-5 w-5 text-green-600 dark:text-green-400" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs text-muted-foreground">Alerts</p>
                  <p className="text-2xl font-bold mt-1">
                    {(stats.entries_by_type['pulse.alert.fired'] ?? 0) +
                     (stats.entries_by_type['pulse.alert.triggered'] ?? 0) +
                     (stats.entries_by_type['pulse.alert.resolved'] ?? 0)}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-red-500/10">
                  <Bell className="h-5 w-5 text-red-600 dark:text-red-400" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Chart */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Event Frequency</CardTitle>
        </CardHeader>
        <CardContent>
          {chartLoading && <Skeleton className="w-full h-[280px]" />}

          {!chartLoading && dailyBuckets.length === 0 && (
            <div className="flex items-center justify-center h-[280px] text-muted-foreground text-sm">
              No events in this time range.
            </div>
          )}

          {!chartLoading && dailyBuckets.length > 0 && (
            <ResponsiveContainer width="100%" height={280}>
              <AreaChart data={dailyBuckets} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid
                  strokeDasharray="3 3"
                  stroke="currentColor"
                  strokeOpacity={0.08}
                />
                <XAxis
                  dataKey="date"
                  tickFormatter={formatChartDate}
                  tick={{ fill: 'currentColor', fontSize: 12 }}
                  tickLine={{ stroke: 'currentColor', strokeOpacity: 0.2 }}
                  axisLine={{ stroke: 'currentColor', strokeOpacity: 0.2 }}
                  minTickGap={40}
                />
                <YAxis
                  tick={{ fill: 'currentColor', fontSize: 12 }}
                  tickLine={{ stroke: 'currentColor', strokeOpacity: 0.2 }}
                  axisLine={{ stroke: 'currentColor', strokeOpacity: 0.2 }}
                  allowDecimals={false}
                  width={40}
                />
                <Tooltip content={<ChartTooltip />} />
                {eventTypes.map((type) => {
                  const typeKey = type.replace(/\./g, '_')
                  const cfg = EVENT_TYPE_CONFIG[type]
                  const color = cfg?.chartColor ?? 'var(--nv-chart-sage)'
                  return (
                    <Area
                      key={type}
                      type="monotone"
                      dataKey={typeKey}
                      stackId="events"
                      stroke={color}
                      fill={color}
                      fillOpacity={0.2}
                      strokeWidth={1.5}
                      dot={false}
                    />
                  )
                })}
              </AreaChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>

      {/* Filter controls */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowFilter(!showFilter)}
          >
            <ChevronDown className={cn('h-4 w-4 mr-1.5 transition-transform', showFilter && 'rotate-180')} />
            Filter
          </Button>

          {eventTypeFilter && (
            <span className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs bg-muted">
              {EVENT_TYPE_CONFIG[eventTypeFilter]?.label ?? eventTypeFilter}
              <button
                className="text-muted-foreground hover:text-foreground"
                onClick={() => { setEventTypeFilter(''); setPage(1) }}
              >
                x
              </button>
            </span>
          )}
        </div>
        <p className="text-sm text-muted-foreground">
          {total} event{total !== 1 ? 's' : ''}
        </p>
      </div>

      {/* Filter panel */}
      {showFilter && (
        <div className="flex flex-wrap gap-2">
          {Object.entries(EVENT_TYPE_CONFIG).map(([type, config]) => (
            <button
              key={type}
              className={cn(
                'px-3 py-1 rounded-full text-xs font-medium transition-colors border',
                eventTypeFilter === type
                  ? 'border-foreground/30 bg-foreground/10'
                  : 'border-transparent bg-muted hover:bg-muted/80',
              )}
              onClick={() => {
                setEventTypeFilter(eventTypeFilter === type ? '' : type)
                setPage(1)
              }}
            >
              {config.label}
            </button>
          ))}
        </div>
      )}

      {/* Error state */}
      {listError && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <AlertCircle className="h-12 w-12 text-red-400 mb-4" />
          <h3 className="text-lg font-medium">Failed to load events</h3>
          <p className="text-sm text-muted-foreground mt-1 max-w-sm">
            {listError instanceof Error ? listError.message : 'An unexpected error occurred.'}
          </p>
          <Button variant="outline" className="mt-4 gap-2" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
            Try Again
          </Button>
        </div>
      )}

      {/* Loading state */}
      {listLoading && !listError && (
        <div className="space-y-3">
          {[...Array(5)].map((_, i) => (
            <Card key={i}>
              <CardContent className="p-4">
                <div className="flex items-start gap-3">
                  <Skeleton className="h-8 w-8 rounded-full shrink-0" />
                  <div className="flex-1 space-y-2">
                    <Skeleton className="h-4 w-3/4" />
                    <Skeleton className="h-3 w-1/3" />
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Empty state */}
      {!listLoading && !listError && entries.length === 0 && (
        <Card>
          <CardContent className="py-16">
            <div className="flex flex-col items-center text-center">
              <History className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium">No events in this time range</h3>
              <p className="text-sm text-muted-foreground mt-2 max-w-md">
                Network change events will appear here automatically as your
                infrastructure changes -- device discoveries, scan results,
                alerts, and more.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Event list */}
      {!listLoading && !listError && entries.length > 0 && (
        <div className="space-y-2">
          {entries.map((entry) => (
            <TimelineEventCard key={entry.id} entry={entry} />
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
          >
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {page} of {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => setPage(page + 1)}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Chart tooltip
// ---------------------------------------------------------------------------

function ChartTooltip({ active, payload, label }: TooltipContentProps<number, string>) {
  if (!active || !payload || payload.length === 0) return null

  const total = payload.reduce((sum, item) => sum + ((item.value as number) || 0), 0)

  return (
    <div className="rounded-lg border bg-card px-3 py-2 shadow-md">
      <p className="text-xs text-muted-foreground mb-1">
        {formatChartDate(label as string)}
      </p>
      <p className="text-sm font-medium mb-1">{total} event{total !== 1 ? 's' : ''}</p>
      {payload.map((item) => {
        const eventType = (item.dataKey as string).replace(/_/g, '.')
        const cfg = EVENT_TYPE_CONFIG[eventType]
        if (!cfg || !item.value) return null
        return (
          <p key={item.dataKey as string} className="text-xs text-muted-foreground">
            {cfg.label}: {item.value}
          </p>
        )
      })}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Event card
// ---------------------------------------------------------------------------

function TimelineEventCard({ entry }: { entry: ChangelogEntry }) {
  const [expanded, setExpanded] = useState(false)
  const typeConfig = EVENT_TYPE_CONFIG[entry.event_type] ?? {
    label: 'Event',
    color: 'text-muted-foreground bg-muted',
  }

  return (
    <Card
      className="hover:border-green-500/30 transition-colors cursor-pointer"
      onClick={() => setExpanded(!expanded)}
    >
      <CardContent className="p-4">
        <div className="flex items-start gap-3">
          <div className="shrink-0 mt-0.5">
            {eventIcon(entry.event_type)}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-2">
              <p className="text-sm font-medium leading-snug">{entry.summary}</p>
              <span className={cn(
                'inline-flex items-center px-2 py-0.5 rounded text-xs shrink-0',
                typeConfig.color,
              )}>
                {typeConfig.label}
              </span>
            </div>
            <div className="flex items-center gap-3 mt-1.5 text-xs text-muted-foreground flex-wrap">
              <span className="flex items-center gap-1" title={formatDate(entry.created_at)}>
                <Clock className="h-3 w-3" />
                {formatRelativeTime(entry.created_at)}
              </span>
              {entry.source_module && (
                <span>via {entry.source_module}</span>
              )}
              {entry.device_id && (
                <Link
                  to={`/devices/${entry.device_id}`}
                  className="text-[var(--nv-green-400)] hover:underline"
                  onClick={(e) => e.stopPropagation()}
                >
                  View device
                </Link>
              )}
            </div>
            {expanded && entry.details != null && (
              <pre className="mt-3 p-3 rounded bg-muted/50 text-xs font-mono overflow-x-auto max-h-64 overflow-y-auto">
                {JSON.stringify(entry.details, null, 2)}
              </pre>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
