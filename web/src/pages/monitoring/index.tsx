import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Activity,
  HeartPulse,
  Bell,
  AlertTriangle,
  Plus,
  Trash2,
  Pencil,
  X,
  Check as CheckIcon,
  Send,
  Power,
  Eye,
  Filter,
} from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import {
  listChecks,
  createCheck,
  updateCheck,
  deleteCheck,
  toggleCheck,
  listAlerts,
  acknowledgeAlert,
  resolveAlert,
  listChannels,
  createChannel,
  deleteChannel,
  testChannel,
} from '@/api/pulse'
import type {
  Check,
  Alert,
  CheckType,
  CreateCheckRequest,
  CreateNotificationRequest,
} from '@/api/types'

type TabId = 'checks' | 'alerts' | 'notifications'

const tabs: { id: TabId; label: string; icon: React.ElementType }[] = [
  { id: 'checks', label: 'Checks', icon: HeartPulse },
  { id: 'alerts', label: 'Alerts', icon: AlertTriangle },
  { id: 'notifications', label: 'Notifications', icon: Bell },
]

export function MonitoringPage() {
  const [activeTab, setActiveTab] = useState<TabId>('checks')

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold flex items-center gap-2">
          <Activity className="h-6 w-6" />
          Monitoring
        </h1>
        <p className="text-sm text-muted-foreground mt-1">
          Health checks, alerts, and notification channels
        </p>
      </div>

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
      {activeTab === 'checks' && <ChecksTab />}
      {activeTab === 'alerts' && <AlertsTab />}
      {activeTab === 'notifications' && <NotificationsTab />}
    </div>
  )
}

// ============================================================================
// Checks Tab
// ============================================================================

function ChecksTab() {
  const queryClient = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editingCheck, setEditingCheck] = useState<Check | null>(null)

  const { data: checks, isLoading } = useQuery({
    queryKey: ['checks'],
    queryFn: listChecks,
  })

  const createMutation = useMutation({
    mutationFn: (req: CreateCheckRequest) => createCheck(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['checks'] })
      toast.success('Check created')
      setShowForm(false)
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to create check'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, req }: { id: string; req: Partial<CreateCheckRequest> }) =>
      updateCheck(id, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['checks'] })
      toast.success('Check updated')
      setEditingCheck(null)
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to update check'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteCheck(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['checks'] })
      toast.success('Check deleted')
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to delete check'),
  })

  const toggleMutation = useMutation({
    mutationFn: (id: string) => toggleCheck(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['checks'] })
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to toggle check'),
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
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {checks?.length ?? 0} check{(checks?.length ?? 0) !== 1 ? 's' : ''} configured
        </p>
        <Button size="sm" onClick={() => setShowForm(!showForm)} className="gap-2">
          {showForm ? <X className="h-4 w-4" /> : <Plus className="h-4 w-4" />}
          {showForm ? 'Cancel' : 'Create Check'}
        </Button>
      </div>

      {/* Create Check Form */}
      {showForm && (
        <CheckForm
          onSubmit={(req) => createMutation.mutate(req)}
          isPending={createMutation.isPending}
          onCancel={() => setShowForm(false)}
        />
      )}

      {/* Edit Check Form */}
      {editingCheck && (
        <CheckForm
          initial={editingCheck}
          onSubmit={(req) => updateMutation.mutate({ id: editingCheck.id, req })}
          isPending={updateMutation.isPending}
          onCancel={() => setEditingCheck(null)}
        />
      )}

      {/* Checks Table */}
      {!checks || checks.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <HeartPulse className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No checks configured</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Create a health check to monitor device availability.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="rounded-lg border overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">Device</th>
                  <th className="px-4 py-3 text-left font-medium">Type</th>
                  <th className="px-4 py-3 text-left font-medium">Target</th>
                  <th className="px-4 py-3 text-left font-medium">Interval</th>
                  <th className="px-4 py-3 text-left font-medium">Enabled</th>
                  <th className="px-4 py-3 text-left font-medium w-24">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {checks.map((check) => (
                  <tr key={check.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3 font-mono text-xs">
                      {check.device_id.slice(0, 8)}...
                    </td>
                    <td className="px-4 py-3">
                      <CheckTypeBadge type={check.check_type} />
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{check.target}</td>
                    <td className="px-4 py-3 text-muted-foreground">{check.interval_seconds}s</td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => toggleMutation.mutate(check.id)}
                        disabled={toggleMutation.isPending}
                        className={cn(
                          'inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs transition-colors',
                          check.enabled
                            ? 'bg-green-500/10 text-green-600 dark:text-green-400 hover:bg-green-500/20'
                            : 'bg-muted text-muted-foreground hover:bg-muted/80'
                        )}
                      >
                        <Power className="h-3 w-3" />
                        {check.enabled ? 'On' : 'Off'}
                      </button>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7"
                          onClick={() => setEditingCheck(check)}
                          title="Edit check"
                        >
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 text-muted-foreground hover:text-red-500"
                          onClick={() => deleteMutation.mutate(check.id)}
                          disabled={deleteMutation.isPending}
                          title="Delete check"
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
        </div>
      )}
    </div>
  )
}

function CheckTypeBadge({ type }: { type: CheckType }) {
  const config: Record<CheckType, { bg: string; text: string }> = {
    icmp: { bg: 'bg-blue-500/10', text: 'text-blue-600 dark:text-blue-400' },
    tcp: { bg: 'bg-purple-500/10', text: 'text-purple-600 dark:text-purple-400' },
    http: { bg: 'bg-emerald-500/10', text: 'text-emerald-600 dark:text-emerald-400' },
  }
  const c = config[type] ?? config.icmp
  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs font-medium uppercase', c.bg, c.text)}>
      {type}
    </span>
  )
}

function CheckForm({
  initial,
  onSubmit,
  isPending,
  onCancel,
}: {
  initial?: Check
  onSubmit: (req: CreateCheckRequest) => void
  isPending: boolean
  onCancel: () => void
}) {
  const [deviceId, setDeviceId] = useState(initial?.device_id ?? '')
  const [checkType, setCheckType] = useState<CheckType>(initial?.check_type ?? 'icmp')
  const [target, setTarget] = useState(initial?.target ?? '')
  const [interval, setInterval] = useState(String(initial?.interval_seconds ?? 60))

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    onSubmit({
      device_id: deviceId,
      check_type: checkType,
      target,
      interval_seconds: Number(interval),
    })
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium">
          {initial ? 'Edit Check' : 'Create Check'}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
          <Input
            placeholder="Device ID"
            value={deviceId}
            onChange={(e) => setDeviceId(e.target.value)}
            required
            disabled={!!initial}
          />
          <select
            value={checkType}
            onChange={(e) => setCheckType(e.target.value as CheckType)}
            className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          >
            <option value="icmp">ICMP</option>
            <option value="tcp">TCP</option>
            <option value="http">HTTP</option>
          </select>
          <Input
            placeholder="Target (IP or URL)"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            required
          />
          <Input
            type="number"
            placeholder="Interval (s)"
            value={interval}
            onChange={(e) => setInterval(e.target.value)}
            min={5}
          />
          <div className="flex items-center gap-2">
            <Button type="submit" size="sm" disabled={isPending} className="gap-1">
              <CheckIcon className="h-3.5 w-3.5" />
              {isPending ? 'Saving...' : 'Save'}
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
// Alerts Tab
// ============================================================================

function AlertsTab() {
  const queryClient = useQueryClient()
  const [severityFilter, setSeverityFilter] = useState<string>('')
  const [activeOnly, setActiveOnly] = useState(false)

  const { data: alerts, isLoading } = useQuery({
    queryKey: ['alerts', { severity: severityFilter, active: activeOnly }],
    queryFn: () =>
      listAlerts({
        severity: severityFilter || undefined,
        active: activeOnly || undefined,
      }),
  })

  const acknowledgeMutation = useMutation({
    mutationFn: (id: string) => acknowledgeAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
      toast.success('Alert acknowledged')
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to acknowledge alert'),
  })

  const resolveMutation = useMutation({
    mutationFn: (id: string) => resolveAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
      toast.success('Alert resolved')
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to resolve alert'),
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
          <Filter className="h-4 w-4 text-muted-foreground" />
          <select
            value={severityFilter}
            onChange={(e) => setSeverityFilter(e.target.value)}
            className="flex h-8 rounded-md border border-input bg-transparent px-2 py-1 text-xs shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          >
            <option value="">All Severities</option>
            <option value="critical">Critical</option>
            <option value="warning">Warning</option>
            <option value="info">Info</option>
          </select>
        </div>
        <button
          onClick={() => setActiveOnly(!activeOnly)}
          className={cn(
            'inline-flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-medium transition-colors border',
            activeOnly
              ? 'bg-primary text-primary-foreground border-primary'
              : 'bg-transparent text-muted-foreground border-input hover:bg-muted'
          )}
        >
          <Eye className="h-3.5 w-3.5" />
          Active Only
        </button>
        <p className="text-sm text-muted-foreground ml-auto">
          {alerts?.length ?? 0} alert{(alerts?.length ?? 0) !== 1 ? 's' : ''}
        </p>
      </div>

      {/* Alerts Table */}
      {!alerts || alerts.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <AlertTriangle className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No alerts</h3>
            <p className="text-sm text-muted-foreground mt-1">
              {activeOnly ? 'No active alerts at this time.' : 'No alerts have been triggered.'}
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="rounded-lg border overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">Device</th>
                  <th className="px-4 py-3 text-left font-medium">Severity</th>
                  <th className="px-4 py-3 text-left font-medium">Message</th>
                  <th className="px-4 py-3 text-left font-medium">Triggered At</th>
                  <th className="px-4 py-3 text-left font-medium">Status</th>
                  <th className="px-4 py-3 text-left font-medium w-32">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {alerts.map((alert) => (
                  <tr key={alert.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3 font-mono text-xs">
                      {alert.device_id.slice(0, 8)}...
                    </td>
                    <td className="px-4 py-3">
                      <SeverityBadge severity={alert.severity} />
                    </td>
                    <td className="px-4 py-3 max-w-xs truncate" title={alert.message}>
                      {alert.message}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground text-xs">
                      {new Date(alert.triggered_at).toLocaleString()}
                    </td>
                    <td className="px-4 py-3">
                      <AlertStatusBadge alert={alert} />
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        {!alert.acknowledged_at && !alert.resolved_at && (
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-7 text-xs gap-1"
                            onClick={() => acknowledgeMutation.mutate(alert.id)}
                            disabled={acknowledgeMutation.isPending}
                          >
                            <Eye className="h-3 w-3" />
                            Ack
                          </Button>
                        )}
                        {!alert.resolved_at && (
                          <Button
                            variant="outline"
                            size="sm"
                            className="h-7 text-xs gap-1"
                            onClick={() => resolveMutation.mutate(alert.id)}
                            disabled={resolveMutation.isPending}
                          >
                            <CheckIcon className="h-3 w-3" />
                            Resolve
                          </Button>
                        )}
                      </div>
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

function SeverityBadge({ severity }: { severity: string }) {
  const config: Record<string, { bg: string; text: string }> = {
    critical: { bg: 'bg-red-500/10', text: 'text-red-600 dark:text-red-400' },
    warning: { bg: 'bg-amber-500/10', text: 'text-amber-600 dark:text-amber-400' },
    info: { bg: 'bg-blue-500/10', text: 'text-blue-600 dark:text-blue-400' },
  }
  const c = config[severity] ?? config.info
  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize', c.bg, c.text)}>
      {severity}
    </span>
  )
}

function AlertStatusBadge({ alert }: { alert: Alert }) {
  if (alert.resolved_at) {
    return (
      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-500/10 text-green-600 dark:text-green-400">
        Resolved
      </span>
    )
  }
  if (alert.acknowledged_at) {
    return (
      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-500/10 text-blue-600 dark:text-blue-400">
        Acknowledged
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-red-500/10 text-red-600 dark:text-red-400">
      <span className="h-1.5 w-1.5 rounded-full bg-red-500 animate-pulse" />
      Active
    </span>
  )
}

// ============================================================================
// Notifications Tab
// ============================================================================

function NotificationsTab() {
  const queryClient = useQueryClient()
  const [showForm, setShowForm] = useState(false)

  const { data: channels, isLoading } = useQuery({
    queryKey: ['notification-channels'],
    queryFn: listChannels,
  })

  const createMutation = useMutation({
    mutationFn: (req: CreateNotificationRequest) => createChannel(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notification-channels'] })
      toast.success('Channel created')
      setShowForm(false)
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to create channel'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteChannel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notification-channels'] })
      toast.success('Channel deleted')
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to delete channel'),
  })

  const testMutation = useMutation({
    mutationFn: (id: string) => testChannel(id),
    onSuccess: () => toast.success('Test notification sent'),
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Failed to send test'),
  })

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[...Array(3)].map((_, i) => (
          <Skeleton key={i} className="h-12" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {channels?.length ?? 0} channel{(channels?.length ?? 0) !== 1 ? 's' : ''} configured
        </p>
        <Button size="sm" onClick={() => setShowForm(!showForm)} className="gap-2">
          {showForm ? <X className="h-4 w-4" /> : <Plus className="h-4 w-4" />}
          {showForm ? 'Cancel' : 'Create Channel'}
        </Button>
      </div>

      {/* Create Channel Form */}
      {showForm && (
        <ChannelForm
          onSubmit={(req) => createMutation.mutate(req)}
          isPending={createMutation.isPending}
          onCancel={() => setShowForm(false)}
        />
      )}

      {/* Channels Table */}
      {!channels || channels.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <Bell className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No notification channels</h3>
            <p className="text-sm text-muted-foreground mt-1">
              Create a channel to receive alerts via webhook or email.
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
                  <th className="px-4 py-3 text-left font-medium">Enabled</th>
                  <th className="px-4 py-3 text-left font-medium">Created At</th>
                  <th className="px-4 py-3 text-left font-medium w-40">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {channels.map((channel) => (
                  <tr key={channel.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3 font-medium">{channel.name}</td>
                    <td className="px-4 py-3">
                      <ChannelTypeBadge type={channel.type} />
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={cn(
                          'inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs',
                          channel.enabled
                            ? 'bg-green-500/10 text-green-600 dark:text-green-400'
                            : 'bg-muted text-muted-foreground'
                        )}
                      >
                        <Power className="h-3 w-3" />
                        {channel.enabled ? 'On' : 'Off'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground text-xs">
                      {new Date(channel.created_at).toLocaleString()}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        <Button
                          variant="outline"
                          size="sm"
                          className="h-7 text-xs gap-1"
                          onClick={() => testMutation.mutate(channel.id)}
                          disabled={testMutation.isPending}
                          title="Send test notification"
                        >
                          <Send className="h-3 w-3" />
                          Test
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 text-muted-foreground hover:text-red-500"
                          onClick={() => deleteMutation.mutate(channel.id)}
                          disabled={deleteMutation.isPending}
                          title="Delete channel"
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
        </div>
      )}
    </div>
  )
}

function ChannelTypeBadge({ type }: { type: string }) {
  const config: Record<string, { bg: string; text: string }> = {
    webhook: { bg: 'bg-indigo-500/10', text: 'text-indigo-600 dark:text-indigo-400' },
    email: { bg: 'bg-teal-500/10', text: 'text-teal-600 dark:text-teal-400' },
  }
  const c = config[type] ?? { bg: 'bg-muted', text: 'text-muted-foreground' }
  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize', c.bg, c.text)}>
      {type}
    </span>
  )
}

function ChannelForm({
  onSubmit,
  isPending,
  onCancel,
}: {
  onSubmit: (req: CreateNotificationRequest) => void
  isPending: boolean
  onCancel: () => void
}) {
  const [name, setName] = useState('')
  const [type, setType] = useState('webhook')
  const [config, setConfig] = useState('{}')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    onSubmit({ name, type, config })
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium">Create Channel</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <Input
              placeholder="Channel name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
            <select
              value={type}
              onChange={(e) => setType(e.target.value)}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            >
              <option value="webhook">Webhook</option>
              <option value="email">Email</option>
            </select>
            <div className="flex items-center gap-2">
              <Button type="submit" size="sm" disabled={isPending} className="gap-1">
                <CheckIcon className="h-3.5 w-3.5" />
                {isPending ? 'Creating...' : 'Create'}
              </Button>
              <Button type="button" variant="outline" size="sm" onClick={onCancel}>
                Cancel
              </Button>
            </div>
          </div>
          <textarea
            value={config}
            onChange={(e) => setConfig(e.target.value)}
            placeholder='{"url": "https://hooks.example.com/..."}'
            className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring min-h-[80px] font-mono"
            rows={3}
          />
        </form>
      </CardContent>
    </Card>
  )
}
