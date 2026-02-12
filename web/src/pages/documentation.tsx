import { useQuery } from '@tanstack/react-query'
import { FileText, Loader2, RefreshCw, Clock } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { listApplications } from '@/api/docs'
import type { DocsApplication } from '@/api/docs'
import { cn } from '@/lib/utils'

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
  const {
    data,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['docs-applications'],
    queryFn: () => listApplications(),
  })

  const applications = data?.items ?? []
  const total = data?.total ?? 0

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
        <div className="rounded-lg border border-red-500/50 bg-red-500/10 p-4">
          <p className="text-sm text-red-400">
            {error instanceof Error ? error.message : 'Failed to load applications'}
          </p>
          <Button variant="outline" size="sm" onClick={() => refetch()} className="mt-2">
            Try Again
          </Button>
        </div>
      )}

      {/* Loading state */}
      {isLoading && !error && (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {/* Empty state */}
      {!isLoading && !error && applications.length === 0 && (
        <Card>
          <CardContent className="py-16">
            <div className="text-center">
              <FileText className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium">No applications tracked yet</h3>
              <p className="text-sm text-muted-foreground mt-2 max-w-md mx-auto">
                Applications will appear here when collectors discover Docker containers
                and other services on your network.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Application cards */}
      {!isLoading && !error && applications.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {applications.map((app) => (
            <ApplicationCard key={app.id} application={app} />
          ))}
        </div>
      )}
    </div>
  )
}

function ApplicationCard({ application }: { application: DocsApplication }) {
  const statusStyle = STATUS_CONFIG[application.status] ?? STATUS_CONFIG.inactive
  const typeLabel = APP_TYPE_LABELS[application.app_type] ?? application.app_type

  return (
    <Card className="hover:border-green-500/50 transition-colors">
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
