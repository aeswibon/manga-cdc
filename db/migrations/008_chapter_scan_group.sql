-- +goose Up
ALTER TABLE chapters ADD COLUMN scan_group VARCHAR(255);

-- +goose Down
ALTER TABLE chapters DROP COLUMN IF EXISTS scan_group;
