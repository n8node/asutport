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
	ID        uuid.UUID
	Name      string
	Type      string
	Slug      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
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
	OrgName string
	OrgType string
	OrgSlug string
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
