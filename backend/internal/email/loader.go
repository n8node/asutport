package email

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/n8node/asutport/internal/repository"
)

const adminSettingSMTPKey = "email.smtp"

type smtpStored struct {
	Enabled            bool   `json:"enabled"`
	FromEmail          string `json:"from_email"`
	FromName           string `json:"from_name"`
	ForceFromEmail     bool   `json:"force_from_email"`
	ForceFromName      bool   `json:"force_from_name"`
	ReplyToFromEmail   bool   `json:"reply_to_from_email"`
	AdminNotifyEmail   string `json:"admin_notify_email"`
	AdminNotifyEnabled bool   `json:"admin_notify_enabled"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Encryption         string `json:"encryption"`
	AutoTLS            bool   `json:"auto_tls"`
	Auth               bool   `json:"auth"`
	Username           string `json:"username"`
	PasswordEnc        string `json:"password_enc"`
}

type Loader struct {
	repo      *repository.AdminSettingsRepo
	jwtSecret string
}

func NewLoader(repo *repository.AdminSettingsRepo, jwtSecret string) *Loader {
	return &Loader{repo: repo, jwtSecret: jwtSecret}
}

func (l *Loader) Load(ctx context.Context) (Settings, error) {
	def := smtpStored{
		Enabled:            false,
		FromName:           "ASUTPORT",
		Port:               465,
		Encryption:         "ssl",
		AutoTLS:            true,
		Auth:               true,
		AdminNotifyEnabled: true,
	}
	var saved smtpStored
	if err := l.repo.Get(ctx, adminSettingSMTPKey, &saved); err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return Settings{}, err
		}
		saved = def
	}
	password, err := openSecret(l.jwtSecret, saved.PasswordEnc)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		Enabled:            saved.Enabled,
		FromEmail:          saved.FromEmail,
		FromName:           saved.FromName,
		ForceFromEmail:     saved.ForceFromEmail,
		ForceFromName:      saved.ForceFromName,
		ReplyToFromEmail:   saved.ReplyToFromEmail,
		AdminNotifyEmail:   saved.AdminNotifyEmail,
		AdminNotifyEnabled: saved.AdminNotifyEnabled,
		Host:               saved.Host,
		Port:               saved.Port,
		Encryption:         saved.Encryption,
		AutoTLS:            saved.AutoTLS,
		Auth:               saved.Auth,
		Username:           saved.Username,
		Password:           password,
	}, nil
}

func openSecret(jwtSecret, enc string) (string, error) {
	if enc == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(secretKey(jwtSecret))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid secret")
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
