import { useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Download, Upload, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import {
  createTheme,
  type ThemeTokens,
  type ThemeLayer,
} from '@/api/themes'
import { useThemeStore } from '@/stores/theme'

const THEME_FILE_EXTENSION = '.subnetree-theme.json'

interface ExportedTheme {
  subnetree_theme: true
  version: 1
  name: string
  description: string
  base_mode: 'dark' | 'light'
  layers?: ThemeLayer[]
  tokens: ThemeTokens
}

function isValidThemeFile(data: unknown): data is ExportedTheme {
  if (typeof data !== 'object' || data === null) return false
  const obj = data as Record<string, unknown>
  return (
    obj.subnetree_theme === true &&
    obj.version === 1 &&
    typeof obj.name === 'string' &&
    obj.name.length > 0 &&
    (obj.base_mode === 'dark' || obj.base_mode === 'light') &&
    typeof obj.tokens === 'object' &&
    obj.tokens !== null
  )
}

export function ThemeImportExport() {
  const queryClient = useQueryClient()
  const activeTheme = useThemeStore((s) => s.activeTheme)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const importMutation = useMutation({
    mutationFn: (data: ExportedTheme) =>
      createTheme({
        name: data.name,
        description: data.description || '',
        base_mode: data.base_mode,
        layers: data.layers,
        tokens: data.tokens,
      }),
    onSuccess: (theme) => {
      toast.success(`Imported "${theme.name}"`)
      queryClient.invalidateQueries({ queryKey: ['settings', 'themes'] })
    },
    onError: () => {
      toast.error('Failed to import theme')
    },
  })

  function handleExport() {
    if (!activeTheme) {
      toast.error('No active theme to export')
      return
    }

    const exported: ExportedTheme = {
      subnetree_theme: true,
      version: 1,
      name: activeTheme.name,
      description: activeTheme.description || '',
      base_mode: activeTheme.base_mode,
      layers: activeTheme.layers,
      tokens: activeTheme.tokens,
    }

    const blob = new Blob([JSON.stringify(exported, null, 2)], {
      type: 'application/json',
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${activeTheme.name.toLowerCase().replace(/\s+/g, '-')}${THEME_FILE_EXTENSION}`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    toast.success('Theme exported')
  }

  function handleImport(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (e) => {
      try {
        const data = JSON.parse(e.target?.result as string)
        if (!isValidThemeFile(data)) {
          toast.error('Invalid theme file format')
          return
        }
        importMutation.mutate(data)
      } catch {
        toast.error('Failed to parse theme file')
      }
    }
    reader.readAsText(file)

    // Reset input so the same file can be re-imported
    event.target.value = ''
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Download className="h-4 w-4 text-muted-foreground" />
          Import / Export
        </CardTitle>
        <CardDescription>
          Share themes with others by exporting as a file, or import themes from
          the community.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex gap-3">
          <Button
            variant="outline"
            onClick={handleExport}
            disabled={!activeTheme}
            className="gap-2"
          >
            <Download className="h-4 w-4" />
            Export Active Theme
          </Button>
          <Button
            variant="outline"
            onClick={() => fileInputRef.current?.click()}
            disabled={importMutation.isPending}
            className="gap-2"
          >
            {importMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Upload className="h-4 w-4" />
            )}
            Import Theme
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept=".json"
            onChange={handleImport}
            className="hidden"
          />
        </div>
      </CardContent>
    </Card>
  )
}
