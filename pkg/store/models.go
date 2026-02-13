package store

import (
	"encoding/json"
	"time"
)

type MessageRecord struct {
	ID             string          `db:"id" json:"id"`
	SignalID       string          `db:"signal_id" json:"signal_id"`
	SenderID       string          `db:"sender_id" json:"sender_id"`
	Content        string          `db:"content" json:"content"`
	Embedding      []float32       `db:"embedding" json:"-"`
	ExpiresAt      *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	GroupID        *string         `db:"group_id" json:"group_id,omitempty"`
	SourceUUID     *string         `db:"source_uuid" json:"source_uuid,omitempty"`
	IsOutgoing     bool            `db:"is_outgoing" json:"is_outgoing"`
	ViewOnce       bool            `db:"view_once" json:"view_once"`
	HasAttachments bool            `db:"has_attachments" json:"has_attachments"`
	RawJSON        json.RawMessage `db:"raw_json" json:"-"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

type GroupRecord struct {
	ID          string    `db:"id" json:"id"`
	GroupID     string    `db:"group_id" json:"group_id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	AvatarPath  string    `db:"avatar_path" json:"avatar_path,omitempty"`
	MemberCount int       `db:"member_count" json:"member_count"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type ContactRecord struct {
	ID          string    `db:"id" json:"id"`
	SourceUUID  string    `db:"source_uuid" json:"source_uuid"`
	PhoneNumber string    `db:"phone_number" json:"phone_number"`
	ProfileName string    `db:"profile_name" json:"profile_name"`
	Alias       string    `db:"alias" json:"alias"`
	AvatarPath  string    `db:"avatar_path" json:"avatar_path,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type DistinctSender struct {
	SenderID   string `json:"sender_id"`
	SourceUUID string `json:"source_uuid"`
}

type AttachmentRecord struct {
	ID                 string          `db:"id" json:"id"`
	MessageID          string          `db:"message_id" json:"message_id"`
	SignalAttachmentID string          `db:"signal_attachment_id" json:"signal_attachment_id"`
	ContentType        string          `db:"content_type" json:"content_type"`
	Filename           string          `db:"filename" json:"filename"`
	Size               int64           `db:"size" json:"size"`
	LocalPath          string          `db:"local_path" json:"local_path,omitempty"`
	Downloaded         bool            `db:"downloaded" json:"downloaded"`
	ThumbnailPath      string          `db:"thumbnail_path" json:"thumbnail_path,omitempty"`
	Analyzed           bool            `db:"analyzed" json:"analyzed"`
	Analysis           json.RawMessage `db:"analysis" json:"analysis,omitempty"`
	CreatedAt          time.Time       `db:"created_at" json:"created_at"`
}

type MediaSearchResult struct {
	AttachmentRecord
	Rank float32 `json:"rank"`
}

type URLRecord struct {
	ID          string    `db:"id" json:"id"`
	MessageID   string    `db:"message_id" json:"message_id"`
	URL         string    `db:"url" json:"url"`
	Domain      string    `db:"domain" json:"domain"`
	Title       string    `db:"title" json:"title"`
	Description string    `db:"description" json:"description"`
	ImageURL    string    `db:"image_url" json:"image_url,omitempty"`
	Fetched     bool      `db:"fetched" json:"fetched"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type DigestRecord struct {
	ID          string          `db:"id" json:"id"`
	Title       string          `db:"title" json:"title"`
	Summary     string          `db:"summary" json:"summary"`
	Topics      json.RawMessage `db:"topics" json:"topics"`
	Decisions   json.RawMessage `db:"decisions" json:"decisions"`
	ActionItems json.RawMessage `db:"action_items" json:"action_items"`
	PeriodStart time.Time       `db:"period_start" json:"period_start"`
	PeriodEnd   time.Time       `db:"period_end" json:"period_end"`
	GroupID     *string         `db:"group_id" json:"group_id,omitempty"`
	LLMProvider string          `db:"llm_provider" json:"llm_provider"`
	LLMModel    string          `db:"llm_model" json:"llm_model"`
	TokenCount  int             `db:"token_count" json:"token_count"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
}

type MessageFilter struct {
	GroupID   *string
	SenderID  *string
	After     *time.Time
	Before    *time.Time
	HasMedia  *bool
	Limit     int
	Offset    int
}

type SearchFilter struct {
	GroupID  *string
	SenderID *string
	After    *time.Time
	Before   *time.Time
	HasMedia *bool
}

type SearchResult struct {
	ID             string    `json:"id"`
	SignalID       string    `json:"signal_id"`
	SenderID       string    `json:"sender_id"`
	Content        string    `json:"content"`
	GroupID        *string   `json:"group_id,omitempty"`
	SourceUUID     *string   `json:"source_uuid,omitempty"`
	IsOutgoing     bool      `json:"is_outgoing"`
	HasAttachments bool      `json:"has_attachments"`
	Similarity     *float64  `json:"similarity,omitempty"`
	Rank           *float32  `json:"rank,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type DailyInsight struct {
	ID             string          `db:"id" json:"id"`
	Overview       string          `db:"overview" json:"overview"`
	Themes         json.RawMessage `db:"themes" json:"themes"`
	QuoteContent   string          `db:"quote_content" json:"quote_content,omitempty"`
	QuoteSender    string          `db:"quote_sender" json:"quote_sender,omitempty"`
	QuoteCreatedAt *time.Time      `db:"quote_created_at" json:"quote_created_at,omitempty"`
	ImagePath      string          `db:"image_path" json:"image_path,omitempty"`
	Superlatives   json.RawMessage `db:"superlatives" json:"superlatives,omitempty"`
	Snapshot       json.RawMessage `db:"snapshot" json:"snapshot,omitempty"`
	SnapshotDate   *time.Time      `db:"snapshot_date" json:"snapshot_date,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

type DaySnapshot struct {
	MessageCount  int                     `json:"message_count"`
	ActiveSenders int                     `json:"active_senders"`
	BusiestHour   int                     `json:"busiest_hour"`
	Crews         map[string][]CrewMember `json:"crews"`
	TopPairs      []ConversationPair      `json:"top_pairs"`
	VerbLeader    *VerbLeader             `json:"verb_leader"`
	LinkOfDay     *LinkOfDay              `json:"link_of_day"`
	YesterdayRef  *YesterdayRef           `json:"yesterday_ref"`
	IsWeekly      bool                    `json:"is_weekly"`
	WeeklyTotal   int                     `json:"weekly_total,omitempty"`
	BusiestDay    string                  `json:"busiest_day,omitempty"`
	BusiestDayCount int                   `json:"busiest_day_count,omitempty"`
}

type CrewMember struct {
	SenderID string `json:"sender_id"`
	Count    int    `json:"count"`
}

type ConversationPair struct {
	SenderA string `json:"sender_a"`
	SenderB string `json:"sender_b"`
	Count   int    `json:"count"`
}

type VerbLeader struct {
	SenderID string   `json:"sender_id"`
	Count    int      `json:"count"`
	Samples  []string `json:"samples"`
}

type LinkOfDay struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	SenderID string `json:"sender_id"`
}

type YesterdayRef struct {
	Quote string     `json:"quote,omitempty"`
	Link  *LinkOfDay `json:"link,omitempty"`
}

type Superlative struct {
	Label  string `json:"label"`
	Icon   string `json:"icon"`
	Winner string `json:"winner"`
	Value  string `json:"value"`
}

type Stats struct {
	TotalMessages int            `json:"total_messages"`
	TodayMessages int            `json:"today_messages"`
	TotalGroups   int            `json:"total_groups"`
	TotalURLs     int            `json:"total_urls"`
	LatestDigest  *DigestRecord  `json:"latest_digest,omitempty"`
	DailyInsight  *DailyInsight  `json:"daily_insight,omitempty"`
	Superlatives  []Superlative  `json:"superlatives,omitempty"`
}

// Cerebro Knowledge Graph

type CerebroConcept struct {
	ID           string          `db:"id" json:"id"`
	Name         string          `db:"name" json:"name"`
	Category     string          `db:"category" json:"category"`
	Description  string          `db:"description" json:"description"`
	MentionCount int             `db:"mention_count" json:"mention_count"`
	FirstSeen    time.Time       `db:"first_seen" json:"first_seen"`
	LastSeen     time.Time       `db:"last_seen" json:"last_seen"`
	Metadata     json.RawMessage `db:"metadata" json:"metadata"`
	GroupID      *string         `db:"group_id" json:"group_id,omitempty"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
}

type CerebroEdge struct {
	ID       string `db:"id" json:"id"`
	SourceID string `db:"source_id" json:"source_id"`
	TargetID string `db:"target_id" json:"target_id"`
	Relation string `db:"relation" json:"relation"`
	Weight   int    `db:"weight" json:"weight"`
}

type CerebroEnrichment struct {
	ID        string          `db:"id" json:"id"`
	ConceptID string          `db:"concept_id" json:"concept_id"`
	Source    string          `db:"source" json:"source"`
	Content   json.RawMessage `db:"content" json:"content"`
	ExpiresAt *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

type CerebroExtraction struct {
	ID           string    `db:"id" json:"id"`
	BatchStart   time.Time `db:"batch_start" json:"batch_start"`
	BatchEnd     time.Time `db:"batch_end" json:"batch_end"`
	MessageCount int       `db:"message_count" json:"message_count"`
	ConceptCount int       `db:"concept_count" json:"concept_count"`
	EdgeCount    int       `db:"edge_count" json:"edge_count"`
	LLMProvider  string    `db:"llm_provider" json:"llm_provider"`
	LLMModel     string    `db:"llm_model" json:"llm_model"`
	TokenCount   int       `db:"token_count" json:"token_count"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type CerebroGraph struct {
	Concepts []CerebroConcept `json:"concepts"`
	Edges    []CerebroEdge    `json:"edges"`
}

type CerebroConceptDetail struct {
	CerebroConcept
	Edges       []CerebroEdge       `json:"edges"`
	Enrichments []CerebroEnrichment `json:"enrichments"`
}
