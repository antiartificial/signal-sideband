-- Cerebro: Knowledge Graph

CREATE TABLE IF NOT EXISTS cerebro_concepts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('topic', 'person', 'place', 'media', 'event', 'idea')),
    description TEXT NOT NULL DEFAULT '',
    mention_count INT NOT NULL DEFAULT 1,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB NOT NULL DEFAULT '{}',
    group_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cerebro_concepts_unique
    ON cerebro_concepts (lower(name), coalesce(group_id, '__global__'));

CREATE INDEX IF NOT EXISTS idx_cerebro_concepts_category ON cerebro_concepts (category);
CREATE INDEX IF NOT EXISTS idx_cerebro_concepts_mention_count ON cerebro_concepts (mention_count DESC);
CREATE INDEX IF NOT EXISTS idx_cerebro_concepts_group_id ON cerebro_concepts (group_id);

CREATE TABLE IF NOT EXISTS cerebro_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES cerebro_concepts(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES cerebro_concepts(id) ON DELETE CASCADE,
    relation TEXT NOT NULL,
    weight INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cerebro_edges_unique
    ON cerebro_edges (source_id, target_id, relation);

CREATE INDEX IF NOT EXISTS idx_cerebro_edges_source ON cerebro_edges (source_id);
CREATE INDEX IF NOT EXISTS idx_cerebro_edges_target ON cerebro_edges (target_id);

CREATE TABLE IF NOT EXISTS cerebro_enrichments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    concept_id UUID NOT NULL REFERENCES cerebro_concepts(id) ON DELETE CASCADE,
    source TEXT NOT NULL CHECK (source IN ('perplexity', 'grok_x', 'grok_books')),
    content JSONB NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cerebro_enrichments_concept ON cerebro_enrichments (concept_id);
CREATE INDEX IF NOT EXISTS idx_cerebro_enrichments_expires ON cerebro_enrichments (expires_at);

CREATE TABLE IF NOT EXISTS cerebro_extractions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_start TIMESTAMPTZ NOT NULL,
    batch_end TIMESTAMPTZ NOT NULL,
    message_count INT NOT NULL DEFAULT 0,
    concept_count INT NOT NULL DEFAULT 0,
    edge_count INT NOT NULL DEFAULT 0,
    llm_provider TEXT NOT NULL DEFAULT '',
    llm_model TEXT NOT NULL DEFAULT '',
    token_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
