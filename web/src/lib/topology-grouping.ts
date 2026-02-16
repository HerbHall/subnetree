import type { TopologyNode } from '@/api/types'
import type { DeviceNodeData } from '@/components/topology/device-node'
import type { Node } from '@xyflow/react'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type ViewMode = 'all' | 'by-type' | 'by-status' | 'by-subnet'

export interface GroupInfo {
  id: string
  label: string
  nodeIds: string[]
}

/** Data shape for the group (container) node rendered by GroupNode. */
export interface GroupNodeData {
  label: string
  count: number
  groupKind: ViewMode
  [key: string]: unknown
}

export type GroupNodeType = Node<GroupNodeData, 'group'>

// ---------------------------------------------------------------------------
// Subnet helpers
// ---------------------------------------------------------------------------

/** Extract the /24 subnet string from an IP address. Returns null for unusable values. */
export function extractSubnet(ip: string): string | null {
  const parts = ip.split('.')
  if (parts.length !== 4) return null
  // Validate each octet is a number 0-255
  for (const p of parts) {
    const n = Number(p)
    if (!Number.isFinite(n) || n < 0 || n > 255) return null
  }
  return `${parts[0]}.${parts[1]}.${parts[2]}.0/24`
}

// ---------------------------------------------------------------------------
// Grouping functions -- pure, no side effects
// ---------------------------------------------------------------------------

/** Group nodes by device_type. */
export function groupByType(nodes: TopologyNode[]): GroupInfo[] {
  const groups = new Map<string, string[]>()
  for (const node of nodes) {
    const key = node.device_type
    const list = groups.get(key)
    if (list) {
      list.push(node.id)
    } else {
      groups.set(key, [node.id])
    }
  }
  return Array.from(groups.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, ids]) => ({
      id: `group-type-${key}`,
      label: formatTypeLabel(key),
      nodeIds: ids,
    }))
}

/** Group nodes by status. */
export function groupByStatus(nodes: TopologyNode[]): GroupInfo[] {
  const groups = new Map<string, string[]>()
  for (const node of nodes) {
    const key = node.status
    const list = groups.get(key)
    if (list) {
      list.push(node.id)
    } else {
      groups.set(key, [node.id])
    }
  }
  // Use a fixed order for statuses
  const order = ['online', 'degraded', 'offline', 'unknown']
  return order
    .filter((s) => groups.has(s))
    .map((key) => ({
      id: `group-status-${key}`,
      label: formatStatusLabel(key),
      nodeIds: groups.get(key)!,
    }))
}

/** Group nodes by /24 subnet extracted from their first IP address. */
export function groupBySubnet(nodes: TopologyNode[]): GroupInfo[] {
  const groups = new Map<string, string[]>()
  for (const node of nodes) {
    const ip = node.ip_addresses?.[0]
    const subnet = ip ? extractSubnet(ip) : null
    const key = subnet ?? 'Unknown Subnet'
    const list = groups.get(key)
    if (list) {
      list.push(node.id)
    } else {
      groups.set(key, [node.id])
    }
  }
  return Array.from(groups.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, ids]) => ({
      id: `group-subnet-${key.replace(/[./]/g, '-')}`,
      label: key,
      nodeIds: ids,
    }))
}

// ---------------------------------------------------------------------------
// Layout -- positions group nodes and their children in a grid-of-grids
// ---------------------------------------------------------------------------

/** Spacing constants (pixels). */
const GROUP_PADDING_X = 40
const GROUP_PADDING_TOP = 50 // Extra top padding for the group label
const GROUP_PADDING_BOTTOM = 30
const CHILD_CELL_W = 200
const CHILD_CELL_H = 120
const GROUP_GAP = 60

/**
 * Given groups and the original API nodes, produce React Flow nodes:
 *   - One group node per group (type = 'group')
 *   - Device nodes with `parentId` set so they live inside the group
 *
 * Returns { groupNodes, deviceNodes } so the caller can merge them into a
 * single flat array (groups first, then devices -- React Flow requires
 * parents before children).
 */
export function layoutGroups(
  groups: GroupInfo[],
  apiNodes: TopologyNode[],
  viewMode: ViewMode,
): { groupNodes: GroupNodeType[]; deviceNodes: Node<DeviceNodeData, 'device'>[] } {
  const nodeMap = new Map<string, TopologyNode>()
  for (const n of apiNodes) nodeMap.set(n.id, n)

  const groupNodes: GroupNodeType[] = []
  const deviceNodes: Node<DeviceNodeData, 'device'>[] = []

  // Lay groups out horizontally, wrapping after a maximum width
  let groupX = 0
  let groupY = 0
  let rowMaxHeight = 0
  const MAX_ROW_WIDTH = 1600

  for (const group of groups) {
    const memberCount = group.nodeIds.length
    const cols = Math.max(1, Math.ceil(Math.sqrt(memberCount)))
    const rows = Math.ceil(memberCount / cols)
    const groupW = GROUP_PADDING_X * 2 + cols * CHILD_CELL_W
    const groupH = GROUP_PADDING_TOP + rows * CHILD_CELL_H + GROUP_PADDING_BOTTOM

    // Wrap to next row if this group would exceed max width
    if (groupX > 0 && groupX + groupW > MAX_ROW_WIDTH) {
      groupX = 0
      groupY += rowMaxHeight + GROUP_GAP
      rowMaxHeight = 0
    }

    // Create the group (container) node
    groupNodes.push({
      id: group.id,
      type: 'group',
      position: { x: groupX, y: groupY },
      data: {
        label: group.label,
        count: memberCount,
        groupKind: viewMode,
      },
      style: {
        width: groupW,
        height: groupH,
      },
    })

    // Create child device nodes positioned relative to the group
    group.nodeIds.forEach((nodeId, idx) => {
      const apiNode = nodeMap.get(nodeId)
      if (!apiNode) return

      const col = idx % cols
      const row = Math.floor(idx / cols)
      deviceNodes.push({
        id: nodeId,
        type: 'device',
        position: {
          x: GROUP_PADDING_X + col * CHILD_CELL_W,
          y: GROUP_PADDING_TOP + row * CHILD_CELL_H,
        },
        parentId: group.id,
        extent: 'parent' as const,
        data: {
          label: apiNode.label,
          deviceType: apiNode.device_type,
          status: apiNode.status,
          ip: apiNode.ip_addresses?.[0] || 'No IP',
          openPorts: apiNode.open_ports,
        },
      })
    })

    groupX += groupW + GROUP_GAP
    rowMaxHeight = Math.max(rowMaxHeight, groupH)
  }

  return { groupNodes, deviceNodes }
}

// ---------------------------------------------------------------------------
// Label formatting helpers
// ---------------------------------------------------------------------------

const TYPE_LABELS: Record<string, string> = {
  server: 'Servers',
  desktop: 'Desktops',
  laptop: 'Laptops',
  mobile: 'Mobile Devices',
  router: 'Routers',
  switch: 'Switches',
  access_point: 'Access Points',
  firewall: 'Firewalls',
  printer: 'Printers',
  nas: 'NAS',
  iot: 'IoT Devices',
  phone: 'Phones',
  tablet: 'Tablets',
  camera: 'Cameras',
  unknown: 'Unknown',
}

function formatTypeLabel(type: string): string {
  return TYPE_LABELS[type] ?? type.charAt(0).toUpperCase() + type.slice(1)
}

const STATUS_LABELS: Record<string, string> = {
  online: 'Online',
  offline: 'Offline',
  degraded: 'Degraded',
  unknown: 'Unknown',
}

function formatStatusLabel(status: string): string {
  return STATUS_LABELS[status] ?? status.charAt(0).toUpperCase() + status.slice(1)
}
