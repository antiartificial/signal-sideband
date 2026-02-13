import { useState, useMemo } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { getStats, getMessages, getSnapshots, generateDigest, generateInsight } from '../lib/api.ts'
import { useContacts } from '../lib/useContacts.ts'
import type { DailyInsight, DaySnapshot } from '../lib/types.ts'
import Card from '../components/Card.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import { format, subDays, isSameDay, parseISO, isSunday } from 'date-fns'

const HOUR_LABELS = [
  '12am','1am','2am','3am','4am','5am','6am','7am','8am','9am','10am','11am',
  '12pm','1pm','2pm','3pm','4pm','5pm','6pm','7pm','8pm','9pm','10pm','11pm',
]

const CREW_META: Record<string, { icon: string; label: string }> = {
  morning:   { icon: 'fa-sun',         label: 'Morning' },
  afternoon: { icon: 'fa-cloud-sun',   label: 'Afternoon' },
  evening:   { icon: 'fa-sunset',      label: 'Evening' },
  night:     { icon: 'fa-moon-stars',  label: 'Night' },
}

export default function Dashboard() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { resolveName } = useContacts()
  const { data: stats, isLoading } = useQuery({ queryKey: ['stats'], queryFn: getStats })
  const { data: recent } = useQuery({
    queryKey: ['messages', 'recent'],
    queryFn: () => getMessages({ limit: '5' }),
  })
  const { data: snapshots } = useQuery({
    queryKey: ['snapshots'],
    queryFn: () => getSnapshots(7),
  })

  const [selectedDate, setSelectedDate] = useState<string>(format(new Date(), 'yyyy-MM-dd'))
  const [generating, setGenerating] = useState(false)
  const [activeLens, setActiveLens] = useState<string | null>(null)
  const [generatingInsight, setGeneratingInsight] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Build the 7-day picker dates
  const dayPills = useMemo(() => {
    const days: Date[] = []
    const today = new Date()
    for (let i = 6; i >= 0; i--) {
      days.push(subDays(today, i))
    }
    return days
  }, [])

  // Find snapshot for selected date
  const selectedSnapshot = useMemo(() => {
    if (!snapshots) return null
    return snapshots.find(s =>
      s.snapshot_date && isSameDay(parseISO(s.snapshot_date), parseISO(selectedDate))
    ) ?? null
  }, [snapshots, selectedDate])

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
      queryClient.invalidateQueries({ queryKey: ['snapshots'] })
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

      {/* Day picker strip */}
      <div className="flex gap-2 mb-6 overflow-x-auto pb-1">
        {dayPills.map(day => {
          const dateStr = format(day, 'yyyy-MM-dd')
          const isToday = isSameDay(day, new Date())
          const isSelected = dateStr === selectedDate
          const isSun = isSunday(day)
          const hasSnapshot = snapshots?.some(s =>
            s.snapshot_date && isSameDay(parseISO(s.snapshot_date), day)
          )

          return (
            <button
              key={dateStr}
              onClick={() => setSelectedDate(dateStr)}
              className={`flex flex-col items-center px-3 py-2 rounded-xl text-xs font-medium transition-all
                shrink-0 min-w-[56px]
                ${isSelected
                  ? 'bg-apple-blue text-white shadow-sm'
                  : isToday
                    ? 'bg-apple-blue/10 text-apple-blue border border-apple-blue/30'
                    : 'bg-apple-card border border-apple-border text-apple-text hover:border-apple-blue/40'
                }`}
            >
              <span className="text-[10px] uppercase opacity-70">{format(day, 'EEE')}</span>
              <span className="text-base font-semibold">{format(day, 'd')}</span>
              {isSun && <i className="fawsb fa-crown text-[10px] mt-0.5 opacity-60" />}
              {hasSnapshot && !isSelected && (
                <span className="w-1.5 h-1.5 rounded-full bg-apple-blue mt-0.5" />
              )}
            </button>
          )
        })}
      </div>

      {/* Snapshot card for selected day */}
      {selectedSnapshot ? (
        <SnapshotCard
          snapshot={selectedSnapshot}
          resolveName={resolveName}
          onGenerateInsight={handleGenerateInsight}
          generatingInsight={generatingInsight}
        />
      ) : (
        <div className="mb-8">
          <Card className="p-8 text-center">
            <i className="fawsb fa-sparkles text-3xl text-apple-secondary mb-3 block" />
            <p className="text-sm text-apple-secondary mb-1">
              No snapshot for {format(parseISO(selectedDate), 'EEEE, MMMM d')}
            </p>
            {isSameDay(parseISO(selectedDate), new Date()) && (
              <>
                <p className="text-xs text-apple-secondary mb-4">Generate one to see today's overview, themes, and stats.</p>
                <button
                  onClick={handleGenerateInsight}
                  disabled={generatingInsight}
                  className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-apple-blue text-white text-sm font-medium
                    hover:bg-apple-blue/90 active:scale-95 transition-all duration-200 disabled:opacity-50"
                >
                  <i className={`fawsb ${generatingInsight ? 'fa-sparkles animate-pulse' : 'fa-wand-magic-sparkles'}`} />
                  {generatingInsight ? 'Generating...' : 'Generate Daily Insight'}
                </button>
              </>
            )}
          </Card>
        </div>
      )}

      {/* Digest lenses */}
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

      {/* Recent messages */}
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
                  {msg.has_attachments && (
                    <div className={`flex items-center gap-1 mt-1 text-xs ${
                      msg.is_outgoing ? 'text-white/70' : 'text-apple-secondary'
                    }`}>
                      <i className="fawsb fa-paperclip text-[10px]" />
                      <span>attachment</span>
                    </div>
                  )}
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

function SnapshotCard({
  snapshot: insight,
  resolveName,
}: {
  snapshot: DailyInsight
  resolveName: (id: string) => string
  onGenerateInsight: () => void
  generatingInsight: boolean
}) {
  const snap: DaySnapshot | null = insight.snapshot && typeof insight.snapshot === 'object' && 'message_count' in insight.snapshot
    ? insight.snapshot as DaySnapshot
    : null

  const dateLabel = insight.snapshot_date
    ? format(parseISO(insight.snapshot_date), 'EEEE, MMMM d')
    : 'Today'

  return (
    <div className="mb-8 space-y-4">
      {/* Header + Overview */}
      <Card className="p-6">
        <div className="flex items-center gap-2 mb-3">
          <i className="fawsb fa-sparkles text-apple-blue" />
          <h3 className="text-lg font-medium">{dateLabel}</h3>
          {snap?.is_weekly && (
            <span className="ml-auto px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-500/15 text-amber-700 dark:text-amber-300 text-xs font-medium">
              <i className="fawsb fa-crown mr-1" />
              Weekly Edition
            </span>
          )}
        </div>
        {insight.overview && (
          <p className="text-sm text-apple-secondary leading-relaxed">{insight.overview}</p>
        )}

        {/* Theme pills */}
        {insight.themes && insight.themes.length > 0 && (
          <div className="flex flex-wrap gap-2 mt-3">
            {insight.themes.map((theme: string, i: number) => {
              const colors = [
                'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300',
                'bg-purple-100 text-purple-700 dark:bg-purple-500/15 dark:text-purple-300',
                'bg-green-100 text-green-700 dark:bg-green-500/15 dark:text-green-300',
                'bg-orange-100 text-orange-700 dark:bg-orange-500/15 dark:text-orange-300',
                'bg-pink-100 text-pink-700 dark:bg-pink-500/15 dark:text-pink-300',
              ]
              return (
                <span key={i} className={`px-3 py-1 rounded-full text-xs font-medium ${colors[i % colors.length]}`}>
                  {theme}
                </span>
              )
            })}
          </div>
        )}

        {/* Stats row */}
        {snap && snap.message_count > 0 && (
          <div className="flex items-center gap-4 mt-4 text-xs text-apple-secondary">
            <span><i className="fawsb fa-chart-bar mr-1" />{snap.message_count} messages</span>
            <span><i className="fawsb fa-users mr-1" />{snap.active_senders} people</span>
            <span><i className="fawsb fa-clock mr-1" />Peak: {HOUR_LABELS[snap.busiest_hour]}</span>
          </div>
        )}
      </Card>

      {/* Crews */}
      {snap && snap.crews && Object.keys(snap.crews).length > 0 && (
        <Card className="p-6">
          <div className="flex items-center gap-2 mb-3">
            <i className="fawsb fa-people-group text-apple-blue" />
            <h3 className="text-lg font-medium">The Crews</h3>
          </div>
          <div className="grid grid-cols-2 gap-4">
            {(['morning', 'afternoon', 'evening', 'night'] as const).map(period => {
              const meta = CREW_META[period]
              const members = snap.crews[period]
              if (!members || members.length === 0) return null
              return (
                <div key={period}>
                  <div className="flex items-center gap-1.5 mb-2">
                    <i className={`fawsb ${meta.icon} text-apple-secondary text-xs`} />
                    <span className="text-xs font-medium text-apple-secondary">{meta.label}</span>
                  </div>
                  <div className="space-y-1">
                    {members.slice(0, 3).map((m, i) => (
                      <div key={i} className="flex items-center justify-between text-xs">
                        <span className="text-apple-text truncate">{resolveName(m.sender_id)}</span>
                        <span className="text-apple-secondary font-mono ml-2">{m.count}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )
            })}
          </div>
        </Card>
      )}

      {/* Top Conversations */}
      {snap && snap.top_pairs && snap.top_pairs.length > 0 && (
        <Card className="p-6">
          <div className="flex items-center gap-2 mb-3">
            <i className="fawsb fa-comments text-apple-blue" />
            <h3 className="text-lg font-medium">Top Conversations</h3>
          </div>
          <div className="space-y-2">
            {snap.top_pairs.map((pair, i) => (
              <div key={i} className="flex items-center justify-between text-sm">
                <span className="text-apple-text">
                  {resolveName(pair.sender_a)} <span className="text-apple-secondary mx-1">&harr;</span> {resolveName(pair.sender_b)}
                </span>
                <span className="text-xs text-apple-secondary font-mono">{pair.count} exchanges</span>
              </div>
            ))}
          </div>
        </Card>
      )}

      {/* Verb Champion */}
      {snap?.verb_leader && (
        <Card className="p-6">
          <div className="flex items-center gap-2 mb-2">
            <i className="fawsb fa-pen-nib text-apple-blue" />
            <h3 className="text-lg font-medium">Verb Champion</h3>
          </div>
          <p className="text-sm text-apple-text">
            <span className="font-medium">{resolveName(snap.verb_leader.sender_id)}</span>
            <span className="text-apple-secondary"> &mdash; {snap.verb_leader.count} verbs</span>
          </p>
          {snap.verb_leader.samples.length > 0 && (
            <p className="text-xs text-apple-secondary mt-1 italic">
              "{snap.verb_leader.samples.join(', ')}"
            </p>
          )}
        </Card>
      )}

      {/* Link of the Day */}
      {snap?.link_of_day && (
        <Card className="p-6">
          <div className="flex items-center gap-2 mb-2">
            <i className="fawsb fa-link text-apple-blue" />
            <h3 className="text-lg font-medium">Link of the Day</h3>
          </div>
          <a
            href={snap.link_of_day.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-apple-blue hover:underline break-all"
          >
            {snap.link_of_day.title || snap.link_of_day.url}
          </a>
          <p className="text-xs text-apple-secondary mt-1">
            shared by {resolveName(snap.link_of_day.sender_id)}
          </p>
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
              <p className="text-xs text-apple-secondary mt-2">&mdash; {resolveName(insight.quote_sender)}</p>
            )}
          </blockquote>
        </Card>
      )}

      {/* Yesterday's Callback */}
      {snap?.yesterday_ref && (snap.yesterday_ref.quote || snap.yesterday_ref.link) && (
        <Card className="p-6">
          <div className="flex items-center gap-2 mb-2">
            <i className="fawsb fa-clock-rotate-left text-apple-blue" />
            <h3 className="text-lg font-medium">Yesterday's Callback</h3>
          </div>
          {snap.yesterday_ref.quote && (
            <p className="text-sm text-apple-secondary italic mb-2">"{snap.yesterday_ref.quote}"</p>
          )}
          {snap.yesterday_ref.link && (
            <a
              href={snap.yesterday_ref.link.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-apple-blue hover:underline"
            >
              {snap.yesterday_ref.link.title || snap.yesterday_ref.link.url}
            </a>
          )}
        </Card>
      )}

      {/* Sunday Weekly Edition */}
      {snap?.is_weekly && (
        <Card className="p-6 border-amber-200 dark:border-amber-500/30">
          <div className="flex items-center gap-2 mb-3">
            <i className="fawsb fa-trophy text-amber-500" />
            <h3 className="text-lg font-medium">Weekly Recap</h3>
          </div>
          <div className="grid grid-cols-2 gap-4 text-sm">
            {snap.busiest_day && (
              <div>
                <p className="text-xs text-apple-secondary">Busiest Day</p>
                <p className="text-apple-text font-medium">{snap.busiest_day}</p>
                <p className="text-xs text-apple-secondary">{snap.busiest_day_count} messages</p>
              </div>
            )}
            {(snap.weekly_total ?? 0) > 0 && (
              <div>
                <p className="text-xs text-apple-secondary">Weekly Total</p>
                <p className="text-apple-text font-semibold text-xl font-mono">{snap.weekly_total?.toLocaleString()}</p>
                <p className="text-xs text-apple-secondary">messages</p>
              </div>
            )}
          </div>
        </Card>
      )}
    </div>
  )
}
