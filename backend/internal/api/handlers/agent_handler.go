package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"resume-builder/backend/internal/agent"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/service"

	"github.com/cloudwego/eino/schema"
)

const chatHistoryLimit = 40

type AgentHandler struct {
	agent     *agent.Agent
	resumes   *service.ResumeService
	templates *service.TemplateService
}

func NewAgentHandler(a *agent.Agent, resumes *service.ResumeService, templates *service.TemplateService) *AgentHandler {
	return &AgentHandler{agent: a, resumes: resumes, templates: templates}
}

type chatMessageDTO struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequestBody struct {
	Messages []chatMessageDTO `json:"messages"`
	Draft    json.RawMessage  `json:"draft"`
}

type toolEventDTO struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Arguments string `json:"arguments,omitempty"`
	Error     string `json:"error,omitempty"`
}

func toolEventFromCall(tc schema.ToolCall, status string) toolEventDTO {
	return toolEventDTO{
		ID:        tc.ID,
		Name:      tc.Function.Name,
		Status:    status,
		Arguments: tc.Function.Arguments,
	}
}

func (h *AgentHandler) Chat(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	resumeID := r.PathValue("resumeID")

	var body chatRequestBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if len(body.Messages) == 0 {
		http.Error(w, "messages must not be empty", http.StatusBadRequest)
		return
	}

	if _, err := h.resumes.Get(r.Context(), userID, resumeID); err != nil {
		writeError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	history := body.Messages
	if len(history) > chatHistoryLimit {
		history = history[len(history)-chatHistoryLimit:]
	}
	messages := make([]agent.ChatMessage, len(history))
	for i, m := range history {
		messages[i] = agent.ChatMessage{Role: m.Role, Content: m.Content}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	send := func(event string, v any) {
		data, err := json.Marshal(v)
		if err != nil {
			data = []byte(`{}`)
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	fetchTemplates := func(ctx context.Context) (string, error) {
		list, err := h.templates.List(ctx)
		if err != nil {
			return "", err
		}
		data, err := json.Marshal(list)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	iter, sink, err := h.agent.Chat(r.Context(), messages, string(body.Draft), fetchTemplates)
	if err != nil {
		send("error", map[string]string{"message": err.Error()})
		return
	}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			send("error", map[string]string{"message": event.Err.Error()})
			return
		}

		for _, p := range sink.Drain() {
			send("proposal", map[string]any{"action": p})
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			mv := event.Output.MessageOutput

			if mv.Role == schema.Tool {
				name := mv.ToolName
				id := ""
				if mv.Message != nil && mv.Message.ToolName != "" {
					name = mv.Message.ToolName
				}
				if mv.Message != nil {
					id = mv.Message.ToolCallID
				}
				if name != "" {
					send("tool", toolEventDTO{ID: id, Name: name, Status: "done"})
				}
				continue
			}

			streamErr := agent.StreamAssistantOutput(mv, func(text string) {
				send("token", map[string]string{"text": text})
			}, func(tc schema.ToolCall) {
				send("tool", toolEventFromCall(tc, "running"))
			})
			if streamErr != nil {
				send("error", map[string]string{"message": streamErr.Error()})
				return
			}
		}
	}

	for _, p := range sink.Drain() {
		send("proposal", map[string]any{"action": p})
	}
	send("done", map[string]any{})
}
