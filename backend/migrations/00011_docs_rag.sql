-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manufacturer_org_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    kind product_kind NOT NULL DEFAULT 'other',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (manufacturer_org_id, slug)
);

CREATE INDEX idx_products_manufacturer ON products (manufacturer_org_id);

CREATE TYPE doc_source_status AS ENUM (
    'pending',
    'extracting',
    'embedding',
    'ready',
    'failed',
    'skipped_duplicate'
);

CREATE TABLE doc_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    manufacturer_org_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    version TEXT NOT NULL DEFAULT '',
    filename TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT 'application/pdf',
    byte_size BIGINT NOT NULL DEFAULT 0,
    s3_original_key TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    status doc_source_status NOT NULL DEFAULT 'pending',
    page_count INT NOT NULL DEFAULT 0,
    chunk_count INT NOT NULL DEFAULT 0,
    embedding_model TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    tokens_total BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    indexed_at TIMESTAMPTZ
);

CREATE INDEX idx_doc_sources_product ON doc_sources (product_id);
CREATE INDEX idx_doc_sources_manufacturer ON doc_sources (manufacturer_org_id);
CREATE INDEX idx_doc_sources_hash ON doc_sources (manufacturer_org_id, content_hash);
CREATE INDEX idx_doc_sources_status ON doc_sources (status);

CREATE TABLE doc_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    doc_source_id UUID NOT NULL REFERENCES doc_sources (id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    manufacturer_org_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    version TEXT NOT NULL DEFAULT '',
    page_number INT NOT NULL,
    chunk_index INT NOT NULL DEFAULT 0,
    section TEXT NOT NULL DEFAULT '',
    content_md TEXT NOT NULL,
    s3_page_key TEXT NOT NULL DEFAULT '',
    embedding vector(3072),
    token_estimate INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT doc_chunks_page_positive CHECK (page_number > 0)
);

CREATE INDEX idx_doc_chunks_source ON doc_chunks (doc_source_id);
CREATE INDEX idx_doc_chunks_product_version ON doc_chunks (product_id, version);
CREATE INDEX idx_doc_chunks_manufacturer ON doc_chunks (manufacturer_org_id);
CREATE INDEX idx_doc_chunks_content_fts ON doc_chunks USING GIN (to_tsvector('russian', content_md));

-- HNSW works with high-dimensional embeddings (3072); build after first data is fine.
CREATE INDEX idx_doc_chunks_embedding_hnsw ON doc_chunks USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64)
    WHERE embedding IS NOT NULL;

CREATE TABLE usage_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations (id) ON DELETE SET NULL,
    operation TEXT NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    tokens_in BIGINT NOT NULL DEFAULT 0,
    tokens_out BIGINT NOT NULL DEFAULT 0,
    tokens_total BIGINT NOT NULL DEFAULT 0,
    doc_source_id UUID REFERENCES doc_sources (id) ON DELETE SET NULL,
    ticket_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_usage_log_org_created ON usage_log (org_id, created_at DESC);
CREATE INDEX idx_usage_log_operation ON usage_log (operation, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS usage_log;
DROP INDEX IF EXISTS idx_doc_chunks_embedding_hnsw;
DROP INDEX IF EXISTS idx_doc_chunks_content_fts;
DROP TABLE IF EXISTS doc_chunks;
DROP TABLE IF EXISTS doc_sources;
DROP TYPE IF EXISTS doc_source_status;
DROP TABLE IF EXISTS products;
