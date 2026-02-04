import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { jwtDecode } from 'jwt-decode'
import { loginApi, refreshApi, logoutApi } from '@/api/auth'

interface JwtPayload {
  uid: string
  usr: string
  role: string
  exp: number
}

interface AuthUser {
  id: string
  username: string
  role: string
}

interface AuthState {
  accessToken: string | null
  refreshToken: string | null
  user: AuthUser | null
  isAuthenticated: boolean
  isHydrated: boolean

  login: (username: string, password: string) => Promise<void>
  refresh: () => Promise<boolean>
  logout: () => void
  setTokens: (accessToken: string, refreshToken: string) => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      isAuthenticated: false,
      isHydrated: false,

      setTokens: (accessToken: string, refreshToken: string) => {
        const decoded = jwtDecode<JwtPayload>(accessToken)
        set({
          accessToken,
          refreshToken,
          user: { id: decoded.uid, username: decoded.usr, role: decoded.role },
          isAuthenticated: true,
        })
      },

      login: async (username: string, password: string) => {
        const pair = await loginApi(username, password)
        get().setTokens(pair.access_token, pair.refresh_token)
      },

      refresh: async () => {
        const { refreshToken } = get()
        if (!refreshToken) return false
        try {
          const pair = await refreshApi(refreshToken)
          get().setTokens(pair.access_token, pair.refresh_token)
          return true
        } catch {
          get().logout()
          return false
        }
      },

      logout: () => {
        const { refreshToken } = get()
        if (refreshToken) {
          logoutApi(refreshToken).catch(() => {})
        }
        set({
          accessToken: null,
          refreshToken: null,
          user: null,
          isAuthenticated: false,
        })
      },
    }),
    {
      name: 'nv-auth',
      partialize: (state) => ({
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
      }),
      onRehydrateStorage: () => (state) => {
        if (!state) return
        state.isHydrated = true

        if (state.accessToken) {
          try {
            const decoded = jwtDecode<JwtPayload>(state.accessToken)
            if (decoded.exp * 1000 > Date.now()) {
              state.user = { id: decoded.uid, username: decoded.usr, role: decoded.role }
              state.isAuthenticated = true
            } else {
              // Token expired -- will need refresh on next API call
              state.accessToken = null
              state.user = null
              state.isAuthenticated = false
            }
          } catch {
            state.accessToken = null
            state.user = null
            state.isAuthenticated = false
          }
        }
      },
    },
  ),
)
