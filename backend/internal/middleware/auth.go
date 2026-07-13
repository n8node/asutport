package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/config"
	"github.com/n8node/asutport/internal/repository"
)

type AuthDeps struct {
	Cfg      *config.Config
	Users    *repository.UserRepo
	Sessions *repository.SessionRepo
	Members  *repository.OrgMemberRepo
	Keys     *repository.APIKeyRepo
}

func Authenticate(d AuthDeps) func(http.Handler) http.Handler {
	secret := []byte(d.Cfg.JWTSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if p, ok := tryBearer(r, secret, d); ok {
				next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
				return
			}
			if d.Cfg.APIKeySalt != "" && d.Keys != nil {
				if p, ok := tryAPIKey(r, d); ok {
					next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
					return
				}
			}
			writeJSONErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid authentication")
		})
	}
}

func RequireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.PrincipalFromContext(r.Context())
		if !ok || !p.IsSuperAdmin() {
			writeJSONErr(w, http.StatusForbidden, "FORBIDDEN", "superadmin only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireOrgFromToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			writeJSONErr(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
			return
		}
		if p.OrgID == uuid.Nil {
			writeJSONErr(w, http.StatusForbidden, "FORBIDDEN", "organization context required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func tryBearer(r *http.Request, secret []byte, d AuthDeps) (*auth.Principal, bool) {
	h := r.Header.Get("Authorization")
	const pfx = "Bearer "
	if !strings.HasPrefix(h, pfx) {
		return nil, false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(h, pfx))
	if raw == "" || len(secret) < 16 {
		return nil, false
	}
	claims, err := auth.ParseJWT(secret, raw)
	if err != nil {
		return nil, false
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, false
	}
	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		return nil, false
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, false
	}
	u, err := d.Users.GetByID(r.Context(), userID)
	if err != nil || !u.IsActive {
		return nil, false
	}
	if _, err := d.Sessions.GetActiveByID(r.Context(), sessionID); err != nil {
		return nil, false
	}
	if _, err := d.Members.GetMembership(r.Context(), orgID, userID); err != nil {
		return nil, false
	}
	return &auth.Principal{
		UserID:    userID,
		Email:     u.Email,
		OrgID:     orgID,
		Role:      claims.Role,
		SessionID: sessionID,
	}, true
}

func tryAPIKey(r *http.Request, d AuthDeps) (*auth.Principal, bool) {
	raw := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if raw == "" || len(raw) < 16 {
		return nil, false
	}
	prefix := raw[:16]
	row, err := d.Keys.FindActiveByPrefix(r.Context(), prefix)
	if err != nil {
		return nil, false
	}
	want := auth.HashAPIKey(d.Cfg.APIKeySalt, raw)
	if subtle.ConstantTimeCompare([]byte(want), []byte(row.KeyHash)) != 1 {
		return nil, false
	}
	_ = d.Keys.Touch(r.Context(), row.ID)
	member, err := d.Members.PrimaryAdminForOrg(r.Context(), row.OrgID)
	if err != nil {
		return nil, false
	}
	u, err := d.Users.GetByID(r.Context(), member.UserID)
	if err != nil || !u.IsActive {
		return nil, false
	}
	return &auth.Principal{
		UserID: member.UserID,
		Email:  u.Email,
		OrgID:  row.OrgID,
		Role:   member.Role,
	}, true
}

func writeJSONErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": msg},
	})
}

type LoginRateLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	limit   int
	window  time.Duration
}

func NewLoginRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{
		entries: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}
}

func (l *LoginRateLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-l.window)
	var kept []time.Time
	for _, t := range l.entries[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.entries[key] = kept
		return false
	}
	l.entries[key] = append(kept, now)
	return true
}

func LoginRateLimit(l *LoginRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.Header.Get("X-Real-IP")
			if ip == "" {
				ip = r.RemoteAddr
			}
			if !l.Allow(ip) {
				writeJSONErr(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many attempts, try again later")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
