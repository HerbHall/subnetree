import { useAuthStore } from '@/stores/auth'

/** Convenience hook for auth state and actions. */
export function useAuth() {
  const user = useAuthStore((s) => s.user)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isHydrated = useAuthStore((s) => s.isHydrated)
  const login = useAuthStore((s) => s.login)
  const logout = useAuthStore((s) => s.logout)

  return { user, isAuthenticated, isHydrated, login, logout }
}
