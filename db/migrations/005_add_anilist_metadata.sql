-- +goose Up
-- +goose StatementBegin
ALTER TABLE manga_series ADD COLUMN anilist_id INT;
ALTER TABLE manga_series ADD COLUMN mal_id INT;
ALTER TABLE manga_series ADD COLUMN canonical_title VARCHAR(500);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE manga_series DROP COLUMN canonical_title;
ALTER TABLE manga_series DROP COLUMN mal_id;
ALTER TABLE manga_series DROP COLUMN anilist_id;
-- +goose StatementEnd
