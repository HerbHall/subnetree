import { api } from './client'
import type {
  Check,
  CheckDependency,
  CheckResult,
  Alert,
  CreateCheckRequest,
  UpdateCheckRequest,
  MonitoringStatus,
  NotificationChannel,
  CreateNotificationRequest,
  UpdateNotificationRequest,
  MetricSeries,
  MetricName,
  MetricRange,
} from './types'

/**
 * List all monitoring checks (enabled and disabled).
 */
export async function listChecks(): Promise<Check[]> {
  return api.get<Check[]>('/pulse/checks')
}

/**
 * Get monitoring checks for a specific device.
 */
export async function getDeviceChecks(deviceId: string): Promise<Check | null> {
  return api.get<Check | null>(`/pulse/checks/${deviceId}`)
}

/**
 * Create a new monitoring check.
 */
export async function createCheck(req: CreateCheckRequest): Promise<Check> {
  return api.post<Check>('/pulse/checks', req)
}

/**
 * Update an existing monitoring check.
 */
export async function updateCheck(
  id: string,
  req: UpdateCheckRequest
): Promise<Check> {
  return api.put<Check>(`/pulse/checks/${id}`, req)
}

/**
 * Delete a monitoring check and its results.
 */
export async function deleteCheck(id: string): Promise<void> {
  return api.delete<void>(`/pulse/checks/${id}`)
}

/**
 * Toggle a check's enabled/disabled state.
 */
export async function toggleCheck(id: string): Promise<Check> {
  return api.patch<Check>(`/pulse/checks/${id}/toggle`, {})
}

/**
 * Get recent check results for a device.
 */
export async function getDeviceResults(
  deviceId: string,
  limit?: number
): Promise<CheckResult[]> {
  const query = new URLSearchParams()
  if (limit) query.set('limit', limit.toString())
  const qs = query.toString()
  return api.get<CheckResult[]>(`/pulse/results/${deviceId}${qs ? `?${qs}` : ''}`)
}

/**
 * List alerts with optional filtering.
 */
export async function listAlerts(params?: {
  device_id?: string
  severity?: string
  active?: boolean
  suppressed?: boolean
  limit?: number
}): Promise<Alert[]> {
  const query = new URLSearchParams()
  if (params?.device_id) query.set('device_id', params.device_id)
  if (params?.severity) query.set('severity', params.severity)
  if (params?.active !== undefined) query.set('active', params.active.toString())
  if (params?.suppressed !== undefined) query.set('suppressed', params.suppressed.toString())
  if (params?.limit) query.set('limit', params.limit.toString())
  const qs = query.toString()
  return api.get<Alert[]>(`/pulse/alerts${qs ? `?${qs}` : ''}`)
}

/**
 * Get a single alert by ID.
 */
export async function getAlert(id: string): Promise<Alert> {
  return api.get<Alert>(`/pulse/alerts/${id}`)
}

/**
 * Acknowledge an alert.
 */
export async function acknowledgeAlert(id: string): Promise<Alert> {
  return api.post<Alert>(`/pulse/alerts/${id}/acknowledge`, {})
}

/**
 * Resolve an alert.
 */
export async function resolveAlert(id: string): Promise<Alert> {
  return api.post<Alert>(`/pulse/alerts/${id}/resolve`, {})
}

// ============================================================================
// Check Dependencies
// ============================================================================

/**
 * List dependencies for a check.
 */
export async function listCheckDependencies(
  checkId: string
): Promise<CheckDependency[]> {
  return api.get<CheckDependency[]>(`/pulse/checks/${checkId}/dependencies`)
}

/**
 * Add a dependency between a check and an upstream device.
 */
export async function addCheckDependency(
  checkId: string,
  deviceId: string
): Promise<void> {
  await api.post(`/pulse/checks/${checkId}/dependencies`, {
    depends_on_device_id: deviceId,
  })
}

/**
 * Remove a dependency between a check and an upstream device.
 */
export async function removeCheckDependency(
  checkId: string,
  deviceId: string
): Promise<void> {
  await api.delete(`/pulse/checks/${checkId}/dependencies/${deviceId}`)
}

/**
 * Get composite monitoring status for a device.
 */
export async function getDeviceStatus(
  deviceId: string
): Promise<MonitoringStatus> {
  return api.get<MonitoringStatus>(`/pulse/status/${deviceId}`)
}

// ============================================================================
// Notification Channels
// ============================================================================

/**
 * List notification channels.
 */
export async function listChannels(): Promise<NotificationChannel[]> {
  return api.get<NotificationChannel[]>('/pulse/notifications')
}

/**
 * Get a single notification channel by ID.
 */
export async function getChannel(id: string): Promise<NotificationChannel> {
  return api.get<NotificationChannel>(`/pulse/notifications/${id}`)
}

/**
 * Create a notification channel.
 */
export async function createChannel(req: CreateNotificationRequest): Promise<NotificationChannel> {
  return api.post<NotificationChannel>('/pulse/notifications', req)
}

/**
 * Update a notification channel.
 */
export async function updateChannel(id: string, req: UpdateNotificationRequest): Promise<NotificationChannel> {
  return api.put<NotificationChannel>(`/pulse/notifications/${id}`, req)
}

/**
 * Delete a notification channel.
 */
export async function deleteChannel(id: string): Promise<void> {
  return api.delete<void>(`/pulse/notifications/${id}`)
}

/**
 * Test a notification channel by sending a test alert.
 */
export async function testChannel(id: string): Promise<void> {
  return api.post<void>(`/pulse/notifications/${id}/test`, {})
}

// ============================================================================
// Metrics History
// ============================================================================

/**
 * Get time-series metric data for a device.
 */
export async function getDeviceMetrics(
  deviceId: string,
  metric: MetricName,
  range: MetricRange
): Promise<MetricSeries> {
  return api.get<MetricSeries>(
    `/pulse/metrics/${deviceId}?metric=${metric}&range=${range}`
  )
}
