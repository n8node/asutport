package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/config"
	"github.com/n8node/asutport/internal/email"
	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
	s3store "github.com/n8node/asutport/internal/s3"
)

const maxAttachmentBytes = 20 << 20 // 20 MiB

func MaxAttachmentBytes() int64 {
	return maxAttachmentBytes
}

func MaxUploadBodyBytes() int64 {
	// Base64 JSON payloads are ~4/3 of raw file size.
	return maxAttachmentBytes*4/3 + (2 << 20)
}

var allowedAttachmentTypes = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
	"image/jpeg":      true,
	"image/jpg":       true,
}

type TicketService struct {
	cfg           *config.Config
	tickets       *repository.TicketRepo
	orgs          *repository.OrgRepo
	members       *repository.OrgMemberRepo
	users         *repository.UserRepo
	installations *repository.InstallationRepo
	fallbacks     *repository.FallbackRepo
	s3Loader      *s3store.Loader
	notify        *email.Notifier
}

func NewTicketService(
	cfg *config.Config,
	tickets *repository.TicketRepo,
	orgs *repository.OrgRepo,
	members *repository.OrgMemberRepo,
	users *repository.UserRepo,
	installations *repository.InstallationRepo,
	fallbacks *repository.FallbackRepo,
	s3Loader *s3store.Loader,
	notify *email.Notifier,
) *TicketService {
	return &TicketService{
		cfg:           cfg,
		tickets:       tickets,
		orgs:          orgs,
		members:       members,
		users:         users,
		installations: installations,
		fallbacks:     fallbacks,
		s3Loader:      s3Loader,
		notify:        notify,
	}
}

func (s *TicketService) storageClient(ctx context.Context) (*s3store.Client, error) {
	if s.s3Loader == nil {
		return nil, fmt.Errorf("object storage is not configured")
	}
	return s.s3Loader.Client(ctx)
}

func (s *TicketService) CreateOnboardingIfNeeded(ctx context.Context, orgID, userID uuid.UUID) (*models.Ticket, error) {
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if org.OnboardingTicketID != nil {
		return s.tickets.GetByID(ctx, *org.OnboardingTicketID)
	}
	existing, err := s.tickets.GetOnboardingByClientOrg(ctx, orgID)
	if err == nil {
		_ = s.orgs.LinkOnboardingTicket(ctx, orgID, existing.ID)
		return existing, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	subject := fmt.Sprintf("Проверка организации: %s", org.Name)
	ticket, err := s.tickets.Create(ctx, repository.TicketCreateParams{
		ClientOrgID:     orgID,
		Type:            "onboarding",
		Priority:        "question",
		Status:          "waiting_client",
		BallOwnerOrgID:  &orgID,
		Subject:         subject,
		CreatedByUserID: userID,
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.tickets.AddEvent(ctx, ticket.ID, "message", nil, nil, repository.EventPayloadText(repository.OnboardingWelcomeMessage)); err != nil {
		return nil, err
	}
	if err := s.orgs.LinkOnboardingTicket(ctx, orgID, ticket.ID); err != nil {
		return nil, err
	}
	detail, err := s.tickets.GetByID(ctx, ticket.ID)
	if err != nil {
		return nil, err
	}
	_ = s.notifyOnboardingCreated(ctx, org, detail, userID)
	return detail, nil
}

func (s *TicketService) CanAccess(ctx context.Context, ticket *models.Ticket, userID, orgID uuid.UUID, isSuperAdmin bool) bool {
	if isSuperAdmin {
		return true
	}
	if ticket.ClientOrgID == orgID {
		return true
	}
	if ticket.AssignedTargetOrgID != nil && *ticket.AssignedTargetOrgID == orgID {
		return true
	}
	return false
}

func (s *TicketService) ListByAssignedTarget(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.Ticket, int, error) {
	return s.tickets.ListByAssignedTarget(ctx, orgID, limit, offset)
}

func (s *TicketService) CountOpenByAssignedTarget(ctx context.Context, orgID uuid.UUID) (int, error) {
	return s.tickets.CountOpenByAssignedTarget(ctx, orgID)
}

func (s *TicketService) GetOnboardingForOrg(ctx context.Context, orgID uuid.UUID) (*models.Ticket, error) {
	return s.tickets.GetOnboardingByClientOrg(ctx, orgID)
}

func (s *TicketService) GetByID(ctx context.Context, ticketID uuid.UUID) (*models.Ticket, error) {
	return s.tickets.GetByID(ctx, ticketID)
}

func (s *TicketService) ListOnboarding(ctx context.Context, reviewStatus string, limit, offset int) ([]models.Ticket, int, error) {
	return s.tickets.ListOnboarding(ctx, reviewStatus, limit, offset)
}

func (s *TicketService) ListByClientOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.Ticket, int, error) {
	return s.tickets.ListByClientOrg(ctx, orgID, limit, offset)
}

func (s *TicketService) CountOpenByClientOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	return s.tickets.CountOpenByClientOrg(ctx, orgID)
}

func (s *TicketService) CountSLAActiveByClientOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	return s.tickets.CountSLAActiveByClientOrg(ctx, orgID)
}

type CreateSupportTicketInput struct {
	ClientOrgID     uuid.UUID
	InstallationID  *uuid.UUID
	Subject         string
	Type            string
	Priority        string
	CreatedByUserID uuid.UUID
	InitialText     string
}

func (s *TicketService) CreateSupportTicket(ctx context.Context, in CreateSupportTicketInput) (*models.Ticket, error) {
	in.Subject = strings.TrimSpace(in.Subject)
	if in.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	ticketType := normalizeSupportTicketType(in.Type)
	priority := normalizeTicketPriority(in.Priority)
	deadline := slaReactionDeadline(priority)
	ticket, err := s.tickets.Create(ctx, repository.TicketCreateParams{
		ClientOrgID:         in.ClientOrgID,
		InstallationID:      in.InstallationID,
		Type:                ticketType,
		Priority:            priority,
		Status:              "open",
		BallOwnerOrgID:      nil,
		Subject:             in.Subject,
		CreatedByUserID:     in.CreatedByUserID,
		SLAReactionDeadline: &deadline,
	})
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(in.InitialText)
	if text == "" {
		text = in.Subject
	}
	if _, err := s.tickets.AddEvent(ctx, ticket.ID, "message", &in.CreatedByUserID, &in.ClientOrgID, repository.EventPayloadText(text)); err != nil {
		return nil, err
	}
	if err := s.routeSupportTicket(ctx, ticket.ID, ticketType, in.InstallationID); err != nil {
		return nil, err
	}
	return s.tickets.GetByID(ctx, ticket.ID)
}

func normalizeSupportTicketType(raw string) string {
	switch strings.TrimSpace(raw) {
	case "typical", "defect", "warranty", "application", "cross_vendor":
		return strings.TrimSpace(raw)
	default:
		return "typical"
	}
}

func normalizeTicketPriority(raw string) string {
	switch strings.TrimSpace(raw) {
	case "emergency", "degraded", "question":
		return strings.TrimSpace(raw)
	default:
		return "question"
	}
}

func slaReactionDeadline(priority string) time.Time {
	now := time.Now().UTC()
	switch priority {
	case "emergency":
		return now.Add(30 * time.Minute)
	case "degraded":
		return now.Add(2 * time.Hour)
	default:
		return now.Add(8 * time.Hour)
	}
}

func (s *TicketService) ListEvents(ctx context.Context, ticketID uuid.UUID) ([]models.TicketEvent, error) {
	return s.tickets.ListEvents(ctx, ticketID, 200)
}

func (s *TicketService) ListAttachments(ctx context.Context, ticketID uuid.UUID) ([]models.TicketAttachment, error) {
	return s.tickets.ListAttachments(ctx, ticketID)
}

func (s *TicketService) PostMessage(
	ctx context.Context,
	ticket *models.Ticket,
	actorUserID, actorOrgID uuid.UUID,
	isSuperAdmin bool,
	text string,
) (*models.TicketEvent, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("message text is required")
	}
	if ticket.Status == "closed" {
		return nil, fmt.Errorf("ticket is closed")
	}
	var actorOrgPtr *uuid.UUID
	if !isSuperAdmin {
		actorOrgPtr = &actorOrgID
	}
	event, err := s.tickets.AddEvent(ctx, ticket.ID, "message", &actorUserID, actorOrgPtr, repository.EventPayloadText(text))
	if err != nil {
		return nil, err
	}
	var ballOwner *uuid.UUID
	var status string
	switch ticketActorRole(ticket, actorOrgID, isSuperAdmin) {
	case "platform":
		status = "waiting_client"
		ballOwner = &ticket.ClientOrgID
	case "vendor":
		status = "waiting_client"
		ballOwner = &ticket.ClientOrgID
	case "client":
		if ticket.AssignedTargetOrgID != nil {
			status = "waiting_vendor"
			ballOwner = ticket.AssignedTargetOrgID
		} else {
			status = "waiting_platform"
			ballOwner = nil
		}
	default:
		status = "waiting_platform"
		ballOwner = nil
	}
	if err := s.tickets.UpdateStatus(ctx, ticket.ID, status, ballOwner); err != nil {
		return nil, err
	}
	_ = s.notifyTicketMessage(ctx, ticket, actorUserID, isSuperAdmin, text)
	return event, nil
}

func (s *TicketService) ResolveTicket(
	ctx context.Context,
	ticket *models.Ticket,
	actorUserID, actorOrgID uuid.UUID,
	isSuperAdmin bool,
	note string,
) error {
	if ticket.Status == "closed" || ticket.Status == "resolved" {
		return fmt.Errorf("ticket is already closed")
	}
	role := ticketActorRole(ticket, actorOrgID, isSuperAdmin)
	if role == "" {
		return fmt.Errorf("access denied")
	}
	note = strings.TrimSpace(note)
	if note == "" {
		note = "Обращение решено."
	}
	var actorOrgPtr *uuid.UUID
	if !isSuperAdmin {
		actorOrgPtr = &actorOrgID
	}
	if _, err := s.tickets.AddEvent(ctx, ticket.ID, "resolved", &actorUserID, actorOrgPtr, repository.EventPayloadResolved(note)); err != nil {
		return err
	}
	return s.tickets.UpdateStatus(ctx, ticket.ID, "resolved", nil)
}

type PresignAttachmentInput struct {
	Filename    string
	ContentType string
	SizeBytes   int64
}

type PresignAttachmentResult struct {
	AttachmentID uuid.UUID
	UploadURL    string
	S3Key        string
}

func (s *TicketService) PresignAttachment(
	ctx context.Context,
	ticket *models.Ticket,
	userID, orgID uuid.UUID,
	in PresignAttachmentInput,
) (*PresignAttachmentResult, error) {
	if s.s3Loader == nil {
		return nil, fmt.Errorf("object storage is not configured")
	}
	if ticket.Status == "closed" {
		return nil, fmt.Errorf("ticket is closed")
	}
	filename := sanitizeFilename(in.Filename)
	if filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	ct, err := normalizeAttachmentContentType(filename, in.ContentType)
	if err != nil {
		return nil, err
	}
	if in.SizeBytes <= 0 || in.SizeBytes > maxAttachmentBytes {
		return nil, fmt.Errorf("invalid file size")
	}
	s3, err := s.storageClient(ctx)
	if err != nil {
		return nil, err
	}
	attID := uuid.New()
	storageFilename := sanitizeS3StorageName(filename)
	key := s3store.TicketAttachmentKey(ticket.ID.String(), attID.String(), storageFilename)
	att, err := s.tickets.CreateAttachment(ctx, models.TicketAttachment{
		TicketID:         ticket.ID,
		S3Key:            key,
		Filename:         filename,
		ContentType:      ct,
		SizeBytes:        in.SizeBytes,
		UploadedByUserID: userID,
		UploadedByOrgID:  orgID,
		Status:           "pending",
	})
	if err != nil {
		return nil, err
	}
	url, err := s3.PresignPut(ctx, key, ct, 0)
	if err != nil {
		return nil, err
	}
	return &PresignAttachmentResult{
		AttachmentID: att.ID,
		UploadURL:    url,
		S3Key:        key,
	}, nil
}

func (s *TicketService) UploadAttachment(
	ctx context.Context,
	ticket *models.Ticket,
	userID, orgID uuid.UUID,
	isSuperAdmin bool,
	filename, contentType string,
	body io.Reader,
	sizeBytes int64,
) (*models.TicketEvent, error) {
	if s.s3Loader == nil {
		return nil, fmt.Errorf("object storage is not configured")
	}
	if ticket.Status == "closed" {
		return nil, fmt.Errorf("ticket is closed")
	}
	filename = sanitizeFilename(filename)
	if filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	ct, err := normalizeAttachmentContentType(filename, contentType)
	if err != nil {
		return nil, err
	}
	if sizeBytes <= 0 || sizeBytes > maxAttachmentBytes {
		return nil, fmt.Errorf("invalid file size")
	}
	s3, err := s.storageClient(ctx)
	if err != nil {
		return nil, err
	}
	attID := uuid.New()
	storageFilename := sanitizeS3StorageName(filename)
	key := s3store.TicketAttachmentKey(ticket.ID.String(), attID.String(), storageFilename)
	att, err := s.tickets.CreateAttachment(ctx, models.TicketAttachment{
		ID:               attID,
		TicketID:         ticket.ID,
		S3Key:            key,
		Filename:         filename,
		ContentType:      ct,
		SizeBytes:        sizeBytes,
		UploadedByUserID: userID,
		UploadedByOrgID:  orgID,
		Status:           "pending",
	})
	if err != nil {
		return nil, err
	}
	if err := s3.PutObject(ctx, key, ct, body, sizeBytes); err != nil {
		return nil, fmt.Errorf("storage upload failed")
	}
	return s.completeAttachmentRecord(ctx, ticket, att, userID, orgID, isSuperAdmin)
}

func (s *TicketService) CompleteAttachment(
	ctx context.Context,
	ticket *models.Ticket,
	attachmentID, userID, orgID uuid.UUID,
	isSuperAdmin bool,
) (*models.TicketEvent, error) {
	att, err := s.tickets.GetAttachment(ctx, attachmentID)
	if err != nil {
		return nil, err
	}
	if att.TicketID != ticket.ID {
		return nil, repository.ErrNotFound
	}
	if att.Status != "pending" {
		return nil, fmt.Errorf("attachment already completed")
	}
	if !isSuperAdmin && att.UploadedByOrgID != orgID {
		return nil, repository.ErrNotFound
	}
	return s.completeAttachmentRecord(ctx, ticket, att, userID, orgID, isSuperAdmin)
}

func (s *TicketService) completeAttachmentRecord(
	ctx context.Context,
	ticket *models.Ticket,
	att *models.TicketAttachment,
	userID, orgID uuid.UUID,
	isSuperAdmin bool,
) (*models.TicketEvent, error) {
	var actorOrgPtr *uuid.UUID
	if !isSuperAdmin {
		actorOrgPtr = &orgID
	}
	event, err := s.tickets.AddEvent(ctx, ticket.ID, "attachment_added", &userID, actorOrgPtr, repository.EventPayloadAttachment(att.ID, att.Filename))
	if err != nil {
		return nil, err
	}
	if err := s.tickets.CompleteAttachment(ctx, att.ID, event.ID); err != nil {
		return nil, err
	}
	var ballOwner *uuid.UUID
	status := "waiting_platform"
	switch ticketActorRole(ticket, orgID, isSuperAdmin) {
	case "platform":
		status = "waiting_client"
		ballOwner = &ticket.ClientOrgID
	case "vendor":
		status = "waiting_client"
		ballOwner = &ticket.ClientOrgID
	case "client":
		if ticket.AssignedTargetOrgID != nil {
			status = "waiting_vendor"
			ballOwner = ticket.AssignedTargetOrgID
		}
	}
	if err := s.tickets.UpdateStatus(ctx, ticket.ID, status, ballOwner); err != nil {
		return nil, err
	}
	_ = s.notifyTicketAttachment(ctx, ticket, att)
	return event, nil
}

func (s *TicketService) AttachmentDownloadURL(ctx context.Context, ticket *models.Ticket, attachmentID uuid.UUID) (string, error) {
	s3, err := s.storageClient(ctx)
	if err != nil {
		return "", err
	}
	att, err := s.tickets.GetAttachment(ctx, attachmentID)
	if err != nil {
		return "", err
	}
	if att.TicketID != ticket.ID || att.Status != "completed" {
		return "", repository.ErrNotFound
	}
	return s3.PresignGet(ctx, att.S3Key, 0)
}

func (s *TicketService) ApproveOrg(
	ctx context.Context,
	ticket *models.Ticket,
	reviewerID uuid.UUID,
	rationale string,
) error {
	if ticket.Type != "onboarding" {
		return fmt.Errorf("invalid ticket type")
	}
	if ticket.ClientReviewStatus != "pending_review" {
		return fmt.Errorf("organization is not pending review")
	}
	rationale = strings.TrimSpace(rationale)
	if rationale == "" {
		rationale = "Организация активирована после проверки документов."
	}
	if _, err := s.tickets.AddEvent(ctx, ticket.ID, "org_approved", &reviewerID, nil, repository.EventPayloadReview(rationale)); err != nil {
		return err
	}
	if err := s.orgs.UpdateReviewStatus(ctx, ticket.ClientOrgID, reviewerID, "active"); err != nil {
		return err
	}
	if err := s.tickets.UpdateStatus(ctx, ticket.ID, "closed", nil); err != nil {
		return err
	}
	_ = s.notifyOrgReviewResult(ctx, ticket, true, rationale)
	return nil
}

func (s *TicketService) RejectOrg(
	ctx context.Context,
	ticket *models.Ticket,
	reviewerID uuid.UUID,
	rationale string,
) error {
	if ticket.Type != "onboarding" {
		return fmt.Errorf("invalid ticket type")
	}
	rationale = strings.TrimSpace(rationale)
	if rationale == "" {
		return fmt.Errorf("rejection rationale is required")
	}
	if _, err := s.tickets.AddEvent(ctx, ticket.ID, "org_rejected", &reviewerID, nil, repository.EventPayloadReview(rationale)); err != nil {
		return err
	}
	if err := s.orgs.UpdateReviewStatus(ctx, ticket.ClientOrgID, reviewerID, "rejected"); err != nil {
		return err
	}
	if err := s.tickets.UpdateStatus(ctx, ticket.ID, "closed", nil); err != nil {
		return err
	}
	_ = s.notifyOrgReviewResult(ctx, ticket, false, rationale)
	return nil
}

func sanitizeFilename(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	base = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return -1
		}
		return r
	}, base)
	if base == "." || base == ".." {
		return ""
	}
	return base
}

func normalizeAttachmentContentType(filename, contentType string) (string, error) {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch ct {
	case "image/pjpeg":
		ct = "image/jpeg"
	case "application/x-pdf":
		ct = "application/pdf"
	}
	if ct == "" || ct == "application/octet-stream" {
		switch strings.ToLower(filepath.Ext(filename)) {
		case ".pdf":
			ct = "application/pdf"
		case ".png":
			ct = "image/png"
		case ".jpg", ".jpeg":
			ct = "image/jpeg"
		}
	}
	if !allowedAttachmentTypes[ct] {
		return "", fmt.Errorf("unsupported file type")
	}
	return ct, nil
}

func sanitizeS3StorageName(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	var b strings.Builder
	for _, r := range stem {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		out = "attachment"
	}
	if ext == "" {
		return out
	}
	return out + ext
}

func (s *TicketService) notifyOnboardingCreated(ctx context.Context, org *models.Organization, ticket *models.Ticket, userID uuid.UUID) error {
	if s.notify == nil {
		return nil
	}
	u, _ := s.users.GetByID(ctx, userID)
	userEmail := ""
	name := ""
	if u != nil {
		userEmail = u.Email
		name = u.FullName
	}
	return s.notify.NotifyOnboardingTicket(ctx, email.OnboardingTicketMail{
		UserEmail:      userEmail,
		FullName:       name,
		OrgName:        org.Name,
		TicketID:       ticket.ID.String(),
		TicketURL:      s.cfg.PublicAppBaseURL() + "/dashboard/onboarding",
		AdminTicketURL: s.cfg.PublicAppBaseURL() + "/admin/tickets/" + ticket.ID.String(),
	})
}

func (s *TicketService) notifyTicketMessage(ctx context.Context, ticket *models.Ticket, actorUserID uuid.UUID, isSuperAdmin bool, text string) error {
	if s.notify == nil {
		return nil
	}
	clientEmail := s.ownerEmail(ctx, ticket.ClientOrgID)
	return s.notify.NotifyTicketActivity(ctx, email.TicketActivityMail{
		TicketID:      ticket.ID.String(),
		OrgName:       ticket.ClientOrgName,
		Subject:       ticket.Subject,
		Preview:       truncate(text, 200),
		IsAdminTarget: !isSuperAdmin,
		ClientURL:     s.cfg.PublicAppBaseURL() + "/dashboard/onboarding",
		AdminURL:      s.cfg.PublicAppBaseURL() + "/admin/tickets/" + ticket.ID.String(),
	}, clientEmail)
}

func (s *TicketService) notifyTicketAttachment(ctx context.Context, ticket *models.Ticket, att *models.TicketAttachment) error {
	if s.notify == nil || att == nil {
		return nil
	}
	mail := email.TicketActivityMail{
		TicketID:           ticket.ID.String(),
		OrgName:            ticket.ClientOrgName,
		Subject:            ticket.Subject,
		Preview:            "Загружен файл: " + path.Base(att.Filename),
		IsAdminTarget:      true,
		ClientURL:          s.cfg.PublicAppBaseURL() + "/dashboard/onboarding",
		AdminURL:           s.cfg.PublicAppBaseURL() + "/admin/tickets/" + ticket.ID.String(),
		AttachmentFilename: att.Filename,
	}
	if s.s3Loader != nil {
		if s3, err := s.storageClient(ctx); err == nil {
			if url, err := s3.PresignGet(ctx, att.S3Key, 0); err == nil {
				mail.AttachmentURL = url
			}
		}
	}
	return s.notify.NotifyTicketActivity(ctx, mail, "")
}

func (s *TicketService) notifyOrgReviewResult(ctx context.Context, ticket *models.Ticket, approved bool, rationale string) error {
	if s.notify == nil {
		return nil
	}
	return s.notify.NotifyOrgReviewResult(ctx, email.OrgReviewResultMail{
		OrgName:   ticket.ClientOrgName,
		Approved:  approved,
		Rationale: rationale,
		LoginURL:  s.cfg.PublicAppBaseURL() + "/login",
	}, s.ownerEmail(ctx, ticket.ClientOrgID))
}

func (s *TicketService) ownerEmail(ctx context.Context, orgID uuid.UUID) string {
	member, err := s.members.PrimaryAdminForOrg(ctx, orgID)
	if err != nil {
		return ""
	}
	u, err := s.users.GetByID(ctx, member.UserID)
	if err != nil {
		return ""
	}
	return u.Email
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
