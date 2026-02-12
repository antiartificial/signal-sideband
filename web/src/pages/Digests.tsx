import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { getDigests } from '../lib/api.ts'
import Card from '../components/Card.tsx'
import Badge from '../components/Badge.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import EmptyState from '../components/EmptyState.tsx'
import Pagination from '../components/Pagination.tsx'
import { format } from 'date-fns'

export default function Digests() {
  const navigate = useNavigate()
  const [offset, setOffset] = useState(0)
  const limit = 12

  const { data, isLoading } = useQuery({
    queryKey: ['digests', offset],
    queryFn: () => getDigests({ limit: String(limit), offset: String(offset) }),
  })

  if (isLoading) return <LoadingSpinner />

  return (
    <div>
      <h2 className="text-2xl font-semibold tracking-tight mb-6">Digests</h2>

      {(!data?.data || data.data.length === 0) ? (
        <EmptyState
          title="No digests yet"
          description="Generate your first digest to see a summary of conversations."
        />
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {data.data.map(digest => (
              <Card
                key={digest.id}
                className="p-6"
                onClick={() => navigate(`/digests/${digest.id}`)}
              >
                <h3 className="text-base font-semibold mb-2 line-clamp-2">{digest.title}</h3>
                <p className="text-sm text-apple-secondary line-clamp-3 mb-3">
                  {digest.summary.slice(0, 180)}...
                </p>
                <div className="flex items-center gap-2 flex-wrap mb-2">
                  {(digest.topics || []).slice(0, 3).map((topic: string) => (
                    <Badge key={topic}>{topic}</Badge>
                  ))}
                </div>
                <p className="text-xs text-apple-secondary">
                  {format(new Date(digest.period_start), 'MMM d')} – {format(new Date(digest.period_end), 'MMM d, yyyy')}
                </p>
              </Card>
            ))}
          </div>
          <Pagination total={data.total} limit={limit} offset={offset} onChange={setOffset} />
        </>
      )}
    </div>
  )
}
