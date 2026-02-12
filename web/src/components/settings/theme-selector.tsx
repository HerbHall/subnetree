import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  Palette,
  Trash2,
  Sun,
  Moon,
  Check,
  Loader2,
  Plus,
  Copy,
  Paintbrush,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  listThemes,
  createTheme,
  deleteTheme,
  setActiveTheme as setActiveThemeApi,
  type ThemeDefinition,
} from '@/api/themes'
import { useThemeStore } from '@/stores/theme'

interface ThemeSelectorProps {
  onCustomize?: (theme: ThemeDefinition) => void
}

export function ThemeSelector({ onCustomize }: ThemeSelectorProps) {
  const queryClient = useQueryClient()
  const activeThemeId = useThemeStore((s) => s.activeThemeId)
  const setStoreTheme = useThemeStore((s) => s.setActiveTheme)
  const [showNewTheme, setShowNewTheme] = useState(false)
  const [newThemeName, setNewThemeName] = useState('')
  const [newThemeMode, setNewThemeMode] = useState<'dark' | 'light'>('dark')

  const {
    data: themes,
    isLoading,
  } = useQuery({
    queryKey: ['settings', 'themes'],
    queryFn: listThemes,
  })

  const activateMutation = useMutation({
    mutationFn: async (theme: ThemeDefinition) => {
      await setActiveThemeApi(theme.id)
      return theme
    },
    onSuccess: (theme) => {
      setStoreTheme(theme)
      toast.success(`Switched to "${theme.name}"`)
      queryClient.invalidateQueries({ queryKey: ['settings', 'themes'] })
    },
    onError: () => {
      toast.error('Failed to activate theme')
    },
  })

  const createMutation = useMutation({
    mutationFn: () =>
      createTheme({
        name: newThemeName.trim(),
        base_mode: newThemeMode,
        tokens: {},
      }),
    onSuccess: (theme) => {
      toast.success(`Created "${theme.name}"`)
      setShowNewTheme(false)
      setNewThemeName('')
      queryClient.invalidateQueries({ queryKey: ['settings', 'themes'] })
    },
    onError: () => {
      toast.error('Failed to create theme')
    },
  })

  const duplicateMutation = useMutation({
    mutationFn: (source: ThemeDefinition) =>
      createTheme({
        name: `${source.name} (Copy)`,
        base_mode: source.base_mode,
        tokens: source.tokens,
      }),
    onSuccess: (theme) => {
      toast.success(`Created "${theme.name}"`)
      queryClient.invalidateQueries({ queryKey: ['settings', 'themes'] })
    },
    onError: () => {
      toast.error('Failed to duplicate theme')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteTheme,
    onSuccess: () => {
      toast.success('Theme deleted')
      queryClient.invalidateQueries({ queryKey: ['settings', 'themes'] })
    },
    onError: () => {
      toast.error('Failed to delete theme')
    },
  })

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Palette className="h-4 w-4 text-muted-foreground" />
              Themes
            </CardTitle>
            <CardDescription className="mt-1.5">
              Select a theme or create a custom one. Themes control all colors,
              fonts, and visual styling.
            </CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowNewTheme(!showNewTheme)}
            className="gap-1.5"
          >
            <Plus className="h-3.5 w-3.5" />
            New
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {/* New theme form */}
        {showNewTheme && (
          <div className="mb-4 p-3 rounded-lg border border-dashed space-y-3">
            <div className="space-y-2">
              <Label htmlFor="new-theme-name">Theme Name</Label>
              <Input
                id="new-theme-name"
                value={newThemeName}
                onChange={(e) => setNewThemeName(e.target.value)}
                placeholder="My Custom Theme"
                autoFocus
              />
            </div>
            <div className="space-y-2">
              <Label>Base Mode</Label>
              <div className="flex gap-2">
                <Button
                  type="button"
                  variant={newThemeMode === 'dark' ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setNewThemeMode('dark')}
                  className="gap-1.5"
                >
                  <Moon className="h-3.5 w-3.5" />
                  Dark
                </Button>
                <Button
                  type="button"
                  variant={newThemeMode === 'light' ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setNewThemeMode('light')}
                  className="gap-1.5"
                >
                  <Sun className="h-3.5 w-3.5" />
                  Light
                </Button>
              </div>
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                onClick={() => createMutation.mutate()}
                disabled={!newThemeName.trim() || createMutation.isPending}
              >
                {createMutation.isPending ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin mr-1.5" />
                ) : null}
                Create
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setShowNewTheme(false)
                  setNewThemeName('')
                }}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}

        {/* Theme list */}
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="space-y-2">
            {themes?.map((theme) => (
              <ThemeCard
                key={theme.id}
                theme={theme}
                isActive={theme.id === activeThemeId}
                onActivate={() => activateMutation.mutate(theme)}
                onCustomize={onCustomize ? () => onCustomize(theme) : undefined}
                onDuplicate={() => duplicateMutation.mutate(theme)}
                onDelete={() => deleteMutation.mutate(theme.id)}
                isActivating={activateMutation.isPending}
                isDeleting={deleteMutation.isPending}
              />
            ))}
            {themes?.length === 0 && (
              <p className="text-sm text-muted-foreground text-center py-4">
                No themes found. Create one to get started.
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ThemeCard({
  theme,
  isActive,
  onActivate,
  onCustomize,
  onDuplicate,
  onDelete,
  isActivating,
  isDeleting,
}: {
  theme: ThemeDefinition
  isActive: boolean
  onActivate: () => void
  onCustomize?: () => void
  onDuplicate: () => void
  onDelete: () => void
  isActivating: boolean
  isDeleting: boolean
}) {
  const tokenCount = Object.values(theme.tokens).reduce(
    (sum, cat) => sum + (cat ? Object.keys(cat).length : 0),
    0,
  )

  return (
    <div
      className={`flex items-center gap-3 p-3 rounded-lg border transition-colors ${
        isActive
          ? 'border-primary bg-primary/5'
          : 'border-border hover:border-muted-foreground/30'
      }`}
    >
      {/* Color preview swatch */}
      <div className="shrink-0">
        <div
          className={`h-10 w-10 rounded-md border ${
            theme.base_mode === 'dark' ? 'bg-zinc-900' : 'bg-zinc-100'
          } flex items-center justify-center`}
        >
          {theme.base_mode === 'dark' ? (
            <Moon className="h-4 w-4 text-zinc-400" />
          ) : (
            <Sun className="h-4 w-4 text-zinc-600" />
          )}
        </div>
      </div>

      {/* Theme info */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium truncate">{theme.name}</span>
          {theme.built_in && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground shrink-0">
              Built-in
            </span>
          )}
          {isActive && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-green-500/10 text-green-500 shrink-0">
              Active
            </span>
          )}
          {theme.layers && theme.layers.length > 0 && theme.layers.length < 4 && (
            <>
              {theme.layers.map((layer) => (
                <span
                  key={layer}
                  className="text-[10px] px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-500 shrink-0 capitalize"
                >
                  {layer}
                </span>
              ))}
            </>
          )}
        </div>
        <div className="flex items-center gap-2 mt-0.5">
          <span className="text-xs text-muted-foreground capitalize">
            {theme.base_mode}
          </span>
          {tokenCount > 0 && (
            <span className="text-xs text-muted-foreground">
              {tokenCount} custom token{tokenCount !== 1 ? 's' : ''}
            </span>
          )}
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1 shrink-0">
        {!isActive && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onActivate}
            disabled={isActivating}
            className="h-8 gap-1.5"
          >
            <Check className="h-3.5 w-3.5" />
            Use
          </Button>
        )}
        {onCustomize && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onCustomize}
            className="h-8 gap-1.5"
            title="Customize theme"
          >
            <Paintbrush className="h-3.5 w-3.5" />
            Edit
          </Button>
        )}
        <Button
          variant="ghost"
          size="icon"
          onClick={onDuplicate}
          className="h-8 w-8"
          title="Duplicate theme"
        >
          <Copy className="h-3.5 w-3.5" />
        </Button>
        {!theme.built_in && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onDelete}
            disabled={isDeleting || isActive}
            className="h-8 w-8 text-destructive hover:text-destructive"
            title={isActive ? 'Cannot delete active theme' : 'Delete theme'}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        )}
      </div>
    </div>
  )
}
