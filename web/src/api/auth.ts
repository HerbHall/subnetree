import type { TokenPair, User } from './types'

const BASE_URL = '/api/v1'

/**
 * Auth API calls that bypass the authenticated client.
 * Login/refresh/setup don't need a JWT, and refresh needs
 * special handling to avoid circular refresh loops.
 */

export async function loginApi(username: string, password: string): Promise<TokenPair> {
  const res = await fetch(`${BASE_URL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  })
  if (!res.ok) {
    const problem = await res.json().catch(() => ({}))
    throw new Error(problem.detail || 'Login failed')
  }
  return res.json()
}

export async function refreshApi(refreshToken: string): Promise<TokenPair> {
  const res = await fetch(`${BASE_URL}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  })
  if (!res.ok) {
    throw new Error('Token refresh failed')
  }
  return res.json()
}

export async function setupApi(
  username: string,
  email: string,
  password: string,
): Promise<User> {
  const res = await fetch(`${BASE_URL}/auth/setup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, email, password }),
  })
  if (!res.ok) {
    const problem = await res.json().catch(() => ({}))
    throw new Error(problem.detail || 'Setup failed')
  }
  return res.json()
}

export async function logoutApi(refreshToken: string): Promise<void> {
  await fetch(`${BASE_URL}/auth/logout`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  })
}
