import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ThemeDefinition } from '@/api/themes'
import { flattenTokens } from '@/lib/theme-defaults'

interface ThemeState {
  /** ID of the active theme */
  activeThemeId: string
  /** Full active theme definition (null until loaded from API) */
  activeTheme: ThemeDefinition | null
  /** Draft overrides being edited (null when not in edit mode) */
  draftOverrides: Record<string, string> | null
  /** Whether the store has been rehydrated from localStorage */
  isHydrated: boolean

  /** Set the active theme and apply it to the document */
  setActiveTheme: (theme: ThemeDefinition) => void
  /** Set a single draft override (live preview during editing) */
  setDraftOverride: (varName: string, value: string) => void
  /** Clear all draft overrides (cancel editing) */
  clearDraftOverrides: () => void
  /** Commit draft overrides and return the flat map for saving */
  commitDraft: () => Record<string, string> | null
  /** Apply a theme's tokens to the document element */
  applyTheme: (theme: ThemeDefinition) => void
  /** Remove all custom CSS properties from the document */
  clearAppliedTokens: () => void
}

/**
 * Apply a flat map of CSS custom properties to document.documentElement.
 */
function applyTokensToDOM(tokens: Record<string, string>) {
  const root = document.documentElement
  for (const [varName, value] of Object.entries(tokens)) {
    root.style.setProperty(varName, value)
  }
}

/**
 * Remove CSS custom properties from document.documentElement.
 */
function removeTokensFromDOM(varNames: string[]) {
  const root = document.documentElement
  for (const varName of varNames) {
    root.style.removeProperty(varName)
  }
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      activeThemeId: 'builtin-forest-dark',
      activeTheme: null,
      draftOverrides: null,
      isHydrated: false,

      setActiveTheme: (theme: ThemeDefinition) => {
        set({ activeThemeId: theme.id, activeTheme: theme })
        get().applyTheme(theme)
      },

      setDraftOverride: (varName: string, value: string) => {
        const current = get().draftOverrides ?? {}
        const updated = { ...current, [varName]: value }
        set({ draftOverrides: updated })
        // Apply immediately for live preview
        document.documentElement.style.setProperty(varName, value)
      },

      clearDraftOverrides: () => {
        const draft = get().draftOverrides
        if (draft) {
          removeTokensFromDOM(Object.keys(draft))
        }
        set({ draftOverrides: null })
        // Re-apply the active theme to restore correct state
        const theme = get().activeTheme
        if (theme) {
          get().applyTheme(theme)
        }
      },

      commitDraft: () => {
        const draft = get().draftOverrides
        set({ draftOverrides: null })
        return draft
      },

      applyTheme: (theme: ThemeDefinition) => {
        // First clear any existing overrides
        get().clearAppliedTokens()

        // Set base mode (dark/light)
        document.documentElement.setAttribute('data-theme', theme.base_mode)

        // Apply token overrides
        const flat = flattenTokens(theme.tokens)
        if (Object.keys(flat).length > 0) {
          applyTokensToDOM(flat)
        }
      },

      clearAppliedTokens: () => {
        // Remove all --nv-* inline styles that were applied by the theme
        const root = document.documentElement
        const style = root.style
        const toRemove: string[] = []
        for (let i = 0; i < style.length; i++) {
          const prop = style[i]
          if (prop.startsWith('--nv-')) {
            toRemove.push(prop)
          }
        }
        removeTokensFromDOM(toRemove)
      },
    }),
    {
      name: 'nv-theme',
      partialize: (state) => ({
        activeThemeId: state.activeThemeId,
        activeTheme: state.activeTheme,
      }),
      onRehydrateStorage: () => (state) => {
        if (!state) return
        state.isHydrated = true

        // Apply theme immediately on rehydration (before React renders)
        if (state.activeTheme) {
          state.applyTheme(state.activeTheme)
        }
      },
    },
  ),
)
