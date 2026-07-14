package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID
	Email           string
	PasswordHash    string
	FullName        string
	IsActive        bool
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Organization struct {
	ID            uuid.UUID
	Name          string
	Type          string
	Slug          string
	IsActive      bool
	LegalName     string
	INN           string
	Website       string
	ContactPhone  string
	ReviewComment string
	IsPersonal    bool
	ReviewStatus  string
	ReviewedAt          *time.Time
	ReviewedBy          *uuid.UUID
	OnboardingTicketID  *uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type OrgMember struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	UserID    uuid.UUID
	Role      string
	CreatedAt time.Time
}

type OrgMembership struct {
	OrgMember
	OrgName         string
	OrgType         string
	OrgSlug         string
	OrgReviewStatus string
	OrgIsPersonal   bool
}

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	OrgID            uuid.UUID
	RefreshTokenHash string
	UserAgent        string
	IPAddress        string
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
}

type APIKey struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	Name       string
	KeyPrefix  string
	KeyHash    string
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

type UserMessengerLink struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	Provider             string
	ExternalUserID       string
	Username             string
	DisplayName          string
	IsVerified           bool
	NotificationsEnabled bool
	LinkedAt             *time.Time
	RevokedAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type AdminUserMembership struct {
	OrgID         uuid.UUID
	OrgName       string
	OrgType       string
	OrgSlug       string
	Role          string
	ReviewStatus  string
	IsPersonal    bool
	OrgIsActive   bool
	INN           string
	Website       string
	ContactPhone  string
	MemberSince   time.Time
}

type AdminUserListRow struct {
	User
	LastLoginAt    *time.Time
	ActiveSessions int
	LastIP         string
	LastUserAgent  string
	Memberships    []AdminUserMembership
	Messengers     []UserMessengerLink
	AccessLevel    string
}

type AdminUserSession struct {
	Session
	OrgName string
}

type AdminOrgOwner struct {
	UserID   uuid.UUID
	Email    string
	FullName string
	Role     string
}

type AdminOrgMember struct {
	UserID    uuid.UUID
	Email     string
	FullName  string
	Role      string
	IsActive  bool
	CreatedAt time.Time
}

type AdminOrgMetrics struct {
	Installations      *int `json:"installations"`
	TicketQuotaUsed    *int `json:"ticket_quota_used"`
	TicketQuotaLimit   *int `json:"ticket_quota_limit"`
	OpenTickets        *int `json:"open_tickets"`
	DocSources         *int `json:"doc_sources"`
	Products           *int `json:"products"`
	SupportZoneLoaded  bool `json:"support_zone_loaded"`
	GoldenSetReady     bool `json:"golden_set_ready"`
	EntitlementLinks   *int `json:"entitlement_links"`
	FallbackEvents30d  *int `json:"fallback_events_30d"`
	ApplicationTickets *int `json:"application_tickets"`
	PlanName           string `json:"plan_name"`
	MRRRub             *int `json:"mrr_rub"`
}

type AdminOrgListRow struct {
	Organization
	MemberCount int
	Owner       *AdminOrgOwner
	Metrics     AdminOrgMetrics
}

type RegistrationVerification struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	OrgID       uuid.UUID
	RegID       string
	AccountType string
	ExpiresAt   time.Time
	UsedAt      *time.Time
	CreatedAt   time.Time
}

type Ticket struct {
	ID                   uuid.UUID
	ClientOrgID          uuid.UUID
	InstallationID       *uuid.UUID
	Type                 string
	Priority             string
	Status               string
	BallOwnerOrgID       *uuid.UUID
	AssignedTargetOrgID  *uuid.UUID
	Subject              string
	SLAReactionDeadline  *time.Time
	CreatedByUserID      *uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
	ClientOrgName        string
	ClientOrgType        string
	ClientOrgINN         string
	ClientReviewStatus   string
	BallOwnerOrgName     string
	AssignedTargetName   string
}

type FallbackLogEntry struct {
	ID             uuid.UUID
	TicketID       uuid.UUID
	NeededRole     string
	MissingOrgName string
	CreatedAt      time.Time
}

type Installation struct {
	ID                    uuid.UUID
	ClientOrgID           uuid.UUID
	Name                  string
	SiteAddress           string
	Criticality           string
	SnapshotAllowed       bool
	EmergencyContactName  string
	EmergencyContactPhone string
	Environment           []byte
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type InstallationProduct struct {
	ID               uuid.UUID
	InstallationID   uuid.UUID
	ManufacturerName string
	ProductName      string
	Kind             string
	Version          string
	Notes            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type SupplyRecord struct {
	ID                    uuid.UUID
	InstallationProductID uuid.UUID
	SerialOrLicense       string
	SupplierName          string
	IntegratorName        string
	PurchaseDate          *time.Time
	WarrantyUntil         *time.Time
	ContractRef           string
	VerifyStatus          string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ClientDashboardSummary struct {
	InstallationsCount int
	OpenTicketsCount   int
	SLAActiveCount     int
	CoveragePercent    int
	ProfileComplete    bool
	ProductsCount      int
	SupplyRecordsCount int
}

type TicketEvent struct {
	ID          uuid.UUID
	TicketID    uuid.UUID
	Kind        string
	ActorUserID *uuid.UUID
	ActorOrgID  *uuid.UUID
	Payload     []byte
	CreatedAt   time.Time
	ActorName   string
	ActorEmail  string
	IsPlatform  bool
}

type TicketAttachment struct {
	ID                uuid.UUID
	TicketID          uuid.UUID
	EventID           *uuid.UUID
	S3Key             string
	Filename          string
	ContentType       string
	SizeBytes         int64
	UploadedByUserID  uuid.UUID
	UploadedByOrgID   uuid.UUID
	Status            string
	CreatedAt         time.Time
}
