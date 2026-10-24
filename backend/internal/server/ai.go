package server

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

type AIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// AIClient talks to any OpenAI-compatible chat completions API (OpenAI, Gemini's OpenAI endpoint, OpenRouter, ...).
type AIClient struct {
	cfg  AIConfig
	http *http.Client
}

func NewAIClient(cfg AIConfig) *AIClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &AIClient{cfg: cfg, http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *AIClient) Enabled() bool { return c != nil && c.cfg.APIKey != "" }

func (c *AIClient) chatJSON(ctx context.Context, system, user string, out any) error {
	body, err := json.Marshal(map[string]any{
		"model":           c.cfg.Model,
		"temperature":     0,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AI API returned %d: %.200s", resp.StatusCode, raw)
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return err
	}
	if len(parsed.Choices) == 0 {
		return fmt.Errorf("AI API returned no choices")
	}
	return json.Unmarshal([]byte(parsed.Choices[0].Message.Content), out)
}
