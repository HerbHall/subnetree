import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Monitor, AlertTriangle, ArrowRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { getInventorySummary } from '@/api/devices'
import { cn } from '@/lib/utils'

const DEVICE_TYPE_LABELS: Record<string, string> = {
  server: 'Server',
  desktop: 'Desktop',
  laptop: 'Laptop',
  mobile: 'Mobile',
  router: 'Router',
  switch: 'Switch',
  access_point: 'AP',
  firewall: 'Firewall',
  printer: 'Printer',
  nas: 'NAS',
  iot: 'IoT',
  phone: 'Phone',
  tablet: 'Tablet',
  camera: 'Camera',
  unknown: 'Unknown',
}

export function InventorySummaryWidget() {
  const { data: summary, isLoading } = useQuery({
    queryKey: ['inventorySummary'],
    queryFn: () => getInventorySummary(),
    refetchInterval: 30 * 1000,
  })

  return (
    <Card>
      <CardHeader className="pb-3 flex flex-row items-center justify-between">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Monitor className="h-4 w-4 text-muted-foreground" />
          Device Inventory
        </CardTitle>
        <Button variant="ghost" size="sm" asChild className="gap-1 text-xs">
          <Link to="/devices">
            View All
            <ArrowRight className="h-3 w-3" />
          </Link>
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            <div className="grid grid-cols-3 gap-4">
              {[...Array(3)].map((_, i) => (
                <div key={i} className="text-center">
                  <Skeleton className="h-8 w-12 mx-auto" />
                  <Skeleton className="h-3 w-16 mx-auto mt-1" />
                </div>
              ))}
            </div>
            <Skeleton className="h-4 w-full" />
          </div>
        ) : !summary || summary.total_devices === 0 ? (
          <div className="flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              No devices discovered yet. Run a network scan to get started.
            </p>
            <Button variant="outline" size="sm" asChild className="gap-2 shrink-0">
              <Link to="/devices">
                <Monitor className="h-3.5 w-3.5" />
                Devices
              </Link>
            </Button>
          </div>
        ) : (
          <div className="space-y-3">
            {/* Counts */}
            <div className="grid grid-cols-3 gap-4">
              <div className="text-center">
                <p className="text-2xl font-bold">{summary.total_devices}</p>
                <p className="text-xs text-muted-foreground">Total</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold text-green-600 dark:text-green-400">
                  {summary.online_count}
                </p>
                <p className="text-xs text-muted-foreground">Online</p>
              </div>
              <div className="text-center">
                <p className={cn(
                  'text-2xl font-bold',
                  summary.stale_count > 0
                    ? 'text-amber-600 dark:text-amber-400'
                    : 'text-muted-foreground'
                )}>
                  {summary.stale_count}
                </p>
                <p className="text-xs text-muted-foreground flex items-center justify-center gap-1">
                  {summary.stale_count > 0 && (
                    <AlertTriangle className="h-3 w-3 text-amber-500" />
                  )}
                  Stale
                </p>
              </div>
            </div>

            {/* Category breakdown as badges */}
            {Object.keys(summary.by_type).length > 0 && (
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-xs text-muted-foreground">Types:</span>
                {Object.entries(summary.by_type)
                  .sort(([, a], [, b]) => b - a)
                  .map(([type, count]) => (
                    <span
                      key={type}
                      className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-muted text-muted-foreground"
                    >
                      {DEVICE_TYPE_LABELS[type] || type}: {count}
                    </span>
                  ))}
              </div>
            )}

            {/* Stale warning link */}
            {summary.stale_count > 0 && (
              <Link
                to="/devices?status=stale"
                className="flex items-center gap-2 text-xs text-amber-600 dark:text-amber-400 hover:underline"
              >
                <AlertTriangle className="h-3.5 w-3.5" />
                {summary.stale_count} device{summary.stale_count !== 1 ? 's' : ''} not seen in 30+ days
              </Link>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
