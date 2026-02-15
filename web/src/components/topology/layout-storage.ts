import type { Node } from '@xyflow/react'

export interface SavedLayout {
  id: string
  name: string
  nodes: Array<{ id: string; position: { x: number; y: number } }>
  createdAt: string
}

const STORAGE_KEY = 'subnetree-topology-layouts'

/** Read all saved layouts from localStorage. */
function readLayouts(): SavedLayout[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    return JSON.parse(raw) as SavedLayout[]
  } catch {
    return []
  }
}

/** Write layouts array to localStorage. */
function writeLayouts(layouts: SavedLayout[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(layouts))
}

/** Save the current node positions as a named layout. */
export function saveLayout(
  name: string,
  nodes: Array<{ id: string; position: { x: number; y: number } }>
): SavedLayout {
  const layouts = readLayouts()
  const layout: SavedLayout = {
    id: crypto.randomUUID(),
    name,
    nodes: nodes.map((n) => ({ id: n.id, position: { ...n.position } })),
    createdAt: new Date().toISOString(),
  }
  layouts.push(layout)
  writeLayouts(layouts)
  return layout
}

/** Load a single saved layout by ID. */
export function loadLayout(id: string): SavedLayout | null {
  const layouts = readLayouts()
  return layouts.find((l) => l.id === id) ?? null
}

/** List all saved layouts, most recent first. */
export function listLayouts(): SavedLayout[] {
  return readLayouts().sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  )
}

/** Delete a saved layout by ID. */
export function deleteLayout(id: string): void {
  const layouts = readLayouts().filter((l) => l.id !== id)
  writeLayouts(layouts)
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
