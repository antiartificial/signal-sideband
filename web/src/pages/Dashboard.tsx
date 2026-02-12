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

  const statCards = [
    { label: 'Total Messages', value: stats?.total_messages ?? 0 },
    { label: 'Today', value: stats?.today_messages ?? 0 },
    { label: 'Groups', value: stats?.total_groups ?? 0 },
    { label: 'Links Collected', value: stats?.total_urls ?? 0 },
  ]

  return (
    <div>
      <h2 className="text-2xl font-semibold tracking-tight mb-6">Dashboard</h2>

      {/* Stats grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        {statCards.map(s => (
          <Card key={s.label} className="p-5">
            <p className="text-sm text-apple-secondary mb-1">{s.label}</p>
            <p className="text-3xl font-semibold tracking-tight">{s.value.toLocaleString()}</p>
          </Card>
        ))}
      </div>

      {/* Latest digest */}
      {stats?.latest_digest && (
        <div className="mb-8">
          <h3 className="text-lg font-medium mb-3">Latest Digest</h3>
          <Card
            className="p-6"
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
        <h3 className="text-lg font-medium mb-3">Recent Messages</h3>
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
