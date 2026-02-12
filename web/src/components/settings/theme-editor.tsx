import { useState, useCallback, useMemo } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  Save,
  X,
  RotateCcw,
  ChevronDown,
  ChevronRight,
  Loader2,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  updateTheme,
  createTheme,
  type ThemeDefinition,
} from '@/api/themes'
import { useThemeStore } from '@/stores/theme'
import {
  TOKEN_CATEGORIES,
  LAYER_CATEGORIES,
  getDefault,
  flattenTokens,
  unflattenTokens,
} from '@/lib/theme-defaults'
import { ColorPicker, TextValueEditor } from './color-picker'
import { ThemePreview } from './theme-preview'

interface ThemeEditorProps {
  theme: ThemeDefinition
  onClose: () => void
}

const COLOR_CATEGORIES = [
  'backgrounds', 'text', 'borders', 'buttons', 'inputs',
  'sidebar', 'status', 'charts',
]

export function ThemeEditor({ theme, onClose }: ThemeEditorProps) {
  const queryClient = useQueryClient()
  const setStoreTheme = useThemeStore((s) => s.setActiveTheme)
  const setDraftOverride = useThemeStore((s) => s.setDraftOverride)
  const clearDraftOverrides = useThemeStore((s) => s.clearDraftOverrides)

  // Track overrides locally for the editor UI
  const [overrides, setOverrides] = useState<Record<string, string>>(() =>
    flattenTokens(theme.tokens),
  )
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    () => new Set(['backgrounds']),
  )
  const [isBuiltInCopy, setIsBuiltInCopy] = useState(false)
  const [copyTheme, setCopyTheme] = useState<ThemeDefinition | null>(null)

  // The theme we're actually editing (copy if built-in)
  const editingTheme = copyTheme ?? theme

  const overrideCount = Object.keys(overrides).length

  const toggleSection = useCallback((section: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev)
      if (next.has(section)) {
        next.delete(section)
      } else {
        next.add(section)
      }
      return next
    })
  }, [])

  function handleTokenChange(varName: string, value: string) {
    const defaultValue = getDefault(varName, editingTheme.base_mode)
    if (value === defaultValue) {
      // Remove override if it matches the default
      setOverrides((prev) => {
        const next = { ...prev }
        delete next[varName]
        return next
      })
      // Reset the CSS property to default
      document.documentElement.style.removeProperty(varName)
    } else {
      setOverrides((prev) => ({ ...prev, [varName]: value }))
      setDraftOverride(varName, value)
    }
  }

  function handleReset(varName: string) {
    setOverrides((prev) => {
      const next = { ...prev }
      delete next[varName]
      return next
    })
    document.documentElement.style.removeProperty(varName)
  }

  function handleResetAll() {
    setOverrides({})
    clearDraftOverrides()
    // Re-apply the base theme without overrides
    useThemeStore.getState().applyTheme({ ...editingTheme, tokens: {} })
  }

  const getTokenValue = useCallback(
    (varName: string) => {
      return overrides[varName] ?? getDefault(varName, editingTheme.base_mode)
    },
    [overrides, editingTheme.base_mode],
  )

  // Create copy mutation (for built-in themes)
  const copyMutation = useMutation({
    mutationFn: () =>
      createTheme({
        name: `${theme.name} (Custom)`,
        base_mode: theme.base_mode,
        tokens: theme.tokens,
      }),
    onSuccess: (newTheme) => {
      setCopyTheme(newTheme)
      setIsBuiltInCopy(true)
      toast.success(`Created editable copy "${newTheme.name}"`)
      queryClient.invalidateQueries({ queryKey: ['settings', 'themes'] })
    },
    onError: () => {
      toast.error('Failed to create copy')
    },
  })

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: () => {
      const tokens = unflattenTokens(overrides)
      return updateTheme(editingTheme.id, { tokens })
    },
    onSuccess: (updatedTheme) => {
      clearDraftOverrides()
      setStoreTheme(updatedTheme)
      toast.success('Theme saved')
      queryClient.invalidateQueries({ queryKey: ['settings', 'themes'] })
    },
    onError: () => {
      toast.error('Failed to save theme')
    },
  })

  function handleCancel() {
    clearDraftOverrides()
    onClose()
  }

  function handleSave() {
    if (theme.built_in && !isBuiltInCopy) {
      copyMutation.mutate()
      return
    }
    saveMutation.mutate()
  }

  // Memoize the resolved values for each category
  const categoryValues = useMemo(() => {
    const result: Record<string, { varName: string; value: string; isOverridden: boolean; defaultValue: string }[]> = {}
    for (const [catKey, catDef] of Object.entries(TOKEN_CATEGORIES)) {
      result[catKey] = catDef.vars.map((varName) => ({
        varName,
        value: getTokenValue(varName),
        isOverridden: varName in overrides,
        defaultValue: getDefault(varName, editingTheme.base_mode),
      }))
    }
    return result
  }, [overrides, editingTheme.base_mode, getTokenValue])

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">
            Editing: {editingTheme.name}
          </h3>
          <p className="text-sm text-muted-foreground">
            {overrideCount} custom override{overrideCount !== 1 ? 's' : ''}
            {' '}&middot; {editingTheme.base_mode} mode
          </p>
        </div>
        <div className="flex gap-2">
          {overrideCount > 0 && (
            <Button variant="ghost" size="sm" onClick={handleResetAll} className="gap-1.5">
              <RotateCcw className="h-3.5 w-3.5" />
              Reset All
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={handleCancel} className="gap-1.5">
            <X className="h-3.5 w-3.5" />
            Cancel
          </Button>
          <Button
            size="sm"
            onClick={handleSave}
            disabled={saveMutation.isPending || copyMutation.isPending}
            className="gap-1.5"
          >
            {(saveMutation.isPending || copyMutation.isPending) ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Save className="h-3.5 w-3.5" />
            )}
            {theme.built_in && !isBuiltInCopy ? 'Save as Copy' : 'Save'}
          </Button>
        </div>
      </div>

      {theme.built_in && !isBuiltInCopy && (
        <div className="rounded-md bg-muted/50 p-3 text-sm text-muted-foreground">
          Built-in themes are read-only. Your changes will be saved as a new custom theme.
        </div>
      )}

      {/* Two-column layout */}
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-6">
        {/* Left: Token editors */}
        <div className="space-y-2">
          {Object.entries(LAYER_CATEGORIES).map(([layerKey, layerDef]) => (
            <div key={layerKey} className="space-y-2">
              <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground px-1">
                {layerDef.label}
              </h4>
              {layerDef.categories.map((catKey) => {
                const catDef = TOKEN_CATEGORIES[catKey]
                if (!catDef) return null
                const isExpanded = expandedSections.has(catKey)
                const catOverrides = categoryValues[catKey]?.filter((t) => t.isOverridden).length ?? 0
                const isColor = COLOR_CATEGORIES.includes(catKey)

                return (
                  <Card key={catKey}>
                    <button
                      type="button"
                      onClick={() => toggleSection(catKey)}
                      className="w-full flex items-center justify-between p-3 text-left hover:bg-muted/30 transition-colors rounded-t-lg"
                    >
                      <div className="flex items-center gap-2">
                        {isExpanded ? (
                          <ChevronDown className="h-4 w-4 text-muted-foreground" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        )}
                        <span className="text-sm font-medium">{catDef.label}</span>
                        <span className="text-xs text-muted-foreground">
                          ({catDef.vars.length})
                        </span>
                      </div>
                      {catOverrides > 0 && (
                        <span className="text-[10px] px-1.5 py-0.5 rounded bg-primary/10 text-primary">
                          {catOverrides} changed
                        </span>
                      )}
                    </button>
                    {isExpanded && (
                      <CardContent className="pt-0 pb-3 px-3">
                        <div className="space-y-0.5">
                          {categoryValues[catKey]?.map((token) =>
                            isColor ? (
                              <ColorPicker
                                key={token.varName}
                                varName={token.varName}
                                value={token.value}
                                defaultValue={token.defaultValue}
                                onChange={(v) => handleTokenChange(token.varName, v)}
                                onReset={() => handleReset(token.varName)}
                                isOverridden={token.isOverridden}
                              />
                            ) : (
                              <TextValueEditor
                                key={token.varName}
                                varName={token.varName}
                                value={token.value}
                                defaultValue={token.defaultValue}
                                onChange={(v) => handleTokenChange(token.varName, v)}
                                onReset={() => handleReset(token.varName)}
                                isOverridden={token.isOverridden}
                              />
                            ),
                          )}
                        </div>
                      </CardContent>
                    )}
                  </Card>
                )
              })}
            </div>
          ))}
        </div>

        {/* Right: Live preview */}
        <div className="sticky top-4">
          <ThemePreview />
        </div>
      </div>
    </div>
  )
}
