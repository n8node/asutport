package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const pageParseSystemPrompt = `You parse one page of industrial automation documentation (PLC, SCADA, HMI, drives, sensors).
Return structured markdown ONLY (no fences, no commentary).
Rules:
- Preserve tables as markdown tables (merged cells as repeated values if needed).
- Formulas: keep LaTeX if visible AND add a short plain-language reading.
- Schemes/drawings: describe as [image: ...] with key labels and connections.
- Keep register addresses, parameter names, units, and version notes exactly.
- Empty or blank page → return empty string.
- Language of the page: keep original (usually Russian).`

type ChatUsage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}

type chatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ParsePageImage sends a PNG page to the vision model and returns markdown.
func (c *Client) ParsePageImage(ctx context.Context, model string, pageNum int, png []byte) (string, ChatUsage, error) {
	var u ChatUsage
	if err := c.configured(); err != nil {
		return "", u, err
	}
	m := strings.TrimSpace(model)
	if m == "" {
		return "", u, fmt.Errorf("llm: vision model is required")
	}
	if len(png) == 0 {
		return "", u, fmt.Errorf("llm: empty page image")
	}

	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	userParts := []map[string]any{
		{"type": "text", "text": fmt.Sprintf("Parse page %d of the manual into markdown for RAG indexing.", pageNum)},
		{"type": "image_url", "image_url": map[string]string{"url": dataURI}},
	}
	userContent, err := json.Marshal(userParts)
	if err != nil {
		return "", u, err
	}
	sysContent, _ := json.Marshal(pageParseSystemPrompt)
	type msg struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	body, err := json.Marshal(map[string]any{
		"model":       m,
		"messages":    []msg{{Role: "system", Content: sysContent}, {Role: "user", Content: userContent}},
		"temperature": 0.1,
	})
	if err != nil {
		return "", u, err
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 180 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", u, err
	}
	c.setHeaders(req)

	res, err := httpClient.Do(req)
	if err != nil {
		return "", u, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", u, fmt.Errorf("vision http %d: %s", res.StatusCode, truncate(string(raw), 400))
	}
	var out chatResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", u, err
	}
	if out.Error != nil && out.Error.Message != "" {
		return "", u, fmt.Errorf("vision: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", u, fmt.Errorf("vision: empty choices")
	}
	text := strings.TrimSpace(out.Choices[0].Message.Content)
	text = strings.TrimPrefix(text, "```markdown")
	text = strings.TrimPrefix(text, "```md")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	u.PromptTokens = int64(out.Usage.PromptTokens)
	u.CompletionTokens = int64(out.Usage.CompletionTokens)
	u.TotalTokens = int64(out.Usage.TotalTokens)
	if u.TotalTokens == 0 {
		u.TotalTokens = u.PromptTokens + u.CompletionTokens
	}
	return text, u, nil
}
