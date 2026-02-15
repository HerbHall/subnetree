import { useQuery } from '@tanstack/react-query'
import {
  Sparkles,
  CheckCircle2,
  AlertTriangle,
  AlertCircle,
  Cpu,
  HardDrive,
  Database,
} from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { getRecommendations } from '@/api/insight'
import type { Recommendation } from '@/api/insight'

const MAX_VISIBLE = 5

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

function typeIcon(type: string) {
  switch (type) {
    case 'cpu':
      return <Cpu className="h-3.5 w-3.5 text-muted-foreground" />
    case 'memory':
      return <HardDrive className="h-3.5 w-3.5 text-muted-foreground" />
    case 'disk':
      return <Database className="h-3.5 w-3.5 text-muted-foreground" />
    default:
      return null
  }
}

export function RecommendationsWidget() {
  const { data: recommendations, isLoading } = useQuery({
    queryKey: ['insight', 'recommendations'],
    queryFn: getRecommendations,
    refetchInterval: 300000,
  })

  const visible = recommendations?.slice(0, MAX_VISIBLE) ?? []
  const totalCount = recommendations?.length ?? 0

  return (
    <Card>
      <CardHeader className="pb-3 flex flex-row items-center justify-between">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-muted-foreground" />
          AI Recommendations
        </CardTitle>
        {totalCount > MAX_VISIBLE && (
          <span className="text-xs text-muted-foreground">
            View all {totalCount}
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
            <span>Your network is performing well!</span>
          </div>
        ) : (
          <div className="space-y-2">
            {visible.map((rec: Recommendation) => (
              <div
                key={rec.id}
                className="flex items-start gap-3 p-2 rounded-lg hover:bg-muted/50"
              >
                {severityIcon(rec.severity)}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium truncate">{rec.title}</p>
                    {typeIcon(rec.type)}
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
                    {rec.description}
                  </p>
                  <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                    <span>{rec.current_value.toFixed(1)}%</span>
                    <span>/</span>
                    <span>{rec.threshold.toFixed(0)}% threshold</span>
                  </div>
                </div>
                <span
                  className={cn(
                    'text-xs px-1.5 py-0.5 rounded shrink-0',
                    severityBadgeClass(rec.severity)
                  )}
                >
                  {rec.severity}
                </span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
