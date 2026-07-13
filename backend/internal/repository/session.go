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

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Create(ctx context.Context, userID, orgID uuid.UUID, refreshHash, userAgent, ip string, expiresAt time.Time) (*models.Session, error) {
	id := uuid.New()
	q := `INSERT INTO sessions (id, user_id, org_id, refresh_token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, org_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at`
	row := r.pool.QueryRow(ctx, q, id, userID, orgID, refreshHash, userAgent, ip, expiresAt)
	return scanSession(row)
}

func (r *SessionRepo) GetActiveByRefreshHash(ctx context.Context, refreshHash string) (*models.Session, error) {
	q := `SELECT id, user_id, org_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at
		FROM sessions
		WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`
	row := r.pool.QueryRow(ctx, q, refreshHash)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *SessionRepo) GetActiveByID(ctx context.Context, id uuid.UUID) (*models.Session, error) {
	q := `SELECT id, user_id, org_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at
		FROM sessions
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > now()`
	row := r.pool.QueryRow(ctx, q, id)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *SessionRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanSession(row pgx.Row) (*models.Session, error) {
	var s models.Session
	if err := row.Scan(&s.ID, &s.UserID, &s.OrgID, &s.RefreshTokenHash, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) Rotate(ctx context.Context, oldID uuid.UUID, refreshHash string, expiresAt time.Time) (*models.Session, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var s models.Session
	q := `SELECT id, user_id, org_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at
		FROM sessions WHERE id = $1 AND revoked_at IS NULL AND expires_at > now() FOR UPDATE`
	row := tx.QueryRow(ctx, q, oldID)
	if err := row.Scan(&s.ID, &s.UserID, &s.OrgID, &s.RefreshTokenHash, &s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock session: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE id = $1`, oldID); err != nil {
		return nil, err
	}
	newID := uuid.New()
	ins := `INSERT INTO sessions (id, user_id, org_id, refresh_token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, org_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at`
	row = tx.QueryRow(ctx, ins, newID, s.UserID, s.OrgID, refreshHash, s.UserAgent, s.IPAddress, expiresAt)
	out, err := scanSession(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
