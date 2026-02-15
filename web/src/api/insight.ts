import { api } from './client'

export interface NLQueryRequest {
  query: string
}

export interface NLQueryResponse {
  query: string
  answer: string
  structured?: unknown
  model?: string
}

export interface Anomaly {
  id: string
  device_id: string
  metric_name: string
  severity: string
  type: string
  value: number
  expected: number
  deviation: number
  detected_at: string
  resolved_at?: string
  description: string
}

export interface Recommendation {
  id: string
  device_id: string
  type: string
  severity: string
  title: string
  description: string
  metric: string
  current_value: number
  threshold: number
  generated_at: string
}

export interface Forecast {
  device_id: string
  metric_name: string
  current_value: number
  predicted_value: number
  time_to_threshold?: number
  threshold: number
  confidence: number
  slope: number
  generated_at: string
}

export interface AlertGroup {
  id: string
  root_cause: string
  device_ids: string[]
  alert_count: number
  description: string
  created_at: string
}

/**
 * Submit a natural language query to the Insight module.
 */
export async function submitNLQuery(req: NLQueryRequest): Promise<NLQueryResponse> {
  return api.post<NLQueryResponse>('/insight/query', req)
}

/**
 * List detected anomalies across all devices.
 */
export async function listAnomalies(limit?: number): Promise<Anomaly[]> {
  const query = new URLSearchParams()
  if (limit) query.set('limit', limit.toString())
  const qs = query.toString()
  return api.get<Anomaly[]>(`/insight/anomalies${qs ? `?${qs}` : ''}`)
}

/**
 * Get anomalies for a specific device.
 */
export async function getDeviceAnomalies(deviceId: string): Promise<Anomaly[]> {
  return api.get<Anomaly[]>(`/insight/anomalies/${deviceId}`)
}

/**
 * Get capacity forecasts for a specific device.
 */
export async function getDeviceForecasts(deviceId: string): Promise<Forecast[]> {
  return api.get<Forecast[]>(`/insight/forecasts/${deviceId}`)
}

/**
 * Get alert correlation groups involving a specific device.
 */
export async function getDeviceCorrelations(deviceId: string): Promise<AlertGroup[]> {
  return api.get<AlertGroup[]>(`/insight/correlations/${deviceId}`)
}

/**
 * Get AI optimization recommendations.
 */
export async function getRecommendations(): Promise<Recommendation[]> {
  return api.get<Recommendation[]>('/insight/recommendations')
}
