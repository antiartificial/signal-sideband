import { useState, useEffect, useRef, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import cytoscape from 'cytoscape'
import { getCerebroGraph, getCerebroConcept, enrichCerebroConcept, extractCerebro } from '../lib/api.ts'
import type { CerebroConceptDetail, CerebroEnrichment, PerplexityEnrichment, GrokXEnrichment, GrokBooksEnrichment } from '../lib/types.ts'

const CATEGORY_COLORS: Record<string, string> = {
  topic: '#3B82F6',
  person: '#F97316',
  place: '#22C55E',
  media: '#A855F7',
  event: '#EF4444',
  idea: '#EAB308',
}

const CATEGORY_LABELS: Record<string, string> = {
  topic: 'Topic',
  person: 'Person',
  place: 'Place',
  media: 'Media',
  event: 'Event',
  idea: 'Idea',
}

export default function Cerebro() {
  const queryClient = useQueryClient()
  const containerRef = useRef<HTMLDivElement>(null)
  const cyRef = useRef<cytoscape.Core | null>(null)
  const [selectedConcept, setSelectedConcept] = useState<CerebroConceptDetail | null>(null)
  const [showPanel, setShowPanel] = useState(false)
  const [extracting, setExtracting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { data: graph, isLoading } = useQuery({
    queryKey: ['cerebro-graph'],
    queryFn: () => getCerebroGraph({ limit: 60 }),
  })

  const enrichMutation = useMutation({
    mutationFn: (id: string) => enrichCerebroConcept(id),
    onSuccess: (data) => {
      setSelectedConcept(data)
      queryClient.invalidateQueries({ queryKey: ['cerebro-graph'] })
    },
  })

  const handleExtract = async () => {
    setExtracting(true)
    setError(null)
    try {
      await extractCerebro()
      queryClient.invalidateQueries({ queryKey: ['cerebro-graph'] })
    } catch (e: any) {
      setError(e.message || 'Extraction failed')
    } finally {
      setExtracting(false)
    }
  }

  const handleNodeTap = useCallback(async (id: string) => {
    try {
      const detail = await getCerebroConcept(id)
      setSelectedConcept(detail)
      setShowPanel(true)
    } catch {
      // ignore
    }
  }, [])

  // Initialize Cytoscape
  useEffect(() => {
    if (!containerRef.current || !graph || graph.concepts.length === 0) return

    const maxMentions = Math.max(...graph.concepts.map(c => c.mention_count), 1)

    const elements: cytoscape.ElementDefinition[] = [
      ...graph.concepts.map(c => ({
        data: {
          id: c.id,
          label: c.name,
          category: c.category,
          mentionCount: c.mention_count,
          color: CATEGORY_COLORS[c.category] || '#6B7280',
          size: 20 + (c.mention_count / maxMentions) * 40,
        },
      })),
      ...graph.edges.map(e => ({
        data: {
          id: e.id,
          source: e.source_id,
          target: e.target_id,
          label: e.relation,
          weight: e.weight,
        },
      })),
    ]

    if (cyRef.current) {
      cyRef.current.destroy()
    }

    const cy = cytoscape({
      container: containerRef.current,
      elements,
      boxSelectionEnabled: false,
      wheelSensitivity: 0.3,
      style: [
        {
          selector: 'node',
          style: {
            'background-color': 'data(color)',
            'label': 'data(label)',
            'width': 'data(size)',
            'height': 'data(size)',
            'font-size': '10px',
            'color': '#E5E7EB',
            'text-valign': 'bottom',
            'text-margin-y': 6,
            'text-outline-width': 2,
            'text-outline-color': '#1F2937',
            'min-zoomed-font-size': 8,
            'border-width': 2,
            'border-color': '#374151',
          } as any,
        },
        {
          selector: 'edge',
          style: {
            'width': 1.5,
            'line-color': '#4B5563',
            'target-arrow-color': '#4B5563',
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
            'arrow-scale': 0.8,
            'label': 'data(label)',
            'font-size': '8px',
            'color': '#6B7280',
            'text-rotation': 'autorotate',
            'text-outline-width': 1,
            'text-outline-color': '#1F2937',
          } as any,
        },
        {
          selector: 'node:selected',
          style: {
            'border-width': 3,
            'border-color': '#FFFFFF',
          },
        },
      ],
      layout: {
        name: 'cose',
        animate: true,
        animationDuration: 500,
        nodeRepulsion: () => 8000,
        idealEdgeLength: () => 100,
        gravity: 0.3,
        padding: 40,
      } as any,
    })

    cy.on('tap', 'node', (evt) => {
      const nodeId = evt.target.id()
      handleNodeTap(nodeId)
    })

    cy.on('tap', (evt) => {
      if (evt.target === cy) {
        setShowPanel(false)
        setSelectedConcept(null)
      }
    })

    cyRef.current = cy

    return () => {
      cy.destroy()
      cyRef.current = null
    }
  }, [graph, handleNodeTap])

  const isEmpty = !graph || graph.concepts.length === 0

  return (
    <div className="flex flex-col" style={{ height: 'calc(100vh - 8rem)' }}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-apple-border shrink-0">
        <div className="flex items-center gap-2">
          <i className="fawsb fa-circle-nodes text-apple-blue text-lg" />
          <h1 className="text-lg font-semibold text-apple-text">Cerebro</h1>
        </div>
        <button
          onClick={handleExtract}
          disabled={extracting}
          className="px-3 py-1.5 bg-apple-blue text-white text-sm rounded-lg hover:bg-blue-600
            disabled:opacity-50 transition-all flex items-center gap-1.5"
        >
          {extracting ? (
            <>
              <i className="fawsb fa-spinner fa-spin text-xs" />
              Scanning...
            </>
          ) : (
            <>
              <i className="fawsb fa-bolt text-xs" />
              Scan Messages
            </>
          )}
        </button>
      </div>

      {/* Error */}
      {error && (
        <div className="mx-4 mt-2 px-3 py-2 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">
          {error}
        </div>
      )}

      {/* Legend */}
      {!isEmpty && (
        <div className="flex items-center gap-3 px-4 py-2 border-b border-apple-border overflow-x-auto shrink-0">
          {Object.entries(CATEGORY_COLORS).map(([cat, color]) => (
            <div key={cat} className="flex items-center gap-1.5 text-xs text-apple-secondary whitespace-nowrap">
              <span className="w-2.5 h-2.5 rounded-full shrink-0" style={{ backgroundColor: color }} />
              {CATEGORY_LABELS[cat]}
            </div>
          ))}
        </div>
      )}

      {/* Main content */}
      <div className="flex-1 flex min-h-0 relative">
        {isLoading ? (
          <div className="flex-1 flex items-center justify-center">
            <i className="fawsb fa-spinner fa-spin text-2xl text-apple-secondary" />
          </div>
        ) : isEmpty ? (
          <div className="flex-1 flex flex-col items-center justify-center text-apple-secondary gap-4">
            <i className="fawsb fa-circle-nodes text-5xl opacity-30" />
            <p className="text-sm">No concepts extracted yet</p>
            <button
              onClick={handleExtract}
              disabled={extracting}
              className="px-4 py-2 bg-apple-blue text-white text-sm rounded-lg hover:bg-blue-600
                disabled:opacity-50 transition-all flex items-center gap-2"
            >
              <i className="fawsb fa-bolt text-xs" />
              Scan Messages to Build Graph
            </button>
          </div>
        ) : (
          <>
            {/* Graph */}
            <div
              ref={containerRef}
              className={`flex-1 min-h-0 ${showPanel ? 'md:mr-80' : ''} transition-all`}
            />

            {/* Desktop side panel */}
            {showPanel && selectedConcept && (
              <div className="hidden md:flex flex-col w-80 border-l border-apple-border bg-apple-sidebar absolute right-0 top-0 bottom-0 overflow-y-auto animate-slide-in">
                <ConceptPanel
                  concept={selectedConcept}
                  onClose={() => { setShowPanel(false); setSelectedConcept(null) }}
                  onEnrich={() => enrichMutation.mutate(selectedConcept.id)}
                  enriching={enrichMutation.isPending}
                />
              </div>
            )}

            {/* Mobile bottom sheet */}
            {showPanel && selectedConcept && (
              <div className="md:hidden fixed inset-0 z-50">
                <div
                  className="absolute inset-0 bg-black/40"
                  onClick={() => { setShowPanel(false); setSelectedConcept(null) }}
                />
                <div className="absolute bottom-0 left-0 right-0 max-h-[60vh] bg-apple-sidebar border-t border-apple-border
                  rounded-t-2xl overflow-y-auto animate-slide-up safe-area-pb">
                  <ConceptPanel
                    concept={selectedConcept}
                    onClose={() => { setShowPanel(false); setSelectedConcept(null) }}
                    onEnrich={() => enrichMutation.mutate(selectedConcept.id)}
                    enriching={enrichMutation.isPending}
                  />
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function ConceptPanel({
  concept,
  onClose,
  onEnrich,
  enriching,
}: {
  concept: CerebroConceptDetail
  onClose: () => void
  onEnrich: () => void
  enriching: boolean
}) {
  return (
    <div className="p-4">
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <span
            className="w-3 h-3 rounded-full shrink-0"
            style={{ backgroundColor: CATEGORY_COLORS[concept.category] || '#6B7280' }}
          />
          <h2 className="text-base font-semibold text-apple-text">{concept.name}</h2>
        </div>
        <button onClick={onClose} className="text-apple-secondary hover:text-apple-text p-1">
          <i className="fawsb fa-xmark" />
        </button>
      </div>

      {/* Meta */}
      <div className="flex items-center gap-3 text-xs text-apple-secondary mb-3">
        <span className="px-1.5 py-0.5 rounded bg-apple-accent-dim capitalize">{concept.category}</span>
        <span>{concept.mention_count} mentions</span>
      </div>

      {/* Description */}
      {concept.description && (
        <p className="text-sm text-apple-text mb-4">{concept.description}</p>
      )}

      {/* Edges */}
      {concept.edges.length > 0 && (
        <div className="mb-4">
          <h3 className="text-xs font-medium text-apple-secondary uppercase tracking-wide mb-2">Connections</h3>
          <div className="space-y-1">
            {concept.edges.slice(0, 10).map(e => (
              <div key={e.id} className="text-xs text-apple-text flex items-center gap-1.5">
                <i className="fawsb fa-arrow-right text-apple-secondary text-[10px]" />
                <span className="text-apple-secondary">{e.relation}</span>
                <span className="font-medium">
                  {e.source_id === concept.id ? '(target)' : '(source)'}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Enrich button */}
      <button
        onClick={onEnrich}
        disabled={enriching}
        className="w-full mb-4 px-3 py-2 bg-apple-blue/10 text-apple-blue text-sm rounded-lg
          hover:bg-apple-blue/20 disabled:opacity-50 transition-all flex items-center justify-center gap-2"
      >
        {enriching ? (
          <>
            <i className="fawsb fa-spinner fa-spin text-xs" />
            Enriching...
          </>
        ) : (
          <>
            <i className="fawsb fa-brain text-xs" />
            Enrich with Cerebro
          </>
        )}
      </button>

      {/* Enrichments */}
      {concept.enrichments.length > 0 && (
        <div className="space-y-3">
          {concept.enrichments.map(e => (
            <EnrichmentCard key={e.id} enrichment={e} />
          ))}
        </div>
      )}
    </div>
  )
}

function EnrichmentCard({ enrichment }: { enrichment: CerebroEnrichment }) {
  const sourceLabels: Record<string, string> = {
    perplexity: 'Knowledge',
    grok_x: 'X / Trending',
    grok_books: 'Books & Articles',
  }
  const sourceIcons: Record<string, string> = {
    perplexity: 'fa-globe',
    grok_x: 'fa-hashtag',
    grok_books: 'fa-book',
  }

  const content = enrichment.content as Record<string, unknown>

  return (
    <div className="border border-apple-border rounded-lg p-3">
      <div className="flex items-center gap-2 mb-2">
        <i className={`fawsb ${sourceIcons[enrichment.source] || 'fa-circle-info'} text-apple-blue text-xs`} />
        <span className="text-xs font-medium text-apple-text">
          {sourceLabels[enrichment.source] || enrichment.source}
        </span>
        {enrichment.expires_at && (
          <span className="text-[10px] text-apple-secondary ml-auto">TTL</span>
        )}
      </div>

      {enrichment.source === 'perplexity' && <PerplexityCard content={content as unknown as PerplexityEnrichment} />}
      {enrichment.source === 'grok_x' && <GrokXCard content={content as unknown as GrokXEnrichment} />}
      {enrichment.source === 'grok_books' && <GrokBooksCard content={content as unknown as GrokBooksEnrichment} />}
    </div>
  )
}

function PerplexityCard({ content }: { content: PerplexityEnrichment }) {
  return (
    <div className="text-xs space-y-2">
      {content.summary && <p className="text-apple-text">{content.summary}</p>}
      {content.key_facts?.length > 0 && (
        <div>
          <span className="text-apple-secondary font-medium">Key Facts:</span>
          <ul className="mt-1 space-y-0.5">
            {content.key_facts.map((f, i) => (
              <li key={i} className="text-apple-text flex gap-1.5">
                <span className="text-apple-secondary">-</span> {f}
              </li>
            ))}
          </ul>
        </div>
      )}
      {content.related_topics?.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {content.related_topics.map((t, i) => (
            <span key={i} className="px-1.5 py-0.5 bg-apple-accent-dim rounded text-apple-secondary text-[10px]">
              {t}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}

function GrokXCard({ content }: { content: GrokXEnrichment }) {
  return (
    <div className="text-xs space-y-2">
      {content.sentiment && (
        <div className="flex items-center gap-2">
          <span className="text-apple-secondary">Sentiment:</span>
          <span className="text-apple-text capitalize">{content.sentiment}</span>
          {content.trending_score && (
            <span className="px-1.5 py-0.5 bg-apple-accent-dim rounded text-apple-secondary text-[10px] ml-auto">
              {content.trending_score}
            </span>
          )}
        </div>
      )}
      {content.trending_posts?.length > 0 && (
        <ul className="space-y-1.5">
          {content.trending_posts.map((p, i) => (
            <li key={i} className="text-apple-text">
              <p>{p.summary}</p>
              {p.context && <p className="text-apple-secondary mt-0.5">{p.context}</p>}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function GrokBooksCard({ content }: { content: GrokBooksEnrichment }) {
  return (
    <div className="text-xs space-y-2">
      {content.books?.length > 0 && (
        <div>
          <span className="text-apple-secondary font-medium">Books:</span>
          <ul className="mt-1 space-y-1">
            {content.books.map((b, i) => (
              <li key={i} className="text-apple-text">
                <span className="font-medium">{b.title}</span>
                {b.author && <span className="text-apple-secondary"> by {b.author}</span>}
                {b.relevance && <p className="text-apple-secondary mt-0.5">{b.relevance}</p>}
              </li>
            ))}
          </ul>
        </div>
      )}
      {content.articles?.length > 0 && (
        <div>
          <span className="text-apple-secondary font-medium">Articles:</span>
          <ul className="mt-1 space-y-1">
            {content.articles.map((a, i) => (
              <li key={i} className="text-apple-text">
                <span className="font-medium">{a.title}</span>
                {a.source && <span className="text-apple-secondary"> - {a.source}</span>}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
