export interface Stats {
  total_messages: number
  today_messages: number
  total_groups: number
  total_urls: number
  latest_digest: DigestRecord | null
}

export interface MessageRecord {
  id: string
  signal_id: string
  sender_id: string
  content: string
  group_id: string | null
  source_uuid: string | null
  is_outgoing: boolean
  view_once: boolean
  has_attachments: boolean
  created_at: string
}

export interface SearchResult {
  id: string
  signal_id: string
  sender_id: string
  content: string
  group_id: string | null
  source_uuid: string | null
  is_outgoing: boolean
  has_attachments: boolean
  similarity?: number
  rank?: number
  created_at: string
}

export interface GroupWithCount {
  id: string
  group_id: string
  name: string
  description: string
  avatar_path: string
  member_count: number
  message_count: number
  created_at: string
  updated_at: string
}

export interface DigestRecord {
  id: string
  title: string
  summary: string
  topics: string[]
  decisions: string[]
  action_items: string[]
  period_start: string
  period_end: string
  group_id: string | null
  llm_provider: string
  llm_model: string
  token_count: number
  created_at: string
}

export interface URLRecord {
  id: string
  message_id: string
  url: string
  domain: string
  title: string
  description: string
  image_url: string
  fetched: boolean
  created_at: string
}

export interface AttachmentRecord {
  id: string
  message_id: string
  signal_attachment_id: string
  content_type: string
  filename: string
  size: number
  local_path: string
  downloaded: boolean
  created_at: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  limit: number
  offset: number
}
