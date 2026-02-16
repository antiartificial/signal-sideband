import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getURLs } from '../lib/api.ts'
import { useContacts } from '../lib/useContacts.ts'
import Card from '../components/Card.tsx'
import Badge from '../components/Badge.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import EmptyState from '../components/EmptyState.tsx'
import Pagination from '../components/Pagination.tsx'
import { format, parseISO } from 'date-fns'
import type { URLRecord } from '../lib/types.ts'

type SortMode = 'date' | 'alpha'

export default function URLCollection() {
  const [offset, setOffset] = useState(0)
  const [search, setSearch] = useState('')
  const [sender, setSender] = useState('')
  const [tag, setTag] = useState('')
  const [sort, setSort] = useState<SortMode>('date')
  const [groupByDay, setGroupByDay] = useState(false)
  const { contacts } = useContacts()
  const limit = 40

  const params: Record<string, string> = { limit: String(limit), offset: String(offset) }
  if (search) params.search = search
  if (sender) params.sender = sender
  if (tag) params.tag = tag
  if (sort !== 'date') params.sort = sort

  const { data, isLoading } = useQuery({
    queryKey: ['urls', offset, search, sender, tag, sort],
    queryFn: () => getURLs(params),
  })

  const resetFilters = () => {
    setSearch('')
    setSender('')
    setTag('')
    setOffset(0)
  }

  // Collect all unique senders and tags from results for filter options
  const senders = useMemo(() => {
    if (!data?.data) return []
    const set = new Set<string>()
    data.data.forEach(u => { if (u.shared_by) set.add(u.shared_by) })
    return [...set].sort()
  }, [data])

  const allTags = useMemo(() => {
    if (!data?.data) return []
    const map = new Map<string, number>()
    data.data.forEach(u => u.tags?.forEach(t => map.set(t, (map.get(t) ?? 0) + 1)))
    return [...map.entries()].sort((a, b) => b[1] - a[1])
  }, [data])

  // Group by day
  const grouped = useMemo((): [string, URLRecord[]][] => {
    if (!data?.data || !groupByDay) return []
    const groups = new Map<string, URLRecord[]>()
    for (const u of data.data) {
      const day = format(parseISO(u.created_at), 'yyyy-MM-dd')
      if (!groups.has(day)) groups.set(day, [])
      groups.get(day)!.push(u)
    }
    return [...groups.entries()]
  }, [data, groupByDay])

  const hasFilters = search || sender || tag

  if (isLoading) return <LoadingSpinner />

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-2xl font-semibold tracking-tight">Links</h2>
        <div className="flex items-center gap-2">
          {/* Sort toggle */}
          <button
            onClick={() => { setSort(s => s === 'date' ? 'alpha' : 'date'); setOffset(0) }}
            className="p-2 rounded-lg border border-apple-border text-apple-secondary hover:text-apple-text hover:border-apple-blue/40 transition-colors text-xs"
            title={sort === 'date' ? 'Sorted by date' : 'Sorted A-Z'}
          >
            <i className={`fawsb ${sort === 'date' ? 'fa-clock' : 'fa-arrow-down-a-z'} text-sm`} />
          </button>
          {/* Group by day toggle */}
          <button
            onClick={() => setGroupByDay(g => !g)}
            className={`p-2 rounded-lg border text-xs transition-colors ${
              groupByDay
                ? 'border-apple-blue bg-apple-blue/10 text-apple-blue'
                : 'border-apple-border text-apple-secondary hover:text-apple-text hover:border-apple-blue/40'
            }`}
            title="Group by day"
          >
            <i className="fawsb fa-calendar-days text-sm" />
          </button>
        </div>
      </div>

      {/* Filter bar */}
      <div className="flex flex-wrap gap-2 mb-4">
        <input
          type="text"
          value={search}
          onChange={e => { setSearch(e.target.value); setOffset(0) }}
          placeholder="Search links..."
          className="px-3 py-1.5 rounded-lg border border-apple-border bg-white dark:bg-apple-card text-sm focus:outline-none focus:ring-2 focus:ring-apple-blue/30 focus:border-apple-blue w-48"
        />
        {/* Sender filter */}
        {(senders.length > 0 || sender) && (
          <select
            value={sender}
            onChange={e => { setSender(e.target.value); setOffset(0) }}
            className="px-3 py-1.5 rounded-lg border border-apple-border bg-white dark:bg-apple-card text-sm focus:outline-none focus:ring-2 focus:ring-apple-blue/30 text-apple-text"
          >
            <option value="">All members</option>
            {/* Show all contacts as options so filtering works even when not in current page */}
            {contacts.map(c => {
              const name = c.alias || c.profile_name || c.sender_id
              if (!name) return null
              return <option key={c.source_uuid} value={name}>{name}</option>
            })}
          </select>
        )}
        {hasFilters && (
          <button
            onClick={resetFilters}
            className="px-3 py-1.5 rounded-lg text-xs text-apple-secondary hover:text-apple-text transition-colors"
          >
            <i className="fawsb fa-xmark mr-1" />Clear
          </button>
        )}
      </div>

      {/* Tag pills */}
      {allTags.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mb-4">
          {allTags.slice(0, 20).map(([t, count]) => (
            <button
              key={t}
              onClick={() => { setTag(tag === t ? '' : t); setOffset(0) }}
              className={`px-2.5 py-0.5 text-xs rounded-full transition-colors ${
                tag === t
                  ? 'bg-blue-500 text-white'
                  : 'bg-blue-500/15 text-blue-400 hover:bg-blue-500/25'
              }`}
            >
              {t} <span className="opacity-60">{count}</span>
            </button>
          ))}
        </div>
      )}

      {(!data?.data || data.data.length === 0) ? (
        <EmptyState title="No links found" description={hasFilters ? 'Try adjusting your filters.' : 'URLs from messages will be collected here with previews.'} />
      ) : groupByDay ? (
        /* Grouped by day view */
        <>
          {grouped.map(([day, urls]) => (
            <div key={day} className="mb-6">
              <h3 className="text-sm font-medium text-apple-secondary mb-2 sticky top-0 bg-apple-bg py-1 z-10">
                <i className="fawsb fa-calendar-day mr-1.5" />
                {format(parseISO(day), 'EEEE, MMMM d')}
                <span className="ml-2 text-xs opacity-60">{urls.length} link{urls.length !== 1 ? 's' : ''}</span>
              </h3>
              <div className="space-y-3">
                {urls.map(u => <URLCard key={u.id} u={u} onTagClick={t => { setTag(tag === t ? '' : t); setOffset(0) }} activeTag={tag} />)}
              </div>
            </div>
          ))}
          <Pagination total={data.total} limit={limit} offset={offset} onChange={setOffset} />
        </>
      ) : (
        /* Flat list view */
        <>
          <div className="space-y-3">
            {data.data.map(u => <URLCard key={u.id} u={u} onTagClick={t => { setTag(tag === t ? '' : t); setOffset(0) }} activeTag={tag} />)}
          </div>
          <Pagination total={data.total} limit={limit} offset={offset} onChange={setOffset} />
        </>
      )}
    </div>
  )
}

function URLCard({ u, onTagClick, activeTag }: { u: URLRecord; onTagClick: (tag: string) => void; activeTag: string }) {
  return (
    <a href={u.url} target="_blank" rel="noopener noreferrer" className="block">
      <Card className="p-4 hover:shadow-md transition-shadow">
        <div className="flex gap-4">
          {u.image_url && (
            <img
              src={u.image_url}
              alt=""
              className="w-20 h-20 rounded-lg object-cover shrink-0"
              loading="lazy"
            />
          )}
          <div className="min-w-0 flex-1">
            <h3 className="text-sm font-semibold mb-1 line-clamp-1">
              {u.title || u.url}
            </h3>
            {u.description && (
              <p className="text-xs text-apple-secondary line-clamp-2 mb-1">
                {u.description}
              </p>
            )}
            {u.summary && u.summary !== u.description && (
              <p className="text-xs text-apple-secondary/80 italic line-clamp-1 mb-1">
                {u.summary}
              </p>
            )}
            <div className="flex items-center gap-2">
              <Badge>{u.domain}</Badge>
              <span className="text-xs text-apple-secondary">
                {format(new Date(u.created_at), 'MMM d, yyyy')}
              </span>
              {u.shared_by && (
                <span className="text-xs text-apple-secondary">
                  via {u.shared_by}
                </span>
              )}
            </div>
            {u.tags && u.tags.length > 0 && (
              <div className="flex flex-wrap gap-1 mt-1">
                {u.tags.map(t => (
                  <span
                    key={t}
                    onClick={e => { e.preventDefault(); e.stopPropagation(); onTagClick(t) }}
                    className={`px-2 py-0.5 text-xs rounded-full cursor-pointer transition-colors ${
                      activeTag === t
                        ? 'bg-blue-500 text-white'
                        : 'bg-blue-500/20 text-blue-300 hover:bg-blue-500/30'
                    }`}
                  >
                    {t}
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>
      </Card>
    </a>
  )
}
