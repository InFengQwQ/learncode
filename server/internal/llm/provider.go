package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
			Timeout: 0, // no overall timeout — model generation can take minutes
			Transport: &http.Transport{
				DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
			},
		},
	}
}

func (p *openAIProvider) Name() string { return p.name }

func (p *openAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.model
	}

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
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
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
	// Reasoning models (e.g., qwen3.5) may produce reasoning_content
	// but leave content empty when max_tokens is too low.
	// Fall back to reasoning_content as a best-effort recovery.
	if content == "" {
		content = parsed.Choices[0].Message.ReasoningContent
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
