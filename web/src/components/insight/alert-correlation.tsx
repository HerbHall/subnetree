import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { GitBranch, CheckCircle2 } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { getDeviceCorrelations } from '@/api/insight'
import type { AlertGroup } from '@/api/insight'

interface AlertCorrelationWidgetProps {
  deviceId: string
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return 'just now'
  if (diffMin < 60) return `${diffMin}m ago`
  const diffHrs = Math.floor(diffMin / 60)
  if (diffHrs < 24) return `${diffHrs}h ago`
  const diffDays = Math.floor(diffHrs / 24)
  return `${diffDays}d ago`
}

export function AlertCorrelationWidget({ deviceId }: AlertCorrelationWidgetProps) {
  const { data: groups, isLoading } = useQuery({
    queryKey: ['insight', 'correlations', deviceId],
    queryFn: () => getDeviceCorrelations(deviceId),
    refetchInterval: 60000,
  })

  const visible = groups ?? []

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <GitBranch className="h-4 w-4 text-muted-foreground" />
          Correlated Alerts
        </CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            {[...Array(2)].map((_, i) => (
              <Skeleton key={i} className="h-16" />
            ))}
          </div>
        ) : visible.length === 0 ? (
          <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <CheckCircle2 className="h-5 w-5 text-green-500" />
            <span>No correlated alerts</span>
          </div>
        ) : (
          <div className="space-y-3">
            {visible.map((group: AlertGroup) => (
              <div
                key={group.id}
                className="p-3 rounded-lg border bg-muted/30"
              >
                <div className="flex items-center justify-between mb-1.5">
                  <p className="text-sm font-medium">
                    {group.description || 'Correlated alert group'}
                  </p>
                  <span className="text-xs text-muted-foreground shrink-0 ml-2">
                    {formatRelativeTime(group.created_at)}
                  </span>
                </div>
                {group.root_cause && (
                  <p className="text-xs text-muted-foreground mb-1.5">
                    Root cause:{' '}
                    <Link
                      to={`/devices/${group.root_cause}`}
                      className="text-primary hover:underline"
                    >
                      {group.root_cause}
                    </Link>
                  </p>
                )}
                <div className="flex items-center gap-4 text-xs text-muted-foreground">
                  <span>{group.alert_count} alert{group.alert_count !== 1 ? 's' : ''}</span>
                  <span>{group.device_ids.length} device{group.device_ids.length !== 1 ? 's' : ''} affected</span>
                </div>
                {group.device_ids.length > 0 && (
                  <div className="flex flex-wrap gap-1.5 mt-2">
                    {group.device_ids
                      .filter((did) => did !== deviceId)
                      .slice(0, 5)
                      .map((did) => (
                        <Link
                          key={did}
                          to={`/devices/${did}`}
                          className="text-xs px-2 py-0.5 rounded-full bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
                        >
                          {did}
                        </Link>
                      ))}
                    {group.device_ids.filter((did) => did !== deviceId).length > 5 && (
                      <span className="text-xs px-2 py-0.5 text-muted-foreground">
                        +{group.device_ids.filter((did) => did !== deviceId).length - 5} more
                      </span>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
