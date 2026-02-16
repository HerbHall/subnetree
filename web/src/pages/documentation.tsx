import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  FileText,
  Loader2,
  RefreshCw,
  Clock,
  Play,
  CheckCircle2,
  XCircle,
  AlertCircle,
  History,
  ArrowLeft,
  Trash2,
  GitCompare,
  Download,
  Radio,
  Wifi,
  WifiOff,
  Search,
  Bell,
  BellOff,
  ChevronDown,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  listApplications,
  listCollectors,
  triggerCollection,
  getApplicationHistory,
  getSnapshotDiff,
  deleteSnapshot,
} from '@/api/docs'
import type { DocsApplication, DocsSnapshot } from '@/api/docs'
import {
  getChangelog,
  getStats,
  exportChangelog,
} from '@/api/autodoc'
import type { ChangelogEntry } from '@/api/autodoc'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'

type DocTab = 'snapshots' | 'changelog'

const APP_TYPE_LABELS: Record<string, string> = {
  docker: 'Docker',
  systemd: 'Systemd',
  kubernetes: 'Kubernetes',
  compose: 'Compose',
  manual: 'Manual',
}

const STATUS_CONFIG: Record<string, { bg: string; text: string; dot: string }> = {
  active: { bg: 'bg-green-500/10', text: 'text-green-500', dot: 'bg-green-500' },
  inactive: { bg: 'bg-gray-500/10', text: 'text-gray-500', dot: 'bg-gray-500' },
  error: { bg: 'bg-red-500/10', text: 'text-red-500', dot: 'bg-red-500' },
}

const EVENT_TYPE_LABELS: Record<string, { label: string; color: string }> = {
  'recon.device.discovered': { label: 'Discovered', color: 'text-green-500 bg-green-500/10' },
  'recon.device.updated': { label: 'Updated', color: 'text-blue-500 bg-blue-500/10' },
  'recon.device.lost': { label: 'Lost', color: 'text-orange-500 bg-orange-500/10' },
  'recon.scan.completed': { label: 'Scan', color: 'text-purple-500 bg-purple-500/10' },
  'pulse.alert.triggered': { label: 'Alert', color: 'text-red-500 bg-red-500/10' },
  'pulse.alert.resolved': { label: 'Resolved', color: 'text-green-500 bg-green-500/10' },
}

export function DocumentationPage() {
  const [activeTab, setActiveTab] = useState<DocTab>('changelog')
  const [selectedApp, setSelectedApp] = useState<DocsApplication | null>(null)
  const [compareIds, setCompareIds] = useState<[string | null, string | null]>([null, null])

  if (selectedApp) {
    return (
      <ApplicationHistoryView
        application={selectedApp}
        compareIds={compareIds}
        setCompareIds={setCompareIds}
        onBack={() => {
          setSelectedApp(null)
          setCompareIds([null, null])
        }}
      />
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Documentation</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Infrastructure that documents itself
          </p>
        </div>
      </div>

      {/* Tab switcher */}
      <div className="flex gap-1 p-1 bg-muted/50 rounded-lg w-fit">
        <button
          className={cn(
            'px-4 py-1.5 rounded-md text-sm font-medium transition-colors',
            activeTab === 'changelog'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground',
          )}
          onClick={() => setActiveTab('changelog')}
        >
          Changelog
        </button>
        <button
          className={cn(
            'px-4 py-1.5 rounded-md text-sm font-medium transition-colors',
            activeTab === 'snapshots'
              ? 'bg-background text-foreground shadow-sm'
              : 'text-muted-foreground hover:text-foreground',
          )}
          onClick={() => setActiveTab('snapshots')}
        >
          Config Snapshots
        </button>
      </div>

      {activeTab === 'changelog' ? (
        <ChangelogView />
      ) : (
        <SnapshotsView
          onSelectApp={(app) => setSelectedApp(app)}
        />
      )}
    </div>
  )
}

// ============================================================================
// Changelog View
// ============================================================================

function ChangelogView() {
  const [page, setPage] = useState(1)
  const [eventTypeFilter, setEventTypeFilter] = useState('')
  const [showFilter, setShowFilter] = useState(false)
  const perPage = 20

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['autodoc-changelog', page, perPage, eventTypeFilter],
    queryFn: () => getChangelog({
      page,
      per_page: perPage,
      event_type: eventTypeFilter || undefined,
    }),
  })

  const { data: stats } = useQuery({
    queryKey: ['autodoc-stats'],
    queryFn: () => getStats(),
  })

  const entries = data?.entries ?? []
  const total = data?.total ?? 0
  const totalPages = Math.ceil(total / perPage)

  async function handleExport(range_: string) {
    try {
      const md = await exportChangelog(range_)
      const blob = new Blob([md], { type: 'text/markdown' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `changelog-${range_}.md`
      a.click()
      URL.revokeObjectURL(url)
      toast.success('Changelog exported')
    } catch {
      toast.error('Failed to export changelog')
    }
  }

  return (
    <>
      {/* Stats cards */}
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
                  <p className="text-xs text-muted-foreground">Scans</p>
                  <p className="text-2xl font-bold mt-1">
                    {stats.entries_by_type['recon.scan.completed'] ?? 0}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-purple-500/10">
                  <Search className="h-5 w-5 text-purple-600 dark:text-purple-400" />
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
                    {(stats.entries_by_type['pulse.alert.triggered'] ?? 0) +
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

      {/* Controls */}
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
              {EVENT_TYPE_LABELS[eventTypeFilter]?.label ?? eventTypeFilter}
              <button
                className="text-muted-foreground hover:text-foreground"
                onClick={() => { setEventTypeFilter(''); setPage(1) }}
              >
                x
              </button>
            </span>
          )}
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => handleExport('7d')}>
            <Download className="h-4 w-4 mr-1.5" />
            Export 7d
          </Button>
          <Button variant="outline" size="sm" onClick={() => handleExport('30d')}>
            <Download className="h-4 w-4 mr-1.5" />
            Export 30d
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={() => refetch()}
            disabled={isLoading}
          >
            <RefreshCw className={cn('h-4 w-4', isLoading && 'animate-spin')} />
          </Button>
        </div>
      </div>

      {/* Filter panel */}
      {showFilter && (
        <div className="flex flex-wrap gap-2">
          {Object.entries(EVENT_TYPE_LABELS).map(([type, config]) => (
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
      {error && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <AlertCircle className="h-12 w-12 text-red-400 mb-4" />
          <h3 className="text-lg font-medium">Failed to load changelog</h3>
          <p className="text-sm text-muted-foreground mt-1 max-w-sm">
            {error instanceof Error ? error.message : 'An unexpected error occurred.'}
          </p>
          <Button variant="outline" className="mt-4 gap-2" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
            Try Again
          </Button>
        </div>
      )}

      {/* Loading state */}
      {isLoading && !error && (
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
      {!isLoading && !error && entries.length === 0 && (
        <Card>
          <CardContent className="py-16">
            <div className="flex flex-col items-center text-center">
              <History className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium">No changelog entries yet</h3>
              <p className="text-sm text-muted-foreground mt-2 max-w-md">
                Events will appear here automatically as your infrastructure changes --
                device discoveries, scan results, alerts, and more.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Timeline */}
      {!isLoading && !error && entries.length > 0 && (
        <div className="space-y-2">
          {entries.map((entry) => (
            <ChangelogEntryCard key={entry.id} entry={entry} />
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
    </>
  )
}

function ChangelogEntryCard({ entry }: { entry: ChangelogEntry }) {
  const [expanded, setExpanded] = useState(false)
  const typeConfig = EVENT_TYPE_LABELS[entry.event_type] ?? { label: 'Event', color: 'text-muted-foreground bg-muted' }

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
            <div className="flex items-center gap-3 mt-1.5 text-xs text-muted-foreground">
              <span className="flex items-center gap-1">
                <Clock className="h-3 w-3" />
                {formatRelativeTime(entry.created_at)}
              </span>
              {entry.source_module && (
                <span>via {entry.source_module}</span>
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

function eventIcon(eventType: string) {
  const iconClass = 'h-5 w-5'
  switch (eventType) {
    case 'recon.device.discovered':
      return <Wifi className={cn(iconClass, 'text-green-500')} />
    case 'recon.device.updated':
      return <Radio className={cn(iconClass, 'text-blue-500')} />
    case 'recon.device.lost':
      return <WifiOff className={cn(iconClass, 'text-orange-500')} />
    case 'recon.scan.completed':
      return <Search className={cn(iconClass, 'text-purple-500')} />
    case 'pulse.alert.triggered':
      return <Bell className={cn(iconClass, 'text-red-500')} />
    case 'pulse.alert.resolved':
      return <BellOff className={cn(iconClass, 'text-green-500')} />
    default:
      return <FileText className={cn(iconClass, 'text-muted-foreground')} />
  }
}

// ============================================================================
// Config Snapshots View (existing functionality)
// ============================================================================

function SnapshotsView({ onSelectApp }: { onSelectApp: (app: DocsApplication) => void }) {
  const queryClient = useQueryClient()

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['docs-applications'],
    queryFn: () => listApplications(),
  })

  const { data: collectors } = useQuery({
    queryKey: ['docs-collectors'],
    queryFn: () => listCollectors(),
  })

  const collectMutation = useMutation({
    mutationFn: () => triggerCollection(),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['docs-applications'] })
      queryClient.invalidateQueries({ queryKey: ['docs-collectors'] })
      toast.success(
        `Collection complete: ${result.apps_discovered} apps discovered, ${result.snapshots_created} snapshots created`,
      )
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Collection failed')
    },
  })

  const applications = data?.items ?? []
  const total = data?.total ?? 0
  const availableCollectors = collectors?.filter((c) => c.available) ?? []

  return (
    <>
      {/* Controls */}
      <div className="flex items-center gap-2 justify-end">
        {collectors && collectors.length > 0 && (
          <div className="hidden sm:flex items-center gap-1.5 mr-2">
            {collectors.map((c) => (
              <span
                key={c.name}
                className={cn(
                  'inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs',
                  c.available
                    ? 'bg-green-500/10 text-green-500'
                    : 'bg-gray-500/10 text-gray-500',
                )}
              >
                {c.available ? (
                  <CheckCircle2 className="h-3 w-3" />
                ) : (
                  <XCircle className="h-3 w-3" />
                )}
                {c.name}
              </span>
            ))}
          </div>
        )}

        <Button
          variant="outline"
          size="sm"
          onClick={() => collectMutation.mutate()}
          disabled={collectMutation.isPending || availableCollectors.length === 0}
        >
          {collectMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin mr-1.5" />
          ) : (
            <Play className="h-4 w-4 mr-1.5" />
          )}
          Collect Now
        </Button>

        <Button
          variant="outline"
          size="icon"
          onClick={() => refetch()}
          disabled={isLoading}
        >
          <RefreshCw className={cn('h-4 w-4', isLoading && 'animate-spin')} />
        </Button>
      </div>

      {/* Stats */}
      {!isLoading && !error && applications.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-3">
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs text-muted-foreground">Total Applications</p>
                  <p className="text-2xl font-bold mt-1">{total}</p>
                </div>
                <div className="p-2.5 rounded-lg bg-muted/50">
                  <FileText className="h-5 w-5 text-muted-foreground" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs text-muted-foreground">Active</p>
                  <p className="text-2xl font-bold mt-1">
                    {applications.filter((a) => a.status === 'active').length}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-green-500/10">
                  <FileText className="h-5 w-5 text-green-600 dark:text-green-400" />
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs text-muted-foreground">Inactive</p>
                  <p className="text-2xl font-bold mt-1">
                    {applications.filter((a) => a.status === 'inactive').length}
                  </p>
                </div>
                <div className="p-2.5 rounded-lg bg-gray-500/10">
                  <FileText className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <AlertCircle className="h-12 w-12 text-red-400 mb-4" />
          <h3 className="text-lg font-medium">Failed to load applications</h3>
          <p className="text-sm text-muted-foreground mt-1 max-w-sm">
            {error instanceof Error ? error.message : 'An unexpected error occurred while loading application data.'}
          </p>
          <Button variant="outline" className="mt-4 gap-2" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
            Try Again
          </Button>
        </div>
      )}

      {/* Loading state */}
      {isLoading && !error && (
        <>
          <div className="grid gap-4 sm:grid-cols-3">
            {[...Array(3)].map((_, i) => (
              <Card key={i}>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div className="space-y-2">
                      <Skeleton className="h-3 w-28" />
                      <Skeleton className="h-8 w-12" />
                    </div>
                    <Skeleton className="h-10 w-10 rounded-lg" />
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {[...Array(6)].map((_, i) => (
              <Card key={i}>
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-5 w-16 rounded-full" />
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <Skeleton className="h-3 w-12" />
                      <Skeleton className="h-5 w-16 rounded" />
                    </div>
                    <div className="flex items-center justify-between">
                      <Skeleton className="h-3 w-16" />
                      <Skeleton className="h-3 w-20" />
                    </div>
                    <div className="flex items-center justify-between">
                      <Skeleton className="h-3 w-14" />
                      <Skeleton className="h-3 w-16" />
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </>
      )}

      {/* Empty state */}
      {!isLoading && !error && applications.length === 0 && (
        <Card>
          <CardContent className="py-16">
            <div className="flex flex-col items-center text-center">
              <FileText className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium">No applications tracked yet</h3>
              <p className="text-sm text-muted-foreground mt-2 max-w-md">
                Applications will appear here when collectors discover Docker containers
                and other services on your network.
              </p>
              {availableCollectors.length > 0 ? (
                <Button
                  className="mt-4 gap-2"
                  onClick={() => collectMutation.mutate()}
                  disabled={collectMutation.isPending}
                >
                  {collectMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Play className="h-4 w-4" />
                  )}
                  {collectMutation.isPending ? 'Collecting...' : 'Collect Now'}
                </Button>
              ) : (
                <p className="text-xs text-muted-foreground mt-4">
                  No collectors are currently available. Ensure Docker or systemd is running on a monitored host.
                </p>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Application cards */}
      {!isLoading && !error && applications.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {applications.map((app) => (
            <ApplicationCard
              key={app.id}
              application={app}
              onSelect={() => onSelectApp(app)}
            />
          ))}
        </div>
      )}
    </>
  )
}

function ApplicationCard({
  application,
  onSelect,
}: {
  application: DocsApplication
  onSelect: () => void
}) {
  const statusStyle = STATUS_CONFIG[application.status] ?? STATUS_CONFIG.inactive
  const typeLabel = APP_TYPE_LABELS[application.app_type] ?? application.app_type

  return (
    <Card
      className="hover:border-green-500/50 transition-colors cursor-pointer"
      onClick={onSelect}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <CardTitle className="text-sm font-medium truncate">
            {application.name}
          </CardTitle>
          <span
            className={cn(
              'inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs shrink-0',
              statusStyle.bg,
              statusStyle.text,
            )}
          >
            <span className={cn('h-1.5 w-1.5 rounded-full', statusStyle.dot)} />
            <span className="capitalize">{application.status}</span>
          </span>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-2 text-sm">
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Type</span>
            <span className="px-2 py-0.5 rounded bg-muted text-xs font-medium">
              {typeLabel}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Collector</span>
            <span className="text-xs font-mono">{application.collector}</span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Updated</span>
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <Clock className="h-3 w-3" />
              {formatRelativeTime(application.updated_at)}
            </span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function ApplicationHistoryView({
  application,
  compareIds,
  setCompareIds,
  onBack,
}: {
  application: DocsApplication
  compareIds: [string | null, string | null]
  setCompareIds: (ids: [string | null, string | null]) => void
  onBack: () => void
}) {
  const queryClient = useQueryClient()

  const { data: history, isLoading } = useQuery({
    queryKey: ['docs-history', application.id],
    queryFn: () => getApplicationHistory(application.id),
  })

  const { data: diffResult, isLoading: isDiffLoading } = useQuery({
    queryKey: ['docs-diff', compareIds[0], compareIds[1]],
    queryFn: () => getSnapshotDiff(compareIds[0]!, compareIds[1]!),
    enabled: !!compareIds[0] && !!compareIds[1],
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteSnapshot(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['docs-history', application.id] })
      toast.success('Snapshot deleted')
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Failed to delete snapshot')
    },
  })

  const snapshots = history?.snapshots ?? []
  const [firstId, secondId] = compareIds

  function toggleCompare(id: string) {
    if (firstId === id) {
      setCompareIds([null, secondId])
    } else if (secondId === id) {
      setCompareIds([firstId, null])
    } else if (!firstId) {
      setCompareIds([id, secondId])
    } else if (!secondId) {
      setCompareIds([firstId, id])
    } else {
      setCompareIds([secondId, id])
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={onBack}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-semibold">{application.name}</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Snapshot history
          </p>
        </div>
      </div>

      {/* Compare button */}
      {firstId && secondId && (
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={isDiffLoading}
          >
            <GitCompare className="h-4 w-4 mr-1.5" />
            {isDiffLoading ? 'Computing diff...' : 'Comparing snapshots'}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setCompareIds([null, null])}
          >
            Clear selection
          </Button>
        </div>
      )}

      {!firstId && !secondId && snapshots.length >= 2 && (
        <p className="text-sm text-muted-foreground">
          Select two snapshots to compare their configurations.
        </p>
      )}

      {/* Diff viewer */}
      {diffResult && diffResult.diff_text && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">
              Configuration Diff
            </CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="text-xs font-mono overflow-x-auto rounded bg-muted/50 p-4 leading-relaxed">
              {diffResult.diff_text.split('\n').map((line, i) => (
                <span
                  key={i}
                  className={cn(
                    'block',
                    line.startsWith('+') && !line.startsWith('@@') && 'text-green-500 bg-green-500/10',
                    line.startsWith('-') && !line.startsWith('@@') && 'text-red-500 bg-red-500/10',
                    line.startsWith('@@') && 'text-blue-500',
                  )}
                >
                  {line}
                </span>
              ))}
            </pre>
          </CardContent>
        </Card>
      )}

      {diffResult && !diffResult.diff_text && (
        <Card>
          <CardContent className="py-8">
            <p className="text-sm text-center text-muted-foreground">
              No differences between the selected snapshots.
            </p>
          </CardContent>
        </Card>
      )}

      {/* Loading state */}
      {isLoading && (
        <div className="space-y-3">
          {[...Array(4)].map((_, i) => (
            <Card key={i}>
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-2 flex-1">
                    <div className="flex items-center gap-2">
                      <Skeleton className="h-3 w-16" />
                      <Skeleton className="h-5 w-12 rounded" />
                      <Skeleton className="h-3 w-20" />
                    </div>
                    <div className="flex items-center gap-1">
                      <Skeleton className="h-3 w-3 rounded" />
                      <Skeleton className="h-3 w-16" />
                      <Skeleton className="h-3 w-24" />
                    </div>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <Skeleton className="h-8 w-24 rounded-md" />
                    <Skeleton className="h-8 w-8 rounded-md" />
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Empty state */}
      {!isLoading && snapshots.length === 0 && (
        <Card>
          <CardContent className="py-12">
            <div className="text-center">
              <History className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
              <h3 className="text-base font-medium">No snapshots yet</h3>
              <p className="text-sm text-muted-foreground mt-1">
                Snapshots will appear after the next collection run.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Snapshot timeline */}
      {!isLoading && snapshots.length > 0 && (
        <div className="space-y-3">
          {snapshots.map((snap) => (
            <SnapshotCard
              key={snap.id}
              snapshot={snap}
              isSelected={firstId === snap.id || secondId === snap.id}
              onToggleCompare={() => toggleCompare(snap.id)}
              onDelete={() => deleteMutation.mutate(snap.id)}
              isDeleting={deleteMutation.isPending}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function SnapshotCard({
  snapshot,
  isSelected,
  onToggleCompare,
  onDelete,
  isDeleting,
}: {
  snapshot: DocsSnapshot
  isSelected: boolean
  onToggleCompare: () => void
  onDelete: () => void
  isDeleting: boolean
}) {
  return (
    <Card className={cn(isSelected && 'border-blue-500/50 bg-blue-500/5')}>
      <CardContent className="p-4">
        <div className="flex items-center justify-between">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="text-xs font-mono text-muted-foreground truncate">
                {snapshot.id.slice(0, 8)}
              </span>
              <span className="px-1.5 py-0.5 rounded bg-muted text-xs">
                {snapshot.format}
              </span>
              <span className="text-xs text-muted-foreground">
                {snapshot.size_bytes} bytes
              </span>
            </div>
            <div className="flex items-center gap-1 mt-1 text-xs text-muted-foreground">
              <Clock className="h-3 w-3" />
              {formatRelativeTime(snapshot.captured_at)}
              <span className="ml-2">via {snapshot.source}</span>
            </div>
          </div>

          <div className="flex items-center gap-1.5 shrink-0 ml-4">
            <Button
              variant={isSelected ? 'default' : 'outline'}
              size="sm"
              onClick={onToggleCompare}
            >
              <GitCompare className="h-3.5 w-3.5 mr-1" />
              {isSelected ? 'Selected' : 'Compare'}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-muted-foreground hover:text-red-500"
              onClick={onDelete}
              disabled={isDeleting}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
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
