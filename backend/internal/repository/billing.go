package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n8node/asutport/internal/models"
)

type BillingRepo struct {
	pool *pgxpool.Pool
}

func NewBillingRepo(pool *pgxpool.Pool) *BillingRepo {
	return &BillingRepo{pool: pool}
}

type PlanUpsertParams struct {
	OrgType         string
	Name            string
	Slug            string
	PriceMonthlyRub int
	TicketQuota     *int
	OveragePriceRub int
	SLAMatrix       json.RawMessage
	Features        json.RawMessage
	IsPublic        bool
	IsArchived      bool
	SortOrder       int
}

func (r *BillingRepo) ListPlans(ctx context.Context, orgType string, includeArchived bool) ([]models.Plan, error) {
	q := `SELECT id, org_type::text, name, slug, price_monthly_rub, ticket_quota, overage_price_rub,
		sla_matrix, features, is_public, is_archived, sort_order, created_at, updated_at
		FROM plans WHERE ($1 = '' OR org_type::text = $1)`
	args := []any{orgType}
	if !includeArchived {
		q += ` AND is_archived = FALSE`
	}
	q += ` ORDER BY sort_order ASC, name ASC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()
	return scanPlans(rows)
}

func (r *BillingRepo) GetPlanByID(ctx context.Context, id uuid.UUID) (*models.Plan, error) {
	row := r.pool.QueryRow(ctx, planSelectQ+` WHERE id = $1`, id)
	plan, err := scanPlan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return plan, err
}

func (r *BillingRepo) GetPlanBySlug(ctx context.Context, orgType, slug string) (*models.Plan, error) {
	row := r.pool.QueryRow(ctx, planSelectQ+` WHERE org_type::text = $1 AND slug = $2`, orgType, slug)
	plan, err := scanPlan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return plan, err
}

func (r *BillingRepo) CreatePlan(ctx context.Context, p PlanUpsertParams) (*models.Plan, error) {
	sla := p.SLAMatrix
	if len(sla) == 0 {
		sla = json.RawMessage(`{}`)
	}
	features := p.Features
	if len(features) == 0 {
		features = json.RawMessage(`{}`)
	}
	id := uuid.New()
	row := r.pool.QueryRow(ctx,
		`INSERT INTO plans (
			id, org_type, name, slug, price_monthly_rub, ticket_quota, overage_price_rub,
			sla_matrix, features, is_public, is_archived, sort_order
		) VALUES ($1, $2::plan_org_type, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, org_type::text, name, slug, price_monthly_rub, ticket_quota, overage_price_rub,
			sla_matrix, features, is_public, is_archived, sort_order, created_at, updated_at`,
		id, p.OrgType, p.Name, p.Slug, p.PriceMonthlyRub, p.TicketQuota, p.OveragePriceRub,
		sla, features, p.IsPublic, p.IsArchived, p.SortOrder,
	)
	return scanPlan(row)
}

func (r *BillingRepo) UpdatePlan(ctx context.Context, id uuid.UUID, p PlanUpsertParams) (*models.Plan, error) {
	sla := p.SLAMatrix
	if len(sla) == 0 {
		sla = json.RawMessage(`{}`)
	}
	features := p.Features
	if len(features) == 0 {
		features = json.RawMessage(`{}`)
	}
	row := r.pool.QueryRow(ctx,
		`UPDATE plans SET
			org_type = $2::plan_org_type, name = $3, slug = $4, price_monthly_rub = $5,
			ticket_quota = $6, overage_price_rub = $7, sla_matrix = $8, features = $9,
			is_public = $10, is_archived = $11, sort_order = $12, updated_at = now()
		WHERE id = $1
		RETURNING id, org_type::text, name, slug, price_monthly_rub, ticket_quota, overage_price_rub,
			sla_matrix, features, is_public, is_archived, sort_order, created_at, updated_at`,
		id, p.OrgType, p.Name, p.Slug, p.PriceMonthlyRub, p.TicketQuota, p.OveragePriceRub,
		sla, features, p.IsPublic, p.IsArchived, p.SortOrder,
	)
	plan, err := scanPlan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return plan, err
}

func (r *BillingRepo) GetSubscriptionByOrgID(ctx context.Context, orgID uuid.UUID) (*models.Subscription, error) {
	row := r.pool.QueryRow(ctx, subscriptionSelectQ+` WHERE s.org_id = $1`, orgID)
	sub, err := scanSubscription(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return sub, err
}

func (r *BillingRepo) CreateSubscription(ctx context.Context, orgID, planID uuid.UUID, periodStart, periodEnd time.Time) (*models.Subscription, error) {
	id := uuid.New()
	row := r.pool.QueryRow(ctx,
		`INSERT INTO subscriptions (id, org_id, plan_id, status, current_period_start, current_period_end)
		VALUES ($1, $2, $3, 'active', $4, $5)
		RETURNING id, org_id, plan_id, status::text, current_period_start, current_period_end,
			cancel_at_period_end, created_at, updated_at`,
		id, orgID, planID, periodStart, periodEnd,
	)
	sub, err := scanSubscriptionBase(row)
	if err != nil {
		return nil, wrapInsert(err, "insert subscription")
	}
	return r.enrichSubscription(ctx, sub)
}

func (r *BillingRepo) UpdateSubscriptionPlan(ctx context.Context, orgID, planID uuid.UUID) (*models.Subscription, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE subscriptions SET plan_id = $2, updated_at = now() WHERE org_id = $1
		RETURNING id, org_id, plan_id, status::text, current_period_start, current_period_end,
			cancel_at_period_end, created_at, updated_at`,
		orgID, planID,
	)
	sub, err := scanSubscriptionBase(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.enrichSubscription(ctx, sub)
}

func (r *BillingRepo) ListPaymentsByOrg(ctx context.Context, orgID uuid.UUID, limit int) ([]models.Payment, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, subscription_id, ticket_id, type::text, amount_kopecks, status::text,
			COALESCE(invoice_s3_key, ''), COALESCE(note, ''), recorded_by, created_at, updated_at
		FROM payments WHERE org_id = $1 ORDER BY created_at DESC LIMIT $2`,
		orgID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()
	var out []models.Payment
	for rows.Next() {
		var p models.Payment
		if err := rows.Scan(
			&p.ID, &p.OrgID, &p.SubscriptionID, &p.TicketID, &p.Type, &p.AmountKopecks, &p.Status,
			&p.InvoiceS3Key, &p.Note, &p.RecordedBy, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type PaymentCreateParams struct {
	OrgID          uuid.UUID
	SubscriptionID *uuid.UUID
	TicketID       *uuid.UUID
	Type           string
	AmountKopecks  int
	Status         string
	InvoiceS3Key   string
	Note           string
	RecordedBy     *uuid.UUID
}

func (r *BillingRepo) CreatePayment(ctx context.Context, p PaymentCreateParams) (*models.Payment, error) {
	if p.Status == "" {
		p.Status = "pending"
	}
	id := uuid.New()
	row := r.pool.QueryRow(ctx,
		`INSERT INTO payments (
			id, org_id, subscription_id, ticket_id, type, amount_kopecks, status, invoice_s3_key, note, recorded_by
		) VALUES ($1, $2, $3, $4, $5::payment_type, $6, $7::payment_status, NULLIF($8, ''), $9, $10)
		RETURNING id, org_id, subscription_id, ticket_id, type::text, amount_kopecks, status::text,
			COALESCE(invoice_s3_key, ''), COALESCE(note, ''), recorded_by, created_at, updated_at`,
		id, p.OrgID, p.SubscriptionID, p.TicketID, p.Type, p.AmountKopecks, p.Status,
		p.InvoiceS3Key, p.Note, p.RecordedBy,
	)
	var out models.Payment
	err := row.Scan(
		&out.ID, &out.OrgID, &out.SubscriptionID, &out.TicketID, &out.Type, &out.AmountKopecks, &out.Status,
		&out.InvoiceS3Key, &out.Note, &out.RecordedBy, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, wrapInsert(err, "insert payment")
	}
	return &out, nil
}

type MRRTotals struct {
	Total        int
	Client       int
	Manufacturer int
	Partner      int
}

func (r *BillingRepo) SumActiveMRR(ctx context.Context) (MRRTotals, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(p.price_monthly_rub), 0)::int,
			COALESCE(SUM(p.price_monthly_rub) FILTER (WHERE p.org_type = 'client'), 0)::int,
			COALESCE(SUM(p.price_monthly_rub) FILTER (WHERE p.org_type = 'manufacturer'), 0)::int,
			COALESCE(SUM(p.price_monthly_rub) FILTER (WHERE p.org_type = 'partner'), 0)::int
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.status = 'active'`,
	)
	var t MRRTotals
	err := row.Scan(&t.Total, &t.Client, &t.Manufacturer, &t.Partner)
	return t, err
}

func (r *BillingRepo) CountActiveSubscriptions(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM subscriptions WHERE status = 'active'`).Scan(&n)
	return n, err
}

const planSelectQ = `SELECT id, org_type::text, name, slug, price_monthly_rub, ticket_quota, overage_price_rub,
	sla_matrix, features, is_public, is_archived, sort_order, created_at, updated_at FROM plans`

const subscriptionSelectQ = `SELECT s.id, s.org_id, s.plan_id, s.status::text, s.current_period_start, s.current_period_end,
	s.cancel_at_period_end, s.created_at, s.updated_at,
	p.name, p.slug, p.org_type::text, p.ticket_quota, p.overage_price_rub, p.price_monthly_rub
	FROM subscriptions s
	JOIN plans p ON p.id = s.plan_id`

func scanPlan(row pgx.Row) (*models.Plan, error) {
	var p models.Plan
	err := row.Scan(
		&p.ID, &p.OrgType, &p.Name, &p.Slug, &p.PriceMonthlyRub, &p.TicketQuota, &p.OveragePriceRub,
		&p.SLAMatrix, &p.Features, &p.IsPublic, &p.IsArchived, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPlans(rows pgx.Rows) ([]models.Plan, error) {
	var out []models.Plan
	for rows.Next() {
		var p models.Plan
		if err := rows.Scan(
			&p.ID, &p.OrgType, &p.Name, &p.Slug, &p.PriceMonthlyRub, &p.TicketQuota, &p.OveragePriceRub,
			&p.SLAMatrix, &p.Features, &p.IsPublic, &p.IsArchived, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func scanSubscription(row pgx.Row) (*models.Subscription, error) {
	var s models.Subscription
	err := row.Scan(
		&s.ID, &s.OrgID, &s.PlanID, &s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
		&s.CancelAtPeriodEnd, &s.CreatedAt, &s.UpdatedAt,
		&s.PlanName, &s.PlanSlug, &s.PlanOrgType, &s.TicketQuota, &s.OveragePriceRub, &s.PriceMonthlyRub,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func scanSubscriptionBase(row pgx.Row) (*models.Subscription, error) {
	var s models.Subscription
	err := row.Scan(
		&s.ID, &s.OrgID, &s.PlanID, &s.Status, &s.CurrentPeriodStart, &s.CurrentPeriodEnd,
		&s.CancelAtPeriodEnd, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *BillingRepo) enrichSubscription(ctx context.Context, sub *models.Subscription) (*models.Subscription, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT name, slug, org_type::text, ticket_quota, overage_price_rub, price_monthly_rub FROM plans WHERE id = $1`,
		sub.PlanID,
	)
	err := row.Scan(&sub.PlanName, &sub.PlanSlug, &sub.PlanOrgType, &sub.TicketQuota, &sub.OveragePriceRub, &sub.PriceMonthlyRub)
	if err != nil {
		return nil, err
	}
	return sub, nil
}
