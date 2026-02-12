import type { Stats, PaginatedResponse, MessageRecord, SearchResult, GroupWithCount, DigestRecord, URLRecord, AttachmentRecord } from './types.ts'

const BASE = '/api'
const TOKEN_KEY = 'auth_token'

export function setAuthToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearAuthToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export function isAuthenticated(): boolean {
  return !!localStorage.getItem(TOKEN_KEY)
}

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const token = localStorage.getItem(TOKEN_KEY)
  const headers = new Headers(init?.headers)
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(url, { ...init, headers })
  if (res.status === 401) {
    clearAuthToken()
    window.location.reload()
    throw new Error('unauthorized')
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || res.statusText)
  }
  return res.json()
}

export async function getAuthStatus(): Promise<{ required: boolean }> {
  const res = await fetch(`${BASE}/auth/status`)
  return res.json()
}

export async function login(password: string): Promise<string> {
  const res = await fetch(`${BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || 'login failed')
  }
  const data: { token: string } = await res.json()
  setAuthToken(data.token)
  return data.token
}

export function getStats() {
  return fetchJSON<Stats>(`${BASE}/stats`)
}

export function getMessages(params: Record<string, string> = {}) {
  const qs = new URLSearchParams(params).toString()
  return fetchJSON<PaginatedResponse<MessageRecord>>(`${BASE}/messages?${qs}`)
}

export interface SearchFilters {
  group_id?: string
  sender_id?: string
  after?: string
  before?: string
  has_media?: boolean
}

export function searchMessages(q: string, mode: string = 'fulltext', limit: number = 20, filters: SearchFilters = {}) {
  const params = new URLSearchParams({ q, mode, limit: String(limit) })
  if (filters.group_id) params.set('group_id', filters.group_id)
  if (filters.sender_id) params.set('sender_id', filters.sender_id)
  if (filters.after) params.set('after', filters.after)
  if (filters.before) params.set('before', filters.before)
  if (filters.has_media) params.set('has_media', 'true')
  return fetchJSON<SearchResult[]>(`${BASE}/messages/search?${params}`)
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
