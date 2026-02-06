import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useAuthStore } from './auth'

// Mock jwt-decode
vi.mock('jwt-decode', () => ({
  jwtDecode: vi.fn((token: string) => {
    if (token === 'valid-token') {
      return {
        uid: 'user-123',
        usr: 'testuser',
        role: 'admin',
        exp: Math.floor(Date.now() / 1000) + 3600, // 1 hour from now
      }
    }
    if (token === 'expired-token') {
      return {
        uid: 'user-123',
        usr: 'testuser',
        role: 'admin',
        exp: Math.floor(Date.now() / 1000) - 3600, // 1 hour ago
      }
    }
    throw new Error('Invalid token')
  }),
}))

// Mock API calls
vi.mock('@/api/auth', () => ({
  loginApi: vi.fn(),
  refreshApi: vi.fn(),
  logoutApi: vi.fn(),
}))

describe('useAuthStore', () => {
  beforeEach(() => {
    // Reset store state before each test
    useAuthStore.setState({
      accessToken: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      isHydrated: false,
    })
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('initial state', () => {
    it('should have default values', () => {
      const state = useAuthStore.getState()
      expect(state.accessToken).toBeNull()
      expect(state.refreshToken).toBeNull()
      expect(state.user).toBeNull()
      expect(state.isAuthenticated).toBe(false)
    })
  })

  describe('setTokens', () => {
    it('should decode token and set user info', () => {
      const { setTokens } = useAuthStore.getState()
      setTokens('valid-token', 'refresh-token')

      const state = useAuthStore.getState()
      expect(state.accessToken).toBe('valid-token')
      expect(state.refreshToken).toBe('refresh-token')
      expect(state.isAuthenticated).toBe(true)
      expect(state.user).toEqual({
        id: 'user-123',
        username: 'testuser',
        role: 'admin',
      })
    })
  })

  describe('login', () => {
    it('should call loginApi and set tokens on success', async () => {
      const { loginApi } = await import('@/api/auth')
      vi.mocked(loginApi).mockResolvedValue({
        access_token: 'valid-token',
        refresh_token: 'refresh-token',
        expires_in: 900,
      })

      const { login } = useAuthStore.getState()
      await login('testuser', 'password123')

      expect(loginApi).toHaveBeenCalledWith('testuser', 'password123')
      const state = useAuthStore.getState()
      expect(state.isAuthenticated).toBe(true)
      expect(state.user?.username).toBe('testuser')
    })

    it('should throw on login failure', async () => {
      const { loginApi } = await import('@/api/auth')
      vi.mocked(loginApi).mockRejectedValue(new Error('Invalid credentials'))

      const { login } = useAuthStore.getState()
      await expect(login('testuser', 'wrongpass')).rejects.toThrow('Invalid credentials')
    })
  })

  describe('refresh', () => {
    it('should refresh tokens successfully', async () => {
      const { refreshApi } = await import('@/api/auth')
      vi.mocked(refreshApi).mockResolvedValue({
        access_token: 'valid-token',
        refresh_token: 'new-refresh-token',
        expires_in: 900,
      })

      // Set initial tokens
      useAuthStore.setState({
        accessToken: 'old-token',
        refreshToken: 'old-refresh-token',
        isAuthenticated: true,
      })

      const { refresh } = useAuthStore.getState()
      const result = await refresh()

      expect(result).toBe(true)
      expect(refreshApi).toHaveBeenCalledWith('old-refresh-token')
      const state = useAuthStore.getState()
      expect(state.refreshToken).toBe('new-refresh-token')
    })

    it('should return false and logout on refresh failure', async () => {
      const { refreshApi, logoutApi } = await import('@/api/auth')
      vi.mocked(refreshApi).mockRejectedValue(new Error('Token expired'))
      vi.mocked(logoutApi).mockResolvedValue(undefined)

      useAuthStore.setState({
        accessToken: 'old-token',
        refreshToken: 'old-refresh-token',
        isAuthenticated: true,
        user: { id: '1', username: 'test', role: 'admin' },
      })

      const { refresh } = useAuthStore.getState()
      const result = await refresh()

      expect(result).toBe(false)
      const state = useAuthStore.getState()
      expect(state.isAuthenticated).toBe(false)
      expect(state.user).toBeNull()
    })

    it('should return false when no refresh token exists', async () => {
      const { refresh } = useAuthStore.getState()
      const result = await refresh()
      expect(result).toBe(false)
    })
  })

  describe('logout', () => {
    it('should clear auth state', async () => {
      const { logoutApi } = await import('@/api/auth')
      vi.mocked(logoutApi).mockResolvedValue(undefined)

      useAuthStore.setState({
        accessToken: 'token',
        refreshToken: 'refresh-token',
        isAuthenticated: true,
        user: { id: '1', username: 'test', role: 'admin' },
      })

      const { logout } = useAuthStore.getState()
      logout()

      const state = useAuthStore.getState()
      expect(state.accessToken).toBeNull()
      expect(state.refreshToken).toBeNull()
      expect(state.user).toBeNull()
      expect(state.isAuthenticated).toBe(false)
      expect(logoutApi).toHaveBeenCalledWith('refresh-token')
    })

    it('should still clear state even if logoutApi fails', async () => {
      const { logoutApi } = await import('@/api/auth')
      vi.mocked(logoutApi).mockRejectedValue(new Error('Network error'))

      useAuthStore.setState({
        accessToken: 'token',
        refreshToken: 'refresh-token',
        isAuthenticated: true,
        user: { id: '1', username: 'test', role: 'admin' },
      })

      const { logout } = useAuthStore.getState()
      logout()

      const state = useAuthStore.getState()
      expect(state.isAuthenticated).toBe(false)
    })
  })
})
