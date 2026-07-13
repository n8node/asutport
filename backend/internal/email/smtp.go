package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Settings struct {
	Enabled          bool
	FromEmail        string
	FromName         string
	ForceFromEmail   bool
	ForceFromName    bool
	ReplyToFromEmail bool
	Host             string
	Port             int
	Encryption       string
	AutoTLS          bool
	Auth             bool
	Username         string
	Password         string
}

func Validate(s Settings) error {
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("invalid SMTP port")
	}
	switch strings.ToLower(s.Encryption) {
	case "none", "ssl", "tls":
	default:
		return fmt.Errorf("invalid SMTP encryption")
	}
	if s.Enabled {
		if _, err := mail.ParseAddress(s.FromEmail); err != nil {
			return fmt.Errorf("valid sender email is required")
		}
		if strings.TrimSpace(s.Host) == "" {
			return fmt.Errorf("SMTP host is required")
		}
		if s.Auth && strings.TrimSpace(s.Username) == "" {
			return fmt.Errorf("SMTP username is required")
		}
	}
	return nil
}

type Message struct {
	To      string
	Subject string
	HTML    string
}

func Send(ctx context.Context, s Settings, msg Message) error {
	if !s.Enabled {
		return fmt.Errorf("отправка email отключена в настройках")
	}
	if err := Validate(s); err != nil {
		return err
	}
	if _, err := mail.ParseAddress(msg.To); err != nil {
		return fmt.Errorf("некорректный адрес получателя")
	}
	if strings.TrimSpace(msg.Subject) == "" {
		return fmt.Errorf("subject is required")
	}

	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	var err error
	if strings.ToLower(s.Encryption) == "ssl" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("SMTP connect failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return fmt.Errorf("SMTP client failed: %w", err)
	}
	defer client.Close()

	enc := strings.ToLower(s.Encryption)
	if enc == "tls" || (enc == "none" && s.AutoTLS) {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("SMTP STARTTLS failed: %w", err)
			}
		}
	}
	if s.Auth {
		if strings.TrimSpace(s.Password) == "" {
			return fmt.Errorf("SMTP password is required")
		}
		if err := client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	from := s.FromEmail
	fromName := s.FromName
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP sender rejected: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("SMTP recipient rejected: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	headers := []string{
		"From: " + (&mail.Address{Name: fromName, Address: from}).String(),
		"To: " + msg.To,
		"Subject: " + msg.Subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	if s.ReplyToFromEmail {
		headers = append(headers, "Reply-To: "+from)
	}
	body := strings.Join(headers, "\r\n") + "\r\n\r\n" + msg.HTML
	if _, err := io.WriteString(w, body); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}
	return client.Quit()
}

func RegistrationHTML(confirmURL, fullName string) string {
	name := strings.TrimSpace(fullName)
	if name == "" {
		name = "коллега"
	}
	return fmt.Sprintf(`<p>Здравствуйте, %s!</p>
<p>Вы зарегистрировались на платформе ASUTPORT. Подтвердите адрес email, чтобы завершить регистрацию:</p>
<p><a href="%s">Подтвердить регистрацию</a></p>
<p>Если кнопка не открывается, скопируйте ссылку в браузер:</p>
<p><a href="%s">%s</a></p>
<p>Ссылка действует 48 часов. Если вы не регистрировались на ASUTPORT — просто проигнорируйте это письмо.</p>`, name, confirmURL, confirmURL, confirmURL)
}
