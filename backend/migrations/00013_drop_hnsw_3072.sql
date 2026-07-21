-- +goose Up
-- Safety: drop ANN index if an older 00011 attempt created it (vector dim > 2000).
DROP INDEX IF EXISTS idx_doc_chunks_embedding_hnsw;

-- +goose Down
SELECT 1;
