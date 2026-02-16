import { memo } from 'react'
import { type NodeProps, type Node } from '@xyflow/react'
import type { GroupNodeData } from '@/lib/topology-grouping'

export type GroupNodeType = Node<GroupNodeData, 'group'>

/** Background color for group containers, keyed by groupKind + label context. */
const STATUS_COLORS: Record<string, string> = {
  online: 'rgba(74, 222, 128, 0.06)',
  offline: 'rgba(248, 113, 113, 0.06)',
  degraded: 'rgba(251, 191, 36, 0.06)',
  unknown: 'rgba(148, 163, 184, 0.06)',
}

/** Border color for group containers by status. */
const STATUS_BORDER_COLORS: Record<string, string> = {
  online: 'rgba(74, 222, 128, 0.25)',
  offline: 'rgba(248, 113, 113, 0.25)',
  degraded: 'rgba(251, 191, 36, 0.25)',
  unknown: 'rgba(148, 163, 184, 0.25)',
}

/**
 * Determine subtle background and border colors based on the group kind and label.
 * Status groups get colored backgrounds; type and subnet groups get neutral styling.
 */
function getGroupColors(data: GroupNodeData): { bg: string; border: string } {
  if (data.groupKind === 'by-status') {
    const statusKey = data.label.toLowerCase()
    return {
      bg: STATUS_COLORS[statusKey] ?? 'rgba(148, 163, 184, 0.04)',
      border: STATUS_BORDER_COLORS[statusKey] ?? 'rgba(148, 163, 184, 0.15)',
    }
  }
  // Type and subnet groups use a neutral, slightly tinted background
  return {
    bg: 'rgba(148, 163, 184, 0.04)',
    border: 'rgba(148, 163, 184, 0.15)',
  }
}

/**
 * GroupNode renders a labeled container around grouped device nodes.
 * Used when the topology is in a grouped view mode (by-type, by-status, by-subnet).
 */
export const GroupNode = memo(function GroupNode({
  data,
}: NodeProps<GroupNodeType>) {
  const { bg, border } = getGroupColors(data)

  return (
    <div
      className="w-full h-full rounded-xl"
      style={{
        backgroundColor: bg,
        border: `1px dashed ${border}`,
      }}
    >
      {/* Group header label */}
      <div
        className="flex items-center gap-2 px-3 py-2"
      >
        <span
          className="text-xs font-semibold tracking-wide"
          style={{ color: 'var(--nv-text-primary)' }}
        >
          {data.label}
        </span>
        <span
          className="inline-flex items-center justify-center rounded-full px-1.5 py-0.5 text-[10px] font-medium"
          style={{
            backgroundColor: 'var(--nv-bg-active)',
            color: 'var(--nv-text-secondary)',
          }}
        >
          {data.count}
        </span>
      </div>
    </div>
  )
})
