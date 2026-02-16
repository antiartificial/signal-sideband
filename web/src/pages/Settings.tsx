import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { getContacts, updateContactAlias, getVersion } from '../lib/api.ts'
import Card from '../components/Card.tsx'
import LoadingSpinner from '../components/LoadingSpinner.tsx'

export default function Settings() {
  const queryClient = useQueryClient()
  const { data: contacts, isLoading } = useQuery({
    queryKey: ['contacts'],
    queryFn: getContacts,
  })
  const { data: versionInfo } = useQuery({
    queryKey: ['version'],
    queryFn: getVersion,
    staleTime: Infinity,
  })

  const [editingUUID, setEditingUUID] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [saving, setSaving] = useState(false)

  const startEdit = (uuid: string, currentAlias: string) => {
    setEditingUUID(uuid)
    setEditValue(currentAlias)
  }

  const saveAlias = async (uuid: string) => {
    setSaving(true)
    try {
      await updateContactAlias(uuid, editValue)
      queryClient.invalidateQueries({ queryKey: ['contacts'] })
      setEditingUUID(null)
    } catch {
      // keep editing on failure
    } finally {
      setSaving(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent, uuid: string) => {
    if (e.key === 'Enter') saveAlias(uuid)
    if (e.key === 'Escape') setEditingUUID(null)
  }

  if (isLoading) return <LoadingSpinner />

  // Sort by message count descending (most active first)
  const sorted = [...(contacts ?? [])].sort((a, b) => (b.message_count ?? 0) - (a.message_count ?? 0))

  return (
    <div>
      <h2 className="text-2xl font-semibold tracking-tight mb-6">Settings</h2>

      <h3 className="text-lg font-medium mb-3">
        <i className="fawsb fa-address-book text-apple-secondary mr-2" />
        Group Members
      </h3>
      <p className="text-sm text-apple-secondary mb-4">
        Set an alias to override how a member's name appears throughout the dashboard.
      </p>

      <Card className="overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-apple-border text-left text-apple-secondary">
              <th className="px-4 py-3 font-medium">Member</th>
              <th className="px-4 py-3 font-medium">Alias</th>
              <th className="px-4 py-3 font-medium text-right hidden sm:table-cell">Messages</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map(c => {
              const uuid = c.source_uuid || c.sender_id
              const isEditing = editingUUID === uuid
              const displayName = c.profile_name || c.sender_id
              return (
                <tr key={uuid} className="border-b border-apple-border/50 hover:bg-apple-accent-dim transition-colors">
                  <td className="px-4 py-3">
                    <div className="font-medium text-apple-text">{displayName}</div>
                    {c.profile_name && c.sender_id && c.sender_id !== c.profile_name && (
                      <div className="text-[11px] font-mono text-apple-secondary mt-0.5">{c.sender_id}</div>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {isEditing ? (
                      <input
                        autoFocus
                        type="text"
                        value={editValue}
                        onChange={e => setEditValue(e.target.value)}
                        onBlur={() => saveAlias(uuid)}
                        onKeyDown={e => handleKeyDown(e, uuid)}
                        disabled={saving}
                        placeholder={displayName}
                        className="px-2 py-1 w-full rounded-md border border-apple-blue bg-apple-card text-sm
                          focus:outline-none focus:ring-2 focus:ring-apple-blue/30"
                      />
                    ) : (
                      <button
                        onClick={() => startEdit(uuid, c.alias)}
                        className="text-left w-full group"
                      >
                        {c.alias ? (
                          <span className="font-medium text-apple-text">{c.alias}</span>
                        ) : (
                          <span className="text-apple-secondary italic text-xs">set alias...</span>
                        )}
                        <i className="fawsb fa-pen text-[10px] text-apple-secondary opacity-0 group-hover:opacity-100 ml-2 transition-opacity" />
                      </button>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right font-mono text-xs text-apple-secondary hidden sm:table-cell">
                    {(c.message_count ?? 0).toLocaleString()}
                  </td>
                </tr>
              )
            })}
            {sorted.length === 0 && (
              <tr>
                <td colSpan={3} className="px-4 py-8 text-center text-apple-secondary">
                  No members found yet. Messages need to be received first.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {/* Version info */}
      <div className="mt-8 text-xs text-apple-secondary text-center font-mono">
        Signal Sideband {versionInfo?.version ?? '...'}
      </div>
    </div>
  )
}
