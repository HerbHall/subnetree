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
  type DeviceNodeType,
} from '@/components/topology/device-node'
import {
  TopologyEdge as TopologyEdgeComponent,
  type TopologyEdgeType,
} from '@/components/topology/topology-edge'
import {
  TopologyToolbar,
  type LayoutAlgorithm,
} from '@/components/topology/topology-toolbar'
import { TopologyFilters } from '@/components/topology/topology-filters'
import { NodeDetailPanel } from '@/components/topology/node-detail-panel'
import { EdgeInfoPopover } from '@/components/topology/edge-info-popover'
import type {
  TopologyNode as APINode,
  TopologyEdge as APIEdge,
  DeviceStatus,
  DeviceType,
} from '@/api/types'

// Register custom node & edge types outside the component to avoid re-creation
const nodeTypes: NodeTypes = { device: DeviceNode }
const edgeTypes: EdgeTypes = { topology: TopologyEdgeComponent }

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

/**
 * Simple hierarchical (tree) layout via BFS.
 * Root nodes: routers/switches/firewalls, or nodes with the most connections.
 * Layers are assigned by BFS distance from roots.
 */
function hierarchicalLayout(
  nodes: APINode[],
  edges: APIEdge[]
): DeviceNodeType[] {
  if (nodes.length === 0) return []

  // Build adjacency list
  const adjacency = new Map<string, string[]>()
  for (const n of nodes) {
    adjacency.set(n.id, [])
  }
  for (const e of edges) {
    adjacency.get(e.source)?.push(e.target)
    adjacency.get(e.target)?.push(e.source)
  }

  // Find root nodes: routers, switches, firewalls first
  const infraTypes = new Set(['router', 'switch', 'firewall', 'access_point'])
  const roots = nodes.filter((n) => infraTypes.has(n.device_type))

  // If no infrastructure nodes, pick the most connected node
  if (roots.length === 0) {
    const sorted = [...nodes].sort(
      (a, b) =>
        (adjacency.get(b.id)?.length ?? 0) -
        (adjacency.get(a.id)?.length ?? 0)
    )
    roots.push(sorted[0])
  }

  // BFS to assign layers
  const layers = new Map<string, number>()
  const queue: string[] = []
  for (const r of roots) {
    if (!layers.has(r.id)) {
      layers.set(r.id, 0)
      queue.push(r.id)
    }
  }

  let head = 0
  while (head < queue.length) {
    const current = queue[head++]
    const currentLayer = layers.get(current)!
    for (const neighbor of adjacency.get(current) ?? []) {
      if (!layers.has(neighbor)) {
        layers.set(neighbor, currentLayer + 1)
        queue.push(neighbor)
      }
    }
  }

  // Assign layers to any disconnected nodes
  for (const n of nodes) {
    if (!layers.has(n.id)) {
      layers.set(n.id, 0)
    }
  }

  // Group nodes by layer
  const layerGroups = new Map<number, APINode[]>()
  for (const n of nodes) {
    const layer = layers.get(n.id) ?? 0
    if (!layerGroups.has(layer)) layerGroups.set(layer, [])
    layerGroups.get(layer)!.push(n)
  }

  const layerSpacing = 150
  const nodeSpacing = 180

  return nodes.map((node): DeviceNodeType => {
    const layer = layers.get(node.id) ?? 0
    const group = layerGroups.get(layer)!
    const indexInLayer = group.indexOf(node)
    const layerWidth = (group.length - 1) * nodeSpacing
    const x = 400 - layerWidth / 2 + indexInLayer * nodeSpacing
    const y = 100 + layer * layerSpacing

    return {
      id: node.id,
      type: 'device',
      position: { x, y },
      data: {
        label: node.label,
        deviceType: node.device_type,
        status: node.status,
        ip: node.ip_addresses?.[0] || 'No IP',
      },
    }
  })
}

/**
 * Grid layout: arrange nodes in an even grid, grouped by device type.
 */
function gridLayout(nodes: APINode[]): DeviceNodeType[] {
  if (nodes.length === 0) return []

  // Sort by device type then label for consistent ordering
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

/** Convert API edges to React Flow edge format. */
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

/** Apply a layout algorithm to nodes. */
function applyLayout(
  algorithm: LayoutAlgorithm,
  nodes: APINode[],
  edges: APIEdge[]
): DeviceNodeType[] {
  switch (algorithm) {
    case 'hierarchical':
      return hierarchicalLayout(nodes, edges)
    case 'grid':
      return gridLayout(nodes)
    case 'circular':
    default:
      return circularLayout(nodes)
  }
}

/** Check if a topology node matches a search query. */
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
  const [showMinimap, setShowMinimap] = useState(false)

  // Filter state
  const [statusFilter, setStatusFilter] = useState<Set<DeviceStatus>>(
    () => new Set(ALL_STATUSES)
  )
  const [typeFilter, setTypeFilter] = useState<Set<DeviceType>>(() => new Set<DeviceType>())
  const [typeFilterInitialized, setTypeFilterInitialized] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')

  // Selection state
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [selectedEdge, setSelectedEdge] = useState<SelectedEdgeState | null>(null)

  const {
    data: topology,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['topology'],
    queryFn: getTopology,
  })

  // Initialize type filter when data loads (all types enabled by default)
  const presentTypes = useMemo(() => {
    if (!topology) return []
    const types = new Set<DeviceType>()
    for (const n of topology.nodes) {
      types.add(n.device_type)
    }
    return Array.from(types).sort()
  }, [topology])

  // Initialize type filter to include all present types when data first loads
  if (presentTypes.length > 0 && !typeFilterInitialized) {
    setTypeFilter(new Set(presentTypes))
    setTypeFilterInitialized(true)
  }

  // Build a lookup of API nodes by id for filter/search checks
  const apiNodeMap = useMemo(() => {
    if (!topology) return new Map<string, APINode>()
    const map = new Map<string, APINode>()
    for (const n of topology.nodes) {
      map.set(n.id, n)
    }
    return map
  }, [topology])

  // Build a lookup of API edges by id
  const apiEdgeMap = useMemo(() => {
    if (!topology) return new Map<string, APIEdge>()
    const map = new Map<string, APIEdge>()
    for (const e of topology.edges) {
      map.set(e.id, e)
    }
    return map
  }, [topology])

  // Resolve the selected API node for the detail panel
  const selectedNode = useMemo(() => {
    if (!selectedNodeId) return null
    return apiNodeMap.get(selectedNodeId) ?? null
  }, [selectedNodeId, apiNodeMap])

  // Determine which node IDs are hidden by filters
  const hiddenNodeIds = useMemo(() => {
    const hidden = new Set<string>()
    for (const [id, node] of apiNodeMap) {
      if (!statusFilter.has(node.status)) {
        hidden.add(id)
      } else if (!typeFilter.has(node.device_type)) {
        hidden.add(id)
      }
    }
    return hidden
  }, [apiNodeMap, statusFilter, typeFilter])

  // Determine which node IDs match the search
  const searchMatchIds = useMemo(() => {
    if (!searchQuery.trim()) return null // null = no search active
    const matches = new Set<string>()
    for (const [id, node] of apiNodeMap) {
      if (nodeMatchesSearch(node, searchQuery.trim())) {
        matches.add(id)
      }
    }
    return matches
  }, [apiNodeMap, searchQuery])

  // Transform API data to React Flow format using the selected layout
  const layoutNodes = useMemo<DeviceNodeType[]>(
    () =>
      topology ? applyLayout(layout, topology.nodes, topology.edges) : [],
    [topology, layout]
  )

  // Apply filter/search state to nodes
  const initialNodes = useMemo<DeviceNodeType[]>(() => {
    return layoutNodes.map((node) => ({
      ...node,
      hidden: hiddenNodeIds.has(node.id),
      data: {
        ...node.data,
        highlighted: searchMatchIds !== null && searchMatchIds.has(node.id),
        dimmed: searchMatchIds !== null && !searchMatchIds.has(node.id),
      },
    }))
  }, [layoutNodes, hiddenNodeIds, searchMatchIds])

  // Apply filter state to edges (hide if either endpoint is hidden)
  const initialEdges = useMemo<TopologyEdgeType[]>(() => {
    if (!topology) return []
    return toFlowEdges(topology.edges).map((edge) => ({
      ...edge,
      hidden: hiddenNodeIds.has(edge.source) || hiddenNodeIds.has(edge.target),
    }))
  }, [topology, hiddenNodeIds])

  const visibleCount = useMemo(
    () => initialNodes.filter((n) => !n.hidden).length,
    [initialNodes]
  )

  // React Flow state -- allows drag-to-reposition
  const [nodes, setNodes, onNodesChangeRaw] = useNodesState<DeviceNodeType>(initialNodes)
  const [edges, setEdges, onEdgesChangeRaw] = useEdgesState<TopologyEdgeType>(initialEdges)

  // Sync React Flow state when layout/filters/data change
  useEffect(() => {
    setNodes(initialNodes)
  }, [initialNodes, setNodes])

  useEffect(() => {
    setEdges(initialEdges)
  }, [initialEdges, setEdges])

  const onNodesChange: OnNodesChange<DeviceNodeType> = useCallback(
    (changes) => onNodesChangeRaw(changes),
    [onNodesChangeRaw]
  )

  const onEdgesChange: OnEdgesChange<TopologyEdgeType> = useCallback(
    (changes) => onEdgesChangeRaw(changes),
    [onEdgesChangeRaw]
  )

  const handleLayoutChange = useCallback((newLayout: LayoutAlgorithm) => {
    setLayout(newLayout)
  }, [])

  const handleMinimapToggle = useCallback(() => {
    setShowMinimap((prev) => !prev)
  }, [])

  const handleStatusFilterChange = useCallback((status: DeviceStatus) => {
    setStatusFilter((prev) => {
      const next = new Set(prev)
      if (next.has(status)) {
        next.delete(status)
      } else {
        next.add(status)
      }
      return next
    })
  }, [])

  const handleTypeFilterChange = useCallback((type: DeviceType) => {
    setTypeFilter((prev) => {
      const next = new Set(prev)
      if (next.has(type)) {
        next.delete(type)
      } else {
        next.add(type)
      }
      return next
    })
  }, [])

  const handleSearchChange = useCallback((query: string) => {
    setSearchQuery(query)
  }, [])

  const handleFilterReset = useCallback(() => {
    setStatusFilter(new Set(ALL_STATUSES))
    setTypeFilter(new Set(presentTypes))
    setSearchQuery('')
  }, [presentTypes])

  // Node click: show detail panel
  const handleNodeClick: NodeMouseHandler<DeviceNodeType> = useCallback(
    (_event, node) => {
      setSelectedNodeId(node.id)
      setSelectedEdge(null)
    },
    []
  )

  // Edge click: show info popover near click position
  const handleEdgeClick: EdgeMouseHandler<TopologyEdgeType> = useCallback(
    (event, edge) => {
      const apiEdge = apiEdgeMap.get(edge.id)
      const sourceNode = apiNodeMap.get(edge.source)
      const targetNode = apiNodeMap.get(edge.target)

      setSelectedEdge({
        id: edge.id,
        sourceLabel: sourceNode?.label || edge.source,
        targetLabel: targetNode?.label || edge.target,
        linkType: apiEdge?.link_type || edge.data?.linkType || '',
        speed: apiEdge?.speed,
        position: { x: event.clientX, y: event.clientY },
      })
      setSelectedNodeId(null)
    },
    [apiEdgeMap, apiNodeMap]
  )

  // Pane click: dismiss all selections
  const handlePaneClick = useCallback(() => {
    setSelectedNodeId(null)
    setSelectedEdge(null)
  }, [])

  const handleNodeDetailClose = useCallback(() => {
    setSelectedNodeId(null)
  }, [])

  const handleEdgePopoverDismiss = useCallback(() => {
    setSelectedEdge(null)
  }, [])

  // Ref for the React Flow viewport container (used for PNG export)
  const flowRef = useRef<HTMLDivElement>(null)

  // Loading state
  if (isLoading) {
    return (
      <div className="h-[calc(100vh-4rem)] w-full relative rounded-lg border bg-muted/10">
        {/* Toolbar skeleton */}
        <div className="absolute top-4 left-1/2 -translate-x-1/2 z-10">
          <Skeleton className="h-10 w-64 rounded-lg" />
        </div>
        {/* Filter panel skeleton */}
        <div className="absolute top-4 left-4 z-10 space-y-2">
          <Skeleton className="h-8 w-48 rounded-lg" />
          <Skeleton className="h-6 w-36 rounded-lg" />
        </div>
        {/* Topology node skeletons arranged in a circle pattern */}
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="relative w-[400px] h-[400px]">
            {[...Array(6)].map((_, i) => {
              const angle = (2 * Math.PI * i) / 6 - Math.PI / 2
              const x = 175 + 150 * Math.cos(angle)
              const y = 175 + 150 * Math.sin(angle)
              return (
                <Skeleton
                  key={i}
                  className="absolute h-12 w-12 rounded-lg"
                  style={{ left: `${x}px`, top: `${y}px` }}
                />
              )
            })}
            {/* Center loading indicator */}
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

  // Error state
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

  // Empty state
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
            <Button asChild>
              <a href="/dashboard">Scan Network</a>
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div ref={flowRef} className="h-[calc(100vh-4rem)] w-full relative">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={handleNodeClick}
        onEdgeClick={handleEdgeClick}
        onPaneClick={handlePaneClick}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        minZoom={0.1}
        maxZoom={4}
        fitView
        fitViewOptions={{ padding: 0.2, maxZoom: 1.5 }}
        proOptions={{ hideAttribution: true }}
        className="rounded-lg"
        style={{ backgroundColor: 'var(--nv-bg-surface)' }}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          color="var(--nv-border-subtle)"
        />
        <Controls
          showInteractive={false}
          className="!bg-[var(--nv-bg-card)] !border-[var(--nv-border-default)] !shadow-md [&>button]:!bg-[var(--nv-bg-card)] [&>button]:!border-[var(--nv-border-subtle)] [&>button]:!fill-[var(--nv-text-secondary)] [&>button:hover]:!bg-[var(--nv-bg-hover)]"
        />
        {showMinimap && (
          <MiniMap
            nodeStrokeWidth={3}
            style={{
              backgroundColor: 'var(--nv-bg-card)',
              border: '1px solid var(--nv-border-default)',
            }}
            maskColor="rgba(0, 0, 0, 0.3)"
          />
        )}
        <Panel position="top-center">
          <TopologyToolbar
            layout={layout}
            onLayoutChange={handleLayoutChange}
            showMinimap={showMinimap}
            onMinimapToggle={handleMinimapToggle}
            flowRef={flowRef}
          />
        </Panel>
        <Panel position="top-left">
          <TopologyFilters
            nodes={topology.nodes}
            statusFilter={statusFilter}
            onStatusFilterChange={handleStatusFilterChange}
            typeFilter={typeFilter}
            onTypeFilterChange={handleTypeFilterChange}
            searchQuery={searchQuery}
            onSearchChange={handleSearchChange}
            onReset={handleFilterReset}
            visibleCount={visibleCount}
          />
        </Panel>
      </ReactFlow>

      {/* Node detail sidebar */}
      {selectedNode && (
        <NodeDetailPanel node={selectedNode} onClose={handleNodeDetailClose} />
      )}

      {/* Edge info popover */}
      {selectedEdge && (
        <EdgeInfoPopover
          edge={selectedEdge}
          position={selectedEdge.position}
          onDismiss={handleEdgePopoverDismiss}
        />
      )}
    </div>
  )
}
