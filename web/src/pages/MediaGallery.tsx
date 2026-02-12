import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getMedia, mediaURL } from '../lib/api.ts'
import Card from '../components/Card.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import EmptyState from '../components/EmptyState.tsx'
import Pagination from '../components/Pagination.tsx'
import { format } from 'date-fns'

export default function MediaGallery() {
  const [offset, setOffset] = useState(0)
  const [sort, setSort] = useState('date_desc')
  const limit = 24

  const { data, isLoading } = useQuery({
    queryKey: ['media', offset, sort],
    queryFn: () => getMedia({ limit: String(limit), offset: String(offset), sort }),
  })

  if (isLoading) return <LoadingSpinner />

  if (!data?.data || data.data.length === 0) {
    return (
      <div>
        <h2 className="text-2xl font-semibold tracking-tight mb-6">Media Gallery</h2>
        <EmptyState title="No media yet" description="Attachments from Signal messages will appear here." />
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-semibold tracking-tight">Media Gallery</h2>
        <select
          value={sort}
          onChange={e => { setSort(e.target.value); setOffset(0) }}
          className="px-3 py-1.5 rounded-lg border border-apple-border bg-white text-sm"
        >
          <option value="date_desc">Newest</option>
          <option value="date_asc">Oldest</option>
          <option value="size_desc">Largest</option>
          <option value="size_asc">Smallest</option>
          <option value="type">By type</option>
        </select>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
        {data.data.map(att => {
          const isImage = att.content_type.startsWith('image/')
          const isVideo = att.content_type.startsWith('video/')

          return (
            <Card key={att.id} className="overflow-hidden">
              {isImage && att.downloaded ? (
                <img
                  src={mediaURL(att.id)}
                  alt={att.filename || 'attachment'}
                  className="w-full aspect-square object-cover"
                  loading="lazy"
                />
              ) : isVideo && att.downloaded ? (
                <video
                  src={mediaURL(att.id)}
                  className="w-full aspect-square object-cover"
                  controls
                />
              ) : (
                <div className="w-full aspect-square bg-gray-50 flex flex-col items-center justify-center">
                  <span className="text-2xl text-apple-secondary mb-1">
                    {isImage ? '⧉' : isVideo ? '▶' : '◎'}
                  </span>
                  <span className="text-xs text-apple-secondary">{att.content_type}</span>
                  {!att.downloaded && (
                    <span className="text-xs text-apple-secondary mt-1">Downloading...</span>
                  )}
                </div>
              )}
              <div className="px-3 py-2">
                <p className="text-xs text-apple-text truncate">{att.filename || att.signal_attachment_id}</p>
                <p className="text-xs text-apple-secondary">
                  {format(new Date(att.created_at), 'MMM d')}
                  {att.size > 0 && ` · ${(att.size / 1024).toFixed(0)} KB`}
                </p>
              </div>
            </Card>
          )
        })}
      </div>

      <Pagination total={data.total} limit={limit} offset={offset} onChange={setOffset} />
    </div>
  )
}
