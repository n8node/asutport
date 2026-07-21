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
	Ticket    TicketHandlers
	Client    ClientHandlers
	Vendor    VendorHandlers
	AdminOrg  AdminOrgHandlers
	AdminUser AdminUserHandlers
	APIKey    APIKeyHandlers
	Admin     AdminHandlers
	Billing   BillingHandlers
	Docs      DocsHandlers
	AuthDeps  appmw.AuthDeps
	LoginRL   *appmw.LoginRateLimiter
}

type AuthHandlers struct {
	Register           http.HandlerFunc
	Login              http.HandlerFunc
	VerifyRegistration http.HandlerFunc
	Refresh            http.HandlerFunc
	Logout             http.HandlerFunc
	Me                 http.HandlerFunc
	Switch             http.HandlerFunc
}

type OrgHandlers struct {
	ListMine http.HandlerFunc
	Current  http.HandlerFunc
}

type TicketHandlers struct {
	GetOnboarding       http.HandlerFunc
	Get                 http.HandlerFunc
	ListEvents          http.HandlerFunc
	PostMessage         http.HandlerFunc
	PresignAttachment   http.HandlerFunc
	UploadAttachment    http.HandlerFunc
	CompleteAttachment  http.HandlerFunc
	AttachmentURL       http.HandlerFunc
	Resolve             http.HandlerFunc
	ListOnboardingAdmin http.HandlerFunc
	ApproveOrg          http.HandlerFunc
	RejectOrg           http.HandlerFunc
}

type ClientHandlers struct {
	Dashboard          http.HandlerFunc
	ListInstallations  http.HandlerFunc
	CreateInstallation http.HandlerFunc
	UpdateInstallation http.HandlerFunc
	ListProducts       http.HandlerFunc
	CreateProduct      http.HandlerFunc
	UpdateProduct      http.HandlerFunc
	DeleteProduct      http.HandlerFunc
	ListSupplyRecords  http.HandlerFunc
	CreateSupplyRecord http.HandlerFunc
	DeleteSupplyRecord http.HandlerFunc
	ListTickets        http.HandlerFunc
	CreateTicket       http.HandlerFunc
}

type VendorHandlers struct {
	Dashboard   http.HandlerFunc
	ListTickets http.HandlerFunc
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
	Delete         http.HandlerFunc
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
	LLMGet      http.HandlerFunc
	LLMPatch    http.HandlerFunc
	LLMTest     http.HandlerFunc
	LLMModels   http.HandlerFunc
}

type DocsHandlers struct {
	CreateProduct http.HandlerFunc
	ListProducts  http.HandlerFunc
	ListSources   http.HandlerFunc
	GetSource     http.HandlerFunc
	Upload        http.HandlerFunc
	Reprocess     http.HandlerFunc
	Search        http.HandlerFunc
	OriginalURL   http.HandlerFunc
	PageURL       http.HandlerFunc
}

type BillingHandlers struct {
	ClientSummary         http.HandlerFunc
	ClientQuotaCheck      http.HandlerFunc
	VendorSummary         http.HandlerFunc
	AdminOverview         http.HandlerFunc
	AdminListPlans        http.HandlerFunc
	AdminCreatePlan       http.HandlerFunc
	AdminUpdatePlan       http.HandlerFunc
	AdminAssignSubscription http.HandlerFunc
	AdminRecordPayment    http.HandlerFunc
	AdminListPayments     http.HandlerFunc
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
			r.Get("/verify-registration", h.Auth.VerifyRegistration)

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

			r.Get("/tickets/onboarding", h.Ticket.GetOnboarding)
			r.Route("/tickets/{ticketID}", func(r chi.Router) {
				r.Get("/", h.Ticket.Get)
				r.Get("/events", h.Ticket.ListEvents)
				r.Post("/messages", h.Ticket.PostMessage)
				r.Post("/resolve", h.Ticket.Resolve)
				r.Post("/attachments/presign", h.Ticket.PresignAttachment)
				r.Post("/attachments/upload", h.Ticket.UploadAttachment)
				r.Post("/attachments/{attachmentID}/complete", h.Ticket.CompleteAttachment)
				r.Get("/attachments/{attachmentID}/url", h.Ticket.AttachmentURL)
			})

			r.Route("/client", func(r chi.Router) {
				r.Get("/dashboard", h.Client.Dashboard)
				r.Get("/installations", h.Client.ListInstallations)
				r.Post("/installations", h.Client.CreateInstallation)
				r.Patch("/installations/{installationID}", h.Client.UpdateInstallation)
				r.Get("/installations/{installationID}/products", h.Client.ListProducts)
				r.Post("/installations/{installationID}/products", h.Client.CreateProduct)
				r.Patch("/products/{productID}", h.Client.UpdateProduct)
				r.Delete("/products/{productID}", h.Client.DeleteProduct)
				r.Get("/supply-records", h.Client.ListSupplyRecords)
				r.Post("/supply-records", h.Client.CreateSupplyRecord)
				r.Delete("/supply-records/{recordID}", h.Client.DeleteSupplyRecord)
				r.Get("/tickets", h.Client.ListTickets)
				r.Post("/tickets", h.Client.CreateTicket)
				r.Get("/billing", h.Billing.ClientSummary)
				r.Get("/billing/quota-check", h.Billing.ClientQuotaCheck)
			})

			r.Route("/vendor", func(r chi.Router) {
				r.Get("/dashboard", h.Vendor.Dashboard)
				r.Get("/tickets", h.Vendor.ListTickets)
				r.Get("/billing", h.Billing.VendorSummary)
				r.Get("/products", h.Docs.ListProducts)
				r.Post("/products", h.Docs.CreateProduct)
				r.Get("/docs", h.Docs.ListSources)
				r.Post("/docs/upload", h.Docs.Upload)
				r.Get("/docs/{sourceID}", h.Docs.GetSource)
				r.Post("/docs/{sourceID}/reprocess", h.Docs.Reprocess)
				r.Get("/docs/{sourceID}/original-url", h.Docs.OriginalURL)
				r.Get("/docs/{sourceID}/pages/{page}/url", h.Docs.PageURL)
				r.Post("/rag/search", h.Docs.Search)
			})

			r.Post("/rag/search", h.Docs.Search)
			r.Get("/sources/{sourceID}/pages/{page}/url", h.Docs.PageURL)

			r.Group(func(r chi.Router) {
				r.Use(appmw.RequireSuperAdmin)
				r.Get("/admin/tickets/onboarding", h.Ticket.ListOnboardingAdmin)
				r.Post("/admin/tickets/{ticketID}/approve-org", h.Ticket.ApproveOrg)
				r.Post("/admin/tickets/{ticketID}/reject-org", h.Ticket.RejectOrg)
				r.Get("/admin/orgs", h.AdminOrg.List)
				r.Get("/admin/orgs/{orgID}", h.AdminOrg.Get)
				r.Patch("/admin/orgs/{orgID}", h.AdminOrg.Patch)
				r.Patch("/admin/orgs/{orgID}/review", h.AdminOrg.UpdateReview)
				r.Get("/admin/billing/overview", h.Billing.AdminOverview)
				r.Get("/admin/plans", h.Billing.AdminListPlans)
				r.Post("/admin/plans", h.Billing.AdminCreatePlan)
				r.Patch("/admin/plans/{planID}", h.Billing.AdminUpdatePlan)
				r.Post("/admin/orgs/{orgID}/subscription", h.Billing.AdminAssignSubscription)
				r.Get("/admin/orgs/{orgID}/payments", h.Billing.AdminListPayments)
				r.Post("/admin/payments", h.Billing.AdminRecordPayment)
				r.Get("/admin/users", h.AdminUser.List)
				r.Get("/admin/users/{userID}", h.AdminUser.Get)
				r.Patch("/admin/users/{userID}", h.AdminUser.PatchActive)
				r.Post("/admin/users/{userID}/revoke-sessions", h.AdminUser.RevokeSessions)
				r.Delete("/admin/users/{userID}", h.AdminUser.Delete)
				r.Get("/admin/settings/s3", h.Admin.S3Get)
				r.Patch("/admin/settings/s3", h.Admin.S3Patch)
				r.Post("/admin/settings/s3/test", h.Admin.S3Test)
				r.Get("/admin/settings/s3/cors-hints", h.Admin.S3CorsHints)
				r.Get("/admin/settings/smtp", h.Admin.SMTPGet)
				r.Patch("/admin/settings/smtp", h.Admin.SMTPPatch)
				r.Post("/admin/settings/smtp/test", h.Admin.SMTPTest)
				r.Get("/admin/settings/llm", h.Admin.LLMGet)
				r.Patch("/admin/settings/llm", h.Admin.LLMPatch)
				r.Post("/admin/settings/llm/test", h.Admin.LLMTest)
				r.Get("/admin/settings/llm/models", h.Admin.LLMModels)
				r.Get("/admin/products", h.Docs.ListProducts)
				r.Post("/admin/products", h.Docs.CreateProduct)
				r.Get("/admin/docs", h.Docs.ListSources)
				r.Post("/admin/docs/upload", h.Docs.Upload)
				r.Get("/admin/docs/{sourceID}", h.Docs.GetSource)
				r.Post("/admin/docs/{sourceID}/reprocess", h.Docs.Reprocess)
				r.Get("/admin/docs/{sourceID}/original-url", h.Docs.OriginalURL)
				r.Get("/admin/docs/{sourceID}/pages/{page}/url", h.Docs.PageURL)
				r.Post("/admin/rag/search", h.Docs.Search)
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
