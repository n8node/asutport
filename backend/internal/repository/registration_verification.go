package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n8node/asutport/internal/models"
)

const registrationIDPrefix = "77"

type RegistrationVerificationRepo struct {
	pool *pgxpool.Pool
}

func NewRegistrationVerificationRepo(pool *pgxpool.Pool) *RegistrationVerificationRepo {
	return &RegistrationVerificationRepo{pool: pool}
}

func NewRegistrationID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return registrationIDPrefix + hex.EncodeToString(buf), nil
}

func (r *RegistrationVerificationRepo) Create(
	ctx context.Context,
	userID, orgID uuid.UUID,
	regID, accountType string,
	expiresAt time.Time,
) (*models.RegistrationVerification, error) {
	id := uuid.New()
	q := `INSERT INTO registration_verifications (id, user_id, org_id, reg_id, account_type, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, org_id, reg_id, account_type, expires_at, used_at, created_at`
	row := r.pool.QueryRow(ctx, q, id, userID, orgID, regID, accountType, expiresAt)
	return scanRegistrationVerification(row)
}

func (r *RegistrationVerificationRepo) GetActiveByRegID(ctx context.Context, regID string) (*models.RegistrationVerification, error) {
	q := `SELECT id, user_id, org_id, reg_id, account_type, expires_at, used_at, created_at
		FROM registration_verifications
		WHERE reg_id = $1 AND used_at IS NULL AND expires_at > now()`
	row := r.pool.QueryRow(ctx, q, regID)
	v, err := scanRegistrationVerification(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return v, err
}

func (r *RegistrationVerificationRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE registration_verifications SET used_at = now() WHERE id = $1 AND used_at IS NULL`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RegistrationVerificationRepo) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM registration_verifications WHERE user_id = $1`, userID)
	return err
}

func scanRegistrationVerification(row pgx.Row) (*models.RegistrationVerification, error) {
	var v models.RegistrationVerification
	if err := row.Scan(
		&v.ID, &v.UserID, &v.OrgID, &v.RegID, &v.AccountType, &v.ExpiresAt, &v.UsedAt, &v.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *RegistrationVerificationRepo) CleanupRegistration(ctx context.Context, userID, orgID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM registration_verifications WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete verifications: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE organizations SET onboarding_ticket_id = NULL WHERE id = $1`, orgID); err != nil {
		return fmt.Errorf("clear onboarding ticket: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, orgID); err != nil {
		return fmt.Errorf("delete org: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return tx.Commit(ctx)
}
