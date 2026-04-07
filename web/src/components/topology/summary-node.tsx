import { memo } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import { CircleHelp, ChevronRight } from 'lucide-react'

export interface SummaryNodeData {
  label: string
  count: number
  expanded?: boolean
  highlighted?: boolean
  dimmed?: boolean
  [key: string]: unknown
}

export type SummaryNodeType = Node<SummaryNodeData, 'summary'>

export const SummaryNode = memo(function SummaryNode({
  data,
  selected,
}: NodeProps<SummaryNodeType>) {
  const opacity = data.dimmed ? 0.35 : 1

  return (
    <div
      className="relative rounded-lg px-3 py-2 text-xs transition-all cursor-pointer"
      style={{
        backgroundColor: 'var(--nv-bg-card)',
        border: `1.5px dashed ${selected ? 'var(--nv-topo-node-online)' : 'var(--nv-topo-node-unknown)'}`,
        opacity,
        minWidth: 120,
        boxShadow: selected ? '0 0 8px var(--nv-topo-node-online)' : undefined,
      }}
    >
      <Handle type="target" position={Position.Top} className="!bg-transparent !border-0 !w-0 !h-0" />

      <div className="flex items-center gap-2">
        <CircleHelp
          className="h-4 w-4 flex-shrink-0"
          style={{ color: 'var(--nv-text-muted)' }}
        />
        <div className="flex-1 min-w-0">
          <div className="font-medium truncate" style={{ color: 'var(--nv-text-primary)' }}>
            {data.label}
          </div>
          <div style={{ color: 'var(--nv-text-muted)', fontSize: '10px' }}>
            {data.count} device{data.count !== 1 ? 's' : ''}
          </div>
        </div>
        <ChevronRight
          className="h-3.5 w-3.5 flex-shrink-0 transition-transform"
          style={{
            color: 'var(--nv-text-muted)',
            transform: data.expanded ? 'rotate(90deg)' : undefined,
          }}
        />
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-transparent !border-0 !w-0 !h-0" />
    </div>
  )
})
