import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { getStats, getMessages, generateDigest, generateInsight } from '../lib/api.ts'
import { useContacts } from '../lib/useContacts.ts'
import Card from '../components/Card.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import { format, subDays } from 'date-fns'

export default function Dashboard() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { resolveName } = useContacts()
  const { data: stats, isLoading } = useQuery({ queryKey: ['stats'], queryFn: getStats })
  const { data: recent } = useQuery({
    queryKey: ['messages', 'recent'],
    queryFn: () => getMessages({ limit: '5' }),
  })
  const [generating, setGenerating] = useState(false)
  const [activeLens, setActiveLens] = useState<string | null>(null)
  const [generatingInsight, setGeneratingInsight] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const lenses = [
    { id: 'default', label: 'Standard', icon: 'fa-newspaper', desc: 'Straight newsletter' },
    { id: 'gondor', label: 'Gondor', icon: 'fa-shield-halved', desc: 'Chronicles of the Citadel' },
    { id: 'confucius', label: 'Confucius', icon: 'fa-book-open', desc: 'The Master says...' },
    { id: 'city-wok', label: 'City Wok', icon: 'fa-fire', desc: 'Goddamn Mongorians!' },
  ]

  const showError = (msg: string) => {
    setError(msg)
    setTimeout(() => setError(null), 8000)
  }

  const handleGenerateInsight = async () => {
    setGeneratingInsight(true)
    setError(null)
    try {
      await generateInsight()
      queryClient.invalidateQueries({ queryKey: ['stats'] })
    } catch (e: any) {
      showError(`Insight generation failed: ${e.message}`)
    } finally {
      setGeneratingInsight(false)
    }
  }

  const handleGenerateDigest = async (lens: string) => {
    setGenerating(true)
    setActiveLens(lens)
    setError(null)
    try {
      const now = new Date()
      const yesterday = subDays(now, 1)
      await generateDigest(
        format(yesterday, "yyyy-MM-dd'T'HH:mm:ssxxx"),
        format(now, "yyyy-MM-dd'T'HH:mm:ssxxx"),
        undefined,
        lens === 'default' ? undefined : lens,
      )
      queryClient.invalidateQueries({ queryKey: ['stats'] })
      queryClient.invalidateQueries({ queryKey: ['digests'] })
    } catch (e: any) {
      showError(`Digest generation failed: ${e.message}`)
    } finally {
      setGenerating(false)
      setActiveLens(null)
    }
  }

  if (isLoading) return <LoadingSpinner />

  const insight = stats?.daily_insight

  const statCards = [
    { label: 'Total Messages', value: stats?.total_messages ?? 0, icon: 'fa-comments' },
    { label: 'Today', value: stats?.today_messages ?? 0, icon: 'fa-calendar' },
    { label: 'Groups', value: stats?.total_groups ?? 0, icon: 'fa-users' },
    { label: 'Links Collected', value: stats?.total_urls ?? 0, icon: 'fa-link' },
  ]

  return (
    <div>
      <h2 className="text-2xl font-semibold tracking-tight mb-6">Dashboard</h2>

      {/* Error banner */}
      {error && (
        <div className="mb-4 p-3 rounded-lg bg-red-50 dark:bg-red-500/10 border border-red-200 dark:border-red-500/20 text-sm text-red-700 dark:text-red-400 flex items-center gap-2">
          <i className="fawsb fa-triangle-exclamation" />
          {error}
          <button onClick={() => setError(null)} className="ml-auto text-red-400 hover:text-red-600">
            <i className="fawsb fa-xmark" />
          </button>
        </div>
      )}

      {/* Stats grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        {statCards.map((s, i) => (
          <Card key={s.label} className="p-5" style={{ animationDelay: `${i * 80}ms` }}>
            <div className="flex items-center gap-2 mb-1">
              <i className={`fawsb ${s.icon} text-apple-blue text-sm`} />
              <p className="text-sm text-apple-secondary">{s.label}</p>
            </div>
            <p className="text-3xl font-semibold tracking-tight font-mono">{s.value.toLocaleString()}</p>
          </Card>
        ))}
      </div>

      {/* Daily insight */}
      {insight ? (
        <div className="mb-8 space-y-4">
          {/* Overview */}
          {insight.overview && (
            <Card className="p-6">
              <div className="flex items-center gap-2 mb-3">
                <i className="fawsb fa-sparkles text-apple-blue" />
                <h3 className="text-lg font-medium">Today's Overview</h3>
              </div>
              <p className="text-sm text-apple-secondary leading-relaxed">{insight.overview}</p>
            </Card>
          )}

          {/* Themes */}
          {insight.themes && insight.themes.length > 0 && (
            <Card className="p-6">
              <div className="flex items-center gap-2 mb-3">
                <i className="fawsb fa-tag text-apple-blue" />
                <h3 className="text-lg font-medium">Topics & Themes</h3>
              </div>
              <div className="flex flex-wrap gap-2">
                {insight.themes.map((theme: string, i: number) => {
                  const colors = [
                    'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300',
                    'bg-purple-100 text-purple-700 dark:bg-purple-500/15 dark:text-purple-300',
                    'bg-green-100 text-green-700 dark:bg-green-500/15 dark:text-green-300',
                    'bg-orange-100 text-orange-700 dark:bg-orange-500/15 dark:text-orange-300',
                    'bg-pink-100 text-pink-700 dark:bg-pink-500/15 dark:text-pink-300',
                  ]
                  return (
                    <span key={i} className={`px-3 py-1 rounded-full text-sm font-medium ${colors[i % colors.length]}`}>
                      {theme}
                    </span>
                  )
                })}
              </div>
            </Card>
          )}

          {/* Quote of the Day */}
          {insight.quote_content && (
            <Card className="p-6">
              <div className="flex items-center gap-2 mb-3">
                <i className="fawsb fa-quotes text-apple-blue" />
                <h3 className="text-lg font-medium">Quote of the Day</h3>
              </div>
              <blockquote className="border-l-3 border-apple-blue pl-4 py-1">
                <p className="text-sm text-apple-text italic leading-relaxed">"{insight.quote_content}"</p>
                {insight.quote_sender && (
                  <p className="text-xs text-apple-secondary mt-2">— {insight.quote_sender}</p>
                )}
              </blockquote>
            </Card>
          )}
        </div>
      ) : (
        <div className="mb-8">
          <Card className="p-8 text-center">
            <i className="fawsb fa-sparkles text-3xl text-apple-secondary mb-3 block" />
            <p className="text-sm text-apple-secondary mb-4">No daily insight yet. Generate one to see today's overview, themes, and quote of the day.</p>
            <button
              onClick={handleGenerateInsight}
              disabled={generatingInsight}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-apple-blue text-white text-sm font-medium
                hover:bg-apple-blue/90 active:scale-95 transition-all duration-200 disabled:opacity-50"
            >
              <i className={`fawsb ${generatingInsight ? 'fa-sparkles animate-pulse' : 'fa-wand-magic-sparkles'}`} />
              {generatingInsight ? 'Generating insight...' : 'Generate Daily Insight'}
            </button>
          </Card>
        </div>
      )}

      {/* Digest lenses — fortune cookie style */}
      <div className="mb-8">
        <h3 className="text-lg font-medium mb-3">
          <i className="fawsb fa-newspaper text-apple-secondary mr-2" />
          Crack Open a Digest
        </h3>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-4">
          {lenses.map(lens => (
            <button
              key={lens.id}
              onClick={() => handleGenerateDigest(lens.id)}
              disabled={generating}
              className={`relative p-4 rounded-xl border text-left transition-all duration-200
                active:scale-95 disabled:opacity-50 group
                ${generating && activeLens === lens.id
                  ? 'border-apple-blue bg-apple-blue/10'
                  : 'border-apple-border bg-apple-card hover:border-apple-blue/50 hover:shadow-sm'
                }`}
            >
              <i className={`fawsb ${lens.icon} text-lg mb-2 block
                ${generating && activeLens === lens.id ? 'text-apple-blue animate-pulse' : 'text-apple-secondary group-hover:text-apple-blue'}
                transition-colors`}
              />
              <p className="text-sm font-medium text-apple-text">{lens.label}</p>
              <p className="text-xs text-apple-secondary mt-0.5">{lens.desc}</p>
              {generating && activeLens === lens.id && (
                <span className="absolute top-3 right-3 text-xs text-apple-blue font-mono">brewing...</span>
              )}
            </button>
          ))}
        </div>

        {stats?.latest_digest ? (
          <Card
            className="p-6 cursor-pointer hover:shadow-md transition-shadow"
            onClick={() => navigate(`/digests/${stats.latest_digest!.id}`)}
          >
            <div className="flex items-center gap-2 mb-2">
              <h4 className="text-base font-semibold">{stats.latest_digest.title}</h4>
            </div>
            <p className="text-sm text-apple-secondary line-clamp-3">
              {stats.latest_digest.summary.slice(0, 200)}...
            </p>
            <p className="text-xs text-apple-secondary mt-3">
              {format(new Date(stats.latest_digest.created_at), 'MMM d, yyyy')}
            </p>
          </Card>
        ) : (
          <Card className="p-8 text-center">
            <p className="text-sm text-apple-secondary">No digests yet. Pick a lens above and crack one open.</p>
          </Card>
        )}
      </div>

      {/* Superlatives */}
      {stats?.superlatives && stats.superlatives.length > 0 && (
        <div className="mb-8">
          <h3 className="text-lg font-medium mb-3">
            <i className="fawsb fa-crown text-apple-secondary mr-2" />
            Superlatives
          </h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3">
            {stats.superlatives.map((s, i) => (
              <Card key={s.label} className="p-4" style={{ animationDelay: `${i * 60}ms` }}>
                <i className={`fawsb ${s.icon} text-apple-blue text-lg mb-2 block`} />
                <p className="text-xs font-semibold text-apple-blue uppercase tracking-wide">{s.label}</p>
                <p className="text-sm text-apple-text font-medium mt-1 truncate" title={s.winner}>
                  {s.winner === 'self' ? 'You' : s.winner}
                </p>
                <p className="text-xs text-apple-secondary mt-0.5">{s.value}</p>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Recent messages — chat thread */}
      <div>
        <h3 className="text-lg font-medium mb-3">
          <i className="fawsb fa-clock text-apple-secondary mr-2" />
          Recent Messages
        </h3>
        <Card className="p-4 space-y-3">
          {recent?.data.map(msg => (
            <div
              key={msg.id}
              className={`flex ${msg.is_outgoing ? 'justify-end' : 'justify-start'}`}
            >
              <div className={`max-w-[80%] ${msg.is_outgoing ? 'items-end' : 'items-start'}`}>
                {!msg.is_outgoing && (
                  <p className="text-xs font-medium text-apple-secondary mb-0.5 px-1 truncate max-w-[200px]">
                    {resolveName(msg.source_uuid || msg.sender_id)}
                  </p>
                )}
                <div
                  className={`px-3.5 py-2 rounded-2xl text-sm leading-relaxed ${
                    msg.is_outgoing
                      ? 'bg-apple-blue text-white rounded-br-md'
                      : 'bg-gray-100 dark:bg-white/[0.06] text-apple-text rounded-bl-md'
                  }`}
                >
                  {msg.content}
                </div>
                <p className={`text-[10px] text-apple-secondary mt-0.5 px-1 ${msg.is_outgoing ? 'text-right' : ''}`}>
                  {format(new Date(msg.created_at), 'h:mm a')}
                </p>
              </div>
            </div>
          ))}
          {(!recent?.data || recent.data.length === 0) && (
            <div className="py-8 text-center text-sm text-apple-secondary">
              No messages yet
            </div>
          )}
        </Card>
      </div>
    </div>
  )
}
