import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  Server,
  Monitor,
  Laptop,
  Smartphone,
  Router,
  Network,
  Wifi,
  Shield,
  Printer,
  HardDrive,
  Cpu,
  Phone,
  Tablet,
  Camera,
  CircleHelp,
  Bot,
  Globe,
  Clock,
  Tag,
  Terminal,
  ExternalLink,
  Copy,
  Check,
  History,
  Activity,
  Edit2,
  Save,
  X,
  Radar,
  Trash2,
  Package,
  MapPin,
  User,
  Briefcase,
  Container,
  Settings2,
  AppWindow,
  Layers,
  type LucideIcon,
} from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  getDevice,
  updateDevice,
  deleteDevice,
  getDeviceStatusHistory,
  getDeviceScanHistory,
  type DeviceStatusEvent,
} from '@/api/devices'
import { getDeviceServices, getDeviceUtilization, updateDesiredState } from '@/api/services'
import type { DeviceType, DeviceStatus, Scan, Service, ServiceType, DesiredState } from '@/api/types'
import { cn } from '@/lib/utils'

// Device type icons (shared with device-card)
const deviceTypeIcons: Record<DeviceType, LucideIcon> = {
  server: Server,
  desktop: Monitor,
  laptop: Laptop,
  mobile: Smartphone,
  router: Router,
  switch: Network,
  access_point: Wifi,
  firewall: Shield,
  printer: Printer,
  nas: HardDrive,
  iot: Cpu,
  phone: Phone,
  tablet: Tablet,
  camera: Camera,
  unknown: CircleHelp,
}

const deviceTypeLabels: Record<DeviceType, string> = {
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
  iot: 'IoT Device',
  phone: 'Phone',
  tablet: 'Tablet',
  camera: 'Camera',
  unknown: 'Unknown',
}

const statusConfig: Record<DeviceStatus, { bg: string; text: string; label: string }> = {
  online: { bg: 'bg-green-500', text: 'text-green-600 dark:text-green-400', label: 'Online' },
  offline: { bg: 'bg-red-500', text: 'text-red-600 dark:text-red-400', label: 'Offline' },
  degraded: { bg: 'bg-amber-500', text: 'text-amber-600 dark:text-amber-400', label: 'Degraded' },
  unknown: { bg: 'bg-gray-400', text: 'text-gray-600 dark:text-gray-400', label: 'Unknown' },
}

export function DeviceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [copiedText, setCopiedText] = useState<string | null>(null)
  const [isEditingNotes, setIsEditingNotes] = useState(false)
  const [isEditingTags, setIsEditingTags] = useState(false)
  const [isEditingType, setIsEditingType] = useState(false)
  const [editedNotes, setEditedNotes] = useState('')
  const [editedTags, setEditedTags] = useState('')
  const [editedType, setEditedType] = useState<DeviceType>('unknown')
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [isEditingLocation, setIsEditingLocation] = useState(false)
  const [isEditingCategory, setIsEditingCategory] = useState(false)
  const [isEditingRole, setIsEditingRole] = useState(false)
  const [isEditingOwner, setIsEditingOwner] = useState(false)
  const [editedLocation, setEditedLocation] = useState('')
  const [editedCategory, setEditedCategory] = useState('')
  const [editedRole, setEditedRole] = useState('')
  const [editedOwner, setEditedOwner] = useState('')

  // Fetch device details
  const {
    data: device,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['device', id],
    queryFn: () => getDevice(id!),
    enabled: !!id,
  })

  // Fetch status history
  const { data: statusHistory } = useQuery({
    queryKey: ['device-status-history', id],
    queryFn: () => getDeviceStatusHistory(id!),
    enabled: !!id,
  })

  // Fetch scan history
  const { data: scanHistory } = useQuery({
    queryKey: ['device-scan-history', id],
    queryFn: () => getDeviceScanHistory(id!),
    enabled: !!id,
  })

  // Fetch device services (only for agent-managed devices)
  const { data: deviceServices } = useQuery({
    queryKey: ['device-services', id],
    queryFn: () => getDeviceServices(id!),
    enabled: !!id && !!device?.agent_id,
  })

  // Fetch device utilization (only for agent-managed devices)
  const { data: deviceUtilization } = useQuery({
    queryKey: ['device-utilization', id],
    queryFn: () => getDeviceUtilization(id!),
    enabled: !!id && !!device?.agent_id,
  })

  // Update desired state mutation
  const desiredStateMutation = useMutation({
    mutationFn: ({ serviceId, state }: { serviceId: string; state: DesiredState }) =>
      updateDesiredState(serviceId, state),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['device-services', id] })
      toast.success('Desired state updated')
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Failed to update desired state')
    },
  })

  // Update device mutation
  const updateMutation = useMutation({
    mutationFn: (data: {
      notes?: string
      tags?: string[]
      device_type?: DeviceType
      location?: string
      category?: string
      primary_role?: string
      owner?: string
    }) => updateDevice(id!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['device', id] })
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      queryClient.invalidateQueries({ queryKey: ['inventorySummary'] })
      setIsEditingNotes(false)
      setIsEditingTags(false)
      setIsEditingType(false)
      setIsEditingLocation(false)
      setIsEditingCategory(false)
      setIsEditingRole(false)
      setIsEditingOwner(false)
      toast.success('Device updated')
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Failed to update device')
    },
  })

  // Delete device mutation
  const deleteMutation = useMutation({
    mutationFn: () => deleteDevice(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      queryClient.invalidateQueries({ queryKey: ['topology'] })
      toast.success('Device deleted')
      navigate('/devices')
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Failed to delete device')
      setShowDeleteConfirm(false)
    },
  })

  async function copyToClipboard(text: string) {
    await navigator.clipboard.writeText(text)
    setCopiedText(text)
    setTimeout(() => setCopiedText(null), 2000)
  }

  function startEditNotes() {
    setEditedNotes(device?.notes || '')
    setIsEditingNotes(true)
  }

  function saveNotes() {
    updateMutation.mutate({ notes: editedNotes })
  }

  function cancelEditNotes() {
    setIsEditingNotes(false)
    setEditedNotes('')
  }

  function startEditTags() {
    setEditedTags(device?.tags?.join(', ') || '')
    setIsEditingTags(true)
  }

  function saveTags() {
    const tags = editedTags
      .split(',')
      .map((t) => t.trim())
      .filter((t) => t.length > 0)
    updateMutation.mutate({ tags })
  }

  function cancelEditTags() {
    setIsEditingTags(false)
    setEditedTags('')
  }

  function startEditType() {
    setEditedType(device?.device_type || 'unknown')
    setIsEditingType(true)
  }

  function saveType() {
    updateMutation.mutate({ device_type: editedType })
  }

  function cancelEditType() {
    setIsEditingType(false)
  }

  function startEditLocation() {
    setEditedLocation(device?.location || '')
    setIsEditingLocation(true)
  }

  function saveLocation() {
    updateMutation.mutate({ location: editedLocation })
  }

  function cancelEditLocation() {
    setIsEditingLocation(false)
    setEditedLocation('')
  }

  function startEditCategory() {
    setEditedCategory(device?.category || '')
    setIsEditingCategory(true)
  }

  function saveCategory() {
    updateMutation.mutate({ category: editedCategory })
  }

  function cancelEditCategory() {
    setIsEditingCategory(false)
    setEditedCategory('')
  }

  function startEditRole() {
    setEditedRole(device?.primary_role || '')
    setIsEditingRole(true)
  }

  function saveRole() {
    updateMutation.mutate({ primary_role: editedRole })
  }

  function cancelEditRole() {
    setIsEditingRole(false)
    setEditedRole('')
  }

  function startEditOwner() {
    setEditedOwner(device?.owner || '')
    setIsEditingOwner(true)
  }

  function saveOwner() {
    updateMutation.mutate({ owner: editedOwner })
  }

  function cancelEditOwner() {
    setIsEditingOwner(false)
    setEditedOwner('')
  }

  function formatTimestamp(timestamp: string) {
    return new Date(timestamp).toLocaleString()
  }

  function formatRelativeTime(timestamp: string) {
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)

    if (days > 0) return `${days} day${days > 1 ? 's' : ''} ago`
    if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''} ago`
    if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''} ago`
    return 'Just now'
  }

  if (isLoading) {
    return <DeviceDetailSkeleton />
  }

  if (error || !device) {
    return (
      <div className="space-y-4">
        <Link
          to="/devices"
          className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Devices
        </Link>
        <div className="rounded-lg border border-red-200 bg-red-50 dark:bg-red-950/20 dark:border-red-900 p-6 text-center">
          <p className="text-red-600 dark:text-red-400">
            {error instanceof Error ? error.message : 'Device not found'}
          </p>
          <Button variant="outline" size="sm" asChild className="mt-4">
            <Link to="/devices">Return to Device List</Link>
          </Button>
        </div>
      </div>
    )
  }

  const Icon = deviceTypeIcons[device.device_type] || CircleHelp
  const typeLabel = deviceTypeLabels[device.device_type] || 'Unknown'
  const status = statusConfig[device.status] || statusConfig.unknown
  const primaryIp = device.ip_addresses?.[0]

  return (
    <div className="space-y-6">
      {/* Back link */}
      <Link
        to="/devices"
        className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Devices
      </Link>

      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex items-start gap-4">
          <div className={cn('p-3 rounded-xl', status.bg + '/10')}>
            <Icon className={cn('h-8 w-8', status.text)} />
          </div>
          <div>
            <h1 className="text-2xl font-semibold">
              {device.hostname || primaryIp || 'Unnamed Device'}
            </h1>
            <div className="flex items-center gap-3 mt-1">
              <span className="text-sm text-muted-foreground">{typeLabel}</span>
              <span className="text-muted-foreground">|</span>
              <div className="flex items-center gap-1.5">
                <span className={cn('h-2 w-2 rounded-full', status.bg)} />
                <span className={cn('text-sm font-medium', status.text)}>{status.label}</span>
              </div>
            </div>
          </div>
        </div>

        {/* Quick Actions */}
        <div className="flex items-center gap-2">
          {primaryIp && (
            <>
              <Button variant="outline" size="sm" className="gap-2" asChild>
                <a href={`http://${primaryIp}`} target="_blank" rel="noopener noreferrer">
                  <Globe className="h-4 w-4" />
                  Open Web UI
                </a>
              </Button>
              <Button variant="outline" size="sm" className="gap-2" disabled title="Coming soon">
                <Terminal className="h-4 w-4" />
                SSH
              </Button>
            </>
          )}
          <Button
            variant="outline"
            size="sm"
            className="gap-2 text-red-500 hover:text-red-600 hover:bg-red-500/10"
            onClick={() => setShowDeleteConfirm(true)}
          >
            <Trash2 className="h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      {/* Info Cards Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {/* Network Info */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Network className="h-4 w-4 text-muted-foreground" />
              Network
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {/* IP Addresses */}
            <div>
              <p className="text-xs text-muted-foreground mb-1">
                IP Address{device.ip_addresses?.length > 1 ? 'es' : ''}
              </p>
              <div className="space-y-1">
                {device.ip_addresses?.map((ip) => (
                  <div key={ip} className="flex items-center gap-2">
                    <code className="text-sm font-mono bg-muted px-2 py-0.5 rounded">{ip}</code>
                    <button
                      onClick={() => copyToClipboard(ip)}
                      className="text-muted-foreground hover:text-foreground transition-colors"
                      title="Copy to clipboard"
                    >
                      {copiedText === ip ? (
                        <Check className="h-3.5 w-3.5 text-green-500" />
                      ) : (
                        <Copy className="h-3.5 w-3.5" />
                      )}
                    </button>
                  </div>
                )) || <span className="text-sm text-muted-foreground">No IP addresses</span>}
              </div>
            </div>

            {/* MAC Address */}
            {device.mac_address && (
              <div>
                <p className="text-xs text-muted-foreground mb-1">MAC Address</p>
                <div className="flex items-center gap-2">
                  <code className="text-sm font-mono bg-muted px-2 py-0.5 rounded">
                    {device.mac_address}
                  </code>
                  <button
                    onClick={() => copyToClipboard(device.mac_address)}
                    className="text-muted-foreground hover:text-foreground transition-colors"
                    title="Copy to clipboard"
                  >
                    {copiedText === device.mac_address ? (
                      <Check className="h-3.5 w-3.5 text-green-500" />
                    ) : (
                      <Copy className="h-3.5 w-3.5" />
                    )}
                  </button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Device Info */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Tag className="h-4 w-4 text-muted-foreground" />
              Device Info
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="text-xs text-muted-foreground mb-1">Type</p>
              {isEditingType ? (
                <div className="flex items-center gap-2">
                  <select
                    value={editedType}
                    onChange={(e) => setEditedType(e.target.value as DeviceType)}
                    className="h-8 px-2 rounded-md border bg-background text-sm"
                  >
                    {Object.entries(deviceTypeLabels).map(([value, label]) => (
                      <option key={value} value={value}>
                        {label}
                      </option>
                    ))}
                  </select>
                  <Button
                    size="icon"
                    className="h-7 w-7"
                    onClick={saveType}
                    disabled={updateMutation.isPending}
                  >
                    <Save className="h-3.5 w-3.5" />
                  </Button>
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-7 w-7"
                    onClick={cancelEditType}
                    disabled={updateMutation.isPending}
                  >
                    <X className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <p className="text-sm">{typeLabel}</p>
                  <button
                    onClick={startEditType}
                    className="text-muted-foreground hover:text-foreground transition-colors"
                    title="Edit device type"
                  >
                    <Edit2 className="h-3 w-3" />
                  </button>
                </div>
              )}
            </div>

            {device.manufacturer && (
              <div>
                <p className="text-xs text-muted-foreground mb-1">Manufacturer</p>
                <p className="text-sm">{device.manufacturer}</p>
              </div>
            )}

            {device.os && (
              <div>
                <p className="text-xs text-muted-foreground mb-1">Operating System</p>
                <p className="text-sm">{device.os}</p>
              </div>
            )}

            <div>
              <p className="text-xs text-muted-foreground mb-1">Discovery Method</p>
              <p className="text-sm capitalize">{device.discovery_method || 'Unknown'}</p>
            </div>

            <div>
              <p className="text-xs text-muted-foreground mb-1">Device ID</p>
              <code className="text-xs font-mono text-muted-foreground">{device.id}</code>
            </div>
          </CardContent>
        </Card>

        {/* Status / Activity */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Clock className="h-4 w-4 text-muted-foreground" />
              Status
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <p className="text-xs text-muted-foreground mb-1">Current Status</p>
              <div className="flex items-center gap-2">
                <span className={cn('h-2.5 w-2.5 rounded-full', status.bg)} />
                <span className={cn('text-sm font-medium', status.text)}>{status.label}</span>
              </div>
            </div>

            <div>
              <p className="text-xs text-muted-foreground mb-1">First Seen</p>
              <p className="text-sm">
                {device.first_seen ? formatTimestamp(device.first_seen) : 'Unknown'}
              </p>
            </div>

            <div>
              <p className="text-xs text-muted-foreground mb-1">Last Seen</p>
              <p className="text-sm">
                {device.last_seen
                  ? `${formatTimestamp(device.last_seen)} (${formatRelativeTime(device.last_seen)})`
                  : 'Unknown'}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Scout Agent Link */}
      {device.agent_id && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Bot className="h-4 w-4 text-muted-foreground" />
              Scout Agent
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                This device is managed by a Scout agent
              </p>
              <Button variant="outline" size="sm" asChild className="gap-2">
                <Link to={`/agents/${device.agent_id}`}>
                  <Bot className="h-3.5 w-3.5" />
                  View Agent
                </Link>
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Services Section (only for agent-managed devices) */}
      {device.agent_id && (
        <DeviceServicesSection
          services={deviceServices}
          utilization={deviceUtilization}
          onUpdateDesiredState={(serviceId, state) =>
            desiredStateMutation.mutate({ serviceId, state })
          }
          isPending={desiredStateMutation.isPending}
        />
      )}

      {/* Inventory Details */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Package className="h-4 w-4 text-muted-foreground" />
            Inventory Details
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            {/* Location */}
            <div>
              <p className="text-xs text-muted-foreground mb-1 flex items-center gap-1">
                <MapPin className="h-3 w-3" />
                Location
              </p>
              {isEditingLocation ? (
                <div className="flex items-center gap-2">
                  <Input
                    value={editedLocation}
                    onChange={(e) => setEditedLocation(e.target.value)}
                    placeholder="e.g., Server Room A, Office 2F"
                    className="h-8 text-sm"
                    autoFocus
                  />
                  <Button
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={saveLocation}
                    disabled={updateMutation.isPending}
                  >
                    <Save className="h-3.5 w-3.5" />
                  </Button>
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={cancelEditLocation}
                    disabled={updateMutation.isPending}
                  >
                    <X className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <p className="text-sm">
                    {device.location || <span className="text-muted-foreground">Not set</span>}
                  </p>
                  <button
                    onClick={startEditLocation}
                    className="text-muted-foreground hover:text-foreground transition-colors"
                    title="Edit location"
                  >
                    <Edit2 className="h-3 w-3" />
                  </button>
                </div>
              )}
            </div>

            {/* Category */}
            <div>
              <p className="text-xs text-muted-foreground mb-1 flex items-center gap-1">
                <Tag className="h-3 w-3" />
                Category
              </p>
              {isEditingCategory ? (
                <div className="flex items-center gap-2">
                  <select
                    value={editedCategory}
                    onChange={(e) => setEditedCategory(e.target.value)}
                    className="h-8 px-2 rounded-md border bg-background text-sm"
                  >
                    <option value="">None</option>
                    <option value="production">Production</option>
                    <option value="development">Development</option>
                    <option value="network">Network</option>
                    <option value="storage">Storage</option>
                    <option value="iot">IoT</option>
                    <option value="personal">Personal</option>
                    <option value="shared">Shared</option>
                    <option value="other">Other</option>
                  </select>
                  <Button
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={saveCategory}
                    disabled={updateMutation.isPending}
                  >
                    <Save className="h-3.5 w-3.5" />
                  </Button>
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={cancelEditCategory}
                    disabled={updateMutation.isPending}
                  >
                    <X className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <p className="text-sm capitalize">
                    {device.category || <span className="text-muted-foreground normal-case">Not set</span>}
                  </p>
                  <button
                    onClick={startEditCategory}
                    className="text-muted-foreground hover:text-foreground transition-colors"
                    title="Edit category"
                  >
                    <Edit2 className="h-3 w-3" />
                  </button>
                </div>
              )}
            </div>

            {/* Primary Role */}
            <div>
              <p className="text-xs text-muted-foreground mb-1 flex items-center gap-1">
                <Briefcase className="h-3 w-3" />
                Primary Role
              </p>
              {isEditingRole ? (
                <div className="flex items-center gap-2">
                  <Input
                    value={editedRole}
                    onChange={(e) => setEditedRole(e.target.value)}
                    placeholder="e.g., Web Server, DNS, File Share"
                    className="h-8 text-sm"
                    autoFocus
                  />
                  <Button
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={saveRole}
                    disabled={updateMutation.isPending}
                  >
                    <Save className="h-3.5 w-3.5" />
                  </Button>
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={cancelEditRole}
                    disabled={updateMutation.isPending}
                  >
                    <X className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <p className="text-sm">
                    {device.primary_role || <span className="text-muted-foreground">Not set</span>}
                  </p>
                  <button
                    onClick={startEditRole}
                    className="text-muted-foreground hover:text-foreground transition-colors"
                    title="Edit primary role"
                  >
                    <Edit2 className="h-3 w-3" />
                  </button>
                </div>
              )}
            </div>

            {/* Owner */}
            <div>
              <p className="text-xs text-muted-foreground mb-1 flex items-center gap-1">
                <User className="h-3 w-3" />
                Owner
              </p>
              {isEditingOwner ? (
                <div className="flex items-center gap-2">
                  <Input
                    value={editedOwner}
                    onChange={(e) => setEditedOwner(e.target.value)}
                    placeholder="e.g., John Doe, IT Team"
                    className="h-8 text-sm"
                    autoFocus
                  />
                  <Button
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={saveOwner}
                    disabled={updateMutation.isPending}
                  >
                    <Save className="h-3.5 w-3.5" />
                  </Button>
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-7 w-7 shrink-0"
                    onClick={cancelEditOwner}
                    disabled={updateMutation.isPending}
                  >
                    <X className="h-3.5 w-3.5" />
                  </Button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <p className="text-sm">
                    {device.owner || <span className="text-muted-foreground">Not set</span>}
                  </p>
                  <button
                    onClick={startEditOwner}
                    className="text-muted-foreground hover:text-foreground transition-colors"
                    title="Edit owner"
                  >
                    <Edit2 className="h-3 w-3" />
                  </button>
                </div>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Notes Section */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Edit2 className="h-4 w-4 text-muted-foreground" />
              Notes
            </CardTitle>
            {!isEditingNotes && (
              <Button variant="ghost" size="sm" onClick={startEditNotes} className="gap-2">
                <Edit2 className="h-3.5 w-3.5" />
                Edit
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {isEditingNotes ? (
            <div className="space-y-3">
              <textarea
                value={editedNotes}
                onChange={(e) => setEditedNotes(e.target.value)}
                className="w-full min-h-[100px] p-3 rounded-md border bg-background text-sm resize-y"
                placeholder="Add notes about this device..."
              />
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  onClick={saveNotes}
                  disabled={updateMutation.isPending}
                  className="gap-2"
                >
                  <Save className="h-3.5 w-3.5" />
                  {updateMutation.isPending ? 'Saving...' : 'Save'}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={cancelEditNotes}
                  disabled={updateMutation.isPending}
                  className="gap-2"
                >
                  <X className="h-3.5 w-3.5" />
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              {device.notes || 'No notes added yet. Click Edit to add notes about this device.'}
            </p>
          )}
        </CardContent>
      </Card>

      {/* Tags Section */}
      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Tag className="h-4 w-4 text-muted-foreground" />
              Tags
            </CardTitle>
            {!isEditingTags && (
              <Button variant="ghost" size="sm" onClick={startEditTags} className="gap-2">
                <Edit2 className="h-3.5 w-3.5" />
                Edit
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {isEditingTags ? (
            <div className="space-y-3">
              <Input
                value={editedTags}
                onChange={(e) => setEditedTags(e.target.value)}
                placeholder="Enter tags separated by commas (e.g., production, critical, web-server)"
              />
              <p className="text-xs text-muted-foreground">Separate tags with commas</p>
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  onClick={saveTags}
                  disabled={updateMutation.isPending}
                  className="gap-2"
                >
                  <Save className="h-3.5 w-3.5" />
                  {updateMutation.isPending ? 'Saving...' : 'Save'}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={cancelEditTags}
                  disabled={updateMutation.isPending}
                  className="gap-2"
                >
                  <X className="h-3.5 w-3.5" />
                  Cancel
                </Button>
              </div>
            </div>
          ) : device.tags && device.tags.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {device.tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-primary/10 text-primary"
                >
                  {tag}
                </span>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              No tags added yet. Click Edit to add tags.
            </p>
          )}
        </CardContent>
      </Card>

      {/* Status Timeline */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Activity className="h-4 w-4 text-muted-foreground" />
            Status Timeline
          </CardTitle>
        </CardHeader>
        <CardContent>
          <StatusTimeline events={statusHistory || []} />
        </CardContent>
      </Card>

      {/* Scan History */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <History className="h-4 w-4 text-muted-foreground" />
            Scan History
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ScanHistoryList scans={scanHistory || []} />
        </CardContent>
      </Card>

      {/* Quick Access Section */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <ExternalLink className="h-4 w-4 text-muted-foreground" />
            Quick Access
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-2">
            {primaryIp && (
              <>
                <Button variant="outline" size="sm" asChild>
                  <a href={`http://${primaryIp}`} target="_blank" rel="noopener noreferrer">
                    HTTP ({primaryIp})
                  </a>
                </Button>
                <Button variant="outline" size="sm" asChild>
                  <a href={`https://${primaryIp}`} target="_blank" rel="noopener noreferrer">
                    HTTPS ({primaryIp})
                  </a>
                </Button>
              </>
            )}
            <Button variant="outline" size="sm" disabled title="Requires credentials - Coming soon">
              SSH
            </Button>
            <Button variant="outline" size="sm" disabled title="Requires credentials - Coming soon">
              RDP
            </Button>
            <Button variant="outline" size="sm" disabled title="Coming soon">
              VNC
            </Button>
          </div>
          <p className="text-xs text-muted-foreground mt-3">
            Remote access protocols will be available once credentials are configured in the Vault.
          </p>
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="fixed inset-0 bg-black/60 backdrop-blur-sm"
            onClick={() => setShowDeleteConfirm(false)}
          />
          <div className="relative z-50 w-full max-w-sm rounded-lg border bg-card p-6 shadow-lg">
            <h3 className="text-lg font-semibold mb-2">Delete Device</h3>
            <p className="text-sm text-muted-foreground mb-1">
              Are you sure you want to delete{' '}
              <span className="font-medium text-foreground">
                {device.hostname || 'this device'}
              </span>
              ?
            </p>
            <p className="text-sm text-muted-foreground mb-4">
              This action cannot be undone. All associated data will be permanently removed.
            </p>
            <div className="flex items-center justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setShowDeleteConfirm(false)}
                disabled={deleteMutation.isPending}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={() => deleteMutation.mutate()}
                disabled={deleteMutation.isPending}
              >
                {deleteMutation.isPending ? 'Deleting...' : 'Delete Device'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function StatusTimeline({ events }: { events: DeviceStatusEvent[] }) {
  if (events.length === 0) {
    return (
      <div className="text-center py-6">
        <Activity className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
        <p className="text-sm text-muted-foreground">No status history available yet.</p>
        <p className="text-xs text-muted-foreground mt-1">
          Status changes will appear here as the device is monitored.
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {events.slice(0, 10).map((event, index) => {
        const config = statusConfig[event.status] || statusConfig.unknown
        return (
          <div key={event.id || index} className="flex items-start gap-3">
            <div className="relative">
              <span className={cn('h-2.5 w-2.5 rounded-full block mt-1.5', config.bg)} />
              {index < events.length - 1 && (
                <div className="absolute top-4 left-1 w-0.5 h-full -translate-x-1/2 bg-muted" />
              )}
            </div>
            <div className="flex-1 pb-3">
              <div className="flex items-center gap-2">
                <span className={cn('text-sm font-medium', config.text)}>{config.label}</span>
                <span className="text-xs text-muted-foreground">
                  {new Date(event.timestamp).toLocaleString()}
                </span>
              </div>
            </div>
          </div>
        )
      })}
      {events.length > 10 && (
        <p className="text-xs text-muted-foreground text-center">
          Showing latest 10 of {events.length} status changes
        </p>
      )}
    </div>
  )
}

function ScanHistoryList({ scans }: { scans: Scan[] }) {
  if (scans.length === 0) {
    return (
      <div className="text-center py-6">
        <Radar className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
        <p className="text-sm text-muted-foreground">No scan history available yet.</p>
        <p className="text-xs text-muted-foreground mt-1">
          Scans that discover or update this device will appear here.
        </p>
      </div>
    )
  }

  const statusColors: Record<string, string> = {
    completed: 'text-green-600 dark:text-green-400',
    running: 'text-blue-600 dark:text-blue-400',
    pending: 'text-amber-600 dark:text-amber-400',
    failed: 'text-red-600 dark:text-red-400',
    cancelled: 'text-gray-600 dark:text-gray-400',
  }

  return (
    <div className="space-y-2">
      <div className="rounded-lg border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-muted/50">
            <tr>
              <th className="px-4 py-2 text-left font-medium">Started</th>
              <th className="px-4 py-2 text-left font-medium">Target</th>
              <th className="px-4 py-2 text-left font-medium">Status</th>
              <th className="px-4 py-2 text-left font-medium">Devices Found</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {scans.slice(0, 10).map((scan) => (
              <tr key={scan.id} className="hover:bg-muted/30 transition-colors">
                <td className="px-4 py-2 text-muted-foreground">
                  {new Date(scan.started_at).toLocaleString()}
                </td>
                <td className="px-4 py-2 font-mono text-xs">{scan.target_cidr}</td>
                <td className="px-4 py-2">
                  <span className={cn('capitalize', statusColors[scan.status] || '')}>
                    {scan.status}
                  </span>
                </td>
                <td className="px-4 py-2">{scan.devices_found}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {scans.length > 10 && (
        <p className="text-xs text-muted-foreground text-center">
          Showing latest 10 of {scans.length} scans
        </p>
      )}
    </div>
  )
}

const svcTypeIcons: Record<ServiceType, React.ElementType> = {
  'docker-container': Container,
  'windows-service': Monitor,
  'systemd-service': Settings2,
  'application': AppWindow,
}

const svcTypeLabels: Record<ServiceType, string> = {
  'docker-container': 'Docker',
  'windows-service': 'Windows',
  'systemd-service': 'Systemd',
  'application': 'App',
}

const svcStatusConfig: Record<string, { bg: string; text: string; label: string }> = {
  running: { bg: 'bg-green-500', text: 'text-green-600 dark:text-green-400', label: 'Running' },
  stopped: { bg: 'bg-gray-400', text: 'text-gray-600 dark:text-gray-400', label: 'Stopped' },
  failed: { bg: 'bg-red-500', text: 'text-red-600 dark:text-red-400', label: 'Failed' },
  unknown: { bg: 'bg-amber-500', text: 'text-amber-600 dark:text-amber-400', label: 'Unknown' },
}

const gradeColors: Record<string, { text: string; bg: string }> = {
  A: { text: 'text-green-600 dark:text-green-400', bg: 'bg-green-500/10' },
  B: { text: 'text-teal-600 dark:text-teal-400', bg: 'bg-teal-500/10' },
  C: { text: 'text-yellow-600 dark:text-yellow-400', bg: 'bg-yellow-500/10' },
  D: { text: 'text-orange-600 dark:text-orange-400', bg: 'bg-orange-500/10' },
  F: { text: 'text-red-600 dark:text-red-400', bg: 'bg-red-500/10' },
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

function DeviceServicesSection({
  services,
  utilization,
  onUpdateDesiredState,
  isPending,
}: {
  services?: Service[]
  utilization?: import('@/api/types').UtilizationSummary
  onUpdateDesiredState: (serviceId: string, state: DesiredState) => void
  isPending: boolean
}) {
  const grade = utilization?.grade || '-'
  const gc = gradeColors[grade] || { text: 'text-muted-foreground', bg: 'bg-muted' }

  // Sort: running first, then by name
  const sorted = [...(services || [])].sort((a, b) => {
    if (a.status === 'running' && b.status !== 'running') return -1
    if (a.status !== 'running' && b.status === 'running') return 1
    return a.name.localeCompare(b.name)
  })

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Layers className="h-4 w-4 text-muted-foreground" />
          Services
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Utilization Summary */}
        {utilization && (
          <div className="flex items-center gap-4 flex-wrap p-3 rounded-lg bg-muted/30">
            <span className={cn('inline-flex items-center px-2.5 py-1 rounded-md text-sm font-bold', gc.bg, gc.text)}>
              {grade}
            </span>
            <div className="flex items-center gap-4 text-sm">
              <span>CPU: <strong>{utilization.cpu_percent.toFixed(1)}%</strong></span>
              <span>Memory: <strong>{utilization.memory_percent.toFixed(1)}%</strong></span>
              {utilization.disk_percent > 0 && (
                <span>Disk: <strong>{utilization.disk_percent.toFixed(1)}%</strong></span>
              )}
            </div>
            {utilization.headroom > 0 && (
              <span className="text-xs text-muted-foreground">
                {utilization.headroom.toFixed(1)}% headroom
              </span>
            )}
          </div>
        )}

        {/* Services Table */}
        {sorted.length === 0 ? (
          <div className="text-center py-6">
            <Layers className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">No services discovered on this device yet</p>
            <p className="text-xs text-muted-foreground mt-1">
              Services are auto-populated from Scout agent data.
            </p>
          </div>
        ) : (
          <div className="rounded-lg border overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-2 text-left font-medium">Name</th>
                  <th className="px-4 py-2 text-left font-medium">Type</th>
                  <th className="px-4 py-2 text-left font-medium">Status</th>
                  <th className="px-4 py-2 text-left font-medium">Desired</th>
                  <th className="px-4 py-2 text-right font-medium">CPU</th>
                  <th className="px-4 py-2 text-right font-medium">Memory</th>
                  <th className="px-4 py-2 text-left font-medium">Ports</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {sorted.map((svc) => {
                  const TypeIcon = svcTypeIcons[svc.service_type] || AppWindow
                  const typeLabel = svcTypeLabels[svc.service_type] || 'Unknown'
                  const ss = svcStatusConfig[svc.status] || svcStatusConfig.unknown

                  return (
                    <tr key={svc.id} className="hover:bg-muted/30 transition-colors">
                      <td className="px-4 py-2">
                        <span className="font-medium">{svc.display_name || svc.name}</span>
                      </td>
                      <td className="px-4 py-2">
                        <div className="flex items-center gap-1.5">
                          <TypeIcon className="h-3.5 w-3.5 text-muted-foreground" />
                          <span className="text-xs">{typeLabel}</span>
                        </div>
                      </td>
                      <td className="px-4 py-2">
                        <div className="flex items-center gap-1.5">
                          <span className={cn('h-1.5 w-1.5 rounded-full', ss.bg)} />
                          <span className={cn('text-xs', ss.text)}>{ss.label}</span>
                        </div>
                      </td>
                      <td className="px-4 py-2">
                        <select
                          value={svc.desired_state}
                          onChange={(e) =>
                            onUpdateDesiredState(svc.id, e.target.value as DesiredState)
                          }
                          disabled={isPending}
                          className="h-7 px-1.5 rounded border bg-background text-xs"
                        >
                          <option value="should-run">Should Run</option>
                          <option value="should-stop">Should Stop</option>
                          <option value="monitoring-only">Monitor Only</option>
                        </select>
                      </td>
                      <td className="px-4 py-2 text-right text-muted-foreground">
                        {svc.cpu_percent > 0 ? `${svc.cpu_percent.toFixed(1)}%` : '-'}
                      </td>
                      <td className="px-4 py-2 text-right text-muted-foreground">
                        {svc.memory_bytes > 0 ? formatBytes(svc.memory_bytes) : '-'}
                      </td>
                      <td className="px-4 py-2 text-muted-foreground">
                        {svc.ports && svc.ports.length > 0 ? (
                          <span className="text-xs font-mono">{svc.ports.join(', ')}</span>
                        ) : (
                          '-'
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function DeviceDetailSkeleton() {
  return (
    <div className="space-y-6 animate-pulse">
      <div className="h-5 w-32 bg-muted rounded" />

      <div className="flex items-start gap-4">
        <div className="h-14 w-14 bg-muted rounded-xl" />
        <div className="space-y-2">
          <div className="h-7 w-48 bg-muted rounded" />
          <div className="h-4 w-32 bg-muted rounded" />
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="rounded-lg border bg-card p-6">
            <div className="h-4 w-24 bg-muted rounded mb-4" />
            <div className="space-y-3">
              <div className="h-4 w-full bg-muted rounded" />
              <div className="h-4 w-3/4 bg-muted rounded" />
            </div>
          </div>
        ))}
      </div>

      <div className="rounded-lg border bg-card p-6">
        <div className="h-4 w-20 bg-muted rounded mb-4" />
        <div className="h-20 w-full bg-muted rounded" />
      </div>
    </div>
  )
}
