-- +goose Up
-- +goose StatementBegin
CREATE TABLE source_credentials (
    source VARCHAR(50) PRIMARY KEY,
    encrypted_payload BYTEA NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE source_credentials;
-- +goose StatementEnd
