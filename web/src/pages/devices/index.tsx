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
  Plus,
  Trash2,
  Monitor,
  Wifi,
  AlertCircle,
  AlertTriangle,
  MapPin,
  Tag,
  User,
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { DeviceCard, DeviceCardCompact } from '@/components/device-card'
import { CreateDeviceDialog } from '@/components/create-device-dialog'
import {
  listDevices,
  deleteDevice,
  triggerScan,
  getInventorySummary,
  bulkUpdateDevices,
} from '@/api/devices'
import { useScanStore } from '@/stores/scan'
import type { DeviceStatus, DeviceType } from '@/api/types'
import { cn } from '@/lib/utils'
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts'

type ViewMode = 'grid' | 'list' | 'table'
type SortField = 'hostname' | 'ip' | 'mac' | 'manufacturer' | 'device_type' | 'status'
type SortDirection = 'asc' | 'desc'

const PAGE_SIZE_OPTIONS = [25, 50, 100]

const DEVICE_TYPE_LABELS: Record<DeviceType, string> = {
  server: 'Server',
  desktop: 'Desktop',
  laptop: 'Laptop',
  mobile: 'Mobile',
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

const CATEGORY_OPTIONS = [
  'production',
  'development',
  'network',
  'storage',
  'iot',
  'personal',
  'shared',
  'other',
]

export function DevicesPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const activeScan = useScanStore((s) => s.activeScan)

  // State from URL params or defaults
  const statusFilter = (searchParams.get('status') as DeviceStatus | 'all') || 'all'
  const typeFilter = (searchParams.get('type') as DeviceType | 'all') || 'all'
  const categoryFilter = searchParams.get('category') || 'all'
  const ownerFilter = searchParams.get('owner') || 'all'

  // Local UI state
  const [viewMode, setViewMode] = useState<ViewMode>('table')
  const [searchQuery, setSearchQuery] = useState('')
  const [sortField, setSortField] = useState<SortField>('hostname')
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc')
  const [pageSize, setPageSize] = useState(25)
  const [currentPage, setCurrentPage] = useState(1)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [bulkCategory, setBulkCategory] = useState('')
  const [bulkOwner, setBulkOwner] = useState('')
  const [bulkLocation, setBulkLocation] = useState('')
  const [showBulkCategory, setShowBulkCategory] = useState(false)
  const [showBulkOwner, setShowBulkOwner] = useState(false)
  const [showBulkLocation, setShowBulkLocation] = useState(false)

  // Fetch inventory summary
  const { data: inventorySummary } = useQuery({
    queryKey: ['inventorySummary'],
    queryFn: () => getInventorySummary(),
  })

  // Fetch devices with server-side pagination and filtering
  const {
    data: deviceList,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['devices', pageSize, currentPage, statusFilter, typeFilter, categoryFilter, ownerFilter],
    queryFn: () =>
      listDevices({
        limit: pageSize,
        offset: (currentPage - 1) * pageSize,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        type: typeFilter !== 'all' ? typeFilter : undefined,
        category: categoryFilter !== 'all' ? categoryFilter : undefined,
        owner: ownerFilter !== 'all' ? ownerFilter : undefined,
      }),
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
        queryClient.invalidateQueries({ queryKey: ['devices'] })
        queryClient.invalidateQueries({ queryKey: ['topology'] })
        queryClient.invalidateQueries({ queryKey: ['inventorySummary'] })
      }, 2000)
    },
  })

  // Delete device mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteDevice(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      queryClient.invalidateQueries({ queryKey: ['topology'] })
      queryClient.invalidateQueries({ queryKey: ['inventorySummary'] })
      toast.success('Device deleted')
      setDeletingId(null)
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Failed to delete device')
      setDeletingId(null)
    },
  })

  // Bulk update mutation
  const bulkUpdateMutation = useMutation({
    mutationFn: bulkUpdateDevices,
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      queryClient.invalidateQueries({ queryKey: ['inventorySummary'] })
      toast.success(`Updated ${result.updated} device${result.updated !== 1 ? 's' : ''}`)
      setSelectedIds(new Set())
      closeBulkInputs()
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Failed to update devices')
    },
  })

  function closeBulkInputs() {
    setShowBulkCategory(false)
    setShowBulkOwner(false)
    setShowBulkLocation(false)
    setBulkCategory('')
    setBulkOwner('')
    setBulkLocation('')
  }

  function handleBulkSetCategory() {
    if (!bulkCategory) return
    bulkUpdateMutation.mutate({
      device_ids: Array.from(selectedIds),
      updates: { category: bulkCategory },
    })
  }

  function handleBulkSetOwner() {
    if (!bulkOwner.trim()) return
    bulkUpdateMutation.mutate({
      device_ids: Array.from(selectedIds),
      updates: { owner: bulkOwner.trim() },
    })
  }

  function handleBulkSetLocation() {
    if (!bulkLocation.trim()) return
    bulkUpdateMutation.mutate({
      device_ids: Array.from(selectedIds),
      updates: { location: bulkLocation.trim() },
    })
  }

  function handleDeleteClick(e: React.MouseEvent, deviceId: string) {
    e.preventDefault()
    e.stopPropagation()
    setDeletingId(deviceId)
  }

  function confirmDelete() {
    if (deletingId) {
      deleteMutation.mutate(deletingId)
    }
  }

  function cancelDelete() {
    setDeletingId(null)
  }

  function toggleSelect(deviceId: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(deviceId)) {
        next.delete(deviceId)
      } else {
        next.add(deviceId)
      }
      return next
    })
  }

  const devices = useMemo(() => deviceList?.devices || [], [deviceList?.devices])
  const totalDevices = deviceList?.total || 0

  // Client-side search filtering (search is not a server-side param here)
  const filteredDevices = useMemo(() => {
    if (!searchQuery) return devices

    const query = searchQuery.toLowerCase()
    return devices.filter(
      (d) =>
        d.hostname?.toLowerCase().includes(query) ||
        d.ip_addresses?.some((ip) => ip.includes(query)) ||
        d.manufacturer?.toLowerCase().includes(query) ||
        d.mac_address?.toLowerCase().includes(query) ||
        d.owner?.toLowerCase().includes(query) ||
        d.location?.toLowerCase().includes(query)
    )
  }, [devices, searchQuery])

  // Client-side sorting
  const sortedDevices = useMemo(() => {
    const result = [...filteredDevices]

    result.sort((a, b) => {
      let aVal: string = ''
      let bVal: string = ''

      switch (sortField) {
        case 'hostname':
          aVal = a.hostname || ''
          bVal = b.hostname || ''
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
  }, [filteredDevices, sortField, sortDirection])

  function toggleSelectAll() {
    if (selectedIds.size === sortedDevices.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(sortedDevices.map((d) => d.id)))
    }
  }

  // Server-side pagination totals
  const totalPages = Math.ceil(totalDevices / pageSize)

  // Reset to page 1 when filters change
  const updateFilter = (key: string, value: string) => {
    setCurrentPage(1)
    setSelectedIds(new Set())
    if (value === 'all') {
      searchParams.delete(key)
    } else {
      searchParams.set(key, value)
    }
    setSearchParams(searchParams)
  }

  // Count devices by status and type (from current page data for display)
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

  // Collect unique owners from devices in view for the filter dropdown
  const ownerOptions = useMemo(() => {
    const owners = new Set<string>()
    devices.forEach((d) => {
      if (d.owner) owners.add(d.owner)
    })
    return Array.from(owners).sort()
  }, [devices])

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
            {searchQuery ? `${filteredDevices.length} matching of ` : ''}
            {totalDevices} device{totalDevices !== 1 ? 's' : ''}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => setCreateDialogOpen(true)}
            className="gap-2"
          >
            <Plus className="h-4 w-4" />
            Add Device
          </Button>

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

      {/* Inventory Summary Cards */}
      {inventorySummary && (
        <div className="grid gap-3 grid-cols-2 lg:grid-cols-4">
          <SummaryCard
            label="Total Devices"
            value={inventorySummary.total_devices}
            icon={<Monitor className="h-4 w-4" />}
          />
          <SummaryCard
            label="Online"
            value={inventorySummary.online_count}
            icon={<Wifi className="h-4 w-4" />}
            className="text-green-600 dark:text-green-400"
          />
          <SummaryCard
            label="Offline"
            value={inventorySummary.offline_count}
            icon={<Wifi className="h-4 w-4" />}
            className="text-red-600 dark:text-red-400"
          />
          <SummaryCard
            label="Stale"
            value={inventorySummary.stale_count}
            icon={<AlertTriangle className="h-4 w-4" />}
            className={inventorySummary.stale_count > 0 ? 'text-amber-600 dark:text-amber-400' : undefined}
            warning={inventorySummary.stale_count > 0}
          />
        </div>
      )}

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
            }}
            className="pl-9"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* Status filter */}
          <div className="flex items-center gap-1">
            <StatusPill
              label="All"
              count={totalDevices}
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

          {/* Category filter */}
          <select
            value={categoryFilter}
            onChange={(e) => updateFilter('category', e.target.value)}
            className="h-8 px-2 rounded-md border bg-background text-sm"
          >
            <option value="all">All Categories</option>
            {CATEGORY_OPTIONS.map((cat) => (
              <option key={cat} value={cat}>
                {cat.charAt(0).toUpperCase() + cat.slice(1)}
                {inventorySummary?.by_category[cat] !== undefined
                  ? ` (${inventorySummary.by_category[cat]})`
                  : ''}
              </option>
            ))}
          </select>

          {/* Owner filter */}
          <select
            value={ownerFilter}
            onChange={(e) => updateFilter('owner', e.target.value)}
            className="h-8 px-2 rounded-md border bg-background text-sm"
          >
            <option value="all">All Owners</option>
            {ownerOptions.map((owner) => (
              <option key={owner} value={owner}>
                {owner}
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
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <AlertCircle className="h-12 w-12 text-red-400 mb-4" />
          <h3 className="text-lg font-medium">Failed to load devices</h3>
          <p className="text-sm text-muted-foreground mt-1 max-w-sm">
            {error instanceof Error ? error.message : 'An unexpected error occurred while fetching device data.'}
          </p>
          <Button variant="outline" className="mt-4 gap-2" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
            Try Again
          </Button>
        </div>
      )}

      {/* Loading state */}
      {isLoading && !error && viewMode === 'table' && (
        <div className="rounded-lg border overflow-hidden">
          <div className="bg-muted/50 px-4 py-3">
            <div className="flex items-center gap-4">
              <Skeleton className="h-4 w-4 rounded" />
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-4 w-16" />
            </div>
          </div>
          <div className="divide-y">
            {[...Array(8)].map((_, i) => (
              <div key={i} className="px-4 py-3">
                <div className="flex items-center gap-4">
                  <Skeleton className="h-4 w-4 rounded" />
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-4 w-28 font-mono" />
                  <Skeleton className="h-4 w-24" />
                  <Skeleton className="h-4 w-16" />
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-4 w-16" />
                  <Skeleton className="h-5 w-16 rounded-full" />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
      {isLoading && !error && viewMode !== 'table' && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {[...Array(8)].map((_, i) => (
            <DeviceCardSkeleton key={i} />
          ))}
        </div>
      )}

      {/* Empty state */}
      {!isLoading && !error && sortedDevices.length === 0 && (
        <div className="text-center py-12">
          <Radar className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          {totalDevices === 0 ? (
            <>
              <h3 className="text-lg font-medium">No devices discovered</h3>
              <p className="text-sm text-muted-foreground mt-1 mb-4">
                Run a network scan or add a device manually.
              </p>
              <div className="flex items-center justify-center gap-2">
                <Button onClick={() => setCreateDialogOpen(true)} variant="outline" className="gap-2">
                  <Plus className="h-4 w-4" />
                  Add Device
                </Button>
                <Button onClick={() => scanMutation.mutate()} disabled={scanMutation.isPending}>
                  {scanMutation.isPending ? 'Scanning...' : 'Scan Network'}
                </Button>
              </div>
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
      {!isLoading && !error && sortedDevices.length > 0 && viewMode === 'table' && (
        <div className="rounded-lg border overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium w-10">
                    <input
                      type="checkbox"
                      checked={selectedIds.size === sortedDevices.length && sortedDevices.length > 0}
                      onChange={toggleSelectAll}
                      className="rounded border-muted-foreground/50"
                    />
                  </th>
                  <SortableHeader field="hostname" current={sortField} onClick={handleSort}>
                    Hostname {getSortIcon('hostname')}
                  </SortableHeader>
                  <SortableHeader field="ip" current={sortField} onClick={handleSort}>
                    IP Address {getSortIcon('ip')}
                  </SortableHeader>
                  <SortableHeader field="manufacturer" current={sortField} onClick={handleSort}>
                    Manufacturer {getSortIcon('manufacturer')}
                  </SortableHeader>
                  <SortableHeader field="device_type" current={sortField} onClick={handleSort}>
                    Type {getSortIcon('device_type')}
                  </SortableHeader>
                  <th className="px-4 py-3 text-left font-medium">Category</th>
                  <th className="px-4 py-3 text-left font-medium">Owner</th>
                  <SortableHeader field="status" current={sortField} onClick={handleSort}>
                    Status {getSortIcon('status')}
                  </SortableHeader>
                  <th className="px-4 py-3 text-left font-medium w-12">
                    <span className="sr-only">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {sortedDevices.map((device) => (
                  <tr
                    key={device.id}
                    className={cn(
                      'hover:bg-muted/30 transition-colors cursor-pointer',
                      selectedIds.has(device.id) && 'bg-primary/5'
                    )}
                  >
                    <td className="px-4 py-3">
                      <input
                        type="checkbox"
                        checked={selectedIds.has(device.id)}
                        onChange={() => toggleSelect(device.id)}
                        className="rounded border-muted-foreground/50"
                        onClick={(e) => e.stopPropagation()}
                      />
                    </td>
                    <td className="px-4 py-3">
                      <Link
                        to={`/devices/${device.id}`}
                        className="font-medium hover:text-primary"
                      >
                        {device.hostname || 'Unknown'}
                      </Link>
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {device.ip_addresses?.[0] || '-'}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {device.manufacturer || '-'}
                    </td>
                    <td className="px-4 py-3">
                      <span className="capitalize">{device.device_type}</span>
                    </td>
                    <td className="px-4 py-3">
                      {device.category ? (
                        <span className="capitalize text-sm">{device.category}</span>
                      ) : (
                        <span className="text-muted-foreground/50 text-sm">--</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {device.owner ? (
                        <span className="text-sm">{device.owner}</span>
                      ) : (
                        <span className="text-muted-foreground/50 text-sm">--</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={device.status} />
                    </td>
                    <td className="px-4 py-3">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-muted-foreground hover:text-red-500"
                        onClick={(e) => handleDeleteClick(e, device.id)}
                        title="Delete device"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Grid View */}
      {!isLoading && !error && sortedDevices.length > 0 && viewMode === 'grid' && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {sortedDevices.map((device) => (
            <DeviceCard key={device.id} device={device} />
          ))}
        </div>
      )}

      {/* List View */}
      {!isLoading && !error && sortedDevices.length > 0 && viewMode === 'list' && (
        <div className="space-y-2">
          {sortedDevices.map((device) => (
            <DeviceCardCompact key={device.id} device={device} />
          ))}
        </div>
      )}

      {/* Pagination */}
      {!isLoading && !error && totalDevices > 0 && (
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
              {Math.min(currentPage * pageSize, totalDevices)} of{' '}
              {totalDevices}
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

      {/* Bulk Action Bar */}
      {selectedIds.size > 0 && (
        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40">
          <div className="flex items-center gap-3 rounded-lg border bg-card px-4 py-3 shadow-lg">
            <span className="inline-flex items-center justify-center h-6 min-w-[24px] px-1.5 rounded-full bg-primary text-primary-foreground text-xs font-medium">
              {selectedIds.size}
            </span>
            <span className="text-sm text-muted-foreground">selected</span>

            <div className="h-4 w-px bg-border" />

            {/* Set Category */}
            <div className="relative">
              <Button
                variant="outline"
                size="sm"
                className="gap-1.5"
                onClick={() => {
                  setShowBulkCategory(!showBulkCategory)
                  setShowBulkOwner(false)
                  setShowBulkLocation(false)
                }}
              >
                <Tag className="h-3.5 w-3.5" />
                Set Category
              </Button>
              {showBulkCategory && (
                <div className="absolute bottom-full mb-2 left-0 rounded-lg border bg-card p-3 shadow-lg min-w-[200px]">
                  <select
                    value={bulkCategory}
                    onChange={(e) => setBulkCategory(e.target.value)}
                    className="w-full h-8 px-2 rounded-md border bg-background text-sm mb-2"
                  >
                    <option value="">Select category...</option>
                    {CATEGORY_OPTIONS.map((cat) => (
                      <option key={cat} value={cat}>
                        {cat.charAt(0).toUpperCase() + cat.slice(1)}
                      </option>
                    ))}
                  </select>
                  <Button
                    size="sm"
                    className="w-full"
                    onClick={handleBulkSetCategory}
                    disabled={!bulkCategory || bulkUpdateMutation.isPending}
                  >
                    Apply
                  </Button>
                </div>
              )}
            </div>

            {/* Set Owner */}
            <div className="relative">
              <Button
                variant="outline"
                size="sm"
                className="gap-1.5"
                onClick={() => {
                  setShowBulkOwner(!showBulkOwner)
                  setShowBulkCategory(false)
                  setShowBulkLocation(false)
                }}
              >
                <User className="h-3.5 w-3.5" />
                Set Owner
              </Button>
              {showBulkOwner && (
                <div className="absolute bottom-full mb-2 left-0 rounded-lg border bg-card p-3 shadow-lg min-w-[200px]">
                  <Input
                    placeholder="Owner name..."
                    value={bulkOwner}
                    onChange={(e) => setBulkOwner(e.target.value)}
                    className="mb-2"
                    autoFocus
                  />
                  <Button
                    size="sm"
                    className="w-full"
                    onClick={handleBulkSetOwner}
                    disabled={!bulkOwner.trim() || bulkUpdateMutation.isPending}
                  >
                    Apply
                  </Button>
                </div>
              )}
            </div>

            {/* Set Location */}
            <div className="relative">
              <Button
                variant="outline"
                size="sm"
                className="gap-1.5"
                onClick={() => {
                  setShowBulkLocation(!showBulkLocation)
                  setShowBulkCategory(false)
                  setShowBulkOwner(false)
                }}
              >
                <MapPin className="h-3.5 w-3.5" />
                Set Location
              </Button>
              {showBulkLocation && (
                <div className="absolute bottom-full mb-2 left-0 rounded-lg border bg-card p-3 shadow-lg min-w-[200px]">
                  <Input
                    placeholder="Location..."
                    value={bulkLocation}
                    onChange={(e) => setBulkLocation(e.target.value)}
                    className="mb-2"
                    autoFocus
                  />
                  <Button
                    size="sm"
                    className="w-full"
                    onClick={handleBulkSetLocation}
                    disabled={!bulkLocation.trim() || bulkUpdateMutation.isPending}
                  >
                    Apply
                  </Button>
                </div>
              )}
            </div>

            <div className="h-4 w-px bg-border" />

            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={() => {
                setSelectedIds(new Set())
                closeBulkInputs()
              }}
              title="Clear selection"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* Create Device Dialog */}
      <CreateDeviceDialog open={createDialogOpen} onOpenChange={setCreateDialogOpen} />

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmDialog
        open={!!deletingId}
        isPending={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onCancel={cancelDelete}
      />
    </div>
  )
}

// Inventory summary card
function SummaryCard({
  label,
  value,
  icon,
  className,
  warning,
}: {
  label: string
  value: number
  icon: React.ReactNode
  className?: string
  warning?: boolean
}) {
  return (
    <div className={cn(
      'rounded-lg border bg-card p-4',
      warning && 'border-amber-500/50'
    )}>
      <div className="flex items-center gap-2 mb-1">
        <span className={cn('text-muted-foreground', className)}>{icon}</span>
        <span className="text-xs text-muted-foreground">{label}</span>
      </div>
      <p className={cn('text-2xl font-semibold', className)}>{value}</p>
    </div>
  )
}

// Delete confirmation dialog
function DeleteConfirmDialog({
  open,
  isPending,
  onConfirm,
  onCancel,
}: {
  open: boolean
  isPending: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/60 backdrop-blur-sm" onClick={onCancel} />
      <div className="relative z-50 w-full max-w-sm rounded-lg border bg-card p-6 shadow-lg">
        <h3 className="text-lg font-semibold mb-2">Delete Device</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Are you sure you want to delete this device? This action cannot be undone.
        </p>
        <div className="flex items-center justify-end gap-2">
          <Button variant="outline" onClick={onCancel} disabled={isPending}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={isPending}
          >
            {isPending ? 'Deleting...' : 'Delete'}
          </Button>
        </div>
      </div>
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
    <div className="rounded-lg border bg-card p-4">
      <div className="flex items-start justify-between mb-3">
        <Skeleton className="h-11 w-11 rounded-lg" />
        <Skeleton className="h-5 w-16 rounded-full" />
      </div>
      <Skeleton className="h-4 w-3/4 mb-2" />
      <Skeleton className="h-3 w-1/2 mb-3" />
      <div className="flex justify-between">
        <Skeleton className="h-3 w-16" />
        <Skeleton className="h-3 w-20" />
      </div>
    </div>
  )
}
