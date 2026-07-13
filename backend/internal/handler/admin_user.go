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
)

type AdminUserHandler struct {
	users *repository.AdminUserRepo
}

func NewAdminUserHandler(users *repository.AdminUserRepo) *AdminUserHandler {
	return &AdminUserHandler{users: users}
}

func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	params, err := repository.ParseAdminUserListParams(r.URL.Query())
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query parameters")
		return
	}
	list, total, err := h.users.List(r.Context(), params)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list users")
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, row := range list {
		items = append(items, adminUserListDTO(row))
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

func (h *AdminUserHandler) Get(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user id")
		return
	}
	row, sessions, err := h.users.GetDetail(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load user")
		return
	}
	dto := adminUserListDTO(*row)
	dto["sessions"] = adminSessionDTOs(sessions)
	WriteJSON(w, http.StatusOK, map[string]any{"data": dto})
}

type patchUserActiveReq struct {
	IsActive bool `json:"is_active"`
}

func (h *AdminUserHandler) PatchActive(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user id")
		return
	}
	var req patchUserActiveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	u, err := h.users.SetActive(r.Context(), userID, req.IsActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not update user")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":        u.ID.String(),
			"is_active": u.IsActive,
		},
	})
}

func (h *AdminUserHandler) RevokeSessions(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid user id")
		return
	}
	n, err := h.users.RevokeAllSessions(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not revoke sessions")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"revoked": n,
		},
	})
}

func requireSuperAdmin(w http.ResponseWriter, r *http.Request) bool {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok || !p.IsSuperAdmin() {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "superadmin only")
		return false
	}
	return true
}

func adminUserListDTO(row models.AdminUserListRow) map[string]any {
	return map[string]any{
		"id":              row.ID.String(),
		"email":           row.Email,
		"full_name":       row.FullName,
		"is_active":       row.IsActive,
		"access_level":    row.AccessLevel,
		"created_at":      row.CreatedAt,
		"updated_at":      row.UpdatedAt,
		"last_login_at":   row.LastLoginAt,
		"active_sessions": row.ActiveSessions,
		"last_ip":         row.LastIP,
		"last_user_agent": row.LastUserAgent,
		"memberships":     adminMembershipDTOs(row.Memberships),
		"messengers":      adminMessengerDTOs(row.Messengers),
	}
}

func adminMembershipDTOs(list []models.AdminUserMembership) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		out = append(out, map[string]any{
			"org_id":        m.OrgID.String(),
			"org_name":      m.OrgName,
			"org_type":      m.OrgType,
			"org_slug":      m.OrgSlug,
			"role":          m.Role,
			"review_status": m.ReviewStatus,
			"is_personal":   m.IsPersonal,
			"org_is_active": m.OrgIsActive,
			"inn":           m.INN,
			"website":       m.Website,
			"contact_phone": m.ContactPhone,
			"member_since":  m.MemberSince,
		})
	}
	return out
}

func adminMessengerDTOs(list []models.UserMessengerLink) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		out = append(out, map[string]any{
			"id":                    m.ID.String(),
			"provider":              m.Provider,
			"external_user_id":      m.ExternalUserID,
			"username":              m.Username,
			"display_name":          m.DisplayName,
			"is_verified":           m.IsVerified,
			"notifications_enabled": m.NotificationsEnabled,
			"linked_at":             m.LinkedAt,
			"created_at":            m.CreatedAt,
		})
	}
	return out
}

func adminSessionDTOs(list []models.AdminUserSession) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, s := range list {
		active := s.RevokedAt == nil && s.ExpiresAt.After(time.Now().UTC())
		out = append(out, map[string]any{
			"id":         s.ID.String(),
			"org_id":     s.OrgID.String(),
			"org_name":   s.OrgName,
			"ip_address": s.IPAddress,
			"user_agent": shortenUA(s.UserAgent),
			"expires_at": s.ExpiresAt,
			"revoked_at": s.RevokedAt,
			"created_at": s.CreatedAt,
			"is_active":  active && s.RevokedAt == nil,
		})
	}
	return out
}

func shortenUA(ua string) string {
	ua = strings.TrimSpace(ua)
	if len(ua) <= 120 {
		return ua
	}
	return ua[:117] + "..."
}
