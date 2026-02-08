import { api } from './client'
import type { HealthResponse } from './types'

/**
 * Fetch health and version information from the server.
 */
export async function getHealth(): Promise<HealthResponse> {
  return api.get<HealthResponse>('/health')
}
