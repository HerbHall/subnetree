import { useState, useCallback, useMemo, useRef } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { RefreshCw, AlertCircle, Bot, Trash2, Search } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { listAgents, deleteAgent } from '@/api/agents'
import { useKeyboardShortcuts } from '@/hooks/use-keyboard-shortcuts'
import { cn } from '@/lib/utils'
import type { AgentStatus } from '@/api/types'

function formatRelativeTime(isoString: string): string {
  const now = Date.now()
  const then = new Date(isoString).getTime()
  const diffSeconds = Math.floor((now - then) / 1000)

  if (diffSeconds < 60) return 'just now'
  const diffMinutes = Math.floor(diffSeconds / 60)
  if (diffMinutes < 60) return `${diffMinutes}m ago`
  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 30) return `${diffDays}d ago`
  const diffMonths = Math.floor(diffDays / 30)
  return `${diffMonths}mo ago`
}

export function AgentsPage() {
  const queryClient = useQueryClient()
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const searchInputRef = useRef<HTMLInputElement>(null)

  const {
    data: agents,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['agents'],
    queryFn: listAgents,
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteAgent(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['agents'] })
      toast.success('Agent deleted')
      setDeletingId(null)
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Failed to delete agent')
      setDeletingId(null)
    },
  })

  const focusSearch = useCallback(() => searchInputRef.current?.focus(), [])
  const refreshData = useCallback(() => { refetch() }, [refetch])

  useKeyboardShortcuts(
    useMemo(
      () => [
        { key: '/', handler: focusSearch, description: 'Focus search' },
        { key: 'r', handler: refreshData, description: 'Refresh data' },
      ],
      [focusSearch, refreshData]
    )
  )

  function handleDeleteClick(e: React.MouseEvent, agentId: string) {
    e.preventDefault()
    e.stopPropagation()
    setDeletingId(agentId)
  }

  function confirmDelete() {
    if (deletingId) {
      deleteMutation.mutate(deletingId)
    }
  }

  function cancelDelete() {
    setDeletingId(null)
  }

  const filteredAgents = useMemo(() => {
    if (!agents) return []
    if (!searchQuery) return agents

    const query = searchQuery.toLowerCase()
    return agents.filter(
      (a) =>
        a.hostname?.toLowerCase().includes(query) ||
        a.platform?.toLowerCase().includes(query) ||
        a.agent_version?.toLowerCase().includes(query) ||
        a.device_id?.toLowerCase().includes(query)
    )
  }, [agents, searchQuery])

  const totalAgents = agents?.length ?? 0

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Agents</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {searchQuery ? `${filteredAgents.length} matching of ` : ''}
            {totalAgents} agent{totalAgents !== 1 ? 's' : ''}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon" onClick={() => refetch()} disabled={isLoading}>
            <RefreshCw className={cn('h-4 w-4', isLoading && 'animate-spin')} />
          </Button>
        </div>
      </div>

      {/* Search */}
      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          ref={searchInputRef}
          placeholder="Search hostname, platform, version..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* Error state */}
      {error && (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <AlertCircle className="h-12 w-12 text-red-400 mb-4" />
          <h3 className="text-lg font-medium">Failed to load agents</h3>
          <p className="text-sm text-muted-foreground mt-1 max-w-sm">
            {error instanceof Error ? error.message : 'An unexpected error occurred while fetching agent data.'}
          </p>
          <Button variant="outline" className="mt-4 gap-2" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
            Try Again
          </Button>
        </div>
      )}

      {/* Loading state */}
      {isLoading && !error && (
        <div className="rounded-lg border overflow-hidden">
          <div className="bg-muted/50 px-4 py-3">
            <div className="flex items-center gap-4">
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-4 w-12" />
            </div>
          </div>
          <div className="divide-y">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="px-4 py-3">
                <div className="flex items-center gap-4">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-5 w-20 rounded-full" />
                  <Skeleton className="h-4 w-16" />
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-4 w-8" />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Empty state */}
      {!isLoading && !error && filteredAgents.length === 0 && (
        <div className="text-center py-12">
          <Bot className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
          {totalAgents === 0 ? (
            <>
              <h3 className="text-lg font-medium">No agents enrolled</h3>
              <p className="text-sm text-muted-foreground mt-1 max-w-md mx-auto">
                Install the Scout agent on devices you want to monitor. Create an enrollment
                token and use it to register agents with the server.
              </p>
            </>
          ) : (
            <>
              <h3 className="text-lg font-medium">No matching agents</h3>
              <p className="text-sm text-muted-foreground mt-1">
                Try adjusting your search query.
              </p>
            </>
          )}
        </div>
      )}

      {/* Agent table */}
      {!isLoading && !error && filteredAgents.length > 0 && (
        <div className="rounded-lg border overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">Hostname</th>
                  <th className="px-4 py-3 text-left font-medium">Platform</th>
                  <th className="px-4 py-3 text-left font-medium">Status</th>
                  <th className="px-4 py-3 text-left font-medium">Version</th>
                  <th className="px-4 py-3 text-left font-medium">Last Check-in</th>
                  <th className="px-4 py-3 text-left font-medium">Enrolled</th>
                  <th className="px-4 py-3 text-left font-medium w-12">
                    <span className="sr-only">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {filteredAgents.map((agent) => (
                  <tr
                    key={agent.id}
                    className="hover:bg-muted/30 transition-colors"
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/agents/${agent.id}`}
                        className="font-medium hover:text-primary"
                      >
                        {agent.hostname || 'Unknown'}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {agent.platform || '-'}
                    </td>
                    <td className="px-4 py-3">
                      <AgentStatusBadge status={agent.status} />
                    </td>
                    <td className="px-4 py-3 font-mono text-xs">
                      {agent.agent_version || '-'}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {agent.last_check_in
                        ? formatRelativeTime(agent.last_check_in)
                        : 'Never'}
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {formatRelativeTime(agent.enrolled_at)}
                    </td>
                    <td className="px-4 py-3">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-muted-foreground hover:text-red-500"
                        onClick={(e) => handleDeleteClick(e, agent.id)}
                        title="Delete agent"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmDialog
        open={!!deletingId}
        isPending={deleteMutation.isPending}
        onConfirm={confirmDelete}
        onCancel={cancelDelete}
      />
    </div>
  )
}

function AgentStatusBadge({ status }: { status: AgentStatus }) {
  const config = {
    connected: { bg: 'bg-green-500/10', text: 'text-green-500', dot: 'bg-green-500' },
    pending: { bg: 'bg-amber-500/10', text: 'text-amber-500', dot: 'bg-amber-500' },
    disconnected: { bg: 'bg-red-500/10', text: 'text-red-500', dot: 'bg-red-500' },
  }
  const c = config[status] ?? config.disconnected

  return (
    <span className={cn('inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs', c.bg, c.text)}>
      <span className={cn('h-1.5 w-1.5 rounded-full', c.dot)} />
      <span className="capitalize">{status}</span>
    </span>
  )
}

function DeleteConfirmDialog({
  open,
  isPending,
  onConfirm,
  onCancel,
}: {
  open: boolean
  isPending: boolean
  onConfirm: () => void
  onCancel: () => void
}) {
  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/60 backdrop-blur-sm" onClick={onCancel} />
      <div className="relative z-50 w-full max-w-sm rounded-lg border bg-card p-6 shadow-lg">
        <h3 className="text-lg font-semibold mb-2">Delete Agent</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Are you sure you want to delete this agent? The agent will need to
          re-enroll to reconnect.
        </p>
        <div className="flex items-center justify-end gap-2">
          <Button variant="outline" onClick={onCancel} disabled={isPending}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={isPending}
          >
            {isPending ? 'Deleting...' : 'Delete'}
          </Button>
        </div>
      </div>
    </div>
  )
}
