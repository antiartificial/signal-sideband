package store

import (
	"encoding/json"
	"time"
)

type MessageRecord struct {
	ID             string          `db:"id"`
	SignalID       string          `db:"signal_id"`
	SenderID       string          `db:"sender_id"`
	Content        string          `db:"content"`
	Embedding      []float32       `db:"embedding"`
	ExpiresAt      *time.Time      `db:"expires_at"`
	GroupID        *string         `db:"group_id"`
	SourceUUID     *string         `db:"source_uuid"`
	IsOutgoing     bool            `db:"is_outgoing"`
	ViewOnce       bool            `db:"view_once"`
	HasAttachments bool            `db:"has_attachments"`
	RawJSON        json.RawMessage `db:"raw_json"`
	CreatedAt      time.Time       `db:"created_at"`
}

type GroupRecord struct {
	ID          string    `db:"id"`
	GroupID     string    `db:"group_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	AvatarPath  string    `db:"avatar_path"`
	MemberCount int       `db:"member_count"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type ContactRecord struct {
	ID          string    `db:"id"`
	SourceUUID  string    `db:"source_uuid"`
	PhoneNumber string    `db:"phone_number"`
	ProfileName string    `db:"profile_name"`
	AvatarPath  string    `db:"avatar_path"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type AttachmentRecord struct {
	ID                 string    `db:"id"`
	MessageID          string    `db:"message_id"`
	SignalAttachmentID string    `db:"signal_attachment_id"`
	ContentType        string    `db:"content_type"`
	Filename           string    `db:"filename"`
	Size               int64     `db:"size"`
	LocalPath          string    `db:"local_path"`
	Downloaded         bool      `db:"downloaded"`
	CreatedAt          time.Time `db:"created_at"`
}

type URLRecord struct {
	ID          string    `db:"id"`
	MessageID   string    `db:"message_id"`
	URL         string    `db:"url"`
	Domain      string    `db:"domain"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	ImageURL    string    `db:"image_url"`
	Fetched     bool      `db:"fetched"`
	CreatedAt   time.Time `db:"created_at"`
}

type DigestRecord struct {
	ID          string          `db:"id"`
	Title       string          `db:"title"`
	Summary     string          `db:"summary"`
	Topics      json.RawMessage `db:"topics"`
	Decisions   json.RawMessage `db:"decisions"`
	ActionItems json.RawMessage `db:"action_items"`
	PeriodStart time.Time       `db:"period_start"`
	PeriodEnd   time.Time       `db:"period_end"`
	GroupID     *string         `db:"group_id"`
	LLMProvider string          `db:"llm_provider"`
	LLMModel    string          `db:"llm_model"`
	TokenCount  int             `db:"token_count"`
	CreatedAt   time.Time       `db:"created_at"`
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

type Stats struct {
	TotalMessages int    `json:"total_messages"`
	TodayMessages int    `json:"today_messages"`
	TotalGroups   int    `json:"total_groups"`
	TotalURLs     int    `json:"total_urls"`
	LatestDigest  *DigestRecord `json:"latest_digest,omitempty"`
}
