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

func (r *OrgRepo) Create(ctx context.Context, name, orgType, slug string) (*models.Organization, error) {
	id := uuid.New()
	q := `INSERT INTO organizations (id, name, type, slug)
		VALUES ($1, $2, $3::org_type, $4)
		RETURNING id, name, type::text, slug, is_active, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q, id, name, orgType, slug)
	o, err := scanOrg(row)
	if err != nil {
		return nil, wrapInsert(err, "insert organization")
	}
	return o, nil
}

func (r *OrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Organization, error) {
	q := `SELECT id, name, type::text, slug, is_active, created_at, updated_at FROM organizations WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	o, err := scanOrg(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

func (r *OrgRepo) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	q := `SELECT id, name, type::text, slug, is_active, created_at, updated_at FROM organizations WHERE slug = $1`
	row := r.pool.QueryRow(ctx, q, slug)
	o, err := scanOrg(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

func scanOrg(row pgx.Row) (*models.Organization, error) {
	var o models.Organization
	if err := row.Scan(&o.ID, &o.Name, &o.Type, &o.Slug, &o.IsActive, &o.CreatedAt, &o.UpdatedAt); err != nil {
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
		o.name, o.type::text, o.slug
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
		if err := rows.Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt, &m.OrgName, &m.OrgType, &m.OrgSlug); err != nil {
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
