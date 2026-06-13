-- +goose Up
CREATE INDEX IF NOT EXISTS idx_manga_series_is_active ON manga_series(is_active);
CREATE INDEX IF NOT EXISTS idx_chapters_is_new_release_date ON chapters(is_new, release_date DESC);
CREATE INDEX IF NOT EXISTS idx_chapters_series_id ON chapters(series_id);
CREATE INDEX IF NOT EXISTS idx_notification_logs_created_at ON notification_logs(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_manga_series_is_active;
DROP INDEX IF EXISTS idx_chapters_is_new_release_date;
DROP INDEX IF EXISTS idx_chapters_series_id;
DROP INDEX IF EXISTS idx_notification_logs_created_at;
