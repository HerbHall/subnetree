import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, AlertCircle, CheckCircle2 } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { getDeviceAnomalies } from '@/api/insight'
import type { Anomaly } from '@/api/insight'

const MAX_VISIBLE = 10

function severityIcon(severity: string) {
  switch (severity) {
    case 'critical':
      return <AlertCircle className="h-4 w-4 text-red-500 shrink-0" />
    case 'warning':
      return <AlertTriangle className="h-4 w-4 text-amber-500 shrink-0" />
    default:
      return <AlertCircle className="h-4 w-4 text-blue-500 shrink-0" />
  }
}

function severityBadgeClass(severity: string) {
  switch (severity) {
    case 'critical':
      return 'bg-red-500/10 text-red-600 dark:text-red-400'
    case 'warning':
      return 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
    default:
      return 'bg-blue-500/10 text-blue-600 dark:text-blue-400'
  }
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

interface AnomalyIndicatorsProps {
  deviceId: string
}

export function AnomalyIndicators({ deviceId }: AnomalyIndicatorsProps) {
  const { data: anomalies, isLoading } = useQuery({
    queryKey: ['insight', 'anomalies', deviceId],
    queryFn: () => getDeviceAnomalies(deviceId),
    refetchInterval: 60000,
  })

  const visible = anomalies?.slice(0, MAX_VISIBLE) ?? []
  const totalCount = anomalies?.length ?? 0

  return (
    <Card>
      <CardHeader className="pb-3 flex flex-row items-center justify-between">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          Anomalies
        </CardTitle>
        {totalCount > MAX_VISIBLE && (
          <span className="text-xs text-muted-foreground">
            {totalCount} total
          </span>
        )}
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-2">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-14" />
            ))}
          </div>
        ) : visible.length === 0 ? (
          <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <CheckCircle2 className="h-5 w-5 text-green-500" />
            <span>No anomalies detected</span>
          </div>
        ) : (
          <div className="space-y-2">
            {visible.map((anomaly: Anomaly) => (
              <div
                key={anomaly.id}
                className="flex items-start gap-3 p-2 rounded-lg hover:bg-muted/50"
              >
                {severityIcon(anomaly.severity)}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium truncate">
                      {anomaly.metric_name}
                    </p>
                    <span
                      className={cn(
                        'text-xs px-1.5 py-0.5 rounded shrink-0',
                        severityBadgeClass(anomaly.severity)
                      )}
                    >
                      {anomaly.severity}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Value: {anomaly.value.toFixed(1)} (expected: {anomaly.expected.toFixed(1)})
                  </p>
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {formatRelativeTime(anomaly.detected_at)}
                    {anomaly.resolved_at && ' (resolved)'}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
