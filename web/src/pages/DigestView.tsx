import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import Markdown from 'react-markdown'
import { getDigest } from '../lib/api.ts'
import Card from '../components/Card.tsx'
import Badge from '../components/Badge.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import { format } from 'date-fns'

export default function DigestView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: digest, isLoading } = useQuery({
    queryKey: ['digest', id],
    queryFn: () => getDigest(id!),
    enabled: !!id,
  })

  if (isLoading) return <LoadingSpinner />
  if (!digest) return null

  return (
    <div>
      <button
        onClick={() => navigate('/digests')}
        className="text-sm text-apple-blue hover:underline mb-4 inline-block"
      >
        &larr; All Digests
      </button>

      <article className="max-w-[720px]">
        {/* Header */}
        <header className="mb-8">
          <h1 className="text-3xl font-bold tracking-tight mb-2">{digest.title}</h1>
          <p className="text-sm text-apple-secondary">
            {format(new Date(digest.period_start), 'MMMM d')} – {format(new Date(digest.period_end), 'MMMM d, yyyy')}
            {digest.llm_model && (
              <span className="ml-3 text-apple-secondary/60">via {digest.llm_model}</span>
            )}
          </p>
        </header>

        {/* Topics */}
        {digest.topics && digest.topics.length > 0 && (
          <div className="flex gap-2 flex-wrap mb-6">
            {digest.topics.map((topic: string) => (
              <Badge key={topic} variant="blue">{topic}</Badge>
            ))}
          </div>
        )}

        {/* Summary */}
        <Card className="p-8 mb-6">
          <div className="prose">
            <Markdown>{digest.summary}</Markdown>
          </div>
        </Card>

        {/* Decisions */}
        {digest.decisions && digest.decisions.length > 0 && (
          <Card className="p-6 mb-4">
            <h2 className="text-base font-semibold mb-3">Decisions</h2>
            <ul className="space-y-2">
              {digest.decisions.map((d: string, i: number) => (
                <li key={i} className="flex items-start gap-2 text-sm">
                  <span className="text-apple-blue mt-0.5 shrink-0">●</span>
                  {d}
                </li>
              ))}
            </ul>
          </Card>
        )}

        {/* Action Items */}
        {digest.action_items && digest.action_items.length > 0 && (
          <Card className="p-6">
            <h2 className="text-base font-semibold mb-3">Action Items</h2>
            <ul className="space-y-2">
              {digest.action_items.map((a: string, i: number) => (
                <li key={i} className="flex items-start gap-2 text-sm">
                  <span className="text-green-600 mt-0.5 shrink-0">☐</span>
                  {a}
                </li>
              ))}
            </ul>
          </Card>
        )}
      </article>
    </div>
  )
}
