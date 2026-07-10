package aiagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/timemachine-auto/timemachine/internal/config"
)

// StubAI is a deterministic AI backend used when no real provider is configured.
// It answers from the KB-like heuristics so the bot stays useful offline.
type StubAI struct{}

func (StubAI) Ask(ctx context.Context, system, user string) (string, error) {
	low := strings.ToLower(user)
	switch {
	case strings.Contains(low, "часы") || strings.Contains(low, "работае"):
		return "Мы работаем Пн–Сб 09:00–20:00, воскресенье — выходной.", nil
	case strings.Contains(low, "адрес") || strings.Contains(low, "где"):
		return "Москва, ул. Гаражная 12. Запись — на сайте или здесь.", nil
	case strings.Contains(low, "гаранти"):
		return "На работы — 6 месяцев, на расходники — по гарантии производителя.", nil
	}
	return "", nil
}

// OpenAIBackend talks to any OpenAI-compatible chat completions API.
type OpenAIBackend struct {
	cfg    config.AIConfig
	client *http.Client
}

func NewOpenAIBackend(cfg config.AIConfig) *OpenAIBackend {
	return &OpenAIBackend{cfg: cfg,
		client: &http.Client{Timeout: 30 * 1e9}}
}

func (o *OpenAIBackend) Ask(ctx context.Context, system, user string) (string, error) {
	if o.cfg.APIKey == "" {
		return "", fmt.Errorf("AI_API_KEY not set")
	}
	body, _ := json.Marshal(map[string]any{
		"model":      o.cfg.Model,
		"max_tokens": o.cfg.MaxTokens,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.3,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(o.cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ai api %d: %s", resp.StatusCode, string(rb))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", nil
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

// BuildAI returns the configured AI backend or a stub.
func BuildAI(cfg config.AIConfig) AIBackend {
	if !cfg.Enabled {
		return nil
	}
	switch cfg.Provider {
	case "openai":
		return NewOpenAIBackend(cfg)
	default:
		return StubAI{}
	}
}
