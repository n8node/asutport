package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n8node/asutport/internal/models"
)

type AdminOrgListParams struct {
	Type         string
	ReviewStatus string
	Search       string
	IsActive     *bool
	IsPersonal   *bool
	Limit        int
	Offset       int
}

type AdminOrgRepo struct {
	pool *pgxpool.Pool
}

func NewAdminOrgRepo(pool *pgxpool.Pool) *AdminOrgRepo {
	return &AdminOrgRepo{pool: pool}
}

func (r *AdminOrgRepo) List(ctx context.Context, p AdminOrgListParams) ([]models.AdminOrgListRow, int, error) {
	if p.Limit <= 0 {
		p.Limit = 100
	}
	if p.Limit > 200 {
		p.Limit = 200
	}
	if p.Offset < 0 {
		p.Offset = 0
	}

	where, args := buildAdminOrgWhere(p)
	countQ := `SELECT COUNT(*) FROM organizations o WHERE ` + where
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count admin orgs: %w", err)
	}

	listArgs := append(args, p.Limit, p.Offset)
	q := fmt.Sprintf(`SELECT o.id, o.name, o.type::text, o.slug, o.is_active, o.legal_name, o.inn, o.website, o.contact_phone,
		o.review_comment, o.is_personal, o.review_status::text, o.reviewed_at, o.reviewed_by, o.created_at, o.updated_at,
		(SELECT COUNT(*)::int FROM org_members om WHERE om.org_id = o.id) AS member_count
		FROM organizations o
		WHERE %s
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(listArgs)-1, len(listArgs))

	rows, err := r.pool.Query(ctx, q, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin orgs: %w", err)
	}
	defer rows.Close()

	var out []models.AdminOrgListRow
	var ids []uuid.UUID
	for rows.Next() {
		var row models.AdminOrgListRow
		if err := rows.Scan(
			&row.ID, &row.Name, &row.Type, &row.Slug, &row.IsActive, &row.LegalName, &row.INN, &row.Website,
			&row.ContactPhone, &row.ReviewComment, &row.IsPersonal, &row.ReviewStatus, &row.ReviewedAt,
			&row.ReviewedBy, &row.CreatedAt, &row.UpdatedAt, &row.MemberCount,
		); err != nil {
			return nil, 0, err
		}
		row.Metrics = stubOrgMetrics(row.Type)
		out = append(out, row)
		ids = append(ids, row.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	owners, err := r.ownersForOrgs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if o, ok := owners[out[i].ID]; ok {
			out[i].Owner = &o
		}
	}
	return out, total, nil
}

func (r *AdminOrgRepo) GetDetail(ctx context.Context, orgID uuid.UUID) (*models.AdminOrgListRow, []models.AdminOrgMember, error) {
	q := `SELECT id, name, type::text, slug, is_active, legal_name, inn, website, contact_phone,
		review_comment, is_personal, review_status::text, reviewed_at, reviewed_by, created_at, updated_at
		FROM organizations WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, orgID)
	var org models.AdminOrgListRow
	if err := row.Scan(
		&org.ID, &org.Name, &org.Type, &org.Slug, &org.IsActive, &org.LegalName, &org.INN, &org.Website,
		&org.ContactPhone, &org.ReviewComment, &org.IsPersonal, &org.ReviewStatus, &org.ReviewedAt,
		&org.ReviewedBy, &org.CreatedAt, &org.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM org_members WHERE org_id = $1`, orgID).Scan(&org.MemberCount)
	org.Metrics = stubOrgMetrics(org.Type)

	owners, err := r.ownersForOrgs(ctx, []uuid.UUID{orgID})
	if err != nil {
		return nil, nil, err
	}
	if o, ok := owners[orgID]; ok {
		org.Owner = &o
	}

	members, err := r.listMembers(ctx, orgID)
	if err != nil {
		return nil, nil, err
	}
	return &org, members, nil
}

func (r *AdminOrgRepo) SetActive(ctx context.Context, orgID uuid.UUID, active bool) (*models.Organization, error) {
	q := `UPDATE organizations SET is_active = $2, updated_at = now() WHERE id = $1
		RETURNING id, name, type::text, slug, is_active, legal_name, inn, website, contact_phone,
			review_comment, is_personal, review_status::text, reviewed_at, reviewed_by, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q, orgID, active)
	o, err := scanOrg(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

func (r *AdminOrgRepo) ownersForOrgs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID]models.AdminOrgOwner, error) {
	if len(orgIDs) == 0 {
		return map[uuid.UUID]models.AdminOrgOwner{}, nil
	}
	q := `SELECT DISTINCT ON (om.org_id) om.org_id, u.id, u.email, u.full_name, om.role::text
		FROM org_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.org_id = ANY($1) AND om.role IN ('superadmin', 'owner', 'admin')
		ORDER BY om.org_id, CASE om.role WHEN 'superadmin' THEN 0 WHEN 'owner' THEN 1 ELSE 2 END, om.created_at ASC`
	rows, err := r.pool.Query(ctx, q, orgIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]models.AdminOrgOwner)
	for rows.Next() {
		var orgID, userID uuid.UUID
		var o models.AdminOrgOwner
		if err := rows.Scan(&orgID, &userID, &o.Email, &o.FullName, &o.Role); err != nil {
			return nil, err
		}
		o.UserID = userID
		out[orgID] = o
	}
	return out, rows.Err()
}

func (r *AdminOrgRepo) listMembers(ctx context.Context, orgID uuid.UUID) ([]models.AdminOrgMember, error) {
	q := `SELECT u.id, u.email, u.full_name, om.role::text, u.is_active, om.created_at
		FROM org_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.org_id = $1
		ORDER BY CASE om.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 WHEN 'superadmin' THEN 2 ELSE 3 END, om.created_at ASC`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AdminOrgMember
	for rows.Next() {
		var m models.AdminOrgMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.FullName, &m.Role, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func buildAdminOrgWhere(p AdminOrgListParams) (string, []any) {
	var clauses []string
	var args []any
	n := 1

	if t := strings.TrimSpace(p.Type); t != "" {
		clauses = append(clauses, fmt.Sprintf("o.type::text = $%d", n))
		args = append(args, t)
		n++
	}
	if rs := strings.TrimSpace(p.ReviewStatus); rs != "" {
		clauses = append(clauses, fmt.Sprintf("o.review_status::text = $%d", n))
		args = append(args, rs)
		n++
	}
	if s := strings.TrimSpace(p.Search); s != "" {
		clauses = append(clauses, fmt.Sprintf("(o.name ILIKE $%d OR o.legal_name ILIKE $%d OR o.inn ILIKE $%d OR o.slug ILIKE $%d)", n, n, n, n))
		args = append(args, "%"+s+"%")
		n++
	}
	if p.IsActive != nil {
		clauses = append(clauses, fmt.Sprintf("o.is_active = $%d", n))
		args = append(args, *p.IsActive)
		n++
	}
	if p.IsPersonal != nil {
		clauses = append(clauses, fmt.Sprintf("o.is_personal = $%d", n))
		args = append(args, *p.IsPersonal)
		n++
	}

	where := "TRUE"
	if len(clauses) > 0 {
		where = strings.Join(clauses, " AND ")
	}
	return where, args
}

func stubOrgMetrics(orgType string) models.AdminOrgMetrics {
	m := models.AdminOrgMetrics{PlanName: ""}
	switch orgType {
	case "client_org":
		// metrics appear after billing/installations phases
	case "manufacturer":
		m.SupportZoneLoaded = false
		m.GoldenSetReady = false
	case "vendor", "integrator":
		// entitlement/fallback after phase 6
	}
	return m
}

func ParseAdminOrgListParams(rQuery map[string][]string) (AdminOrgListParams, error) {
	get := func(k string) string {
		if v, ok := rQuery[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	p := AdminOrgListParams{
		Type:         get("type"),
		ReviewStatus: get("review_status"),
		Search:       get("search"),
	}
	var err error
	if p.IsActive, err = parseBoolQuery(get("is_active")); err != nil {
		return p, err
	}
	if p.IsPersonal, err = parseBoolQuery(get("is_personal")); err != nil {
		return p, err
	}
	if v := get("limit"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &p.Limit); err != nil {
			return p, err
		}
	}
	if v := get("offset"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &p.Offset); err != nil {
			return p, err
		}
	}
	return p, nil
}
