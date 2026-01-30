-- Enable the vector extension
create extension if not exists vector;

-- Create the messages table
create table if not exists messages (
  id uuid primary key default gen_random_uuid(),
  signal_id text unique,        -- Signal's internal message ID
  sender_id text,
  content text,
  embedding vector(1536),       -- For OpenAI ada-002 or similar
  expires_at timestamptz,       -- Respecting disappearing messages
  created_at timestamptz default now()
);

-- Index for faster vector similarity search (IVFFlat is standard, but HNSW is better for recall)
-- Ensure you have enough data before creating this, or create it empty.
-- create index on messages using ivfflat (embedding vector_cosine_ops)
-- with (lists = 100);

-- RPC function for semantic search
create or replace function match_messages (
  query_embedding vector(1536),
  match_threshold float,
  match_count int
)
returns table (
  content text,
  similarity float,
  created_at timestamptz
)
language sql stable
as $$
  select
    messages.content,
    1 - (messages.embedding <=> query_embedding) as similarity,
    messages.created_at
  from messages
  where 1 - (messages.embedding <=> query_embedding) > match_threshold
  and (expires_at is null or expires_at > now()) -- Don't return expired messages
  order by similarity desc
  limit match_count;
$$;
