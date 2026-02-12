import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { searchMessages } from '../lib/api.ts'
import Card from '../components/Card.tsx'
import EmptyState from '../components/EmptyState.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import { format } from 'date-fns'

export default function Search() {
  const [query, setQuery] = useState('')
  const [mode, setMode] = useState<'fulltext' | 'semantic'>('fulltext')
  const [submitted, setSubmitted] = useState('')

  const { data: results, isLoading } = useQuery({
    queryKey: ['search', submitted, mode],
    queryFn: () => searchMessages(submitted, mode),
    enabled: submitted.length > 0,
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (query.trim()) setSubmitted(query.trim())
  }

  return (
    <div>
      <h2 className="text-2xl font-semibold tracking-tight mb-6">Search Messages</h2>

      <form onSubmit={handleSubmit} className="mb-6">
        <div className="flex gap-3">
          <input
            type="text"
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search messages..."
            className="flex-1 px-4 py-2.5 rounded-xl border border-apple-border bg-white text-sm focus:outline-none focus:ring-2 focus:ring-apple-blue/30 focus:border-apple-blue transition-colors"
          />
          <button
            type="submit"
            className="px-5 py-2.5 bg-apple-blue text-white text-sm font-medium rounded-xl hover:bg-apple-blue-hover transition-colors"
          >
            Search
          </button>
        </div>

        {/* Mode toggle */}
        <div className="flex gap-1 mt-3 bg-gray-100 rounded-lg p-0.5 w-fit">
          <button
            type="button"
            onClick={() => setMode('fulltext')}
            className={`px-3 py-1.5 text-xs rounded-md transition-colors ${
              mode === 'fulltext' ? 'bg-white shadow-sm font-medium' : 'text-apple-secondary'
            }`}
          >
            Full Text
          </button>
          <button
            type="button"
            onClick={() => setMode('semantic')}
            className={`px-3 py-1.5 text-xs rounded-md transition-colors ${
              mode === 'semantic' ? 'bg-white shadow-sm font-medium' : 'text-apple-secondary'
            }`}
          >
            Semantic
          </button>
        </div>
      </form>

      {isLoading && <LoadingSpinner message="Searching..." />}

      {results && results.length === 0 && (
        <EmptyState title="No results" description="Try a different search query or mode." />
      )}

      {results && results.length > 0 && (
        <div className="space-y-2">
          {results.map(r => (
            <Card key={r.id} className="px-5 py-4">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-sm font-medium">
                  {r.is_outgoing ? 'You' : r.sender_id}
                </span>
                <div className="flex items-center gap-2">
                  {r.similarity != null && (
                    <span className="text-xs text-apple-blue">
                      {(r.similarity * 100).toFixed(0)}% match
                    </span>
                  )}
                  <span className="text-xs text-apple-secondary">
                    {format(new Date(r.created_at), 'MMM d, h:mm a')}
                  </span>
                </div>
              </div>
              <p className="text-sm text-apple-text">{r.content}</p>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
