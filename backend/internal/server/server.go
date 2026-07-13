package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	appmw "github.com/n8node/asutport/internal/middleware"
)

type Handlers struct {
	Health    http.Handler
	Auth      AuthHandlers
	Org       OrgHandlers
	AdminOrg  AdminOrgHandlers
	AdminUser AdminUserHandlers
	APIKey    APIKeyHandlers
	Admin     AdminHandlers
	AuthDeps  appmw.AuthDeps
	LoginRL   *appmw.LoginRateLimiter
}

type AuthHandlers struct {
	Register http.HandlerFunc
	Login    http.HandlerFunc
	Refresh  http.HandlerFunc
	Logout   http.HandlerFunc
	Me       http.HandlerFunc
	Switch   http.HandlerFunc
}

type OrgHandlers struct {
	ListMine http.HandlerFunc
	Current  http.HandlerFunc
}

type AdminOrgHandlers struct {
	List         http.HandlerFunc
	Get          http.HandlerFunc
	Patch        http.HandlerFunc
	UpdateReview http.HandlerFunc
}

type AdminUserHandlers struct {
	List           http.HandlerFunc
	Get            http.HandlerFunc
	PatchActive    http.HandlerFunc
	RevokeSessions http.HandlerFunc
}

type APIKeyHandlers struct {
	List   http.HandlerFunc
	Create http.HandlerFunc
	Revoke http.HandlerFunc
}

type AdminHandlers struct {
	S3Get       http.HandlerFunc
	S3Patch     http.HandlerFunc
	S3Test      http.HandlerFunc
	S3CorsHints http.HandlerFunc
	SMTPGet     http.HandlerFunc
	SMTPPatch   http.HandlerFunc
	SMTPTest    http.HandlerFunc
}

type Options struct {
	Logger      *slog.Logger
	Handlers    Handlers
	CORSOrigins []string
}

func New(opts Options) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	if opts.Logger != nil {
		r.Use(appmw.SlogMiddleware(opts.Logger))
	}

	origins := opts.CORSOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	r.Use(appmw.CORS(origins))

	h := opts.Handlers
	r.Get("/health", h.Health.ServeHTTP)

	r.Route("/api", func(r chi.Router) {
		r.Get("/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":"pong"}`))
		})

		r.Route("/v1/auth", func(r chi.Router) {
			if h.LoginRL != nil {
				r.With(appmw.LoginRateLimit(h.LoginRL)).Post("/register", h.Auth.Register)
				r.With(appmw.LoginRateLimit(h.LoginRL)).Post("/login", h.Auth.Login)
			} else {
				r.Post("/register", h.Auth.Register)
				r.Post("/login", h.Auth.Login)
			}
			r.Post("/refresh", h.Auth.Refresh)

			r.Group(func(r chi.Router) {
				r.Use(appmw.Authenticate(h.AuthDeps))
				r.Post("/logout", h.Auth.Logout)
				r.Get("/me", h.Auth.Me)
				r.Post("/switch-org", h.Auth.Switch)
			})
		})

		r.Route("/v1", func(r chi.Router) {
			r.Use(appmw.Authenticate(h.AuthDeps))
			r.Use(appmw.RequireOrgFromToken)

			r.Get("/org", h.Org.Current)
			r.Get("/orgs", h.Org.ListMine)

			r.Group(func(r chi.Router) {
				r.Use(appmw.RequireSuperAdmin)
				r.Get("/admin/orgs", h.AdminOrg.List)
				r.Get("/admin/orgs/{orgID}", h.AdminOrg.Get)
				r.Patch("/admin/orgs/{orgID}", h.AdminOrg.Patch)
				r.Patch("/admin/orgs/{orgID}/review", h.AdminOrg.UpdateReview)
				r.Get("/admin/users", h.AdminUser.List)
				r.Get("/admin/users/{userID}", h.AdminUser.Get)
				r.Patch("/admin/users/{userID}", h.AdminUser.PatchActive)
				r.Post("/admin/users/{userID}/revoke-sessions", h.AdminUser.RevokeSessions)
				r.Get("/admin/settings/s3", h.Admin.S3Get)
				r.Patch("/admin/settings/s3", h.Admin.S3Patch)
				r.Post("/admin/settings/s3/test", h.Admin.S3Test)
				r.Get("/admin/settings/s3/cors-hints", h.Admin.S3CorsHints)
				r.Get("/admin/settings/smtp", h.Admin.SMTPGet)
				r.Patch("/admin/settings/smtp", h.Admin.SMTPPatch)
				r.Post("/admin/settings/smtp/test", h.Admin.SMTPTest)
			})

			r.Route("/orgs/{orgID}/api-keys", func(r chi.Router) {
				r.Get("/", h.APIKey.List)
				r.Post("/", h.APIKey.Create)
				r.Delete("/{keyID}", h.APIKey.Revoke)
			})
		})
	})

	return r
}
