package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"learncode/internal/config"
)

type Service struct {
	providers map[string]Provider
	default_  string
	mu        sync.Mutex
	total     TokenUsage
}

func NewService(cfg config.LLMConfig) (*Service, error) {
	providers := make(map[string]Provider)
	for _, p := range cfg.Providers {
		if p.Endpoint == "" {
			continue
		}
		providers[p.Name] = NewProvider(p.Name, p.Endpoint, p.APIKey, p.Models)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no llm providers configured")
	}
	if _, ok := providers[cfg.Default]; !ok {
		for k := range providers {
			cfg.Default = k
			break
		}
	}
	return &Service{
		providers: providers,
		default_:  cfg.Default,
	}, nil
}

func (s *Service) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, *TokenUsage, error) {
	return s.ChatWithTemp(ctx, systemPrompt, userPrompt, 0.3, 4096)
}

func (s *Service) ChatWithTemp(ctx context.Context, systemPrompt, userPrompt string, temperature float64, maxTokens int) (string, *TokenUsage, error) {
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	req := ChatRequest{
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	p, ok := s.providers[s.default_]
	if !ok {
		return "", nil, fmt.Errorf("default provider %q not found", s.default_)
	}

	slog.Info("ChatWithTemp: calling default provider",
		"provider", s.default_, "model", p.Name(), "temperature", temperature, "max_tokens", maxTokens)

	resp, err := p.Chat(ctx, req)
	if err != nil {
		slog.Warn("ChatWithTemp: default provider failed, starting fallback",
			"provider", s.default_, "error", err)

		fallbackCtx := ctx
		if ctx.Err() != nil {
			fallbackCtx = context.Background()
		}
		for name, alt := range s.providers {
			if name == s.default_ {
				continue
			}
			slog.Info("ChatWithTemp: trying fallback provider", "provider", name, "model", alt.Name())
			resp, err = alt.Chat(fallbackCtx, req)
			if err == nil {
				slog.Info("ChatWithTemp: fallback provider succeeded", "provider", name)
				break
			}
			slog.Warn("ChatWithTemp: fallback provider also failed", "provider", name, "error", err)
		}
		if err != nil {
			slog.Error("ChatWithTemp: all providers failed", "error", err)
			return "", nil, fmt.Errorf("all providers failed: %w", err)
		}
	}

	slog.Info("ChatWithTemp: success",
		"provider", s.default_,
		"prompt_tokens", resp.Usage.PromptTokens,
		"completion_tokens", resp.Usage.CompletionTokens,
		"content_length", len(resp.Content))

	s.mu.Lock()
	s.total.PromptTokens += resp.Usage.PromptTokens
	s.total.CompletionTokens += resp.Usage.CompletionTokens
	s.total.TotalTokens += resp.Usage.TotalTokens
	s.mu.Unlock()

	return resp.Content, &TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}, nil
}

func (s *Service) Reload(cfg config.LLMConfig) error {
	providers := make(map[string]Provider)
	for _, p := range cfg.Providers {
		if p.Endpoint == "" {
			continue
		}
		providers[p.Name] = NewProvider(p.Name, p.Endpoint, p.APIKey, p.Models)
	}
	if len(providers) == 0 {
		return fmt.Errorf("no llm providers configured")
	}
	if _, ok := providers[cfg.Default]; !ok {
		for k := range providers {
			cfg.Default = k
			break
		}
	}
	s.mu.Lock()
	s.providers = providers
	s.default_ = cfg.Default
	s.mu.Unlock()
	return nil
}

func (s *Service) TotalUsage() TokenUsage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.total
}
