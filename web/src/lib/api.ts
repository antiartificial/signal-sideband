import type { Stats, PaginatedResponse, MessageRecord, SearchResult, GroupWithCount, DigestRecord, URLRecord, AttachmentRecord } from './types.ts'

const BASE = '/api'

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init)
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || res.statusText)
  }
  return res.json()
}

export function getStats() {
  return fetchJSON<Stats>(`${BASE}/stats`)
}

export function getMessages(params: Record<string, string> = {}) {
  const qs = new URLSearchParams(params).toString()
  return fetchJSON<PaginatedResponse<MessageRecord>>(`${BASE}/messages?${qs}`)
}

export function searchMessages(q: string, mode: string = 'fulltext', limit: number = 20) {
  return fetchJSON<SearchResult[]>(`${BASE}/messages/search?q=${encodeURIComponent(q)}&mode=${mode}&limit=${limit}`)
}

export function getGroups() {
  return fetchJSON<GroupWithCount[]>(`${BASE}/groups`)
}

export function getDigests(params: Record<string, string> = {}) {
  const qs = new URLSearchParams(params).toString()
  return fetchJSON<PaginatedResponse<DigestRecord>>(`${BASE}/digests?${qs}`)
}

export function getDigest(id: string) {
  return fetchJSON<DigestRecord>(`${BASE}/digests/${id}`)
}

export function generateDigest(periodStart: string, periodEnd: string, groupId?: string) {
  return fetchJSON<DigestRecord>(`${BASE}/digests/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ period_start: periodStart, period_end: periodEnd, group_id: groupId }),
  })
}

export function getURLs(params: Record<string, string> = {}) {
  const qs = new URLSearchParams(params).toString()
  return fetchJSON<PaginatedResponse<URLRecord>>(`${BASE}/urls?${qs}`)
}

export function getMedia(params: Record<string, string> = {}) {
  const qs = new URLSearchParams(params).toString()
  return fetchJSON<PaginatedResponse<AttachmentRecord>>(`${BASE}/media?${qs}`)
}

export function mediaURL(id: string) {
  return `${BASE}/media/${id}`
}
