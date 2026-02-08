import { useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useThemeStore } from '@/stores/theme'
import { getActiveTheme, getTheme } from '@/api/themes'
import { useAuthStore } from '@/stores/auth'

/**
 * ThemeProvider syncs the active theme from the backend and applies CSS
 * overrides to document.documentElement. It should wrap the app inside
 * QueryClientProvider.
 *
 * On first render it uses the localStorage-cached theme (via Zustand persist)
 * to avoid FOUC. Then it fetches the server's active theme in the background
 * and updates if needed.
 */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const setActiveTheme = useThemeStore((s) => s.setActiveTheme)
  const currentThemeId = useThemeStore((s) => s.activeThemeId)
  const hasAppliedRef = useRef(false)

  // Fetch the server's active theme reference (only when authenticated)
  const { data: activeRef } = useQuery({
    queryKey: ['theme', 'active'],
    queryFn: getActiveTheme,
    enabled: isAuthenticated,
    staleTime: 60 * 1000,
  })

  // Fetch the full theme definition when the active ID changes
  const serverThemeId = activeRef?.theme_id
  const { data: serverTheme } = useQuery({
    queryKey: ['theme', serverThemeId],
    queryFn: () => getTheme(serverThemeId!),
    enabled: !!serverThemeId,
    staleTime: 60 * 1000,
  })

  // Apply server theme if it differs from the cached one
  useEffect(() => {
    if (!serverTheme) return
    if (serverTheme.id !== currentThemeId || !hasAppliedRef.current) {
      setActiveTheme(serverTheme)
      hasAppliedRef.current = true
    }
  }, [serverTheme, currentThemeId, setActiveTheme])

  return <>{children}</>
}
