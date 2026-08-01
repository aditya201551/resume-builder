package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"resume-builder/backend/internal/service"
)

// rewriteInstruction is the system prompt for the resume_assistant agent.
// It currently only covers content-block rewriting; grammar-pass and
// professional-summary generation (also scoped for Phase 2 in the PRD) are
// separate instructions/agents to add later rather than growing this one
// prompt to cover unrelated tasks.
const rewriteInstruction = `You rewrite resume bullet/paragraph content to be concise, ` +
	`achievement-focused, and ATS-friendly. Use get_full_resume for context about the ` +
	`candidate's role, seniority, and skills when it would improve the rewrite. ` +
	`Return only the rewritten Markdown content — no preamble, no commentary.`

// Agent is the Phase 2 AI assistant: Eino ChatModelAgents wired directly to
// the existing service layer (see doc.go), sharing one Claude chat model
// across two runners with different jobs:
//   - runner: single-shot content-block rewrite (SuggestContentRewrite)
//   - chatRunner: multi-turn, multi-tool-call conversation that proposes
//     structured edits across the resume (Chat, see chat.go)
type Agent struct {
	runner     *adk.Runner
	chatRunner *adk.Runner
}

// New builds the agent's Claude chat model, tool sets, and Eino runners.
// resumes is the same *service.ResumeService instance cmd/api's HTTP
// handlers use — the agent shares the service layer rather than
// duplicating it, but only for the read-only rewrite-agent tool; the chat
// agent's tools operate on the client-supplied draft, never the database
// (see chat.go's package comment for why).
func New(ctx context.Context, cfg *Config, resumes *service.ResumeService) (*Agent, error) {
	chatModel, err := claude.NewChatModel(ctx, &claude.Config{
		APIKey:    cfg.AnthropicAPIKey,
		Model:     cfg.ClaudeModel,
		MaxTokens: cfg.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("create claude chat model: %w", err)
	}

	tools := []tool.BaseTool{
		newGetFullResumeTool(resumes),
	}

	rewriteAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "resume_assistant",
		Description: "Suggests improved phrasing for resume content blocks.",
		Instruction: rewriteInstruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create chat model agent: %w", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: rewriteAgent,
		// Non-streaming — SuggestContentRewrite returns one complete
		// suggestion, drained into a single string by assistantText below.
		EnableStreaming: false,
	})

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

	return &Agent{runner: runner, chatRunner: chatRunner}, nil
}

// SuggestContentRewrite asks the agent to rewrite one work-experience/project
// content block. It does not persist anything — the caller (the
// content-blocks PATCH endpoint) writes the change only once the user
// accepts the suggestion, per the PRD's "original preserved until accepted"
// requirement. ctx must carry an authenticated user (see auth.UserIDFromContext)
// since the get_full_resume tool enforces resume ownership.
func (a *Agent) SuggestContentRewrite(ctx context.Context, resumeID, currentContent string) (string, error) {
	prompt := fmt.Sprintf(
		"Resume ID: %s\n\nCurrent content block (Markdown):\n%s\n\nRewrite this block.",
		resumeID, currentContent,
	)

	iter := a.runner.Query(ctx, prompt)

	var out strings.Builder
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}

		text, err := assistantText(event)
		if err != nil {
			return "", err
		}
		out.WriteString(text)
	}

	if out.Len() == 0 {
		return "", errors.New("agent produced no output")
	}
	return out.String(), nil
}

// assistantText extracts assistant-authored text from a single agent event,
// ignoring tool-call/tool-result events. EnableStreaming is false in New, so
// the streaming branch below is defensive rather than the expected path.
func assistantText(event *adk.AgentEvent) (string, error) {
	if event.Output == nil || event.Output.MessageOutput == nil {
		return "", nil
	}

	mv := event.Output.MessageOutput
	if mv.Role != schema.Assistant {
		return "", nil
	}

	if !mv.IsStreaming {
		return mv.Message.Content, nil
	}

	var out strings.Builder
	for {
		chunk, err := mv.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		out.WriteString(chunk.Content)
	}
	return out.String(), nil
}
