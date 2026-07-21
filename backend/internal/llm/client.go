package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const EmbedDim = 3072 // text-embedding-3-large (must match doc_chunks.embedding)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	base := strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "https://openrouter.ai/api/v1"
	}
	return &Client{
		BaseURL: base,
		APIKey:  strings.TrimSpace(apiKey),
		HTTP:    &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *Client) configured() error {
	if c == nil || c.APIKey == "" {
		return fmt.Errorf("llm: API key is not configured")
	}
	return nil
}

type embedRequest struct {
	Model          string `json:"model"`
	Input          any    `json:"input"`
	EncodingFormat string `json:"encoding_format,omitempty"`
	Dimensions     int    `json:"dimensions,omitempty"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Usage *struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Embed batches texts. Enforces EmbedDim unless dimensions override is set on the request via model defaults.
func (c *Client) Embed(ctx context.Context, model string, texts []string) ([][]float32, int, error) {
	if err := c.configured(); err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(model) == "" {
		return nil, 0, fmt.Errorf("llm: embedding model is required")
	}
	if len(texts) == 0 {
		return nil, 0, nil
	}
	body, err := json.Marshal(embedRequest{
		Model:          model,
		Input:          texts,
		EncodingFormat: "float",
		Dimensions:     EmbedDim,
	})
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	c.setHeaders(req)

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, 0, fmt.Errorf("embeddings http %d: %s", res.StatusCode, truncate(string(raw), 400))
	}
	var out embedResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, 0, err
	}
	if out.Error != nil && out.Error.Message != "" {
		return nil, 0, fmt.Errorf("embeddings: %s", out.Error.Message)
	}
	if len(out.Data) != len(texts) {
		return nil, 0, fmt.Errorf("embeddings: expected %d vectors, got %d", len(texts), len(out.Data))
	}
	vecs := make([][]float32, len(texts))
	for i := range texts {
		e := out.Data[i].Embedding
		if len(e) != EmbedDim {
			return nil, 0, fmt.Errorf("embeddings: dim %d != %d (use text-embedding-3-large)", len(e), EmbedDim)
		}
		v := make([]float32, EmbedDim)
		for j, x := range e {
			v[j] = float32(x)
		}
		vecs[i] = v
	}
	tokens := 0
	if out.Usage != nil {
		tokens = out.Usage.TotalTokens
		if tokens == 0 {
			tokens = out.Usage.PromptTokens
		}
	}
	return vecs, tokens, nil
}

func (c *Client) EmbedOne(ctx context.Context, model, text string) ([]float32, int, error) {
	vecs, tokens, err := c.Embed(ctx, model, []string{text})
	if err != nil {
		return nil, 0, err
	}
	if len(vecs) != 1 {
		return nil, 0, fmt.Errorf("embeddings: expected 1 vector")
	}
	return vecs[0], tokens, nil
}

// ListModels returns provider model ids via GET /models.
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	if err := c.configured(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("models http %d: %s", res.StatusCode, truncate(string(raw), 300))
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(payload.Data))
	for _, m := range payload.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

// TestConnection probes GET /models and returns latency.
func (c *Client) TestConnection(ctx context.Context) (latencyMS int64, err error) {
	start := time.Now()
	_, err = c.ListModels(ctx)
	latencyMS = time.Since(start).Milliseconds()
	return latencyMS, err
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if strings.Contains(strings.ToLower(c.BaseURL), "openrouter.ai") {
		req.Header.Set("HTTP-Referer", "https://asutport.ru")
		req.Header.Set("X-Title", "ASUTPORT")
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
