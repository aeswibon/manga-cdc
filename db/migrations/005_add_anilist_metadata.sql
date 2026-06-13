-- +goose Up
-- +goose StatementBegin
ALTER TABLE manga_series ADD COLUMN IF NOT EXISTS anilist_id INT;
ALTER TABLE manga_series ADD COLUMN IF NOT EXISTS mal_id INT;
ALTER TABLE manga_series ADD COLUMN IF NOT EXISTS canonical_title VARCHAR(500);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE manga_series DROP COLUMN canonical_title;
ALTER TABLE manga_series DROP COLUMN mal_id;
ALTER TABLE manga_series DROP COLUMN anilist_id;
-- +goose StatementEnd
