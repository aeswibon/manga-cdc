-- +goose Up
ALTER TABLE chapters ADD COLUMN IF NOT EXISTS scan_group VARCHAR(255);

-- +goose Down
ALTER TABLE chapters DROP COLUMN IF EXISTS scan_group;
