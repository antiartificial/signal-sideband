-- 005_daily_superlatives.sql
-- Cache superlatives in daily insights

ALTER TABLE daily_insights ADD COLUMN IF NOT EXISTS superlatives jsonb;
