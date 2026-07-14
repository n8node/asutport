package handler

import (
	"net/http"
	"strconv"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
	"github.com/n8node/asutport/internal/service"
)

type VendorHandler struct {
	tickets *service.TicketService
	orgs    *repository.OrgRepo
}

func NewVendorHandler(tickets *service.TicketService, orgs *repository.OrgRepo) *VendorHandler {
	return &VendorHandler{tickets: tickets, orgs: orgs}
}

func (h *VendorHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	p, _, ok := h.requireVendorOrg(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, total, err := h.tickets.ListByAssignedTarget(r.Context(), p.OrgID, limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list tickets")
		return
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, vendorTicketDTO(&list[i]))
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"total": total},
	})
}

func (h *VendorHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	p, _, ok := h.requireVendorOrg(w, r)
	if !ok {
		return
	}
	open, err := h.tickets.CountOpenByAssignedTarget(r.Context(), p.OrgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load dashboard")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"open_escalations_count": open,
		},
	})
}

func (h *VendorHandler) requireVendorOrg(w http.ResponseWriter, r *http.Request) (*auth.Principal, *models.Organization, bool) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return nil, nil, false
	}
	org, err := h.orgs.GetByID(r.Context(), p.OrgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return nil, nil, false
	}
	switch org.Type {
	case "manufacturer", "vendor", "integrator":
	default:
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "доступно только для кабинета производителя или партнёра")
		return nil, nil, false
	}
	if org.ReviewStatus != "active" {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "организация ещё не активирована")
		return nil, nil, false
	}
	return p, org, true
}

func vendorTicketDTO(t *models.Ticket) map[string]any {
	if t == nil {
		return nil
	}
	dto := map[string]any{
		"id":              t.ID.String(),
		"client_org_id":   t.ClientOrgID.String(),
		"client_org_name": t.ClientOrgName,
		"type":            t.Type,
		"priority":        t.Priority,
		"status":          t.Status,
		"subject":         t.Subject,
		"created_at":      t.CreatedAt,
		"updated_at":      t.UpdatedAt,
	}
	if t.SLAReactionDeadline != nil {
		dto["sla_reaction_deadline"] = t.SLAReactionDeadline
	}
	if t.BallOwnerOrgID != nil {
		dto["ball_owner_org_id"] = t.BallOwnerOrgID.String()
		dto["ball_owner_org_name"] = t.BallOwnerOrgName
	}
	return dto
}
