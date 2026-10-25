package provider

import (
	"context"
	"fmt"

	"mobile-agent/internal/config"
	"mobile-agent/internal/types"
)

type Provider interface {
	Name() string
	Complete(ctx context.Context, req types.CompletionRequest) (*types.CompletionResponse, error)
}

func New(cfg *config.Config, name, model string) (Provider, error) {
	p, ok := cfg.Providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}
	if model != "" {
		p.Model = model
	}
	return NewOpenAI(OpenAIConfig{BaseURL: p.BaseURL, APIKey: p.APIKey, Model: p.Model}, p.Model)
}
