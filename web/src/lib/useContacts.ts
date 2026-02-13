import { useQuery } from '@tanstack/react-query'
import { useMemo, useCallback } from 'react'
import { getContacts } from './api.ts'

export function useContacts() {
  const { data: contacts, refetch } = useQuery({
    queryKey: ['contacts'],
    queryFn: getContacts,
    staleTime: 5 * 60 * 1000,
  })

  const lookup = useMemo(() => {
    const map: Record<string, string> = {}
    if (!contacts) return map
    for (const c of contacts) {
      const name = c.alias || c.profile_name || ''
      if (!name) continue
      if (c.source_uuid) map[c.source_uuid] = name
      if (c.sender_id) map[c.sender_id] = name
    }
    return map
  }, [contacts])

  const resolveName = useCallback(
    (id: string): string => lookup[id] || id,
    [lookup],
  )

  return { contacts: contacts ?? [], resolveName, refetch }
}
