import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Server, Container, Loader2 } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { getProxmoxVMs, type ProxmoxResource } from '@/api/proxmox'
import { cn } from '@/lib/utils'

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  if (days > 0) return `${days}d ${hours}h`
  const mins = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${mins}m`
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`
  return `${(bytes / 1073741824).toFixed(1)} GB`
}

function StatusBadge({ status }: { status: string }) {
  const isOnline = status === 'online'
  return (
    <span
      className={cn(
        'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
        isOnline
          ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
          : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400',
      )}
    >
      {isOnline ? 'Online' : 'Offline'}
    </span>
  )
}

function TypeBadge({ deviceType }: { deviceType: string }) {
  const isVM = deviceType === 'virtual_machine'
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium',
        isVM
          ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
          : 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
      )}
    >
      {isVM ? <Server className="h-3 w-3" /> : <Container className="h-3 w-3" />}
      {isVM ? 'VM' : 'LXC'}
    </span>
  )
}

function MemoryBar({ used, total }: { used: number; total: number }) {
  const pct = total > 0 ? Math.round((used / total) * 100) : 0
  return (
    <div className="flex items-center gap-2 min-w-[140px]">
      <Progress value={pct} className="h-2 flex-1" />
      <span className="text-xs text-muted-foreground whitespace-nowrap">
        {used} / {total} MB
      </span>
    </div>
  )
}

function ResourceRow({ r }: { r: ProxmoxResource }) {
  return (
    <tr className="border-b last:border-0 hover:bg-muted/50">
      <td className="py-2 px-3">
        <Link
          to={`/devices/${r.device_id}`}
          className="text-sm font-medium text-primary hover:underline"
        >
          {r.device_name || r.device_id}
        </Link>
      </td>
      <td className="py-2 px-3">
        <TypeBadge deviceType={r.device_type} />
      </td>
      <td className="py-2 px-3">
        <StatusBadge status={r.status} />
      </td>
      <td className="py-2 px-3 text-sm text-right">
        {r.status === 'online' ? `${r.cpu_percent.toFixed(1)}%` : '-'}
      </td>
      <td className="py-2 px-3">
        {r.status === 'online' ? (
          <MemoryBar used={r.mem_used_mb} total={r.mem_total_mb} />
        ) : (
          <span className="text-sm text-muted-foreground">-</span>
        )}
      </td>
      <td className="py-2 px-3 text-sm text-right">
        {r.status === 'online'
          ? `${r.disk_used_gb} / ${r.disk_total_gb} GB`
          : '-'}
      </td>
      <td className="py-2 px-3 text-sm text-right">
        {r.status === 'online' ? formatUptime(r.uptime_sec) : '-'}
      </td>
      <td className="py-2 px-3 text-sm text-right">
        {r.status === 'online'
          ? `${formatBytes(r.netin_bytes)} / ${formatBytes(r.netout_bytes)}`
          : '-'}
      </td>
    </tr>
  )
}

export function ProxmoxResources({ deviceId }: { deviceId: string }) {
  const { data: resources, isLoading, error } = useQuery({
    queryKey: ['proxmox-resources', deviceId],
    queryFn: () => getProxmoxVMs(deviceId),
  })

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Server className="h-4 w-4 text-muted-foreground" />
          Virtualization
        </CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground py-4">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading VMs and containers...
          </div>
        )}

        {error != null && (
          <p className="text-sm text-destructive py-2">
            Failed to load virtualization data.
          </p>
        )}

        {!isLoading && !error && (!resources || resources.length === 0) && (
          <p className="text-sm text-muted-foreground py-2">
            No VMs or containers found for this host.
          </p>
        )}

        {resources && resources.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-muted-foreground text-xs">
                  <th className="py-2 px-3 text-left font-medium">Name</th>
                  <th className="py-2 px-3 text-left font-medium">Type</th>
                  <th className="py-2 px-3 text-left font-medium">Status</th>
                  <th className="py-2 px-3 text-right font-medium">CPU</th>
                  <th className="py-2 px-3 text-left font-medium">Memory</th>
                  <th className="py-2 px-3 text-right font-medium">Disk</th>
                  <th className="py-2 px-3 text-right font-medium">Uptime</th>
                  <th className="py-2 px-3 text-right font-medium">Net I/O</th>
                </tr>
              </thead>
              <tbody>
                {resources.map((r) => (
                  <ResourceRow key={r.device_id} r={r} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
