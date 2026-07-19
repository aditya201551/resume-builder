package handlers

import (
	"net/http"

	"resume-builder/backend/internal/agent"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/service"
)

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
