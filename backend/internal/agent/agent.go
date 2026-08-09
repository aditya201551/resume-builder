package agent

import (
	"context"
	"fmt"

	"resume-builder/backend/internal/ai"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

type Agent struct {
	chatRunner   *adk.Runner
	modelOptions []model.Option
}

func New(ctx context.Context, cfg *Config) (*Agent, error) {
	chatModel, err := ai.NewChatModel(ctx, cfg.AI, cfg.MaxTokens)
	if err != nil {
		return nil, err
	}

	chatAgent, err := newChatAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("create chat agent: %w", err)
	}

	chatRunner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           chatAgent,
		EnableStreaming: true,
	})

	return &Agent{chatRunner: chatRunner, modelOptions: cfg.AI.ChatModelOptions()}, nil
}
