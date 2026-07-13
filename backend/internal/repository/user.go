package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n8node/asutport/internal/models"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, email, passwordHash, fullName string) (*models.User, error) {
	id := uuid.New()
	q := `INSERT INTO users (id, email, password_hash, full_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, full_name, is_active, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q, id, email, passwordHash, fullName)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	q := `SELECT id, email, password_hash, full_name, is_active, created_at, updated_at FROM users WHERE email = $1`
	row := r.pool.QueryRow(ctx, q, email)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	q := `SELECT id, email, password_hash, full_name, is_active, created_at, updated_at FROM users WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func wrapInsert(err error, action string) error {
	if isUniqueViolation(err) {
		return ErrConflict
	}
	return fmt.Errorf("%s: %w", action, err)
}
