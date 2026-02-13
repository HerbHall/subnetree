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
import { cn } from '@/lib/utils'
import { toast } from 'sonner'

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

export function DocumentationPage() {
  const queryClient = useQueryClient()
  const [selectedApp, setSelectedApp] = useState<DocsApplication | null>(null)
  const [compareIds, setCompareIds] = useState<[string | null, string | null]>([null, null])

  const {
    data,
    isLoading,
    error,
    refetch,
  } = useQuery({
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
            Track and version infrastructure configurations
          </p>
        </div>

        <div className="flex items-center gap-2">
          {/* Collector status badges */}
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
              onSelect={() => setSelectedApp(app)}
            />
          ))}
        </div>
      )}
    </div>
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
