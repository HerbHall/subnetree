import { memo } from 'react'
import { Layers, LayoutGrid, Activity, Globe } from 'lucide-react'
import type { ViewMode } from '@/lib/topology-grouping'

interface TopologyViewTabsProps {
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
}

const viewOptions: { value: ViewMode; label: string; Icon: typeof Layers }[] = [
  { value: 'all', label: 'All Devices', Icon: Layers },
  { value: 'by-type', label: 'By Type', Icon: LayoutGrid },
  { value: 'by-status', label: 'By Status', Icon: Activity },
  { value: 'by-subnet', label: 'By Subnet', Icon: Globe },
]

/**
 * Tab bar for switching between topology view modes.
 * Rendered above the topology canvas, below the page header.
 */
export const TopologyViewTabs = memo(function TopologyViewTabs({
  viewMode,
  onViewModeChange,
}: TopologyViewTabsProps) {
  return (
    <div
      className="flex items-center gap-1 rounded-lg px-2 py-1.5 shadow-md"
      style={{
        backgroundColor: 'var(--nv-bg-card)',
        border: '1px solid var(--nv-border-default)',
        backdropFilter: 'blur(8px)',
      }}
    >
      {viewOptions.map(({ value, label, Icon }) => (
        <button
          key={value}
          onClick={() => onViewModeChange(value)}
          title={`View: ${label}`}
          className="flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium transition-colors"
          style={{
            backgroundColor: viewMode === value ? 'var(--nv-bg-active)' : 'transparent',
            color: viewMode === value ? 'var(--nv-text-accent)' : 'var(--nv-text-secondary)',
          }}
        >
          <Icon className="h-3.5 w-3.5" />
          <span className="hidden sm:inline">{label}</span>
        </button>
      ))}
    </div>
  )
})
