import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  Layers,
  Container,
  Monitor,
  Settings2,
  AppWindow,
  ChevronDown,
  ChevronRight,
  AlertCircle,
  RefreshCw,
  Bot,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { listServices } from '@/api/services'
import type { Service, ServiceType, ServiceStatus } from '@/api/types'
import { cn } from '@/lib/utils'

const serviceTypeIcons: Record<ServiceType, React.ElementType> = {
  'docker-container': Container,
  'windows-service': Monitor,
  'systemd-service': Settings2,
  'application': AppWindow,
}

const serviceTypeLabels: Record<ServiceType, string> = {
  'docker-container': 'Docker',
  'windows-service': 'Windows',
  'systemd-service': 'Systemd',
  'application': 'App',
}

const statusConfig: Record<ServiceStatus, { bg: string; text: string; label: string }> = {
  running: { bg: 'bg-green-500', text: 'text-green-600 dark:text-green-400', label: 'Running' },
  stopped: { bg: 'bg-gray-400', text: 'text-gray-600 dark:text-gray-400', label: 'Stopped' },
  failed: { bg: 'bg-red-500', text: 'text-red-600 dark:text-red-400', label: 'Failed' },
  unknown: { bg: 'bg-amber-500', text: 'text-amber-600 dark:text-amber-400', label: 'Unknown' },
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

interface DeviceGroup {
  deviceId: string
  services: Service[]
}

export function ServiceMapPage() {
  const {
    data: services,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['services'],
    queryFn: () => listServices(),
  })

  // Group services by device
  const deviceGroups: DeviceGroup[] = (() => {
    if (!services || services.length === 0) return []
    const map = new Map<string, Service[]>()
    for (const svc of services) {
      const existing = map.get(svc.device_id) || []
      existing.push(svc)
      map.set(svc.device_id, existing)
    }
    return Array.from(map.entries())
      .map(([deviceId, svcs]) => ({ deviceId, services: svcs }))
      .sort((a, b) => {
        // Sort by service count descending
        return b.services.length - a.services.length
      })
  })()

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Service Map</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Services discovered across your infrastructure
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()} className="gap-2">
          <RefreshCw className="h-4 w-4" />
          Refresh
        </Button>
      </div>

      {/* Error State */}
      {error && (
        <Card className="border-red-500/50 bg-red-500/10">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <AlertCircle className="h-5 w-5 text-red-400 shrink-0" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-red-400">Failed to load services</p>
                <p className="text-xs text-red-400/70 mt-0.5">
                  {error instanceof Error ? error.message : 'An unexpected error occurred'}
                </p>
              </div>
              <Button variant="outline" size="sm" onClick={() => refetch()} className="shrink-0">
                <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
                Retry
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Loading State */}
      {isLoading && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[...Array(6)].map((_, i) => (
            <Card key={i}>
              <CardHeader className="pb-3">
                <Skeleton className="h-5 w-32" />
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-3/4" />
                  <Skeleton className="h-4 w-1/2" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Empty State */}
      {!isLoading && !error && deviceGroups.length === 0 && (
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center text-center">
              <Layers className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold">No services discovered</h3>
              <p className="text-sm text-muted-foreground mt-2 max-w-md">
                Services are auto-discovered from Scout agents running on your devices.
                Deploy agents to start mapping your infrastructure.
              </p>
              <Button variant="outline" size="sm" asChild className="mt-4 gap-2">
                <Link to="/agents">
                  <Bot className="h-4 w-4" />
                  Manage Agents
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Device Cards Grid */}
      {!isLoading && !error && deviceGroups.length > 0 && (
        <>
          <div className="flex items-center gap-4 text-sm text-muted-foreground">
            <span>{deviceGroups.length} device{deviceGroups.length !== 1 ? 's' : ''}</span>
            <span className="text-muted-foreground/50">|</span>
            <span>{services?.length ?? 0} service{(services?.length ?? 0) !== 1 ? 's' : ''}</span>
          </div>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {deviceGroups.map((group) => (
              <DeviceServiceCard key={group.deviceId} group={group} />
            ))}
          </div>
        </>
      )}
    </div>
  )
}

function DeviceServiceCard({ group }: { group: DeviceGroup }) {
  const [expanded, setExpanded] = useState(false)
  const { deviceId, services } = group

  const runningCount = services.filter((s) => s.status === 'running').length
  const totalCpu = services.reduce((sum, s) => sum + s.cpu_percent, 0)

  // Sort: running first, then by name
  const sorted = [...services].sort((a, b) => {
    if (a.status === 'running' && b.status !== 'running') return -1
    if (a.status !== 'running' && b.status === 'running') return 1
    return a.name.localeCompare(b.name)
  })

  const displayServices = expanded ? sorted : sorted.slice(0, 3)

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">
            <Link
              to={`/devices/${deviceId}`}
              className="hover:text-primary transition-colors"
            >
              {deviceId.substring(0, 8)}...
            </Link>
          </CardTitle>
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">
              {runningCount}/{services.length} running
            </span>
            {totalCpu > 0 && (
              <span className="text-xs text-muted-foreground">
                {totalCpu.toFixed(1)}% CPU
              </span>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {displayServices.map((svc) => (
            <ServiceRow key={svc.id} service={svc} />
          ))}
          {services.length > 3 && (
            <button
              onClick={() => setExpanded(!expanded)}
              className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors w-full justify-center pt-1"
            >
              {expanded ? (
                <>
                  <ChevronDown className="h-3 w-3" />
                  Show less
                </>
              ) : (
                <>
                  <ChevronRight className="h-3 w-3" />
                  Show {services.length - 3} more
                </>
              )}
            </button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function ServiceRow({ service }: { service: Service }) {
  const TypeIcon = serviceTypeIcons[service.service_type] || AppWindow
  const typeLabel = serviceTypeLabels[service.service_type] || 'Unknown'
  const status = statusConfig[service.status] || statusConfig.unknown

  return (
    <div className="flex items-center gap-2 py-1">
      <TypeIcon className="h-3.5 w-3.5 text-muted-foreground shrink-0" title={typeLabel} />
      <div className="flex-1 min-w-0">
        <p className="text-sm truncate">{service.display_name || service.name}</p>
      </div>
      <div className="flex items-center gap-2 shrink-0">
        {service.cpu_percent > 0 && (
          <span className="text-xs text-muted-foreground">{service.cpu_percent.toFixed(1)}%</span>
        )}
        {service.memory_bytes > 0 && (
          <span className="text-xs text-muted-foreground">{formatBytes(service.memory_bytes)}</span>
        )}
        <div className="flex items-center gap-1">
          <span className={cn('h-1.5 w-1.5 rounded-full', status.bg)} />
          <span className={cn('text-xs', status.text)}>{status.label}</span>
        </div>
      </div>
    </div>
  )
}
