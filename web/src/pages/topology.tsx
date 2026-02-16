import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  Panel,
  type NodeTypes,
  type EdgeTypes,
  type Edge,
  type Node,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  type OnNodesChange,
  type OnEdgesChange,
  type NodeMouseHandler,
  type EdgeMouseHandler,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Loader2, AlertCircle, Network, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { getTopology } from '@/api/devices'
import {
  DeviceNode,
  type DeviceNodeData,
  type DeviceNodeType,
} from '@/components/topology/device-node'
import {
  TopologyEdge as TopologyEdgeComponent,
  type TopologyEdgeType,
} from '@/components/topology/topology-edge'
import { UtilizationEdge } from '@/components/topology/utilization-edge'
import {
  TopologyToolbar,
  type LayoutAlgorithm,
} from '@/components/topology/topology-toolbar'
import {
  elkLayout,
  type ElkDirection,
} from '@/components/topology/elk-layout'
import { TopologyFilters } from '@/components/topology/topology-filters'
import { TopologyViewTabs } from '@/components/topology/topology-view-tabs'
import { GroupNode } from '@/components/topology/group-node'
import { NodeDetailPanel } from '@/components/topology/node-detail-panel'
import { EdgeInfoPopover } from '@/components/topology/edge-info-popover'
import {
  saveLayout as saveLayoutToStorage,
  loadLayout as loadLayoutFromStorage,
  listLayouts,
  deleteLayout as deleteLayoutFromStorage,
  applyLayout as applyLayoutPositions,
  type SavedLayout,
} from '@/components/topology/layout-storage'
import {
  type ViewMode,
  groupByType,
  groupByStatus,
  groupBySubnet,
  layoutGroups,
  type GroupNodeData,
} from '@/lib/topology-grouping'
import type {
  TopologyNode as APINode,
  TopologyEdge as APIEdge,
  DeviceStatus,
  DeviceType,
} from '@/api/types'

// Register custom node & edge types outside the component to avoid re-creation
const nodeTypes: NodeTypes = { device: DeviceNode, group: GroupNode }
const defaultEdgeTypes: EdgeTypes = { topology: TopologyEdgeComponent }
const utilizationEdgeTypes: EdgeTypes = {
  topology: TopologyEdgeComponent,
  utilization: UtilizationEdge,
}

const ALL_STATUSES: DeviceStatus[] = ['online', 'offline', 'degraded', 'unknown']

// Selected edge state includes position for the popover and resolved labels
interface SelectedEdgeState {
  id: string
  sourceLabel: string
  targetLabel: string
  linkType: string
  speed?: number
  position: { x: number; y: number }
}

// ---------------------------------------------------------------------------
// Layout algorithms
// ---------------------------------------------------------------------------

/** Arrange nodes in a circular layout. */
function circularLayout(nodes: APINode[]): DeviceNodeType[] {
  const count = nodes.length
  if (count === 0) return []

  const centerX = 400
  const centerY = 300
  const radius = Math.max(150, count * 40)

  return nodes.map((node, i): DeviceNodeType => {
    const angle = (2 * Math.PI * i) / count - Math.PI / 2
    return {
      id: node.id,
      type: 'device',
      position:
        count === 1
          ? { x: centerX, y: centerY }
          : {
              x: centerX + radius * Math.cos(angle),
              y: centerY + radius * Math.sin(angle),
            },
      data: {
        label: node.label,
        deviceType: node.device_type,
        status: node.status,
        ip: node.ip_addresses?.[0] || 'No IP',
      },
    }
  })
}

/** Convert API nodes to flow nodes with temporary positions (used as fallback). */
function toFlowNodes(nodes: APINode[]): DeviceNodeType[] {
  return nodes.map((node): DeviceNodeType => ({
    id: node.id,
    type: 'device',
    position: { x: 0, y: 0 },
    data: {
      label: node.label,
      deviceType: node.device_type,
      status: node.status,
      ip: node.ip_addresses?.[0] || 'No IP',
    },
  }))
}

function gridLayout(nodes: APINode[]): DeviceNodeType[] {
  if (nodes.length === 0) return []

  const sorted = [...nodes].sort((a, b) => {
    const typeCmp = a.device_type.localeCompare(b.device_type)
    if (typeCmp !== 0) return typeCmp
    return a.label.localeCompare(b.label)
  })

  const cols = Math.max(1, Math.ceil(Math.sqrt(sorted.length)))
  const cellW = 200
  const cellH = 120
  const offsetX = 100
  const offsetY = 100

  return sorted.map((node, i): DeviceNodeType => ({
    id: node.id,
    type: 'device',
    position: {
      x: offsetX + (i % cols) * cellW,
      y: offsetY + Math.floor(i / cols) * cellH,
    },
    data: {
      label: node.label,
      deviceType: node.device_type,
      status: node.status,
      ip: node.ip_addresses?.[0] || 'No IP',
    },
  }))
}

function toFlowEdges(edges: APIEdge[]): TopologyEdgeType[] {
  return edges.map(
    (e): TopologyEdgeType => ({
      id: e.id,
      source: e.source,
      target: e.target,
      type: 'topology',
      data: { linkType: e.link_type },
    })
  )
}

/** Apply a synchronous layout algorithm. Hierarchical uses async elkjs separately. */
function applySyncLayout(algorithm: LayoutAlgorithm, nodes: APINode[]): DeviceNodeType[] {
  switch (algorithm) {
    case 'grid':
      return gridLayout(nodes)
    case 'circular':
    default:
      return circularLayout(nodes)
  }
}

function nodeMatchesSearch(node: APINode, query: string): boolean {
  const q = query.toLowerCase()
  if (node.label.toLowerCase().includes(q)) return true
  if (node.manufacturer?.toLowerCase().includes(q)) return true
  if (node.ip_addresses?.some((ip) => ip.toLowerCase().includes(q))) return true
  if (node.mac_address?.toLowerCase().includes(q)) return true
  return false
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function TopologyPage() {
  const [layout, setLayout] = useState<LayoutAlgorithm>('circular')
  const [direction, setDirection] = useState<ElkDirection>('DOWN')
  const [showMinimap, setShowMinimap] = useState(false)
  const [showUtilization, setShowUtilization] = useState(false)
  const [savedLayouts, setSavedLayouts] = useState<SavedLayout[]>([])
  const [viewMode, setViewMode] = useState<ViewMode>('all')

  // Load saved layouts from API on mount.
  useEffect(() => {
    listLayouts().then(setSavedLayouts)
  }, [])

  const edgeTypes = showUtilization ? utilizationEdgeTypes : defaultEdgeTypes

  const [statusFilter, setStatusFilter] = useState<Set<DeviceStatus>>(
    () => new Set(ALL_STATUSES)
  )
  const [typeFilter, setTypeFilter] = useState<Set<DeviceType>>(() => new Set<DeviceType>())
  const [typeFilterInitialized, setTypeFilterInitialized] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')

  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [selectedEdge, setSelectedEdge] = useState<SelectedEdgeState | null>(null)

  const { data: topology, isLoading, error, refetch } = useQuery({
    queryKey: ['topology'],
    queryFn: getTopology,
  })

  const presentTypes = useMemo(() => {
    if (!topology) return []
    const types = new Set<DeviceType>()
    for (const n of topology.nodes) { types.add(n.device_type) }
    return Array.from(types).sort()
  }, [topology])

  if (presentTypes.length > 0 && !typeFilterInitialized) {
    setTypeFilter(new Set(presentTypes))
    setTypeFilterInitialized(true)
  }

  const apiNodeMap = useMemo(() => {
    if (!topology) return new Map<string, APINode>()
    const map = new Map<string, APINode>()
    for (const n of topology.nodes) { map.set(n.id, n) }
    return map
  }, [topology])

  const apiEdgeMap = useMemo(() => {
    if (!topology) return new Map<string, APIEdge>()
    const map = new Map<string, APIEdge>()
    for (const e of topology.edges) { map.set(e.id, e) }
    return map
  }, [topology])

  const selectedNode = useMemo(() => {
    if (!selectedNodeId) return null
    return apiNodeMap.get(selectedNodeId) ?? null
  }, [selectedNodeId, apiNodeMap])

  const hiddenNodeIds = useMemo(() => {
    const hidden = new Set<string>()
    for (const [id, node] of apiNodeMap) {
      if (!statusFilter.has(node.status)) { hidden.add(id) }
      else if (!typeFilter.has(node.device_type)) { hidden.add(id) }
    }
    return hidden
  }, [apiNodeMap, statusFilter, typeFilter])

  const searchMatchIds = useMemo(() => {
    if (!searchQuery.trim()) return null
    const matches = new Set<string>()
    for (const [id, node] of apiNodeMap) {
      if (nodeMatchesSearch(node, searchQuery.trim())) { matches.add(id) }
    }
    return matches
  }, [apiNodeMap, searchQuery])

  // Synchronous layouts (circular / grid) computed via useMemo
  const syncLayoutNodes = useMemo<DeviceNodeType[]>(() => {
    if (!topology || topology.nodes.length === 0) return []
    if (layout === 'hierarchical') return []
    return applySyncLayout(layout, topology.nodes)
  }, [topology, layout])

  // Async elkjs layout for hierarchical mode
  const [elkNodes, setElkNodes] = useState<DeviceNodeType[]>([])

  useEffect(() => {
    if (!topology || topology.nodes.length === 0 || layout !== 'hierarchical') {
      return
    }
    const flowNodes = toFlowNodes(topology.nodes)
    const flowEdges = toFlowEdges(topology.edges)
    let cancelled = false
    elkLayout(flowNodes, flowEdges, direction).then((positioned) => {
      if (!cancelled) setElkNodes(positioned)
    })
    return () => { cancelled = true }
  }, [topology, layout, direction])

  const layoutNodes = layout === 'hierarchical' ? elkNodes : syncLayoutNodes

  // Compute grouped layout when a grouped view mode is active
  const groupedResult = useMemo(() => {
    if (viewMode === 'all' || !topology || topology.nodes.length === 0) return null
    const groupFn =
      viewMode === 'by-type' ? groupByType
      : viewMode === 'by-status' ? groupByStatus
      : groupBySubnet
    const groups = groupFn(topology.nodes)
    return layoutGroups(groups, topology.nodes, viewMode)
  }, [topology, viewMode])

  type AnyNodeType = Node<DeviceNodeData | GroupNodeData>

  const initialNodes = useMemo<AnyNodeType[]>(() => {
    if (viewMode !== 'all' && groupedResult) {
      // Grouped mode: group container nodes first, then device children
      const allNodes: AnyNodeType[] = [
        ...groupedResult.groupNodes.map((gn) => ({
          ...gn,
          // Group nodes are visible if any child is visible
          hidden: gn.data.count === 0,
        })),
        ...groupedResult.deviceNodes.map((dn) => ({
          ...dn,
          hidden: hiddenNodeIds.has(dn.id),
          data: {
            ...dn.data,
            highlighted: searchMatchIds !== null && searchMatchIds.has(dn.id),
            dimmed: searchMatchIds !== null && !searchMatchIds.has(dn.id),
          },
        })),
      ]
      return allNodes
    }
    // Flat "all devices" mode -- original behavior
    return layoutNodes.map((node) => ({
      ...node,
      hidden: hiddenNodeIds.has(node.id),
      data: {
        ...node.data,
        highlighted: searchMatchIds !== null && searchMatchIds.has(node.id),
        dimmed: searchMatchIds !== null && !searchMatchIds.has(node.id),
      },
    }))
  }, [viewMode, groupedResult, layoutNodes, hiddenNodeIds, searchMatchIds])

  const initialEdges = useMemo<Edge[]>(() => {
    if (!topology) return []
    if (showUtilization) {
      return topology.edges.map((e): Edge => {
        const hash = e.id.split('').reduce((acc, ch) => acc + ch.charCodeAt(0), 0)
        const utilization = e.speed
          ? Math.min(100, Math.round((e.speed / 10000) * 100))
          : hash % 101
        return {
          id: e.id, source: e.source, target: e.target, type: 'utilization',
          data: { utilization, speed: e.speed, linkType: e.link_type },
          hidden: hiddenNodeIds.has(e.source) || hiddenNodeIds.has(e.target),
        }
      })
    }
    return toFlowEdges(topology.edges).map((edge) => ({
      ...edge,
      hidden: hiddenNodeIds.has(edge.source) || hiddenNodeIds.has(edge.target),
    }))
  }, [topology, hiddenNodeIds, showUtilization])

  const visibleCount = useMemo(
    () => initialNodes.filter((n) => !n.hidden && n.type !== 'group').length,
    [initialNodes]
  )

  const [nodes, setNodes, onNodesChangeRaw] = useNodesState(initialNodes)
  const [edges, setEdges, onEdgesChangeRaw] = useEdgesState(initialEdges)

  useEffect(() => { setNodes(initialNodes) }, [initialNodes, setNodes])
  useEffect(() => { setEdges(initialEdges) }, [initialEdges, setEdges])

  const onNodesChange: OnNodesChange<AnyNodeType> = useCallback(
    (changes) => onNodesChangeRaw(changes), [onNodesChangeRaw]
  )
  const onEdgesChange: OnEdgesChange = useCallback(
    (changes) => onEdgesChangeRaw(changes), [onEdgesChangeRaw]
  )

  const handleLayoutChange = useCallback((newLayout: LayoutAlgorithm) => { setLayout(newLayout) }, [])
  const handleDirectionChange = useCallback((dir: ElkDirection) => { setDirection(dir) }, [])
  const handleViewModeChange = useCallback((mode: ViewMode) => { setViewMode(mode) }, [])
  const handleMinimapToggle = useCallback(() => { setShowMinimap((prev) => !prev) }, [])
  const handleUtilizationToggle = useCallback(() => { setShowUtilization((prev) => !prev) }, [])

  const handleSaveLayout = useCallback(async (name: string) => {
    await saveLayoutToStorage(name, nodes.map((n) => ({ id: n.id, position: n.position })))
    const updated = await listLayouts()
    setSavedLayouts(updated)
  }, [nodes])

  const handleLoadLayout = useCallback(async (id: string) => {
    const saved = await loadLayoutFromStorage(id)
    if (!saved) return
    setNodes((current) => applyLayoutPositions(current, saved))
  }, [setNodes])

  const handleDeleteLayout = useCallback(async (id: string) => {
    await deleteLayoutFromStorage(id)
    const updated = await listLayouts()
    setSavedLayouts(updated)
  }, [])

  const handleStatusFilterChange = useCallback((status: DeviceStatus) => {
    setStatusFilter((prev) => {
      const next = new Set(prev)
      if (next.has(status)) { next.delete(status) } else { next.add(status) }
      return next
    })
  }, [])

  const handleTypeFilterChange = useCallback((type: DeviceType) => {
    setTypeFilter((prev) => {
      const next = new Set(prev)
      if (next.has(type)) { next.delete(type) } else { next.add(type) }
      return next
    })
  }, [])

  const handleSearchChange = useCallback((query: string) => { setSearchQuery(query) }, [])

  const handleFilterReset = useCallback(() => {
    setStatusFilter(new Set(ALL_STATUSES))
    setTypeFilter(new Set(presentTypes))
    setSearchQuery('')
  }, [presentTypes])

  const handleNodeClick: NodeMouseHandler<AnyNodeType> = useCallback(
    (_event, node) => {
      // Only select device nodes for the detail panel, not group containers
      if (node.type === 'group') return
      setSelectedNodeId(node.id)
      setSelectedEdge(null)
    }, []
  )

  const handleEdgeClick: EdgeMouseHandler = useCallback(
    (event, edge) => {
      const apiEdge = apiEdgeMap.get(edge.id)
      const sourceNode = apiNodeMap.get(edge.source)
      const targetNode = apiNodeMap.get(edge.target)
      const edgeData = edge.data as Record<string, unknown> | undefined
      setSelectedEdge({
        id: edge.id,
        sourceLabel: sourceNode?.label || edge.source,
        targetLabel: targetNode?.label || edge.target,
        linkType: apiEdge?.link_type || (edgeData?.linkType as string) || '',
        speed: apiEdge?.speed,
        position: { x: event.clientX, y: event.clientY },
      })
      setSelectedNodeId(null)
    }, [apiEdgeMap, apiNodeMap]
  )

  const handlePaneClick = useCallback(() => { setSelectedNodeId(null); setSelectedEdge(null) }, [])
  const handleNodeDetailClose = useCallback(() => { setSelectedNodeId(null) }, [])
  const handleEdgePopoverDismiss = useCallback(() => { setSelectedEdge(null) }, [])

  const flowRef = useRef<HTMLDivElement>(null)

  if (isLoading) {
    return (
      <div className="h-[calc(100vh-4rem)] w-full relative rounded-lg border bg-muted/10">
        <div className="absolute top-4 left-1/2 -translate-x-1/2 z-10">
          <Skeleton className="h-10 w-64 rounded-lg" />
        </div>
        <div className="absolute top-4 left-4 z-10 space-y-2">
          <Skeleton className="h-8 w-48 rounded-lg" />
          <Skeleton className="h-6 w-36 rounded-lg" />
        </div>
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="relative w-[400px] h-[400px]">
            {[...Array(6)].map((_, i) => {
              const angle = (2 * Math.PI * i) / 6 - Math.PI / 2
              const x = 175 + 150 * Math.cos(angle)
              const y = 175 + 150 * Math.sin(angle)
              return (
                <Skeleton key={i} className="absolute h-12 w-12 rounded-lg" style={{ left: `${x}px`, top: `${y}px` }} />
              )
            })}
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="flex flex-col items-center gap-2">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                <p className="text-xs text-muted-foreground">Loading topology...</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-[calc(100vh-4rem)] items-center justify-center">
        <Card className="max-w-md border-red-500/50">
          <CardContent className="p-6 text-center">
            <AlertCircle className="h-12 w-12 mx-auto text-red-400 mb-4" />
            <h2 className="text-lg font-semibold mb-2">Failed to load topology</h2>
            <p className="text-sm text-muted-foreground mb-4">
              {error instanceof Error ? error.message : 'An unexpected error occurred'}
            </p>
            <Button variant="outline" className="gap-2" onClick={() => refetch()}>
              <RefreshCw className="h-4 w-4" />
              Try Again
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!topology || topology.nodes.length === 0) {
    return (
      <div className="flex h-[calc(100vh-4rem)] items-center justify-center">
        <Card className="max-w-md">
          <CardContent className="p-6 text-center">
            <Network className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
            <h2 className="text-lg font-semibold mb-2">No devices discovered yet</h2>
            <p className="text-sm text-muted-foreground mb-4">
              Run a network scan to discover devices and build your topology map.
            </p>
            <Button asChild><a href="/dashboard">Scan Network</a></Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div ref={flowRef} className="h-[calc(100vh-4rem)] w-full relative">
      <ReactFlow
        nodes={nodes} edges={edges}
        onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
        onNodeClick={handleNodeClick} onEdgeClick={handleEdgeClick} onPaneClick={handlePaneClick}
        nodeTypes={nodeTypes} edgeTypes={edgeTypes}
        minZoom={0.1} maxZoom={4} fitView
        fitViewOptions={{ padding: 0.2, maxZoom: 1.5 }}
        proOptions={{ hideAttribution: true }}
        className="rounded-lg" style={{ backgroundColor: 'var(--nv-bg-surface)' }}
      >
        <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="var(--nv-border-subtle)" />
        <Controls showInteractive={false} className="!bg-[var(--nv-bg-card)] !border-[var(--nv-border-default)] !shadow-md [&>button]:!bg-[var(--nv-bg-card)] [&>button]:!border-[var(--nv-border-subtle)] [&>button]:!fill-[var(--nv-text-secondary)] [&>button:hover]:!bg-[var(--nv-bg-hover)]" />
        {showMinimap && (
          <MiniMap nodeStrokeWidth={3} style={{ backgroundColor: 'var(--nv-bg-card)', border: '1px solid var(--nv-border-default)' }} maskColor="rgba(0, 0, 0, 0.3)" />
        )}
        <Panel position="top-center">
          <div className="flex flex-col items-center gap-2">
            <TopologyViewTabs viewMode={viewMode} onViewModeChange={handleViewModeChange} />
            <TopologyToolbar
              layout={layout} onLayoutChange={handleLayoutChange}
              direction={direction} onDirectionChange={handleDirectionChange}
              showMinimap={showMinimap} onMinimapToggle={handleMinimapToggle} flowRef={flowRef}
              showUtilization={showUtilization} onUtilizationToggle={handleUtilizationToggle}
              savedLayouts={savedLayouts} onSaveLayout={handleSaveLayout}
              onLoadLayout={handleLoadLayout} onDeleteLayout={handleDeleteLayout}
            />
          </div>
        </Panel>
        <Panel position="top-left">
          <TopologyFilters
            nodes={topology.nodes} statusFilter={statusFilter} onStatusFilterChange={handleStatusFilterChange}
            typeFilter={typeFilter} onTypeFilterChange={handleTypeFilterChange}
            searchQuery={searchQuery} onSearchChange={handleSearchChange}
            onReset={handleFilterReset} visibleCount={visibleCount}
          />
        </Panel>
      </ReactFlow>
      {selectedNode && <NodeDetailPanel node={selectedNode} onClose={handleNodeDetailClose} />}
      {selectedEdge && <EdgeInfoPopover edge={selectedEdge} position={selectedEdge.position} onDismiss={handleEdgePopoverDismiss} />}
    </div>
  )
}
