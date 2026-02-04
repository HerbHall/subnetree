import { useEffect, useState } from 'react'
import { RefreshCw, Radar, LayoutGrid, List, Search } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DeviceCard, DeviceCardCompact } from '@/components/device-card'
import { getTopology, triggerScan } from '@/api/devices'
import type { TopologyNode, DeviceStatus } from '@/api/types'
import { cn } from '@/lib/utils'

type ViewMode = 'grid' | 'list'

export function DevicesPage() {
  const [devices, setDevices] = useState<TopologyNode[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [scanning, setScanning] = useState(false)
  const [viewMode, setViewMode] = useState<ViewMode>('grid')
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<DeviceStatus | 'all'>('all')

  // Fetch devices on mount
  useEffect(() => {
    fetchDevices()
  }, [])

  async function fetchDevices() {
    setLoading(true)
    setError(null)
    try {
      const topology = await getTopology()
      setDevices(topology.nodes || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch devices')
    } finally {
      setLoading(false)
    }
  }

  async function handleScan() {
    setScanning(true)
    try {
      await triggerScan()
      // Wait a moment then refresh the device list
      setTimeout(() => {
        fetchDevices()
        setScanning(false)
      }, 2000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Scan failed')
      setScanning(false)
    }
  }

  // Filter devices based on search and status
  const filteredDevices = devices.filter((device) => {
    const matchesSearch =
      searchQuery === '' ||
      device.label?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      device.ip_addresses?.some((ip) => ip.includes(searchQuery)) ||
      device.manufacturer?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      device.mac_address?.toLowerCase().includes(searchQuery.toLowerCase())

    const matchesStatus = statusFilter === 'all' || device.status === statusFilter

    return matchesSearch && matchesStatus
  })

  // Count devices by status
  const statusCounts = devices.reduce(
    (acc, device) => {
      acc[device.status] = (acc[device.status] || 0) + 1
      return acc
    },
    {} as Record<string, number>
  )

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Devices</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {devices.length} device{devices.length !== 1 ? 's' : ''} discovered
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button
            onClick={handleScan}
            disabled={scanning}
            className="gap-2"
          >
            {scanning ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Radar className="h-4 w-4" />
            )}
            {scanning ? 'Scanning...' : 'Scan Network'}
          </Button>

          <Button variant="outline" size="icon" onClick={fetchDevices} disabled={loading}>
            <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
          </Button>
        </div>
      </div>

      {/* Filters and View Toggle */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        {/* Search */}
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search by name, IP, or manufacturer..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-9"
          />
        </div>

        <div className="flex items-center gap-2">
          {/* Status filter pills */}
          <div className="flex items-center gap-1">
            <StatusPill
              label="All"
              count={devices.length}
              active={statusFilter === 'all'}
              onClick={() => setStatusFilter('all')}
            />
            <StatusPill
              label="Online"
              count={statusCounts.online || 0}
              status="online"
              active={statusFilter === 'online'}
              onClick={() => setStatusFilter('online')}
            />
            <StatusPill
              label="Offline"
              count={statusCounts.offline || 0}
              status="offline"
              active={statusFilter === 'offline'}
              onClick={() => setStatusFilter('offline')}
            />
          </div>

          {/* View toggle */}
          <div className="flex items-center border rounded-md">
            <Button
              variant={viewMode === 'grid' ? 'secondary' : 'ghost'}
              size="icon"
              className="h-8 w-8 rounded-r-none"
              onClick={() => setViewMode('grid')}
            >
              <LayoutGrid className="h-4 w-4" />
            </Button>
            <Button
              variant={viewMode === 'list' ? 'secondary' : 'ghost'}
              size="icon"
              className="h-8 w-8 rounded-l-none"
              onClick={() => setViewMode('list')}
            >
              <List className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>

      {/* Error state */}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 dark:bg-red-950/20 dark:border-red-900 p-4">
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
          <Button variant="outline" size="sm" onClick={fetchDevices} className="mt-2">
            Try Again
          </Button>
        </div>
      )}

      {/* Loading state */}
      {loading && !error && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {[...Array(8)].map((_, i) => (
            <DeviceCardSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Empty state */}
      {!loading && !error && filteredDevices.length === 0 && (
        <div className="text-center py-12">
          <Radar className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          {devices.length === 0 ? (
            <>
              <h3 className="text-lg font-medium">No devices discovered</h3>
              <p className="text-sm text-muted-foreground mt-1 mb-4">
                Run a network scan to discover devices on your network.
              </p>
              <Button onClick={handleScan} disabled={scanning}>
                {scanning ? 'Scanning...' : 'Scan Network'}
              </Button>
            </>
          ) : (
            <>
              <h3 className="text-lg font-medium">No matching devices</h3>
              <p className="text-sm text-muted-foreground mt-1">
                Try adjusting your search or filters.
              </p>
            </>
          )}
        </div>
      )}

      {/* Device grid/list */}
      {!loading && !error && filteredDevices.length > 0 && (
        viewMode === 'grid' ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {filteredDevices.map((device) => (
              <DeviceCard key={device.id} device={device} />
            ))}
          </div>
        ) : (
          <div className="space-y-2">
            {filteredDevices.map((device) => (
              <DeviceCardCompact key={device.id} device={device} />
            ))}
          </div>
        )
      )}
    </div>
  )
}

// Status filter pill component
function StatusPill({
  label,
  count,
  status,
  active,
  onClick,
}: {
  label: string
  count: number
  status?: DeviceStatus
  active: boolean
  onClick: () => void
}) {
  const colors = {
    online: 'bg-green-500',
    offline: 'bg-red-500',
    degraded: 'bg-amber-500',
    unknown: 'bg-gray-400',
  }

  return (
    <button
      onClick={onClick}
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium transition-colors',
        active
          ? 'bg-primary text-primary-foreground'
          : 'bg-muted hover:bg-muted/80 text-muted-foreground'
      )}
    >
      {status && <span className={cn('h-1.5 w-1.5 rounded-full', colors[status])} />}
      {label}
      <span className="opacity-60">({count})</span>
    </button>
  )
}

// Loading skeleton for device cards
function DeviceCardSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-4 animate-pulse">
      <div className="flex items-start justify-between mb-3">
        <div className="h-11 w-11 rounded-lg bg-muted" />
        <div className="h-5 w-16 rounded bg-muted" />
      </div>
      <div className="h-4 w-3/4 rounded bg-muted mb-2" />
      <div className="h-3 w-1/2 rounded bg-muted mb-3" />
      <div className="flex justify-between">
        <div className="h-3 w-16 rounded bg-muted" />
        <div className="h-3 w-20 rounded bg-muted" />
      </div>
    </div>
  )
}
