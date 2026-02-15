import type { Node } from '@xyflow/react'
import {
  listTopologyLayouts,
  createTopologyLayout,
  deleteTopologyLayout,
  type TopologyLayoutAPI,
} from '@/api/devices'

export interface SavedLayout {
  id: string
  name: string
  nodes: Array<{ id: string; position: { x: number; y: number } }>
  createdAt: string
}

const STORAGE_KEY = 'subnetree-topology-layouts'

/** Convert API response to SavedLayout. */
function fromAPI(api: TopologyLayoutAPI): SavedLayout {
  return {
    id: api.id,
    name: api.name,
    nodes: JSON.parse(api.positions),
    createdAt: api.created_at,
  }
}

/** Read layouts from localStorage (fallback). */
function readLocalLayouts(): SavedLayout[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    return JSON.parse(raw) as SavedLayout[]
  } catch {
    return []
  }
}

/** Save the current node positions as a named layout. */
export async function saveLayout(
  name: string,
  nodes: Array<{ id: string; position: { x: number; y: number } }>
): Promise<SavedLayout> {
  const positions = JSON.stringify(
    nodes.map((n) => ({ id: n.id, position: { ...n.position } }))
  )
  try {
    const result = await createTopologyLayout(name, positions)
    return fromAPI(result)
  } catch {
    // Fallback to localStorage
    const layout: SavedLayout = {
      id: crypto.randomUUID(),
      name,
      nodes: nodes.map((n) => ({ id: n.id, position: { ...n.position } })),
      createdAt: new Date().toISOString(),
    }
    const layouts = readLocalLayouts()
    layouts.push(layout)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(layouts))
    return layout
  }
}

/** Load a single saved layout by ID. */
export async function loadLayout(id: string): Promise<SavedLayout | null> {
  try {
    const layouts = await listTopologyLayouts()
    const found = layouts.find((l) => l.id === id)
    return found ? fromAPI(found) : null
  } catch {
    return readLocalLayouts().find((l) => l.id === id) ?? null
  }
}

/** List all saved layouts, most recent first. */
export async function listLayouts(): Promise<SavedLayout[]> {
  try {
    const layouts = await listTopologyLayouts()
    return layouts.map(fromAPI)
  } catch {
    return readLocalLayouts().sort(
      (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    )
  }
}

/** Delete a saved layout by ID. */
export async function deleteLayout(id: string): Promise<void> {
  try {
    await deleteTopologyLayout(id)
  } catch {
    const layouts = readLocalLayouts().filter((l) => l.id !== id)
    localStorage.setItem(STORAGE_KEY, JSON.stringify(layouts))
  }
}

/**
 * Apply a saved layout to the current set of nodes.
 * Nodes present in the saved layout get their positions updated;
 * nodes not in the saved layout keep their current positions.
 */
export function applyLayout<T extends Node>(
  currentNodes: T[],
  saved: SavedLayout
): T[] {
  const positionMap = new Map(
    saved.nodes.map((n) => [n.id, n.position])
  )
  return currentNodes.map((node) => {
    const savedPos = positionMap.get(node.id)
    if (savedPos) {
      return { ...node, position: { ...savedPos } }
    }
    return node
  })
}
