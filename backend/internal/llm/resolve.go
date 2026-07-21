package llm

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/n8node/asutport/internal/config"
	"github.com/n8node/asutport/internal/repository"
)

const AdminSettingLLMKey = "llm.platform"

// PlatformSettings is stored in admin_settings (encrypted API key).
type PlatformSettings struct {
	Enabled         bool   `json:"enabled"`
	Provider        string `json:"provider"`
	BaseURL         string `json:"base_url"`
	APIKeyEnc       string `json:"api_key_enc"`
	QualifyModel    string `json:"qualify_model"`
	AnswerModel     string `json:"answer_model"`
	VisionModel     string `json:"vision_model"`
	KBModel         string `json:"kb_model"`
	EmbedModel      string `json:"embed_model"`
	LastTestOK      bool   `json:"last_test_ok"`
	LastTestLatency int64  `json:"last_test_latency_ms"`
	LastTestAt      string `json:"last_test_at,omitempty"`
}

// Resolved is the runtime gateway config.
type Resolved struct {
	Enabled       bool
	Provider      string
	BaseURL       string
	APIKey        string
	FromDB        bool
	QualifyModel  string
	AnswerModel   string
	VisionModel   string
	KBModel       string
	EmbedModel    string
	HasAPIKey     bool
	LastTestOK    bool
	LastTestMS    int64
	LastTestAt    string
	EnvKeyPresent bool
}

type Resolver struct {
	repo *repository.AdminSettingsRepo
	cfg  *config.Config

	mu    sync.Mutex
	cache *Resolved
	until time.Time
}

func NewResolver(repo *repository.AdminSettingsRepo, cfg *config.Config) *Resolver {
	return &Resolver{repo: repo, cfg: cfg}
}

func (r *Resolver) Invalidate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = nil
	r.until = time.Time{}
}

func (r *Resolver) Resolve(ctx context.Context) (*Resolved, error) {
	r.mu.Lock()
	if r.cache != nil && time.Now().Before(r.until) {
		out := *r.cache
		r.mu.Unlock()
		return &out, nil
	}
	r.mu.Unlock()

	out := r.fromEnv()
	saved, err := r.LoadStored(ctx)
	if err == nil {
		out.LastTestOK = saved.LastTestOK
		out.LastTestMS = saved.LastTestLatency
		out.LastTestAt = saved.LastTestAt
		if saved.APIKeyEnc != "" {
			out.HasAPIKey = true
		}
		key, derr := OpenSecret(r.cfg.JWTSecret, saved.APIKeyEnc, "")
		useDB := saved.Enabled && derr == nil && strings.TrimSpace(key) != ""
		if useDB {
			base := strings.TrimSpace(saved.BaseURL)
			if base == "" {
				base = out.BaseURL
			}
			out.Enabled = true
			out.FromDB = true
			out.Provider = firstNonEmpty(saved.Provider, "openrouter")
			out.BaseURL = strings.TrimSuffix(base, "/")
			out.APIKey = strings.TrimSpace(key)
			out.HasAPIKey = true
		}
		out.QualifyModel = firstNonEmpty(saved.QualifyModel, out.QualifyModel)
		out.AnswerModel = firstNonEmpty(saved.AnswerModel, out.AnswerModel)
		out.VisionModel = firstNonEmpty(saved.VisionModel, out.VisionModel)
		out.KBModel = firstNonEmpty(saved.KBModel, out.KBModel)
		out.EmbedModel = firstNonEmpty(saved.EmbedModel, out.EmbedModel)
		if saved.Enabled {
			out.Enabled = useDB || out.APIKey != ""
		}
	}

	r.mu.Lock()
	r.cache = out
	r.until = time.Now().Add(5 * time.Second)
	r.mu.Unlock()
	return out, nil
}

func (r *Resolver) fromEnv() *Resolved {
	key := strings.TrimSpace(r.cfg.OpenRouterAPIKey)
	base := strings.TrimSuffix(strings.TrimSpace(r.cfg.OpenRouterBaseURL), "/")
	if base == "" {
		base = "https://openrouter.ai/api/v1"
	}
	return &Resolved{
		Enabled:       key != "",
		Provider:      "openrouter",
		BaseURL:       base,
		APIKey:        key,
		FromDB:        false,
		QualifyModel:  r.cfg.OpenRouterModelQualify,
		AnswerModel:   r.cfg.OpenRouterModelAnswer,
		VisionModel:   r.cfg.OpenRouterModelVision,
		KBModel:       r.cfg.OpenRouterModelKB,
		EmbedModel:    r.cfg.OpenRouterModelEmbed,
		HasAPIKey:     key != "",
		EnvKeyPresent: key != "",
	}
}

func (r *Resolver) Client(ctx context.Context) (*Client, *Resolved, error) {
	res, err := r.Resolve(ctx)
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(res.APIKey) == "" {
		return nil, res, errors.New("llm gateway is not configured")
	}
	return NewClient(res.BaseURL, res.APIKey), res, nil
}

func (r *Resolver) LoadStored(ctx context.Context) (PlatformSettings, error) {
	var saved PlatformSettings
	err := r.repo.Get(ctx, AdminSettingLLMKey, &saved)
	if errors.Is(err, repository.ErrNotFound) {
		return PlatformSettings{
			Provider:     "openrouter",
			BaseURL:      strings.TrimSuffix(strings.TrimSpace(r.cfg.OpenRouterBaseURL), "/"),
			QualifyModel: r.cfg.OpenRouterModelQualify,
			AnswerModel:  r.cfg.OpenRouterModelAnswer,
			VisionModel:  r.cfg.OpenRouterModelVision,
			KBModel:      r.cfg.OpenRouterModelKB,
			EmbedModel:   r.cfg.OpenRouterModelEmbed,
		}, nil
	}
	if err != nil {
		return PlatformSettings{}, err
	}
	if saved.Provider == "" {
		saved.Provider = "openrouter"
	}
	if saved.BaseURL == "" {
		saved.BaseURL = strings.TrimSuffix(strings.TrimSpace(r.cfg.OpenRouterBaseURL), "/")
	}
	if saved.QualifyModel == "" {
		saved.QualifyModel = r.cfg.OpenRouterModelQualify
	}
	if saved.AnswerModel == "" {
		saved.AnswerModel = r.cfg.OpenRouterModelAnswer
	}
	if saved.VisionModel == "" {
		saved.VisionModel = r.cfg.OpenRouterModelVision
	}
	if saved.KBModel == "" {
		saved.KBModel = r.cfg.OpenRouterModelKB
	}
	if saved.EmbedModel == "" {
		saved.EmbedModel = r.cfg.OpenRouterModelEmbed
	}
	return saved, nil
}

func (r *Resolver) SaveStored(ctx context.Context, s PlatformSettings) error {
	if err := r.repo.Upsert(ctx, AdminSettingLLMKey, s); err != nil {
		return err
	}
	r.Invalidate()
	return nil
}

func SealSecret(jwtSecret, value string) (string, error) {
	block, err := aes.NewCipher(settingsSecretKey(jwtSecret))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

func OpenSecret(jwtSecret, enc, fallback string) (string, error) {
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// PublicDTO hides the API key.
func PublicDTO(s PlatformSettings, envKeyPresent bool) map[string]any {
	return map[string]any{
		"enabled":              s.Enabled,
		"provider":             s.Provider,
		"base_url":             s.BaseURL,
		"has_api_key":          s.APIKeyEnc != "",
		"qualify_model":        s.QualifyModel,
		"answer_model":         s.AnswerModel,
		"vision_model":         s.VisionModel,
		"kb_model":             s.KBModel,
		"embed_model":          s.EmbedModel,
		"embed_dim":            EmbedDim,
		"last_test_ok":         s.LastTestOK,
		"last_test_latency_ms": s.LastTestLatency,
		"last_test_at":         s.LastTestAt,
		"env_key_present":      envKeyPresent,
	}
}
