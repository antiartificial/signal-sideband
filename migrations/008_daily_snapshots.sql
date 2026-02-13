-- 008: Add snapshot data to daily insights
ALTER TABLE daily_insights ADD COLUMN IF NOT EXISTS snapshot JSONB DEFAULT '{}'::jsonb;
ALTER TABLE daily_insights ADD COLUMN IF NOT EXISTS snapshot_date DATE;
CREATE INDEX IF NOT EXISTS idx_daily_insights_snapshot_date ON daily_insights (snapshot_date DESC);
