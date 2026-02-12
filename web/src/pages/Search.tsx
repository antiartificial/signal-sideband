import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { searchMessages, getGroups, type SearchFilters } from '../lib/api.ts'
import Card from '../components/Card.tsx'
import EmptyState from '../components/EmptyState.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import { format } from 'date-fns'

export default function Search() {
  const [query, setQuery] = useState('')
  const [mode, setMode] = useState<'fulltext' | 'semantic'>('fulltext')
  const [submitted, setSubmitted] = useState('')
  const [showFilters, setShowFilters] = useState(false)
  const [filters, setFilters] = useState<SearchFilters>({})

  const { data: groups } = useQuery({
    queryKey: ['groups'],
    queryFn: getGroups,
    enabled: showFilters,
  })

  const { data: results, isLoading } = useQuery({
    queryKey: ['search', submitted, mode, filters],
    queryFn: () => searchMessages(submitted, mode, 20, filters),
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
            className="flex-1 px-4 py-2.5 rounded-xl border border-apple-border bg-apple-card text-sm focus:outline-none focus:ring-2 focus:ring-apple-blue/30 focus:border-apple-blue transition-colors"
          />
          <button
            type="submit"
            className="px-5 py-2.5 bg-apple-blue text-white text-sm font-medium rounded-xl hover:bg-apple-blue-hover transition-colors"
          >
            Search
          </button>
        </div>

        <div className="flex items-center gap-3 mt-3">
          {/* Mode toggle */}
          <div className="flex gap-1 bg-gray-100 dark:bg-white/5 rounded-lg p-0.5">
            <button
              type="button"
              onClick={() => setMode('fulltext')}
              className={`px-3 py-1.5 text-xs rounded-md transition-colors ${
                mode === 'fulltext' ? 'bg-apple-card shadow-sm font-medium text-apple-blue' : 'text-apple-secondary'
              }`}
            >
              Full Text
            </button>
            <button
              type="button"
              onClick={() => setMode('semantic')}
              className={`px-3 py-1.5 text-xs rounded-md transition-colors ${
                mode === 'semantic' ? 'bg-apple-card shadow-sm font-medium text-apple-blue' : 'text-apple-secondary'
              }`}
            >
              Semantic
            </button>
          </div>

          {/* Filter toggle */}
          <button
            type="button"
            onClick={() => setShowFilters(!showFilters)}
            className={`px-3 py-1.5 text-xs rounded-md border transition-colors ${
              showFilters ? 'border-apple-blue text-apple-blue bg-apple-blue/5' : 'border-apple-border text-apple-secondary'
            }`}
          >
            Filters
          </button>
        </div>

        {/* Filter panel */}
        {showFilters && (
          <div className="mt-3 p-4 bg-gray-50 dark:bg-white/[0.03] rounded-xl border border-apple-border space-y-3">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-apple-secondary mb-1">Group</label>
                <select
                  value={filters.group_id || ''}
                  onChange={e => setFilters(f => ({ ...f, group_id: e.target.value || undefined }))}
                  className="w-full px-3 py-2 rounded-lg border border-apple-border bg-apple-card text-sm"
                >
                  <option value="">All groups</option>
                  {groups?.map(g => (
                    <option key={g.group_id} value={g.group_id}>{g.name || g.group_id}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs text-apple-secondary mb-1">Sender</label>
                <input
                  type="text"
                  value={filters.sender_id || ''}
                  onChange={e => setFilters(f => ({ ...f, sender_id: e.target.value || undefined }))}
                  placeholder="Phone or UUID"
                  className="w-full px-3 py-2 rounded-lg border border-apple-border bg-apple-card text-sm"
                />
              </div>
              <div>
                <label className="block text-xs text-apple-secondary mb-1">After</label>
                <input
                  type="date"
                  value={filters.after?.split('T')[0] || ''}
                  onChange={e => setFilters(f => ({ ...f, after: e.target.value ? e.target.value + 'T00:00:00Z' : undefined }))}
                  className="w-full px-3 py-2 rounded-lg border border-apple-border bg-apple-card text-sm"
                />
              </div>
              <div>
                <label className="block text-xs text-apple-secondary mb-1">Before</label>
                <input
                  type="date"
                  value={filters.before?.split('T')[0] || ''}
                  onChange={e => setFilters(f => ({ ...f, before: e.target.value ? e.target.value + 'T23:59:59Z' : undefined }))}
                  className="w-full px-3 py-2 rounded-lg border border-apple-border bg-apple-card text-sm"
                />
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={filters.has_media || false}
                onChange={e => setFilters(f => ({ ...f, has_media: e.target.checked || undefined }))}
                className="rounded border-apple-border"
              />
              Has media
            </label>
          </div>
        )}
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
