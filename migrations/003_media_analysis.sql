-- 003_media_analysis.sql
-- Add thumbnail, AI analysis columns to attachments + daily_insights table

ALTER TABLE attachments ADD COLUMN IF NOT EXISTS thumbnail_path text;
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS analyzed boolean DEFAULT false;
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS analysis jsonb;
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS analysis_tsv tsvector;

CREATE INDEX IF NOT EXISTS idx_attachments_analyzed ON attachments(analyzed) WHERE analyzed = false;
CREATE INDEX IF NOT EXISTS idx_attachments_analysis_tsv ON attachments USING GIN(analysis_tsv);

-- Auto-maintain tsvector from analysis JSONB fields
CREATE OR REPLACE FUNCTION attachments_analysis_tsv_trigger() RETURNS trigger AS $$
BEGIN
  IF NEW.analysis IS NOT NULL THEN
    NEW.analysis_tsv := to_tsvector('english',
      COALESCE(NEW.analysis->>'description','') || ' ' ||
      COALESCE(NEW.analysis->>'text_content','') || ' ' ||
      COALESCE(NEW.analysis->>'colors','') || ' ' ||
      COALESCE(NEW.analysis->>'objects','') || ' ' ||
      COALESCE(NEW.analysis->>'scene',''));
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

CREATE TRIGGER trg_attachments_analysis_tsv
  BEFORE INSERT OR UPDATE OF analysis ON attachments
  FOR EACH ROW EXECUTE FUNCTION attachments_analysis_tsv_trigger();

-- Daily insights table for dashboard
CREATE TABLE IF NOT EXISTS daily_insights (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  overview text NOT NULL,
  themes jsonb NOT NULL DEFAULT '[]',
  quote_content text,
  quote_sender text,
  quote_created_at timestamptz,
  created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_daily_insights_created ON daily_insights(created_at DESC);
