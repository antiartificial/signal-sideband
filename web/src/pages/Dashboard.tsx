import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { getStats, getMessages } from '../lib/api.ts'
import Card from '../components/Card.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'
import { format } from 'date-fns'

export default function Dashboard() {
  const navigate = useNavigate()
  const { data: stats, isLoading } = useQuery({ queryKey: ['stats'], queryFn: getStats })
  const { data: recent } = useQuery({
    queryKey: ['messages', 'recent'],
    queryFn: () => getMessages({ limit: '5' }),
  })

  if (isLoading) return <LoadingSpinner />

  const insight = stats?.daily_insight

  const statCards = [
    { label: 'Total Messages', value: stats?.total_messages ?? 0, icon: 'fa-messages' },
    { label: 'Today', value: stats?.today_messages ?? 0, icon: 'fa-calendar-day' },
    { label: 'Groups', value: stats?.total_groups ?? 0, icon: 'fa-users' },
    { label: 'Links Collected', value: stats?.total_urls ?? 0, icon: 'fa-link' },
  ]

  return (
    <div>
      <h2 className="text-2xl font-semibold tracking-tight mb-6">Dashboard</h2>

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
      {insight && (
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
                <i className="fawsb fa-tags text-apple-blue" />
                <h3 className="text-lg font-medium">Topics & Themes</h3>
              </div>
              <div className="flex flex-wrap gap-2">
                {insight.themes.map((theme, i) => {
                  const colors = [
                    'bg-blue-100 text-blue-700',
                    'bg-purple-100 text-purple-700',
                    'bg-green-100 text-green-700',
                    'bg-orange-100 text-orange-700',
                    'bg-pink-100 text-pink-700',
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
                <i className="fawsb fa-quote-left text-apple-blue" />
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
      )}

      {/* Latest digest */}
      {stats?.latest_digest && (
        <div className="mb-8">
          <h3 className="text-lg font-medium mb-3">
            <i className="fawsb fa-newspaper text-apple-secondary mr-2" />
            Latest Digest
          </h3>
          <Card
            className="p-6 cursor-pointer hover:shadow-md transition-shadow"
            onClick={() => navigate(`/digests/${stats.latest_digest!.id}`)}
          >
            <h4 className="text-base font-semibold mb-2">{stats.latest_digest.title}</h4>
            <p className="text-sm text-apple-secondary line-clamp-3">
              {stats.latest_digest.summary.slice(0, 200)}...
            </p>
            <p className="text-xs text-apple-secondary mt-3">
              {format(new Date(stats.latest_digest.created_at), 'MMM d, yyyy')}
            </p>
          </Card>
        </div>
      )}

      {/* Recent activity */}
      <div>
        <h3 className="text-lg font-medium mb-3">
          <i className="fawsb fa-clock-rotate-left text-apple-secondary mr-2" />
          Recent Messages
        </h3>
        <Card className="divide-y divide-apple-border">
          {recent?.data.map(msg => (
            <div key={msg.id} className="px-5 py-3.5">
              <div className="flex items-center justify-between mb-1">
                <span className="text-sm font-medium">
                  {msg.is_outgoing ? 'You' : msg.sender_id}
                </span>
                <span className="text-xs text-apple-secondary">
                  {format(new Date(msg.created_at), 'MMM d, h:mm a')}
                </span>
              </div>
              <p className="text-sm text-apple-secondary line-clamp-2">{msg.content}</p>
            </div>
          ))}
          {(!recent?.data || recent.data.length === 0) && (
            <div className="px-5 py-8 text-center text-sm text-apple-secondary">
              No messages yet
            </div>
          )}
        </Card>
      </div>
    </div>
  )
}
