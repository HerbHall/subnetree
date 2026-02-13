import { useState, useMemo } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  ChevronLeft,
  Cpu,
  Package,
  Activity,
  AlertCircle,
  RefreshCw,
  Monitor,
  ChevronDown,
  ChevronRight,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { getAgent, getAgentHardware, getAgentSoftware, getAgentServices } from '@/api/agents'
import type { AgentStatus, HardwareProfile, SoftwareInventory, ServiceInfo } from '@/api/types'
import { cn } from '@/lib/utils'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const val = bytes / Math.pow(1024, i)
  return `${val.toFixed(i > 1 ? 1 : 0)} ${units[i]}`
}

function formatRelativeTime(isoString: string): string {
  const now = Date.now()
  const then = new Date(isoString).getTime()
  const diffSeconds = Math.floor((now - then) / 1000)

  if (diffSeconds < 60) return 'just now'
  const diffMinutes = Math.floor(diffSeconds / 60)
  if (diffMinutes < 60) return `${diffMinutes}m ago`
  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 30) return `${diffDays}d ago`
  const diffMonths = Math.floor(diffDays / 30)
  return `${diffMonths}mo ago`
}

function formatTimestamp(isoString: string): string {
  return new Date(isoString).toLocaleString()
}

function formatSpeed(mbps: number): string {
  if (mbps >= 1000) return `${(mbps / 1000).toFixed(0)} Gbps`
  return `${mbps} Mbps`
}

export function AgentDetailPage() {
  const { id } = useParams<{ id: string }>()

  const {
    data: agent,
    isLoading: agentLoading,
    error: agentError,
    refetch: refetchAgent,
  } = useQuery({
    queryKey: ['agent', id],
    queryFn: () => getAgent(id!),
    enabled: !!id,
  })

  const { data: hardware } = useQuery({
    queryKey: ['agent', id, 'hardware'],
    queryFn: () => getAgentHardware(id!),
    enabled: !!id,
    retry: false,
  })

  const { data: software } = useQuery({
    queryKey: ['agent', id, 'software'],
    queryFn: () => getAgentSoftware(id!),
    enabled: !!id,
    retry: false,
  })

  const { data: services } = useQuery({
    queryKey: ['agent', id, 'services'],
    queryFn: () => getAgentServices(id!),
    enabled: !!id,
    retry: false,
  })

  if (agentLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <Skeleton className="h-10 w-10 rounded" />
          <div>
            <Skeleton className="h-7 w-48 mb-2" />
            <Skeleton className="h-4 w-32" />
          </div>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-32 rounded-lg" />
          ))}
        </div>
        <Skeleton className="h-48 rounded-lg" />
        <Skeleton className="h-48 rounded-lg" />
      </div>
    )
  }

  if (agentError || !agent) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" asChild>
            <Link to="/agents">
              <ChevronLeft className="h-4 w-4" />
            </Link>
          </Button>
          <h1 className="text-2xl font-semibold">Agent Detail</h1>
        </div>
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <AlertCircle className="h-12 w-12 text-red-400 mb-4" />
          <h3 className="text-lg font-medium">Failed to load agent</h3>
          <p className="text-sm text-muted-foreground mt-1 max-w-sm">
            {agentError instanceof Error ? agentError.message : 'Agent not found or an unexpected error occurred.'}
          </p>
          <Button variant="outline" className="mt-4 gap-2" onClick={() => refetchAgent()}>
            <RefreshCw className="h-4 w-4" />
            Try Again
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/agents">
            <ChevronLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold">{agent.hostname || 'Unknown Agent'}</h1>
            <AgentStatusBadge status={agent.status} />
            <span className="text-xs bg-muted px-2 py-0.5 rounded capitalize">
              {agent.platform}
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            Agent ID: <span className="font-mono text-xs">{agent.id}</span>
          </p>
        </div>
      </div>

      {/* Metadata grid */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Monitor className="h-4 w-4 text-muted-foreground" />
            Agent Info
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <InfoRow label="Agent Version" value={agent.agent_version || '-'} mono />
            <InfoRow label="Proto Version" value={String(agent.proto_version)} />
            <InfoRow
              label="Last Check-in"
              value={agent.last_check_in ? formatRelativeTime(agent.last_check_in) : 'Never'}
            />
            <InfoRow label="Enrolled" value={formatTimestamp(agent.enrolled_at)} />
            {agent.device_id && (
              <div>
                <p className="text-xs text-muted-foreground mb-1">Linked Device</p>
                <Link
                  to={`/devices/${agent.device_id}`}
                  className="text-sm text-primary hover:underline font-mono"
                >
                  {agent.device_id.slice(0, 8)}...
                </Link>
              </div>
            )}
            {agent.cert_serial && (
              <InfoRow label="Cert Serial" value={agent.cert_serial} mono />
            )}
            {agent.cert_expires_at && (
              <InfoRow label="Cert Expires" value={formatTimestamp(agent.cert_expires_at)} />
            )}
          </div>
        </CardContent>
      </Card>

      {/* Hardware card */}
      <HardwareCard hardware={hardware} />

      {/* Software card */}
      <SoftwareCard software={software} />

      {/* Services card */}
      <ServicesCard services={services} />
    </div>
  )
}

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground mb-1">{label}</p>
      <p className={cn('text-sm', mono && 'font-mono')}>{value}</p>
    </div>
  )
}

function AgentStatusBadge({ status }: { status: AgentStatus }) {
  const config = {
    connected: { bg: 'bg-green-500/10', text: 'text-green-500', dot: 'bg-green-500' },
    pending: { bg: 'bg-amber-500/10', text: 'text-amber-500', dot: 'bg-amber-500' },
    disconnected: { bg: 'bg-red-500/10', text: 'text-red-500', dot: 'bg-red-500' },
  }
  const c = config[status] ?? config.disconnected

  return (
    <span className={cn('inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs', c.bg, c.text)}>
      <span className={cn('h-1.5 w-1.5 rounded-full', c.dot)} />
      <span className="capitalize">{status}</span>
    </span>
  )
}

function AwaitingReport() {
  return (
    <p className="text-sm text-muted-foreground italic py-4 text-center">
      Awaiting first report from agent
    </p>
  )
}

function HardwareCard({ hardware }: { hardware?: HardwareProfile }) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Cpu className="h-4 w-4 text-muted-foreground" />
          Hardware
        </CardTitle>
      </CardHeader>
      <CardContent>
        {!hardware ? (
          <AwaitingReport />
        ) : (
          <div className="space-y-5">
            {/* CPU + RAM */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <InfoRow label="CPU" value={hardware.cpu_model || '-'} />
              <InfoRow label="Cores / Threads" value={`${hardware.cpu_cores} / ${hardware.cpu_threads}`} />
              <InfoRow label="RAM" value={formatBytes(hardware.ram_bytes)} />
              {hardware.system_manufacturer && (
                <InfoRow label="System" value={`${hardware.system_manufacturer} ${hardware.system_model || ''}`.trim()} />
              )}
            </div>

            {/* Disks */}
            {hardware.disks?.length > 0 && (
              <div>
                <p className="text-xs text-muted-foreground mb-2">Disks ({hardware.disks.length})</p>
                <div className="rounded border overflow-hidden">
                  <table className="w-full text-sm">
                    <thead className="bg-muted/50">
                      <tr>
                        <th className="px-3 py-2 text-left font-medium text-xs">Name</th>
                        <th className="px-3 py-2 text-left font-medium text-xs">Size</th>
                        <th className="px-3 py-2 text-left font-medium text-xs">Type</th>
                        <th className="px-3 py-2 text-left font-medium text-xs">Model</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y">
                      {hardware.disks.map((d, i) => (
                        <tr key={i}>
                          <td className="px-3 py-1.5">{d.name}</td>
                          <td className="px-3 py-1.5 font-mono text-xs">{formatBytes(d.size_bytes)}</td>
                          <td className="px-3 py-1.5">
                            <span className="text-xs bg-muted px-1.5 py-0.5 rounded">{d.disk_type || '-'}</span>
                          </td>
                          <td className="px-3 py-1.5 text-muted-foreground">{d.model || '-'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* GPUs */}
            {hardware.gpus?.length > 0 && (
              <div>
                <p className="text-xs text-muted-foreground mb-2">GPUs ({hardware.gpus.length})</p>
                <div className="space-y-1">
                  {hardware.gpus.map((g, i) => (
                    <div key={i} className="flex items-center gap-3 text-sm">
                      <span>{g.model}</span>
                      {g.vram_bytes > 0 && (
                        <span className="text-xs text-muted-foreground">{formatBytes(g.vram_bytes)} VRAM</span>
                      )}
                      {g.driver_version && (
                        <span className="text-xs text-muted-foreground">Driver: {g.driver_version}</span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* NICs */}
            {hardware.nics?.length > 0 && (
              <div>
                <p className="text-xs text-muted-foreground mb-2">Network Interfaces ({hardware.nics.length})</p>
                <div className="rounded border overflow-hidden">
                  <table className="w-full text-sm">
                    <thead className="bg-muted/50">
                      <tr>
                        <th className="px-3 py-2 text-left font-medium text-xs">Name</th>
                        <th className="px-3 py-2 text-left font-medium text-xs">MAC Address</th>
                        <th className="px-3 py-2 text-left font-medium text-xs">Speed</th>
                        <th className="px-3 py-2 text-left font-medium text-xs">Type</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y">
                      {hardware.nics.map((n, i) => (
                        <tr key={i}>
                          <td className="px-3 py-1.5">{n.name}</td>
                          <td className="px-3 py-1.5 font-mono text-xs">{n.mac_address || '-'}</td>
                          <td className="px-3 py-1.5">{n.speed_mbps > 0 ? formatSpeed(n.speed_mbps) : '-'}</td>
                          <td className="px-3 py-1.5 text-muted-foreground capitalize">{n.nic_type || '-'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function SoftwareCard({ software }: { software?: SoftwareInventory }) {
  const [packagesExpanded, setPackagesExpanded] = useState(false)
  const packageCount = software?.packages?.length ?? 0
  const defaultExpanded = packageCount <= 20

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Package className="h-4 w-4 text-muted-foreground" />
          Software
        </CardTitle>
      </CardHeader>
      <CardContent>
        {!software ? (
          <AwaitingReport />
        ) : (
          <div className="space-y-5">
            {/* OS Info */}
            <div className="grid gap-4 sm:grid-cols-3">
              <InfoRow label="OS" value={software.os_name || '-'} />
              <InfoRow label="Version" value={software.os_version || '-'} />
              {software.os_build && <InfoRow label="Build" value={software.os_build} mono />}
            </div>

            {/* Installed Packages */}
            {packageCount > 0 && (
              <div>
                <button
                  onClick={() => setPackagesExpanded(!packagesExpanded)}
                  className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors mb-2"
                >
                  {(defaultExpanded || packagesExpanded) ? (
                    <ChevronDown className="h-3.5 w-3.5" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5" />
                  )}
                  Installed Packages ({packageCount})
                </button>
                {(defaultExpanded || packagesExpanded) && (
                  <div className="rounded border overflow-hidden">
                    <table className="w-full text-sm">
                      <thead className="bg-muted/50">
                        <tr>
                          <th className="px-3 py-2 text-left font-medium text-xs">Name</th>
                          <th className="px-3 py-2 text-left font-medium text-xs">Version</th>
                          <th className="px-3 py-2 text-left font-medium text-xs">Publisher</th>
                          <th className="px-3 py-2 text-left font-medium text-xs">Installed</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y">
                        {software.packages.map((p, i) => (
                          <tr key={i}>
                            <td className="px-3 py-1.5">{p.name}</td>
                            <td className="px-3 py-1.5 font-mono text-xs">{p.version || '-'}</td>
                            <td className="px-3 py-1.5 text-muted-foreground">{p.publisher || '-'}</td>
                            <td className="px-3 py-1.5 text-muted-foreground">{p.install_date || '-'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}

            {/* Docker Containers */}
            {software.docker_containers?.length > 0 && (
              <div>
                <p className="text-xs text-muted-foreground mb-2">
                  Docker Containers ({software.docker_containers.length})
                </p>
                <div className="rounded border overflow-hidden">
                  <table className="w-full text-sm">
                    <thead className="bg-muted/50">
                      <tr>
                        <th className="px-3 py-2 text-left font-medium text-xs">Name</th>
                        <th className="px-3 py-2 text-left font-medium text-xs">Image</th>
                        <th className="px-3 py-2 text-left font-medium text-xs">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y">
                      {software.docker_containers.map((c, i) => (
                        <tr key={i}>
                          <td className="px-3 py-1.5">{c.name}</td>
                          <td className="px-3 py-1.5 font-mono text-xs">{c.image}</td>
                          <td className="px-3 py-1.5">
                            <ContainerStatusBadge status={c.status} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ContainerStatusBadge({ status }: { status: string }) {
  const lower = status.toLowerCase()
  const isRunning = lower.startsWith('running') || lower === 'up'
  const isCreated = lower === 'created'

  const bg = isRunning ? 'bg-green-500/10' : isCreated ? 'bg-amber-500/10' : 'bg-gray-500/10'
  const text = isRunning ? 'text-green-500' : isCreated ? 'text-amber-500' : 'text-gray-500'

  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs', bg, text)}>
      {status}
    </span>
  )
}

function ServicesCard({ services }: { services?: ServiceInfo[] }) {
  const sortedServices = useMemo(() => {
    if (!services) return []
    return [...services].sort((a, b) => {
      const statusOrder: Record<string, number> = { running: 0, stopped: 1, failed: 2 }
      const aOrder = statusOrder[a.status] ?? 3
      const bOrder = statusOrder[b.status] ?? 3
      if (aOrder !== bOrder) return aOrder - bOrder
      return a.name.localeCompare(b.name)
    })
  }, [services])

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Activity className="h-4 w-4 text-muted-foreground" />
          Services{services ? ` (${services.length})` : ''}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {!services ? (
          <AwaitingReport />
        ) : services.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">No services reported</p>
        ) : (
          <div className="rounded border overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted/50">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium text-xs">Name</th>
                    <th className="px-3 py-2 text-left font-medium text-xs">Display Name</th>
                    <th className="px-3 py-2 text-left font-medium text-xs">Status</th>
                    <th className="px-3 py-2 text-left font-medium text-xs">Start Type</th>
                    <th className="px-3 py-2 text-right font-medium text-xs">CPU%</th>
                    <th className="px-3 py-2 text-right font-medium text-xs">Memory</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {sortedServices.map((s, i) => (
                    <tr key={i}>
                      <td className="px-3 py-1.5 font-mono text-xs">{s.name}</td>
                      <td className="px-3 py-1.5">{s.display_name || '-'}</td>
                      <td className="px-3 py-1.5">
                        <ServiceStatusBadge status={s.status} />
                      </td>
                      <td className="px-3 py-1.5">
                        <StartTypeBadge startType={s.start_type} />
                      </td>
                      <td className="px-3 py-1.5 text-right font-mono text-xs">
                        {s.cpu_percent > 0 ? s.cpu_percent.toFixed(1) : '-'}
                      </td>
                      <td className="px-3 py-1.5 text-right font-mono text-xs">
                        {s.memory_bytes > 0 ? formatBytes(s.memory_bytes) : '-'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ServiceStatusBadge({ status }: { status: string }) {
  const config: Record<string, { bg: string; text: string }> = {
    running: { bg: 'bg-green-500/10', text: 'text-green-500' },
    stopped: { bg: 'bg-red-500/10', text: 'text-red-500' },
    failed: { bg: 'bg-red-500/10', text: 'text-red-500' },
  }
  const c = config[status] ?? { bg: 'bg-gray-500/10', text: 'text-gray-500' }

  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs capitalize', c.bg, c.text)}>
      {status}
    </span>
  )
}

function StartTypeBadge({ startType }: { startType: string }) {
  const config: Record<string, { bg: string; text: string }> = {
    auto: { bg: 'bg-blue-500/10', text: 'text-blue-500' },
    manual: { bg: 'bg-gray-500/10', text: 'text-gray-500' },
    disabled: { bg: 'bg-gray-400/10', text: 'text-gray-400' },
  }
  const c = config[startType] ?? { bg: 'bg-gray-500/10', text: 'text-gray-500' }

  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs capitalize', c.bg, c.text)}>
      {startType}
    </span>
  )
}
