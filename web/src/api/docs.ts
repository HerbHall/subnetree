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
