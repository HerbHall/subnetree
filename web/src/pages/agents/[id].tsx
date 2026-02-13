import { useParams, Link } from 'react-router-dom'
import { ChevronLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'

export function AgentDetailPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/agents">
            <ChevronLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-semibold">Agent Detail</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Agent ID: <span className="font-mono">{id}</span>
          </p>
        </div>
      </div>

      <div className="rounded-lg border bg-card p-8 text-center">
        <p className="text-muted-foreground">
          Agent detail view coming in Phase 02-02.
        </p>
      </div>
    </div>
  )
}
