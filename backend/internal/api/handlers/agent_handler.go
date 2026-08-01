package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"resume-builder/backend/internal/agent"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/service"
)

// chatHistoryLimit caps how many prior messages are forwarded to the model
// per request. History is entirely client-owned (see chat.go's package
// comment on the agent package) — the backend keeps no session, so this
// only bounds token usage per call; older turns are simply dropped and the
// client still has them locally.
const chatHistoryLimit = 40

// AgentHandler exposes the Phase 2 AI assistant over HTTP. It is nil-safe
// at the router level: NewRouter only registers these routes when an Agent
// was actually constructed (i.e. ANTHROPIC_API_KEY is set), so the API
// keeps working without the AI assistant configured.
type AgentHandler struct {
	agent   *agent.Agent
	resumes *service.ResumeService
}

func NewAgentHandler(a *agent.Agent, resumes *service.ResumeService) *AgentHandler {
	return &AgentHandler{agent: a, resumes: resumes}
}

// SuggestContentRewrite returns an AI-suggested rewrite for a resume content
// block. It never writes to storage — the client PATCHes
// /resumes/{resumeID}/content-blocks/{kind}/{id} separately once the user
// accepts the suggestion.
func (h *AgentHandler) SuggestContentRewrite(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	resumeID := r.PathValue("resumeID")

	var body struct {
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	// Ownership check up front so a user can't probe another user's resume
	// through this endpoint before the agent ever runs.
	if _, err := h.resumes.Get(r.Context(), userID, resumeID); err != nil {
		writeError(w, err)
		return
	}

	suggestion, err := h.agent.SuggestContentRewrite(r.Context(), resumeID, body.Content)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"suggestion": suggestion})
}

type chatMessageDTO struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequestBody struct {
	Messages []chatMessageDTO `json:"messages"`
	Draft    json.RawMessage  `json:"draft"`
}

// Chat streams a multi-turn, multi-tool-call conversation over SSE. The
// agent never writes to storage — it emits "proposal" events shaped like
// the frontend's DraftAction union (see internal/agent/proposal.go), which
// the client applies to its local draft only once the user accepts them.
// History and the current resume draft are both supplied by the client on
// every call; this handler and the agent underneath it are stateless.
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

	// Ownership check up front, before any SSE headers go out or any model
	// call is made — same pattern as SuggestContentRewrite above.
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

	iter, sink, err := h.agent.Chat(r.Context(), messages, string(body.Draft))
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
			streamErr := agent.StreamAssistantText(event.Output.MessageOutput, func(text string) {
				send("token", map[string]string{"text": text})
			})
			if streamErr != nil {
				send("error", map[string]string{"message": streamErr.Error()})
				return
			}
		}
	}

	// Tools can fire on the same iterator step as the final event; drain
	// once more so nothing decided right at the end is dropped.
	for _, p := range sink.Drain() {
		send("proposal", map[string]any{"action": p})
	}
	send("done", map[string]any{})
}
