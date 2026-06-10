CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE manga_series (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(500) NOT NULL,
    alt_titles JSONB,
    author VARCHAR(255),
    artist VARCHAR(255),
    description TEXT,
    cover_url TEXT,
    status VARCHAR(20) CHECK (status IN ('ONGOING', 'COMPLETED', 'HIATUS', 'CANCELLED')),
    source_url TEXT NOT NULL,
    latest_chapter DECIMAL(10,1),
    last_checked TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE chapters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id UUID NOT NULL REFERENCES manga_series(id) ON DELETE CASCADE,
    chapter_num DECIMAL(10,1) NOT NULL,
    title VARCHAR(500),
    url TEXT NOT NULL,
    release_date TIMESTAMPTZ,
    is_new BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(series_id, chapter_num)
);

CREATE TABLE notification_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id UUID NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'SENT', 'FAILED')),
    channel VARCHAR(50) NOT NULL,
    error_message TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_chapters_series_id ON chapters(series_id);
CREATE INDEX idx_chapters_is_new ON chapters(is_new);
CREATE INDEX idx_notification_log_status ON notification_log(status);
