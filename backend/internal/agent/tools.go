package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
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

// --- resume_chat tools -----------------------------------------------------
//
// Everything below reads/writes via session values (see sinkFromContext and
// sessionKeyDraft above) instead of resume_id arguments or the database —
// the chat agent operates on the client's local draft, which may be ahead of
// what's persisted (see chat.go's package comment), and letting the model
// name an arbitrary resume_id would let it target a resume the user doesn't
// currently have open. propose_* tools never touch storage; they only
// append a Proposal to the run's sink for the HTTP handler to stream out.

// Every propose_* payload passes through normalizeFields (fields.go) before
// it becomes a Proposal — see that file for why an untyped `fields` object
// can't be trusted as the model sends it.

// toolProblem reports a problem the model itself can fix — bad arguments,
// unknown fields, a missing id — as the tool's *result* rather than as a Go
// error.
//
// This distinction matters more than it looks: eino treats an error returned
// from InvokableRun as fatal to the whole run (compose/tool_node.go wraps it
// as "failed to stream tool call ..." and aborts), so returning an error here
// would turn one malformed tool call into a dead conversation — exactly what
// happens when the model's output is truncated mid-JSON by the token limit.
// Returned as a result, the text lands in the transcript as that tool call's
// output, and the model reads it and retries.
//
// Genuine faults (no proposal sink in context — a wiring bug, not something
// the model can act on) stay real errors.
func toolProblem(format string, a ...any) (string, error) {
	return "ERROR: " + fmt.Sprintf(format, a...) +
		". Nothing was staged for this call. Fix the arguments and call the tool again.", nil
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
				Type: schema.Object,
				Desc: "The new entry's fields. Use exactly these names for the chosen entity — " +
					describeFlatEntityFields() +
					". content is Markdown (use '- ' bullet lines); technologies is an array of strings; " +
					"is_current is a boolean; dates are strings like \"2023-01\".",
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
		return toolProblem("could not parse the arguments (%v) — they may have been cut short, so keep this call small", err)
	}
	if err := validateFlatEntity(args.Entity); err != nil {
		return toolProblem("%v", err)
	}
	fields, err := normalizeFields(args.Entity, flatEntitySpecs[args.Entity], args.Fields, true)
	if err != nil {
		return toolProblem("%v", err)
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	id := tempID()
	sink.add(Proposal{Type: "flat_create", Entity: args.Entity, TempID: id, Fields: fields, ToolCallID: compose.GetToolCallID(ctx)})
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
		return toolProblem("could not parse the arguments (%v) — they may have been cut short, so keep this call small", err)
	}
	if err := validateFlatEntity(args.Entity); err != nil {
		return toolProblem("%v", err)
	}
	if args.ID == "" {
		return toolProblem("id is required — use the id shown by read_resume")
	}
	patch, err := normalizeFields(args.Entity, flatEntitySpecs[args.Entity], args.Patch, false)
	if err != nil {
		return toolProblem("%v", err)
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	sink.add(Proposal{Type: "flat_update", Entity: args.Entity, ID: args.ID, Patch: patch, ToolCallID: compose.GetToolCallID(ctx)})
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
		return toolProblem("could not parse the arguments (%v)", err)
	}
	if err := validateFlatEntity(args.Entity); err != nil {
		return toolProblem("%v", err)
	}
	if args.ID == "" {
		return toolProblem("id is required — use the id shown by read_resume")
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	sink.add(Proposal{Type: "flat_delete", Entity: args.Entity, ID: args.ID, ToolCallID: compose.GetToolCallID(ctx)})
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
				Desc: describeSkillChangeFields(),
			},
		}),
	}, nil
}

// skillSpecFor picks the spec matching the action's target (group vs item).
func skillSpecFor(action string) entitySpec {
	if strings.HasSuffix(action, "_item") {
		return skillItemSpec
	}
	return skillGroupSpec
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
		return toolProblem("could not parse the arguments (%v) — they may have been cut short, so keep this call small", err)
	}
	proposalType, ok := skillActions[args.Action]
	if !ok {
		return toolProblem("unknown action %q", args.Action)
	}

	// Arguments are fully validated before the sink is touched, so a bad
	// payload is always reported to the model rather than surfacing as a
	// context/wiring error.
	spec := skillSpecFor(args.Action)
	p := Proposal{Type: proposalType, GroupID: args.GroupID, ID: args.ID, ToolCallID: compose.GetToolCallID(ctx)}
	switch args.Action {
	case "create_group":
		fields, err := normalizeFields("skill group", spec, args.Data, true)
		if err != nil {
			return toolProblem("%v", err)
		}
		p.TempID = tempID()
		p.Fields = fields
	case "create_item":
		if args.GroupID == "" {
			return toolProblem("group_id is required for create_item — propose the group first and pass back the id it returns")
		}
		fields, err := normalizeFields("skill", spec, args.Data, true)
		if err != nil {
			return toolProblem("%v", err)
		}
		p.TempID = tempID()
		p.Fields = fields
	case "update_group", "update_item":
		if args.ID == "" {
			return toolProblem("id is required for %s", args.Action)
		}
		patch, err := normalizeFields("skill", spec, args.Data, false)
		if err != nil {
			return toolProblem("%v", err)
		}
		p.Patch = patch
	case "delete_group", "delete_item":
		if args.ID == "" {
			return toolProblem("id is required for %s", args.Action)
		}
	}

	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}
	sink.add(p)
	// create_group must report its tempId: it's the only way a follow-up
	// create_item can name the group it belongs to, since the group has no
	// real ID until the user accepts it client-side.
	if p.TempID != "" {
		return fmt.Sprintf("proposed: %s (id %s — use this as group_id for items in this group)", args.Action, p.TempID), nil
	}
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
				Desc: describeCustomSectionChangeFields(),
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
		return toolProblem("could not parse the arguments (%v) — they may have been cut short, so keep this call small", err)
	}
	proposalType, ok := customSectionActions[args.Action]
	if !ok {
		return toolProblem("unknown action %q", args.Action)
	}

	// Validate before touching the sink — see propose_skill_change above.
	spec := customSectionSpec
	label := "custom section"
	if strings.HasSuffix(args.Action, "_entry") {
		spec, label = customEntrySpec, "custom section entry"
	}

	p := Proposal{Type: proposalType, SectionID: args.SectionID, ID: args.ID, ToolCallID: compose.GetToolCallID(ctx)}
	switch args.Action {
	case "create_section":
		fields, err := normalizeFields(label, spec, args.Data, true)
		if err != nil {
			return toolProblem("%v", err)
		}
		p.TempID = tempID()
		p.Fields = fields
	case "create_entry":
		if args.SectionID == "" {
			return toolProblem("section_id is required for create_entry — propose the section first and pass back the id it returns")
		}
		fields, err := normalizeFields(label, spec, args.Data, true)
		if err != nil {
			return toolProblem("%v", err)
		}
		p.TempID = tempID()
		p.Fields = fields
	case "update_section", "update_entry":
		if args.ID == "" {
			return toolProblem("id is required for %s", args.Action)
		}
		patch, err := normalizeFields(label, spec, args.Data, false)
		if err != nil {
			return toolProblem("%v", err)
		}
		p.Patch = patch
	case "delete_section", "delete_entry":
		if args.ID == "" {
			return toolProblem("id is required for %s", args.Action)
		}
	}

	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}
	sink.add(p)
	// As with skill groups, a new section's tempId is what a follow-up
	// create_entry needs for section_id.
	if p.TempID != "" {
		return fmt.Sprintf("proposed: %s (id %s — use this as section_id for entries in this section)", args.Action, p.TempID), nil
	}
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
		return toolProblem("could not parse the arguments (%v) — they may have been cut short, so keep this call small", err)
	}
	patch, err := normalizeFields("contact info", resumeMetaSpec, args.Patch, false)
	if err != nil {
		return toolProblem("%v", err)
	}
	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}

	sink.add(Proposal{Type: "update_meta", Patch: patch, ToolCallID: compose.GetToolCallID(ctx)})
	return "proposed: update contact info", nil
}
