import { useState, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  Network,
  Wifi,
  WifiOff,
  Save,
  Loader2,
  CheckCircle2,
  RotateCcw,
  Palette,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import {
  getNetworkInterfaces,
  getScanInterface,
  setScanInterface,
} from '@/api/settings'
import type { NetworkInterface } from '@/api/settings'
import { ThemeSelector } from '@/components/settings/theme-selector'
import { ThemeImportExport } from '@/components/settings/theme-import-export'
import { ThemeEditor } from '@/components/settings/theme-editor'
import type { ThemeDefinition } from '@/api/themes'

type SettingsTab = 'network' | 'appearance'

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('network')

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold">Settings</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Server configuration and preferences
        </p>
      </div>

      {/* Tab bar */}
      <div className="flex border-b">
        <TabButton
          active={activeTab === 'network'}
          onClick={() => setActiveTab('network')}
          icon={<Network className="h-4 w-4" />}
          label="Network"
        />
        <TabButton
          active={activeTab === 'appearance'}
          onClick={() => setActiveTab('appearance')}
          icon={<Palette className="h-4 w-4" />}
          label="Appearance"
        />
      </div>

      {/* Tab content */}
      {activeTab === 'network' && <NetworkTab />}
      {activeTab === 'appearance' && <AppearanceTab />}
    </div>
  )
}

function TabButton({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean
  onClick: () => void
  icon: React.ReactNode
  label: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
        active
          ? 'border-primary text-foreground'
          : 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground/30'
      }`}
    >
      {icon}
      {label}
    </button>
  )
}

function AppearanceTab() {
  const [editingTheme, setEditingTheme] = useState<ThemeDefinition | null>(null)

  const handleCustomize = useCallback((theme: ThemeDefinition) => {
    setEditingTheme(theme)
  }, [])

  if (editingTheme) {
    return (
      <ThemeEditor
        theme={editingTheme}
        onClose={() => setEditingTheme(null)}
      />
    )
  }

  return (
    <div className="space-y-6">
      <ThemeSelector onCustomize={handleCustomize} />
      <ThemeImportExport />
    </div>
  )
}

function NetworkTab() {
  const queryClient = useQueryClient()

  const {
    data: interfaces,
    isLoading: interfacesLoading,
    error: interfacesError,
  } = useQuery({
    queryKey: ['settings', 'interfaces'],
    queryFn: getNetworkInterfaces,
  })

  const {
    data: currentSetting,
    isLoading: settingLoading,
  } = useQuery({
    queryKey: ['settings', 'scan-interface'],
    queryFn: getScanInterface,
  })

  const [localOverride, setLocalOverride] = useState<string | null>(null)
  const selectedInterface = localOverride ?? currentSetting?.interface_name ?? ''
  const setSelectedInterface = (value: string) => setLocalOverride(value)

  const saveMutation = useMutation({
    mutationFn: (interfaceName: string) => setScanInterface(interfaceName),
    onSuccess: () => {
      toast.success('Scan interface saved successfully')
      setLocalOverride(null)
      queryClient.invalidateQueries({ queryKey: ['settings', 'scan-interface'] })
    },
    onError: () => {
      toast.error('Failed to save scan interface')
    },
  })

  const isLoading = interfacesLoading || settingLoading
  const hasChanges = localOverride !== null && localOverride !== (currentSetting?.interface_name ?? '')
  const upInterfaces = interfaces?.filter((iface) => iface.status === 'up') ?? []

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Network className="h-4 w-4 text-muted-foreground" />
          Scan Interface
        </CardTitle>
        <CardDescription>
          Select the network interface used for device discovery scans.
          Choose &quot;Auto-detect&quot; to let SubNetree pick the best
          interface automatically.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : interfacesError ? (
          <p className="text-sm text-red-400 py-4">
            Failed to load network interfaces. Ensure the server is running.
          </p>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="scan-interface">Network Interface</Label>
              <select
                id="scan-interface"
                value={selectedInterface}
                onChange={(e) => setSelectedInterface(e.target.value)}
                disabled={saveMutation.isPending}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <option value="">Auto-detect</option>
                {interfaces?.map((iface) => (
                  <option key={iface.name} value={iface.name}>
                    {iface.name} - {iface.ip_address} ({iface.subnet})
                  </option>
                ))}
              </select>
            </div>

            {interfaces && interfaces.length > 0 && (
              <div className="rounded-md border">
                <div className="px-4 py-2 border-b bg-muted/30">
                  <p className="text-xs font-medium text-muted-foreground">
                    Available Interfaces
                  </p>
                </div>
                <div className="divide-y">
                  {interfaces.map((iface) => (
                    <InterfaceRow
                      key={iface.name}
                      iface={iface}
                      selected={iface.name === selectedInterface}
                      onSelect={() => setSelectedInterface(iface.name)}
                    />
                  ))}
                </div>
                {upInterfaces.length === 0 && (
                  <div className="px-4 py-3 text-sm text-muted-foreground text-center">
                    No active interfaces detected.
                  </div>
                )}
              </div>
            )}

            <div className="flex items-center gap-3 pt-2">
              <Button
                onClick={() => saveMutation.mutate(selectedInterface)}
                disabled={!hasChanges || saveMutation.isPending}
                className="gap-2"
              >
                {saveMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : saveMutation.isSuccess && !hasChanges ? (
                  <CheckCircle2 className="h-4 w-4" />
                ) : (
                  <Save className="h-4 w-4" />
                )}
                {saveMutation.isPending ? 'Saving...' : 'Save'}
              </Button>
              {hasChanges && (
                <Button
                  variant="ghost"
                  onClick={() => setLocalOverride(null)}
                  disabled={saveMutation.isPending}
                  className="gap-2"
                >
                  <RotateCcw className="h-4 w-4" />
                  Reset
                </Button>
              )}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function InterfaceRow({
  iface,
  selected,
  onSelect,
}: {
  iface: NetworkInterface
  selected: boolean
  onSelect: () => void
}) {
  const isUp = iface.status === 'up'

  return (
    <button
      type="button"
      onClick={onSelect}
      className={`w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-muted/50 transition-colors ${
        selected ? 'bg-muted/50 border-l-2 border-l-green-500' : ''
      }`}
    >
      {isUp ? (
        <Wifi className="h-4 w-4 text-green-500 shrink-0" />
      ) : (
        <WifiOff className="h-4 w-4 text-muted-foreground shrink-0" />
      )}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{iface.name}</span>
          <span
            className={`text-[10px] px-1.5 py-0.5 rounded ${
              isUp
                ? 'bg-green-500/10 text-green-500'
                : 'bg-muted text-muted-foreground'
            }`}
          >
            {iface.status}
          </span>
          {selected && (
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-green-500/10 text-green-500">
              selected
            </span>
          )}
        </div>
        <div className="flex items-center gap-3 mt-0.5">
          <span className="text-xs text-muted-foreground font-mono">
            {iface.ip_address}
          </span>
          <span className="text-xs text-muted-foreground">{iface.subnet}</span>
          {iface.mac && (
            <span className="text-xs text-muted-foreground font-mono">
              {iface.mac}
            </span>
          )}
        </div>
      </div>
    </button>
  )
}
