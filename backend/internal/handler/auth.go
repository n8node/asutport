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
	Email         string `json:"email"`
	Password      string `json:"password"`
	FullName      string `json:"full_name"`
	AccountType   string `json:"account_type"`
	OrgName       string `json:"org_name"`
	LegalName     string `json:"legal_name"`
	INN           string `json:"inn"`
	Website       string `json:"website"`
	ContactPhone  string `json:"contact_phone"`
	ReviewComment string `json:"review_comment"`
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
	OrgType      string `json:"org_type"`
	ReviewStatus string `json:"review_status"`
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
	accountType, orgType, isPersonal, reviewStatus, err := registrationOrgType(req.AccountType)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if orgName == "" && !isPersonal {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "organization name is required")
		return
	}
	inn := strings.TrimSpace(req.INN)
	if requiresINN(accountType) && inn == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "inn is required")
		return
	}
	if orgName == "" {
		orgName = fullName
	}
	legalName := strings.TrimSpace(req.LegalName)
	if legalName == "" {
		legalName = orgName
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

	slugBase := orgName
	if isPersonal {
		slugBase = strings.Split(req.Email, "@")[0]
	}
	org, err := h.orgs.CreateWithReview(r.Context(), repository.OrgCreateParams{
		Name:          orgName,
		Type:          orgType,
		Slug:          service.UniqueSlug(slugBase),
		LegalName:     legalName,
		INN:           inn,
		Website:       strings.TrimSpace(req.Website),
		ContactPhone:  strings.TrimSpace(req.ContactPhone),
		ReviewComment: strings.TrimSpace(req.ReviewComment),
		IsPersonal:    isPersonal,
		ReviewStatus:  reviewStatus,
	})
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
		OrgMember:       *member,
		OrgName:         org.Name,
		OrgType:         org.Type,
		OrgSlug:         org.Slug,
		OrgReviewStatus: org.ReviewStatus,
		OrgIsPersonal:   org.IsPersonal,
	}
	pair, err := h.authSvc.IssueForMembership(r.Context(), u, mem, r.UserAgent(), ClientIP(r))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not issue token")
		return
	}
	writeToken(w, http.StatusCreated, pair)
}

func requiresINN(accountType string) bool {
	switch accountType {
	case "manufacturer", "vendor", "integrator":
		return true
	default:
		return false
	}
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
		mem = &models.OrgMembership{
			OrgMember:       *m,
			OrgName:         org.Name,
			OrgType:         org.Type,
			OrgSlug:         org.Slug,
			OrgReviewStatus: org.ReviewStatus,
			OrgIsPersonal:   org.IsPersonal,
		}
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
				"id":        u.ID.String(),
				"email":     u.Email,
				"full_name": u.FullName,
				"is_active": u.IsActive,
			},
			"org": map[string]any{
				"id":            org.ID.String(),
				"name":          org.Name,
				"type":          org.Type,
				"slug":          org.Slug,
				"role":          p.Role,
				"review_status": org.ReviewStatus,
				"is_personal":   org.IsPersonal,
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
	mem := &models.OrgMembership{
		OrgMember:       *m,
		OrgName:         org.Name,
		OrgType:         org.Type,
		OrgSlug:         org.Slug,
		OrgReviewStatus: org.ReviewStatus,
		OrgIsPersonal:   org.IsPersonal,
	}
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
	res.Data.OrgType = pair.OrgType
	res.Data.ReviewStatus = pair.ReviewStatus
	WriteJSON(w, status, res)
}

func membershipDTOs(list []models.OrgMembership) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		out = append(out, map[string]any{
			"org_id":        m.OrgID.String(),
			"org_name":      m.OrgName,
			"org_type":      m.OrgType,
			"org_slug":      m.OrgSlug,
			"role":          m.Role,
			"review_status": m.OrgReviewStatus,
			"is_personal":   m.OrgIsPersonal,
		})
	}
	return out
}

func registrationOrgType(raw string) (accountType, orgType string, isPersonal bool, reviewStatus string, err error) {
	accountType = strings.TrimSpace(raw)
	if accountType == "" {
		accountType = "client_personal"
	}
	switch accountType {
	case "client_personal":
		return accountType, "client_org", true, "active", nil
	case "client_org":
		return accountType, "client_org", false, "active", nil
	case "manufacturer":
		return accountType, "manufacturer", false, "pending_review", nil
	case "vendor":
		return accountType, "vendor", false, "pending_review", nil
	case "integrator":
		return accountType, "integrator", false, "pending_review", nil
	default:
		return "", "", false, "", validationErr("invalid account_type")
	}
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
