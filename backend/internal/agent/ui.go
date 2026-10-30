package agent

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// Generated from the frontend component catalog. See npm run generate:openui.
//
//go:embed openui_prompt.txt
var openUIPrompt string

type UIRewriteTarget struct {
	Entity   string `json:"entity"`
	ID       string `json:"id"`
	Field    string `json:"field"`
	Original string `json:"original"`
}

type AssistantUI struct {
	ID       string           `json:"id"`
	Version  int              `json:"version"`
	Kind     string           `json:"kind"`
	Code     string           `json:"code"`
	Fallback string           `json:"fallback"`
	Target   *UIRewriteTarget `json:"target,omitempty"`
}

type showResumeUITool struct{}

func newShowResumeUITool() tool.InvokableTool { return &showResumeUITool{} }
func (t *showResumeUITool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "show_resume_ui",
		Desc: "Show an interactive resume card only for an explicit request for guided input or alternatives, or blocking missing facts or an ambiguity that benefits from structured input. When the request is clear and required facts are available, use normal mutation tools instead; complete new entries are added directly. Missing optional details never require a form. Use experience_form for a NEW work experience only when a form is warranted; prefill every supplied or unambiguously implied fact, including dates and current status, leaving only unknowns empty. Never invent a title, duties or metrics. Use rewrite_comparison for requested alternatives or an unresolved choice between materially different directions, not an ordinary rewrite. First call read_resume for any existing target. The app supplies the original text and validates the user's choice. Do NOT also propose the same change: the card creates an ordinary reviewable proposal after submission. Keep regular assistant replies as prose; put OpenUI Lang ONLY in this tool's code argument. Use at most two cards per turn.\n\n" + openUIPrompt,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"kind":     {Type: schema.String, Enum: []string{"experience_form", "rewrite_comparison"}, Required: true},
			"code":     {Type: schema.String, Desc: "OpenUI Lang program using the component signatures above. One AssistantPanel root, exactly one interactive card. No Markdown fences or external tools.", Required: true},
			"fallback": {Type: schema.String, Desc: "Brief plain-text explanation of what this card helps the user do, without invented facts.", Required: true},
			"entity":   {Type: schema.String, Enum: []string{"resume", "work_experiences", "projects"}, Desc: "Required for rewrites; omit for new experience forms."},
			"entry_id": {Type: schema.String, Desc: "Exact existing entry id from read_resume; required for rewrites. For resume, use the current resume's id."},
			"field":    {Type: schema.String, Enum: []string{"summary", "headline", "content"}, Desc: "For rewrites: summary/headline for resume, content for work_experiences/projects."},
		}),
	}, nil
}

type showUIArgs struct {
	Kind     string `json:"kind"`
	Code     string `json:"code"`
	Fallback string `json:"fallback"`
	Entity   string `json:"entity"`
	EntryID  string `json:"entry_id"`
	Field    string `json:"field"`
}

func prepareResumeUI(args showUIArgs, draftJSON string) (AssistantUI, error) {
	ui := AssistantUI{Version: 1, Kind: args.Kind, Code: args.Code, Fallback: args.Fallback}
	if strings.TrimSpace(args.Code) == "" || len(args.Code) > 16000 || strings.Contains(args.Code, "```") {
		return ui, fmt.Errorf("code must be a raw OpenUI Lang program of up to 16000 bytes")
	}
	if strings.TrimSpace(args.Fallback) == "" || len(args.Fallback) > 1000 {
		return ui, fmt.Errorf("fallback must contain 1 to 1000 bytes of plain text")
	}
	if args.Kind == "experience_form" {
		return ui, nil
	}
	if args.Kind != "rewrite_comparison" {
		return ui, fmt.Errorf("unknown card kind")
	}
	var draft struct {
		Resume struct {
			ID       string `json:"id"`
			Summary  string `json:"summary"`
			Headline string `json:"headline"`
		} `json:"resume"`
		Work []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"work_experiences"`
		Projects []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"projects"`
	}
	if err := json.Unmarshal([]byte(draftJSON), &draft); err != nil {
		return ui, fmt.Errorf("could not read current resume")
	}
	if args.EntryID == "" {
		return ui, fmt.Errorf("entry_id is required for rewrites")
	}
	target := &UIRewriteTarget{Entity: args.Entity, ID: args.EntryID, Field: args.Field}
	found := false
	if args.Entity == "resume" && args.EntryID == draft.Resume.ID {
		switch args.Field {
		case "summary":
			target.Original = draft.Resume.Summary
			found = true
		case "headline":
			target.Original = draft.Resume.Headline
			found = true
		}
	} else if args.Field == "content" {
		entries := draft.Work
		if args.Entity == "projects" {
			entries = draft.Projects
		} else if args.Entity != "work_experiences" {
			return ui, fmt.Errorf("unsupported rewrite entity")
		}
		for _, entry := range entries {
			if entry.ID == args.EntryID {
				target.Original = entry.Content
				found = true
				break
			}
		}
	}
	if !found {
		return ui, fmt.Errorf("rewrite target not found or unsupported; use exact IDs and fields from read_resume")
	}
	if len(target.Original) > 16000 {
		return ui, fmt.Errorf("this text is too long for an interactive comparison; use ordinary small rewrite proposals")
	}
	ui.Target = target
	return ui, nil
}

func (t *showResumeUITool) InvokableRun(ctx context.Context, arguments string, _ ...tool.Option) (string, error) {
	var args showUIArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return toolProblem("could not parse the interactive card")
	}
	draft, ok := adk.GetSessionValue(ctx, sessionKeyDraft)
	draftJSON, valid := draft.(string)
	if !ok || !valid {
		return "", fmt.Errorf("no resume draft in context")
	}
	ui, err := prepareResumeUI(args, draftJSON)
	if err != nil {
		return toolProblem("%v", err)
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}
	ui.ID = compose.GetToolCallID(ctx)
	if ui.ID == "" {
		ui.ID = tempID()
	}
	if !sink.addUI(ui) {
		return toolProblem("at most two interactive cards may be shown in one turn")
	}
	return "Interactive card shown. No resume content has changed. Wait for the user's selection or form submission; do not also propose this change.", nil
}
