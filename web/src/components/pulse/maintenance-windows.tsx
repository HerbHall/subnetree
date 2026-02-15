import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Pencil,
  Trash2,
  X,
  Calendar,
  Clock,
  Power,
} from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'
import {
  listMaintWindows,
  createMaintWindow,
  updateMaintWindow,
  deleteMaintWindow,
} from '@/api/pulse-maintenance'
import type { MaintWindow, CreateMaintWindowRequest, UpdateMaintWindowRequest } from '@/api/pulse-maintenance'

// ---------------------------------------------------------------------------
// Status badges
// ---------------------------------------------------------------------------

function StatusBadge({ window }: { window: MaintWindow }) {
  const now = new Date()
  const start = new Date(window.start_time)
  const end = new Date(window.end_time)

  if (!window.enabled) {
    return <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">Disabled</span>
  }
  if (window.recurrence !== 'once') {
    return <span className="inline-flex items-center rounded-full bg-blue-100 dark:bg-blue-900/30 px-2 py-0.5 text-xs text-blue-700 dark:text-blue-300">Recurring</span>
  }
  if (now >= start && now <= end) {
    return <span className="inline-flex items-center rounded-full bg-green-100 dark:bg-green-900/30 px-2 py-0.5 text-xs text-green-700 dark:text-green-300">Active</span>
  }
  if (now < start) {
    return <span className="inline-flex items-center rounded-full bg-yellow-100 dark:bg-yellow-900/30 px-2 py-0.5 text-xs text-yellow-700 dark:text-yellow-300">Upcoming</span>
  }
  return <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">Expired</span>
}

function RecurrenceBadge({ recurrence }: { recurrence: string }) {
  const labels: Record<string, string> = {
    once: 'Once',
    daily: 'Daily',
    weekly: 'Weekly',
    monthly: 'Monthly',
  }
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
      <Clock className="h-3 w-3" />
      {labels[recurrence] ?? recurrence}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Form
// ---------------------------------------------------------------------------

interface MaintWindowFormProps {
  initial?: MaintWindow
  onSubmit: (data: CreateMaintWindowRequest | UpdateMaintWindowRequest) => void
  onCancel: () => void
  isLoading: boolean
}

function MaintWindowForm({ initial, onSubmit, onCancel, isLoading }: MaintWindowFormProps) {
  const [name, setName] = useState(initial?.name ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [startTime, setStartTime] = useState(initial?.start_time ? initial.start_time.slice(0, 16) : '')
  const [endTime, setEndTime] = useState(initial?.end_time ? initial.end_time.slice(0, 16) : '')
  const [recurrence, setRecurrence] = useState<string>(initial?.recurrence ?? 'once')
  const [deviceIds, setDeviceIds] = useState(initial?.device_ids?.join(', ') ?? '')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const ids = deviceIds.split(',').map((s) => s.trim()).filter(Boolean)
    const data: CreateMaintWindowRequest = {
      name,
      description,
      start_time: new Date(startTime).toISOString(),
      end_time: new Date(endTime).toISOString(),
      recurrence,
      device_ids: ids,
    }
    onSubmit(data)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-1">Name</label>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Weekly patch window" required />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Recurrence</label>
          <select
            value={recurrence}
            onChange={(e) => setRecurrence(e.target.value)}
            className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          >
            <option value="once">Once</option>
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Start Time</label>
          <Input type="datetime-local" value={startTime} onChange={(e) => setStartTime(e.target.value)} required />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">End Time</label>
          <Input type="datetime-local" value={endTime} onChange={(e) => setEndTime(e.target.value)} required />
        </div>
      </div>
      <div>
        <label className="block text-sm font-medium mb-1">Description</label>
        <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Optional description" />
      </div>
      <div>
        <label className="block text-sm font-medium mb-1">Device IDs (comma-separated)</label>
        <Input value={deviceIds} onChange={(e) => setDeviceIds(e.target.value)} placeholder="dev-1, dev-2, dev-3" required />
      </div>
      <div className="flex gap-2 justify-end">
        <Button type="button" variant="ghost" onClick={onCancel} disabled={isLoading}>
          Cancel
        </Button>
        <Button type="submit" disabled={isLoading}>
          {initial ? 'Update' : 'Create'}
        </Button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Main panel
// ---------------------------------------------------------------------------

export function MaintenanceWindowsPanel() {
  const queryClient = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<MaintWindow | null>(null)

  const { data: windows = [], isLoading } = useQuery({
    queryKey: ['pulse', 'maintenance-windows'],
    queryFn: listMaintWindows,
  })

  const createMut = useMutation({
    mutationFn: (req: CreateMaintWindowRequest) => createMaintWindow(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pulse', 'maintenance-windows'] })
      setShowForm(false)
      toast.success('Maintenance window created')
    },
    onError: () => toast.error('Failed to create maintenance window'),
  })

  const updateMut = useMutation({
    mutationFn: ({ id, req }: { id: string; req: UpdateMaintWindowRequest }) => updateMaintWindow(id, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pulse', 'maintenance-windows'] })
      setEditing(null)
      toast.success('Maintenance window updated')
    },
    onError: () => toast.error('Failed to update maintenance window'),
  })

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteMaintWindow(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pulse', 'maintenance-windows'] })
      toast.success('Maintenance window deleted')
    },
    onError: () => toast.error('Failed to delete maintenance window'),
  })

  const toggleMut = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      updateMaintWindow(id, { enabled }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pulse', 'maintenance-windows'] })
    },
    onError: () => toast.error('Failed to toggle maintenance window'),
  })

  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="animate-pulse space-y-3">
            <div className="h-4 bg-muted rounded w-1/3" />
            <div className="h-4 bg-muted rounded w-1/2" />
            <div className="h-4 bg-muted rounded w-2/5" />
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Maintenance Windows</h3>
        {!showForm && !editing && (
          <Button size="sm" onClick={() => setShowForm(true)}>
            <Plus className="h-4 w-4 mr-1" />
            Add Window
          </Button>
        )}
      </div>

      {showForm && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">New Maintenance Window</CardTitle>
          </CardHeader>
          <CardContent>
            <MaintWindowForm
              onSubmit={(data) => createMut.mutate(data as CreateMaintWindowRequest)}
              onCancel={() => setShowForm(false)}
              isLoading={createMut.isPending}
            />
          </CardContent>
        </Card>
      )}

      {editing && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center justify-between">
              Edit Maintenance Window
              <Button size="icon" variant="ghost" onClick={() => setEditing(null)}>
                <X className="h-4 w-4" />
              </Button>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <MaintWindowForm
              initial={editing}
              onSubmit={(data) => updateMut.mutate({ id: editing.id, req: data as UpdateMaintWindowRequest })}
              onCancel={() => setEditing(null)}
              isLoading={updateMut.isPending}
            />
          </CardContent>
        </Card>
      )}

      {windows.length === 0 && !showForm ? (
        <Card>
          <CardContent className="p-6 text-center text-muted-foreground">
            <Calendar className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>No maintenance windows configured.</p>
            <p className="text-xs mt-1">
              Create maintenance windows to suppress alerts during scheduled downtime.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {windows.map((mw) => (
            <Card key={mw.id} className={cn(!mw.enabled && 'opacity-60')}>
              <CardContent className="p-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-medium">{mw.name}</span>
                      <StatusBadge window={mw} />
                      <RecurrenceBadge recurrence={mw.recurrence} />
                    </div>
                    {mw.description && (
                      <p className="text-sm text-muted-foreground mt-1">{mw.description}</p>
                    )}
                    <div className="text-xs text-muted-foreground mt-2 flex flex-wrap gap-x-4 gap-y-1">
                      <span>
                        Start: {new Date(mw.start_time).toLocaleString()}
                      </span>
                      <span>
                        End: {new Date(mw.end_time).toLocaleString()}
                      </span>
                      <span>
                        Devices: {mw.device_ids.length}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    <Button
                      size="icon"
                      variant="ghost"
                      title={mw.enabled ? 'Disable' : 'Enable'}
                      onClick={() => toggleMut.mutate({ id: mw.id, enabled: !mw.enabled })}
                    >
                      <Power className={cn('h-4 w-4', mw.enabled ? 'text-green-600' : 'text-muted-foreground')} />
                    </Button>
                    <Button size="icon" variant="ghost" onClick={() => { setEditing(mw); setShowForm(false) }}>
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="text-destructive hover:text-destructive"
                      onClick={() => {
                        if (confirm(`Delete maintenance window "${mw.name}"?`)) {
                          deleteMut.mutate(mw.id)
                        }
                      }}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
