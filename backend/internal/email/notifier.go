package email

import (
	"context"
	"strings"
)

type Notifier struct {
	loader *Loader
}

func NewNotifier(loader *Loader) *Notifier {
	return &Notifier{loader: loader}
}

func (n *Notifier) NotifyUserRegistered(ctx context.Context, data AdminRegistrationMail) error {
	settings, err := n.loader.Load(ctx)
	if err != nil {
		return err
	}
	if !settings.AdminNotifyEnabled {
		return nil
	}
	to := strings.TrimSpace(settings.AdminNotifyEmail)
	if to == "" {
		return nil
	}
	return Send(ctx, settings, Message{
		To:      to,
		Subject: SubjectAdminUserRegistered,
		Text:    AdminRegistrationText(data),
		HTML:    AdminRegistrationHTML(data),
	})
}

func (n *Notifier) NotifyOnboardingTicket(ctx context.Context, data OnboardingTicketMail) error {
	settings, err := n.loader.Load(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(data.UserEmail) != "" {
		_ = Send(ctx, settings, Message{
			To:      data.UserEmail,
			Subject: SubjectOnboardingTicket,
			Text:    OnboardingTicketText(data),
			HTML:    OnboardingTicketHTML(data),
		})
	}
	if settings.AdminNotifyEnabled {
		to := strings.TrimSpace(settings.AdminNotifyEmail)
		if to != "" {
			_ = Send(ctx, settings, Message{
				To:      to,
				Subject: SubjectOnboardingTicket,
				Text:    OnboardingTicketText(data),
				HTML:    OnboardingTicketAdminHTML(data),
			})
		}
	}
	return nil
}

func (n *Notifier) NotifyTicketActivity(ctx context.Context, data TicketActivityMail, clientEmail string) error {
	settings, err := n.loader.Load(ctx)
	if err != nil {
		return err
	}
	if data.IsAdminTarget {
		to := strings.TrimSpace(settings.AdminNotifyEmail)
		if !settings.AdminNotifyEnabled || to == "" {
			return nil
		}
		return Send(ctx, settings, Message{
			To:      to,
			Subject: SubjectTicketActivity,
			Text:    TicketActivityText(data),
			HTML:    TicketActivityHTML(data),
		})
	}
	to := strings.TrimSpace(clientEmail)
	if to == "" {
		return nil
	}
	data.IsAdminTarget = false
	return Send(ctx, settings, Message{
		To:      to,
		Subject: SubjectTicketActivity,
		Text:    TicketActivityText(data),
		HTML:    TicketActivityHTML(data),
	})
}

func (n *Notifier) NotifyOrgReviewResult(ctx context.Context, data OrgReviewResultMail, userEmail string) error {
	settings, err := n.loader.Load(ctx)
	if err != nil {
		return err
	}
	to := strings.TrimSpace(userEmail)
	if to == "" {
		return nil
	}
	subject := SubjectOrgReviewApproved
	if !data.Approved {
		subject = SubjectOrgReviewRejected
	}
	return Send(ctx, settings, Message{
		To:      to,
		Subject: subject,
		Text:    OrgReviewResultText(data),
		HTML:    OrgReviewResultHTML(data),
	})
}

func (n *Notifier) AdminPanelURL(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "/app/admin"
	}
	return base + "/admin"
}
