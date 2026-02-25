import { memo } from 'react'
import type { TopologyBackgroundSettings } from './background-storage'

interface TopologyBackgroundProps {
  settings: TopologyBackgroundSettings
}

/**
 * Renders a background overlay behind the React Flow canvas.
 * Positioned absolutely and uses pointer-events: none so it
 * does not interfere with node/edge interaction.
 */
export const TopologyBackground = memo(function TopologyBackground({
  settings,
}: TopologyBackgroundProps) {
  if (settings.type === 'none') return null

  const opacityFraction = settings.opacity / 100

  if (settings.type === 'grid') {
    return (
      <div
        className="absolute inset-0 pointer-events-none z-0"
        style={{
          opacity: opacityFraction,
          backgroundImage:
            `repeating-linear-gradient(0deg, var(--nv-border-default) 0px, var(--nv-border-default) 1px, transparent 1px, transparent 40px),` +
            `repeating-linear-gradient(90deg, var(--nv-border-default) 0px, var(--nv-border-default) 1px, transparent 1px, transparent 40px)`,
          backgroundSize: '40px 40px',
        }}
      />
    )
  }

  if (settings.type === 'dots') {
    return (
      <div
        className="absolute inset-0 pointer-events-none z-0"
        style={{
          opacity: opacityFraction,
          backgroundImage:
            `radial-gradient(circle, var(--nv-border-default) 1.5px, transparent 1.5px)`,
          backgroundSize: '24px 24px',
        }}
      />
    )
  }

  if (settings.type === 'image' && settings.imageData) {
    return (
      <div className="absolute inset-0 pointer-events-none z-0 overflow-hidden">
        <img
          src={settings.imageData}
          alt="Topology background"
          className="w-full h-full object-contain"
          style={{ opacity: opacityFraction }}
          draggable={false}
        />
      </div>
    )
  }

  return null
})
