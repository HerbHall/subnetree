import { api } from './client'
import type {
  CredentialMeta,
  CredentialData,
  CreateCredentialRequest,
  UpdateCredentialRequest,
  VaultStatus,
  AuditEntry,
} from '@/pages/vault-types'

/**
 * List all credentials (metadata only, no secrets).
 */
export async function listCredentials(): Promise<CredentialMeta[]> {
  return api.get<CredentialMeta[]>('/vault/credentials')
}

/**
 * Create a new credential.
 */
export async function createCredential(req: CreateCredentialRequest): Promise<CredentialMeta> {
  return api.post<CredentialMeta>('/vault/credentials', req)
}

/**
 * Get a single credential by ID (metadata only).
 */
export async function getCredential(id: string): Promise<CredentialMeta> {
  return api.get<CredentialMeta>(`/vault/credentials/${id}`)
}

/**
 * Update an existing credential.
 */
export async function updateCredential(id: string, req: UpdateCredentialRequest): Promise<CredentialMeta> {
  return api.put<CredentialMeta>(`/vault/credentials/${id}`, req)
}

/**
 * Delete a credential.
 */
export async function deleteCredential(id: string): Promise<void> {
  return api.delete<void>(`/vault/credentials/${id}`)
}

/**
 * Retrieve decrypted credential data (requires purpose for audit).
 */
export async function getCredentialData(id: string, purpose?: string): Promise<CredentialData> {
  const qs = purpose ? `?purpose=${encodeURIComponent(purpose)}` : ''
  return api.get<CredentialData>(`/vault/credentials/${id}/data${qs}`)
}

/**
 * List credentials assigned to a specific device.
 */
export async function listDeviceCredentials(deviceId: string): Promise<CredentialMeta[]> {
  return api.get<CredentialMeta[]>(`/vault/device-credentials/${deviceId}`)
}

/**
 * Get vault status (sealed/unsealed, credential count).
 */
export async function getVaultStatus(): Promise<VaultStatus> {
  return api.get<VaultStatus>('/vault/status')
}

/**
 * Seal the vault (encrypts all data at rest).
 */
export async function sealVault(): Promise<void> {
  return api.post<void>('/vault/seal', {})
}

/**
 * Unseal the vault with a passphrase.
 */
export async function unsealVault(passphrase: string): Promise<void> {
  return api.post<void>('/vault/unseal', { passphrase })
}

/**
 * Get audit log entries with optional filtering.
 */
export async function getAuditLog(params?: {
  credential_id?: string
  limit?: number
}): Promise<AuditEntry[]> {
  const query = new URLSearchParams()
  if (params?.credential_id) query.set('credential_id', params.credential_id)
  if (params?.limit) query.set('limit', params.limit.toString())
  const qs = query.toString()
  return api.get<AuditEntry[]>(`/vault/audit${qs ? `?${qs}` : ''}`)
}
