package s3store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/n8node/asutport/internal/config"
	"github.com/n8node/asutport/internal/repository"
)

const adminSettingS3Key = "storage.s3"

type storedS3Settings struct {
	Enabled      bool   `json:"enabled"`
	Endpoint     string `json:"endpoint"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	AccessKeyID  string `json:"access_key_id"`
	SecretEnc    string `json:"secret_access_key_enc"`
	UsePathStyle bool   `json:"use_path_style"`
}

type Loader struct {
	repo *repository.AdminSettingsRepo
	env  *config.Config
}

func NewLoader(repo *repository.AdminSettingsRepo, env *config.Config) *Loader {
	return &Loader{repo: repo, env: env}
}

func (l *Loader) Client(ctx context.Context) (*Client, error) {
	settings, err := l.load(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, fmt.Errorf("object storage is not configured")
	}
	secret, err := openSettingsSecret(l.env.JWTSecret, settings.SecretEnc, l.env.S3SecretKey)
	if err != nil {
		return nil, fmt.Errorf("object storage is not configured")
	}
	region := strings.TrimSpace(settings.Region)
	if region == "" {
		region = "us-east-1"
	}
	endpoint := strings.TrimRight(strings.TrimSpace(settings.Endpoint), "/")
	bucket := strings.TrimSpace(settings.Bucket)
	accessKey := strings.TrimSpace(settings.AccessKeyID)
	if endpoint == "" || bucket == "" || accessKey == "" || secret == "" {
		return nil, fmt.Errorf("object storage is not configured")
	}
	return NewClient(&config.Config{
		S3Endpoint:     endpoint,
		S3Region:       region,
		S3Bucket:       bucket,
		S3AccessKey:    accessKey,
		S3SecretKey:    secret,
		S3UsePathStyle: settings.UsePathStyle,
	})
}

func (l *Loader) Ping(ctx context.Context) error {
	client, err := l.Client(ctx)
	if err != nil {
		return err
	}
	return client.Ping(ctx)
}

func (l *Loader) load(ctx context.Context) (storedS3Settings, error) {
	def := storedS3Settings{
		Enabled:      l.env.S3Configured(),
		Endpoint:     strings.TrimRight(strings.TrimSpace(l.env.S3Endpoint), "/"),
		Bucket:       strings.TrimSpace(l.env.S3Bucket),
		Region:       strings.TrimSpace(l.env.S3Region),
		AccessKeyID:  strings.TrimSpace(l.env.S3AccessKey),
		UsePathStyle: l.env.S3UsePathStyle,
	}
	var saved storedS3Settings
	if err := l.repo.Get(ctx, adminSettingS3Key, &saved); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return def, nil
		}
		return storedS3Settings{}, err
	}
	return saved, nil
}

func openSettingsSecret(jwtSecret, enc, fallback string) (string, error) {
	if enc == "" {
		return strings.TrimSpace(fallback), nil
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(settingsSecretKey(jwtSecret))
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

func settingsSecretKey(secret string) []byte {
	sum := sha256.Sum256([]byte("asutport-admin-settings:" + secret))
	return sum[:]
}
