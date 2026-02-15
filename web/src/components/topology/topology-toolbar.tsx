import { memo, useState, useRef, useEffect, type RefObject } from 'react'
import { useReactFlow } from '@xyflow/react'
import { toPng } from 'html-to-image'
import { toast } from 'sonner'
import {
  ZoomIn,
  ZoomOut,
  Maximize,
  Map,
  CircleDot,
  GitBranch,
  LayoutGrid,
  Download,
  Activity,
  Save,
  FolderOpen,
  Trash2,
} from 'lucide-react'
import type { SavedLayout } from './layout-storage'

export type LayoutAlgorithm = 'circular' | 'hierarchical' | 'grid'

interface TopologyToolbarProps {
  layout: LayoutAlgorithm
  onLayoutChange: (layout: LayoutAlgorithm) => void
  showMinimap: boolean
  onMinimapToggle: () => void
  flowRef: RefObject<HTMLDivElement | null>
  showUtilization: boolean
  onUtilizationToggle: () => void
  savedLayouts: SavedLayout[]
  onSaveLayout: (name: string) => void
  onLoadLayout: (id: string) => void
  onDeleteLayout: (id: string) => void
}

const layoutOptions: { value: LayoutAlgorithm; label: string; Icon: typeof CircleDot }[] = [
  { value: 'circular', label: 'Circular', Icon: CircleDot },
  { value: 'hierarchical', label: 'Hierarchical', Icon: GitBranch },
  { value: 'grid', label: 'Grid', Icon: LayoutGrid },
]

/** Toolbar separator element. */
function Separator() {
  return (
    <div
      className="h-5 w-px mx-1"
      style={{ backgroundColor: 'var(--nv-border-default)' }}
    />
  )
}

export const TopologyToolbar = memo(function TopologyToolbar({
  layout,
  onLayoutChange,
  showMinimap,
  onMinimapToggle,
  flowRef,
  showUtilization,
  onUtilizationToggle,
  savedLayouts,
  onSaveLayout,
  onLoadLayout,
  onDeleteLayout,
}: TopologyToolbarProps) {
  const { zoomIn, zoomOut, fitView } = useReactFlow()
  const [exporting, setExporting] = useState(false)
  const [layoutsOpen, setLayoutsOpen] = useState(false)
  const [saveName, setSaveName] = useState('')
  const layoutsRef = useRef<HTMLDivElement>(null)

  // Close layouts dropdown when clicking outside
  useEffect(() => {
    if (!layoutsOpen) return
    function handleClickOutside(e: MouseEvent) {
      if (layoutsRef.current && !layoutsRef.current.contains(e.target as globalThis.Node)) {
        setLayoutsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [layoutsOpen])

  const handleExport = async () => {
    const element = flowRef.current
    if (!element) return

    setExporting(true)
    try {
      const dataUrl = await toPng(element, {
        backgroundColor: '#1a1a2e',
        quality: 1,
        pixelRatio: 2,
      })
      const link = document.createElement('a')
      const date = new Date().toISOString().split('T')[0]
      link.download = `subnetree-topology-${date}.png`
      link.href = dataUrl
      link.click()
      toast.success('Topology exported as PNG')
    } catch {
      toast.error('Failed to export topology')
    } finally {
      setExporting(false)
    }
  }

  const handleSave = () => {
    const name = saveName.trim()
    if (!name) return
    onSaveLayout(name)
    setSaveName('')
    toast.success(`Layout "${name}" saved`)
  }

  const handleDelete = (id: string, name: string) => {
    onDeleteLayout(id)
    toast.success(`Layout "${name}" deleted`)
  }

  return (
    <div
      className="flex items-center gap-1 rounded-lg px-2 py-1.5 shadow-md"
      style={{
        backgroundColor: 'var(--nv-bg-card)',
        border: '1px solid var(--nv-border-default)',
        backdropFilter: 'blur(8px)',
      }}
    >
      {/* Layout selector */}
      {layoutOptions.map(({ value, label, Icon }) => (
        <button
          key={value}
          onClick={() => onLayoutChange(value)}
          title={`${label} layout`}
          className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-xs font-medium transition-colors"
          style={{
            backgroundColor: layout === value ? 'var(--nv-bg-active)' : 'transparent',
            color: layout === value ? 'var(--nv-text-accent)' : 'var(--nv-text-secondary)',
          }}
        >
          <Icon className="h-3.5 w-3.5" />
          <span className="hidden sm:inline">{label}</span>
        </button>
      ))}

      <Separator />

      {/* Zoom controls */}
      <button
        onClick={() => zoomIn({ duration: 200 })}
        title="Zoom in"
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
        style={{ color: 'var(--nv-text-secondary)' }}
      >
        <ZoomIn className="h-4 w-4" />
      </button>
      <button
        onClick={() => zoomOut({ duration: 200 })}
        title="Zoom out"
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
        style={{ color: 'var(--nv-text-secondary)' }}
      >
        <ZoomOut className="h-4 w-4" />
      </button>
      <button
        onClick={() => fitView({ duration: 300, padding: 0.2 })}
        title="Fit to view"
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
        style={{ color: 'var(--nv-text-secondary)' }}
      >
        <Maximize className="h-4 w-4" />
      </button>

      <Separator />

      {/* Utilization toggle */}
      <button
        onClick={onUtilizationToggle}
        title={showUtilization ? 'Hide utilization overlay' : 'Show utilization overlay'}
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
        style={{
          color: showUtilization ? 'var(--nv-text-accent)' : 'var(--nv-text-secondary)',
        }}
      >
        <Activity className="h-4 w-4" />
      </button>

      {/* Minimap toggle */}
      <button
        onClick={onMinimapToggle}
        title={showMinimap ? 'Hide minimap' : 'Show minimap'}
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
        style={{
          color: showMinimap ? 'var(--nv-text-accent)' : 'var(--nv-text-secondary)',
        }}
      >
        <Map className="h-4 w-4" />
      </button>

      <Separator />

      {/* Saved layouts dropdown */}
      <div ref={layoutsRef} className="relative">
        <button
          onClick={() => setLayoutsOpen((prev) => !prev)}
          title="Saved layouts"
          className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)]"
          style={{
            color: layoutsOpen ? 'var(--nv-text-accent)' : 'var(--nv-text-secondary)',
          }}
        >
          <FolderOpen className="h-4 w-4" />
        </button>
        {layoutsOpen && (
          <div
            className="absolute top-full right-0 mt-2 w-64 rounded-lg p-3 shadow-lg z-50"
            style={{
              backgroundColor: 'var(--nv-bg-card)',
              border: '1px solid var(--nv-border-default)',
            }}
          >
            {/* Save current layout */}
            <div className="flex gap-1.5 mb-2">
              <input
                type="text"
                value={saveName}
                onChange={(e) => setSaveName(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleSave()}
                placeholder="Layout name..."
                className="flex-1 rounded-md px-2 py-1 text-xs"
                style={{
                  backgroundColor: 'var(--nv-bg-surface)',
                  border: '1px solid var(--nv-border-subtle)',
                  color: 'var(--nv-text-primary)',
                }}
              />
              <button
                onClick={handleSave}
                disabled={!saveName.trim()}
                title="Save layout"
                className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)] disabled:opacity-40"
                style={{ color: 'var(--nv-text-secondary)' }}
              >
                <Save className="h-3.5 w-3.5" />
              </button>
            </div>

            {/* List of saved layouts */}
            {savedLayouts.length === 0 ? (
              <p
                className="text-[10px] text-center py-2"
                style={{ color: 'var(--nv-text-secondary)' }}
              >
                No saved layouts
              </p>
            ) : (
              <ul className="space-y-1 max-h-48 overflow-y-auto">
                {savedLayouts.map((sl) => (
                  <li
                    key={sl.id}
                    className="flex items-center gap-1 rounded-md px-2 py-1 text-xs group"
                    style={{ color: 'var(--nv-text-primary)' }}
                  >
                    <button
                      onClick={() => {
                        onLoadLayout(sl.id)
                        setLayoutsOpen(false)
                        toast.success(`Layout "${sl.name}" applied`)
                      }}
                      className="flex-1 text-left truncate hover:underline"
                      title={`Load "${sl.name}"`}
                    >
                      {sl.name}
                    </button>
                    <span
                      className="text-[10px] flex-shrink-0"
                      style={{ color: 'var(--nv-text-secondary)' }}
                    >
                      {new Date(sl.createdAt).toLocaleDateString()}
                    </span>
                    <button
                      onClick={() => handleDelete(sl.id, sl.name)}
                      title={`Delete "${sl.name}"`}
                      className="opacity-0 group-hover:opacity-100 rounded p-0.5 transition-opacity hover:bg-[var(--nv-bg-hover)]"
                      style={{ color: 'var(--nv-text-secondary)' }}
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>

      {/* Export PNG */}
      <button
        onClick={handleExport}
        disabled={exporting}
        title="Export as PNG"
        className="rounded-md p-1.5 transition-colors hover:bg-[var(--nv-bg-hover)] disabled:opacity-50"
        style={{ color: 'var(--nv-text-secondary)' }}
      >
        <Download className="h-4 w-4" />
      </button>
    </div>
  )
})
