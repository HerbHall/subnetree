import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import {
  ArrowLeft,
  Server,
  Monitor,
  Laptop,
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
  Globe,
  Clock,
  Tag,
  Terminal,
  ExternalLink,
  Copy,
  Check,
  type LucideIcon,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { getTopology } from '@/api/devices'
import type { TopologyNode, DeviceType, DeviceStatus } from '@/api/types'
import { cn } from '@/lib/utils'

// Device type icons (shared with device-card)
const deviceTypeIcons: Record<DeviceType, LucideIcon> = {
  server: Server,
  desktop: Monitor,
  laptop: Laptop,
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
  const [device, setDevice] = useState<TopologyNode | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [copiedIp, setCopiedIp] = useState<string | null>(null)

  useEffect(() => {
    async function fetchDevice() {
      if (!id) return
      setLoading(true)
      setError(null)
      try {
        const topology = await getTopology()
        const found = topology.nodes?.find((n) => n.id === id)
        if (found) {
          setDevice(found)
        } else {
          setError('Device not found')
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch device')
      } finally {
        setLoading(false)
      }
    }
    fetchDevice()
  }, [id])

  async function copyToClipboard(text: string) {
    await navigator.clipboard.writeText(text)
    setCopiedIp(text)
    setTimeout(() => setCopiedIp(null), 2000)
  }

  if (loading) {
    return <DeviceDetailSkeleton />
  }

  if (error || !device) {
    return (
      <div className="space-y-4">
        <Link to="/devices" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="h-4 w-4" />
          Back to Devices
        </Link>
        <div className="rounded-lg border border-red-200 bg-red-50 dark:bg-red-950/20 dark:border-red-900 p-6 text-center">
          <p className="text-red-600 dark:text-red-400">{error || 'Device not found'}</p>
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
      <Link to="/devices" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors">
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
            <h1 className="text-2xl font-semibold">{device.label || 'Unnamed Device'}</h1>
            <div className="flex items-center gap-3 mt-1">
              <span className="text-sm text-muted-foreground">{typeLabel}</span>
              <span className="text-muted-foreground">•</span>
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
              <p className="text-xs text-muted-foreground mb-1">IP Address{device.ip_addresses?.length > 1 ? 'es' : ''}</p>
              <div className="space-y-1">
                {device.ip_addresses?.map((ip) => (
                  <div key={ip} className="flex items-center gap-2">
                    <code className="text-sm font-mono bg-muted px-2 py-0.5 rounded">{ip}</code>
                    <button
                      onClick={() => copyToClipboard(ip)}
                      className="text-muted-foreground hover:text-foreground transition-colors"
                      title="Copy to clipboard"
                    >
                      {copiedIp === ip ? (
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
                  <code className="text-sm font-mono bg-muted px-2 py-0.5 rounded">{device.mac_address}</code>
                  <button
                    onClick={() => copyToClipboard(device.mac_address)}
                    className="text-muted-foreground hover:text-foreground transition-colors"
                    title="Copy to clipboard"
                  >
                    {copiedIp === device.mac_address ? (
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
              <p className="text-sm">{typeLabel}</p>
            </div>

            {device.manufacturer && (
              <div>
                <p className="text-xs text-muted-foreground mb-1">Manufacturer</p>
                <p className="text-sm">{device.manufacturer}</p>
              </div>
            )}

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
              <p className="text-xs text-muted-foreground mb-1">Last Seen</p>
              <p className="text-sm text-muted-foreground">Recently</p>
            </div>
          </CardContent>
        </Card>
      </div>

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
    </div>
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
    </div>
  )
}
