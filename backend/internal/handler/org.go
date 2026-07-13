package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/config"
	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
)

type APIKeyHandler struct {
	cfg   *config.Config
	keys  *repository.APIKeyRepo
	membs *repository.OrgMemberRepo
}

func NewAPIKeyHandler(cfg *config.Config, keys *repository.APIKeyRepo, membs *repository.OrgMemberRepo) *APIKeyHandler {
	return &APIKeyHandler{cfg: cfg, keys: keys, membs: membs}
}

type createAPIKeyReq struct {
	Name string `json:"name"`
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid org id")
		return
	}
	if err := h.requireOrgAdmin(r, orgID); err != nil {
		writeRepoAuthErr(w, err)
		return
	}
	keys, err := h.keys.ListByOrg(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list api keys")
		return
	}
	items := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		items = append(items, map[string]any{
			"id":           k.ID.String(),
			"name":         k.Name,
			"key_prefix":   k.KeyPrefix,
			"last_used_at": k.LastUsedAt,
			"created_at":   k.CreatedAt,
		})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid org id")
		return
	}
	if err := h.requireOrgAdmin(r, orgID); err != nil {
		writeRepoAuthErr(w, err)
		return
	}
	var req createAPIKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	raw, prefix, err := auth.NewAPIKeyRaw()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not generate api key")
		return
	}
	hash := auth.HashAPIKey(h.cfg.APIKeySalt, raw)
	k, err := h.keys.Create(r.Context(), orgID, req.Name, hash, prefix)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not create api key")
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{
		"data": map[string]any{
			"id":         k.ID.String(),
			"name":       k.Name,
			"key_prefix": k.KeyPrefix,
			"api_key":    raw,
			"created_at": k.CreatedAt,
		},
	})
}

func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid org id")
		return
	}
	keyID, err := uuid.Parse(chi.URLParam(r, "keyID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid key id")
		return
	}
	if err := h.requireOrgAdmin(r, orgID); err != nil {
		writeRepoAuthErr(w, err)
		return
	}
	if err := h.keys.Revoke(r.Context(), orgID, keyID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "api key not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not revoke api key")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "ok"}})
}

func (h *APIKeyHandler) requireOrgAdmin(r *http.Request, orgID uuid.UUID) error {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		return errForbidden("missing authentication")
	}
	if p.IsSuperAdmin() {
		return nil
	}
	if p.OrgID != orgID {
		return errForbidden("organization mismatch")
	}
	if p.Role != "owner" && p.Role != "admin" {
		return errForbidden("admin role required")
	}
	return nil
}

type OrgHandler struct {
	members *repository.OrgMemberRepo
	orgs    *repository.OrgRepo
}

func NewOrgHandler(members *repository.OrgMemberRepo, orgs *repository.OrgRepo) *OrgHandler {
	return &OrgHandler{members: members, orgs: orgs}
}

func (h *OrgHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	list, err := h.members.ListByUser(r.Context(), p.UserID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list organizations")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": membershipDTOs(list)})
}

func (h *OrgHandler) Current(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	org, err := h.orgs.GetByID(r.Context(), p.OrgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":             org.ID.String(),
			"name":           org.Name,
			"type":           org.Type,
			"slug":           org.Slug,
			"role":           p.Role,
			"legal_name":     org.LegalName,
			"inn":            org.INN,
			"website":        org.Website,
			"contact_phone":  org.ContactPhone,
			"review_comment": org.ReviewComment,
			"is_personal":    org.IsPersonal,
			"review_status":  org.ReviewStatus,
		},
	})
}

func (h *OrgHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok || !p.IsSuperAdmin() {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "superadmin only")
		return
	}
	list, err := h.orgs.List(r.Context(), repository.OrgListParams{
		ReviewStatus: strings.TrimSpace(r.URL.Query().Get("review_status")),
		Type:         strings.TrimSpace(r.URL.Query().Get("type")),
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list organizations")
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, org := range list {
		items = append(items, orgDTO(org))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

type updateReviewReq struct {
	Status string `json:"status"`
}

func (h *OrgHandler) AdminUpdateReview(w http.ResponseWriter, r *http.Request) {
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
	if err := h.orgs.UpdateReviewStatus(r.Context(), orgID, p.UserID, req.Status); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not update organization")
		return
	}
	org, err := h.orgs.GetByID(r.Context(), orgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": orgDTO(*org)})
}

func validReviewStatus(status string) bool {
	switch status {
	case "pending_email", "pending_review", "active", "rejected", "suspended":
		return true
	default:
		return false
	}
}

func orgDTO(org models.Organization) map[string]any {
	return map[string]any{
		"id":             org.ID.String(),
		"name":           org.Name,
		"type":           org.Type,
		"slug":           org.Slug,
		"is_active":      org.IsActive,
		"legal_name":     org.LegalName,
		"inn":            org.INN,
		"website":        org.Website,
		"contact_phone":  org.ContactPhone,
		"review_comment": org.ReviewComment,
		"is_personal":    org.IsPersonal,
		"review_status":  org.ReviewStatus,
		"reviewed_at":    org.ReviewedAt,
		"reviewed_by":    org.ReviewedBy,
		"created_at":     org.CreatedAt,
	}
}

type forbiddenErr string

func errForbidden(msg string) error { return forbiddenErr(msg) }

func (e forbiddenErr) Error() string { return string(e) }

func writeRepoAuthErr(w http.ResponseWriter, err error) {
	var fe forbiddenErr
	if errors.As(err, &fe) {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", string(fe))
		return
	}
	WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
}
