package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
	"github.com/n8node/asutport/internal/service"
)

type BillingHandler struct {
	billing *service.BillingService
	orgs    *repository.OrgRepo
}

func NewBillingHandler(billing *service.BillingService, orgs *repository.OrgRepo) *BillingHandler {
	return &BillingHandler{billing: billing, orgs: orgs}
}

func (h *BillingHandler) ClientSummary(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireClientOrg(w, r)
	if !ok {
		return
	}
	summary, err := h.billing.OrgSummary(r.Context(), org.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load billing")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": billingSummaryDTO(summary)})
}

func (h *BillingHandler) ClientQuotaCheck(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireClientOrg(w, r)
	if !ok {
		return
	}
	priority := r.URL.Query().Get("priority")
	check, err := h.billing.CheckTicketQuota(r.Context(), org.ID, priority)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not check quota")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": ticketQuotaCheckDTO(check)})
}

func (h *BillingHandler) VendorSummary(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireVendorOrg(w, r)
	if !ok {
		return
	}
	summary, err := h.billing.OrgSummary(r.Context(), org.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load billing")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": billingSummaryDTO(summary)})
}

func (h *BillingHandler) AdminOverview(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	overview, err := h.billing.AdminOverview(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load billing overview")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": overview})
}

func (h *BillingHandler) AdminListPlans(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	orgType := strings.TrimSpace(r.URL.Query().Get("org_type"))
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	plans, err := h.billing.ListPlans(r.Context(), orgType, includeArchived)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list plans")
		return
	}
	items := make([]map[string]any, 0, len(plans))
	for _, p := range plans {
		items = append(items, planDTO(&p))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

type planReq struct {
	OrgType         string          `json:"org_type"`
	Name            string          `json:"name"`
	Slug            string          `json:"slug"`
	PriceMonthlyRub int             `json:"price_monthly_rub"`
	TicketQuota     *int            `json:"ticket_quota"`
	OveragePriceRub int             `json:"overage_price_rub"`
	SLAMatrix       json.RawMessage `json:"sla_matrix"`
	Features        json.RawMessage `json:"features"`
	IsPublic        *bool           `json:"is_public"`
	IsArchived      *bool           `json:"is_archived"`
	SortOrder       *int            `json:"sort_order"`
}

func (h *BillingHandler) AdminCreatePlan(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	var req planReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	plan, err := h.billing.CreatePlan(r.Context(), planUpsertFromReq(req))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"data": planDTO(plan)})
}

func (h *BillingHandler) AdminUpdatePlan(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	planID, err := uuid.Parse(chi.URLParam(r, "planID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid plan id")
		return
	}
	var req planReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	plan, err := h.billing.UpdatePlan(r.Context(), planID, planUpsertFromReq(req))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "plan not found")
			return
		}
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": planDTO(plan)})
}

type assignSubscriptionReq struct {
	PlanID string `json:"plan_id"`
}

func (h *BillingHandler) AdminAssignSubscription(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid org id")
		return
	}
	var req assignSubscriptionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	planID, err := uuid.Parse(strings.TrimSpace(req.PlanID))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid plan id")
		return
	}
	sub, err := h.billing.AssignSubscription(r.Context(), orgID, planID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "plan or organization not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not assign subscription")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": subscriptionDTO(sub)})
}

type recordPaymentReq struct {
	OrgID          string `json:"org_id"`
	SubscriptionID string `json:"subscription_id"`
	TicketID       string `json:"ticket_id"`
	Type           string `json:"type"`
	AmountRub      int    `json:"amount_rub"`
	Status         string `json:"status"`
	InvoiceS3Key   string `json:"invoice_s3_key"`
	Note           string `json:"note"`
}

func (h *BillingHandler) AdminRecordPayment(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok || !p.IsSuperAdmin() {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "superadmin only")
		return
	}
	var req recordPaymentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	orgID, err := uuid.Parse(strings.TrimSpace(req.OrgID))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid org id")
		return
	}
	in := service.RecordPaymentInput{
		OrgID:         orgID,
		Type:          req.Type,
		AmountKopecks: req.AmountRub * 100,
		Status:        req.Status,
		InvoiceS3Key:  req.InvoiceS3Key,
		Note:          req.Note,
		RecordedBy:    p.UserID,
	}
	if strings.TrimSpace(req.SubscriptionID) != "" {
		id, parseErr := uuid.Parse(req.SubscriptionID)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid subscription id")
			return
		}
		in.SubscriptionID = &id
	}
	if strings.TrimSpace(req.TicketID) != "" {
		id, parseErr := uuid.Parse(req.TicketID)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid ticket id")
			return
		}
		in.TicketID = &id
	}
	payment, err := h.billing.RecordPayment(r.Context(), in)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"data": paymentDTO(payment)})
}

func (h *BillingHandler) AdminListPayments(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid org id")
		return
	}
	payments, err := h.billing.ListPayments(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list payments")
		return
	}
	items := make([]map[string]any, 0, len(payments))
	for _, p := range payments {
		items = append(items, paymentDTO(&p))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *BillingHandler) requireClientOrg(w http.ResponseWriter, r *http.Request) (*auth.Principal, *models.Organization, bool) {
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
	if org.Type != "client_org" || org.ReviewStatus != "active" {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "доступно только для активного кабинета эксплуатации")
		return nil, nil, false
	}
	return p, org, true
}

func (h *BillingHandler) requireVendorOrg(w http.ResponseWriter, r *http.Request) (*auth.Principal, *models.Organization, bool) {
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
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "доступно только для кабинета вендора")
		return nil, nil, false
	}
	if org.ReviewStatus != "active" {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "организация ещё не активирована")
		return nil, nil, false
	}
	return p, org, true
}

func planUpsertFromReq(req planReq) repository.PlanUpsertParams {
	p := repository.PlanUpsertParams{
		OrgType:         req.OrgType,
		Name:            req.Name,
		Slug:            req.Slug,
		PriceMonthlyRub: req.PriceMonthlyRub,
		TicketQuota:     req.TicketQuota,
		OveragePriceRub: req.OveragePriceRub,
		SLAMatrix:       req.SLAMatrix,
		Features:        req.Features,
		IsPublic:        true,
		IsArchived:      false,
	}
	if req.IsPublic != nil {
		p.IsPublic = *req.IsPublic
	}
	if req.IsArchived != nil {
		p.IsArchived = *req.IsArchived
	}
	if req.SortOrder != nil {
		p.SortOrder = *req.SortOrder
	}
	return p
}

func planDTO(p *models.Plan) map[string]any {
	dto := map[string]any{
		"id":                p.ID.String(),
		"org_type":          p.OrgType,
		"name":              p.Name,
		"slug":              p.Slug,
		"price_monthly_rub": p.PriceMonthlyRub,
		"overage_price_rub": p.OveragePriceRub,
		"is_public":         p.IsPublic,
		"is_archived":       p.IsArchived,
		"sort_order":        p.SortOrder,
	}
	if p.TicketQuota != nil {
		dto["ticket_quota"] = *p.TicketQuota
	}
	if len(p.SLAMatrix) > 0 {
		dto["sla_matrix"] = json.RawMessage(p.SLAMatrix)
	}
	if len(p.Features) > 0 {
		dto["features"] = json.RawMessage(p.Features)
	}
	return dto
}

func subscriptionDTO(s *models.Subscription) map[string]any {
	return map[string]any{
		"id":                   s.ID.String(),
		"org_id":               s.OrgID.String(),
		"plan_id":              s.PlanID.String(),
		"status":               s.Status,
		"current_period_start": s.CurrentPeriodStart.Format(time.RFC3339),
		"current_period_end":   s.CurrentPeriodEnd.Format(time.RFC3339),
		"cancel_at_period_end": s.CancelAtPeriodEnd,
		"plan_name":            s.PlanName,
		"plan_slug":            s.PlanSlug,
		"plan_org_type":        s.PlanOrgType,
		"price_monthly_rub":    s.PriceMonthlyRub,
	}
}

func paymentDTO(p *models.Payment) map[string]any {
	dto := map[string]any{
		"id":              p.ID.String(),
		"org_id":          p.OrgID.String(),
		"type":            p.Type,
		"amount_kopecks":  p.AmountKopecks,
		"amount_rub":      p.AmountKopecks / 100,
		"status":          p.Status,
		"note":            p.Note,
		"created_at":      p.CreatedAt.Format(time.RFC3339),
	}
	if p.SubscriptionID != nil {
		dto["subscription_id"] = p.SubscriptionID.String()
	}
	if p.TicketID != nil {
		dto["ticket_id"] = p.TicketID.String()
	}
	if p.InvoiceS3Key != "" {
		dto["invoice_s3_key"] = p.InvoiceS3Key
	}
	return dto
}

func billingSummaryDTO(s *models.BillingSummary) map[string]any {
	plans := make([]map[string]any, 0, len(s.PublicPlans))
	for i := range s.PublicPlans {
		plans = append(plans, planDTO(&s.PublicPlans[i]))
	}
	payments := make([]map[string]any, 0, len(s.RecentPayments))
	for i := range s.RecentPayments {
		payments = append(payments, paymentDTO(&s.RecentPayments[i]))
	}
	dto := map[string]any{
		"plan":                planDTO(&s.Plan),
		"subscription":        subscriptionDTO(&s.Subscription),
		"tickets_used":        s.TicketsUsed,
		"overage_price_rub":   s.OveragePriceRub,
		"period_start":        s.PeriodStart.Format(time.RFC3339),
		"period_end":          s.PeriodEnd.Format(time.RFC3339),
		"recent_payments":     payments,
		"public_plans":        plans,
	}
	if s.TicketQuota != nil {
		dto["ticket_quota"] = *s.TicketQuota
	}
	return dto
}

func ticketQuotaCheckDTO(c *models.TicketQuotaCheck) map[string]any {
	dto := map[string]any{
		"allowed":           c.Allowed,
		"is_overage":        c.IsOverage,
		"tickets_used":      c.TicketsUsed,
		"overage_price_rub": c.OveragePriceRub,
		"plan_name":         c.PlanName,
		"priority":          c.Priority,
	}
	if c.Warning != "" {
		dto["warning"] = c.Warning
	}
	if c.TicketQuota != nil {
		dto["ticket_quota"] = *c.TicketQuota
	}
	return dto
}
