import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getURLs } from '../lib/api.ts'
import Card from '../components/Card.tsx'
import Badge from '../components/Badge.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import EmptyState from '../components/EmptyState.tsx'
import Pagination from '../components/Pagination.tsx'
import { format } from 'date-fns'

export default function URLCollection() {
  const [offset, setOffset] = useState(0)
  const [domain, setDomain] = useState('')
  const limit = 20

  const params: Record<string, string> = { limit: String(limit), offset: String(offset) }
  if (domain) params.domain = domain

  const { data, isLoading } = useQuery({
    queryKey: ['urls', offset, domain],
    queryFn: () => getURLs(params),
  })

  if (isLoading) return <LoadingSpinner />

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-semibold tracking-tight">Links</h2>
        <input
          type="text"
          value={domain}
          onChange={e => { setDomain(e.target.value); setOffset(0) }}
          placeholder="Filter by domain..."
          className="px-3 py-2 rounded-xl border border-apple-border bg-white text-sm focus:outline-none focus:ring-2 focus:ring-apple-blue/30 focus:border-apple-blue w-56"
        />
      </div>

      {(!data?.data || data.data.length === 0) ? (
        <EmptyState title="No links yet" description="URLs from messages will be collected here with previews." />
      ) : (
        <>
          <div className="space-y-3">
            {data.data.map(u => (
              <a
                key={u.id}
                href={u.url}
                target="_blank"
                rel="noopener noreferrer"
                className="block"
              >
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
                        <p className="text-xs text-apple-secondary line-clamp-2 mb-2">
                          {u.description}
                        </p>
                      )}
                      <div className="flex items-center gap-2">
                        <Badge>{u.domain}</Badge>
                        <span className="text-xs text-apple-secondary">
                          {format(new Date(u.created_at), 'MMM d, yyyy')}
                        </span>
                      </div>
                    </div>
                  </div>
                </Card>
              </a>
            ))}
          </div>
          <Pagination total={data.total} limit={limit} offset={offset} onChange={setOffset} />
        </>
      )}
    </div>
  )
}
