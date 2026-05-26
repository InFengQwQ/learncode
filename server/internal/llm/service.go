package llm

import (
	"context"
	"fmt"
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
		// use first available as default
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

// ChatWithTemp sends a chat request with explicit temperature and max_tokens.
// Used when callers need to vary temperature (e.g., retry loops).
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

	resp, err := p.Chat(ctx, req)
	if err != nil {
		// try other providers
		for name, alt := range s.providers {
			if name == s.default_ {
				continue
			}
			resp, err = alt.Chat(ctx, req)
			if err == nil {
				break
			}
		}
		if err != nil {
			return "", nil, fmt.Errorf("all providers failed: %w", err)
		}
	}

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

// Reload recreates providers from new config without restarting the server.
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
