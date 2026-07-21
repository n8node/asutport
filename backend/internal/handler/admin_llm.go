package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/n8node/asutport/internal/config"
	"github.com/n8node/asutport/internal/llm"
)

type AdminLLMHandler struct {
	cfg *config.Config
	res *llm.Resolver
}

func NewAdminLLMHandler(cfg *config.Config, res *llm.Resolver) *AdminLLMHandler {
	return &AdminLLMHandler{cfg: cfg, res: res}
}

func (h *AdminLLMHandler) Get(w http.ResponseWriter, r *http.Request) {
	saved, err := h.res.LoadStored(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to load LLM settings")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": llm.PublicDTO(saved, strings.TrimSpace(h.cfg.OpenRouterAPIKey) != ""),
	})
}

type llmPatchRequest struct {
	Enabled      *bool  `json:"enabled"`
	Provider     string `json:"provider"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	QualifyModel string `json:"qualify_model"`
	AnswerModel  string `json:"answer_model"`
	VisionModel  string `json:"vision_model"`
	KBModel      string `json:"kb_model"`
	EmbedModel   string `json:"embed_model"`
}

func (h *AdminLLMHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var req llmPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json")
		return
	}
	saved, err := h.res.LoadStored(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to load LLM settings")
		return
	}
	if req.Enabled != nil {
		saved.Enabled = *req.Enabled
	}
	if strings.TrimSpace(req.Provider) != "" {
		saved.Provider = strings.TrimSpace(req.Provider)
	}
	if strings.TrimSpace(req.BaseURL) != "" {
		saved.BaseURL = strings.TrimSuffix(strings.TrimSpace(req.BaseURL), "/")
	}
	if strings.TrimSpace(req.QualifyModel) != "" {
		saved.QualifyModel = strings.TrimSpace(req.QualifyModel)
	}
	if strings.TrimSpace(req.AnswerModel) != "" {
		saved.AnswerModel = strings.TrimSpace(req.AnswerModel)
	}
	if strings.TrimSpace(req.VisionModel) != "" {
		saved.VisionModel = strings.TrimSpace(req.VisionModel)
	}
	if strings.TrimSpace(req.KBModel) != "" {
		saved.KBModel = strings.TrimSpace(req.KBModel)
	}
	if strings.TrimSpace(req.EmbedModel) != "" {
		embed := strings.TrimSpace(req.EmbedModel)
		if !strings.Contains(embed, "text-embedding-3-large") && !strings.Contains(embed, "embedding-3-large") {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"embed model must be text-embedding-3-large (vector dim 3072)")
			return
		}
		saved.EmbedModel = embed
	}
	if key := strings.TrimSpace(req.APIKey); key != "" {
		enc, err := llm.SealSecret(h.cfg.JWTSecret, key)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to protect API key")
			return
		}
		saved.APIKeyEnc = enc
	}
	if saved.Enabled && saved.APIKeyEnc == "" && strings.TrimSpace(h.cfg.OpenRouterAPIKey) == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "API key is required to enable platform LLM")
		return
	}
	if err := h.res.SaveStored(r.Context(), saved); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to save LLM settings")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": llm.PublicDTO(saved, strings.TrimSpace(h.cfg.OpenRouterAPIKey) != ""),
	})
}

type llmTestRequest struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

func (h *AdminLLMHandler) Test(w http.ResponseWriter, r *http.Request) {
	var req llmTestRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	saved, err := h.res.LoadStored(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to load LLM settings")
		return
	}
	base := strings.TrimSpace(req.BaseURL)
	if base == "" {
		base = saved.BaseURL
	}
	if base == "" {
		base = h.cfg.OpenRouterBaseURL
	}
	key := strings.TrimSpace(req.APIKey)
	if key == "" {
		key, _ = llm.OpenSecret(h.cfg.JWTSecret, saved.APIKeyEnc, h.cfg.OpenRouterAPIKey)
	}
	client := llm.NewClient(base, key)
	latency, terr := client.TestConnection(r.Context())
	saved.LastTestLatency = latency
	saved.LastTestAt = time.Now().UTC().Format(time.RFC3339)
	saved.LastTestOK = terr == nil
	_ = h.res.SaveStored(r.Context(), saved)

	msg := "connection ok"
	if terr != nil {
		msg = "connection failed"
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"ok":         terr == nil,
			"latency_ms": latency,
			"message":    msg,
		},
	})
}

func (h *AdminLLMHandler) Models(w http.ResponseWriter, r *http.Request) {
	saved, err := h.res.LoadStored(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to load LLM settings")
		return
	}
	key, kerr := llm.OpenSecret(h.cfg.JWTSecret, saved.APIKeyEnc, h.cfg.OpenRouterAPIKey)
	if kerr != nil || strings.TrimSpace(key) == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "save and test API key first")
		return
	}
	if !saved.LastTestOK {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "successful connection test required")
		return
	}
	base := saved.BaseURL
	if base == "" {
		base = h.cfg.OpenRouterBaseURL
	}
	client := llm.NewClient(base, key)
	models, err := client.ListModels(r.Context())
	if err != nil {
		WriteError(w, http.StatusBadGateway, "UPSTREAM", "could not load model list")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"models":      models,
			"recommended": llm.Recommended(models),
		},
	})
}
