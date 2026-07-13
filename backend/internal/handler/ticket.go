package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
	"github.com/n8node/asutport/internal/service"
)

type TicketHandler struct {
	tickets *service.TicketService
	orgs    *repository.OrgRepo
}

func NewTicketHandler(tickets *service.TicketService, orgs *repository.OrgRepo) *TicketHandler {
	return &TicketHandler{tickets: tickets, orgs: orgs}
}

func (h *TicketHandler) GetOnboarding(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	ticket, err := h.tickets.GetOnboardingForOrg(r.Context(), p.OrgID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "onboarding ticket not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load ticket")
		return
	}
	attachments, _ := h.tickets.ListAttachments(r.Context(), ticket.ID)
	WriteJSON(w, http.StatusOK, map[string]any{"data": ticketDTO(ticket, attachments)})
}

func (h *TicketHandler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket id")
		return
	}
	ticket, err := h.loadAuthorizedTicket(w, r, ticketID, p)
	if err != nil || ticket == nil {
		return
	}
	if err := h.ensureOnboardingAccess(w, r, ticket, p); err != nil {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxAttachmentBytes()+1<<20)
	if err := r.ParseMultipartForm(service.MaxAttachmentBytes() + 1<<20); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file too large or invalid form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()
	data, err := readUploadFile(file, service.MaxAttachmentBytes())
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	event, err := h.tickets.UploadAttachment(
		r.Context(),
		ticket,
		p.UserID,
		p.OrgID,
		p.IsSuperAdmin(),
		header.Filename,
		header.Header.Get("Content-Type"),
		bytes.NewReader(data),
		int64(len(data)),
	)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	updated, _ := h.tickets.GetByID(r.Context(), ticket.ID)
	attachments, _ := h.tickets.ListAttachments(r.Context(), ticket.ID)
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"event":       eventDTO(*event),
			"ticket":      ticketDTO(updated, attachments),
			"attachments": attachmentDTOs(attachments),
		},
	})
}

func (h *TicketHandler) Get(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket id")
		return
	}
	ticket, err := h.loadAuthorizedTicket(w, r, ticketID, p)
	if err != nil || ticket == nil {
		return
	}
	attachments, _ := h.tickets.ListAttachments(r.Context(), ticket.ID)
	WriteJSON(w, http.StatusOK, map[string]any{"data": ticketDTO(ticket, attachments)})
}

func (h *TicketHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket id")
		return
	}
	ticket, err := h.loadAuthorizedTicket(w, r, ticketID, p)
	if err != nil || ticket == nil {
		return
	}
	events, err := h.tickets.ListEvents(r.Context(), ticket.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load events")
		return
	}
	items := make([]map[string]any, 0, len(events))
	for _, e := range events {
		items = append(items, eventDTO(e))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

type postMessageReq struct {
	Text string `json:"text"`
}

func (h *TicketHandler) PostMessage(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket id")
		return
	}
	ticket, err := h.loadAuthorizedTicket(w, r, ticketID, p)
	if err != nil || ticket == nil {
		return
	}
	if err := h.ensureOnboardingAccess(w, r, ticket, p); err != nil {
		return
	}
	var req postMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	event, err := h.tickets.PostMessage(r.Context(), ticket, p.UserID, p.OrgID, p.IsSuperAdmin(), req.Text)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	updated, _ := h.tickets.GetByID(r.Context(), ticket.ID)
	attachments, _ := h.tickets.ListAttachments(r.Context(), ticket.ID)
	WriteJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"event":  eventDTO(*event),
			"ticket": ticketDTO(updated, attachments),
		},
	})
}

type presignAttachmentReq struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
}

func (h *TicketHandler) PresignAttachment(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket id")
		return
	}
	ticket, err := h.loadAuthorizedTicket(w, r, ticketID, p)
	if err != nil || ticket == nil {
		return
	}
	if err := h.ensureOnboardingAccess(w, r, ticket, p); err != nil {
		return
	}
	var req presignAttachmentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	out, err := h.tickets.PresignAttachment(r.Context(), ticket, p.UserID, p.OrgID, service.PresignAttachmentInput{
		Filename:    req.Filename,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"attachment_id": out.AttachmentID.String(),
			"upload_url":    out.UploadURL,
			"s3_key":        out.S3Key,
		},
	})
}

func (h *TicketHandler) CompleteAttachment(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket id")
		return
	}
	attachmentID, err := uuid.Parse(chi.URLParam(r, "attachmentID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid attachment id")
		return
	}
	ticket, err := h.loadAuthorizedTicket(w, r, ticketID, p)
	if err != nil || ticket == nil {
		return
	}
	if err := h.ensureOnboardingAccess(w, r, ticket, p); err != nil {
		return
	}
	event, err := h.tickets.CompleteAttachment(r.Context(), ticket, attachmentID, p.UserID, p.OrgID, p.IsSuperAdmin())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "attachment not found")
			return
		}
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	updated, _ := h.tickets.GetByID(r.Context(), ticket.ID)
	attachments, _ := h.tickets.ListAttachments(r.Context(), ticket.ID)
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"event":  eventDTO(*event),
			"ticket": ticketDTO(updated, attachments),
		},
	})
}

func (h *TicketHandler) AttachmentURL(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket id")
		return
	}
	attachmentID, err := uuid.Parse(chi.URLParam(r, "attachmentID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid attachment id")
		return
	}
	ticket, err := h.loadAuthorizedTicket(w, r, ticketID, p)
	if err != nil || ticket == nil {
		return
	}
	url, err := h.tickets.AttachmentDownloadURL(r.Context(), ticket, attachmentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "attachment not found")
			return
		}
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"url": url}})
}

func (h *TicketHandler) ListOnboardingAdmin(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	reviewStatus := strings.TrimSpace(r.URL.Query().Get("review_status"))
	if reviewStatus == "" {
		reviewStatus = "pending_review"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, total, err := h.tickets.ListOnboarding(r.Context(), reviewStatus, limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list tickets")
		return
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, ticketDTO(&list[i], nil))
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"total": total, "limit": limit, "offset": offset},
	})
}

type reviewActionReq struct {
	Rationale string `json:"rationale"`
}

func (h *TicketHandler) ApproveOrg(w http.ResponseWriter, r *http.Request) {
	h.reviewOrg(w, r, true)
}

func (h *TicketHandler) RejectOrg(w http.ResponseWriter, r *http.Request) {
	h.reviewOrg(w, r, false)
}

func (h *TicketHandler) reviewOrg(w http.ResponseWriter, r *http.Request, approve bool) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok || !p.IsSuperAdmin() {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "superadmin only")
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "ticketID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket id")
		return
	}
	ticket, err := h.tickets.GetByID(r.Context(), ticketID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "ticket not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load ticket")
		return
	}
	if ticket.Type != "onboarding" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket type")
		return
	}
	var req reviewActionReq
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if approve {
		err = h.tickets.ApproveOrg(r.Context(), ticket, p.UserID, req.Rationale)
	} else {
		err = h.tickets.RejectOrg(r.Context(), ticket, p.UserID, req.Rationale)
	}
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	updated, _ := h.tickets.GetByID(r.Context(), ticket.ID)
	attachments, _ := h.tickets.ListAttachments(r.Context(), ticket.ID)
	WriteJSON(w, http.StatusOK, map[string]any{"data": ticketDTO(updated, attachments)})
}

func (h *TicketHandler) loadAuthorizedTicket(w http.ResponseWriter, r *http.Request, ticketID uuid.UUID, p *auth.Principal) (*models.Ticket, error) {
	ticket, err := h.tickets.GetByID(r.Context(), ticketID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "ticket not found")
			return nil, err
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load ticket")
		return nil, err
	}
	if !h.tickets.CanAccess(r.Context(), ticket, p.UserID, p.OrgID, p.IsSuperAdmin()) {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
		return nil, repository.ErrNotFound
	}
	return ticket, nil
}

func (h *TicketHandler) ensureOnboardingAccess(w http.ResponseWriter, r *http.Request, ticket *models.Ticket, p *auth.Principal) error {
	if p.IsSuperAdmin() {
		return nil
	}
	if ticket.Type != "onboarding" {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "ticket type not allowed")
		return errors.New("forbidden")
	}
	org, err := h.orgs.GetByID(r.Context(), p.OrgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return err
	}
	if org.ReviewStatus != "pending_review" && ticket.Status != "closed" {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "organization is not pending review")
		return errors.New("forbidden")
	}
	return nil
}

func ticketDTO(t *models.Ticket, attachments []models.TicketAttachment) map[string]any {
	if t == nil {
		return nil
	}
	dto := map[string]any{
		"id":                   t.ID.String(),
		"client_org_id":        t.ClientOrgID.String(),
		"type":                 t.Type,
		"priority":             t.Priority,
		"status":               t.Status,
		"subject":              t.Subject,
		"client_org_name":      t.ClientOrgName,
		"client_org_type":      t.ClientOrgType,
		"client_org_inn":       t.ClientOrgINN,
		"client_review_status": t.ClientReviewStatus,
		"created_at":           t.CreatedAt,
		"updated_at":           t.UpdatedAt,
		"attachments":          attachmentDTOs(attachments),
	}
	if t.InstallationID != nil {
		dto["installation_id"] = t.InstallationID.String()
	}
	if t.BallOwnerOrgID != nil {
		dto["ball_owner_org_id"] = t.BallOwnerOrgID.String()
	}
	if t.CreatedByUserID != nil {
		dto["created_by_user_id"] = t.CreatedByUserID.String()
	}
	return dto
}

func attachmentDTOs(list []models.TicketAttachment) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, map[string]any{
			"id":          a.ID.String(),
			"filename":    a.Filename,
			"content_type": a.ContentType,
			"size_bytes":  a.SizeBytes,
			"created_at":  a.CreatedAt,
		})
	}
	return out
}

func eventDTO(e models.TicketEvent) map[string]any {
	var payload any
	_ = json.Unmarshal(e.Payload, &payload)
	dto := map[string]any{
		"id":         e.ID.String(),
		"ticket_id":  e.TicketID.String(),
		"kind":       e.Kind,
		"payload":    payload,
		"created_at": e.CreatedAt,
		"is_platform": e.IsPlatform,
	}
	if e.ActorUserID != nil {
		dto["actor_user_id"] = e.ActorUserID.String()
	}
	if e.ActorOrgID != nil {
		dto["actor_org_id"] = e.ActorOrgID.String()
	}
	if e.ActorName != "" {
		dto["actor_name"] = e.ActorName
	}
	if e.ActorEmail != "" {
		dto["actor_email"] = e.ActorEmail
	}
	return dto
}

func readUploadFile(r io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("could not read file")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file is empty")
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file too large")
	}
	return data, nil
}

func userFacingErr(err error) string {
	if err == nil {
		return "request failed"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "required"),
		strings.Contains(msg, "unsupported"),
		strings.Contains(msg, "invalid"),
		strings.Contains(msg, "empty"),
		strings.Contains(msg, "large"),
		strings.Contains(msg, "closed"),
		strings.Contains(msg, "pending review"),
		strings.Contains(msg, "not configured"),
		strings.Contains(msg, "upload failed"),
		strings.Contains(msg, "could not read"):
		return msg
	default:
		return "request failed"
	}
}
