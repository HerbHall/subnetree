import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Shield,
  Lock,
  Unlock,
  Key,
  Plus,
  Trash2,
  Edit2,
  Eye,
  EyeOff,
  X,
  Check,
  FileText,
  AlertTriangle,
} from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { HelpPopover, HelpIcon, FieldHelp } from '@/components/contextual-help'
import {
  listCredentials,
  createCredential,
  updateCredential,
  deleteCredential,
  getCredentialData,
  getVaultStatus,
  sealVault,
  unsealVault,
  getAuditLog,
} from '@/api/vault'
import {
  CREDENTIAL_TYPES,
  type CredentialMeta,
  type CredentialType,
  type CreateCredentialRequest,
  type UpdateCredentialRequest,
} from '@/pages/vault-types'

type TabId = 'credentials' | 'audit'

const tabs: { id: TabId; label: string; icon: React.ElementType }[] = [
  { id: 'credentials', label: 'Credentials', icon: Key },
  { id: 'audit', label: 'Audit Log', icon: FileText },
]

export function VaultPage() {
  const [activeTab, setActiveTab] = useState<TabId>('credentials')
  const queryClient = useQueryClient()

  const { data: vaultStatus, isLoading: statusLoading } = useQuery({
    queryKey: ['vault-status'],
    queryFn: getVaultStatus,
  })

  const sealMutation = useMutation({
    mutationFn: sealVault,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['vault-status'] })
      toast.success('Vault sealed')
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to seal vault'),
  })

  const unsealMutation = useMutation({
    mutationFn: (passphrase: string) => unsealVault(passphrase),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['vault-status'] })
      queryClient.invalidateQueries({ queryKey: ['credentials'] })
      toast.success('Vault unsealed')
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to unseal vault'),
  })

  if (statusLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-24" />
        <Skeleton className="h-64" />
      </div>
    )
  }

  const isSealed = vaultStatus?.sealed ?? true
  const isInitialized = vaultStatus?.initialized ?? false

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold flex items-center gap-2">
          <Shield className="h-6 w-6" />
          Credential Vault
        </h1>
        <p className="text-sm text-muted-foreground mt-1">
          Encrypted credential storage for network devices
        </p>
      </div>

      {/* Vault Status Banner */}
      <VaultStatusBanner
        isSealed={isSealed}
        isInitialized={isInitialized}
        credentialCount={vaultStatus?.credential_count ?? 0}
        onSeal={() => sealMutation.mutate()}
        sealPending={sealMutation.isPending}
      />

      {/* Sealed / Uninitialized state */}
      {isSealed && (
        <UnsealForm
          isInitialized={isInitialized}
          onUnseal={(passphrase) => unsealMutation.mutate(passphrase)}
          isPending={unsealMutation.isPending}
        />
      )}

      {/* Main content when unsealed */}
      {!isSealed && (
        <>
          {/* Tab bar */}
          <div className="flex items-center gap-1 border-b">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={cn(
                  'flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 transition-colors -mb-px',
                  activeTab === tab.id
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground/30'
                )}
              >
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab content */}
          {activeTab === 'credentials' && <CredentialsTab />}
          {activeTab === 'audit' && <AuditTab />}
        </>
      )}
    </div>
  )
}

// ============================================================================
// Vault Status Banner
// ============================================================================

function VaultStatusBanner({
  isSealed,
  isInitialized,
  credentialCount,
  onSeal,
  sealPending,
}: {
  isSealed: boolean
  isInitialized: boolean
  credentialCount: number
  onSeal: () => void
  sealPending: boolean
}) {
  return (
    <Card>
      <CardContent className="py-4">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <div className="flex items-center gap-3">
            {isSealed ? (
              <Lock className="h-5 w-5 text-amber-500" />
            ) : (
              <Unlock className="h-5 w-5 text-green-500" />
            )}
            <div>
              <p className="text-sm font-medium">
                Vault is {isSealed ? 'sealed' : 'unsealed'}
                {!isInitialized && ' (not initialized)'}
                <HelpPopover title="Vault Seal / Unseal">
                  <p className="text-xs text-muted-foreground">
                    When sealed, the vault encrypts all stored credentials and they cannot be used.
                    Unsealing requires your passphrase and allows SubNetree to use credentials for scanning and remote access.
                  </p>
                </HelpPopover>
              </p>
              {!isSealed && (
                <p className="text-xs text-muted-foreground">
                  {credentialCount} credential{credentialCount !== 1 ? 's' : ''} stored
                </p>
              )}
            </div>
          </div>
          {!isSealed && (
            <Button
              variant="outline"
              size="sm"
              onClick={onSeal}
              disabled={sealPending}
              className="gap-2"
            >
              <Lock className="h-3.5 w-3.5" />
              {sealPending ? 'Sealing...' : 'Seal Vault'}
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

// ============================================================================
// Unseal Form
// ============================================================================

function UnsealForm({
  isInitialized,
  onUnseal,
  isPending,
}: {
  isInitialized: boolean
  onUnseal: (passphrase: string) => void
  isPending: boolean
}) {
  const [passphrase, setPassphrase] = useState('')
  const [confirmPassphrase, setConfirmPassphrase] = useState('')
  const [showPassphrase, setShowPassphrase] = useState(false)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!isInitialized && passphrase !== confirmPassphrase) {
      toast.error('Passphrases do not match')
      return
    }
    onUnseal(passphrase)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Key className="h-4 w-4 text-muted-foreground" />
          {isInitialized ? 'Unseal Vault' : 'Initialize Vault'}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4 max-w-md">
          <p className="text-sm text-muted-foreground">
            {isInitialized
              ? 'Enter the vault passphrase to unseal and access credentials.'
              : 'Set a passphrase to initialize the vault. This passphrase encrypts all stored credentials.'}
          </p>
          <div className="space-y-2">
            <Label htmlFor="passphrase">Passphrase</Label>
            <div className="relative">
              <Input
                id="passphrase"
                type={showPassphrase ? 'text' : 'password'}
                value={passphrase}
                onChange={(e) => setPassphrase(e.target.value)}
                placeholder="Enter vault passphrase"
                required
                autoFocus
              />
              <button
                type="button"
                onClick={() => setShowPassphrase(!showPassphrase)}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showPassphrase ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          {!isInitialized && (
            <div className="space-y-2">
              <Label htmlFor="confirm-passphrase">Confirm Passphrase</Label>
              <Input
                id="confirm-passphrase"
                type={showPassphrase ? 'text' : 'password'}
                value={confirmPassphrase}
                onChange={(e) => setConfirmPassphrase(e.target.value)}
                placeholder="Confirm vault passphrase"
                required
              />
            </div>
          )}
          <Button type="submit" disabled={isPending} className="gap-2">
            <Unlock className="h-4 w-4" />
            {isPending ? 'Processing...' : isInitialized ? 'Unseal' : 'Initialize & Unseal'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

// ============================================================================
// Credentials Tab
// ============================================================================

function CredentialsTab() {
  const queryClient = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editingCredential, setEditingCredential] = useState<CredentialMeta | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [revealedData, setRevealedData] = useState<Record<string, unknown> | null>(null)
  const [revealedId, setRevealedId] = useState<string | null>(null)

  const { data: credentials, isLoading } = useQuery({
    queryKey: ['credentials'],
    queryFn: listCredentials,
  })

  const createMutation = useMutation({
    mutationFn: (req: CreateCredentialRequest) => createCredential(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['credentials'] })
      queryClient.invalidateQueries({ queryKey: ['vault-status'] })
      toast.success('Credential created')
      setShowForm(false)
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to create credential'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, req }: { id: string; req: UpdateCredentialRequest }) =>
      updateCredential(id, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['credentials'] })
      toast.success('Credential updated')
      setEditingCredential(null)
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to update credential'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteCredential(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['credentials'] })
      queryClient.invalidateQueries({ queryKey: ['vault-status'] })
      toast.success('Credential deleted')
      setDeletingId(null)
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to delete credential'),
  })

  const revealMutation = useMutation({
    mutationFn: (id: string) => getCredentialData(id, 'manual_inspection'),
    onSuccess: (data) => {
      setRevealedData(data.data)
      setRevealedId(data.id)
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to retrieve credential data'),
  })

  function handleReveal(id: string) {
    if (revealedId === id) {
      setRevealedData(null)
      setRevealedId(null)
    } else {
      revealMutation.mutate(id)
    }
  }

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[...Array(5)].map((_, i) => (
          <Skeleton key={i} className="h-12" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {credentials?.length ?? 0} credential{(credentials?.length ?? 0) !== 1 ? 's' : ''} stored
        </p>
        <Button size="sm" onClick={() => { setShowForm(!showForm); setEditingCredential(null) }} className="gap-2">
          {showForm ? <X className="h-4 w-4" /> : <Plus className="h-4 w-4" />}
          {showForm ? 'Cancel' : 'Add Credential'}
        </Button>
      </div>

      {/* Create Form */}
      {showForm && !editingCredential && (
        <CredentialForm
          onSubmit={(req) => createMutation.mutate(req as CreateCredentialRequest)}
          isPending={createMutation.isPending}
          onCancel={() => setShowForm(false)}
        />
      )}

      {/* Edit Form */}
      {editingCredential && (
        <CredentialForm
          initial={editingCredential}
          onSubmit={(req) =>
            updateMutation.mutate({ id: editingCredential.id, req: req as UpdateCredentialRequest })
          }
          isPending={updateMutation.isPending}
          onCancel={() => setEditingCredential(null)}
        />
      )}

      {/* Credentials Table */}
      {!credentials || credentials.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <Key className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No credentials stored</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Add credentials to securely manage access to your network devices.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="rounded-lg border overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">Name</th>
                  <th className="px-4 py-3 text-left font-medium">Type</th>
                  <th className="px-4 py-3 text-left font-medium">Device</th>
                  <th className="px-4 py-3 text-left font-medium">Description</th>
                  <th className="px-4 py-3 text-left font-medium">Created</th>
                  <th className="px-4 py-3 text-left font-medium w-36">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {credentials.map((cred) => (
                  <tr key={cred.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3 font-medium">{cred.name}</td>
                    <td className="px-4 py-3">
                      <CredentialTypeBadge type={cred.type} />
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {cred.device_id ? (
                        <span className="font-mono text-xs">{cred.device_id.slice(0, 8)}...</span>
                      ) : (
                        <span className="text-xs">Unassigned</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground max-w-xs truncate">
                      {cred.description || '-'}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground text-xs">
                      {new Date(cred.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7"
                          onClick={() => handleReveal(cred.id)}
                          disabled={revealMutation.isPending}
                          title={revealedId === cred.id ? 'Hide secret data' : 'Reveal secret data'}
                        >
                          {revealedId === cred.id ? (
                            <EyeOff className="h-3.5 w-3.5" />
                          ) : (
                            <Eye className="h-3.5 w-3.5" />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7"
                          onClick={() => { setEditingCredential(cred); setShowForm(false) }}
                          title="Edit credential"
                        >
                          <Edit2 className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 text-muted-foreground hover:text-red-500"
                          onClick={() => setDeletingId(cred.id)}
                          title="Delete credential"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Revealed data panel */}
          {revealedId && revealedData && (
            <div className="border-t bg-muted/20 px-4 py-3">
              <div className="flex items-center justify-between mb-2">
                <p className="text-xs font-medium text-muted-foreground">
                  Decrypted data for credential {revealedId.slice(0, 8)}...
                </p>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 gap-1 text-xs"
                  onClick={() => { setRevealedData(null); setRevealedId(null) }}
                >
                  <EyeOff className="h-3 w-3" />
                  Hide
                </Button>
              </div>
              <pre className="text-xs font-mono bg-background border rounded p-3 overflow-x-auto">
                {JSON.stringify(revealedData, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      {deletingId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="fixed inset-0 bg-black/60 backdrop-blur-sm"
            onClick={() => setDeletingId(null)}
          />
          <div className="relative z-50 w-full max-w-sm rounded-lg border bg-card p-6 shadow-lg">
            <h3 className="text-lg font-semibold mb-2 flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-red-500" />
              Delete Credential
            </h3>
            <p className="text-sm text-muted-foreground mb-4">
              Are you sure you want to permanently delete this credential?
              This action cannot be undone.
            </p>
            <div className="flex items-center justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => setDeletingId(null)}
                disabled={deleteMutation.isPending}
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={() => deleteMutation.mutate(deletingId)}
                disabled={deleteMutation.isPending}
              >
                {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

// ============================================================================
// Credential Type Badge
// ============================================================================

function CredentialTypeBadge({ type }: { type: string }) {
  const config: Record<string, { bg: string; text: string }> = {
    ssh_password: { bg: 'bg-blue-500/10', text: 'text-blue-600 dark:text-blue-400' },
    ssh_key: { bg: 'bg-indigo-500/10', text: 'text-indigo-600 dark:text-indigo-400' },
    snmp_v2c: { bg: 'bg-teal-500/10', text: 'text-teal-600 dark:text-teal-400' },
    snmp_v3: { bg: 'bg-emerald-500/10', text: 'text-emerald-600 dark:text-emerald-400' },
    api_key: { bg: 'bg-purple-500/10', text: 'text-purple-600 dark:text-purple-400' },
    http_basic: { bg: 'bg-amber-500/10', text: 'text-amber-600 dark:text-amber-400' },
    custom: { bg: 'bg-gray-500/10', text: 'text-gray-600 dark:text-gray-400' },
  }
  const c = config[type] ?? { bg: 'bg-muted', text: 'text-muted-foreground' }
  const label = CREDENTIAL_TYPES.find((t) => t.value === type)?.label ?? type
  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs font-medium', c.bg, c.text)}>
      {label}
    </span>
  )
}

// ============================================================================
// Credential Form
// ============================================================================

// Type-specific field definitions
const typeFields: Record<CredentialType, { key: string; label: string; type: 'text' | 'password' | 'textarea'; required?: boolean }[]> = {
  ssh_password: [
    { key: 'username', label: 'Username', type: 'text', required: true },
    { key: 'password', label: 'Password', type: 'password', required: true },
  ],
  ssh_key: [
    { key: 'username', label: 'Username', type: 'text', required: true },
    { key: 'private_key', label: 'Private Key', type: 'textarea', required: true },
    { key: 'passphrase', label: 'Passphrase', type: 'password' },
  ],
  snmp_v2c: [
    { key: 'community', label: 'Community String', type: 'password', required: true },
  ],
  snmp_v3: [
    { key: 'username', label: 'Username', type: 'text', required: true },
    { key: 'auth_protocol', label: 'Auth Protocol', type: 'text', required: true },
    { key: 'auth_passphrase', label: 'Auth Passphrase', type: 'password', required: true },
    { key: 'priv_protocol', label: 'Privacy Protocol', type: 'text' },
    { key: 'priv_passphrase', label: 'Privacy Passphrase', type: 'password' },
  ],
  api_key: [
    { key: 'key', label: 'API Key', type: 'password', required: true },
    { key: 'header_name', label: 'Header Name', type: 'text' },
  ],
  http_basic: [
    { key: 'username', label: 'Username', type: 'text', required: true },
    { key: 'password', label: 'Password', type: 'password', required: true },
  ],
  custom: [],
}

function CredentialForm({
  initial,
  onSubmit,
  isPending,
  onCancel,
}: {
  initial?: CredentialMeta
  onSubmit: (req: CreateCredentialRequest | UpdateCredentialRequest) => void
  isPending: boolean
  onCancel: () => void
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [credType, setCredType] = useState<CredentialType>((initial?.type as CredentialType) ?? 'ssh_password')
  const [deviceId, setDeviceId] = useState(initial?.device_id ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({})
  const [customPairs, setCustomPairs] = useState<{ key: string; value: string }[]>([{ key: '', value: '' }])
  const [showSecrets, setShowSecrets] = useState(false)

  function updateField(key: string, value: string) {
    setFieldValues((prev) => ({ ...prev, [key]: value }))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()

    let data: Record<string, unknown>
    if (credType === 'custom') {
      data = {}
      for (const pair of customPairs) {
        if (pair.key.trim()) {
          data[pair.key.trim()] = pair.value
        }
      }
    } else {
      data = { ...fieldValues }
      // Remove empty optional fields
      for (const [k, v] of Object.entries(data)) {
        if (v === '') delete data[k]
      }
    }

    if (initial) {
      // Update -- send only changed fields
      const req: UpdateCredentialRequest = {}
      if (name !== initial.name) req.name = name
      if (deviceId !== (initial.device_id ?? '')) req.device_id = deviceId || undefined
      if (description !== (initial.description ?? '')) req.description = description
      if (Object.keys(data).length > 0) req.data = data
      onSubmit(req)
    } else {
      onSubmit({
        name,
        type: credType,
        device_id: deviceId || undefined,
        description: description || undefined,
        data,
      })
    }
  }

  const fields = typeFields[credType] ?? []

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium">
          {initial ? 'Edit Credential' : 'Create Credential'}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Basic fields */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            <div className="space-y-1">
              <Label htmlFor="cred-name">Name</Label>
              <Input
                id="cred-name"
                placeholder="Credential name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="cred-type">
                Type
                <HelpIcon content="SNMP: community strings for device discovery. SSH: keys or passwords for remote access. API Key: tokens for service integration. HTTP Basic: username/password for web endpoints." />
              </Label>
              <select
                id="cred-type"
                value={credType}
                onChange={(e) => {
                  setCredType(e.target.value as CredentialType)
                  setFieldValues({})
                }}
                disabled={!!initial}
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:opacity-50"
              >
                {CREDENTIAL_TYPES.map((t) => (
                  <option key={t.value} value={t.value}>{t.label}</option>
                ))}
              </select>
            </div>
            <div className="space-y-1">
              <Label htmlFor="cred-device">Device ID (optional)</Label>
              <Input
                id="cred-device"
                placeholder="Assign to device"
                value={deviceId}
                onChange={(e) => setDeviceId(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="cred-desc">Description (optional)</Label>
              <Input
                id="cred-desc"
                placeholder="Brief description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
          </div>

          {/* Type-specific fields */}
          {fields.length > 0 && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                  Credential Data
                </p>
                <button
                  type="button"
                  onClick={() => setShowSecrets(!showSecrets)}
                  className="text-xs text-muted-foreground hover:text-foreground flex items-center gap-1"
                >
                  {showSecrets ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
                  {showSecrets ? 'Hide' : 'Show'} secrets
                </button>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {fields.map((field) => (
                  <div key={field.key} className={cn('space-y-1', field.type === 'textarea' && 'sm:col-span-2')}>
                    <Label htmlFor={`field-${field.key}`}>
                      {field.label}
                      {field.required && <span className="text-red-500 ml-0.5">*</span>}
                    </Label>
                    {field.type === 'textarea' ? (
                      <textarea
                        id={`field-${field.key}`}
                        value={fieldValues[field.key] ?? ''}
                        onChange={(e) => updateField(field.key, e.target.value)}
                        required={field.required}
                        placeholder={`Enter ${field.label.toLowerCase()}`}
                        className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring min-h-[100px] font-mono"
                        rows={4}
                      />
                    ) : (
                      <Input
                        id={`field-${field.key}`}
                        type={field.type === 'password' && !showSecrets ? 'password' : 'text'}
                        value={fieldValues[field.key] ?? ''}
                        onChange={(e) => updateField(field.key, e.target.value)}
                        required={field.required}
                        placeholder={`Enter ${field.label.toLowerCase()}`}
                      />
                    )}
                    {field.key === 'community' && (
                      <FieldHelp text="The read-only community string used for SNMP queries. Default is typically 'public'." />
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Custom key-value pairs */}
          {credType === 'custom' && (
            <div className="space-y-3">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                Custom Key-Value Pairs
              </p>
              {customPairs.map((pair, index) => (
                <div key={index} className="flex items-center gap-2">
                  <Input
                    placeholder="Key"
                    value={pair.key}
                    onChange={(e) => {
                      const updated = [...customPairs]
                      updated[index] = { ...pair, key: e.target.value }
                      setCustomPairs(updated)
                    }}
                    className="flex-1"
                  />
                  <Input
                    placeholder="Value"
                    type={showSecrets ? 'text' : 'password'}
                    value={pair.value}
                    onChange={(e) => {
                      const updated = [...customPairs]
                      updated[index] = { ...pair, value: e.target.value }
                      setCustomPairs(updated)
                    }}
                    className="flex-1"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-9 w-9 shrink-0"
                    onClick={() => {
                      if (customPairs.length > 1) {
                        setCustomPairs(customPairs.filter((_, i) => i !== index))
                      }
                    }}
                    disabled={customPairs.length <= 1}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setCustomPairs([...customPairs, { key: '', value: '' }])}
                className="gap-1"
              >
                <Plus className="h-3.5 w-3.5" />
                Add Pair
              </Button>
            </div>
          )}

          {/* Form actions */}
          <div className="flex items-center gap-2">
            <Button type="submit" size="sm" disabled={isPending} className="gap-1">
              <Check className="h-3.5 w-3.5" />
              {isPending ? 'Saving...' : initial ? 'Update' : 'Create'}
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={onCancel}>
              Cancel
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

// ============================================================================
// Audit Log Tab
// ============================================================================

function AuditTab() {
  const [limit, setLimit] = useState(50)
  const [filterCredentialId, setFilterCredentialId] = useState('')

  const { data: auditEntries, isLoading } = useQuery({
    queryKey: ['vault-audit', { limit, credential_id: filterCredentialId }],
    queryFn: () => getAuditLog({
      limit,
      credential_id: filterCredentialId || undefined,
    }),
  })

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[...Array(5)].map((_, i) => (
          <Skeleton key={i} className="h-12" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2">
          <Input
            placeholder="Filter by credential ID"
            value={filterCredentialId}
            onChange={(e) => setFilterCredentialId(e.target.value)}
            className="h-8 w-64 text-sm"
          />
        </div>
        <div className="flex items-center gap-2">
          <Label htmlFor="audit-limit" className="text-xs text-muted-foreground">
            Limit:
          </Label>
          <select
            id="audit-limit"
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="flex h-8 rounded-md border border-input bg-transparent px-2 py-1 text-xs shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          >
            <option value={25}>25</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
            <option value={250}>250</option>
          </select>
        </div>
        <p className="text-sm text-muted-foreground ml-auto">
          {auditEntries?.length ?? 0} entr{(auditEntries?.length ?? 0) !== 1 ? 'ies' : 'y'}
        </p>
      </div>

      {/* Audit Table */}
      {!auditEntries || auditEntries.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <FileText className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No audit entries</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Vault operations will be logged here for security auditing.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="rounded-lg border overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">Timestamp</th>
                  <th className="px-4 py-3 text-left font-medium">Action</th>
                  <th className="px-4 py-3 text-left font-medium">Credential</th>
                  <th className="px-4 py-3 text-left font-medium">User</th>
                  <th className="px-4 py-3 text-left font-medium">Purpose</th>
                  <th className="px-4 py-3 text-left font-medium">Source IP</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {auditEntries.map((entry) => (
                  <tr key={entry.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3 text-muted-foreground text-xs">
                      {new Date(entry.timestamp).toLocaleString()}
                    </td>
                    <td className="px-4 py-3">
                      <AuditActionBadge action={entry.action} />
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {entry.credential_id.slice(0, 8)}...
                    </td>
                    <td className="px-4 py-3 text-muted-foreground text-xs">
                      {entry.user_id.slice(0, 8)}...
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {entry.purpose || '-'}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                      {entry.source_ip || '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}

function AuditActionBadge({ action }: { action: string }) {
  const config: Record<string, { bg: string; text: string }> = {
    create: { bg: 'bg-green-500/10', text: 'text-green-600 dark:text-green-400' },
    read: { bg: 'bg-blue-500/10', text: 'text-blue-600 dark:text-blue-400' },
    update: { bg: 'bg-amber-500/10', text: 'text-amber-600 dark:text-amber-400' },
    delete: { bg: 'bg-red-500/10', text: 'text-red-600 dark:text-red-400' },
    decrypt: { bg: 'bg-purple-500/10', text: 'text-purple-600 dark:text-purple-400' },
  }
  const c = config[action] ?? { bg: 'bg-muted', text: 'text-muted-foreground' }
  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize', c.bg, c.text)}>
      {action}
    </span>
  )
}
