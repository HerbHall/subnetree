import { memo } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
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
import type { DeviceType, DeviceStatus } from '@/api/types'
import { getServiceBadges } from '@/lib/service-icons'

/** Data shape stored on each React Flow node. */
export interface DeviceNodeData {
  label: string
  deviceType: DeviceType
  status: DeviceStatus
  ip: string
  /** Open port numbers discovered on this device (populated when topology API includes port data). */
  openPorts?: number[]
  /** True when this node matches a search query */
  highlighted?: boolean
  /** True when a search is active but this node does NOT match */
  dimmed?: boolean
  [key: string]: unknown
}

export type DeviceNodeType = Node<DeviceNodeData, 'device'>

// Reuse the same icon mapping as device-card.tsx
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

// Status-to-CSS-variable mapping for the node border/dot
const statusColorVar: Record<DeviceStatus, string> = {
  online: 'var(--nv-topo-node-online)',
  offline: 'var(--nv-topo-node-offline)',
  degraded: 'var(--nv-topo-node-degraded)',
  unknown: 'var(--nv-topo-node-unknown)',
}

export const DeviceNode = memo(function DeviceNode({
  data,
  selected,
}: NodeProps<DeviceNodeType>) {
  const Icon = deviceTypeIcons[data.deviceType] || CircleHelp
  const color = statusColorVar[data.status] || statusColorVar.unknown

  const highlighted = data.highlighted ?? false
  const dimmed = data.dimmed ?? false
  const badges = data.openPorts ? getServiceBadges(data.openPorts) : []

  return (
    <>
      <Handle type="target" position={Position.Top} className="!w-2 !h-2" />
      <div
        className="flex flex-col items-center gap-1 rounded-lg px-3 py-2 min-w-[120px] max-w-[140px] shadow-md transition-all"
        style={{
          backgroundColor: 'var(--nv-bg-card)',
          border: `2px solid ${selected ? 'var(--nv-green-400)' : color}`,
          boxShadow: selected
            ? '0 0 12px rgba(74, 222, 128, 0.3)'
            : highlighted
              ? '0 0 0 3px rgba(74, 222, 128, 0.5), 0 0 16px rgba(74, 222, 128, 0.2)'
              : undefined,
          opacity: dimmed ? 0.35 : 1,
          transition: 'opacity 0.2s, box-shadow 0.2s, border-color 0.2s',
        }}
      >
        {/* Status dot + Icon row */}
        <div className="flex items-center gap-2 w-full">
          <span
            className="h-2 w-2 rounded-full flex-shrink-0"
            style={{ backgroundColor: color }}
          />
          <Icon className="h-4 w-4 flex-shrink-0" style={{ color }} />
          <span
            className="text-xs font-semibold truncate flex-1 text-right"
            style={{ color: 'var(--nv-text-primary)' }}
          >
            {data.label || 'Unnamed'}
          </span>
        </div>
        {/* IP address */}
        <span
          className="text-[10px] font-mono truncate w-full text-center"
          style={{ color: 'var(--nv-text-secondary)' }}
        >
          {data.ip}
        </span>
        {/* Service badges */}
        {badges.length > 0 && (
          <div className="flex items-center gap-1.5 pt-0.5">
            {badges.map((svc) => {
              const SvcIcon = svc.icon
              return (
                <span
                  key={svc.name}
                  title={svc.name}
                  className="flex items-center justify-center rounded"
                  style={{
                    width: 18,
                    height: 18,
                    backgroundColor: 'var(--nv-bg-active)',
                  }}
                >
                  <SvcIcon
                    className="h-3 w-3"
                    style={{ color: 'var(--nv-text-secondary)' }}
                  />
                </span>
              )
            })}
          </div>
        )}
      </div>
      <Handle type="source" position={Position.Bottom} className="!w-2 !h-2" />
    </>
  )
})
