-- +goose Up
CREATE TABLE scraped_rejects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source VARCHAR(50) NOT NULL,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('series', 'chapter')),
    payload JSONB NOT NULL,
    reasons JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_scraped_rejects_source ON scraped_rejects(source);
CREATE INDEX idx_scraped_rejects_created_at ON scraped_rejects(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS scraped_rejects;
