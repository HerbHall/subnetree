import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Monitor,
  Wifi,
  WifiOff,
  AlertTriangle,
  Radar,
  RefreshCw,
  ArrowRight,
  Server,
  Router,
  HardDrive,
  Activity,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DeviceCardCompact } from '@/components/device-card'
import { getTopology, triggerScan } from '@/api/devices'
import type { TopologyNode } from '@/api/types'
import { cn } from '@/lib/utils'

export function DashboardPage() {
  const [devices, setDevices] = useState<TopologyNode[]>([])
  const [loading, setLoading] = useState(true)
  const [scanning, setScanning] = useState(false)

  useEffect(() => {
    fetchDevices()
  }, [])

  async function fetchDevices() {
    setLoading(true)
    try {
      const topology = await getTopology()
      setDevices(topology.nodes || [])
    } catch {
      // Silently fail on dashboard - devices page will show errors
    } finally {
      setLoading(false)
    }
  }

  async function handleScan() {
    setScanning(true)
    try {
      await triggerScan()
      setTimeout(() => {
        fetchDevices()
        setScanning(false)
      }, 2000)
    } catch {
      setScanning(false)
    }
  }

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

  // Recent devices (just show first 5 for now - would sort by last_seen if available)
  const recentDevices = devices.slice(0, 5)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Dashboard</h1>
          <p className="text-sm text-muted-foreground mt-1">Network overview and quick actions</p>
        </div>

        <div className="flex items-center gap-2">
          <Button onClick={handleScan} disabled={scanning} className="gap-2">
            {scanning ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Radar className="h-4 w-4" />
            )}
            {scanning ? 'Scanning...' : 'Scan Network'}
          </Button>
        </div>
      </div>

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

      {/* Main Content Grid */}
      <div className="grid gap-6 lg:grid-cols-3">
        {/* Device Types Breakdown */}
        <Card className="lg:col-span-1">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium">Device Types</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="space-y-3">
                {[...Array(4)].map((_, i) => (
                  <div key={i} className="flex items-center justify-between animate-pulse">
                    <div className="h-4 w-20 bg-muted rounded" />
                    <div className="h-4 w-8 bg-muted rounded" />
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
                  count={(typeCounts.router || 0) + (typeCounts.switch || 0) + (typeCounts.access_point || 0)}
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
        <Card className="lg:col-span-2">
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
                  <div key={i} className="h-16 rounded-lg border bg-muted/30 animate-pulse" />
                ))}
              </div>
            ) : recentDevices.length === 0 ? (
              <div className="text-center py-8">
                <Radar className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
                <p className="text-sm text-muted-foreground mb-3">No devices discovered yet</p>
                <Button size="sm" onClick={handleScan} disabled={scanning}>
                  {scanning ? 'Scanning...' : 'Run First Scan'}
                </Button>
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
          disabled
          badge="Coming Soon"
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
              <div className="h-8 w-12 bg-muted rounded animate-pulse mt-1" />
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
