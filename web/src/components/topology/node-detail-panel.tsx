import { memo, useMemo } from 'react'
import { Link } from 'react-router-dom'
import {
  X,
  ExternalLink,
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
  Container,
  CircleHelp,
  type LucideIcon,
} from 'lucide-react'
import type { TopologyNode, DeviceType, DeviceStatus } from '@/api/types'
import { getServiceByPort, type ServiceIconInfo } from '@/lib/service-icons'

interface NodeDetailPanelProps {
  node: TopologyNode
  onClose: () => void
}

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
  virtual_machine: Server,
  container: Container,
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
  virtual_machine: 'Virtual Machine',
  container: 'Container',
  unknown: 'Unknown',
}

const statusConfig: Record<DeviceStatus, { label: string; color: string }> = {
  online: { label: 'Online', color: 'var(--nv-status-online)' },
  offline: { label: 'Offline', color: 'var(--nv-status-offline)' },
  degraded: { label: 'Degraded', color: 'var(--nv-status-degraded)' },
  unknown: { label: 'Unknown', color: 'var(--nv-status-unknown)' },
}

export const NodeDetailPanel = memo(function NodeDetailPanel({
  node,
  onClose,
}: NodeDetailPanelProps) {
  const Icon = deviceTypeIcons[node.device_type] || CircleHelp
  const typeLabel = deviceTypeLabels[node.device_type] || 'Unknown'
  const status = statusConfig[node.status] || statusConfig.unknown

  // Resolve open ports to service info (stable reference via useMemo)
  const resolvedServices = useMemo(() => {
    if (!node.open_ports || node.open_ports.length === 0) return []
    return node.open_ports
      .map((port) => ({ port, service: getServiceByPort(port) }))
      .filter(
        (entry): entry is { port: number; service: ServiceIconInfo } =>
          entry.service !== undefined,
      )
  }, [node.open_ports])

  return (
    <div
      className="absolute top-0 right-0 h-full w-80 z-10 overflow-y-auto"
      style={{
        backgroundColor: 'var(--nv-bg-card)',
        borderLeft: '1px solid var(--nv-border-default)',
        boxShadow: '-4px 0 16px rgba(0, 0, 0, 0.2)',
        animation: 'slideInRight 0.2s ease-out',
      }}
    >
      {/* Header */}
      <div
        className="flex items-center justify-between p-4"
        style={{ borderBottom: '1px solid var(--nv-border-subtle)' }}
      >
        <h3
          className="text-sm font-semibold"
          style={{ color: 'var(--nv-text-primary)' }}
        >
          Device Details
        </h3>
        <button
          onClick={onClose}
          className="rounded-md p-1 transition-colors hover:bg-[var(--nv-bg-hover)]"
          style={{ color: 'var(--nv-text-secondary)' }}
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Content */}
      <div className="p-4 space-y-4">
        {/* Device type + icon */}
        <div className="flex items-center gap-3">
          <div
            className="p-2.5 rounded-lg"
            style={{
              backgroundColor: 'var(--nv-bg-active)',
            }}
          >
            <Icon className="h-6 w-6" style={{ color: status.color }} />
          </div>
          <div>
            <p
              className="text-sm font-semibold"
              style={{ color: 'var(--nv-text-primary)' }}
            >
              {node.label || 'Unnamed Device'}
            </p>
            <p
              className="text-xs"
              style={{ color: 'var(--nv-text-secondary)' }}
            >
              {typeLabel}
            </p>
          </div>
        </div>

        {/* Status */}
        <div>
          <label
            className="text-[10px] font-medium uppercase tracking-wider block mb-1"
            style={{ color: 'var(--nv-text-muted)' }}
          >
            Status
          </label>
          <div className="flex items-center gap-2">
            <span
              className="h-2.5 w-2.5 rounded-full"
              style={{ backgroundColor: status.color }}
            />
            <span
              className="text-sm font-medium"
              style={{ color: status.color }}
            >
              {status.label}
            </span>
          </div>
        </div>

        {/* IP Addresses */}
        <div>
          <label
            className="text-[10px] font-medium uppercase tracking-wider block mb-1"
            style={{ color: 'var(--nv-text-muted)' }}
          >
            IP Addresses
          </label>
          {node.ip_addresses && node.ip_addresses.length > 0 ? (
            <div className="space-y-1">
              {node.ip_addresses.map((ip) => (
                <p
                  key={ip}
                  className="text-sm font-mono"
                  style={{ color: 'var(--nv-text-primary)' }}
                >
                  {ip}
                </p>
              ))}
            </div>
          ) : (
            <p
              className="text-sm"
              style={{ color: 'var(--nv-text-secondary)' }}
            >
              No IP addresses
            </p>
          )}
        </div>

        {/* MAC Address */}
        <div>
          <label
            className="text-[10px] font-medium uppercase tracking-wider block mb-1"
            style={{ color: 'var(--nv-text-muted)' }}
          >
            MAC Address
          </label>
          <p
            className="text-sm font-mono"
            style={{ color: 'var(--nv-text-primary)' }}
          >
            {node.mac_address || 'Unknown'}
          </p>
        </div>

        {/* Manufacturer */}
        {node.manufacturer && (
          <div>
            <label
              className="text-[10px] font-medium uppercase tracking-wider block mb-1"
              style={{ color: 'var(--nv-text-muted)' }}
            >
              Manufacturer
            </label>
            <p
              className="text-sm"
              style={{ color: 'var(--nv-text-primary)' }}
            >
              {node.manufacturer}
            </p>
          </div>
        )}

        {/* Services (shown only when port data is available) */}
        {resolvedServices.length > 0 && (
          <div>
            <label
              className="text-[10px] font-medium uppercase tracking-wider block mb-1"
              style={{ color: 'var(--nv-text-muted)' }}
            >
              Services
            </label>
            <div className="space-y-1.5">
              {resolvedServices.map(({ port, service }) => {
                const SvcIcon = service.icon
                return (
                  <div key={port} className="flex items-center gap-2">
                    <SvcIcon
                      className="h-4 w-4 flex-shrink-0"
                      style={{ color: 'var(--nv-text-secondary)' }}
                    />
                    <span
                      className="text-sm flex-1"
                      style={{ color: 'var(--nv-text-primary)' }}
                    >
                      {service.name}
                    </span>
                    <span
                      className="text-xs font-mono"
                      style={{ color: 'var(--nv-text-muted)' }}
                    >
                      :{port}
                    </span>
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* View Details link */}
        <Link
          to={`/devices/${node.id}`}
          className="flex items-center justify-center gap-2 w-full rounded-md px-3 py-2 text-sm font-medium transition-colors"
          style={{
            backgroundColor: 'var(--nv-btn-primary-bg)',
            color: 'var(--nv-btn-primary-text)',
          }}
        >
          <ExternalLink className="h-4 w-4" />
          View Details
        </Link>
      </div>

      {/* Slide-in animation */}
      <style>{`
        @keyframes slideInRight {
          from {
            transform: translateX(100%);
            opacity: 0;
          }
          to {
            transform: translateX(0);
            opacity: 1;
          }
        }
      `}</style>
    </div>
  )
})
