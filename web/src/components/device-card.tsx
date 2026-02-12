import { Link } from 'react-router-dom'
import {
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
  type LucideIcon,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import type { TopologyNode, Device, DeviceType, DeviceStatus } from '@/api/types'

// Map device types to icons
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

// Map device types to display labels
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

// Status colors using design tokens
const statusColors: Record<DeviceStatus, { bg: string; text: string; dot: string }> = {
  online: {
    bg: 'bg-green-500/10',
    text: 'text-green-600 dark:text-green-400',
    dot: 'bg-green-500',
  },
  offline: {
    bg: 'bg-red-500/10',
    text: 'text-red-600 dark:text-red-400',
    dot: 'bg-red-500',
  },
  degraded: {
    bg: 'bg-amber-500/10',
    text: 'text-amber-600 dark:text-amber-400',
    dot: 'bg-amber-500',
  },
  unknown: {
    bg: 'bg-gray-500/10',
    text: 'text-gray-600 dark:text-gray-400',
    dot: 'bg-gray-400',
  },
}

/** Common device shape accepted by card components. Works with both TopologyNode and Device. */
type DeviceCardDevice = TopologyNode | Device

interface DeviceCardProps {
  device: DeviceCardDevice
  className?: string
}

/** Get display name from either TopologyNode.label or Device.hostname. */
function getDeviceName(device: DeviceCardDevice): string {
  if ('label' in device && device.label) return device.label
  if ('hostname' in device && device.hostname) return device.hostname
  return 'Unnamed Device'
}

export function DeviceCard({ device, className }: DeviceCardProps) {
  const Icon = deviceTypeIcons[device.device_type] || CircleHelp
  const typeLabel = deviceTypeLabels[device.device_type] || 'Unknown'
  const status = statusColors[device.status] || statusColors.unknown
  const primaryIp = device.ip_addresses?.[0] || 'No IP'

  return (
    <Link to={`/devices/${device.id}`} className="block">
      <Card
        className={cn(
          'transition-all hover:shadow-md hover:border-green-500/50 cursor-pointer',
          'group',
          className
        )}
      >
        <CardContent className="p-4">
          {/* Header: Icon + Status */}
          <div className="flex items-start justify-between mb-3">
            <div
              className={cn(
                'p-2.5 rounded-lg transition-colors',
                status.bg,
                'group-hover:bg-green-500/20'
              )}
            >
              <Icon className={cn('h-6 w-6', status.text)} />
            </div>

            {/* Status indicator */}
            <div className="flex items-center gap-1.5">
              <span className={cn('h-2 w-2 rounded-full animate-pulse', status.dot)} />
              <span className={cn('text-xs font-medium capitalize', status.text)}>
                {device.status}
              </span>
            </div>
          </div>

          {/* Device name */}
          <h3 className="font-semibold text-sm truncate mb-1" title={getDeviceName(device)}>
            {getDeviceName(device)}
          </h3>

          {/* IP Address */}
          <p className="text-xs text-muted-foreground font-mono truncate mb-2" title={primaryIp}>
            {primaryIp}
          </p>

          {/* Footer: Type + Manufacturer */}
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>{typeLabel}</span>
            {device.manufacturer && (
              <span className="truncate max-w-[50%]" title={device.manufacturer}>
                {device.manufacturer}
              </span>
            )}
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}

// Compact variant for denser layouts
export function DeviceCardCompact({ device, className }: DeviceCardProps) {
  const Icon = deviceTypeIcons[device.device_type] || CircleHelp
  const status = statusColors[device.status] || statusColors.unknown
  const primaryIp = device.ip_addresses?.[0] || 'No IP'

  return (
    <Link to={`/devices/${device.id}`} className="block">
      <Card
        className={cn(
          'transition-all hover:shadow-sm hover:border-green-500/50 cursor-pointer',
          className
        )}
      >
        <CardContent className="p-3 flex items-center gap-3">
          {/* Status dot */}
          <span className={cn('h-2.5 w-2.5 rounded-full flex-shrink-0', status.dot)} />

          {/* Icon */}
          <Icon className={cn('h-4 w-4 flex-shrink-0', status.text)} />

          {/* Name + IP */}
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium truncate">{getDeviceName(device)}</p>
            <p className="text-xs text-muted-foreground font-mono truncate">{primaryIp}</p>
          </div>

          {/* Manufacturer */}
          {device.manufacturer && (
            <span className="text-xs text-muted-foreground hidden sm:block truncate max-w-24">
              {device.manufacturer}
            </span>
          )}
        </CardContent>
      </Card>
    </Link>
  )
}
