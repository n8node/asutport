package handler

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/n8node/asutport/internal/config"
	"github.com/n8node/asutport/internal/repository"
	s3store "github.com/n8node/asutport/internal/s3"
)

const (
	adminSettingS3Key   = "storage.s3"
	adminSettingSMTPKey = "email.smtp"
)

type AdminSettingsHandler struct {
	cfg  *config.Config
	repo *repository.AdminSettingsRepo
}

func NewAdminSettingsHandler(cfg *config.Config, repo *repository.AdminSettingsRepo) *AdminSettingsHandler {
	return &AdminSettingsHandler{cfg: cfg, repo: repo}
}

type s3Settings struct {
	Enabled      bool   `json:"enabled"`
	Endpoint     string `json:"endpoint"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	AccessKeyID  string `json:"access_key_id"`
	SecretEnc    string `json:"secret_access_key_enc"`
	UsePathStyle bool   `json:"use_path_style"`
}

type s3SettingsDTO struct {
	Enabled      bool   `json:"enabled"`
	Endpoint     string `json:"endpoint"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	AccessKeyID  string `json:"access_key_id"`
	HasSecret    bool   `json:"has_secret"`
	UsePathStyle bool   `json:"use_path_style"`
}

type s3SettingsPatch struct {
	Enabled         bool    `json:"enabled"`
	Endpoint        string  `json:"endpoint"`
	Bucket          string  `json:"bucket"`
	Region          string  `json:"region"`
	AccessKeyID     string  `json:"access_key_id"`
	SecretAccessKey *string `json:"secret_access_key"`
	UsePathStyle    bool    `json:"use_path_style"`
}

type smtpSettings struct {
	Enabled          bool   `json:"enabled"`
	FromEmail        string `json:"from_email"`
	FromName         string `json:"from_name"`
	ForceFromEmail   bool   `json:"force_from_email"`
	ForceFromName    bool   `json:"force_from_name"`
	ReplyToFromEmail bool   `json:"reply_to_from_email"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Encryption       string `json:"encryption"`
	AutoTLS          bool   `json:"auto_tls"`
	Auth             bool   `json:"auth"`
	Username         string `json:"username"`
	PasswordEnc      string `json:"password_enc"`
}

type smtpSettingsDTO struct {
	Settings         smtpPublicSettings `json:"settings"`
	PasswordSet      bool               `json:"password_set"`
	PasswordHint     string             `json:"password_hint"`
	YandexPresetHost string             `json:"yandex_preset_host"`
	YandexPresetPort int                `json:"yandex_preset_port"`
}

type smtpPublicSettings struct {
	Enabled          bool   `json:"enabled"`
	FromEmail        string `json:"from_email"`
	FromName         string `json:"from_name"`
	ForceFromEmail   bool   `json:"force_from_email"`
	ForceFromName    bool   `json:"force_from_name"`
	ReplyToFromEmail bool   `json:"reply_to_from_email"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	Encryption       string `json:"encryption"`
	AutoTLS          bool   `json:"auto_tls"`
	Auth             bool   `json:"auth"`
	Username         string `json:"username"`
}

type smtpUpdateRequest struct {
	Settings smtpPublicSettings `json:"settings"`
	Password *string            `json:"password"`
}

type smtpTestRequest struct {
	To string `json:"to"`
}

func (h *AdminSettingsHandler) S3Get(w http.ResponseWriter, r *http.Request) {
	settings := h.loadS3(r.Context())
	WriteJSON(w, http.StatusOK, map[string]any{"data": h.s3DTO(settings)})
}

func (h *AdminSettingsHandler) S3Patch(w http.ResponseWriter, r *http.Request) {
	var req s3SettingsPatch
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json")
		return
	}
	current := h.loadS3(r.Context())
	next := s3Settings{
		Enabled:      req.Enabled,
		Endpoint:     strings.TrimRight(strings.TrimSpace(req.Endpoint), "/"),
		Bucket:       strings.TrimSpace(req.Bucket),
		Region:       strings.TrimSpace(req.Region),
		AccessKeyID:  strings.TrimSpace(req.AccessKeyID),
		SecretEnc:    current.SecretEnc,
		UsePathStyle: req.UsePathStyle,
	}
	if next.Region == "" {
		next.Region = "us-east-1"
	}
	if req.SecretAccessKey != nil && strings.TrimSpace(*req.SecretAccessKey) != "" {
		enc, err := h.seal(strings.TrimSpace(*req.SecretAccessKey))
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to protect secret")
			return
		}
		next.SecretEnc = enc
	}
	if next.Enabled && (next.Endpoint == "" || next.Bucket == "" || next.AccessKeyID == "" || next.SecretEnc == "") {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "endpoint, bucket and credentials are required when S3 is enabled")
		return
	}
	if err := h.repo.Upsert(r.Context(), adminSettingS3Key, next); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to save S3 settings")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": h.s3DTO(next)})
}

func (h *AdminSettingsHandler) S3Test(w http.ResponseWriter, r *http.Request) {
	settings := h.loadS3(r.Context())
	secret, err := h.openSecret(settings.SecretEnc, h.cfg.S3SecretKey)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "STORAGE_TEST_FAILED", "saved S3 secret cannot be read")
		return
	}
	client, err := s3store.NewClient(&config.Config{
		S3Endpoint:     settings.Endpoint,
		S3Region:       settings.Region,
		S3Bucket:       settings.Bucket,
		S3AccessKey:    settings.AccessKeyID,
		S3SecretKey:    secret,
		S3UsePathStyle: settings.UsePathStyle,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "STORAGE_TEST_FAILED", "S3 settings are incomplete")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		WriteError(w, http.StatusBadRequest, "STORAGE_TEST_FAILED", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": true}})
}

func (h *AdminSettingsHandler) S3CorsHints(w http.ResponseWriter, _ *http.Request) {
	site := "https://" + strings.TrimSpace(h.cfg.Domain)
	if h.cfg.Domain == "" {
		site = "https://asutport.ru"
	}
	cors := fmt.Sprintf(`<CORSConfiguration>
  <CORSRule>
    <AllowedOrigin>%s</AllowedOrigin>
    <AllowedOrigin>http://localhost:3000</AllowedOrigin>
    <AllowedOrigin>http://127.0.0.1:3000</AllowedOrigin>
    <AllowedMethod>GET</AllowedMethod>
    <AllowedMethod>PUT</AllowedMethod>
    <AllowedMethod>POST</AllowedMethod>
    <AllowedMethod>DELETE</AllowedMethod>
    <AllowedMethod>HEAD</AllowedMethod>
    <AllowedHeader>*</AllowedHeader>
    <ExposeHeader>ETag</ExposeHeader>
  </CORSRule>
</CORSConfiguration>`, site)
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"allowed_origins": []string{site, "http://localhost:3000", "http://127.0.0.1:3000"},
			"cors_xml":        cors,
		},
	})
}

func (h *AdminSettingsHandler) SMTPGet(w http.ResponseWriter, r *http.Request) {
	settings := h.loadSMTP(r.Context())
	WriteJSON(w, http.StatusOK, map[string]any{"data": h.smtpDTO(settings)})
}

func (h *AdminSettingsHandler) SMTPPatch(w http.ResponseWriter, r *http.Request) {
	var req smtpUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json")
		return
	}
	current := h.loadSMTP(r.Context())
	next := smtpSettings{
		Enabled:          req.Settings.Enabled,
		FromEmail:        strings.TrimSpace(req.Settings.FromEmail),
		FromName:         strings.TrimSpace(req.Settings.FromName),
		ForceFromEmail:   req.Settings.ForceFromEmail,
		ForceFromName:    req.Settings.ForceFromName,
		ReplyToFromEmail: req.Settings.ReplyToFromEmail,
		Host:             strings.TrimSpace(req.Settings.Host),
		Port:             req.Settings.Port,
		Encryption:       strings.ToLower(strings.TrimSpace(req.Settings.Encryption)),
		AutoTLS:          req.Settings.AutoTLS,
		Auth:             req.Settings.Auth,
		Username:         strings.TrimSpace(req.Settings.Username),
		PasswordEnc:      current.PasswordEnc,
	}
	if next.Port == 0 {
		next.Port = 465
	}
	if next.Encryption == "" {
		next.Encryption = "ssl"
	}
	if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
		enc, err := h.seal(strings.TrimSpace(*req.Password))
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to protect SMTP password")
			return
		}
		next.PasswordEnc = enc
	}
	if err := validateSMTP(next); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.repo.Upsert(r.Context(), adminSettingSMTPKey, next); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to save SMTP settings")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": h.smtpDTO(next)})
}

func (h *AdminSettingsHandler) SMTPTest(w http.ResponseWriter, r *http.Request) {
	var req smtpTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json")
		return
	}
	settings := h.loadSMTP(r.Context())
	password, err := h.openSecret(settings.PasswordEnc, "")
	if err != nil {
		WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": false, "message": "SMTP password cannot be read"}})
		return
	}
	if err := sendSMTPTest(r.Context(), settings, password, strings.TrimSpace(req.To)); err != nil {
		WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": false, "message": err.Error()}})
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"ok": true, "message": "Письмо отправлено"}})
}

func (h *AdminSettingsHandler) loadS3(ctx context.Context) s3Settings {
	def := s3Settings{
		Enabled:      h.cfg.S3Configured(),
		Endpoint:     strings.TrimRight(strings.TrimSpace(h.cfg.S3Endpoint), "/"),
		Bucket:       strings.TrimSpace(h.cfg.S3Bucket),
		Region:       strings.TrimSpace(h.cfg.S3Region),
		AccessKeyID:  strings.TrimSpace(h.cfg.S3AccessKey),
		UsePathStyle: h.cfg.S3UsePathStyle,
	}
	if strings.TrimSpace(h.cfg.S3SecretKey) != "" {
		if enc, err := h.seal(strings.TrimSpace(h.cfg.S3SecretKey)); err == nil {
			def.SecretEnc = enc
		}
	}
	var saved s3Settings
	if err := h.repo.Get(ctx, adminSettingS3Key, &saved); err != nil {
		return def
	}
	return saved
}

func (h *AdminSettingsHandler) s3DTO(s s3Settings) s3SettingsDTO {
	return s3SettingsDTO{
		Enabled:      s.Enabled,
		Endpoint:     s.Endpoint,
		Bucket:       s.Bucket,
		Region:       s.Region,
		AccessKeyID:  s.AccessKeyID,
		HasSecret:    s.SecretEnc != "",
		UsePathStyle: s.UsePathStyle,
	}
}

func (h *AdminSettingsHandler) loadSMTP(ctx context.Context) smtpSettings {
	def := smtpSettings{
		Enabled:        false,
		FromName:       "ASUTPORT",
		ForceFromEmail: true,
		ForceFromName:  true,
		Host:           "",
		Port:           465,
		Encryption:     "ssl",
		AutoTLS:        true,
		Auth:           true,
	}
	var saved smtpSettings
	if err := h.repo.Get(ctx, adminSettingSMTPKey, &saved); err != nil {
		return def
	}
	return saved
}

func (h *AdminSettingsHandler) smtpDTO(s smtpSettings) smtpSettingsDTO {
	password := ""
	if s.PasswordEnc != "" {
		password, _ = h.openSecret(s.PasswordEnc, "")
	}
	return smtpSettingsDTO{
		Settings: smtpPublicSettings{
			Enabled:          s.Enabled,
			FromEmail:        s.FromEmail,
			FromName:         s.FromName,
			ForceFromEmail:   s.ForceFromEmail,
			ForceFromName:    s.ForceFromName,
			ReplyToFromEmail: s.ReplyToFromEmail,
			Host:             s.Host,
			Port:             s.Port,
			Encryption:       s.Encryption,
			AutoTLS:          s.AutoTLS,
			Auth:             s.Auth,
			Username:         s.Username,
		},
		PasswordSet:      s.PasswordEnc != "",
		PasswordHint:     maskSecret(password),
		YandexPresetHost: "smtp.yandex.ru",
		YandexPresetPort: 465,
	}
}

func validateSMTP(s smtpSettings) error {
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("invalid SMTP port")
	}
	switch s.Encryption {
	case "none", "ssl", "tls":
	default:
		return fmt.Errorf("invalid SMTP encryption")
	}
	if s.Enabled {
		if _, err := mail.ParseAddress(s.FromEmail); err != nil {
			return fmt.Errorf("valid sender email is required")
		}
		if s.Host == "" {
			return fmt.Errorf("SMTP host is required")
		}
		if s.Auth && s.Username == "" {
			return fmt.Errorf("SMTP username is required")
		}
	}
	return nil
}

func sendSMTPTest(ctx context.Context, s smtpSettings, password, to string) error {
	if !s.Enabled {
		return fmt.Errorf("включите отправку email в настройках")
	}
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("укажите корректный email для тестовой отправки")
	}
	if err := validateSMTP(s); err != nil {
		return err
	}
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	var err error
	if s.Encryption == "ssl" {
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

	if s.Encryption == "tls" || (s.Encryption == "none" && s.AutoTLS) {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("SMTP STARTTLS failed: %w", err)
			}
		}
	}
	if s.Auth {
		if password == "" {
			return fmt.Errorf("SMTP password is required")
		}
		if err := client.Auth(smtp.PlainAuth("", s.Username, password, s.Host)); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}
	from := s.FromEmail
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP sender rejected: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP recipient rejected: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	body := strings.Join([]string{
		"From: " + (&mail.Address{Name: s.FromName, Address: from}).String(),
		"To: " + to,
		"Subject: ASUTPORT — тест SMTP",
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		"<p>Это тестовое письмо из админки ASUTPORT. Если вы его получили — SMTP настроен верно.</p>",
	}, "\r\n")
	if _, err := io.WriteString(w, body); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}
	return client.Quit()
}

func (h *AdminSettingsHandler) seal(value string) (string, error) {
	block, err := aes.NewCipher(secretKey(h.cfg.JWTSecret))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

func (h *AdminSettingsHandler) openSecret(enc, fallback string) (string, error) {
	if enc == "" {
		return fallback, nil
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(secretKey(h.cfg.JWTSecret))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid secret")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func secretKey(secret string) []byte {
	sum := sha256.Sum256([]byte("asutport-admin-settings:" + secret))
	return sum[:]
}

func maskSecret(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return "••••"
	}
	return v[:2] + "••••" + v[len(v)-2:]
}
