-- 010_attachment_retry_state.sql
-- Track finite media worker retries so bad media rows do not loop forever.

ALTER TABLE attachments ADD COLUMN IF NOT EXISTS thumbnail_attempts int NOT NULL DEFAULT 0;
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS thumbnail_error text;
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS analysis_attempts int NOT NULL DEFAULT 0;
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS analysis_error text;

CREATE INDEX IF NOT EXISTS idx_attachments_thumbnail_retry
  ON attachments(created_at)
  WHERE downloaded = true
    AND thumbnail_path IS NULL
    AND thumbnail_attempts < 3
    AND (content_type LIKE 'image/%' OR content_type LIKE 'video/%');

CREATE INDEX IF NOT EXISTS idx_attachments_analysis_retry
  ON attachments(created_at)
  WHERE downloaded = true
    AND analyzed = false
    AND analysis_attempts < 3
    AND (content_type LIKE 'image/%' OR content_type LIKE 'video/%');
