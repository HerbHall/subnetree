import { api } from './client'

/**
 * A tracked application (Docker container, service, etc.)
 */
export interface DocsApplication {
  id: string
  name: string
  app_type: string
  device_id: string
  collector: string
  status: string
  metadata: string
  discovered_at: string
  updated_at: string
}

/**
 * A point-in-time snapshot of an application's configuration.
 */
export interface DocsSnapshot {
  id: string
  application_id: string
  content_hash: string
  content: string
  format: string
  size_bytes: number
  source: string
  captured_at: string
}

/**
 * Paginated response for listing applications.
 */
export interface ListApplicationsResponse {
  items: DocsApplication[]
  total: number
}

/**
 * List tracked applications with optional filtering.
 */
export async function listApplications(params?: {
  limit?: number
  offset?: number
  type?: string
  status?: string
}): Promise<ListApplicationsResponse> {
  const searchParams = new URLSearchParams()
  if (params?.limit !== undefined) searchParams.set('limit', String(params.limit))
  if (params?.offset !== undefined) searchParams.set('offset', String(params.offset))
  if (params?.type) searchParams.set('type', params.type)
  if (params?.status) searchParams.set('status', params.status)

  const query = searchParams.toString()
  const path = `/docs/applications${query ? `?${query}` : ''}`
  return api.get<ListApplicationsResponse>(path)
}

/**
 * Get a single application by ID.
 */
export async function getApplication(id: string): Promise<DocsApplication> {
  return api.get<DocsApplication>(`/docs/applications/${id}`)
}

/**
 * List snapshots with optional filtering by application.
 */
export async function listSnapshots(params?: {
  application_id?: string
  limit?: number
  offset?: number
}): Promise<DocsSnapshot[]> {
  const searchParams = new URLSearchParams()
  if (params?.application_id) searchParams.set('application_id', params.application_id)
  if (params?.limit !== undefined) searchParams.set('limit', String(params.limit))
  if (params?.offset !== undefined) searchParams.set('offset', String(params.offset))

  const query = searchParams.toString()
  const path = `/docs/snapshots${query ? `?${query}` : ''}`
  return api.get<DocsSnapshot[]>(path)
}

/**
 * Get a single snapshot by ID.
 */
export async function getSnapshot(id: string): Promise<DocsSnapshot> {
  return api.get<DocsSnapshot>(`/docs/snapshots/${id}`)
}

/**
 * Create a new snapshot for an application.
 */
export async function createSnapshot(data: {
  application_id: string
  content: string
  format: string
}): Promise<DocsSnapshot> {
  return api.post<DocsSnapshot>('/docs/snapshots', data)
}

/**
 * Information about a registered collector.
 */
export interface CollectorInfo {
  name: string
  available: boolean
}

/**
 * Result of a collection run.
 */
export interface CollectionResult {
  apps_discovered: number
  snapshots_created: number
  errors: string[]
}

/**
 * List registered collectors with availability status.
 */
export async function listCollectors(): Promise<CollectorInfo[]> {
  return api.get<CollectorInfo[]>('/docs/collectors')
}

/**
 * Trigger collection from all available collectors.
 */
export async function triggerCollection(): Promise<CollectionResult> {
  return api.post<CollectionResult>('/docs/collect')
}

/**
 * Trigger collection from a specific named collector.
 */
export async function triggerCollectorByName(name: string): Promise<CollectionResult> {
  return api.post<CollectionResult>(`/docs/collect/${name}`)
}

/**
 * Paginated snapshot history for an application.
 */
export interface SnapshotHistory {
  snapshots: DocsSnapshot[]
  total: number
}

/**
 * Result of comparing two snapshots.
 */
export interface DiffResult {
  diff_text: string
  old_snapshot_id: string
  new_snapshot_id: string
}

/**
 * Get paginated snapshot history for an application.
 */
export async function getApplicationHistory(
  appId: string,
  limit = 20,
  offset = 0,
): Promise<SnapshotHistory> {
  const searchParams = new URLSearchParams()
  searchParams.set('limit', String(limit))
  searchParams.set('offset', String(offset))
  return api.get<SnapshotHistory>(`/docs/applications/${appId}/history?${searchParams.toString()}`)
}

/**
 * Get a unified diff between two snapshots.
 */
export async function getSnapshotDiff(
  snapshotId: string,
  otherSnapshotId: string,
): Promise<DiffResult> {
  return api.get<DiffResult>(`/docs/snapshots/${snapshotId}/diff/${otherSnapshotId}`)
}

/**
 * Delete a snapshot by ID.
 */
export async function deleteSnapshot(snapshotId: string): Promise<void> {
  return api.delete<void>(`/docs/snapshots/${snapshotId}`)
}
