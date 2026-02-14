import { useState, useEffect, useCallback, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation } from '@tanstack/react-query'
import {
  Monitor,
  Wifi,
  WifiOff,
  AlertTriangle,
  AlertCircle,
  Radar,
  RefreshCw,
  ArrowRight,
  Server,
  Router,
  HardDrive,
  Activity,
  Clock,
  CheckCircle2,
  XCircle,
  Loader2,
  Pause,
  Rocket,
  Bot,
  BarChart3,
  Layers,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { DeviceCardCompact } from '@/components/device-card'
import { ScanProgressPanel } from '@/components/scan-progress-panel'
import { getTopology, triggerScan, listScans } from '@/api/devices'
import { listAgents } from '@/api/agents'
import { listAlerts } from '@/api/pulse'
import { getFleetSummary } from '@/api/services'
import { useScanProgress } from '@/hooks/use-scan-progress'
import type { Scan, ScanStatus } from '@/api/types'
import { cn } from '@/lib/utils'
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts'

const REFRESH_OPTIONS = [
  { label: '15s', value: 15 * 1000 },
  { label: '30s', value: 30 * 1000 },
  { label: '1m', value: 60 * 1000 },
  { label: '5m', value: 5 * 60 * 1000 },
]

export function DashboardPage() {
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [refreshInterval, setRefreshInterval] = useState(30 * 1000) // Default 30s
  const intervalSeconds = Math.floor(refreshInterval / 1000)
  const [countdown, setCountdown] = useState(intervalSeconds)
  const { activeScan: wsScan, progress: scanProgress } = useScanProgress()

  // Reset countdown during render when dependencies change (React-recommended
  // "adjust state during render" pattern avoids setState-in-effect).
  const [prevAutoRefresh, setPrevAutoRefresh] = useState(autoRefresh)
  const [prevInterval, setPrevInterval] = useState(refreshInterval)
  if (autoRefresh !== prevAutoRefresh || refreshInterval !== prevInterval) {
    setPrevAutoRefresh(autoRefresh)
    setPrevInterval(refreshInterval)
    setCountdown(intervalSeconds)
  }

  // Countdown timer effect -- only the interval callback sets state.
  useEffect(() => {
    if (!autoRefresh) return

    const timer = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          return intervalSeconds
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [autoRefresh, intervalSeconds])

  // Fetch topology data with auto-refresh
  const {
    data: topology,
    isLoading: topologyLoading,
    error: topologyError,
    refetch: refetchTopology,
  } = useQuery({
    queryKey: ['topology'],
    queryFn: getTopology,
    refetchInterval: autoRefresh ? refreshInterval : false,
  })

  // Fetch recent scans with auto-refresh
  const {
    data: recentScans,
    isLoading: scansLoading,
  } = useQuery({
    queryKey: ['scans', 'recent'],
    queryFn: () => listScans(5, 0),
    refetchInterval: autoRefresh ? refreshInterval : false,
  })

  // Fetch agents with auto-refresh
  const {
    data: agents,
    isLoading: agentsLoading,
  } = useQuery({
    queryKey: ['agents'],
    queryFn: listAgents,
    refetchInterval: autoRefresh ? refreshInterval : false,
  })

  // Fetch fleet utilization with auto-refresh
  const {
    data: fleetSummary,
    isLoading: fleetLoading,
  } = useQuery({
    queryKey: ['fleet-summary'],
    queryFn: getFleetSummary,
    refetchInterval: autoRefresh ? refreshInterval : false,
  })

  // Fetch active alerts with auto-refresh
  const {
    data: activeAlerts,
    isLoading: alertsLoading,
  } = useQuery({
    queryKey: ['active-alerts'],
    queryFn: () => listAlerts({ active: true, limit: 5 }),
    refetchInterval: autoRefresh ? refreshInterval : false,
  })

  // Scan mutation
  const scanMutation = useMutation({
    mutationFn: () => triggerScan(),
  })

  // Keyboard shortcuts
  const handleRefresh = useCallback(() => { refetchTopology() }, [refetchTopology])
  const dashboardShortcuts = useMemo(
    () => [{ key: 'r', handler: handleRefresh, description: 'Refresh data' }],
    [handleRefresh]
  )
  useKeyboardShortcuts(dashboardShortcuts)

  const devices = topology?.nodes || []
  const loading = topologyLoading

  // Calculate stats
  const stats = {
    total: devices.length,
    online: devices.filter((d) => d.status === 'online').length,
    offline: devices.filter((d) => d.status === 'offline').length,
    degraded: devices.filter((d) => d.status === 'degraded').length,
  }

  // Count by type
  const typeCounts = devices.reduce(
    (acc, d) => {
      acc[d.device_type] = (acc[d.device_type] || 0) + 1
      return acc
    },
    {} as Record<string, number>
  )

  // Agent stats
  const agentStats = {
    total: agents?.length ?? 0,
    connected: agents?.filter((a) => a.status === 'connected').length ?? 0,
    disconnected: agents?.filter((a) => a.status === 'disconnected').length ?? 0,
    pending: agents?.filter((a) => a.status === 'pending').length ?? 0,
  }

  // Recent devices (first 5)
  const recentDevices = devices.slice(0, 5)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Dashboard</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Network overview and quick actions
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* Auto-refresh controls */}
          <div className="flex items-center gap-1 rounded-md border bg-muted/30 p-1">
            <Button
              variant={autoRefresh ? 'secondary' : 'ghost'}
              size="sm"
              onClick={() => setAutoRefresh(!autoRefresh)}
              className="h-7 px-2 gap-1"
            >
              {autoRefresh ? (
                <RefreshCw className="h-3.5 w-3.5" />
              ) : (
                <Pause className="h-3.5 w-3.5" />
              )}
              <span className="text-xs">
                {autoRefresh ? `${countdown}s` : 'Paused'}
              </span>
            </Button>
            {autoRefresh && (
              <div className="flex">
                {REFRESH_OPTIONS.map((opt) => (
                  <Button
                    key={opt.value}
                    variant={refreshInterval === opt.value ? 'secondary' : 'ghost'}
                    size="sm"
                    onClick={() => setRefreshInterval(opt.value)}
                    className="h-7 px-2 text-xs"
                  >
                    {opt.label}
                  </Button>
                ))}
              </div>
            )}
          </div>
          <Button
            onClick={() => scanMutation.mutate()}
            disabled={scanMutation.isPending || !!wsScan}
            className="gap-2"
          >
            {scanMutation.isPending || wsScan ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Radar className="h-4 w-4" />
            )}
            {wsScan ? 'Scanning...' : 'Scan Network'}
          </Button>
        </div>
      </div>

      {/* Scan Progress Indicator */}
      <ScanProgressPanel activeScan={wsScan} progress={scanProgress} />

      {/* Error State */}
      {topologyError && (
        <Card className="border-red-500/50 bg-red-500/10">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <AlertCircle className="h-5 w-5 text-red-400 shrink-0" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-red-400">Failed to load network data</p>
                <p className="text-xs text-red-400/70 mt-0.5">
                  {topologyError instanceof Error ? topologyError.message : 'An unexpected error occurred'}
                </p>
              </div>
              <Button variant="outline" size="sm" onClick={() => refetchTopology()} className="shrink-0">
                <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
                Retry
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Welcome state when no devices and not loading */}
      {!loading && !topologyError && stats.total === 0 && (
        <Card>
          <CardContent className="py-12">
            <div className="flex flex-col items-center text-center">
              <Rocket className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold">Welcome to SubNetree</h3>
              <p className="text-sm text-muted-foreground mt-2 max-w-md">
                Get started by scanning your network to discover devices, or add a device manually.
              </p>
              <ol className="text-sm text-muted-foreground mt-4 space-y-2 text-left">
                <li className="flex items-start gap-2">
                  <span className="inline-flex items-center justify-center h-5 w-5 rounded-full bg-primary text-primary-foreground text-xs font-medium shrink-0 mt-0.5">1</span>
                  <span>Click <strong>Scan Network</strong> above to discover devices on your network</span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="inline-flex items-center justify-center h-5 w-5 rounded-full bg-primary text-primary-foreground text-xs font-medium shrink-0 mt-0.5">2</span>
                  <span>Review discovered devices in the <Link to="/devices" className="text-primary hover:underline">Devices</Link> page</span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="inline-flex items-center justify-center h-5 w-5 rounded-full bg-primary text-primary-foreground text-xs font-medium shrink-0 mt-0.5">3</span>
                  <span>Explore your <Link to="/topology" className="text-primary hover:underline">Network Map</Link> to see how devices connect</span>
                </li>
              </ol>
              <Button
                className="mt-6 gap-2"
                onClick={() => scanMutation.mutate()}
                disabled={scanMutation.isPending || !!wsScan}
              >
                {scanMutation.isPending || wsScan ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <Radar className="h-4 w-4" />
                )}
                {wsScan ? 'Scanning...' : 'Scan Network'}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Stats Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Devices"
          value={stats.total}
          icon={Monitor}
          loading={loading}
          href="/devices"
        />
        <StatCard
          title="Online"
          value={stats.online}
          icon={Wifi}
          variant="success"
          loading={loading}
          href="/devices?status=online"
        />
        <StatCard
          title="Offline"
          value={stats.offline}
          icon={WifiOff}
          variant="danger"
          loading={loading}
          href="/devices?status=offline"
        />
        <StatCard
          title="Degraded"
          value={stats.degraded}
          icon={AlertTriangle}
          variant="warning"
          loading={loading}
          href="/devices?status=degraded"
        />
      </div>

      {/* Scout Agents Widget */}
      <Card>
        <CardHeader className="pb-3 flex flex-row items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Bot className="h-4 w-4 text-muted-foreground" />
            Scout Agents
          </CardTitle>
          <Button variant="ghost" size="sm" asChild className="gap-1 text-xs">
            <Link to="/agents">
              View All
              <ArrowRight className="h-3 w-3" />
            </Link>
          </Button>
        </CardHeader>
        <CardContent>
          {agentsLoading ? (
            <div className="grid grid-cols-4 gap-4">
              {[...Array(4)].map((_, i) => (
                <div key={i} className="text-center">
                  <Skeleton className="h-8 w-12 mx-auto" />
                  <Skeleton className="h-3 w-16 mx-auto mt-1" />
                </div>
              ))}
            </div>
          ) : agentStats.total === 0 ? (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                No agents enrolled. Deploy Scout agents to monitor devices.
              </p>
              <Button variant="outline" size="sm" asChild className="gap-2 shrink-0">
                <Link to="/settings">
                  Enroll Agent
                </Link>
              </Button>
            </div>
          ) : (
            <div className="grid grid-cols-4 gap-4">
              <div className="text-center">
                <p className="text-2xl font-bold">{agentStats.total}</p>
                <p className="text-xs text-muted-foreground">Total</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold text-green-600 dark:text-green-400">{agentStats.connected}</p>
                <p className="text-xs text-muted-foreground">Connected</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold text-red-600 dark:text-red-400">{agentStats.disconnected}</p>
                <p className="text-xs text-muted-foreground">Disconnected</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold text-amber-600 dark:text-amber-400">{agentStats.pending}</p>
                <p className="text-xs text-muted-foreground">Pending</p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Active Alerts Widget */}
      <Card>
        <CardHeader className="pb-3 flex flex-row items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
            Active Alerts
          </CardTitle>
          <Button variant="ghost" size="sm" asChild className="gap-1 text-xs">
            <Link to="/monitoring">
              View All
              <ArrowRight className="h-3 w-3" />
            </Link>
          </Button>
        </CardHeader>
        <CardContent>
          {alertsLoading ? (
            <div className="space-y-2">
              {[...Array(3)].map((_, i) => (
                <Skeleton key={i} className="h-10" />
              ))}
            </div>
          ) : !activeAlerts || activeAlerts.length === 0 ? (
            <div className="flex items-center gap-3 text-sm text-muted-foreground">
              <CheckCircle2 className="h-5 w-5 text-green-500" />
              <span>All clear -- no active alerts</span>
            </div>
          ) : (
            <div className="space-y-2">
              {activeAlerts.map((alert) => (
                <div key={alert.id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-muted/50">
                  <AlertCircle className={cn('h-4 w-4 shrink-0',
                    alert.severity === 'critical' ? 'text-red-500' :
                    alert.severity === 'warning' ? 'text-amber-500' : 'text-blue-500'
                  )} />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{alert.message}</p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(alert.triggered_at).toLocaleString()}
                    </p>
                  </div>
                  <span className={cn('text-xs px-1.5 py-0.5 rounded',
                    alert.severity === 'critical' ? 'bg-red-500/10 text-red-600 dark:text-red-400' :
                    alert.severity === 'warning' ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400' :
                    'bg-blue-500/10 text-blue-600 dark:text-blue-400'
                  )}>
                    {alert.severity}
                  </span>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Fleet Utilization Widget */}
      <Card>
        <CardHeader className="pb-3 flex flex-row items-center justify-between">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <BarChart3 className="h-4 w-4 text-muted-foreground" />
            Fleet Utilization
          </CardTitle>
          <Button variant="ghost" size="sm" asChild className="gap-1 text-xs">
            <Link to="/services">
              View All
              <ArrowRight className="h-3 w-3" />
            </Link>
          </Button>
        </CardHeader>
        <CardContent>
          {fleetLoading ? (
            <div className="grid grid-cols-4 gap-4">
              {[...Array(4)].map((_, i) => (
                <div key={i} className="text-center">
                  <Skeleton className="h-8 w-12 mx-auto" />
                  <Skeleton className="h-3 w-16 mx-auto mt-1" />
                </div>
              ))}
            </div>
          ) : !fleetSummary || fleetSummary.total_services === 0 ? (
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                No service data -- services are auto-discovered from Scout agents
              </p>
              <Button variant="outline" size="sm" asChild className="gap-2 shrink-0">
                <Link to="/agents">
                  <Layers className="h-3.5 w-3.5" />
                  Manage Agents
                </Link>
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="grid grid-cols-4 gap-4">
                <div className="text-center">
                  <p className="text-2xl font-bold">{fleetSummary.total_services}</p>
                  <p className="text-xs text-muted-foreground">Services</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold">{fleetSummary.avg_cpu.toFixed(1)}%</p>
                  <p className="text-xs text-muted-foreground">Avg CPU</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold">{fleetSummary.avg_memory.toFixed(1)}%</p>
                  <p className="text-xs text-muted-foreground">Avg Memory</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold">{fleetSummary.total_devices}</p>
                  <p className="text-xs text-muted-foreground">Devices</p>
                </div>
              </div>
              {/* Grade Distribution */}
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-xs text-muted-foreground">Grades:</span>
                {['A', 'B', 'C', 'D', 'F'].map((grade) => {
                  const count = fleetSummary.by_grade?.[grade] ?? 0
                  if (count === 0) return null
                  const colors: Record<string, string> = {
                    A: 'bg-green-500/10 text-green-600 dark:text-green-400',
                    B: 'bg-teal-500/10 text-teal-600 dark:text-teal-400',
                    C: 'bg-yellow-500/10 text-yellow-600 dark:text-yellow-400',
                    D: 'bg-orange-500/10 text-orange-600 dark:text-orange-400',
                    F: 'bg-red-500/10 text-red-600 dark:text-red-400',
                  }
                  return (
                    <span
                      key={grade}
                      className={cn('inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium', colors[grade])}
                    >
                      {grade}: {count}
                    </span>
                  )
                })}
              </div>
              {/* Underutilized / Overloaded counts */}
              <div className="flex items-center gap-4 text-xs">
                {(fleetSummary.underutilized?.length ?? 0) > 0 && (
                  <span className="text-teal-600 dark:text-teal-400">
                    {fleetSummary.underutilized?.length} underutilized
                  </span>
                )}
                {(fleetSummary.overloaded?.length ?? 0) > 0 && (
                  <span className="text-red-600 dark:text-red-400">
                    {fleetSummary.overloaded?.length} overloaded
                  </span>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Recent Scans */}
        <Card className="lg:col-span-1">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Recent Scans</CardTitle>
          </CardHeader>
          <CardContent>
            {scansLoading ? (
              <div className="space-y-3">
                {[...Array(4)].map((_, i) => (
                  <Skeleton key={i} className="h-12" />
                ))}
              </div>
            ) : !recentScans || recentScans.length === 0 ? (
              <div className="text-center py-6">
                <Clock className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
                <p className="text-sm text-muted-foreground">No scans yet</p>
                <Button
                  size="sm"
                  className="mt-3"
                  onClick={() => scanMutation.mutate()}
                  disabled={scanMutation.isPending}
                >
                  Run First Scan
                </Button>
              </div>
            ) : (
              <div className="space-y-2">
                {recentScans.map((scan) => (
                  <ScanHistoryRow key={scan.id} scan={scan} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Device Types Breakdown */}
        <Card className="lg:col-span-1">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Device Types</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="space-y-3">
                {[...Array(4)].map((_, i) => (
                  <div key={i} className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Skeleton className="h-4 w-4 rounded" />
                      <Skeleton className="h-4 w-24" />
                    </div>
                    <Skeleton className="h-4 w-8" />
                  </div>
                ))}
              </div>
            ) : devices.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-4">
                No devices discovered yet
              </p>
            ) : (
              <div className="space-y-3">
                <DeviceTypeRow
                  icon={Router}
                  label="Routers & Switches"
                  count={
                    (typeCounts.router || 0) +
                    (typeCounts.switch || 0) +
                    (typeCounts.access_point || 0)
                  }
                />
                <DeviceTypeRow
                  icon={Server}
                  label="Servers & NAS"
                  count={(typeCounts.server || 0) + (typeCounts.nas || 0)}
                />
                <DeviceTypeRow
                  icon={Monitor}
                  label="Desktops & Laptops"
                  count={(typeCounts.desktop || 0) + (typeCounts.laptop || 0)}
                />
                <DeviceTypeRow
                  icon={HardDrive}
                  label="IoT & Other"
                  count={
                    (typeCounts.iot || 0) +
                    (typeCounts.printer || 0) +
                    (typeCounts.camera || 0) +
                    (typeCounts.phone || 0) +
                    (typeCounts.tablet || 0) +
                    (typeCounts.unknown || 0)
                  }
                />
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent Devices */}
        <Card className="lg:col-span-1">
          <CardHeader className="pb-3 flex flex-row items-center justify-between">
            <CardTitle className="text-sm font-medium">Recent Devices</CardTitle>
            <Button variant="ghost" size="sm" asChild className="gap-1 text-xs">
              <Link to="/devices">
                View All
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="space-y-2">
                {[...Array(5)].map((_, i) => (
                  <div key={i} className="rounded-lg border p-3">
                    <div className="flex items-center gap-3">
                      <Skeleton className="h-9 w-9 rounded-lg" />
                      <div className="flex-1 space-y-1.5">
                        <Skeleton className="h-4 w-3/4" />
                        <Skeleton className="h-3 w-1/2" />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : recentDevices.length === 0 ? (
              <div className="text-center py-8">
                <Radar className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
                <p className="text-sm text-muted-foreground">
                  No devices discovered yet
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  Run a scan to discover devices on your network
                </p>
              </div>
            ) : (
              <div className="space-y-2">
                {recentDevices.map((device) => (
                  <DeviceCardCompact key={device.id} device={device} />
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Quick Links */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <QuickLinkCard
          title="Devices"
          description="View and manage all discovered devices"
          icon={Monitor}
          href="/devices"
        />
        <QuickLinkCard
          title="Network Map"
          description="Visual topology of your network"
          icon={Activity}
          href="/topology"
        />
        <QuickLinkCard
          title="Credentials"
          description="Manage stored credentials"
          icon={Server}
          href="/vault"
          disabled
          badge="Coming Soon"
        />
        <QuickLinkCard
          title="Settings"
          description="Configure application settings"
          icon={HardDrive}
          href="/settings"
        />
      </div>
    </div>
  )
}

// Stat card component
function StatCard({
  title,
  value,
  icon: Icon,
  variant = 'default',
  loading,
  href,
}: {
  title: string
  value: number
  icon: React.ElementType
  variant?: 'default' | 'success' | 'danger' | 'warning'
  loading?: boolean
  href?: string
}) {
  const variants = {
    default: {
      icon: 'text-muted-foreground',
      bg: 'bg-muted/50',
    },
    success: {
      icon: 'text-green-600 dark:text-green-400',
      bg: 'bg-green-500/10',
    },
    danger: {
      icon: 'text-red-600 dark:text-red-400',
      bg: 'bg-red-500/10',
    },
    warning: {
      icon: 'text-amber-600 dark:text-amber-400',
      bg: 'bg-amber-500/10',
    },
  }

  const v = variants[variant]

  const content = (
    <Card className={cn(href && 'hover:border-green-500/50 transition-colors cursor-pointer')}>
      <CardContent className="p-4">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs text-muted-foreground">{title}</p>
            {loading ? (
              <Skeleton className="h-8 w-12 mt-1" />
            ) : (
              <p className="text-2xl font-bold mt-1">{value}</p>
            )}
          </div>
          <div className={cn('p-2.5 rounded-lg', v.bg)}>
            <Icon className={cn('h-5 w-5', v.icon)} />
          </div>
        </div>
      </CardContent>
    </Card>
  )

  if (href) {
    return <Link to={href}>{content}</Link>
  }

  return content
}

// Scan history row
function ScanHistoryRow({ scan }: { scan: Scan }) {
  const statusConfig: Record<ScanStatus, { icon: React.ElementType; color: string }> = {
    completed: { icon: CheckCircle2, color: 'text-green-500' },
    running: { icon: Loader2, color: 'text-blue-500' },
    pending: { icon: Clock, color: 'text-muted-foreground' },
    failed: { icon: XCircle, color: 'text-red-500' },
    cancelled: { icon: XCircle, color: 'text-muted-foreground' },
  }

  const config = statusConfig[scan.status]
  const Icon = config.icon
  const isRunning = scan.status === 'running'

  const formatTime = (dateStr: string) => {
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
  }

  return (
    <div className="flex items-center gap-3 p-2 rounded-lg hover:bg-muted/50 transition-colors">
      <Icon className={cn('h-4 w-4', config.color, isRunning && 'animate-spin')} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium truncate">{scan.target_cidr}</p>
        <p className="text-xs text-muted-foreground">{formatTime(scan.started_at)}</p>
      </div>
      <div className="text-right">
        <p className="text-sm font-medium">{scan.devices_found}</p>
        <p className="text-xs text-muted-foreground">devices</p>
      </div>
    </div>
  )
}

// Device type row for breakdown
function DeviceTypeRow({
  icon: Icon,
  label,
  count,
}: {
  icon: React.ElementType
  label: string
  count: number
}) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm">{label}</span>
      </div>
      <span className="text-sm font-medium">{count}</span>
    </div>
  )
}

// Quick link card
function QuickLinkCard({
  title,
  description,
  icon: Icon,
  href,
  disabled,
  badge,
}: {
  title: string
  description: string
  icon: React.ElementType
  href: string
  disabled?: boolean
  badge?: string
}) {
  const content = (
    <Card
      className={cn(
        'transition-all',
        disabled
          ? 'opacity-60 cursor-not-allowed'
          : 'hover:shadow-md hover:border-green-500/50 cursor-pointer'
      )}
    >
      <CardContent className="p-4">
        <div className="flex items-start gap-3">
          <div className="p-2 rounded-lg bg-muted/50">
            <Icon className="h-5 w-5 text-muted-foreground" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="font-medium text-sm">{title}</h3>
              {badge && (
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
                  {badge}
                </span>
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  )

  if (disabled) {
    return content
  }

  return <Link to={href}>{content}</Link>
}
