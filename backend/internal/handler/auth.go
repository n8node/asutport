package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
	"github.com/n8node/asutport/internal/service"
)

type AuthHandler struct {
	users    *repository.UserRepo
	orgs     *repository.OrgRepo
	members  *repository.OrgMemberRepo
	sessions *repository.SessionRepo
	authSvc  *service.AuthService
}

func NewAuthHandler(
	users *repository.UserRepo,
	orgs *repository.OrgRepo,
	members *repository.OrgMemberRepo,
	sessions *repository.SessionRepo,
	authSvc *service.AuthService,
) *AuthHandler {
	return &AuthHandler{
		users:    users,
		orgs:     orgs,
		members:  members,
		sessions: sessions,
		authSvc:  authSvc,
	}
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	OrgName  string `json:"org_name"`
}

type loginReq struct {
	Email    string     `json:"email"`
	Password string     `json:"password"`
	OrgID    *uuid.UUID `json:"org_id"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

type switchOrgReq struct {
	OrgID uuid.UUID `json:"org_id"`
}

type tokenEnvelope struct {
	Data tokenData `json:"data"`
}

type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	OrgID        string `json:"org_id"`
	Role         string `json:"role"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if err := validateEmail(req.Email); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		fullName = strings.Split(req.Email, "@")[0]
	}
	orgName := strings.TrimSpace(req.OrgName)
	if orgName == "" {
		orgName = fullName
	}

	u, err := h.users.Create(r.Context(), req.Email, hash, fullName)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			WriteError(w, http.StatusConflict, "CONFLICT", "email already registered")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "registration failed")
		return
	}

	slugBase := service.SanitizeSlug(strings.Split(req.Email, "@")[0])
	org, err := h.orgs.Create(r.Context(), orgName, "client_org", service.UniqueSlug(slugBase))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not create organization")
		return
	}
	member, err := h.members.Create(r.Context(), org.ID, u.ID, "owner")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not create membership")
		return
	}

	mem := &models.OrgMembership{
		OrgMember: *member,
		OrgName:   org.Name,
		OrgType:   org.Type,
		OrgSlug:   org.Slug,
	}
	pair, err := h.authSvc.IssueForMembership(r.Context(), u, mem, r.UserAgent(), ClientIP(r))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not issue token")
		return
	}
	writeToken(w, http.StatusCreated, pair)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	u, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "login failed")
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.Password) {
		WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		return
	}
	if !u.IsActive {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "account is inactive")
		return
	}

	var mem *models.OrgMembership
	if req.OrgID != nil {
		m, err := h.members.GetMembership(r.Context(), *req.OrgID, u.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				WriteError(w, http.StatusForbidden, "FORBIDDEN", "not a member of this organization")
				return
			}
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "login failed")
			return
		}
		org, err := h.orgs.GetByID(r.Context(), m.OrgID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "login failed")
			return
		}
		mem = &models.OrgMembership{OrgMember: *m, OrgName: org.Name, OrgType: org.Type, OrgSlug: org.Slug}
	} else {
		mem, err = h.members.FirstMembership(r.Context(), u.ID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				WriteError(w, http.StatusForbidden, "FORBIDDEN", "no organization membership")
				return
			}
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "login failed")
			return
		}
	}

	pair, err := h.authSvc.IssueForMembership(r.Context(), u, mem, r.UserAgent(), ClientIP(r))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not issue token")
		return
	}
	writeToken(w, http.StatusOK, pair)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "refresh_token is required")
		return
	}
	pair, err := h.authSvc.Refresh(r.Context(), req.RefreshToken, r.UserAgent(), ClientIP(r))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired refresh token")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "refresh failed")
		return
	}
	writeToken(w, http.StatusOK, pair)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	if err := h.sessions.Revoke(r.Context(), p.SessionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "session not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "logout failed")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "ok"}})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	u, err := h.users.GetByID(r.Context(), p.UserID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load user")
		return
	}
	org, err := h.orgs.GetByID(r.Context(), p.OrgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return
	}
	memberships, _ := h.members.ListByUser(r.Context(), p.UserID)
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"user": map[string]any{
				"id":         u.ID.String(),
				"email":      u.Email,
				"full_name":  u.FullName,
				"is_active":  u.IsActive,
			},
			"org": map[string]any{
				"id":   org.ID.String(),
				"name": org.Name,
				"type": org.Type,
				"slug": org.Slug,
				"role": p.Role,
			},
			"memberships": membershipDTOs(memberships),
		},
	})
}

func (h *AuthHandler) SwitchOrg(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return
	}
	var req switchOrgReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	u, err := h.users.GetByID(r.Context(), p.UserID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "switch failed")
		return
	}
	m, err := h.members.GetMembership(r.Context(), req.OrgID, p.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusForbidden, "FORBIDDEN", "not a member of this organization")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "switch failed")
		return
	}
	org, err := h.orgs.GetByID(r.Context(), req.OrgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "switch failed")
		return
	}
	_ = h.sessions.Revoke(r.Context(), p.SessionID)
	mem := &models.OrgMembership{OrgMember: *m, OrgName: org.Name, OrgType: org.Type, OrgSlug: org.Slug}
	pair, err := h.authSvc.IssueForMembership(r.Context(), u, mem, r.UserAgent(), ClientIP(r))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not issue token")
		return
	}
	writeToken(w, http.StatusOK, pair)
}

func writeToken(w http.ResponseWriter, status int, pair *service.TokenPair) {
	var res tokenEnvelope
	res.Data.AccessToken = pair.AccessToken
	res.Data.RefreshToken = pair.RefreshToken
	res.Data.TokenType = "Bearer"
	res.Data.ExpiresIn = pair.ExpiresIn
	res.Data.OrgID = pair.OrgID.String()
	res.Data.Role = pair.Role
	WriteJSON(w, status, res)
}

func membershipDTOs(list []models.OrgMembership) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		out = append(out, map[string]any{
			"org_id":   m.OrgID.String(),
			"org_name": m.OrgName,
			"org_type": m.OrgType,
			"org_slug": m.OrgSlug,
			"role":     m.Role,
		})
	}
	return out
}

func validateEmail(s string) error {
	if s == "" {
		return validationErr("email is required")
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s {
		return validationErr("invalid email")
	}
	return nil
}

type validationErr string

func (e validationErr) Error() string { return string(e) }
