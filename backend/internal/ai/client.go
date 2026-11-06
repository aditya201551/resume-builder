package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const OpenRouterBaseURL = "https://openrouter.ai/api/v1"

func NewChatModel(ctx context.Context, cfg *Config, maxTokens int) (model.ToolCallingChatModel, error) {
	return newChatModel(ctx, cfg, maxTokens, &http.Client{Timeout: 2 * time.Minute})
}

func newChatModel(ctx context.Context, cfg *Config, maxTokens int, client *http.Client) (model.ToolCallingChatModel, error) {
	switch cfg.Provider {
	case ProviderAnthropic:
		chatModel, err := claude.NewChatModel(ctx, &claude.Config{
			APIKey:     cfg.APIKey,
			Model:      cfg.Model,
			MaxTokens:  maxTokens,
			HTTPClient: client,
		})
		if err != nil {
			return nil, fmt.Errorf("create claude chat model: %w", err)
		}
		return &checkedModel{chatModel}, nil

	case ProviderOpenRouter, ProviderOpenAI, ProviderVercel:
		extra := map[string]any{"store": false}
		if cfg.Provider == ProviderOpenRouter {
			extra["provider"] = map[string]any{"require_parameters": true}
		}
		options := &openai.ChatModelConfig{
			APIKey: cfg.APIKey, BaseURL: BaseURL(cfg.Provider), Model: cfg.Model,
			MaxCompletionTokens: &maxTokens, ExtraFields: extra,
			HTTPClient: client,
		}
		if cfg.Provider == ProviderOpenAI && strings.HasPrefix(cfg.Model, "gpt-5.") {
			options.ReasoningEffort = "none"
		}
		chatModel, err := openai.NewChatModel(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("create compatible chat model: %w", err)
		}
		return &checkedModel{chatModel}, nil

	default:
		return nil, fmt.Errorf("unsupported AI provider %q", cfg.Provider)
	}
}

// checkedModel handles providers that encode a failed SSE request inside HTTP
// 200. Returning a sanitized error prevents a clean EOF being treated as success.
type checkedModel struct{ model.ToolCallingChatModel }

func (m *checkedModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	bound, err := m.ToolCallingChatModel.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &checkedModel{bound}, nil
}
func checkMessage(msg *schema.Message) (*schema.Message, error) {
	if msg != nil && msg.ResponseMeta != nil && msg.ResponseMeta.FinishReason == "error" {
		return nil, errors.New("AI provider could not complete the request")
	}
	return msg, nil
}
func (m *checkedModel) Generate(ctx context.Context, messages []*schema.Message, options ...model.Option) (*schema.Message, error) {
	msg, err := m.ToolCallingChatModel.Generate(ctx, messages, options...)
	if err != nil {
		return nil, err
	}
	return checkMessage(msg)
}
func (m *checkedModel) Stream(ctx context.Context, messages []*schema.Message, options ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	stream, err := m.ToolCallingChatModel.Stream(ctx, messages, options...)
	if err != nil {
		return nil, err
	}
	return schema.StreamReaderWithConvert(stream, checkMessage), nil
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
