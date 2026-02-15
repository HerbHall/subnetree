import { memo, useState } from 'react'
import {
  BaseEdge,
  EdgeLabelRenderer,
  getBezierPath,
  type EdgeProps,
  type Edge,
} from '@xyflow/react'

export interface UtilizationEdgeData {
  utilization?: number // 0-100
  speed?: number
  linkType?: string
  [key: string]: unknown
}

export type UtilizationEdgeType = Edge<UtilizationEdgeData, 'utilization'>

/** Map utilization percentage to a color: green (<50%), yellow (50-80%), red (>80%). */
function getUtilizationColor(utilization: number | undefined): string {
  if (utilization === undefined) return 'var(--nv-topo-link-default)'
  if (utilization < 50) return '#22c55e' // green-500
  if (utilization <= 80) return '#eab308' // yellow-500
  return '#ef4444' // red-500
}

/** Scale stroke width from 1px (0%) to 4px (100%). */
function getStrokeWidth(utilization: number | undefined): number {
  if (utilization === undefined) return 1.5
  return 1 + (utilization / 100) * 3
}

export const UtilizationEdge = memo(function UtilizationEdge(
  props: EdgeProps<UtilizationEdgeType>
) {
  const {
    id,
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
    data,
    selected,
  } = props

  const [hovered, setHovered] = useState(false)

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  })

  const utilization = data?.utilization
  const color = getUtilizationColor(utilization)
  const strokeWidth = getStrokeWidth(utilization)

  return (
    <>
      {/* Invisible wider path for easier hover interaction */}
      <path
        d={edgePath}
        fill="none"
        stroke="transparent"
        strokeWidth={20}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      />
      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          stroke: selected || hovered ? 'var(--nv-topo-link-active)' : color,
          strokeWidth: selected || hovered ? strokeWidth + 1 : strokeWidth,
          transition: 'stroke 0.2s, stroke-width 0.2s',
        }}
      />
      {/* Utilization tooltip on hover */}
      {hovered && (
        <EdgeLabelRenderer>
          <div
            className="nodrag nopan pointer-events-none rounded px-2 py-1 text-[10px] font-mono"
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              backgroundColor: 'var(--nv-bg-elevated)',
              color: 'var(--nv-text-secondary)',
              border: '1px solid var(--nv-border-default)',
            }}
          >
            {utilization !== undefined ? (
              <span>
                <span style={{ color, fontWeight: 600 }}>{utilization}%</span>
                {' utilization'}
                {data?.linkType && ` \u00b7 ${data.linkType}`}
              </span>
            ) : (
              <span>{data?.linkType || 'No data'}</span>
            )}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
})
