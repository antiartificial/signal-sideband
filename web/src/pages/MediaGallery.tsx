import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getMedia, mediaThumbnailURL, mediaURL, searchMedia } from '../lib/api.ts'
import type { AttachmentRecord } from '../lib/types.ts'
import Card from '../components/Card.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import EmptyState from '../components/EmptyState.tsx'
import Pagination from '../components/Pagination.tsx'
import MediaLightbox from '../components/MediaLightbox.tsx'
import { format } from 'date-fns'

type TypeFilter = 'all' | 'images' | 'videos'

export default function MediaGallery() {
  const [offset, setOffset] = useState(0)
  const [sort, setSort] = useState('date_desc')
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [activeSearch, setActiveSearch] = useState('')
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null)
  const limit = 24

  const { data, isLoading } = useQuery({
    queryKey: ['media', offset, sort],
    queryFn: () => getMedia({ limit: String(limit), offset: String(offset), sort }),
    enabled: !activeSearch,
  })

  const { data: searchResults, isLoading: searchLoading } = useQuery({
    queryKey: ['media-search', activeSearch],
    queryFn: () => searchMedia(activeSearch, 100),
    enabled: !!activeSearch,
  })

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setActiveSearch(searchQuery)
    setOffset(0)
  }

  const clearSearch = () => {
    setSearchQuery('')
    setActiveSearch('')
  }

  // Apply type filter
  const allItems = activeSearch ? (searchResults ?? []) : (data?.data ?? [])
  const filteredItems = allItems.filter(att => {
    if (typeFilter === 'images') return att.content_type.startsWith('image/')
    if (typeFilter === 'videos') return att.content_type.startsWith('video/')
    return true
  })

  const loading = activeSearch ? searchLoading : isLoading

  if (loading) return <LoadingSpinner />

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
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

      {/* Search bar */}
      <form onSubmit={handleSearch} className="mb-4 flex gap-2">
        <div className="relative flex-1">
          <i className="fawsb fa-magnifying-glass absolute left-3 top-1/2 -translate-y-1/2 text-apple-secondary text-sm" />
          <input
            type="text"
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            placeholder="Search media by AI analysis..."
            className="w-full pl-9 pr-3 py-2 rounded-lg border border-apple-border bg-white text-sm"
          />
        </div>
        {activeSearch && (
          <button type="button" onClick={clearSearch} className="px-3 py-2 rounded-lg border border-apple-border bg-white text-sm text-apple-secondary hover:text-apple-text">
            Clear
          </button>
        )}
      </form>

      {/* Type filter tabs */}
      <div className="flex gap-1 mb-4">
        {(['all', 'images', 'videos'] as TypeFilter[]).map(t => (
          <button
            key={t}
            onClick={() => setTypeFilter(t)}
            className={`px-3 py-1.5 rounded-lg text-sm transition-colors ${
              typeFilter === t
                ? 'bg-apple-blue text-white font-medium'
                : 'bg-gray-100 text-apple-secondary hover:bg-gray-200'
            }`}
          >
            {t === 'all' ? 'All' : t === 'images' ? 'Images' : 'Videos'}
          </button>
        ))}
      </div>

      {filteredItems.length === 0 ? (
        <EmptyState
          title={activeSearch ? 'No results' : 'No media yet'}
          description={activeSearch ? `No media matched "${activeSearch}"` : 'Attachments from Signal messages will appear here.'}
        />
      ) : (
        <>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
            {filteredItems.map((att, idx) => (
              <MediaCard
                key={att.id}
                att={att}
                onClick={() => setLightboxIndex(idx)}
              />
            ))}
          </div>

          {!activeSearch && data && (
            <Pagination total={data.total} limit={limit} offset={offset} onChange={setOffset} />
          )}
        </>
      )}

      {lightboxIndex !== null && (
        <MediaLightbox
          items={filteredItems}
          currentIndex={lightboxIndex}
          onClose={() => setLightboxIndex(null)}
          onNavigate={setLightboxIndex}
        />
      )}
    </div>
  )
}

function MediaCard({ att, onClick }: { att: AttachmentRecord; onClick: () => void }) {
  const isImage = att.content_type.startsWith('image/')
  const isVideo = att.content_type.startsWith('video/')
  const hasThumb = att.downloaded && (att.thumbnail_path || isImage)

  return (
    <Card className="overflow-hidden group cursor-pointer" onClick={onClick}>
      <div className="relative">
        {hasThumb ? (
          <img
            src={mediaThumbnailURL(att.id)}
            alt={att.filename || 'attachment'}
            className="w-full aspect-square object-cover"
            loading="lazy"
          />
        ) : (
          <div className="w-full aspect-square bg-gray-50 flex flex-col items-center justify-center">
            <i className={`fawsb ${isImage ? 'fa-image' : isVideo ? 'fa-circle-play' : 'fa-file'} text-2xl text-apple-secondary mb-1`} />
            <span className="text-xs text-apple-secondary">{att.content_type}</span>
            {!att.downloaded && (
              <span className="text-xs text-apple-secondary mt-1">Downloading...</span>
            )}
          </div>
        )}

        {/* Video play overlay */}
        {isVideo && hasThumb && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/20 opacity-0 group-hover:opacity-100 transition-opacity">
            <i className="fawsb fa-circle-play text-white text-3xl drop-shadow" />
          </div>
        )}

        {/* Hover actions */}
        {att.downloaded && (
          <div className="absolute top-2 right-2 flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <a
              href={mediaURL(att.id)}
              download={att.filename || att.signal_attachment_id}
              onClick={e => e.stopPropagation()}
              className="w-7 h-7 rounded-full bg-black/50 text-white flex items-center justify-center text-xs hover:bg-black/70"
            >
              <i className="fawsb fa-arrow-down-to-bracket" />
            </a>
            <button
              onClick={e => { e.stopPropagation(); onClick() }}
              className="w-7 h-7 rounded-full bg-black/50 text-white flex items-center justify-center text-xs hover:bg-black/70"
            >
              <i className="fawsb fa-expand" />
            </button>
          </div>
        )}
      </div>

      <div className="px-3 py-2">
        <p className="text-xs text-apple-text truncate">{att.filename || att.signal_attachment_id}</p>
        {att.analysis?.description && (
          <p className="text-xs text-apple-secondary truncate mt-0.5">{att.analysis.description}</p>
        )}
        <p className="text-xs text-apple-secondary">
          {format(new Date(att.created_at), 'MMM d')}
          {att.size > 0 && ` · ${(att.size / 1024).toFixed(0)} KB`}
        </p>
      </div>
    </Card>
  )
}
