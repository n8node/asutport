package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FullName     string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
	ReviewedAt    *time.Time
	ReviewedBy    *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
