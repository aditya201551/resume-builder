package agent

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// chatInstruction is the system prompt for the resume_chat agent — the
// multi-turn, multi-tool-call counterpart to the single-shot rewrite agent
// in agent.go. It never writes to the resume directly; every change goes
// through a propose_* tool and is applied by the user, client-side, only
// after they accept it (see proposal.go and the propose_* tools in tools.go).
const chatInstruction = `You are a resume-writing assistant with tools to read the ` +
	`user's current resume draft and propose changes to it. You can call ` +
	`read_resume to see the full current state before proposing anything, and ` +
	`propose_create/propose_update/propose_delete/propose_skill_change/` +
	`propose_custom_section_change/propose_meta_update to suggest edits.

You never edit anything directly — every propose_* call only stages a change ` +
	`for the user to accept or reject client-side. Nothing you propose is final. ` +
	`Say so if it's not obvious from context.

When the user describes something in natural language (e.g. dictating their work ` +
	`history), turn it into one or more propose_create calls with well-structured ` +
	`fields rather than asking them to fill out a form. Write achievement-focused, ` +
	`ATS-friendly content. Never set sort_order — the client places new entries.

Dates should be plain strings like "2023-01" or "2023-01-15", matching however ` +
	`the existing resume data represents them.`

// ChatMessage is the wire-level shape of one turn in the conversation.
type ChatMessage struct {
	Role    string // "user" | "assistant"
	Content string
}

// Chat starts one turn of the multi-tool-call resume assistant. draftJSON is
// the client's current local draft (see doc.go / package comment on why the
// agent reads this instead of the database — the local-first frontend has no
// autosave, so the DB can be stale relative to what the user is looking at).
//
// The returned ProposalSink accumulates propose_* tool calls made during this
// run; the caller (agent_handler.go's SSE handler) should Drain() it after
// each iterator step to stream proposals to the client as they're decided,
// not just once at the end.
func (a *Agent) Chat(ctx context.Context, history []ChatMessage, draftJSON string) (*adk.AsyncIterator[*adk.AgentEvent], *ProposalSink, error) {
	if a.chatRunner == nil {
		return nil, nil, errors.New("chat agent not configured")
	}

	msgs := make([]*schema.Message, 0, len(history))
	for _, m := range history {
		switch m.Role {
		case "assistant":
			msgs = append(msgs, schema.AssistantMessage(m.Content, nil))
		default:
			msgs = append(msgs, schema.UserMessage(m.Content))
		}
	}
	if len(msgs) == 0 {
		return nil, nil, errors.New("chat requires at least one message")
	}

	sink := &ProposalSink{}
	iter := a.chatRunner.Run(ctx, msgs, adk.WithSessionValues(map[string]any{
		sessionKeyDraft: draftJSON,
		sessionKeySink:  sink,
	}))
	return iter, sink, nil
}

// newChatAgent builds the resume_chat ChatModelAgent sharing chatModel with
// the rewrite agent in agent.go's New().
func newChatAgent(ctx context.Context, chatModel model.BaseModel[*schema.Message]) (*adk.ChatModelAgent, error) {
	tools := []tool.BaseTool{
		newReadResumeTool(),
		newProposeCreateTool(),
		newProposeUpdateTool(),
		newProposeDeleteTool(),
		newProposeSkillChangeTool(),
		newProposeCustomSectionChangeTool(),
		newProposeMetaUpdateTool(),
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "resume_chat",
		Description: "Multi-turn assistant that can read the resume draft and propose structured edits to it.",
		Instruction: chatInstruction,
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools},
		},
	})
}

// StreamAssistantText drains an assistant MessageVariant and calls emit for
// each piece of text as it arrives — the streaming counterpart to agent.go's
// assistantText, which blocks until the whole message is available. Only
// assistant-authored text is surfaced; tool-call/tool-result variants are
// silently skipped, matching assistantText's behavior. Exported so
// agent_handler.go's SSE loop can call it per event.
func StreamAssistantText(mv *adk.MessageVariant, emit func(string)) error {
	if mv == nil || mv.Role != schema.Assistant {
		return nil
	}

	if !mv.IsStreaming {
		if mv.Message != nil && mv.Message.Content != "" {
			emit(mv.Message.Content)
		}
		return nil
	}

	for {
		chunk, err := mv.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read message stream: %w", err)
		}
		if chunk.Content != "" {
			emit(chunk.Content)
		}
	}
}
