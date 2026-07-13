package email

import (
	"fmt"
	"html"
	"strings"
)

const (
	SubjectRegistrationConfirm   = "Подтверждение регистрации на платформе ASUTPORT"
	SubjectAdminUserRegistered   = "ASUTPORT — новая регистрация пользователя"
	SubjectOnboardingTicket        = "ASUTPORT — тикет проверки организации"
	SubjectTicketActivity          = "ASUTPORT — обновление в тикете"
	SubjectOrgReviewApproved       = "ASUTPORT — организация активирована"
	SubjectOrgReviewRejected       = "ASUTPORT — заявка организации отклонена"
	SubjectSMTPTest                = "ASUTPORT — тест SMTP"
)

type RegistrationMail struct {
	FullName   string
	ConfirmURL string
}

type AdminRegistrationMail struct {
	UserEmail     string
	FullName      string
	AccountType   string
	OrgName       string
	OrgType       string
	RegID         string
	RegisteredAt  string
	AdminPanelURL string
}

func RegistrationHTML(data RegistrationMail) string {
	name := strings.TrimSpace(data.FullName)
	if name == "" {
		name = "коллега"
	}
	body := fmt.Sprintf(`
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#1f2933;">Здравствуйте, <strong>%s</strong>!</p>
<p style="margin:0 0 20px;font-size:15px;line-height:1.6;color:#4b5563;">Вы начали регистрацию на платформе технической поддержки АСУ ТП. Подтвердите адрес email, чтобы завершить создание аккаунта.</p>
<table role="presentation" cellspacing="0" cellpadding="0" style="margin:0 0 24px;">
  <tr>
    <td style="border-radius:8px;background:#0d9488;">
      <a href="%s" style="display:inline-block;padding:14px 28px;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;">Подтвердить регистрацию</a>
    </td>
  </tr>
</table>
<p style="margin:0 0 8px;font-size:13px;line-height:1.5;color:#6b7280;">Если кнопка не открывается, скопируйте ссылку в браузер:</p>
<p style="margin:0 0 20px;font-size:13px;line-height:1.5;word-break:break-all;"><a href="%s" style="color:#0d9488;">%s</a></p>
<p style="margin:0;font-size:13px;line-height:1.5;color:#9aa5b1;">Ссылка действует 48 часов. Если вы не регистрировались на ASUTPORT — просто проигнорируйте это письмо.</p>`,
		html.EscapeString(name),
		html.EscapeString(data.ConfirmURL),
		html.EscapeString(data.ConfirmURL),
		html.EscapeString(data.ConfirmURL),
	)
	return layout("Подтверждение регистрации", "Подтвердите email", body)
}

func RegistrationText(data RegistrationMail) string {
	name := strings.TrimSpace(data.FullName)
	if name == "" {
		name = "коллега"
	}
	return fmt.Sprintf(`Здравствуйте, %s!

Вы начали регистрацию на платформе ASUTPORT. Подтвердите адрес email:

%s

Ссылка действует 48 часов. Если вы не регистрировались — проигнорируйте это письмо.`, name, data.ConfirmURL)
}

func AdminRegistrationHTML(data AdminRegistrationMail) string {
	rows := []struct{ label, value string }{
		{"Email", data.UserEmail},
		{"ФИО", data.FullName},
		{"Тип аккаунта", accountTypeLabel(data.AccountType)},
		{"Организация", data.OrgName},
		{"Тип организации", orgTypeLabel(data.OrgType)},
		{"ID регистрации", data.RegID},
		{"Время", data.RegisteredAt},
	}
	var rowsHTML strings.Builder
	for _, row := range rows {
		if strings.TrimSpace(row.value) == "" {
			continue
		}
		rowsHTML.WriteString(fmt.Sprintf(`
<tr>
  <td style="padding:10px 12px;border-bottom:1px solid #e8ecef;font-size:13px;color:#6b7280;width:38%%;vertical-align:top;">%s</td>
  <td style="padding:10px 12px;border-bottom:1px solid #e8ecef;font-size:13px;color:#111827;vertical-align:top;">%s</td>
</tr>`, html.EscapeString(row.label), html.EscapeString(row.value)))
	}
	body := fmt.Sprintf(`
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#1f2933;">На платформе зарегистрирован новый пользователь. Email ещё не подтверждён — ожидается переход по ссылке из письма.</p>
<table role="presentation" cellspacing="0" cellpadding="0" width="100%%" style="margin:0 0 24px;border:1px solid #e8ecef;border-radius:8px;border-collapse:separate;overflow:hidden;">
  %s
</table>
<table role="presentation" cellspacing="0" cellpadding="0">
  <tr>
    <td style="border-radius:8px;background:#1b2025;">
      <a href="%s" style="display:inline-block;padding:12px 22px;font-size:14px;font-weight:600;color:#3fc8b7;text-decoration:none;">Открыть админку</a>
    </td>
  </tr>
</table>`,
		rowsHTML.String(),
		html.EscapeString(data.AdminPanelURL),
	)
	return layout("Новая регистрация", "Уведомление администратора", body)
}

func AdminRegistrationText(data AdminRegistrationMail) string {
	return fmt.Sprintf(`Новая регистрация на ASUTPORT

Email: %s
ФИО: %s
Тип аккаунта: %s
Организация: %s
Тип организации: %s
ID регистрации: %s
Время: %s

Админка: %s`,
		data.UserEmail,
		data.FullName,
		accountTypeLabel(data.AccountType),
		data.OrgName,
		orgTypeLabel(data.OrgType),
		data.RegID,
		data.RegisteredAt,
		data.AdminPanelURL,
	)
}

func SMTPTestHTML() string {
	body := `<p style="margin:0;font-size:15px;line-height:1.6;color:#4b5563;">Это тестовое письмо из админки ASUTPORT. Если вы его получили — SMTP настроен верно.</p>`
	return layout("Тест SMTP", "Проверка SMTP", body)
}

func SMTPTestText() string {
	return "Это тестовое письмо из админки ASUTPORT. Если вы его получили — SMTP настроен верно."
}

func layout(title, eyebrow, bodyHTML string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin:0;padding:0;background:#eef1f4;font-family:'Segoe UI',Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#eef1f4;padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:560px;background:#ffffff;border:1px solid #dfe5eb;border-radius:12px;overflow:hidden;">
          <tr>
            <td style="background:#131619;padding:24px 28px;">
              <table role="presentation" cellspacing="0" cellpadding="0">
                <tr>
                  <td style="width:40px;height:40px;border-radius:10px;background:#0f2f2b;text-align:center;vertical-align:middle;font-size:16px;font-weight:700;color:#3fc8b7;">A</td>
                  <td style="padding-left:12px;vertical-align:middle;">
                    <div style="font-size:14px;font-weight:700;letter-spacing:0.12em;color:#e6eaee;">ASUTPORT</div>
                    <div style="font-size:11px;color:#93a0ac;margin-top:2px;">%s</div>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:28px;">
              %s
            </td>
          </tr>
          <tr>
            <td style="padding:18px 28px;background:#f8fafb;border-top:1px solid #e8ecef;font-size:12px;line-height:1.5;color:#9aa5b1;">
              Письмо отправлено автоматически платформой ASUTPORT. Не отвечайте на него, если не ожидали.
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`,
		html.EscapeString(title),
		html.EscapeString(eyebrow),
		bodyHTML,
	)
}

func accountTypeLabel(raw string) string {
	switch strings.TrimSpace(raw) {
	case "client_personal":
		return "Личный кабинет"
	case "client_org":
		return "Эксплуатация"
	case "manufacturer":
		return "Производитель"
	case "vendor":
		return "Поставщик / вендор"
	case "integrator":
		return "Интегратор"
	default:
		return raw
	}
}

func orgTypeLabel(raw string) string {
	switch strings.TrimSpace(raw) {
	case "client_org":
		return "Клиент"
	case "manufacturer":
		return "Производитель"
	case "vendor":
		return "Поставщик"
	case "integrator":
		return "Интегратор"
	default:
		return raw
	}
}

type OnboardingTicketMail struct {
	UserEmail      string
	FullName       string
	OrgName        string
	TicketID       string
	TicketURL      string
	AdminTicketURL string
}

type TicketActivityMail struct {
	TicketID      string
	OrgName       string
	Subject       string
	Preview       string
	IsAdminTarget bool
	ClientURL     string
	AdminURL      string
}

type OrgReviewResultMail struct {
	OrgName   string
	Approved  bool
	Rationale string
	LoginURL  string
}

func OnboardingTicketHTML(data OnboardingTicketMail) string {
	name := strings.TrimSpace(data.FullName)
	if name == "" {
		name = "коллега"
	}
	body := fmt.Sprintf(`
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#1f2933;">Здравствуйте, <strong>%s</strong>!</p>
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#4b5563;">Email подтверждён. Для активации организации <strong>%s</strong> откройте тикет проверки и приложите подтверждающие документы.</p>
<table role="presentation" cellspacing="0" cellpadding="0" style="margin:0 0 24px;">
  <tr>
    <td style="border-radius:8px;background:#0d9488;">
      <a href="%s" style="display:inline-block;padding:14px 28px;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;">Открыть тикет</a>
    </td>
  </tr>
</table>
<p style="margin:0;font-size:13px;line-height:1.5;color:#9aa5b1;">ID тикета: %s</p>`,
		html.EscapeString(name),
		html.EscapeString(data.OrgName),
		html.EscapeString(data.TicketURL),
		html.EscapeString(data.TicketID),
	)
	return layout("Тикет проверки", "Проверка организации", body)
}

func OnboardingTicketText(data OnboardingTicketMail) string {
	return fmt.Sprintf("Email подтверждён. Откройте тикет проверки организации %s: %s", data.OrgName, data.TicketURL)
}

func OnboardingTicketAdminHTML(data OnboardingTicketMail) string {
	body := fmt.Sprintf(`
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#1f2933;">Создан тикет проверки организации <strong>%s</strong>.</p>
<p style="margin:0 0 16px;font-size:13px;color:#6b7280;">Пользователь: %s (%s)</p>
<table role="presentation" cellspacing="0" cellpadding="0">
  <tr>
    <td style="border-radius:8px;background:#1b2025;">
      <a href="%s" style="display:inline-block;padding:12px 22px;font-size:14px;font-weight:600;color:#3fc8b7;text-decoration:none;">Открыть тикет</a>
    </td>
  </tr>
</table>`,
		html.EscapeString(data.OrgName),
		html.EscapeString(data.FullName),
		html.EscapeString(data.UserEmail),
		html.EscapeString(data.AdminTicketURL),
	)
	return layout("Тикет onboarding", "Уведомление администратора", body)
}

func TicketActivityHTML(data TicketActivityMail) string {
	body := fmt.Sprintf(`
<p style="margin:0 0 12px;font-size:15px;line-height:1.6;color:#1f2933;">%s</p>
<p style="margin:0 0 8px;font-size:13px;color:#6b7280;">Организация: %s</p>
<p style="margin:0 0 20px;font-size:13px;color:#4b5563;">%s</p>`,
		html.EscapeString(data.Subject),
		html.EscapeString(data.OrgName),
		html.EscapeString(data.Preview),
	)
	url := data.ClientURL
	if data.IsAdminTarget {
		url = data.AdminURL
	}
	body += fmt.Sprintf(`<table role="presentation" cellspacing="0" cellpadding="0"><tr><td style="border-radius:8px;background:#0d9488;"><a href="%s" style="display:inline-block;padding:12px 22px;font-size:14px;font-weight:600;color:#ffffff;text-decoration:none;">Открыть тикет</a></td></tr></table>`, html.EscapeString(url))
	return layout("Обновление тикета", "Уведомление", body)
}

func TicketActivityText(data TicketActivityMail) string {
	url := data.ClientURL
	if data.IsAdminTarget {
		url = data.AdminURL
	}
	return fmt.Sprintf("%s\n%s\n%s", data.Subject, data.Preview, url)
}

func OrgReviewResultHTML(data OrgReviewResultMail) string {
	title := "Организация активирована"
	if !data.Approved {
		title = "Заявка отклонена"
	}
	body := fmt.Sprintf(`
<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#1f2933;">Организация <strong>%s</strong>: %s</p>
<p style="margin:0 0 20px;font-size:14px;line-height:1.6;color:#4b5563;">%s</p>
<table role="presentation" cellspacing="0" cellpadding="0"><tr><td style="border-radius:8px;background:#0d9488;"><a href="%s" style="display:inline-block;padding:12px 22px;font-size:14px;font-weight:600;color:#ffffff;text-decoration:none;">Войти в кабинет</a></td></tr></table>`,
		html.EscapeString(data.OrgName),
		html.EscapeString(title),
		html.EscapeString(data.Rationale),
		html.EscapeString(data.LoginURL),
	)
	return layout(title, "Результат проверки", body)
}

func OrgReviewResultText(data OrgReviewResultMail) string {
	status := "активирована"
	if !data.Approved {
		status = "отклонена"
	}
	return fmt.Sprintf("Организация %s %s.\n%s\n%s", data.OrgName, status, data.Rationale, data.LoginURL)
}
