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
  Brain,
  Key,
  Copy,
  Check,
  AlertTriangle,
  Globe,
  RefreshCw,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import {
  getNetworkInterfaces,
  getScanInterface,
  setScanInterface,
} from '@/api/settings'
import type { NetworkInterface } from '@/api/settings'
import { createEnrollmentToken } from '@/api/agents'
import { LLMConfig as LLMConfigPanel } from '@/components/settings/llm-config'
import { ThemeSelector } from '@/components/settings/theme-selector'
import { ThemeImportExport } from '@/components/settings/theme-import-export'
import { ThemeEditor } from '@/components/settings/theme-editor'
import type { ThemeDefinition } from '@/api/themes'
import { HelpIcon, HelpPopover } from '@/components/contextual-help'
import { getTailscaleStatus, triggerTailscaleSync } from '@/api/tailscale'
import type { TailscaleStatus } from '@/api/tailscale'

type SettingsTab = 'network' | 'appearance' | 'llm' | 'agents' | 'tailscale'

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
        <div className="flex items-center">
          <TabButton
            active={activeTab === 'llm'}
            onClick={() => setActiveTab('llm')}
            icon={<Brain className="h-4 w-4" />}
            label="AI / LLM"
          />
          <HelpIcon content="Configure an AI model for natural language device queries and analytics. Supports local models via Ollama or cloud providers." />
        </div>
        <TabButton
          active={activeTab === 'agents'}
          onClick={() => setActiveTab('agents')}
          icon={<Key className="h-4 w-4" />}
          label="Agents"
        />
        <TabButton
          active={activeTab === 'tailscale'}
          onClick={() => setActiveTab('tailscale')}
          icon={<Globe className="h-4 w-4" />}
          label="Tailscale"
        />
      </div>

      {/* Tab content */}
      {activeTab === 'network' && <NetworkTab />}
      {activeTab === 'appearance' && <AppearanceTab />}
      {activeTab === 'llm' && <LLMConfigPanel />}
      {activeTab === 'agents' && <AgentsTab />}
      {activeTab === 'tailscale' && <TailscaleTab />}
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
      <div className="flex items-center gap-2 mb-2">
        <h2 className="text-sm font-medium text-muted-foreground">Theme</h2>
        <HelpIcon content="Customize the dashboard appearance. Changes are applied immediately. You can also import and export themes to share with others." />
      </div>
      <ThemeSelector onCustomize={handleCustomize} />
      <ThemeImportExport />
    </div>
  )
}

const EXPIRY_OPTIONS = [
  { label: '1 hour', value: '1h' },
  { label: '24 hours', value: '24h' },
  { label: '7 days', value: '168h' },
  { label: '30 days', value: '720h' },
]

function AgentsTab() {
  const [description, setDescription] = useState('')
  const [maxUses, setMaxUses] = useState(1)
  const [expiresIn, setExpiresIn] = useState('24h')
  const [generatedToken, setGeneratedToken] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const tokenMutation = useMutation({
    mutationFn: () =>
      createEnrollmentToken({
        description,
        max_uses: maxUses,
        expires_in: expiresIn,
      }),
    onSuccess: (data) => {
      setGeneratedToken(data.token)
      setDescription('')
      setMaxUses(1)
      setExpiresIn('24h')
      toast.success('Enrollment token generated')
    },
    onError: () => {
      toast.error('Failed to generate enrollment token')
    },
  })

  const handleCopy = useCallback(() => {
    if (!generatedToken) return
    navigator.clipboard.writeText(generatedToken)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [generatedToken])

  const handleDismiss = useCallback(() => {
    setGeneratedToken(null)
    setCopied(false)
  }, [])

  return (
    <div className="space-y-4">
      {/* Token display (shown after generation) */}
      {generatedToken && (
        <Card className="border-amber-500/50 bg-amber-500/5">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2 text-amber-600 dark:text-amber-400">
              <AlertTriangle className="h-4 w-4" />
              Token Generated
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Copy this token now. It will not be shown again.
            </p>
            <div className="flex items-center gap-2">
              <Input
                readOnly
                value={generatedToken}
                className="font-mono text-xs"
              />
              <Button
                variant="outline"
                size="icon"
                className="shrink-0"
                onClick={handleCopy}
              >
                {copied ? (
                  <Check className="h-4 w-4 text-green-500" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
            <Button variant="ghost" size="sm" onClick={handleDismiss}>
              Done
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Enrollment form */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Key className="h-4 w-4 text-muted-foreground" />
            Agent Enrollment
            <HelpPopover title="Enrollment Tokens">
              <p className="text-xs text-muted-foreground">
                A one-time token that authorizes a new Scout agent to register with this SubNetree server.
                Generate a token, then pass it to the agent's install command. Tokens expire after the configured duration.
              </p>
            </HelpPopover>
          </CardTitle>
          <CardDescription>
            Generate enrollment tokens for Scout agents to register with this server.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="token-description">Description</Label>
              <Input
                id="token-description"
                placeholder="e.g., Lab server enrollment"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                disabled={tokenMutation.isPending}
              />
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="token-max-uses">Max Uses</Label>
                <Input
                  id="token-max-uses"
                  type="number"
                  min={1}
                  value={maxUses}
                  onChange={(e) => setMaxUses(Math.max(1, parseInt(e.target.value) || 1))}
                  disabled={tokenMutation.isPending}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="token-expires">Expires In</Label>
                <select
                  id="token-expires"
                  value={expiresIn}
                  onChange={(e) => setExpiresIn(e.target.value)}
                  disabled={tokenMutation.isPending}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {EXPIRY_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <Button
              onClick={() => tokenMutation.mutate()}
              disabled={!description.trim() || tokenMutation.isPending}
              className="gap-2"
            >
              {tokenMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Key className="h-4 w-4" />
              )}
              {tokenMutation.isPending ? 'Generating...' : 'Generate Token'}
            </Button>
          </div>
        </CardContent>
      </Card>
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
              <Label htmlFor="scan-interface">
                Network Interface
                <HelpIcon content="The network interface used for scanning. Select the interface connected to the network you want to monitor, or use Auto-detect to let SubNetree choose." />
              </Label>
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

function TailscaleTab() {
  const {
    data: status,
    isLoading,
    error,
  } = useQuery<TailscaleStatus>({
    queryKey: ['settings', 'tailscale-status'],
    queryFn: getTailscaleStatus,
    refetchInterval: 30000,
  })

  const syncMutation = useMutation({
    mutationFn: triggerTailscaleSync,
    onSuccess: (data) => {
      toast.success(
        `Tailscale sync complete: ${data.devices_found} found, ${data.created} created, ${data.updated} updated`
      )
    },
    onError: () => {
      toast.error('Tailscale sync failed')
    },
  })

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <Globe className="h-4 w-4 text-muted-foreground" />
            Tailscale Integration
            <HelpPopover title="Tailscale Integration">
              <p className="text-xs text-muted-foreground">
                Syncs devices from your Tailscale tailnet into SubNetree. Devices are matched by hostname or IP
                and merged with existing records. Configure a Tailscale API key credential in the Vault first.
              </p>
            </HelpPopover>
          </CardTitle>
          <CardDescription>
            Discover and sync devices from your Tailscale tailnet overlay network.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : error ? (
            <p className="text-sm text-red-400 py-4">
              Failed to load Tailscale status. Ensure the server is running.
            </p>
          ) : (
            <div className="space-y-4">
              {/* Status */}
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium">Status:</span>
                {status?.enabled ? (
                  <span className="text-xs px-2 py-0.5 rounded bg-green-500/10 text-green-500">
                    Enabled
                  </span>
                ) : (
                  <span className="text-xs px-2 py-0.5 rounded bg-muted text-muted-foreground">
                    Disabled
                  </span>
                )}
              </div>

              {/* Last sync info */}
              {status?.last_sync_time && (
                <div className="rounded-md border p-3 space-y-2">
                  <p className="text-xs text-muted-foreground">
                    Last sync: {new Date(status.last_sync_time).toLocaleString()}
                  </p>
                  {status.last_sync_result && (
                    <div className="grid grid-cols-4 gap-2 text-center">
                      <div>
                        <p className="text-lg font-semibold">{status.last_sync_result.devices_found}</p>
                        <p className="text-xs text-muted-foreground">Found</p>
                      </div>
                      <div>
                        <p className="text-lg font-semibold text-green-500">{status.last_sync_result.created}</p>
                        <p className="text-xs text-muted-foreground">Created</p>
                      </div>
                      <div>
                        <p className="text-lg font-semibold text-blue-500">{status.last_sync_result.updated}</p>
                        <p className="text-xs text-muted-foreground">Updated</p>
                      </div>
                      <div>
                        <p className="text-lg font-semibold">{status.last_sync_result.unchanged}</p>
                        <p className="text-xs text-muted-foreground">Unchanged</p>
                      </div>
                    </div>
                  )}
                  {status.error && (
                    <p className="text-xs text-red-400 flex items-center gap-1">
                      <AlertTriangle className="h-3 w-3" />
                      {status.error}
                    </p>
                  )}
                </div>
              )}

              {/* Sync button */}
              {status?.enabled && (
                <Button
                  onClick={() => syncMutation.mutate()}
                  disabled={syncMutation.isPending}
                  className="gap-2"
                >
                  {syncMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <RefreshCw className="h-4 w-4" />
                  )}
                  {syncMutation.isPending ? 'Syncing...' : 'Sync Now'}
                </Button>
              )}

              {/* Configuration help */}
              {!status?.enabled && (
                <div className="rounded-md border border-dashed p-4 text-center">
                  <p className="text-sm text-muted-foreground">
                    To enable Tailscale integration, add the following to your configuration file:
                  </p>
                  <pre className="mt-2 text-xs bg-muted/50 rounded p-2 text-left inline-block">
{`plugins:
  tailscale:
    enabled: true
    credential_id: "<vault-credential-id>"
    tailnet: "-"
    sync_interval: 5m`}
                  </pre>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
