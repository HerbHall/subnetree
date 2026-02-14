// Match the Go types from internal/vault/types.go and handlers.go

export interface CredentialMeta {
  id: string
  name: string
  type: string
  device_id?: string
  description?: string
  created_at: string
  updated_at: string
}

export interface CredentialData extends CredentialMeta {
  data: Record<string, unknown>
}

export interface CreateCredentialRequest {
  name: string
  type: string
  device_id?: string
  description?: string
  data: Record<string, unknown>
}

export interface UpdateCredentialRequest {
  name?: string
  device_id?: string
  description?: string
  data?: Record<string, unknown>
}

export interface VaultStatus {
  sealed: boolean
  initialized: boolean
  credential_count: number
}

export interface AuditEntry {
  id: number
  credential_id: string
  user_id: string
  action: string
  purpose?: string
  source_ip?: string
  timestamp: string
}

// Credential type constants -- match Go constants
export const CREDENTIAL_TYPES = [
  { value: 'ssh_password', label: 'SSH Password' },
  { value: 'ssh_key', label: 'SSH Key' },
  { value: 'snmp_v2c', label: 'SNMP v2c' },
  { value: 'snmp_v3', label: 'SNMP v3' },
  { value: 'api_key', label: 'API Key' },
  { value: 'http_basic', label: 'HTTP Basic' },
  { value: 'custom', label: 'Custom' },
] as const

export type CredentialType = typeof CREDENTIAL_TYPES[number]['value']
