package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
)

type routeTarget struct {
	lookupName string
	neededRole string
	orgTypes   []string
}

func (s *TicketService) routeSupportTicket(
	ctx context.Context,
	ticketID uuid.UUID,
	ticketType string,
	installationID *uuid.UUID,
) error {
	target := s.routingTarget(ctx, ticketType, installationID)
	if target.lookupName == "" {
		return s.tickets.UpdateRouting(ctx, ticketID, nil, "waiting_platform", nil)
	}

	org, err := s.orgs.FindActiveByName(ctx, target.lookupName, target.orgTypes...)
	if err == nil && org != nil {
		if _, err := s.tickets.AddEvent(ctx, ticketID, "escalated", nil, nil, repository.EventPayloadEscalation(org.Name)); err != nil {
			return err
		}
		return s.tickets.UpdateRouting(ctx, ticketID, &org.ID, "waiting_vendor", &org.ID)
	}

	msg := fmt.Sprintf(
		"Производитель или поставщик «%s» пока не подключён к ASUTPORT. Обращение принято платформой; мы поможем связаться напрямую или подключить сторону.",
		target.lookupName,
	)
	if _, err := s.tickets.AddEvent(ctx, ticketID, "fallback", nil, nil, repository.EventPayloadFallback(target.neededRole, target.lookupName, msg)); err != nil {
		return err
	}
	if s.fallbacks != nil {
		_ = s.fallbacks.Create(ctx, ticketID, target.neededRole, target.lookupName)
	}
	return s.tickets.UpdateRouting(ctx, ticketID, nil, "waiting_platform", nil)
}

func (s *TicketService) routingTarget(ctx context.Context, ticketType string, installationID *uuid.UUID) routeTarget {
	var manufacturer, supplier, integrator string
	if installationID != nil && s.installations != nil {
		manufacturer, supplier, integrator, _ = s.installations.RoutingHintNames(ctx, *installationID)
	}
	switch ticketType {
	case "defect", "cross_vendor":
		return routeTarget{lookupName: manufacturer, neededRole: "manufacturer", orgTypes: []string{"manufacturer"}}
	case "warranty":
		name := supplier
		if name == "" {
			name = manufacturer
		}
		return routeTarget{lookupName: name, neededRole: "partner", orgTypes: []string{"vendor", "partner"}}
	case "application":
		return routeTarget{lookupName: integrator, neededRole: "integrator", orgTypes: []string{"integrator"}}
	default:
		return routeTarget{}
	}
}

func ticketActorRole(ticket *models.Ticket, orgID uuid.UUID, isSuperAdmin bool) string {
	if isSuperAdmin {
		return "platform"
	}
	if ticket.ClientOrgID == orgID {
		return "client"
	}
	if ticket.AssignedTargetOrgID != nil && *ticket.AssignedTargetOrgID == orgID {
		return "vendor"
	}
	return ""
}
