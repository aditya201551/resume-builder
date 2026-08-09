package ai

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

const OpenRouterBaseURL = "https://openrouter.ai/api/v1"

func NewChatModel(ctx context.Context, cfg *Config, maxTokens int) (model.ToolCallingChatModel, error) {
	switch cfg.Provider {
	case ProviderAnthropic:
		chatModel, err := claude.NewChatModel(ctx, &claude.Config{
			APIKey:    cfg.APIKey,
			Model:     cfg.Model,
			MaxTokens: maxTokens,
		})
		if err != nil {
			return nil, fmt.Errorf("create claude chat model: %w", err)
		}
		return chatModel, nil

	case ProviderOpenRouter:
		chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:    cfg.APIKey,
			BaseURL:   OpenRouterBaseURL,
			Model:     cfg.Model,
			MaxTokens: &maxTokens,
		})
		if err != nil {
			return nil, fmt.Errorf("create openrouter chat model: %w", err)
		}
		return chatModel, nil

	default:
		return nil, fmt.Errorf("unsupported AI provider %q", cfg.Provider)
	}
}

func (c *Config) ChatModelOptions() []model.Option {
	switch c.Provider {
	case ProviderAnthropic:
		return []model.Option{
			claude.WithAutoCacheControl(&claude.CacheControl{TTL: claude.CacheTTL1h}),
		}
	default:
		return nil
	}
}
