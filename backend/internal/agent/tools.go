package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/service"
)

// Session value keys set by Agent.Chat (chat.go) via adk.WithSessionValues
// and read by the propose_*/read_resume tools below via
// adk.GetSessionValue. Keeping these unexported keeps the mechanism private
// to this package — callers only see Chat's exported (history, draftJSON)
// signature.
const (
	sessionKeyDraft = "chat_draft_json"
	sessionKeySink  = "chat_proposal_sink"
)

func sinkFromContext(ctx context.Context) (*ProposalSink, error) {
	v, ok := adk.GetSessionValue(ctx, sessionKeySink)
	if !ok {
		return nil, errors.New("no proposal sink in context (tool called outside Agent.Chat)")
	}
	sink, ok := v.(*ProposalSink)
	if !ok {
		return nil, errors.New("proposal sink has unexpected type")
	}
	return sink, nil
}

// getFullResumeTool lets the agent read a resume's full content before
// drafting a rewrite or summary. It wraps ResumeService.GetFullResume
// directly — no MCP, no network hop, since the agent runs in the same
// process as the rest of the API (see doc.go). If this package is ever
// split into its own service, this is the tool that would move behind an
// MCP server or HTTP call instead of a direct method call.
type getFullResumeTool struct {
	resumes *service.ResumeService
}

func newGetFullResumeTool(resumes *service.ResumeService) tool.InvokableTool {
	return &getFullResumeTool{resumes: resumes}
}

func (t *getFullResumeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_full_resume",
		Desc: "Fetch every section of a resume (work experience, education, skills, projects, certifications, languages, misc entries, custom sections) by resume ID.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"resume_id": {
				Type:     schema.String,
				Desc:     "UUID of the resume to read",
				Required: true,
			},
		}),
	}, nil
}

type getFullResumeArgs struct {
	ResumeID string `json:"resume_id"`
}

func (t *getFullResumeTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args getFullResumeArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse get_full_resume arguments: %w", err)
	}

	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return "", fmt.Errorf("no authenticated user in context")
	}

	full, err := t.resumes.GetFullResume(ctx, userID, args.ResumeID)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(full)
	if err != nil {
		return "", fmt.Errorf("marshal full resume: %w", err)
	}
	return string(out), nil
}

// --- resume_chat tools -----------------------------------------------------
//
// Everything below reads/writes via session values (see sinkFromContext and
// sessionKeyDraft above) instead of resume_id arguments or the database —
// the chat agent operates on the client's local draft, which may be ahead of
// what's persisted (see chat.go's package comment), and letting the model
// name an arbitrary resume_id would let it target a resume the user doesn't
// currently have open. propose_* tools never touch storage; they only
// append a Proposal to the run's sink for the HTTP handler to stream out.

func stripSortOrder(m map[string]any) map[string]any {
	if m == nil {
		return m
	}
	delete(m, "sort_order")
	delete(m, "sortOrder")
	return m
}

// readResumeTool returns the client-supplied draft snapshot for this run
// verbatim, so the model can inspect current state before proposing changes.
type readResumeTool struct{}

func newReadResumeTool() tool.InvokableTool { return &readResumeTool{} }

func (t *readResumeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_resume",
		Desc: "Read the user's current resume draft (work experience, education, skills, projects, certifications, languages, misc entries, custom sections, contact info). Takes no arguments.",
		// Anthropic requires every tool to have an input_schema, even a
		// no-argument one — an empty params map produces {"type":"object",
		// "properties":{}} rather than omitting the field entirely.
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *readResumeTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	v, ok := adk.GetSessionValue(ctx, sessionKeyDraft)
	if !ok {
		return "", errors.New("no resume draft in context (tool called outside Agent.Chat)")
	}
	draftJSON, ok := v.(string)
	if !ok {
		return "", errors.New("resume draft has unexpected type")
	}
	return draftJSON, nil
}

var flatEntityInfo = &schema.ParameterInfo{
	Type:     schema.String,
	Desc:     "Which resume section this entity belongs to.",
	Enum:     []string{"work_experiences", "educations", "projects", "certifications", "languages", "misc_entries"},
	Required: true,
}

func validateFlatEntity(entity string) error {
	if !flatEntities[entity] {
		return fmt.Errorf("unknown entity %q: must be one of work_experiences, educations, projects, certifications, languages, misc_entries", entity)
	}
	return nil
}

// --- propose_create ---------------------------------------------------------

type proposeCreateTool struct{}

func newProposeCreateTool() tool.InvokableTool { return &proposeCreateTool{} }

func (t *proposeCreateTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_create",
		Desc: "Propose adding a new entry to a resume section. This does not save anything — it stages a change for the user to accept. Never include sort_order.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"entity": flatEntityInfo,
			"fields": {
				Type:     schema.Object,
				Desc:     "The new entry's fields, matching that entity's shape (e.g. for work_experiences: company, title, location, start_date, end_date, is_current, content).",
				Required: true,
			},
		}),
	}, nil
}

type proposeCreateArgs struct {
	Entity string         `json:"entity"`
	Fields map[string]any `json:"fields"`
}

func (t *proposeCreateTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args proposeCreateArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse propose_create arguments: %w", err)
	}
	if err := validateFlatEntity(args.Entity); err != nil {
		return "", err
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	id := tempID()
	sink.add(Proposal{Type: "flat_create", Entity: args.Entity, TempID: id, Fields: stripSortOrder(args.Fields)})
	return fmt.Sprintf("proposed: create %s (tempId %s)", args.Entity, id), nil
}

// --- propose_update ----------------------------------------------------------

type proposeUpdateTool struct{}

func newProposeUpdateTool() tool.InvokableTool { return &proposeUpdateTool{} }

func (t *proposeUpdateTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_update",
		Desc: "Propose changing fields on an existing resume entry. This does not save anything — it stages a change for the user to accept. Never include sort_order.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"entity": flatEntityInfo,
			"id": {
				Type:     schema.String,
				Desc:     "ID of the entry to update, as seen via read_resume.",
				Required: true,
			},
			"patch": {
				Type:     schema.Object,
				Desc:     "Only the fields that should change.",
				Required: true,
			},
		}),
	}, nil
}

type proposeUpdateArgs struct {
	Entity string         `json:"entity"`
	ID     string         `json:"id"`
	Patch  map[string]any `json:"patch"`
}

func (t *proposeUpdateTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args proposeUpdateArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse propose_update arguments: %w", err)
	}
	if err := validateFlatEntity(args.Entity); err != nil {
		return "", err
	}
	if args.ID == "" {
		return "", errors.New("id is required")
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	sink.add(Proposal{Type: "flat_update", Entity: args.Entity, ID: args.ID, Patch: stripSortOrder(args.Patch)})
	return fmt.Sprintf("proposed: update %s %s", args.Entity, args.ID), nil
}

// --- propose_delete ----------------------------------------------------------

type proposeDeleteTool struct{}

func newProposeDeleteTool() tool.InvokableTool { return &proposeDeleteTool{} }

func (t *proposeDeleteTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_delete",
		Desc: "Propose removing an existing resume entry. This does not save anything — it stages a change for the user to accept.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"entity": flatEntityInfo,
			"id": {
				Type:     schema.String,
				Desc:     "ID of the entry to delete, as seen via read_resume.",
				Required: true,
			},
		}),
	}, nil
}

type proposeDeleteArgs struct {
	Entity string `json:"entity"`
	ID     string `json:"id"`
}

func (t *proposeDeleteTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args proposeDeleteArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse propose_delete arguments: %w", err)
	}
	if err := validateFlatEntity(args.Entity); err != nil {
		return "", err
	}
	if args.ID == "" {
		return "", errors.New("id is required")
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	sink.add(Proposal{Type: "flat_delete", Entity: args.Entity, ID: args.ID})
	return fmt.Sprintf("proposed: delete %s %s", args.Entity, args.ID), nil
}

// --- propose_skill_change ----------------------------------------------------
//
// One tool covers all six skill-group/skill-item operations instead of six
// separate tools, since they share the same shallow argument shape.

var skillActions = map[string]string{
	"create_group": "skill_group_create",
	"update_group": "skill_group_update",
	"delete_group": "skill_group_delete",
	"create_item":  "skill_item_create",
	"update_item":  "skill_item_update",
	"delete_item":  "skill_item_delete",
}

type proposeSkillChangeTool struct{}

func newProposeSkillChangeTool() tool.InvokableTool { return &proposeSkillChangeTool{} }

func (t *proposeSkillChangeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_skill_change",
		Desc: "Propose a change to skill groups or skill items (skills are grouped, e.g. 'Languages' containing 'Go', 'Python'). This does not save anything. Never include sort_order.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "Which operation to perform.",
				Enum:     []string{"create_group", "update_group", "delete_group", "create_item", "update_item", "delete_item"},
				Required: true,
			},
			"group_id": {
				Type: schema.String,
				Desc: "ID of the skill group. Required for create_item/update_item/delete_item/update_group/delete_group; omit for create_group.",
			},
			"id": {
				Type: schema.String,
				Desc: "ID of the group (for update_group/delete_group) or item (for update_item/delete_item) being changed. Omit for create_* actions.",
			},
			"data": {
				Type: schema.Object,
				Desc: "For create_group: {group_name}. For create_item: {name, proficiency}. For update_group/update_item: only the fields that should change.",
			},
		}),
	}, nil
}

type proposeSkillChangeArgs struct {
	Action  string         `json:"action"`
	GroupID string         `json:"group_id"`
	ID      string         `json:"id"`
	Data    map[string]any `json:"data"`
}

func (t *proposeSkillChangeTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args proposeSkillChangeArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse propose_skill_change arguments: %w", err)
	}
	proposalType, ok := skillActions[args.Action]
	if !ok {
		return "", fmt.Errorf("unknown action %q", args.Action)
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	p := Proposal{Type: proposalType, GroupID: args.GroupID, ID: args.ID}
	switch args.Action {
	case "create_group":
		p.TempID = tempID()
		p.Fields = stripSortOrder(args.Data)
	case "create_item":
		if args.GroupID == "" {
			return "", errors.New("group_id is required for create_item")
		}
		p.TempID = tempID()
		p.Fields = stripSortOrder(args.Data)
	case "update_group", "update_item":
		if args.ID == "" {
			return "", fmt.Errorf("id is required for %s", args.Action)
		}
		p.Patch = stripSortOrder(args.Data)
	case "delete_group", "delete_item":
		if args.ID == "" {
			return "", fmt.Errorf("id is required for %s", args.Action)
		}
	}

	sink.add(p)
	return fmt.Sprintf("proposed: %s", args.Action), nil
}

// --- propose_custom_section_change --------------------------------------------
//
// Same shape as propose_skill_change, for the custom-sections/entries tree.

var customSectionActions = map[string]string{
	"create_section": "custom_section_create",
	"update_section": "custom_section_update",
	"delete_section": "custom_section_delete",
	"create_entry":   "custom_entry_create",
	"update_entry":   "custom_entry_update",
	"delete_entry":   "custom_entry_delete",
}

type proposeCustomSectionChangeTool struct{}

func newProposeCustomSectionChangeTool() tool.InvokableTool { return &proposeCustomSectionChangeTool{} }

func (t *proposeCustomSectionChangeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_custom_section_change",
		Desc: "Propose a change to a custom (user-defined) resume section or its entries. This does not save anything. Never include sort_order.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "Which operation to perform.",
				Enum:     []string{"create_section", "update_section", "delete_section", "create_entry", "update_entry", "delete_entry"},
				Required: true,
			},
			"section_id": {
				Type: schema.String,
				Desc: "ID of the custom section. Required for create_entry/update_entry/delete_entry/update_section/delete_section; omit for create_section.",
			},
			"id": {
				Type: schema.String,
				Desc: "ID of the section (for update_section/delete_section) or entry (for update_entry/delete_entry) being changed. Omit for create_* actions.",
			},
			"data": {
				Type: schema.Object,
				Desc: "For create_section: {title}. For create_entry: {title, description, entry_date}. For update_section/update_entry: only the fields that should change.",
			},
		}),
	}, nil
}

type proposeCustomSectionChangeArgs struct {
	Action    string         `json:"action"`
	SectionID string         `json:"section_id"`
	ID        string         `json:"id"`
	Data      map[string]any `json:"data"`
}

func (t *proposeCustomSectionChangeTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args proposeCustomSectionChangeArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse propose_custom_section_change arguments: %w", err)
	}
	proposalType, ok := customSectionActions[args.Action]
	if !ok {
		return "", fmt.Errorf("unknown action %q", args.Action)
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	p := Proposal{Type: proposalType, SectionID: args.SectionID, ID: args.ID}
	switch args.Action {
	case "create_section":
		p.TempID = tempID()
		p.Fields = stripSortOrder(args.Data)
	case "create_entry":
		if args.SectionID == "" {
			return "", errors.New("section_id is required for create_entry")
		}
		p.TempID = tempID()
		p.Fields = stripSortOrder(args.Data)
	case "update_section", "update_entry":
		if args.ID == "" {
			return "", fmt.Errorf("id is required for %s", args.Action)
		}
		p.Patch = stripSortOrder(args.Data)
	case "delete_section", "delete_entry":
		if args.ID == "" {
			return "", fmt.Errorf("id is required for %s", args.Action)
		}
	}

	sink.add(p)
	return fmt.Sprintf("proposed: %s", args.Action), nil
}

// --- propose_meta_update -------------------------------------------------------

type proposeMetaUpdateTool struct{}

func newProposeMetaUpdateTool() tool.InvokableTool { return &proposeMetaUpdateTool{} }

func (t *proposeMetaUpdateTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_meta_update",
		Desc: "Propose changing the resume's top-level contact info or summary (full_name, headline, email, phone, location, summary, links). This does not save anything.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"patch": {
				Type:     schema.Object,
				Desc:     "Only the fields that should change.",
				Required: true,
			},
		}),
	}, nil
}

type proposeMetaUpdateArgs struct {
	Patch map[string]any `json:"patch"`
}

func (t *proposeMetaUpdateTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args proposeMetaUpdateArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse propose_meta_update arguments: %w", err)
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	sink.add(Proposal{Type: "update_meta", Patch: args.Patch})
	return "proposed: update contact info", nil
}
