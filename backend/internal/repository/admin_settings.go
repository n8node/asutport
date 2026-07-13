package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminSettingsRepo struct {
	pool *pgxpool.Pool
}

func NewAdminSettingsRepo(pool *pgxpool.Pool) *AdminSettingsRepo {
	return &AdminSettingsRepo{pool: pool}
}

func (r *AdminSettingsRepo) Get(ctx context.Context, key string, dest any) error {
	var raw []byte
	err := r.pool.QueryRow(ctx, `
		SELECT value
		FROM admin_settings
		WHERE key = $1
	`, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get admin setting: %w", err)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("decode admin setting: %w", err)
	}
	return nil
}

func (r *AdminSettingsRepo) Upsert(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode admin setting: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO admin_settings (key, value, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, key, raw)
	if err != nil {
		return fmt.Errorf("upsert admin setting: %w", err)
	}
	return nil
}
