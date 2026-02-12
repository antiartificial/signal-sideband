-- 002_expand_schema.sql
-- Expanded schema for newsletter platform

-- Groups table
CREATE TABLE IF NOT EXISTS groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id text UNIQUE NOT NULL,
    name text,
    description text,
    avatar_path text,
    member_count int DEFAULT 0,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- Contacts table
CREATE TABLE IF NOT EXISTS contacts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_uuid text UNIQUE NOT NULL,
    phone_number text,
    profile_name text,
    avatar_path text,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now()
);

-- Expand messages table
ALTER TABLE messages ADD COLUMN IF NOT EXISTS group_id text;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS source_uuid text;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_outgoing boolean DEFAULT false;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS view_once boolean DEFAULT false;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS has_attachments boolean DEFAULT false;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS raw_json jsonb;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS tsv tsvector;

-- Indexes on messages
CREATE INDEX IF NOT EXISTS idx_messages_group_id ON messages(group_id);
CREATE INDEX IF NOT EXISTS idx_messages_source_uuid ON messages(source_uuid);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_tsv ON messages USING GIN(tsv);

-- Auto-update tsvector trigger
CREATE OR REPLACE FUNCTION messages_tsv_trigger() RETURNS trigger AS $$
BEGIN
    NEW.tsv := to_tsvector('english', COALESCE(NEW.content, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_messages_tsv ON messages;
CREATE TRIGGER trg_messages_tsv
    BEFORE INSERT OR UPDATE OF content ON messages
    FOR EACH ROW EXECUTE FUNCTION messages_tsv_trigger();

-- Backfill existing rows
UPDATE messages SET tsv = to_tsvector('english', COALESCE(content, '')) WHERE tsv IS NULL;

-- Attachments table
CREATE TABLE IF NOT EXISTS attachments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id uuid REFERENCES messages(id) ON DELETE CASCADE,
    signal_attachment_id text,
    content_type text,
    filename text,
    size bigint,
    local_path text,
    downloaded boolean DEFAULT false,
    created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);

-- URLs table
CREATE TABLE IF NOT EXISTS urls (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id uuid REFERENCES messages(id) ON DELETE CASCADE,
    url text NOT NULL,
    domain text,
    title text,
    description text,
    image_url text,
    fetched boolean DEFAULT false,
    created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_urls_message_id ON urls(message_id);
CREATE INDEX IF NOT EXISTS idx_urls_domain ON urls(domain);

-- Digests table
CREATE TABLE IF NOT EXISTS digests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title text,
    summary text,
    topics jsonb,
    decisions jsonb,
    action_items jsonb,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    group_id text,
    llm_provider text,
    llm_model text,
    token_count int,
    created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_digests_period ON digests(period_start, period_end);

-- Full-text search function
CREATE OR REPLACE FUNCTION search_messages_fulltext(
    query text,
    max_results int DEFAULT 50
)
RETURNS TABLE (
    id uuid,
    signal_id text,
    sender_id text,
    content text,
    group_id text,
    source_uuid text,
    is_outgoing boolean,
    has_attachments boolean,
    rank real,
    created_at timestamptz
)
LANGUAGE sql STABLE
AS $$
    SELECT
        m.id,
        m.signal_id,
        m.sender_id,
        m.content,
        m.group_id,
        m.source_uuid,
        m.is_outgoing,
        m.has_attachments,
        ts_rank(m.tsv, plainto_tsquery('english', query)) AS rank,
        m.created_at
    FROM messages m
    WHERE m.tsv @@ plainto_tsquery('english', query)
    AND (m.expires_at IS NULL OR m.expires_at > now())
    ORDER BY rank DESC
    LIMIT max_results;
$$;

-- Update match_messages to return new columns
CREATE OR REPLACE FUNCTION match_messages(
    query_embedding vector(1536),
    match_threshold float,
    match_count int
)
RETURNS TABLE (
    id uuid,
    signal_id text,
    sender_id text,
    content text,
    group_id text,
    source_uuid text,
    is_outgoing boolean,
    has_attachments boolean,
    similarity float,
    created_at timestamptz
)
LANGUAGE sql STABLE
AS $$
    SELECT
        m.id,
        m.signal_id,
        m.sender_id,
        m.content,
        m.group_id,
        m.source_uuid,
        m.is_outgoing,
        m.has_attachments,
        1 - (m.embedding <=> query_embedding) AS similarity,
        m.created_at
    FROM messages m
    WHERE 1 - (m.embedding <=> query_embedding) > match_threshold
    AND (m.expires_at IS NULL OR m.expires_at > now())
    ORDER BY similarity DESC
    LIMIT match_count;
$$;
