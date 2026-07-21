-- +goose Up
CREATE TABLE doc_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    doc_source_id UUID NOT NULL REFERENCES doc_sources (id) ON DELETE CASCADE,
    page_number INT NOT NULL,
    s3_page_key TEXT NOT NULL DEFAULT '',
    s3_parsed_key TEXT NOT NULL DEFAULT '',
    text_source TEXT NOT NULL DEFAULT 'local',
    char_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (doc_source_id, page_number),
    CONSTRAINT doc_pages_page_positive CHECK (page_number > 0)
);

CREATE INDEX idx_doc_pages_source ON doc_pages (doc_source_id);

-- +goose Down
DROP TABLE IF EXISTS doc_pages;
