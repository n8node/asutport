package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Settings struct {
	Enabled            bool
	FromEmail          string
	FromName           string
	ForceFromEmail     bool
	ForceFromName      bool
	ReplyToFromEmail   bool
	AdminNotifyEmail   string
	AdminNotifyEnabled bool
	Host               string
	Port               int
	Encryption         string
	AutoTLS            bool
	Auth               bool
	Username           string
	Password           string
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
	if s.AdminNotifyEnabled {
		if to := strings.TrimSpace(s.AdminNotifyEmail); to != "" {
			if _, err := mail.ParseAddress(to); err != nil {
				return fmt.Errorf("invalid admin notify email")
			}
		}
	}
	return nil
}

type Message struct {
	To      string
	Subject string
	Text    string
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
	if strings.TrimSpace(msg.HTML) == "" && strings.TrimSpace(msg.Text) == "" {
		return fmt.Errorf("message body is required")
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

	body, err := buildMIME(from, fromName, msg, s.ReplyToFromEmail)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}
	if _, err := io.WriteString(w, body); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}
	return client.Quit()
}

func buildMIME(from, fromName string, msg Message, replyToFrom bool) (string, error) {
	text := strings.TrimSpace(msg.Text)
	htmlBody := strings.TrimSpace(msg.HTML)
	if text == "" && htmlBody != "" {
		text = stripHTMLFallback(htmlBody)
	}

	var buf bytes.Buffer
	buf.WriteString("From: " + (&mail.Address{Name: fromName, Address: from}).String() + "\r\n")
	buf.WriteString("To: " + msg.To + "\r\n")
	buf.WriteString("Subject: " + encodeHeader(msg.Subject) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	if replyToFrom {
		buf.WriteString("Reply-To: " + from + "\r\n")
	}

	if text != "" && htmlBody != "" {
		boundary, err := randomBoundary()
		if err != nil {
			return "", err
		}
		buf.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")
		if err := writePart(&buf, boundary, "text/plain; charset=UTF-8", text); err != nil {
			return "", err
		}
		if err := writePart(&buf, boundary, "text/html; charset=UTF-8", htmlBody); err != nil {
			return "", err
		}
		buf.WriteString("--" + boundary + "--\r\n")
		return buf.String(), nil
	}

	content := text
	contentType := "text/plain; charset=UTF-8"
	if htmlBody != "" {
		content = htmlBody
		contentType = "text/html; charset=UTF-8"
	}
	buf.WriteString("Content-Type: " + contentType + "\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	qp := quotedprintable.NewWriter(&buf)
	if _, err := qp.Write([]byte(content)); err != nil {
		return "", err
	}
	if err := qp.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writePart(buf *bytes.Buffer, boundary, contentType, content string) error {
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: " + contentType + "\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	qp := quotedprintable.NewWriter(buf)
	if _, err := qp.Write([]byte(content)); err != nil {
		return err
	}
	return qp.Close()
}

func randomBoundary() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "asutport-" + hex.EncodeToString(raw), nil
}

func stripHTMLFallback(s string) string {
	replacer := strings.NewReplacer(
		"<br>", "\n", "<br/>", "\n", "<br />", "\n",
		"</p>", "\n", "</tr>", "\n", "</div>", "\n",
	)
	out := replacer.Replace(s)
	var b strings.Builder
	inTag := false
	for _, r := range out {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
