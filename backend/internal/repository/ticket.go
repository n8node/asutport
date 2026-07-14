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

type TicketRepo struct {
	pool *pgxpool.Pool
}

func NewTicketRepo(pool *pgxpool.Pool) *TicketRepo {
	return &TicketRepo{pool: pool}
}

type TicketCreateParams struct {
	ClientOrgID         uuid.UUID
	InstallationID      *uuid.UUID
	Type                string
	Priority            string
	Status              string
	BallOwnerOrgID      *uuid.UUID
	AssignedTargetOrgID *uuid.UUID
	Subject             string
	CreatedByUserID     uuid.UUID
	SLAReactionDeadline *time.Time
}

func (r *TicketRepo) Create(ctx context.Context, p TicketCreateParams) (*models.Ticket, error) {
	id := uuid.New()
	if p.Priority == "" {
		p.Priority = "question"
	}
	if p.Status == "" {
		p.Status = "open"
	}
	q := `INSERT INTO tickets (
			id, client_org_id, installation_id, type, priority, status, ball_owner_org_id,
			assigned_target_org_id, subject, created_by_user_id, sla_reaction_deadline
		) VALUES ($1, $2, $3, $4::ticket_type, $5::ticket_priority, $6::ticket_status, $7, $8, $9, $10, $11)
		RETURNING id, client_org_id, installation_id, type::text, priority::text, status::text,
			ball_owner_org_id, assigned_target_org_id, subject, created_by_user_id, sla_reaction_deadline, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q,
		id, p.ClientOrgID, p.InstallationID, p.Type, p.Priority, p.Status, p.BallOwnerOrgID,
		p.AssignedTargetOrgID, p.Subject, p.CreatedByUserID, p.SLAReactionDeadline,
	)
	return scanTicket(row)
}

const ticketDetailSelect = `SELECT t.id, t.client_org_id, t.installation_id, t.type::text, t.priority::text, t.status::text,
			t.ball_owner_org_id, t.assigned_target_org_id, t.subject, t.created_by_user_id, t.sla_reaction_deadline, t.created_at, t.updated_at,
			o.name, o.type::text, o.inn, o.review_status::text,
			COALESCE(bo.name, ''), COALESCE(at.name, '')`

const ticketDetailFrom = `FROM tickets t
		JOIN organizations o ON o.id = t.client_org_id
		LEFT JOIN organizations bo ON bo.id = t.ball_owner_org_id
		LEFT JOIN organizations at ON at.id = t.assigned_target_org_id`

func (r *TicketRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Ticket, error) {
	q := ticketDetailSelect + `
		` + ticketDetailFrom + `
		WHERE t.id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	t, err := scanTicketDetail(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *TicketRepo) GetOnboardingByClientOrg(ctx context.Context, orgID uuid.UUID) (*models.Ticket, error) {
	q := ticketDetailSelect + `
		` + ticketDetailFrom + `
		WHERE t.client_org_id = $1 AND t.type = 'onboarding'
		LIMIT 1`
	row := r.pool.QueryRow(ctx, q, orgID)
	t, err := scanTicketDetail(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *TicketRepo) ListOnboarding(ctx context.Context, reviewStatus string, limit, offset int) ([]models.Ticket, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	countQ := `SELECT COUNT(*)
		FROM tickets t
		JOIN organizations o ON o.id = t.client_org_id
		WHERE t.type = 'onboarding' AND ($1 = '' OR o.review_status::text = $1)`
	var total int
	if err := r.pool.QueryRow(ctx, countQ, reviewStatus).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count onboarding tickets: %w", err)
	}
	q := ticketDetailSelect + `
		` + ticketDetailFrom + `
		WHERE t.type = 'onboarding' AND ($1 = '' OR o.review_status::text = $1)
		ORDER BY t.updated_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, reviewStatus, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list onboarding tickets: %w", err)
	}
	defer rows.Close()
	var out []models.Ticket
	for rows.Next() {
		t, err := scanTicketDetail(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

func (r *TicketRepo) ListByClientOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.Ticket, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	countQ := `SELECT COUNT(*) FROM tickets WHERE client_org_id = $1 AND type <> 'onboarding'`
	var total int
	if err := r.pool.QueryRow(ctx, countQ, orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count client tickets: %w", err)
	}
	q := ticketDetailSelect + `
		` + ticketDetailFrom + `
		WHERE t.client_org_id = $1 AND t.type <> 'onboarding'
		ORDER BY t.updated_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list client tickets: %w", err)
	}
	defer rows.Close()
	var out []models.Ticket
	for rows.Next() {
		t, err := scanTicketDetail(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

func (r *TicketRepo) ListByAssignedTarget(ctx context.Context, targetOrgID uuid.UUID, limit, offset int) ([]models.Ticket, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	countQ := `SELECT COUNT(*) FROM tickets
		WHERE assigned_target_org_id = $1 AND type <> 'onboarding' AND status NOT IN ('resolved', 'closed')`
	var total int
	if err := r.pool.QueryRow(ctx, countQ, targetOrgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count vendor tickets: %w", err)
	}
	q := ticketDetailSelect + `
		` + ticketDetailFrom + `
		WHERE t.assigned_target_org_id = $1 AND t.type <> 'onboarding' AND t.status NOT IN ('resolved', 'closed')
		ORDER BY
			CASE t.priority WHEN 'emergency' THEN 0 WHEN 'degraded' THEN 1 ELSE 2 END,
			t.sla_reaction_deadline NULLS LAST,
			t.updated_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, targetOrgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list vendor tickets: %w", err)
	}
	defer rows.Close()
	var out []models.Ticket
	for rows.Next() {
		t, err := scanTicketDetail(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

func (r *TicketRepo) CountOpenByAssignedTarget(ctx context.Context, targetOrgID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tickets
		 WHERE assigned_target_org_id = $1 AND type <> 'onboarding' AND status NOT IN ('resolved', 'closed')`,
		targetOrgID,
	).Scan(&n)
	return n, err
}

func (r *TicketRepo) UpdateRouting(ctx context.Context, ticketID uuid.UUID, assignedTarget *uuid.UUID, status string, ballOwner *uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE tickets SET assigned_target_org_id = $2, status = $3::ticket_status, ball_owner_org_id = $4, updated_at = now() WHERE id = $1`,
		ticketID, assignedTarget, status, ballOwner,
	)
	if err != nil {
		return fmt.Errorf("update ticket routing: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TicketRepo) CountOpenByClientOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tickets
		 WHERE client_org_id = $1 AND type <> 'onboarding' AND status NOT IN ('resolved', 'closed')`,
		orgID,
	).Scan(&n)
	return n, err
}

func (r *TicketRepo) CountSLAActiveByClientOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM tickets
		 WHERE client_org_id = $1 AND type <> 'onboarding'
		   AND status NOT IN ('resolved', 'closed')
		   AND sla_reaction_deadline IS NOT NULL`,
		orgID,
	).Scan(&n)
	return n, err
}

func (r *TicketRepo) UpdateStatus(ctx context.Context, ticketID uuid.UUID, status string, ballOwnerOrgID *uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE tickets SET status = $2::ticket_status, ball_owner_org_id = $3, updated_at = now() WHERE id = $1`,
		ticketID, status, ballOwnerOrgID,
	)
	if err != nil {
		return fmt.Errorf("update ticket status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TicketRepo) Touch(ctx context.Context, ticketID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE tickets SET updated_at = now() WHERE id = $1`, ticketID)
	return err
}

func (r *TicketRepo) AddEvent(ctx context.Context, ticketID uuid.UUID, kind string, actorUserID, actorOrgID *uuid.UUID, payload any) (*models.TicketEvent, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal event payload: %w", err)
	}
	id := uuid.New()
	q := `INSERT INTO ticket_events (id, ticket_id, kind, actor_user_id, actor_org_id, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, ticket_id, kind, actor_user_id, actor_org_id, payload, created_at`
	row := r.pool.QueryRow(ctx, q, id, ticketID, kind, actorUserID, actorOrgID, raw)
	return scanTicketEvent(row)
}

func (r *TicketRepo) ListEvents(ctx context.Context, ticketID uuid.UUID, limit int) ([]models.TicketEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := `SELECT e.id, e.ticket_id, e.kind, e.actor_user_id, e.actor_org_id, e.payload, e.created_at,
			COALESCE(u.full_name, ''), COALESCE(u.email, ''),
			CASE WHEN e.actor_org_id IS NULL AND e.kind IN ('message', 'org_approved', 'org_rejected') THEN TRUE ELSE FALSE END
		FROM ticket_events e
		LEFT JOIN users u ON u.id = e.actor_user_id
		WHERE e.ticket_id = $1
		ORDER BY e.created_at ASC
		LIMIT $2`
	rows, err := r.pool.Query(ctx, q, ticketID, limit)
	if err != nil {
		return nil, fmt.Errorf("list ticket events: %w", err)
	}
	defer rows.Close()
	var out []models.TicketEvent
	for rows.Next() {
		var e models.TicketEvent
		if err := rows.Scan(
			&e.ID, &e.TicketID, &e.Kind, &e.ActorUserID, &e.ActorOrgID, &e.Payload, &e.CreatedAt,
			&e.ActorName, &e.ActorEmail, &e.IsPlatform,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *TicketRepo) CreateAttachment(ctx context.Context, a models.TicketAttachment) (*models.TicketAttachment, error) {
	id := a.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	q := `INSERT INTO ticket_attachments (
			id, ticket_id, s3_key, filename, content_type, size_bytes,
			uploaded_by_user_id, uploaded_by_org_id, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, ticket_id, event_id, s3_key, filename, content_type, size_bytes,
			uploaded_by_user_id, uploaded_by_org_id, status, created_at`
	row := r.pool.QueryRow(ctx, q,
		id, a.TicketID, a.S3Key, a.Filename, a.ContentType, a.SizeBytes,
		a.UploadedByUserID, a.UploadedByOrgID, a.Status,
	)
	return scanAttachment(row)
}

func (r *TicketRepo) GetAttachment(ctx context.Context, id uuid.UUID) (*models.TicketAttachment, error) {
	q := `SELECT id, ticket_id, event_id, s3_key, filename, content_type, size_bytes,
			uploaded_by_user_id, uploaded_by_org_id, status, created_at
		FROM ticket_attachments WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	a, err := scanAttachment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *TicketRepo) CompleteAttachment(ctx context.Context, id, eventID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE ticket_attachments SET status = 'completed', event_id = $2 WHERE id = $1 AND status = 'pending'`,
		id, eventID,
	)
	if err != nil {
		return fmt.Errorf("complete attachment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TicketRepo) ListAttachments(ctx context.Context, ticketID uuid.UUID) ([]models.TicketAttachment, error) {
	q := `SELECT id, ticket_id, event_id, s3_key, filename, content_type, size_bytes,
			uploaded_by_user_id, uploaded_by_org_id, status, created_at
		FROM ticket_attachments
		WHERE ticket_id = $1 AND status = 'completed'
		ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, q, ticketID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()
	var out []models.TicketAttachment
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func scanTicket(row pgx.Row) (*models.Ticket, error) {
	var t models.Ticket
	var installationID *uuid.UUID
	var ballOwner *uuid.UUID
	var assignedTarget *uuid.UUID
	var createdBy *uuid.UUID
	if err := row.Scan(
		&t.ID, &t.ClientOrgID, &installationID, &t.Type, &t.Priority, &t.Status,
		&ballOwner, &assignedTarget, &t.Subject, &createdBy, &t.SLAReactionDeadline, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	t.InstallationID = installationID
	t.BallOwnerOrgID = ballOwner
	t.AssignedTargetOrgID = assignedTarget
	t.CreatedByUserID = createdBy
	return &t, nil
}

func scanTicketDetail(row pgx.Row) (*models.Ticket, error) {
	var t models.Ticket
	var installationID *uuid.UUID
	var ballOwner *uuid.UUID
	var assignedTarget *uuid.UUID
	var createdBy *uuid.UUID
	if err := row.Scan(
		&t.ID, &t.ClientOrgID, &installationID, &t.Type, &t.Priority, &t.Status,
		&ballOwner, &assignedTarget, &t.Subject, &createdBy, &t.SLAReactionDeadline, &t.CreatedAt, &t.UpdatedAt,
		&t.ClientOrgName, &t.ClientOrgType, &t.ClientOrgINN, &t.ClientReviewStatus,
		&t.BallOwnerOrgName, &t.AssignedTargetName,
	); err != nil {
		return nil, err
	}
	t.InstallationID = installationID
	t.BallOwnerOrgID = ballOwner
	t.AssignedTargetOrgID = assignedTarget
	t.CreatedByUserID = createdBy
	return &t, nil
}

func scanTicketEvent(row pgx.Row) (*models.TicketEvent, error) {
	var e models.TicketEvent
	if err := row.Scan(
		&e.ID, &e.TicketID, &e.Kind, &e.ActorUserID, &e.ActorOrgID, &e.Payload, &e.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &e, nil
}

func scanAttachment(row pgx.Row) (*models.TicketAttachment, error) {
	var a models.TicketAttachment
	if err := row.Scan(
		&a.ID, &a.TicketID, &a.EventID, &a.S3Key, &a.Filename, &a.ContentType, &a.SizeBytes,
		&a.UploadedByUserID, &a.UploadedByOrgID, &a.Status, &a.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &a, nil
}

const OnboardingWelcomeMessage = `Здравствуйте! Для активации организации на платформе ASUTPORT пришлите подтверждающие документы в этом тикете.

Рекомендуемые материалы:
— выписка ЕГРЮЛ или подтверждение ИНН;
— документ о полномочиях представителя (при необходимости);
— скан договора или письмо на фирменном бланке.

Можно приложить файлы PDF или изображения (PNG, JPEG).`

func EventPayloadText(text string) map[string]string {
	return map[string]string{"text": text}
}

func EventPayloadAttachment(attachmentID uuid.UUID, filename string) map[string]any {
	return map[string]any{
		"attachment_id": attachmentID.String(),
		"filename":    filename,
	}
}

func EventPayloadReview(rationale string) map[string]string {
	return map[string]string{"rationale": rationale}
}

func EventPayloadEscalation(targetOrgName string) map[string]string {
	return map[string]string{"target_org_name": targetOrgName}
}

func EventPayloadFallback(neededRole, missingOrgName, message string) map[string]string {
	return map[string]string{
		"needed_role":      neededRole,
		"missing_org_name": missingOrgName,
		"message":          message,
	}
}

func EventPayloadResolved(note string) map[string]string {
	return map[string]string{"note": note}
}
