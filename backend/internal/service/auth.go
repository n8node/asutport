package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
)

const (
	AccessTokenTTL  = 30 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

type AuthService struct {
	jwtSecret []byte
	users     *repository.UserRepo
	members   *repository.OrgMemberRepo
	sessions  *repository.SessionRepo
}

func NewAuthService(secret string, users *repository.UserRepo, members *repository.OrgMemberRepo, sessions *repository.SessionRepo) *AuthService {
	return &AuthService{
		jwtSecret: []byte(secret),
		users:     users,
		members:   members,
		sessions:  sessions,
	}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	SessionID    uuid.UUID
	OrgID        uuid.UUID
	Role         string
	OrgType      string
	ReviewStatus string
}

func (s *AuthService) IssueForMembership(ctx context.Context, u *models.User, m *models.OrgMembership, userAgent, ip string) (*TokenPair, error) {
	if !u.IsActive {
		return nil, fmt.Errorf("account inactive")
	}
	rawRefresh, refreshHash, err := auth.NewRefreshToken()
	if err != nil {
		return nil, err
	}
	sess, err := s.sessions.Create(ctx, u.ID, m.OrgID, refreshHash, userAgent, ip, time.Now().UTC().Add(RefreshTokenTTL))
	if err != nil {
		return nil, err
	}
	access, err := auth.SignJWT(s.jwtSecret, u.ID, m.OrgID, sess.ID, u.Email, m.Role, AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(AccessTokenTTL.Seconds()),
		SessionID:    sess.ID,
		OrgID:        m.OrgID,
		Role:         m.Role,
		OrgType:      m.OrgType,
		ReviewStatus: m.OrgReviewStatus,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (*TokenPair, error) {
	hash := auth.HashToken(refreshToken)
	sess, err := s.sessions.GetActiveByRefreshHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	u, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, fmt.Errorf("account inactive")
	}
	m, err := s.members.GetMembership(ctx, sess.OrgID, sess.UserID)
	if err != nil {
		return nil, err
	}
	rawRefresh, refreshHash, err := auth.NewRefreshToken()
	if err != nil {
		return nil, err
	}
	newSess, err := s.sessions.Rotate(ctx, sess.ID, refreshHash, time.Now().UTC().Add(RefreshTokenTTL))
	if err != nil {
		return nil, err
	}
	_ = userAgent
	_ = ip
	access, err := auth.SignJWT(s.jwtSecret, u.ID, m.OrgID, newSess.ID, u.Email, m.Role, AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(AccessTokenTTL.Seconds()),
		SessionID:    newSess.ID,
		OrgID:        m.OrgID,
		Role:         m.Role,
	}, nil
}

func SanitizeSlug(base string) string {
	base = strings.ToLower(strings.TrimSpace(base))
	var b strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "org"
	}
	if len(out) > 48 {
		out = out[:48]
	}
	return out
}

func UniqueSlug(base string) string {
	suffix := strings.ReplaceAll(uuid.New().String()[:8], "-", "")
	return SanitizeSlug(base) + "-" + suffix
}
