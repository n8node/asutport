package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
)

type BillingService struct {
	billing *repository.BillingRepo
	tickets *repository.TicketRepo
	orgs    *repository.OrgRepo
}

func NewBillingService(
	billing *repository.BillingRepo,
	tickets *repository.TicketRepo,
	orgs *repository.OrgRepo,
) *BillingService {
	return &BillingService{billing: billing, tickets: tickets, orgs: orgs}
}

type BillingOverview struct {
	MRRTotalRub            int `json:"mrr_total_rub"`
	MRRClientRub           int `json:"mrr_client_rub"`
	MRRManufacturerRub     int `json:"mrr_manufacturer_rub"`
	MRRPartnerRub          int `json:"mrr_partner_rub"`
	ActiveSubscriptions    int `json:"active_subscriptions"`
}

func (s *BillingService) AdminOverview(ctx context.Context) (*BillingOverview, error) {
	mrr, err := s.billing.SumActiveMRR(ctx)
	if err != nil {
		return nil, err
	}
	active, err := s.billing.CountActiveSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	return &BillingOverview{
		MRRTotalRub:        mrr.Total,
		MRRClientRub:       mrr.Client,
		MRRManufacturerRub: mrr.Manufacturer,
		MRRPartnerRub:      mrr.Partner,
		ActiveSubscriptions: active,
	}, nil
}

func (s *BillingService) EnsureDefaultSubscription(ctx context.Context, orgID uuid.UUID) error {
	if _, err := s.billing.GetSubscriptionByOrgID(ctx, orgID); err == nil {
		return nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return err
	}
	planType, defaultSlug := defaultPlanForOrgType(org.Type)
	if planType == "" || defaultSlug == "" {
		return nil
	}
	plan, err := s.billing.GetPlanBySlug(ctx, planType, defaultSlug)
	if err != nil {
		return fmt.Errorf("default plan %s/%s: %w", planType, defaultSlug, err)
	}
	start := time.Now().UTC()
	end := start.AddDate(0, 1, 0)
	_, err = s.billing.CreateSubscription(ctx, orgID, plan.ID, start, end)
	return err
}

func defaultPlanForOrgType(orgType string) (planOrgType, slug string) {
	switch orgType {
	case "client_org":
		return "client", "free"
	case "manufacturer":
		return "manufacturer", "basic"
	case "vendor":
		return "partner", "channel"
	default:
		return "", ""
	}
}

func (s *BillingService) OrgSummary(ctx context.Context, orgID uuid.UUID) (*models.BillingSummary, error) {
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	planType, _ := defaultPlanForOrgType(org.Type)
	publicPlans, err := s.billing.ListPlans(ctx, planType, false)
	if err != nil {
		return nil, err
	}
	sub, err := s.billing.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		if err := s.EnsureDefaultSubscription(ctx, orgID); err != nil {
			return nil, err
		}
		sub, err = s.billing.GetSubscriptionByOrgID(ctx, orgID)
		if err != nil {
			return nil, err
		}
	}
	plan, err := s.billing.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}
	used, err := s.tickets.CountSupportTicketsInPeriod(ctx, orgID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	if err != nil {
		return nil, err
	}
	payments, err := s.billing.ListPaymentsByOrg(ctx, orgID, 10)
	if err != nil {
		return nil, err
	}
	return &models.BillingSummary{
		Plan:            *plan,
		Subscription:    *sub,
		TicketsUsed:     used,
		TicketQuota:     plan.TicketQuota,
		OveragePriceRub: plan.OveragePriceRub,
		PeriodStart:     sub.CurrentPeriodStart,
		PeriodEnd:       sub.CurrentPeriodEnd,
		RecentPayments:  payments,
		PublicPlans:     publicPlans,
	}, nil
}

func (s *BillingService) CheckTicketQuota(ctx context.Context, orgID uuid.UUID, priority string) (*models.TicketQuotaCheck, error) {
	priority = normalizeTicketPriority(priority)
	summary, err := s.OrgSummary(ctx, orgID)
	if err != nil {
		return nil, err
	}
	check := &models.TicketQuotaCheck{
		Allowed:         true,
		TicketsUsed:     summary.TicketsUsed,
		TicketQuota:     summary.TicketQuota,
		OveragePriceRub: summary.OveragePriceRub,
		PlanName:        summary.Plan.Name,
		Priority:        priority,
	}
	if summary.TicketQuota == nil {
		return check, nil
	}
	quota := *summary.TicketQuota
	if summary.TicketsUsed >= quota {
		check.IsOverage = true
		if priority == "emergency" {
			check.Warning = fmt.Sprintf(
				"Квота тарифа «%s» (%d/%d) исчерпана. Аварийное обращение будет принято; доплата за сверхквотный тикет — %d ₽ постфактум.",
				summary.Plan.Name, summary.TicketsUsed, quota, summary.OveragePriceRub,
			)
		} else {
			check.Warning = fmt.Sprintf(
				"Квота тарифа «%s» (%d/%d) исчерпана. Обращение будет принято; доплата за сверхквотный тикет — %d ₽.",
				summary.Plan.Name, summary.TicketsUsed, quota, summary.OveragePriceRub,
			)
		}
	} else if summary.TicketsUsed == quota-1 {
		check.Warning = fmt.Sprintf(
			"После этого обращения исчерпается квота тарифа «%s» (%d/%d). Сверхквота — %d ₽ за тикет.",
			summary.Plan.Name, summary.TicketsUsed+1, quota, summary.OveragePriceRub,
		)
	}
	return check, nil
}

func (s *BillingService) RecordTicketOverage(ctx context.Context, orgID, ticketID uuid.UUID, priority string) error {
	if priority == "emergency" {
		return nil
	}
	summary, err := s.OrgSummary(ctx, orgID)
	if err != nil {
		return err
	}
	if summary.TicketQuota == nil || summary.OveragePriceRub <= 0 {
		return nil
	}
	if summary.TicketsUsed <= *summary.TicketQuota {
		return nil
	}
	subID := summary.Subscription.ID
	note := fmt.Sprintf("Сверхквотный тикет по тарифу «%s»", summary.Plan.Name)
	_, err = s.billing.CreatePayment(ctx, repository.PaymentCreateParams{
		OrgID:          orgID,
		SubscriptionID: &subID,
		TicketID:       &ticketID,
		Type:           "overage",
		AmountKopecks:  summary.OveragePriceRub * 100,
		Status:         "pending",
		Note:           note,
	})
	return err
}

func (s *BillingService) SLAReactionDeadline(ctx context.Context, orgID uuid.UUID, priority string) (time.Time, error) {
	priority = normalizeTicketPriority(priority)
	minutes := defaultSLAMinutes(priority)
	sub, err := s.billing.GetSubscriptionByOrgID(ctx, orgID)
	if err == nil {
		plan, planErr := s.billing.GetPlanByID(ctx, sub.PlanID)
		if planErr == nil && len(plan.SLAMatrix) > 0 {
			var matrix map[string]int
			if json.Unmarshal(plan.SLAMatrix, &matrix) == nil {
				if v, ok := matrix[priority]; ok && v > 0 {
					minutes = v
				}
			}
		}
	}
	return time.Now().UTC().Add(time.Duration(minutes) * time.Minute), nil
}

func defaultSLAMinutes(priority string) int {
	switch priority {
	case "emergency":
		return 30
	case "degraded":
		return 120
	default:
		return 480
	}
}

func (s *BillingService) ListPlans(ctx context.Context, orgType string, includeArchived bool) ([]models.Plan, error) {
	return s.billing.ListPlans(ctx, orgType, includeArchived)
}

func (s *BillingService) CreatePlan(ctx context.Context, p repository.PlanUpsertParams) (*models.Plan, error) {
	p.OrgType = strings.TrimSpace(p.OrgType)
	p.Name = strings.TrimSpace(p.Name)
	p.Slug = strings.TrimSpace(p.Slug)
	if p.OrgType == "" || p.Name == "" || p.Slug == "" {
		return nil, fmt.Errorf("org_type, name and slug are required")
	}
	return s.billing.CreatePlan(ctx, p)
}

func (s *BillingService) UpdatePlan(ctx context.Context, planID uuid.UUID, p repository.PlanUpsertParams) (*models.Plan, error) {
	p.OrgType = strings.TrimSpace(p.OrgType)
	p.Name = strings.TrimSpace(p.Name)
	p.Slug = strings.TrimSpace(p.Slug)
	if p.OrgType == "" || p.Name == "" || p.Slug == "" {
		return nil, fmt.Errorf("org_type, name and slug are required")
	}
	return s.billing.UpdatePlan(ctx, planID, p)
}

func (s *BillingService) AssignSubscription(ctx context.Context, orgID, planID uuid.UUID) (*models.Subscription, error) {
	if _, err := s.billing.GetPlanByID(ctx, planID); err != nil {
		return nil, err
	}
	_, err := s.billing.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
		start := time.Now().UTC()
		return s.billing.CreateSubscription(ctx, orgID, planID, start, start.AddDate(0, 1, 0))
	}
	return s.billing.UpdateSubscriptionPlan(ctx, orgID, planID)
}

type RecordPaymentInput struct {
	OrgID          uuid.UUID
	SubscriptionID *uuid.UUID
	TicketID       *uuid.UUID
	Type           string
	AmountKopecks  int
	Status         string
	InvoiceS3Key   string
	Note           string
	RecordedBy     uuid.UUID
}

func (s *BillingService) RecordPayment(ctx context.Context, in RecordPaymentInput) (*models.Payment, error) {
	in.Type = strings.TrimSpace(in.Type)
	if in.Type == "" {
		in.Type = "subscription"
	}
	if in.AmountKopecks <= 0 {
		return nil, fmt.Errorf("amount is required")
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "paid"
	}
	rec := &in.RecordedBy
	return s.billing.CreatePayment(ctx, repository.PaymentCreateParams{
		OrgID:          in.OrgID,
		SubscriptionID: in.SubscriptionID,
		TicketID:       in.TicketID,
		Type:           in.Type,
		AmountKopecks:  in.AmountKopecks,
		Status:         status,
		InvoiceS3Key:   strings.TrimSpace(in.InvoiceS3Key),
		Note:           strings.TrimSpace(in.Note),
		RecordedBy:     rec,
	})
}

func (s *BillingService) ListPayments(ctx context.Context, orgID uuid.UUID) ([]models.Payment, error) {
	return s.billing.ListPaymentsByOrg(ctx, orgID, 50)
}
