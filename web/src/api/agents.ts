import { api } from './client'
import type {
  AgentInfo,
  CreateEnrollmentTokenRequest,
  EnrollmentTokenResponse,
  HardwareProfile,
  SoftwareInventory,
  ServiceInfo,
} from './types'

/**
 * List all registered Scout agents.
 */
export async function listAgents(): Promise<AgentInfo[]> {
  return api.get<AgentInfo[]>('/dispatch/agents')
}

/**
 * Get a single agent by ID.
 */
export async function getAgent(id: string): Promise<AgentInfo> {
  return api.get<AgentInfo>(`/dispatch/agents/${id}`)
}

/**
 * Delete an agent by ID.
 */
export async function deleteAgent(id: string): Promise<void> {
  return api.delete<void>(`/dispatch/agents/${id}`)
}

/**
 * Create a new enrollment token for agent registration.
 * The raw token is only returned once in the response.
 */
export async function createEnrollmentToken(
  req: CreateEnrollmentTokenRequest
): Promise<EnrollmentTokenResponse> {
  return api.post<EnrollmentTokenResponse>('/dispatch/enroll', req)
}

/**
 * Get hardware profile for an agent.
 */
export async function getAgentHardware(id: string): Promise<HardwareProfile> {
  return api.get<HardwareProfile>(`/dispatch/agents/${id}/hardware`)
}

/**
 * Get software inventory for an agent.
 */
export async function getAgentSoftware(id: string): Promise<SoftwareInventory> {
  return api.get<SoftwareInventory>(`/dispatch/agents/${id}/software`)
}

/**
 * Get running services for an agent.
 */
export async function getAgentServices(id: string): Promise<ServiceInfo[]> {
  return api.get<ServiceInfo[]>(`/dispatch/agents/${id}/services`)
}
