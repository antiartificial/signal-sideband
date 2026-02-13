export interface DailyInsight {
  id: string
  overview: string
  themes: string[]
  quote_content: string
  quote_sender: string
  quote_created_at: string | null
  image_path: string
  created_at: string
}

export interface MediaAnalysis {
  description: string
  text_content: string
  colors: string
  objects: string
  scene: string
  model?: string
  analyzed_at?: string
}

export interface Superlative {
  label: string
  icon: string
  winner: string
  value: string
}

export interface Stats {
  total_messages: number
  today_messages: number
  total_groups: number
  total_urls: number
  latest_digest: DigestRecord | null
  daily_insight: DailyInsight | null
  superlatives: Superlative[] | null
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
  thumbnail_path: string
  analyzed: boolean
  analysis: MediaAnalysis | null
  created_at: string
}

export interface MediaSearchResult extends AttachmentRecord {
  rank: number
}

export interface ContactRecord {
  source_uuid: string
  phone_number: string
  profile_name: string
  alias: string
  sender_id: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  limit: number
  offset: number
}

// Cerebro Knowledge Graph

export interface CerebroConcept {
  id: string
  name: string
  category: 'topic' | 'person' | 'place' | 'media' | 'event' | 'idea'
  description: string
  mention_count: number
  first_seen: string
  last_seen: string
  metadata: Record<string, unknown>
  group_id: string | null
  created_at: string
}

export interface CerebroEdge {
  id: string
  source_id: string
  target_id: string
  relation: string
  weight: number
}

export interface CerebroEnrichment {
  id: string
  concept_id: string
  source: 'perplexity' | 'grok_x' | 'grok_books'
  content: PerplexityEnrichment | GrokXEnrichment | GrokBooksEnrichment | Record<string, unknown>
  expires_at: string | null
  created_at: string
}

export interface PerplexityEnrichment {
  summary: string
  related_topics: string[]
  key_facts: string[]
  suggested_exploration: string[]
}

export interface GrokXEnrichment {
  trending_posts: { summary: string; context: string }[]
  sentiment: string
  trending_score: string
}

export interface GrokBooksEnrichment {
  books: { title: string; author: string; relevance: string }[]
  articles: { title: string; source: string; relevance: string }[]
}

export interface CerebroGraph {
  concepts: CerebroConcept[]
  edges: CerebroEdge[]
}

export interface CerebroConceptDetail extends CerebroConcept {
  edges: CerebroEdge[]
  enrichments: CerebroEnrichment[]
}

export interface CerebroExtraction {
  id: string
  batch_start: string
  batch_end: string
  message_count: number
  concept_count: number
  edge_count: number
  llm_provider: string
  llm_model: string
  token_count: number
  created_at: string
}
