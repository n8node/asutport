package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n8node/asutport/internal/models"
)

type FallbackRepo struct {
	pool *pgxpool.Pool
}

func NewFallbackRepo(pool *pgxpool.Pool) *FallbackRepo {
	return &FallbackRepo{pool: pool}
}

func (r *FallbackRepo) Create(ctx context.Context, ticketID uuid.UUID, neededRole, missingOrgName string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO fallback_log (ticket_id, needed_role, missing_org_name) VALUES ($1, $2, $3)`,
		ticketID, neededRole, missingOrgName,
	)
	if err != nil {
		return fmt.Errorf("insert fallback_log: %w", err)
	}
	return nil
}

func (r *FallbackRepo) ListByTicket(ctx context.Context, ticketID uuid.UUID) ([]models.FallbackLogEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, ticket_id, needed_role, missing_org_name, created_at
		 FROM fallback_log WHERE ticket_id = $1 ORDER BY created_at ASC`,
		ticketID,
	)
	if err != nil {
		return nil, fmt.Errorf("list fallback_log: %w", err)
	}
	defer rows.Close()
	var out []models.FallbackLogEntry
	for rows.Next() {
		var e models.FallbackLogEntry
		if err := rows.Scan(&e.ID, &e.TicketID, &e.NeededRole, &e.MissingOrgName, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
