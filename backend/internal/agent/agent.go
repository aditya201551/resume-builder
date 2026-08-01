package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino/adk"
)

// Agent wraps the resume_chat Eino ChatModelAgent — the multi-turn,
// multi-tool-call conversation that proposes structured edits across the
// resume (Chat, see chat.go).
type Agent struct {
	chatRunner *adk.Runner
}

// New builds the agent's Claude chat model and the chat agent's runner.
func New(ctx context.Context, cfg *Config) (*Agent, error) {
	chatModel, err := claude.NewChatModel(ctx, &claude.Config{
		APIKey:    cfg.AnthropicAPIKey,
		Model:     cfg.ClaudeModel,
		MaxTokens: cfg.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("create claude chat model: %w", err)
	}

	chatAgent, err := newChatAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("create chat agent: %w", err)
	}

	chatRunner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: chatAgent,
		// Streaming — Chat's caller (agent_handler.go) forwards text as it
		// arrives over SSE rather than waiting for a complete message.
		EnableStreaming: true,
	})

	return &Agent{chatRunner: chatRunner}, nil
}
