import { api } from './client'

/** Raw scan metrics as returned by the recon module. */
export interface RawScanMetric {
  scan_id: string
  duration_ms: number
  ping_phase_ms: number
  enrich_phase_ms: number
  post_process_ms: number
  hosts_scanned: number
  hosts_alive: number
  devices_created: number
  devices_updated: number
  created_at: string
}

/** Time-bucketed aggregate of scan metrics. */
export interface ScanMetricsAggregate {
  id: string
  period: string
  period_start: string
  period_end: string
  scan_count: number
  avg_duration_ms: number
  avg_ping_phase_ms: number
  avg_enrich_ms: number
  avg_devices_found: number
  max_devices_found: number
  min_devices_found: number
  avg_hosts_alive: number
  total_new_devices: number
  failed_scans: number
  created_at: string
}

/** Health score factor breakdown. */
export interface HealthScoreFactor {
  name: string
  score: number
  weight: number
  detail: string
}

/** Health score response from the recon module. */
export interface HealthScoreResponse {
  score: number
  grade: 'green' | 'yellow' | 'red'
  factors: HealthScoreFactor[]
}

/** Fetch the computed network health score. */
export async function getHealthScore(): Promise<HealthScoreResponse> {
  return api.get<HealthScoreResponse>('/recon/metrics/health-score')
}

/** Fetch raw scan metrics (most recent first). */
export async function getRawMetrics(limit = 30): Promise<RawScanMetric[]> {
  return api.get<RawScanMetric[]>(`/recon/metrics/raw?limit=${limit}`)
}

/** Fetch aggregated scan metrics for a given period. */
export async function getAggregateMetrics(
  period: 'weekly' | 'monthly' = 'weekly',
  limit = 52,
): Promise<ScanMetricsAggregate[]> {
  return api.get<ScanMetricsAggregate[]>(
    `/recon/metrics/aggregates?period=${period}&limit=${limit}`,
  )
}
