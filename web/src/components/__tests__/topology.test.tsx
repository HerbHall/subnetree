import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import type { TopologyNode, TopologyEdge, TopologyGraph } from '@/api/types'

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

// Mock @xyflow/react before importing components that use it
vi.mock('@xyflow/react', () => ({
  ReactFlow: ({ children }: { children?: React.ReactNode }) => (
    <div data-testid="react-flow">{children}</div>
  ),
  ReactFlowProvider: ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  ),
  Background: () => null,
  Controls: () => null,
  MiniMap: () => null,
  Panel: ({ children }: { children?: React.ReactNode }) => (
    <div data-testid="react-flow-panel">{children}</div>
  ),
  BaseEdge: () => null,
  EdgeLabelRenderer: ({ children }: { children?: React.ReactNode }) => (
    <div>{children}</div>
  ),
  Handle: () => null,
  Position: { Top: 'top', Bottom: 'bottom', Left: 'left', Right: 'right' },
  MarkerType: { ArrowClosed: 'arrowclosed' },
  BackgroundVariant: { Dots: 'dots', Lines: 'lines', Cross: 'cross' },
  useReactFlow: () => ({
    fitView: vi.fn(),
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    getNodes: vi.fn(() => []),
  }),
  useNodesState: (init: unknown[]) => [init, vi.fn(), vi.fn()],
  useEdgesState: (init: unknown[]) => [init, vi.fn(), vi.fn()],
  getSmoothStepPath: () => ['M0,0 L100,100', 50, 50],
}))

// Mock html-to-image
vi.mock('html-to-image', () => ({
  toPng: vi.fn(() =>
    Promise.resolve('data:image/png;base64,mockImageData')
  ),
}))

// Mock sonner toast
vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

// Mock devices API
const mockGetTopology = vi.fn()
vi.mock('@/api/devices', () => ({
  getTopology: (...args: unknown[]) => mockGetTopology(...args),
  triggerScan: vi.fn(),
}))

// Import components after mocks are in place
import { TopologyPage } from '@/pages/topology'
import { DeviceNode } from '@/components/topology/device-node'
import { NodeDetailPanel } from '@/components/topology/node-detail-panel'
import { EdgeInfoPopover } from '@/components/topology/edge-info-popover'
import { TopologyToolbar } from '@/components/topology/topology-toolbar'
import type { DeviceNodeData } from '@/components/topology/device-node'
import type { NodeProps, Node } from '@xyflow/react'

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

function testTopologyNode(
  overrides: Partial<TopologyNode> & { id: string }
): TopologyNode {
  return {
    label: 'Test Device',
    device_type: 'unknown',
    status: 'online',
    ip_addresses: ['192.168.1.1'],
    mac_address: 'AA:BB:CC:DD:EE:FF',
    manufacturer: 'Test Manufacturer',
    ...overrides,
  }
}

function testTopologyEdge(
  overrides: Partial<TopologyEdge> & { id: string }
): TopologyEdge {
  return {
    source: 'node-1',
    target: 'node-2',
    link_type: 'ethernet',
    ...overrides,
  }
}

function testTopologyGraph(
  nodes: TopologyNode[] = [],
  edges: TopologyEdge[] = []
): TopologyGraph {
  return { nodes, edges }
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = createQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>
  )
}

type DeviceNodeType = Node<DeviceNodeData, 'device'>

function createDeviceNodeProps(
  overrides: Partial<DeviceNodeData> = {}
): NodeProps<DeviceNodeType> {
  const data: DeviceNodeData = {
    label: 'Test Device',
    deviceType: 'server',
    status: 'online',
    ip: '192.168.1.1',
    ...overrides,
  }
  return {
    id: 'test-node-1',
    type: 'device',
    data,
    selected: false,
    dragging: false,
    isConnectable: true,
    zIndex: 0,
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
    width: 140,
    height: 80,
    sourcePosition: undefined,
    targetPosition: undefined,
    dragHandle: undefined,
    parentId: undefined,
    deletable: true,
    selectable: true,
    draggable: true,
    measured: { width: 140, height: 80 },
  } as unknown as NodeProps<DeviceNodeType>
}

// ---------------------------------------------------------------------------
// DeviceNode Tests
// ---------------------------------------------------------------------------

describe('DeviceNode', () => {
  it('renders device label and IP address', () => {
    const props = createDeviceNodeProps({
      label: 'My Server',
      ip: '10.0.0.1',
    })
    render(<DeviceNode {...props} />)

    expect(screen.getByText('My Server')).toBeInTheDocument()
    expect(screen.getByText('10.0.0.1')).toBeInTheDocument()
  })

  it('renders "Unnamed" when label is empty', () => {
    const props = createDeviceNodeProps({ label: '' })
    render(<DeviceNode {...props} />)

    expect(screen.getByText('Unnamed')).toBeInTheDocument()
  })

  it('shows correct status color for online device', () => {
    const props = createDeviceNodeProps({ status: 'online' })
    const { container } = render(<DeviceNode {...props} />)

    // The status dot (h-2 w-2 rounded-full) should have the online color
    const statusDot = container.querySelector('.rounded-full')
    expect(statusDot).toBeTruthy()
    expect(statusDot?.getAttribute('style')).toContain(
      'background-color: var(--nv-topo-node-online)'
    )
  })

  it('shows correct status color for offline device', () => {
    const props = createDeviceNodeProps({ status: 'offline' })
    const { container } = render(<DeviceNode {...props} />)

    const statusDot = container.querySelector('.rounded-full')
    expect(statusDot?.getAttribute('style')).toContain(
      'background-color: var(--nv-topo-node-offline)'
    )
  })

  it('shows correct status color for degraded device', () => {
    const props = createDeviceNodeProps({ status: 'degraded' })
    const { container } = render(<DeviceNode {...props} />)

    const statusDot = container.querySelector('.rounded-full')
    expect(statusDot?.getAttribute('style')).toContain(
      'background-color: var(--nv-topo-node-degraded)'
    )
  })

  it('shows correct status color for unknown device', () => {
    const props = createDeviceNodeProps({ status: 'unknown' })
    const { container } = render(<DeviceNode {...props} />)

    const statusDot = container.querySelector('.rounded-full')
    expect(statusDot?.getAttribute('style')).toContain(
      'background-color: var(--nv-topo-node-unknown)'
    )
  })

  it('applies dimmed opacity when dimmed is true', () => {
    const props = createDeviceNodeProps({ dimmed: true })
    const { container } = render(<DeviceNode {...props} />)

    const nodeContainer = container.querySelector('.rounded-lg')
    expect(nodeContainer).toHaveStyle({ opacity: '0.35' })
  })

  it('applies full opacity when not dimmed', () => {
    const props = createDeviceNodeProps({ dimmed: false })
    const { container } = render(<DeviceNode {...props} />)

    const nodeContainer = container.querySelector('.rounded-lg')
    expect(nodeContainer).toHaveStyle({ opacity: '1' })
  })

  it('applies highlight glow when highlighted is true', () => {
    const props = createDeviceNodeProps({ highlighted: true })
    const { container } = render(<DeviceNode {...props} />)

    const nodeContainer = container.querySelector('.rounded-lg')
    const style = nodeContainer?.getAttribute('style') ?? ''
    expect(style).toContain('box-shadow')
    expect(style).toContain('rgba(74, 222, 128')
  })
})

// ---------------------------------------------------------------------------
// TopologyPage State Tests
// ---------------------------------------------------------------------------

describe('TopologyPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows loading state when data is loading', () => {
    // Return a promise that never resolves to keep it loading
    mockGetTopology.mockReturnValue(new Promise(() => {}))

    renderWithProviders(<TopologyPage />)

    expect(screen.getByText('Loading topology...')).toBeInTheDocument()
  })

  it('shows empty state when no devices exist', async () => {
    mockGetTopology.mockResolvedValue(testTopologyGraph())

    renderWithProviders(<TopologyPage />)

    expect(
      await screen.findByText('No devices discovered yet')
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Run a network scan to discover devices and build your topology map.'
      )
    ).toBeInTheDocument()
    expect(screen.getByText('Scan Network')).toBeInTheDocument()
  })

  it('shows empty state when topology has empty nodes array', async () => {
    mockGetTopology.mockResolvedValue({ nodes: [], edges: [] })

    renderWithProviders(<TopologyPage />)

    expect(
      await screen.findByText('No devices discovered yet')
    ).toBeInTheDocument()
  })

  it('shows error state with retry button on fetch error', async () => {
    mockGetTopology.mockRejectedValue(new Error('Network unreachable'))

    renderWithProviders(<TopologyPage />)

    expect(
      await screen.findByText('Failed to load topology')
    ).toBeInTheDocument()
    expect(screen.getByText('Network unreachable')).toBeInTheDocument()
    expect(screen.getByText('Retry')).toBeInTheDocument()
  })

  it('renders React Flow canvas when data is available', async () => {
    const graph = testTopologyGraph(
      [
        testTopologyNode({ id: 'node-1', label: 'Router' }),
        testTopologyNode({ id: 'node-2', label: 'Switch' }),
      ],
      [testTopologyEdge({ id: 'edge-1', source: 'node-1', target: 'node-2' })]
    )
    mockGetTopology.mockResolvedValue(graph)

    renderWithProviders(<TopologyPage />)

    expect(await screen.findByTestId('react-flow')).toBeInTheDocument()
  })

  it('scan network button links to dashboard', async () => {
    mockGetTopology.mockResolvedValue(testTopologyGraph())

    renderWithProviders(<TopologyPage />)

    const link = await screen.findByText('Scan Network')
    expect(link.closest('a')).toHaveAttribute('href', '/dashboard')
  })

  it('retry button calls refetch on error state', async () => {
    mockGetTopology.mockRejectedValueOnce(new Error('fail'))

    renderWithProviders(<TopologyPage />)

    const retryButton = await screen.findByText('Retry')
    expect(retryButton).toBeInTheDocument()

    // Clicking retry should trigger another fetch
    mockGetTopology.mockResolvedValueOnce(
      testTopologyGraph([testTopologyNode({ id: 'n1', label: 'Device' })])
    )

    const user = userEvent.setup()
    await user.click(retryButton)

    // After retry, mockGetTopology should have been called again
    expect(mockGetTopology).toHaveBeenCalledTimes(2)
  })
})

// ---------------------------------------------------------------------------
// NodeDetailPanel Tests
// ---------------------------------------------------------------------------

describe('NodeDetailPanel', () => {
  const baseNode = testTopologyNode({
    id: 'device-1',
    label: 'Test Server',
    device_type: 'server',
    status: 'online',
    ip_addresses: ['192.168.1.10', '10.0.0.5'],
    mac_address: 'AA:BB:CC:DD:EE:FF',
    manufacturer: 'Dell Inc.',
  })

  it('shows device info when a node is selected', () => {
    render(
      <MemoryRouter>
        <NodeDetailPanel node={baseNode} onClose={vi.fn()} />
      </MemoryRouter>
    )

    expect(screen.getByText('Device Details')).toBeInTheDocument()
    expect(screen.getByText('Test Server')).toBeInTheDocument()
    expect(screen.getByText('Server')).toBeInTheDocument()
    expect(screen.getByText('Online')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.10')).toBeInTheDocument()
    expect(screen.getByText('10.0.0.5')).toBeInTheDocument()
    expect(screen.getByText('AA:BB:CC:DD:EE:FF')).toBeInTheDocument()
    expect(screen.getByText('Dell Inc.')).toBeInTheDocument()
  })

  it('shows "View Details" link with correct href', () => {
    render(
      <MemoryRouter>
        <NodeDetailPanel node={baseNode} onClose={vi.fn()} />
      </MemoryRouter>
    )

    const viewDetailsLink = screen.getByText('View Details')
    expect(viewDetailsLink.closest('a')).toHaveAttribute(
      'href',
      '/devices/device-1'
    )
  })

  it('calls onClose when close button is clicked', async () => {
    const onClose = vi.fn()
    render(
      <MemoryRouter>
        <NodeDetailPanel node={baseNode} onClose={onClose} />
      </MemoryRouter>
    )

    const user = userEvent.setup()
    // The close button has an X icon. Find the button in the header.
    const header = screen.getByText('Device Details').closest('div')!
    const closeButton = within(header).getByRole('button')
    await user.click(closeButton)

    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('shows "No IP addresses" when node has no IPs', () => {
    const noIpNode = testTopologyNode({
      id: 'device-2',
      ip_addresses: [],
    })
    render(
      <MemoryRouter>
        <NodeDetailPanel node={noIpNode} onClose={vi.fn()} />
      </MemoryRouter>
    )

    expect(screen.getByText('No IP addresses')).toBeInTheDocument()
  })

  it('shows "Unknown" for MAC address when not provided', () => {
    const noMacNode = testTopologyNode({
      id: 'device-3',
      device_type: 'server',
      mac_address: '',
    })
    render(
      <MemoryRouter>
        <NodeDetailPanel node={noMacNode} onClose={vi.fn()} />
      </MemoryRouter>
    )

    // The MAC Address section should show "Unknown"
    const macLabel = screen.getByText('MAC Address')
    const macSection = macLabel.closest('div')!
    expect(within(macSection).getByText('Unknown')).toBeInTheDocument()
  })

  it('hides manufacturer section when not available', () => {
    const noManufNode = testTopologyNode({
      id: 'device-4',
      manufacturer: '',
    })
    render(
      <MemoryRouter>
        <NodeDetailPanel node={noManufNode} onClose={vi.fn()} />
      </MemoryRouter>
    )

    // The Manufacturer label should not appear
    expect(screen.queryByText('Manufacturer')).not.toBeInTheDocument()
  })

  it('shows correct type label for different device types', () => {
    const routerNode = testTopologyNode({
      id: 'device-5',
      device_type: 'router',
    })
    render(
      <MemoryRouter>
        <NodeDetailPanel node={routerNode} onClose={vi.fn()} />
      </MemoryRouter>
    )

    expect(screen.getByText('Router')).toBeInTheDocument()
  })
})

// ---------------------------------------------------------------------------
// EdgeInfoPopover Tests
// ---------------------------------------------------------------------------

describe('EdgeInfoPopover', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  const baseEdge = {
    id: 'edge-1',
    sourceLabel: 'Router A',
    targetLabel: 'Switch B',
    linkType: 'ethernet',
    speed: 1000,
  }
  const basePosition = { x: 200, y: 300 }

  it('shows source and target labels', () => {
    render(
      <EdgeInfoPopover
        edge={baseEdge}
        position={basePosition}
        onDismiss={vi.fn()}
      />
    )

    expect(screen.getByText('Router A')).toBeInTheDocument()
    expect(screen.getByText('Switch B')).toBeInTheDocument()
  })

  it('shows link type', () => {
    render(
      <EdgeInfoPopover
        edge={baseEdge}
        position={basePosition}
        onDismiss={vi.fn()}
      />
    )

    expect(screen.getByText('ethernet')).toBeInTheDocument()
  })

  it('formats speed in Gbps when >= 1000 Mbps', () => {
    render(
      <EdgeInfoPopover
        edge={baseEdge}
        position={basePosition}
        onDismiss={vi.fn()}
      />
    )

    expect(screen.getByText('1 Gbps')).toBeInTheDocument()
  })

  it('formats speed in Mbps when < 1000', () => {
    const slowEdge = { ...baseEdge, speed: 100 }
    render(
      <EdgeInfoPopover
        edge={slowEdge}
        position={basePosition}
        onDismiss={vi.fn()}
      />
    )

    expect(screen.getByText('100 Mbps')).toBeInTheDocument()
  })

  it('hides speed when not provided', () => {
    const noSpeedEdge = { ...baseEdge, speed: undefined }
    render(
      <EdgeInfoPopover
        edge={noSpeedEdge}
        position={basePosition}
        onDismiss={vi.fn()}
      />
    )

    expect(screen.queryByText('Speed:')).not.toBeInTheDocument()
  })

  it('auto-dismisses after 5 seconds', () => {
    const onDismiss = vi.fn()
    render(
      <EdgeInfoPopover
        edge={baseEdge}
        position={basePosition}
        onDismiss={onDismiss}
      />
    )

    expect(onDismiss).not.toHaveBeenCalled()
    vi.advanceTimersByTime(5000)
    expect(onDismiss).toHaveBeenCalledTimes(1)
  })
})

// ---------------------------------------------------------------------------
// TopologyToolbar Tests
// ---------------------------------------------------------------------------

describe('TopologyToolbar', () => {
  const defaultProps = {
    layout: 'circular' as const,
    onLayoutChange: vi.fn(),
    showMinimap: false,
    onMinimapToggle: vi.fn(),
    flowRef: { current: document.createElement('div') },
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders layout options', () => {
    render(<TopologyToolbar {...defaultProps} />)

    expect(screen.getByTitle('Circular layout')).toBeInTheDocument()
    expect(screen.getByTitle('Hierarchical layout')).toBeInTheDocument()
    expect(screen.getByTitle('Grid layout')).toBeInTheDocument()
  })

  it('renders zoom controls', () => {
    render(<TopologyToolbar {...defaultProps} />)

    expect(screen.getByTitle('Zoom in')).toBeInTheDocument()
    expect(screen.getByTitle('Zoom out')).toBeInTheDocument()
    expect(screen.getByTitle('Fit to view')).toBeInTheDocument()
  })

  it('renders minimap toggle', () => {
    render(<TopologyToolbar {...defaultProps} />)

    expect(screen.getByTitle('Show minimap')).toBeInTheDocument()
  })

  it('renders export button', () => {
    render(<TopologyToolbar {...defaultProps} />)

    expect(screen.getByTitle('Export as PNG')).toBeInTheDocument()
  })

  it('calls onLayoutChange when layout button is clicked', async () => {
    const onLayoutChange = vi.fn()
    render(
      <TopologyToolbar {...defaultProps} onLayoutChange={onLayoutChange} />
    )

    const user = userEvent.setup()
    await user.click(screen.getByTitle('Grid layout'))

    expect(onLayoutChange).toHaveBeenCalledWith('grid')
  })

  it('calls onMinimapToggle when minimap button is clicked', async () => {
    const onMinimapToggle = vi.fn()
    render(
      <TopologyToolbar {...defaultProps} onMinimapToggle={onMinimapToggle} />
    )

    const user = userEvent.setup()
    await user.click(screen.getByTitle('Show minimap'))

    expect(onMinimapToggle).toHaveBeenCalledTimes(1)
  })

  it('shows "Hide minimap" title when minimap is shown', () => {
    render(<TopologyToolbar {...defaultProps} showMinimap={true} />)

    expect(screen.getByTitle('Hide minimap')).toBeInTheDocument()
  })

  it('exports PNG when export button is clicked', async () => {
    const { toPng } = await import('html-to-image')
    const { toast } = await import('sonner')

    render(<TopologyToolbar {...defaultProps} />)

    const user = userEvent.setup()
    await user.click(screen.getByTitle('Export as PNG'))

    expect(toPng).toHaveBeenCalled()
    expect(toast.success).toHaveBeenCalledWith('Topology exported as PNG')
  })
})
