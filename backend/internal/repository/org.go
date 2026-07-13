package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n8node/asutport/internal/models"
)

type OrgRepo struct {
	pool *pgxpool.Pool
}

func NewOrgRepo(pool *pgxpool.Pool) *OrgRepo {
	return &OrgRepo{pool: pool}
}

type OrgCreateParams struct {
	Name          string
	Type          string
	Slug          string
	LegalName     string
	INN           string
	Website       string
	ContactPhone  string
	ReviewComment string
	IsPersonal    bool
	ReviewStatus  string
}

func (r *OrgRepo) Create(ctx context.Context, name, orgType, slug string) (*models.Organization, error) {
	return r.CreateWithReview(ctx, OrgCreateParams{
		Name:         name,
		Type:         orgType,
		Slug:         slug,
		ReviewStatus: "active",
	})
}

func (r *OrgRepo) CreateWithReview(ctx context.Context, p OrgCreateParams) (*models.Organization, error) {
	id := uuid.New()
	if p.ReviewStatus == "" {
		p.ReviewStatus = "active"
	}
	q := `INSERT INTO organizations (
			id, name, type, slug, legal_name, inn, website, contact_phone,
			review_comment, is_personal, review_status
		)
		VALUES ($1, $2, $3::org_type, $4, $5, $6, $7, $8, $9, $10, $11::org_review_status)
		RETURNING id, name, type::text, slug, is_active, legal_name, inn, website, contact_phone,
			review_comment, is_personal, review_status::text, reviewed_at, reviewed_by, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q,
		id, p.Name, p.Type, p.Slug, p.LegalName, p.INN, p.Website, p.ContactPhone,
		p.ReviewComment, p.IsPersonal, p.ReviewStatus,
	)
	o, err := scanOrg(row)
	if err != nil {
		return nil, wrapInsert(err, "insert organization")
	}
	return o, nil
}

func (r *OrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	q := `SELECT id, name, type::text, slug, is_active, legal_name, inn, website, contact_phone,
		review_comment, is_personal, review_status::text, reviewed_at, reviewed_by, created_at, updated_at
		FROM organizations WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	o, err := scanOrg(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

func (r *OrgRepo) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	q := `SELECT id, name, type::text, slug, is_active, legal_name, inn, website, contact_phone,
		review_comment, is_personal, review_status::text, reviewed_at, reviewed_by, created_at, updated_at
		FROM organizations WHERE slug = $1`
	row := r.pool.QueryRow(ctx, q, slug)
	o, err := scanOrg(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

type OrgListParams struct {
	ReviewStatus string
	Type         string
	Limit        int
	Offset       int
}

func (r *OrgRepo) List(ctx context.Context, p OrgListParams) ([]models.Organization, error) {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 50
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	q := `SELECT id, name, type::text, slug, is_active, legal_name, inn, website, contact_phone,
			review_comment, is_personal, review_status::text, reviewed_at, reviewed_by, created_at, updated_at
		FROM organizations
		WHERE ($1 = '' OR review_status::text = $1)
			AND ($2 = '' OR type::text = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, q, p.ReviewStatus, p.Type, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()
	var out []models.Organization
	for rows.Next() {
		o, err := scanOrg(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func (r *OrgRepo) UpdateReviewStatus(ctx context.Context, orgID, reviewerID uuid.UUID, status string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE organizations
		SET review_status = $2::org_review_status, reviewed_at = now(), reviewed_by = $3, updated_at = now()
		WHERE id = $1`,
		orgID, status, reviewerID,
	)
	if err != nil {
		return fmt.Errorf("update organization review status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanOrg(row pgx.Row) (*models.Organization, error) {
	var o models.Organization
	if err := row.Scan(
		&o.ID, &o.Name, &o.Type, &o.Slug, &o.IsActive, &o.LegalName, &o.INN, &o.Website,
		&o.ContactPhone, &o.ReviewComment, &o.IsPersonal, &o.ReviewStatus, &o.ReviewedAt,
		&o.ReviewedBy, &o.CreatedAt, &o.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &o, nil
}

type OrgMemberRepo struct {
	pool *pgxpool.Pool
}

func NewOrgMemberRepo(pool *pgxpool.Pool) *OrgMemberRepo {
	return &OrgMemberRepo{pool: pool}
}

func (r *OrgMemberRepo) Create(ctx context.Context, orgID, userID uuid.UUID, role string) (*models.OrgMember, error) {
	id := uuid.New()
	q := `INSERT INTO org_members (id, org_id, user_id, role)
		VALUES ($1, $2, $3, $4::org_member_role)
		RETURNING id, org_id, user_id, role::text, created_at`
	row := r.pool.QueryRow(ctx, q, id, orgID, userID, role)
	var m models.OrgMember
	if err := row.Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
		return nil, wrapInsert(err, "insert org_member")
	}
	return &m, nil
}

func (r *OrgMemberRepo) GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*models.OrgMember, error) {
	q := `SELECT id, org_id, user_id, role::text, created_at FROM org_members WHERE org_id = $1 AND user_id = $2`
	row := r.pool.QueryRow(ctx, q, orgID, userID)
	var m models.OrgMember
	if err := row.Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get membership: %w", err)
	}
	return &m, nil
}

func (r *OrgMemberRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.OrgMembership, error) {
	q := `SELECT om.id, om.org_id, om.user_id, om.role::text, om.created_at,
		o.name, o.type::text, o.slug, o.review_status::text, o.is_personal
		FROM org_members om
		JOIN organizations o ON o.id = om.org_id
		WHERE om.user_id = $1 AND o.is_active = TRUE
		ORDER BY om.created_at ASC`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()
	var out []models.OrgMembership
	for rows.Next() {
		var m models.OrgMembership
		if err := rows.Scan(
			&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt, &m.OrgName, &m.OrgType,
			&m.OrgSlug, &m.OrgReviewStatus, &m.OrgIsPersonal,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *OrgMemberRepo) FirstMembership(ctx context.Context, userID uuid.UUID) (*models.OrgMembership, error) {
	list, err := r.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrNotFound
	}
	return &list[0], nil
}

func (r *OrgMemberRepo) PrimaryAdminForOrg(ctx context.Context, orgID uuid.UUID) (*models.OrgMember, error) {
	q := `SELECT id, org_id, user_id, role::text, created_at FROM org_members
		WHERE org_id = $1 AND role IN ('superadmin', 'owner', 'admin')
		ORDER BY CASE role WHEN 'superadmin' THEN 0 WHEN 'owner' THEN 1 ELSE 2 END, created_at ASC
		LIMIT 1`
	row := r.pool.QueryRow(ctx, q, orgID)
	var m models.OrgMember
	if err := row.Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("primary admin for org: %w", err)
	}
	return &m, nil
}
