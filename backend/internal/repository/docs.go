package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Product struct {
	ID                 uuid.UUID `json:"id"`
	ManufacturerOrgID  uuid.UUID `json:"manufacturer_org_id"`
	Slug               string    `json:"slug"`
	Name               string    `json:"name"`
	Kind               string    `json:"kind"`
	Description        string    `json:"description"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type DocSource struct {
	ID                uuid.UUID  `json:"id"`
	ProductID         uuid.UUID  `json:"product_id"`
	ManufacturerOrgID uuid.UUID  `json:"manufacturer_org_id"`
	Version           string     `json:"version"`
	Filename          string     `json:"filename"`
	MimeType          string     `json:"mime_type"`
	ByteSize          int64      `json:"byte_size"`
	S3OriginalKey     string     `json:"s3_original_key"`
	ContentHash       string     `json:"content_hash"`
	Status            string     `json:"status"`
	PageCount         int        `json:"page_count"`
	ChunkCount        int        `json:"chunk_count"`
	EmbeddingModel    string     `json:"embedding_model"`
	ErrorMessage      string     `json:"error_message"`
	TokensTotal       int64      `json:"tokens_total"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	IndexedAt         *time.Time `json:"indexed_at,omitempty"`
	ProductName       string     `json:"product_name,omitempty"`
	ProductSlug       string     `json:"product_slug,omitempty"`
}

type DocChunk struct {
	ID                uuid.UUID `json:"id"`
	DocSourceID       uuid.UUID `json:"doc_source_id"`
	ProductID         uuid.UUID `json:"product_id"`
	ManufacturerOrgID uuid.UUID `json:"manufacturer_org_id"`
	Version           string    `json:"version"`
	PageNumber        int       `json:"page_number"`
	ChunkIndex        int       `json:"chunk_index"`
	Section           string    `json:"section"`
	ContentMD         string    `json:"content_md"`
	S3PageKey         string    `json:"s3_page_key"`
	TokenEstimate     int       `json:"token_estimate"`
}

type RAGHit struct {
	ChunkID     uuid.UUID `json:"chunk_id"`
	DocSourceID uuid.UUID `json:"doc_source_id"`
	ProductID   uuid.UUID `json:"product_id"`
	Version     string    `json:"version"`
	PageNumber  int       `json:"page_number"`
	ContentMD   string    `json:"content_md"`
	S3PageKey   string    `json:"s3_page_key"`
	Score       float64   `json:"score"`
	FromKeyword bool      `json:"from_keyword"`
	Filename    string    `json:"filename,omitempty"`
	ProductName string    `json:"product_name,omitempty"`
}

type DocsRepo struct {
	pool *pgxpool.Pool
}

func NewDocsRepo(pool *pgxpool.Pool) *DocsRepo {
	return &DocsRepo{pool: pool}
}

func (r *DocsRepo) CreateProduct(ctx context.Context, manufacturerOrgID uuid.UUID, slug, name, kind, description string) (*Product, error) {
	var p Product
	err := r.pool.QueryRow(ctx, `
		INSERT INTO products (manufacturer_org_id, slug, name, kind, description)
		VALUES ($1, $2, $3, $4::product_kind, $5)
		RETURNING id, manufacturer_org_id, slug, name, kind::text, description, created_at, updated_at
	`, manufacturerOrgID, slug, name, kind, description).Scan(
		&p.ID, &p.ManufacturerOrgID, &p.Slug, &p.Name, &p.Kind, &p.Description, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	return &p, nil
}

func (r *DocsRepo) ListProducts(ctx context.Context, manufacturerOrgID uuid.UUID) ([]Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, manufacturer_org_id, slug, name, kind::text, description, created_at, updated_at
		FROM products
		WHERE manufacturer_org_id = $1
		ORDER BY name ASC
	`, manufacturerOrgID)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	var out []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.ManufacturerOrgID, &p.Slug, &p.Name, &p.Kind, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *DocsRepo) GetProduct(ctx context.Context, id uuid.UUID) (*Product, error) {
	var p Product
	err := r.pool.QueryRow(ctx, `
		SELECT id, manufacturer_org_id, slug, name, kind::text, description, created_at, updated_at
		FROM products WHERE id = $1
	`, id).Scan(&p.ID, &p.ManufacturerOrgID, &p.Slug, &p.Name, &p.Kind, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}
	return &p, nil
}

func (r *DocsRepo) FindByContentHash(ctx context.Context, manufacturerOrgID uuid.UUID, hash string) (*DocSource, error) {
	var d DocSource
	err := r.pool.QueryRow(ctx, `
		SELECT id, product_id, manufacturer_org_id, version, filename, mime_type, byte_size,
		       s3_original_key, content_hash, status::text, page_count, chunk_count,
		       embedding_model, error_message, tokens_total, created_at, updated_at, indexed_at
		FROM doc_sources
		WHERE manufacturer_org_id = $1 AND content_hash = $2 AND status = 'ready'
		ORDER BY indexed_at DESC NULLS LAST
		LIMIT 1
	`, manufacturerOrgID, hash).Scan(
		&d.ID, &d.ProductID, &d.ManufacturerOrgID, &d.Version, &d.Filename, &d.MimeType, &d.ByteSize,
		&d.S3OriginalKey, &d.ContentHash, &d.Status, &d.PageCount, &d.ChunkCount,
		&d.EmbeddingModel, &d.ErrorMessage, &d.TokensTotal, &d.CreatedAt, &d.UpdatedAt, &d.IndexedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find by hash: %w", err)
	}
	return &d, nil
}

func (r *DocsRepo) CreateDocSource(ctx context.Context, d *DocSource) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO doc_sources (
			product_id, manufacturer_org_id, version, filename, mime_type, byte_size,
			s3_original_key, content_hash, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::doc_source_status)
		RETURNING id, created_at, updated_at
	`, d.ProductID, d.ManufacturerOrgID, d.Version, d.Filename, d.MimeType, d.ByteSize,
		d.S3OriginalKey, d.ContentHash, d.Status,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *DocsRepo) UpdateDocSourceStatus(ctx context.Context, id uuid.UUID, status, errMsg string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE doc_sources
		SET status = $2::doc_source_status, error_message = $3, updated_at = now()
		WHERE id = $1
	`, id, status, errMsg)
	return err
}

func (r *DocsRepo) MarkDocSourceReady(ctx context.Context, id uuid.UUID, pageCount, chunkCount int, model string, tokens int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE doc_sources
		SET status = 'ready',
		    page_count = $2,
		    chunk_count = $3,
		    embedding_model = $4,
		    tokens_total = $5,
		    error_message = '',
		    indexed_at = now(),
		    updated_at = now()
		WHERE id = $1
	`, id, pageCount, chunkCount, model, tokens)
	return err
}

func (r *DocsRepo) GetDocSource(ctx context.Context, id uuid.UUID) (*DocSource, error) {
	var d DocSource
	err := r.pool.QueryRow(ctx, `
		SELECT ds.id, ds.product_id, ds.manufacturer_org_id, ds.version, ds.filename, ds.mime_type, ds.byte_size,
		       ds.s3_original_key, ds.content_hash, ds.status::text, ds.page_count, ds.chunk_count,
		       ds.embedding_model, ds.error_message, ds.tokens_total, ds.created_at, ds.updated_at, ds.indexed_at,
		       p.name, p.slug
		FROM doc_sources ds
		JOIN products p ON p.id = ds.product_id
		WHERE ds.id = $1
	`, id).Scan(
		&d.ID, &d.ProductID, &d.ManufacturerOrgID, &d.Version, &d.Filename, &d.MimeType, &d.ByteSize,
		&d.S3OriginalKey, &d.ContentHash, &d.Status, &d.PageCount, &d.ChunkCount,
		&d.EmbeddingModel, &d.ErrorMessage, &d.TokensTotal, &d.CreatedAt, &d.UpdatedAt, &d.IndexedAt,
		&d.ProductName, &d.ProductSlug,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get doc source: %w", err)
	}
	return &d, nil
}

func (r *DocsRepo) ListDocSources(ctx context.Context, manufacturerOrgID *uuid.UUID, limit int) ([]DocSource, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows pgx.Rows
	var err error
	if manufacturerOrgID != nil {
		rows, err = r.pool.Query(ctx, `
			SELECT ds.id, ds.product_id, ds.manufacturer_org_id, ds.version, ds.filename, ds.mime_type, ds.byte_size,
			       ds.s3_original_key, ds.content_hash, ds.status::text, ds.page_count, ds.chunk_count,
			       ds.embedding_model, ds.error_message, ds.tokens_total, ds.created_at, ds.updated_at, ds.indexed_at,
			       p.name, p.slug
			FROM doc_sources ds
			JOIN products p ON p.id = ds.product_id
			WHERE ds.manufacturer_org_id = $1
			ORDER BY ds.created_at DESC
			LIMIT $2
		`, *manufacturerOrgID, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT ds.id, ds.product_id, ds.manufacturer_org_id, ds.version, ds.filename, ds.mime_type, ds.byte_size,
			       ds.s3_original_key, ds.content_hash, ds.status::text, ds.page_count, ds.chunk_count,
			       ds.embedding_model, ds.error_message, ds.tokens_total, ds.created_at, ds.updated_at, ds.indexed_at,
			       p.name, p.slug
			FROM doc_sources ds
			JOIN products p ON p.id = ds.product_id
			ORDER BY ds.created_at DESC
			LIMIT $1
		`, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list doc sources: %w", err)
	}
	defer rows.Close()
	var out []DocSource
	for rows.Next() {
		var d DocSource
		if err := rows.Scan(
			&d.ID, &d.ProductID, &d.ManufacturerOrgID, &d.Version, &d.Filename, &d.MimeType, &d.ByteSize,
			&d.S3OriginalKey, &d.ContentHash, &d.Status, &d.PageCount, &d.ChunkCount,
			&d.EmbeddingModel, &d.ErrorMessage, &d.TokensTotal, &d.CreatedAt, &d.UpdatedAt, &d.IndexedAt,
			&d.ProductName, &d.ProductSlug,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *DocsRepo) DeleteChunksForSource(ctx context.Context, sourceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM doc_chunks WHERE doc_source_id = $1`, sourceID)
	return err
}

func (r *DocsRepo) DeletePagesForSource(ctx context.Context, sourceID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM doc_pages WHERE doc_source_id = $1`, sourceID)
	return err
}

type DocPage struct {
	DocSourceID uuid.UUID `json:"doc_source_id"`
	PageNumber  int       `json:"page_number"`
	S3PageKey   string    `json:"s3_page_key"`
	S3ParsedKey string    `json:"s3_parsed_key"`
	TextSource  string    `json:"text_source"`
	CharCount   int       `json:"char_count"`
}

func (r *DocsRepo) UpsertPage(ctx context.Context, p DocPage) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO doc_pages (doc_source_id, page_number, s3_page_key, s3_parsed_key, text_source, char_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (doc_source_id, page_number)
		DO UPDATE SET
			s3_page_key = EXCLUDED.s3_page_key,
			s3_parsed_key = EXCLUDED.s3_parsed_key,
			text_source = EXCLUDED.text_source,
			char_count = EXCLUDED.char_count
	`, p.DocSourceID, p.PageNumber, p.S3PageKey, p.S3ParsedKey, p.TextSource, p.CharCount)
	return err
}

func (r *DocsRepo) GetPage(ctx context.Context, sourceID uuid.UUID, pageNumber int) (*DocPage, error) {
	var p DocPage
	err := r.pool.QueryRow(ctx, `
		SELECT doc_source_id, page_number, s3_page_key, s3_parsed_key, text_source, char_count
		FROM doc_pages
		WHERE doc_source_id = $1 AND page_number = $2
	`, sourceID, pageNumber).Scan(&p.DocSourceID, &p.PageNumber, &p.S3PageKey, &p.S3ParsedKey, &p.TextSource, &p.CharCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get page: %w", err)
	}
	return &p, nil
}

func (r *DocsRepo) InsertChunk(ctx context.Context, c DocChunk, embeddingLiteral string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO doc_chunks (
			doc_source_id, product_id, manufacturer_org_id, version,
			page_number, chunk_index, section, content_md, s3_page_key, embedding, token_estimate
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::vector,$11)
		RETURNING id
	`, c.DocSourceID, c.ProductID, c.ManufacturerOrgID, c.Version,
		c.PageNumber, c.ChunkIndex, c.Section, c.ContentMD, c.S3PageKey, embeddingLiteral, c.TokenEstimate,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert chunk: %w", err)
	}
	return id, nil
}

func (r *DocsRepo) SearchSemantic(ctx context.Context, queryVecLiteral string, productIDs []uuid.UUID, version string, limit int, threshold float64) ([]RAGHit, error) {
	if limit <= 0 {
		limit = 10
	}
	version = strings.TrimSpace(version)
	var rows pgx.Rows
	var err error
	if len(productIDs) == 0 {
		rows, err = r.pool.Query(ctx, `
			SELECT dc.id, dc.doc_source_id, dc.product_id, dc.version, dc.page_number, dc.content_md, dc.s3_page_key,
			       1 - (dc.embedding <=> $1::vector) AS score,
			       ds.filename, p.name
			FROM doc_chunks dc
			JOIN doc_sources ds ON ds.id = dc.doc_source_id
			JOIN products p ON p.id = dc.product_id
			WHERE dc.embedding IS NOT NULL
			  AND ($2::text = '' OR dc.version = $2 OR dc.version = '')
			  AND 1 - (dc.embedding <=> $1::vector) >= $3
			ORDER BY dc.embedding <=> $1::vector
			LIMIT $4
		`, queryVecLiteral, version, threshold, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT dc.id, dc.doc_source_id, dc.product_id, dc.version, dc.page_number, dc.content_md, dc.s3_page_key,
			       1 - (dc.embedding <=> $1::vector) AS score,
			       ds.filename, p.name
			FROM doc_chunks dc
			JOIN doc_sources ds ON ds.id = dc.doc_source_id
			JOIN products p ON p.id = dc.product_id
			WHERE dc.embedding IS NOT NULL
			  AND dc.product_id = ANY($2::uuid[])
			  AND ($3::text = '' OR dc.version = $3 OR dc.version = '')
			  AND 1 - (dc.embedding <=> $1::vector) >= $4
			ORDER BY dc.embedding <=> $1::vector
			LIMIT $5
		`, queryVecLiteral, productIDs, version, threshold, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("semantic search: %w", err)
	}
	defer rows.Close()
	return scanHits(rows, false)
}

func (r *DocsRepo) SearchKeyword(ctx context.Context, query string, productIDs []uuid.UUID, version string, limit int) ([]RAGHit, error) {
	if limit <= 0 {
		limit = 10
	}
	q := strings.TrimSpace(query)
	if len(q) < 2 {
		return nil, nil
	}
	version = strings.TrimSpace(version)
	var rows pgx.Rows
	var err error
	if len(productIDs) == 0 {
		rows, err = r.pool.Query(ctx, `
			SELECT dc.id, dc.doc_source_id, dc.product_id, dc.version, dc.page_number, dc.content_md, dc.s3_page_key,
			       ts_rank(to_tsvector('russian', dc.content_md), plainto_tsquery('russian', $1))::float8 AS score,
			       ds.filename, p.name
			FROM doc_chunks dc
			JOIN doc_sources ds ON ds.id = dc.doc_source_id
			JOIN products p ON p.id = dc.product_id
			WHERE to_tsvector('russian', dc.content_md) @@ plainto_tsquery('russian', $1)
			  AND ($2::text = '' OR dc.version = $2 OR dc.version = '')
			ORDER BY score DESC
			LIMIT $3
		`, q, version, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT dc.id, dc.doc_source_id, dc.product_id, dc.version, dc.page_number, dc.content_md, dc.s3_page_key,
			       ts_rank(to_tsvector('russian', dc.content_md), plainto_tsquery('russian', $1))::float8 AS score,
			       ds.filename, p.name
			FROM doc_chunks dc
			JOIN doc_sources ds ON ds.id = dc.doc_source_id
			JOIN products p ON p.id = dc.product_id
			WHERE to_tsvector('russian', dc.content_md) @@ plainto_tsquery('russian', $1)
			  AND dc.product_id = ANY($2::uuid[])
			  AND ($3::text = '' OR dc.version = $3 OR dc.version = '')
			ORDER BY score DESC
			LIMIT $4
		`, q, productIDs, version, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("keyword search: %w", err)
	}
	defer rows.Close()
	return scanHits(rows, true)
}

func (r *DocsRepo) LogUsage(ctx context.Context, orgID *uuid.UUID, operation, model string, tokensIn, tokensTotal int64, docSourceID *uuid.UUID, meta map[string]any) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO usage_log (org_id, operation, model, tokens_in, tokens_total, doc_source_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7::jsonb, '{}'::jsonb))
	`, orgID, operation, model, tokensIn, tokensTotal, docSourceID, mustJSON(meta))
	return err
}

func scanHits(rows pgx.Rows, keyword bool) ([]RAGHit, error) {
	var out []RAGHit
	for rows.Next() {
		var h RAGHit
		if err := rows.Scan(
			&h.ChunkID, &h.DocSourceID, &h.ProductID, &h.Version, &h.PageNumber, &h.ContentMD, &h.S3PageKey,
			&h.Score, &h.Filename, &h.ProductName,
		); err != nil {
			return nil, err
		}
		h.FromKeyword = keyword
		out = append(out, h)
	}
	return out, rows.Err()
}

func mustJSON(v map[string]any) []byte {
	if v == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}
