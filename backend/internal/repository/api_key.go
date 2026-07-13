package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n8node/asutport/internal/models"
)

type APIKeyRepo struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepo(pool *pgxpool.Pool) *APIKeyRepo {
	return &APIKeyRepo{pool: pool}
}

func (r *APIKeyRepo) Create(ctx context.Context, orgID uuid.UUID, name, keyHash, keyPrefix string) (*models.APIKey, error) {
	id := uuid.New()
	q := `INSERT INTO api_keys (id, org_id, name, key_hash, key_prefix)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, org_id, name, key_prefix, key_hash, last_used_at, revoked_at, created_at`
	row := r.pool.QueryRow(ctx, q, id, orgID, name, keyHash, keyPrefix)
	return scanAPIKey(row)
}

func (r *APIKeyRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]models.APIKey, error) {
	q := `SELECT id, org_id, name, key_prefix, key_hash, last_used_at, revoked_at, created_at
		FROM api_keys WHERE org_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()
	var out []models.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *k)
	}
	return out, rows.Err()
}

func (r *APIKeyRepo) Revoke(ctx context.Context, orgID, keyID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND org_id = $2 AND revoked_at IS NULL`,
		keyID, orgID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *APIKeyRepo) FindActiveByPrefix(ctx context.Context, prefix string) (*models.APIKey, error) {
	q := `SELECT id, org_id, name, key_prefix, key_hash, last_used_at, revoked_at, created_at
		FROM api_keys WHERE key_prefix = $1 AND revoked_at IS NULL`
	row := r.pool.QueryRow(ctx, q, prefix)
	k, err := scanAPIKey(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return k, err
}

func (r *APIKeyRepo) Touch(ctx context.Context, keyID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = $2 WHERE id = $1`, keyID, time.Now().UTC())
	return err
}

func scanAPIKey(row pgx.Row) (*models.APIKey, error) {
	var k models.APIKey
	if err := row.Scan(&k.ID, &k.OrgID, &k.Name, &k.KeyPrefix, &k.KeyHash, &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt); err != nil {
		return nil, err
	}
	return &k, nil
}
