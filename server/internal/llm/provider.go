package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type openAIProvider struct {
	name     string
	endpoint string
	apiKey   string
	model    string
	client   *http.Client
}

func NewProvider(name, endpoint, apiKey string, models []string) Provider {
	model := ""
	if len(models) > 0 {
		model = models[0]
	}
	return &openAIProvider{
		name:     name,
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		client: &http.Client{
			Timeout: 0,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{Timeout: 20 * time.Second}).DialContext,
			},
		},
	}
}

func (p *openAIProvider) Name() string { return p.name }

func (p *openAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.model
	}
	slog.Info("llm call start", "provider", p.name, "model", req.Model)

	type openAIRequest struct {
		Model       string        `json:"model"`
		Messages    []ChatMessage `json:"messages"`
		Temperature float64       `json:"temperature,omitempty"`
		MaxTokens   int           `json:"max_tokens,omitempty"`
	}
	body := openAIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.endpoint+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		slog.Warn("llm call failed", "provider", p.name, "model", req.Model, "error", err)
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		slog.Warn("llm call api error", "provider", p.name, "model", req.Model, "status", resp.StatusCode)
		return nil, fmt.Errorf("llm api error %d: %s", resp.StatusCode, errBody.String())
	}

	type openAIChoice struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
	}
	type openAIUsage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}
	type openAIResponse struct {
		Choices []openAIChoice `json:"choices"`
		Usage   openAIUsage    `json:"usage"`
	}

	var parsed openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := parsed.Choices[0].Message.Content
	if content == "" && parsed.Choices[0].Message.ReasoningContent != "" {
		reasoning := parsed.Choices[0].Message.ReasoningContent
		if json := extractJSONFromText(reasoning); json != "" {
			content = json
		} else {
			content = reasoning
		}
	}
	if content == "" {
		return nil, fmt.Errorf("empty response from llm")
	}

	return &ChatResponse{
		Content: content,
		Usage: TokenUsage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		},
	}, nil
}

func extractJSONFromText(text string) string {
	for i := len(text) - 1; i >= 0; i-- {
		if text[i] == '}' {
			depth := 0
			for j := i; j >= 0; j-- {
				if text[j] == '}' {
					depth++
				} else if text[j] == '{' {
					depth--
					if depth == 0 {
						candidate := text[j : i+1]
						var v interface{}
						if json.Unmarshal([]byte(candidate), &v) == nil {
							return candidate
						}
						break
					}
				}
			}
		}
	}
	return ""
}
