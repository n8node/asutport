package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
)

type AdminOrgHandler struct {
	orgs    *repository.AdminOrgRepo
	orgBase *repository.OrgRepo
}

func NewAdminOrgHandler(orgs *repository.AdminOrgRepo, orgBase *repository.OrgRepo) *AdminOrgHandler {
	return &AdminOrgHandler{orgs: orgs, orgBase: orgBase}
}

func (h *AdminOrgHandler) List(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	params, err := repository.ParseAdminOrgListParams(r.URL.Query())
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query parameters")
		return
	}
	list, total, err := h.orgs.List(r.Context(), params)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list organizations")
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, row := range list {
		items = append(items, adminOrgDTO(row, nil))
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{
			"total":  total,
			"limit":  params.Limit,
			"offset": params.Offset,
		},
	})
}

func (h *AdminOrgHandler) Get(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid org id")
		return
	}
	row, members, err := h.orgs.GetDetail(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return
	}
	dto := adminOrgDTO(*row, members)
	WriteJSON(w, http.StatusOK, map[string]any{"data": dto})
}

type patchOrgReq struct {
	IsActive *bool `json:"is_active"`
}

func (h *AdminOrgHandler) Patch(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid org id")
		return
	}
	var req patchOrgReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	if req.IsActive == nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "is_active is required")
		return
	}
	org, err := h.orgs.SetActive(r.Context(), orgID, *req.IsActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not update organization")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": orgDTO(*org)})
}

func (h *AdminOrgHandler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok || !p.IsSuperAdmin() {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "superadmin only")
		return
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid org id")
		return
	}
	var req updateReviewReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	if !validReviewStatus(req.Status) {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid status")
		return
	}
	if err := h.orgBase.UpdateReviewStatus(r.Context(), orgID, p.UserID, req.Status); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not update organization")
		return
	}
	row, members, err := h.orgs.GetDetail(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": adminOrgDTO(*row, members)})
}

func adminOrgDTO(row models.AdminOrgListRow, members []models.AdminOrgMember) map[string]any {
	dto := map[string]any{
		"id":             row.ID.String(),
		"name":           row.Name,
		"type":           row.Type,
		"slug":           row.Slug,
		"is_active":      row.IsActive,
		"legal_name":     row.LegalName,
		"inn":            row.INN,
		"website":        row.Website,
		"contact_phone":  row.ContactPhone,
		"review_comment": row.ReviewComment,
		"is_personal":    row.IsPersonal,
		"review_status":  row.ReviewStatus,
		"reviewed_at":    row.ReviewedAt,
		"reviewed_by":    row.ReviewedBy,
		"created_at":     row.CreatedAt,
		"updated_at":     row.UpdatedAt,
		"member_count":   row.MemberCount,
		"metrics":        row.Metrics,
		"onboarding_stage": onboardingStage(row),
	}
	if row.Owner != nil {
		dto["owner"] = map[string]any{
			"user_id":   row.Owner.UserID.String(),
			"email":     row.Owner.Email,
			"full_name": row.Owner.FullName,
			"role":      row.Owner.Role,
		}
	}
	if members != nil {
		items := make([]map[string]any, 0, len(members))
		for _, m := range members {
			items = append(items, map[string]any{
				"user_id":    m.UserID.String(),
				"email":      m.Email,
				"full_name":  m.FullName,
				"role":       m.Role,
				"is_active":  m.IsActive,
				"created_at": m.CreatedAt,
			})
		}
		dto["members"] = items
	}
	return dto
}

func onboardingStage(row models.AdminOrgListRow) string {
	switch row.Type {
	case "manufacturer":
		switch row.ReviewStatus {
		case "pending_review", "pending_email":
			return "review"
		case "rejected", "suspended":
			return row.ReviewStatus
		case "active":
			if !row.Metrics.SupportZoneLoaded {
				return "onboarding"
			}
			if !row.Metrics.GoldenSetReady {
				return "golden"
			}
			return "active"
		}
	case "vendor", "integrator":
		if row.ReviewStatus == "active" {
			return "active"
		}
		return row.ReviewStatus
	case "client_org":
		if row.ReviewStatus == "active" {
			return "active"
		}
		return row.ReviewStatus
	}
	return row.ReviewStatus
}
