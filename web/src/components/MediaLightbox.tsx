import { useEffect, useCallback, useRef } from 'react'
import { mediaURL } from '../lib/api.ts'
import type { AttachmentRecord } from '../lib/types.ts'

interface Props {
  items: AttachmentRecord[]
  currentIndex: number
  onClose: () => void
  onNavigate: (index: number) => void
}

export default function MediaLightbox({ items, currentIndex, onClose, onNavigate }: Props) {
  const item = items[currentIndex]
  const isImage = item.content_type.startsWith('image/')
  const isVideo = item.content_type.startsWith('video/')
  const touchStartX = useRef<number | null>(null)

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose()
    if (e.key === 'ArrowLeft' && currentIndex > 0) onNavigate(currentIndex - 1)
    if (e.key === 'ArrowRight' && currentIndex < items.length - 1) onNavigate(currentIndex + 1)
  }, [currentIndex, items.length, onClose, onNavigate])

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.body.style.overflow = ''
    }
  }, [handleKeyDown])

  const handleTouchStart = (e: React.TouchEvent) => {
    touchStartX.current = e.touches[0].clientX
  }

  const handleTouchEnd = (e: React.TouchEvent) => {
    if (touchStartX.current === null) return
    const diff = e.changedTouches[0].clientX - touchStartX.current
    if (Math.abs(diff) > 60) {
      if (diff > 0 && currentIndex > 0) onNavigate(currentIndex - 1)
      if (diff < 0 && currentIndex < items.length - 1) onNavigate(currentIndex + 1)
    }
    touchStartX.current = null
  }

  return (
    <div
      className="fixed inset-0 z-50 bg-black/95 flex items-center justify-center animate-fade-in"
      onClick={onClose}
      onTouchStart={handleTouchStart}
      onTouchEnd={handleTouchEnd}
    >
      {/* Top bar */}
      <div className="absolute top-0 left-0 right-0 flex items-center justify-between p-4 safe-area-pt z-10">
        <div className="text-white/60 text-sm font-mono">
          {currentIndex + 1} / {items.length}
        </div>
        <div className="flex items-center gap-2">
          <a
            href={mediaURL(item.id)}
            download={item.filename || item.signal_attachment_id}
            onClick={e => e.stopPropagation()}
            className="text-white/80 hover:text-white text-xl w-11 h-11 flex items-center justify-center
              rounded-full active:bg-white/10 transition-colors"
          >
            <i className="fawsb fa-arrow-down-to-bracket" />
          </a>
          <button
            onClick={onClose}
            className="text-white/80 hover:text-white text-2xl w-11 h-11 flex items-center justify-center
              rounded-full active:bg-white/10 transition-colors"
          >
            <i className="fawsb fa-xmark" />
          </button>
        </div>
      </div>

      {/* Previous — hidden on mobile (swipe instead) */}
      {currentIndex > 0 && (
        <button
          onClick={e => { e.stopPropagation(); onNavigate(currentIndex - 1) }}
          className="hidden md:flex absolute left-4 top-1/2 -translate-y-1/2 text-white/60 hover:text-white text-3xl z-10
            w-12 h-12 items-center justify-center rounded-full hover:bg-white/10 transition-colors"
        >
          <i className="fawsb fa-chevron-left" />
        </button>
      )}

      {/* Next — hidden on mobile (swipe instead) */}
      {currentIndex < items.length - 1 && (
        <button
          onClick={e => { e.stopPropagation(); onNavigate(currentIndex + 1) }}
          className="hidden md:flex absolute right-4 top-1/2 -translate-y-1/2 text-white/60 hover:text-white text-3xl z-10
            w-12 h-12 items-center justify-center rounded-full hover:bg-white/10 transition-colors"
        >
          <i className="fawsb fa-chevron-right" />
        </button>
      )}

      {/* Media content */}
      <div className="w-full h-full flex flex-col items-center justify-center px-4 pt-16 pb-4" onClick={e => e.stopPropagation()}>
        {isImage && (
          <img
            src={mediaURL(item.id)}
            alt={item.filename || 'media'}
            className="max-w-full max-h-[70vh] sm:max-h-[75vh] object-contain rounded animate-scale-in"
          />
        )}
        {isVideo && (
          <video
            src={mediaURL(item.id)}
            controls
            autoPlay
            playsInline
            className="max-w-full max-h-[70vh] sm:max-h-[75vh] rounded animate-scale-in"
          />
        )}
        {!isImage && !isVideo && (
          <div className="text-white text-center animate-fade-in">
            <i className="fawsb fa-file text-5xl mb-4" />
            <p>{item.filename || item.content_type}</p>
          </div>
        )}

        {/* AI analysis overlay */}
        {item.analysis && (
          <div className="mt-3 bg-black/70 backdrop-blur-sm rounded-lg p-4 max-w-2xl w-full text-white/90 text-sm animate-fade-in-up">
            {item.analysis.description && (
              <p className="mb-2">{item.analysis.description}</p>
            )}
            <div className="flex flex-wrap gap-1.5">
              {item.analysis.objects && item.analysis.objects.split(',').map((obj, i) => (
                <span key={i} className="px-2 py-0.5 bg-white/15 rounded text-xs">{obj.trim()}</span>
              ))}
              {item.analysis.colors && item.analysis.colors.split(',').map((color, i) => (
                <span key={`c-${i}`} className="px-2 py-0.5 bg-white/15 rounded text-xs">{color.trim()}</span>
              ))}
              {item.analysis.scene && (
                <span className="px-2 py-0.5 bg-apple-accent/30 rounded text-xs">{item.analysis.scene}</span>
              )}
            </div>
            {item.analysis.text_content && (
              <p className="mt-2 text-white/70 text-xs italic">Text: {item.analysis.text_content}</p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
