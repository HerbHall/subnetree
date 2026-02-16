import { api } from './client'
import type { DiagPingRequest, DiagPingResult, DiagDNSRequest, DiagDNSResult, DiagPortCheckRequest, DiagPortCheckResult } from './types'

/** Run an ICMP ping diagnostic to a target host. */
export async function runDiagPing(req: DiagPingRequest): Promise<DiagPingResult> {
  return api.post<DiagPingResult>('/recon/diag/ping', req)
}

/** Run a DNS lookup (reverse for IPs, forward for hostnames). */
export async function runDiagDNS(req: DiagDNSRequest): Promise<DiagDNSResult> {
  return api.post<DiagDNSResult>('/recon/diag/dns', req)
}

/** Run TCP port checks on a target host. */
export async function runDiagPortCheck(req: DiagPortCheckRequest): Promise<DiagPortCheckResult> {
  return api.post<DiagPortCheckResult>('/recon/diag/port-check', req)
}
