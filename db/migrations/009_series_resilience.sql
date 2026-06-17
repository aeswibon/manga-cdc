-- +goose Up
ALTER TABLE manga_series ADD COLUMN IF NOT EXISTS fallback_sources JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE manga_series ADD COLUMN IF NOT EXISTS last_stale_alert_at TIMESTAMPTZ;
ALTER TABLE manga_series ADD COLUMN IF NOT EXISTS last_status_alert_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE manga_series DROP COLUMN IF EXISTS last_status_alert_at;
ALTER TABLE manga_series DROP COLUMN IF EXISTS last_stale_alert_at;
ALTER TABLE manga_series DROP COLUMN IF EXISTS fallback_sources;
