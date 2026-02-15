import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Sparkles, Search, Loader2, XCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { submitNLQuery } from '@/api/insight'
import type { NLQueryResponse } from '@/api/insight'

export function NLQueryBar() {
  const [query, setQuery] = useState('')
  const [lastResponse, setLastResponse] = useState<NLQueryResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: submitNLQuery,
    onSuccess: (data) => {
      setLastResponse(data)
      setQuery('')
      setError(null)
    },
    onError: (err: Error) => {
      if (err.message?.includes('503')) {
        setError('LLM not configured. Go to Settings to connect an AI provider.')
      } else {
        setError('Failed to process query. Please try again.')
      }
    },
  })

  const handleSubmit = () => {
    if (!query.trim() || mutation.isPending) return
    mutation.mutate({ query: query.trim() })
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-muted-foreground" />
          AI Insight
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Ask about your network..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={mutation.isPending}
              className="pl-9"
            />
          </div>
          <Button
            onClick={handleSubmit}
            disabled={!query.trim() || mutation.isPending}
            size="sm"
            className="gap-1.5"
          >
            {mutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Sparkles className="h-4 w-4" />
            )}
            Ask
          </Button>
        </div>

        {error && (
          <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-500/10 text-amber-600 dark:text-amber-400">
            <XCircle className="h-4 w-4 mt-0.5 shrink-0" />
            <div className="flex-1 text-sm">{error}</div>
            <button
              type="button"
              onClick={() => setError(null)}
              className="text-xs hover:underline shrink-0"
            >
              Dismiss
            </button>
          </div>
        )}

        {lastResponse && (
          <div className="p-3 rounded-lg bg-muted/50 space-y-2">
            <p className="text-sm">{lastResponse.answer}</p>
            {lastResponse.model && (
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
                {lastResponse.model}
              </span>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
