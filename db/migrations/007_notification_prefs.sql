-- +goose Up
ALTER TABLE manga_series
    ADD COLUMN IF NOT EXISTS notification_prefs JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE manga_series DROP COLUMN IF EXISTS notification_prefs;
