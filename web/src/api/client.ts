import { useAuthStore } from '@/stores/auth'

const BASE_URL = '/api/v1'

export class ApiError extends Error {
  constructor(
    public status: number,
    public detail: string,
    public type?: string,
  ) {
    super(detail)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const { accessToken, refresh, logout } = useAuthStore.getState()

  const headers = new Headers(options.headers)
  headers.set('Content-Type', 'application/json')

  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`)
  }

  let response = await fetch(`${BASE_URL}${path}`, { ...options, headers })

  // If 401 and we have a token, attempt refresh once
  if (response.status === 401 && accessToken) {
    const refreshed = await refresh()
    if (refreshed) {
      const newState = useAuthStore.getState()
      headers.set('Authorization', `Bearer ${newState.accessToken}`)
      response = await fetch(`${BASE_URL}${path}`, { ...options, headers })
    } else {
      logout()
      throw new ApiError(401, 'Session expired')
    }
  }

  if (!response.ok) {
    const problem = await response.json().catch(() => ({}))
    throw new ApiError(response.status, problem.detail || response.statusText, problem.type)
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T
  }

  return response.json()
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PUT', body: body ? JSON.stringify(body) : undefined }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}
