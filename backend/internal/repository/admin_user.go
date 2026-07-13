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

type AdminUserListParams struct {
	Search            string
	IsActive          *bool
	Access            string
	OrgType           string
	Role              string
	ReviewStatus      string
	IsPersonal        *bool
	HasActiveSessions *bool
	LastLogin         string
	Limit             int
	Offset            int
}

type AdminUserRepo struct {
	pool *pgxpool.Pool
}

func NewAdminUserRepo(pool *pgxpool.Pool) *AdminUserRepo {
	return &AdminUserRepo{pool: pool}
}

func (r *AdminUserRepo) List(ctx context.Context, p AdminUserListParams) ([]models.AdminUserListRow, int, error) {
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 200 {
		p.Limit = 200
	}
	if p.Offset < 0 {
		p.Offset = 0
	}

	where, args := buildAdminUserWhere(p)
	countQ := `SELECT COUNT(DISTINCT u.id) FROM users u
		LEFT JOIN org_members om ON om.user_id = u.id
		LEFT JOIN organizations o ON o.id = om.org_id
		WHERE ` + where
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count admin users: %w", err)
	}

	listArgs := append(args, p.Limit, p.Offset)
	q := fmt.Sprintf(`SELECT u.id, u.email, u.full_name, u.is_active, u.created_at, u.updated_at,
		(SELECT MAX(s.created_at) FROM sessions s WHERE s.user_id = u.id) AS last_login_at,
		(SELECT COUNT(*)::int FROM sessions s
			WHERE s.user_id = u.id AND s.revoked_at IS NULL AND s.expires_at > now()) AS active_sessions,
		COALESCE((SELECT s.ip_address FROM sessions s WHERE s.user_id = u.id ORDER BY s.created_at DESC LIMIT 1), '') AS last_ip,
		COALESCE((SELECT s.user_agent FROM sessions s WHERE s.user_id = u.id ORDER BY s.created_at DESC LIMIT 1), '') AS last_user_agent
		FROM users u
		LEFT JOIN org_members om ON om.user_id = u.id
		LEFT JOIN organizations o ON o.id = om.org_id
		WHERE %s
		GROUP BY u.id
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(listArgs)-1, len(listArgs))

	rows, err := r.pool.Query(ctx, q, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()

	var out []models.AdminUserListRow
	var ids []uuid.UUID
	for rows.Next() {
		var row models.AdminUserListRow
		if err := rows.Scan(
			&row.ID, &row.Email, &row.FullName, &row.IsActive, &row.CreatedAt, &row.UpdatedAt,
			&row.LastLoginAt, &row.ActiveSessions, &row.LastIP, &row.LastUserAgent,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, row)
		ids = append(ids, row.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return out, total, nil
	}

	memberships, err := r.membershipsForUsers(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	messengers, err := r.messengersForUsers(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		out[i].Memberships = memberships[out[i].ID]
		out[i].Messengers = messengers[out[i].ID]
		out[i].AccessLevel = computeAccessLevel(out[i].IsActive, out[i].Memberships)
	}
	return out, total, nil
}

func (r *AdminUserRepo) GetDetail(ctx context.Context, userID uuid.UUID) (*models.AdminUserListRow, []models.AdminUserSession, error) {
	u, err := scanUser(r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, full_name, is_active, created_at, updated_at FROM users WHERE id = $1`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	row := models.AdminUserListRow{User: *u}
	_ = r.pool.QueryRow(ctx, `SELECT MAX(created_at) FROM sessions WHERE user_id = $1`, userID).Scan(&row.LastLoginAt)
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()`, userID).Scan(&row.ActiveSessions)
	_ = r.pool.QueryRow(ctx, `SELECT COALESCE(ip_address, '') FROM sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, userID).Scan(&row.LastIP)
	_ = r.pool.QueryRow(ctx, `SELECT COALESCE(user_agent, '') FROM sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, userID).Scan(&row.LastUserAgent)

	memberships, err := r.membershipsForUsers(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, nil, err
	}
	row.Memberships = memberships[userID]
	messengers, err := r.messengersForUsers(ctx, []uuid.UUID{userID})
	if err != nil {
		return nil, nil, err
	}
	row.Messengers = messengers[userID]
	row.AccessLevel = computeAccessLevel(row.IsActive, row.Memberships)

	sessions, err := r.listSessionsForUser(ctx, userID, 20)
	if err != nil {
		return nil, nil, err
	}
	return &row, sessions, nil
}

func (r *AdminUserRepo) SetActive(ctx context.Context, userID uuid.UUID, active bool) (*models.User, error) {
	q := `UPDATE users SET is_active = $2, updated_at = now() WHERE id = $1
		RETURNING id, email, password_hash, full_name, is_active, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q, userID, active)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *AdminUserRepo) RevokeAllSessions(ctx context.Context, userID uuid.UUID) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *AdminUserRepo) membershipsForUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]models.AdminUserMembership, error) {
	q := `SELECT om.user_id, om.org_id, o.name, o.type::text, o.slug, om.role::text,
		o.review_status::text, o.is_personal, o.is_active, o.inn, o.website, o.contact_phone, om.created_at
		FROM org_members om
		JOIN organizations o ON o.id = om.org_id
		WHERE om.user_id = ANY($1)
		ORDER BY om.created_at ASC`
	rows, err := r.pool.Query(ctx, q, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID][]models.AdminUserMembership)
	for rows.Next() {
		var userID, orgID uuid.UUID
		var m models.AdminUserMembership
		if err := rows.Scan(
			&userID, &orgID, &m.OrgName, &m.OrgType, &m.OrgSlug, &m.Role,
			&m.ReviewStatus, &m.IsPersonal, &m.OrgIsActive, &m.INN, &m.Website, &m.ContactPhone, &m.MemberSince,
		); err != nil {
			return nil, err
		}
		m.OrgID = orgID
		out[userID] = append(out[userID], m)
	}
	return out, rows.Err()
}

func (r *AdminUserRepo) messengersForUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]models.UserMessengerLink, error) {
	q := `SELECT id, user_id, provider::text, external_user_id, username, display_name,
		is_verified, notifications_enabled, linked_at, revoked_at, created_at, updated_at
		FROM user_messenger_links
		WHERE user_id = ANY($1) AND revoked_at IS NULL
		ORDER BY provider ASC`
	rows, err := r.pool.Query(ctx, q, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID][]models.UserMessengerLink)
	for rows.Next() {
		var link models.UserMessengerLink
		if err := rows.Scan(
			&link.ID, &link.UserID, &link.Provider, &link.ExternalUserID, &link.Username, &link.DisplayName,
			&link.IsVerified, &link.NotificationsEnabled, &link.LinkedAt, &link.RevokedAt, &link.CreatedAt, &link.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out[link.UserID] = append(out[link.UserID], link)
	}
	return out, rows.Err()
}

func (r *AdminUserRepo) listSessionsForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.AdminUserSession, error) {
	q := `SELECT s.id, s.user_id, s.org_id, s.refresh_token_hash, s.user_agent, s.ip_address,
		s.expires_at, s.revoked_at, s.created_at, COALESCE(o.name, '')
		FROM sessions s
		LEFT JOIN organizations o ON o.id = s.org_id
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC
		LIMIT $2`
	rows, err := r.pool.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AdminUserSession
	for rows.Next() {
		var item models.AdminUserSession
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.OrgID, &item.RefreshTokenHash, &item.UserAgent, &item.IPAddress,
			&item.ExpiresAt, &item.RevokedAt, &item.CreatedAt, &item.OrgName,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func buildAdminUserWhere(p AdminUserListParams) (string, []any) {
	var clauses []string
	var args []any
	n := 1

	if s := strings.TrimSpace(p.Search); s != "" {
		clauses = append(clauses, fmt.Sprintf("(u.email ILIKE $%d OR u.full_name ILIKE $%d)", n, n))
		args = append(args, "%"+s+"%")
		n++
	}
	if p.IsActive != nil {
		clauses = append(clauses, fmt.Sprintf("u.is_active = $%d", n))
		args = append(args, *p.IsActive)
		n++
	}
	if t := strings.TrimSpace(p.OrgType); t != "" {
		clauses = append(clauses, fmt.Sprintf("EXISTS (SELECT 1 FROM org_members om2 JOIN organizations o2 ON o2.id = om2.org_id WHERE om2.user_id = u.id AND o2.type::text = $%d)", n))
		args = append(args, t)
		n++
	}
	if role := strings.TrimSpace(p.Role); role != "" {
		clauses = append(clauses, fmt.Sprintf("EXISTS (SELECT 1 FROM org_members om2 WHERE om2.user_id = u.id AND om2.role::text = $%d)", n))
		args = append(args, role)
		n++
	}
	if rs := strings.TrimSpace(p.ReviewStatus); rs != "" {
		clauses = append(clauses, fmt.Sprintf("EXISTS (SELECT 1 FROM org_members om2 JOIN organizations o2 ON o2.id = om2.org_id WHERE om2.user_id = u.id AND o2.review_status::text = $%d)", n))
		args = append(args, rs)
		n++
	}
	if p.IsPersonal != nil {
		clauses = append(clauses, fmt.Sprintf("EXISTS (SELECT 1 FROM org_members om2 JOIN organizations o2 ON o2.id = om2.org_id WHERE om2.user_id = u.id AND o2.is_personal = $%d)", n))
		args = append(args, *p.IsPersonal)
		n++
	}
	if p.HasActiveSessions != nil {
		sub := `EXISTS (SELECT 1 FROM sessions s WHERE s.user_id = u.id AND s.revoked_at IS NULL AND s.expires_at > now())`
		if !*p.HasActiveSessions {
			sub = "NOT " + sub
		}
		clauses = append(clauses, sub)
	}
	switch strings.TrimSpace(p.LastLogin) {
	case "never":
		clauses = append(clauses, "NOT EXISTS (SELECT 1 FROM sessions s WHERE s.user_id = u.id)")
	case "today":
		clauses = append(clauses, "EXISTS (SELECT 1 FROM sessions s WHERE s.user_id = u.id AND s.created_at >= date_trunc('day', now()))")
	case "7d":
		clauses = append(clauses, "EXISTS (SELECT 1 FROM sessions s WHERE s.user_id = u.id AND s.created_at >= now() - interval '7 days')")
	case "30d":
		clauses = append(clauses, "EXISTS (SELECT 1 FROM sessions s WHERE s.user_id = u.id AND s.created_at >= now() - interval '30 days')")
	}
	switch strings.TrimSpace(p.Access) {
	case "none":
		clauses = append(clauses, "u.is_active = FALSE")
	case "full":
		clauses = append(clauses, `u.is_active = TRUE AND EXISTS (
			SELECT 1 FROM org_members om2 JOIN organizations o2 ON o2.id = om2.org_id
			WHERE om2.user_id = u.id AND o2.review_status = 'active' AND o2.is_active = TRUE)`)
	case "limited":
		clauses = append(clauses, `u.is_active = TRUE AND NOT EXISTS (
			SELECT 1 FROM org_members om2 JOIN organizations o2 ON o2.id = om2.org_id
			WHERE om2.user_id = u.id AND o2.review_status = 'active' AND o2.is_active = TRUE)`)
	}

	where := "TRUE"
	if len(clauses) > 0 {
		where = strings.Join(clauses, " AND ")
	}
	return where, args
}

func computeAccessLevel(isActive bool, memberships []models.AdminUserMembership) string {
	if !isActive {
		return "none"
	}
	for _, m := range memberships {
		if m.ReviewStatus == "active" && m.OrgIsActive {
			return "full"
		}
	}
	if len(memberships) == 0 {
		return "limited"
	}
	return "limited"
}

func parseBoolQuery(v string) (*bool, error) {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return nil, nil
	}
	switch v {
	case "1", "true", "yes":
		b := true
		return &b, nil
	case "0", "false", "no":
		b := false
		return &b, nil
	default:
		return nil, fmt.Errorf("invalid bool")
	}
}

func ParseAdminUserListParams(rQuery map[string][]string) (AdminUserListParams, error) {
	get := func(k string) string {
		if v, ok := rQuery[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	p := AdminUserListParams{
		Search:       get("search"),
		Access:       get("access"),
		OrgType:      get("org_type"),
		Role:         get("role"),
		ReviewStatus: get("review_status"),
		LastLogin:    get("last_login"),
	}
	var err error
	if p.IsActive, err = parseBoolQuery(get("is_active")); err != nil {
		return p, err
	}
	if p.IsPersonal, err = parseBoolQuery(get("is_personal")); err != nil {
		return p, err
	}
	if p.HasActiveSessions, err = parseBoolQuery(get("has_active_sessions")); err != nil {
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
