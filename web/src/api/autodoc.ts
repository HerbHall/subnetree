import { api } from './client'

export interface ChangelogEntry {
  id: string
  event_type: string
  summary: string
  details: unknown
  source_module: string
  device_id?: string
  created_at: string
}

export interface ChangelogListResponse {
  entries: ChangelogEntry[]
  total: number
  page: number
  per_page: number
}

export interface ChangelogFilter {
  page?: number
  per_page?: number
  event_type?: string
  since?: string
  until?: string
}

export interface ChangelogStats {
  total_entries: number
  entries_by_type: Record<string, number>
  latest_entry?: ChangelogEntry
  oldest_entry?: ChangelogEntry
}

/**
 * List changelog entries with optional filters and pagination.
 */
export async function getChangelog(filter?: ChangelogFilter): Promise<ChangelogListResponse> {
  const params = new URLSearchParams()
  if (filter?.page) params.set('page', filter.page.toString())
  if (filter?.per_page) params.set('per_page', filter.per_page.toString())
  if (filter?.event_type) params.set('event_type', filter.event_type)
  if (filter?.since) params.set('since', filter.since)
  if (filter?.until) params.set('until', filter.until)
  const qs = params.toString()
  return api.get<ChangelogListResponse>(`/autodoc/changes${qs ? `?${qs}` : ''}`)
}

/**
 * Export changelog as markdown for a given time range.
 */
export async function exportChangelog(range_: string = '7d'): Promise<string> {
  const params = new URLSearchParams()
  params.set('range', range_)
  const response = await fetch(`/api/v1/autodoc/export?${params.toString()}`, {
    headers: {
      Authorization: `Bearer ${localStorage.getItem('access_token') ?? ''}`,
    },
  })
  if (!response.ok) {
    throw new Error('Failed to export changelog')
  }
  return response.text()
}

/**
 * Get aggregate statistics about the changelog.
 */
export async function getStats(): Promise<ChangelogStats> {
  return api.get<ChangelogStats>('/autodoc/stats')
}
