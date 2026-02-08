import { useState, useRef, useCallback, useMemo } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  RefreshCw,
  Radar,
  LayoutGrid,
  List,
  Search,
  Table2,
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DeviceCard, DeviceCardCompact } from '@/components/device-card'
import { getTopology, triggerScan } from '@/api/devices'
import { useScanStore } from '@/stores/scan'
import type { DeviceStatus, DeviceType } from '@/api/types'
import { cn } from '@/lib/utils'
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts'

type ViewMode = 'grid' | 'list' | 'table'
type SortField = 'label' | 'ip' | 'mac' | 'manufacturer' | 'device_type' | 'status'
type SortDirection = 'asc' | 'desc'

const PAGE_SIZE_OPTIONS = [25, 50, 100]

const DEVICE_TYPE_LABELS: Record<DeviceType, string> = {
  server: 'Server',
  desktop: 'Desktop',
  laptop: 'Laptop',
  router: 'Router',
  switch: 'Switch',
  access_point: 'Access Point',
  firewall: 'Firewall',
  printer: 'Printer',
  nas: 'NAS',
  iot: 'IoT',
  phone: 'Phone',
  tablet: 'Tablet',
  camera: 'Camera',
  unknown: 'Unknown',
}

export function DevicesPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const activeScan = useScanStore((s) => s.activeScan)

  // State from URL params or defaults
  const statusFilter = (searchParams.get('status') as DeviceStatus | 'all') || 'all'
  const typeFilter = (searchParams.get('type') as DeviceType | 'all') || 'all'

  // Local UI state
  const [viewMode, setViewMode] = useState<ViewMode>('table')
  const [searchQuery, setSearchQuery] = useState('')
  const [sortField, setSortField] = useState<SortField>('label')
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc')
  const [pageSize, setPageSize] = useState(25)
  const [currentPage, setCurrentPage] = useState(1)

  // Fetch devices with TanStack Query
  const {
    data: topology,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['topology'],
    queryFn: getTopology,
  })

  // Keyboard shortcuts
  const searchInputRef = useRef<HTMLInputElement>(null)
  const focusSearch = useCallback(() => searchInputRef.current?.focus(), [])
  const refreshData = useCallback(() => { refetch() }, [refetch])

  useKeyboardShortcuts(
    useMemo(
      () => [
        { key: '/', handler: focusSearch, description: 'Focus search' },
        { key: 'r', handler: refreshData, description: 'Refresh data' },
      ],
      [focusSearch, refreshData]
    )
  )

  // Scan mutation
  const scanMutation = useMutation({
    mutationFn: () => triggerScan(),
    onSuccess: () => {
      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['topology'] })
      }, 2000)
    },
  })

  const devices = useMemo(() => topology?.nodes || [], [topology?.nodes])

  // Filter, sort, and paginate devices
  const processedDevices = useMemo(() => {
    let result = [...devices]

    // Apply search filter
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      result = result.filter(
        (d) =>
          d.label?.toLowerCase().includes(query) ||
          d.ip_addresses?.some((ip) => ip.includes(query)) ||
          d.manufacturer?.toLowerCase().includes(query) ||
          d.mac_address?.toLowerCase().includes(query)
      )
    }

    // Apply status filter
    if (statusFilter !== 'all') {
      result = result.filter((d) => d.status === statusFilter)
    }

    // Apply device type filter
    if (typeFilter !== 'all') {
      result = result.filter((d) => d.device_type === typeFilter)
    }

    // Apply sorting
    result.sort((a, b) => {
      let aVal: string | number = ''
      let bVal: string | number = ''

      switch (sortField) {
        case 'label':
          aVal = a.label || ''
          bVal = b.label || ''
          break
        case 'ip':
          aVal = a.ip_addresses?.[0] || ''
          bVal = b.ip_addresses?.[0] || ''
          break
        case 'mac':
          aVal = a.mac_address || ''
          bVal = b.mac_address || ''
          break
        case 'manufacturer':
          aVal = a.manufacturer || ''
          bVal = b.manufacturer || ''
          break
        case 'device_type':
          aVal = a.device_type || ''
          bVal = b.device_type || ''
          break
        case 'status':
          aVal = a.status || ''
          bVal = b.status || ''
          break
      }

      if (aVal < bVal) return sortDirection === 'asc' ? -1 : 1
      if (aVal > bVal) return sortDirection === 'asc' ? 1 : -1
      return 0
    })

    return result
  }, [devices, searchQuery, statusFilter, typeFilter, sortField, sortDirection])

  // Pagination
  const totalPages = Math.ceil(processedDevices.length / pageSize)
  const paginatedDevices = processedDevices.slice(
    (currentPage - 1) * pageSize,
    currentPage * pageSize
  )

  // Reset to page 1 when filters change
  const updateFilter = (key: string, value: string) => {
    setCurrentPage(1)
    if (value === 'all') {
      searchParams.delete(key)
    } else {
      searchParams.set(key, value)
    }
    setSearchParams(searchParams)
  }

  // Count devices by status and type
  const statusCounts = devices.reduce(
    (acc, d) => {
      acc[d.status] = (acc[d.status] || 0) + 1
      return acc
    },
    {} as Record<string, number>
  )

  const typeCounts = devices.reduce(
    (acc, d) => {
      acc[d.device_type] = (acc[d.device_type] || 0) + 1
      return acc
    },
    {} as Record<string, number>
  )

  // Handle sort click
  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('asc')
    }
  }

  // Get sort icon for column
  const getSortIcon = (field: SortField) => {
    if (sortField !== field) return <ChevronsUpDown className="h-3.5 w-3.5 opacity-50" />
    return sortDirection === 'asc' ? (
      <ChevronUp className="h-3.5 w-3.5" />
    ) : (
      <ChevronDown className="h-3.5 w-3.5" />
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Devices</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {processedDevices.length} of {devices.length} device
            {devices.length !== 1 ? 's' : ''}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button
            onClick={() => scanMutation.mutate()}
            disabled={scanMutation.isPending || !!activeScan}
            className="gap-2"
          >
            {scanMutation.isPending || activeScan ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Radar className="h-4 w-4" />
            )}
            {activeScan ? 'Scanning...' : 'Scan Network'}
          </Button>

          <Button variant="outline" size="icon" onClick={() => refetch()} disabled={isLoading}>
            <RefreshCw className={cn('h-4 w-4', isLoading && 'animate-spin')} />
          </Button>
        </div>
      </div>

      {/* Filters Row */}
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        {/* Search */}
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            ref={searchInputRef}
            placeholder="Search hostname, IP, MAC, manufacturer..."
            value={searchQuery}
            onChange={(e) => {
              setSearchQuery(e.target.value)
              setCurrentPage(1)
            }}
            className="pl-9"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* Status filter */}
          <div className="flex items-center gap-1">
            <StatusPill
              label="All"
              count={devices.length}
              active={statusFilter === 'all'}
              onClick={() => updateFilter('status', 'all')}
            />
            <StatusPill
              label="Online"
              count={statusCounts.online || 0}
              status="online"
              active={statusFilter === 'online'}
              onClick={() => updateFilter('status', 'online')}
            />
            <StatusPill
              label="Offline"
              count={statusCounts.offline || 0}
              status="offline"
              active={statusFilter === 'offline'}
              onClick={() => updateFilter('status', 'offline')}
            />
          </div>

          {/* Device type filter */}
          <select
            value={typeFilter}
            onChange={(e) => updateFilter('type', e.target.value)}
            className="h-8 px-2 rounded-md border bg-background text-sm"
          >
            <option value="all">All Types</option>
            {Object.entries(typeCounts).map(([type, count]) => (
              <option key={type} value={type}>
                {DEVICE_TYPE_LABELS[type as DeviceType] || type} ({count})
              </option>
            ))}
          </select>

          {/* View toggle */}
          <div className="flex items-center border rounded-md">
            <Button
              variant={viewMode === 'table' ? 'secondary' : 'ghost'}
              size="icon"
              className="h-8 w-8 rounded-none rounded-l-md"
              onClick={() => setViewMode('table')}
              title="Table view"
            >
              <Table2 className="h-4 w-4" />
            </Button>
            <Button
              variant={viewMode === 'grid' ? 'secondary' : 'ghost'}
              size="icon"
              className="h-8 w-8 rounded-none"
              onClick={() => setViewMode('grid')}
              title="Grid view"
            >
              <LayoutGrid className="h-4 w-4" />
            </Button>
            <Button
              variant={viewMode === 'list' ? 'secondary' : 'ghost'}
              size="icon"
              className="h-8 w-8 rounded-none rounded-r-md"
              onClick={() => setViewMode('list')}
              title="List view"
            >
              <List className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>

      {/* Error state */}
      {error && (
        <div className="rounded-lg border border-red-500/50 bg-red-500/10 p-4">
          <p className="text-sm text-red-400">
            {error instanceof Error ? error.message : 'Failed to fetch devices'}
          </p>
          <Button variant="outline" size="sm" onClick={() => refetch()} className="mt-2">
            Try Again
          </Button>
        </div>
      )}

      {/* Loading state */}
      {isLoading && !error && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {[...Array(8)].map((_, i) => (
            <DeviceCardSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Empty state */}
      {!isLoading && !error && processedDevices.length === 0 && (
        <div className="text-center py-12">
          <Radar className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          {devices.length === 0 ? (
            <>
              <h3 className="text-lg font-medium">No devices discovered</h3>
              <p className="text-sm text-muted-foreground mt-1 mb-4">
                Run a network scan to discover devices on your network.
              </p>
              <Button onClick={() => scanMutation.mutate()} disabled={scanMutation.isPending}>
                {scanMutation.isPending ? 'Scanning...' : 'Scan Network'}
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

      {/* Table View */}
      {!isLoading && !error && paginatedDevices.length > 0 && viewMode === 'table' && (
        <div className="rounded-lg border overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <SortableHeader field="label" current={sortField} onClick={handleSort}>
                    Hostname {getSortIcon('label')}
                  </SortableHeader>
                  <SortableHeader field="ip" current={sortField} onClick={handleSort}>
                    IP Address {getSortIcon('ip')}
                  </SortableHeader>
                  <SortableHeader field="mac" current={sortField} onClick={handleSort}>
                    MAC Address {getSortIcon('mac')}
                  </SortableHeader>
                  <SortableHeader field="manufacturer" current={sortField} onClick={handleSort}>
                    Manufacturer {getSortIcon('manufacturer')}
                  </SortableHeader>
                  <SortableHeader field="device_type" current={sortField} onClick={handleSort}>
                    Type {getSortIcon('device_type')}
                  </SortableHeader>
                  <SortableHeader field="status" current={sortField} onClick={handleSort}>
                    Status {getSortIcon('status')}
                  </SortableHeader>
                </tr>
              </thead>
              <tbody className="divide-y">
                {paginatedDevices.map((device) => (
                  <tr
                    key={device.id}
                    className="hover:bg-muted/30 transition-colors cursor-pointer"
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/devices/${device.id}`}
                        className="font-medium hover:text-primary"
                      >
                        {device.label || 'Unknown'}
                      </Link>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {device.ip_addresses?.[0] || '-'}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                      {device.mac_address || '-'}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {device.manufacturer || '-'}
                    </td>
                    <td className="px-4 py-3">
                      <span className="capitalize">{device.device_type}</span>
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={device.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Grid View */}
      {!isLoading && !error && paginatedDevices.length > 0 && viewMode === 'grid' && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {paginatedDevices.map((device) => (
            <DeviceCard key={device.id} device={device} />
          ))}
        </div>
      )}

      {/* List View */}
      {!isLoading && !error && paginatedDevices.length > 0 && viewMode === 'list' && (
        <div className="space-y-2">
          {paginatedDevices.map((device) => (
            <DeviceCardCompact key={device.id} device={device} />
          ))}
        </div>
      )}

      {/* Pagination */}
      {!isLoading && !error && processedDevices.length > 0 && (
        <div className="flex flex-col sm:flex-row items-center justify-between gap-4 pt-4 border-t">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <span>Show</span>
            <select
              value={pageSize}
              onChange={(e) => {
                setPageSize(Number(e.target.value))
                setCurrentPage(1)
              }}
              className="h-8 px-2 rounded-md border bg-background"
            >
              {PAGE_SIZE_OPTIONS.map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
            <span>per page</span>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">
              {(currentPage - 1) * pageSize + 1}-
              {Math.min(currentPage * pageSize, processedDevices.length)} of{' '}
              {processedDevices.length}
            </span>

            <div className="flex items-center gap-1">
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={() => setCurrentPage(1)}
                disabled={currentPage === 1}
              >
                <ChevronLeft className="h-4 w-4" />
                <ChevronLeft className="h-4 w-4 -ml-2" />
              </Button>
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={() => setCurrentPage(currentPage - 1)}
                disabled={currentPage === 1}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <span className="px-2 text-sm">
                {currentPage} / {totalPages || 1}
              </span>
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={() => setCurrentPage(currentPage + 1)}
                disabled={currentPage >= totalPages}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={() => setCurrentPage(totalPages)}
                disabled={currentPage >= totalPages}
              >
                <ChevronRight className="h-4 w-4" />
                <ChevronRight className="h-4 w-4 -ml-2" />
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// Sortable table header
function SortableHeader({
  field,
  current,
  onClick,
  children,
}: {
  field: SortField
  current: SortField
  onClick: (field: SortField) => void
  children: React.ReactNode
}) {
  return (
    <th
      className={cn(
        'px-4 py-3 text-left font-medium cursor-pointer hover:bg-muted/80 transition-colors select-none',
        current === field && 'text-foreground'
      )}
      onClick={() => onClick(field)}
    >
      <div className="flex items-center gap-1">{children}</div>
    </th>
  )
}

// Status badge
function StatusBadge({ status }: { status: DeviceStatus }) {
  const config = {
    online: { bg: 'bg-green-500/10', text: 'text-green-500', dot: 'bg-green-500' },
    offline: { bg: 'bg-red-500/10', text: 'text-red-500', dot: 'bg-red-500' },
    degraded: { bg: 'bg-amber-500/10', text: 'text-amber-500', dot: 'bg-amber-500' },
    unknown: { bg: 'bg-gray-500/10', text: 'text-gray-500', dot: 'bg-gray-500' },
  }
  const c = config[status]

  return (
    <span className={cn('inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs', c.bg, c.text)}>
      <span className={cn('h-1.5 w-1.5 rounded-full', c.dot)} />
      <span className="capitalize">{status}</span>
    </span>
  )
}

// Status filter pill
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

// Loading skeleton
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
