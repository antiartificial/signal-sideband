-- 004_daily_image.sql
-- Add image path to daily insights for picture-of-the-day

ALTER TABLE daily_insights ADD COLUMN IF NOT EXISTS image_path text;
