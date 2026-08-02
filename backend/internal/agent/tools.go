package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"resume-builder/backend/internal/design"

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
	sessionKeyDraft     = "chat_draft_json"
	sessionKeySink      = "chat_proposal_sink"
	sessionKeyTemplates = "chat_templates_json"
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

// TemplatesFetcher fetches the template catalog (id, name, renderer_key,
// supported_modes, supported_groups, default_design — one row per template)
// as JSON, on demand. The agent package defines only this function type, not
// an implementation — Chat's caller (agent_handler.go) supplies the actual
// closure, backed by internal/service.TemplateService, which is a DB read.
// That keeps the DB dependency in the handler layer, where it already
// exists for the resume-ownership check, instead of importing
// internal/service (or internal/repository) into this package — see
// doc.go's split-out plan, which only allows Eino/Claude and per-request
// client-supplied data as dependencies. The fetcher is only invoked if
// list_templates is actually called, so a turn that never asks about
// templates costs zero DB round-trips, not just zero model tokens.
type TemplatesFetcher func(ctx context.Context) (string, error)

// listTemplatesTool calls the TemplatesFetcher supplied to Agent.Chat —
// fetching the catalog fresh from the database, unlike read_resume/the
// draft, which is client-supplied because it may be ahead of what's
// persisted (see chat.go's Chat doc comment). The template catalog has no
// such staleness concern — it's global, not resume-scoped, and nothing the
// user does in the editor can make their own local copy of it stale.
type listTemplatesTool struct{}

func newListTemplatesTool() tool.InvokableTool { return &listTemplatesTool{} }

func (t *listTemplatesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "list_templates",
		Desc:        "List the resume templates available to switch between, including each one's id, name, renderer_key, which layout modes it supports, and which design.Schema() groups it exposes. Fetched fresh from the database when called. Call this only when the user asks about templates or you need a template's id/capabilities — not on every turn.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *listTemplatesTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	fetch, err := templatesFetcherFromContext(ctx)
	if err != nil {
		return "", err
	}
	templatesJSON, err := fetch(ctx)
	if err != nil {
		return toolProblem("could not load the template catalog (%v)", err)
	}
	if templatesJSON == "" {
		return "[]", nil
	}
	return templatesJSON, nil
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
		Desc: "Propose adding a new entry to a resume section (work experience, education, project, certification, language, or misc entry). " +
			"This does not save anything — it stages a change for the user to review and accept or reject in the UI. " +
			"Use this when the user describes new experience in natural language; turn their description into well-structured, " +
			"achievement-focused fields rather than asking them to fill out a form, but only include numbers or outcomes the user " +
			"actually stated — do not invent metrics. Never include sort_order; the client places new entries.",
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
		Desc: "Propose changing one or more fields on an existing resume entry. This does not save anything — it stages a change for the " +
			"user to review and accept or reject in the UI. Requires the exact id of the entry, as returned by read_resume; if you don't " +
			"already have current ids and values for this entry from earlier in this conversation, call read_resume first rather than " +
			"guessing or reusing an id from a different entry. patch should contain only the fields that should change — omitted fields " +
			"are left as they are. If more than one entry could match what the user described, ask which one before calling this. " +
			"Never include sort_order.",
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
		Desc: "Propose removing an existing resume entry. This does not save anything — it stages a change for the user to review and " +
			"accept or reject in the UI. Requires the exact id of the entry, as returned by read_resume; call read_resume first if you " +
			"don't already have it. Only use this when the user has clearly asked to remove a specific entry — if it's ambiguous which " +
			"entry they mean, or if they're describing a change rather than a removal, ask or use propose_update instead.",
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
		Desc: "Propose a change to skill groups or the items within them (skills are grouped, e.g. a \"Languages\" group containing \"Go\" " +
			"and \"Python\" as items). This does not save anything — it stages a change for the user to review and accept or reject in the " +
			"UI. create_group must be proposed before any create_item that belongs to it, in the same turn — pass the id create_group " +
			"returns back as group_id. For update_item/delete_item/update_group/delete_group, you need the item or group's exact id from " +
			"read_resume (or from a create_* you proposed earlier in this conversation) — never guess it. If it's unclear which group an " +
			"item belongs to or which item the user means, ask rather than assume. Never include sort_order.",
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
		Desc: "Propose a change to a custom (user-defined) resume section or its entries. This does not save anything — it stages a " +
			"change for the user to review and accept or reject in the UI. create_section must be proposed before any create_entry that " +
			"belongs to it, in the same turn — pass the id create_section returns back as section_id. For " +
			"update_entry/delete_entry/update_section/delete_section, you need the section or entry's exact id from read_resume (or from " +
			"a create_* proposed earlier in this conversation) — never guess it. Never include sort_order.",
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
		Desc: "Propose changing the resume's top-level contact info or summary (full_name, headline, email, phone, location, summary, " +
			"links). This does not save anything — it stages a change for the user to review and accept or reject in the UI. patch " +
			"should contain only the fields that should change. Use read_resume first if you need to see current values (e.g. to edit " +
			"the existing summary rather than overwrite it blind).",
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

// --- propose_design_update ----------------------------------------------------
//
// The design/styling counterpart to the propose_* content tools above: it
// only stages a change (a "design_update" Proposal, applied client-side only
// once accepted — see proposal.go), it never touches the database, and it
// validates against internal/design.Schema()/ValidateFieldValue() — the same
// pure-Go, DB-free source of truth the REST design PUT handler and the
// Design Mode UI already use (see design.go's package comment, which
// anticipated this exact tool). Importing internal/design does not
// reintroduce the internal/service/internal/auth/cmd/api coupling doc.go's
// split-out plan avoids — it's a sibling leaf package with no DB or HTTP
// dependency of its own.

// designSchemaDescription renders every editable design field (key, type,
// and its enum options / numeric range) from design.Schema() into tool-
// description text, so the model's view of what's editable can never drift
// from what ValidateFieldValue() below actually accepts.
func designSchemaDescription() string {
	var b strings.Builder
	for _, group := range design.Schema() {
		for _, f := range group.Fields {
			b.WriteString(f.Key)
			switch f.Type {
			case design.FieldEnum:
				b.WriteString(" (one of: ")
				b.WriteString(strings.Join(f.Options, "/"))
				b.WriteString(")")
			case design.FieldColor:
				b.WriteString(" (hex color, e.g. #14181d)")
			case design.FieldBool:
				b.WriteString(" (boolean)")
			case design.FieldNumber:
				b.WriteString(fmt.Sprintf(" (number, %v-%v)", *f.Min, *f.Max))
			}
			b.WriteString("; ")
		}
	}
	return b.String()
}

type proposeDesignUpdateTool struct{}

func newProposeDesignUpdateTool() tool.InvokableTool { return &proposeDesignUpdateTool{} }

func (t *proposeDesignUpdateTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_design_update",
		Desc: "Propose changing one or more resume styling/design settings (fonts, colors, spacing, headings, links, footer, etc). " +
			"This does not save anything — it stages a change for the user to review and accept or reject in the UI. Some keys only " +
			"apply to certain templates or layout modes (for example, sidebar-related display isn't meaningful on a single-column " +
			"template) — if you're not sure the current template supports a key you're about to propose, check via read_resume first " +
			"rather than proposing it speculatively. Template choice itself and section order/visibility/titles are separate concerns " +
			"— use propose_template_switch and propose_section_update for those instead. Only propose keys from the list below. " +
			"Valid keys and their allowed values: " + designSchemaDescription(),
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"updates": {
				Type:     schema.Object,
				Desc:     "Map of design field key -> new value, e.g. {\"colors.accent\": \"#3457d5\", \"typography.baseFontSizePt\": 12}. Use the exact dot-path keys listed in this tool's description.",
				Required: true,
			},
		}),
	}, nil
}

type proposeDesignUpdateArgs struct {
	Updates map[string]any `json:"updates"`
}

// currentDesignMap reads the client's current design object (part of the
// same draft JSON read_resume returns) out of session state, as a raw
// map[string]any — not a typed design.ResumeDesign — so setNestedField can
// build a patch that carries forward every sibling value in a touched group
// without needing a full unmarshal/marshal round-trip.
func currentDesignMap(ctx context.Context) (map[string]any, error) {
	v, ok := adk.GetSessionValue(ctx, sessionKeyDraft)
	if !ok {
		return nil, errors.New("no resume draft in context (tool called outside Agent.Chat)")
	}
	draftJSON, ok := v.(string)
	if !ok {
		return nil, errors.New("resume draft has unexpected type")
	}
	var draft struct {
		Design map[string]any `json:"design"`
	}
	if err := json.Unmarshal([]byte(draftJSON), &draft); err != nil {
		return nil, fmt.Errorf("parse draft: %w", err)
	}
	if draft.Design == nil {
		draft.Design = map[string]any{}
	}
	return draft.Design, nil
}

// setNestedField writes value at the dot-path parts inside obj, creating
// intermediate objects as needed and reusing (mutating) any that already
// exist — mirroring frontend/src/lib/designField.ts's buildFieldPatch, which
// spreads each level's existing siblings rather than replacing the whole
// sub-object.
func setNestedField(obj map[string]any, parts []string, value any) {
	if len(parts) == 1 {
		obj[parts[0]] = value
		return
	}
	sub, ok := obj[parts[0]].(map[string]any)
	if !ok {
		sub = map[string]any{}
		obj[parts[0]] = sub
	}
	setNestedField(sub, parts[1:], value)
}

func (t *proposeDesignUpdateTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args proposeDesignUpdateArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolProblem("could not parse the arguments (%v) — they may have been cut short, so keep this call small", err)
	}
	if len(args.Updates) == 0 {
		return toolProblem("updates must contain at least one key/value pair")
	}

	working, err := currentDesignMap(ctx)
	if err != nil {
		return "", err
	}

	touched := map[string]bool{}
	for key, value := range args.Updates {
		if err := design.ValidateFieldValue(key, value); err != nil {
			return toolProblem("%v", err)
		}
		parts := strings.Split(key, ".")
		setNestedField(working, parts, value)
		touched[parts[0]] = true
	}

	patch := make(map[string]any, len(touched))
	for top := range touched {
		patch[top] = working[top]
	}

	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}
	sink.add(Proposal{Type: "design_update", DesignPatch: patch, DesignUpdates: args.Updates, ToolCallID: compose.GetToolCallID(ctx)})

	keys := make([]string, 0, len(args.Updates))
	for k := range args.Updates {
		keys = append(keys, k)
	}
	return fmt.Sprintf("proposed: update design (%s)", strings.Join(keys, ", ")), nil
}

// --- propose_template_switch --------------------------------------------------
//
// Mirrors frontend/src/components/editor/DesignMode.tsx's TemplatePicker
// exactly: switching template fully replaces every design group with the
// target template's default_design, overrides templateId, but carries the
// CURRENT resume's sectionOrder forward unchanged (sectionOrder is
// resume-specific — which sections a person has and how they've arranged
// them — not something a template's defaults should discard). Same
// "design_update" Proposal type as propose_design_update; no new Proposal
// fields, no frontend changes.

type proposeTemplateSwitchTool struct{}

func newProposeTemplateSwitchTool() tool.InvokableTool { return &proposeTemplateSwitchTool{} }

func (t *proposeTemplateSwitchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_template_switch",
		Desc: "Propose switching the resume to a different template. This does not save anything — it stages a change for the user to " +
			"review and accept or reject in the UI. Call list_templates first if you don't already know the target template's id and " +
			"capabilities from earlier in this conversation — don't guess a template id. Note that some previously proposed or accepted " +
			"design settings and section placements may not carry over cleanly to a template with a different layout mode; mention this " +
			"to the user if you know the target template's mode differs from the current one.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"template_id": {
				Type:     schema.String,
				Desc:     "id of the template to switch to, as returned by list_templates.",
				Required: true,
			},
		}),
	}, nil
}

type proposeTemplateSwitchArgs struct {
	TemplateID string `json:"template_id"`
}

// templatesFetcherFromContext mirrors sinkFromContext/currentDesignMap's
// shape for the third session value Chat sets — see Chat's doc comment in
// chat.go for why this is a fetch closure rather than a plain string like
// draftJSON.
func templatesFetcherFromContext(ctx context.Context) (TemplatesFetcher, error) {
	v, ok := adk.GetSessionValue(ctx, sessionKeyTemplates)
	if !ok {
		return nil, errors.New("no template fetcher in context (tool called outside Agent.Chat)")
	}
	fetch, ok := v.(TemplatesFetcher)
	if !ok {
		return nil, errors.New("template fetcher has unexpected type")
	}
	return fetch, nil
}

func (t *proposeTemplateSwitchTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args proposeTemplateSwitchArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolProblem("could not parse the arguments (%v) — they may have been cut short, so keep this call small", err)
	}
	if args.TemplateID == "" {
		return toolProblem("template_id is required — call list_templates to find it")
	}

	fetch, err := templatesFetcherFromContext(ctx)
	if err != nil {
		return "", err
	}
	catalogJSON, err := fetch(ctx)
	if err != nil {
		return toolProblem("could not load the template catalog (%v)", err)
	}
	var catalog []struct {
		ID            string          `json:"id"`
		Name          string          `json:"name"`
		DefaultDesign json.RawMessage `json:"default_design"`
	}
	if err := json.Unmarshal([]byte(catalogJSON), &catalog); err != nil {
		return "", fmt.Errorf("parse template catalog: %w", err)
	}

	var target *struct {
		ID            string          `json:"id"`
		Name          string          `json:"name"`
		DefaultDesign json.RawMessage `json:"default_design"`
	}
	for i := range catalog {
		if catalog[i].ID == args.TemplateID {
			target = &catalog[i]
			break
		}
	}
	if target == nil {
		return toolProblem("no template with id %q — call list_templates to see valid ids", args.TemplateID)
	}

	patch := map[string]any{}
	if err := json.Unmarshal(target.DefaultDesign, &patch); err != nil {
		return "", fmt.Errorf("parse template %s default_design: %w", target.ID, err)
	}
	patch["templateId"] = target.ID

	current, err := currentDesignMap(ctx)
	if err != nil {
		return "", err
	}
	if sectionOrder, ok := current["sectionOrder"]; ok {
		patch["sectionOrder"] = sectionOrder
	}

	// Validate the resulting document defensively before staging — the PUT
	// endpoint this eventually syncs through does the same design.Validate()
	// call with no special-casing for a templateId change (see
	// resume_design_service.go), so anything wrong here would otherwise only
	// surface as a failed save, with no earlier warning to the user.
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return "", fmt.Errorf("marshal merged design: %w", err)
	}
	var merged design.ResumeDesign
	if err := json.Unmarshal(patchJSON, &merged); err != nil {
		return "", fmt.Errorf("unmarshal merged design: %w", err)
	}
	if err := design.Validate(merged); err != nil {
		return toolProblem("switching to %q would produce an invalid design (%v) — this looks like a problem with the template itself, not something you can fix by retrying", target.Name, err)
	}

	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}
	sink.add(Proposal{
		Type:          "design_update",
		DesignPatch:   patch,
		DesignUpdates: map[string]any{"templateId": target.ID, "template": target.Name},
		ToolCallID:    compose.GetToolCallID(ctx),
	})
	return fmt.Sprintf("proposed: switch template to %s (%s)", target.Name, target.ID), nil
}

// --- propose_section_update ---------------------------------------------------
//
// Mirrors frontend/src/components/editor/DesignMode.tsx's SectionsPanel /
// TwoColumnSectionsPanel: every action here ends up staging a
// "design_update" Proposal whose DesignPatch only ever touches the single
// top-level "sectionOrder" key — spreading the whole sectionOrder object
// forward and replacing just the addressed sub-list (sectionOrder.one.sections,
// .two.left, or .two.right), same shallow-merge contract every other
// design_update proposal relies on (resumeDraftReducer.ts's design_update
// case). Ref identity mirrors frontend/src/lib/sectionOrder.ts's
// sectionRefKey: a plain section type ("work_experience"), or
// "custom-<customSectionId>" for a custom section.

// sectionOrderPath maps a column name to its dot-path inside
// ResumeDesign.sectionOrder — "one" is the single flat list one-column
// templates use; "left"/"right" are the two-column template's independent
// lists. "mix" bands aren't exposed here — no template renders that mode yet
// (same scope Schema() and the manual Sections UI already stick to).
func sectionOrderPath(column string) ([]string, error) {
	switch column {
	case "one":
		return []string{"one", "sections"}, nil
	case "left":
		return []string{"two", "left"}, nil
	case "right":
		return []string{"two", "right"}, nil
	default:
		return nil, fmt.Errorf("column must be one of one/left/right, got %q", column)
	}
}

// sectionRefKey mirrors frontend/src/lib/sectionOrder.ts's sectionRefKey.
func sectionRefKey(ref design.SectionRef) string {
	if ref.SectionType == "custom" && ref.CustomSectionID != nil {
		return "custom-" + *ref.CustomSectionID
	}
	return ref.SectionType
}

// parseSectionKey is sectionRefKey's inverse — used to synthesize a default
// ref for a key the current list doesn't have yet (e.g. reorder naming a
// section that isn't in sectionOrder at all so far).
func parseSectionKey(key string) (sectionType string, customSectionID *string) {
	if id, ok := strings.CutPrefix(key, "custom-"); ok {
		return "custom", &id
	}
	return key, nil
}

// getNestedField is the read-side counterpart to setNestedField above —
// walks a dot-path of map[string]any levels and returns whatever's there
// (nil if any level along the way is missing).
func getNestedField(obj map[string]any, parts []string) any {
	var cur any = obj
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

// readSectionRefs reads the ref list at path out of sectionOrder (a raw
// map[string]any, as produced by currentDesignMap) and round-trips it
// through JSON into typed design.SectionRef values, so callers can mutate
// with normal struct field access instead of juggling map[string]any.
func readSectionRefs(sectionOrder map[string]any, path []string) ([]design.SectionRef, error) {
	raw := getNestedField(sectionOrder, path)
	if raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal section list: %w", err)
	}
	var refs []design.SectionRef
	if err := json.Unmarshal(data, &refs); err != nil {
		return nil, fmt.Errorf("unmarshal section list: %w", err)
	}
	return refs, nil
}

// writeSectionRefs is readSectionRefs's inverse — round-trips refs back
// through JSON into a []any and writes it at path via setNestedField (which
// preserves every sibling of sectionOrder/sectionOrder.two along the way,
// exactly like the design-field patch builder above).
func writeSectionRefs(sectionOrder map[string]any, path []string, refs []design.SectionRef) error {
	data, err := json.Marshal(refs)
	if err != nil {
		return fmt.Errorf("marshal section list: %w", err)
	}
	var raw []any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal section list: %w", err)
	}
	setNestedField(sectionOrder, path, raw)
	return nil
}

type proposeSectionUpdateTool struct{}

func newProposeSectionUpdateTool() tool.InvokableTool { return &proposeSectionUpdateTool{} }

func (t *proposeSectionUpdateTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_section_update",
		Desc: "Propose reordering, showing/hiding, renaming, or (on two-column templates only) moving a resume section between the " +
			"sidebar and main columns. This does not save anything — it stages a change for the user to review and accept or reject in " +
			"the UI. Section keys are either a plain section type (" + strings.Join(design.ValidSectionTypes(), "/") + ") or " +
			"\"custom-<id>\" for a custom section, using the id from read_resume's custom_sections. The move action and left/right " +
			"column values only apply to two-column templates; check the resume's design.layout.mode via read_resume first if you " +
			"don't already know it — single-column templates use column: \"one\" for everything. A custom-<id> section key referencing " +
			"a custom section you just proposed (not yet accepted) won't actually take effect until that create is accepted first, so " +
			"mention that to the user if relevant.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action": {
				Type:     schema.String,
				Desc:     "reorder: replace a whole column's order. set_visible/set_title: change one section. move: move a section between columns (two-column templates only).",
				Enum:     []string{"reorder", "set_visible", "set_title", "move"},
				Required: true,
			},
			"column": {
				Type: schema.String,
				Desc: "Which sectionOrder list this applies to. Required for reorder/set_visible/set_title.",
				Enum: []string{"one", "left", "right"},
			},
			"order": {
				Type:     schema.Array,
				Desc:     "reorder: the full list of section keys for \"column\", in the desired order. Any key not currently present is added with default visibility.",
				ElemInfo: &schema.ParameterInfo{Type: schema.String},
			},
			"key": {
				Type: schema.String,
				Desc: "set_visible/set_title/move: the section key being changed.",
			},
			"is_visible": {
				Type: schema.Boolean,
				Desc: "set_visible: whether the section should be shown.",
			},
			"title": {
				Type: schema.String,
				Desc: "set_title: custom display title, or omit/empty to reset to the default title.",
			},
			"from_column": {
				Type: schema.String,
				Desc: "move: source column (\"left\" or \"right\").",
				Enum: []string{"left", "right"},
			},
			"to_column": {
				Type: schema.String,
				Desc: "move: destination column (\"left\" or \"right\").",
				Enum: []string{"left", "right"},
			},
			"to_index": {
				Type: schema.Integer,
				Desc: "move: position within the destination column. Omit to append at the end.",
			},
		}),
	}, nil
}

type proposeSectionUpdateArgs struct {
	Action     string   `json:"action"`
	Column     string   `json:"column"`
	Order      []string `json:"order"`
	Key        string   `json:"key"`
	IsVisible  *bool    `json:"is_visible"`
	Title      *string  `json:"title"`
	FromColumn string   `json:"from_column"`
	ToColumn   string   `json:"to_column"`
	ToIndex    *int     `json:"to_index"`
}

func (t *proposeSectionUpdateTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args proposeSectionUpdateArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolProblem("could not parse the arguments (%v) — they may have been cut short, so keep this call small", err)
	}

	currentDesign, err := currentDesignMap(ctx)
	if err != nil {
		return "", err
	}
	sectionOrder, ok := currentDesign["sectionOrder"].(map[string]any)
	if !ok {
		sectionOrder = map[string]any{}
		currentDesign["sectionOrder"] = sectionOrder
	}

	switch args.Action {
	case "reorder":
		if len(args.Order) == 0 {
			return toolProblem("order must list at least one section key")
		}
		path, err := sectionOrderPath(args.Column)
		if err != nil {
			return toolProblem("%v", err)
		}
		current, err := readSectionRefs(sectionOrder, path)
		if err != nil {
			return "", err
		}
		byKey := make(map[string]design.SectionRef, len(current))
		for _, ref := range current {
			byKey[sectionRefKey(ref)] = ref
		}
		next := make([]design.SectionRef, 0, len(args.Order))
		for _, key := range args.Order {
			if ref, ok := byKey[key]; ok {
				next = append(next, ref)
				continue
			}
			sectionType, customID := parseSectionKey(key)
			next = append(next, design.SectionRef{SectionType: sectionType, CustomSectionID: customID, IsVisible: true})
		}
		for _, ref := range next {
			if err := design.ValidateSectionRef(ref); err != nil {
				return toolProblem("%v", err)
			}
		}
		if err := writeSectionRefs(sectionOrder, path, next); err != nil {
			return "", err
		}

	case "set_visible", "set_title":
		if args.Key == "" {
			return toolProblem("key is required for %s", args.Action)
		}
		path, err := sectionOrderPath(args.Column)
		if err != nil {
			return toolProblem("%v", err)
		}
		refs, err := readSectionRefs(sectionOrder, path)
		if err != nil {
			return "", err
		}
		idx := -1
		for i, ref := range refs {
			if sectionRefKey(ref) == args.Key {
				idx = i
				break
			}
		}
		if idx == -1 {
			return toolProblem("no section %q in column %q — call read_resume to see the current sectionOrder", args.Key, args.Column)
		}
		if args.Action == "set_visible" {
			if args.IsVisible == nil {
				return toolProblem("is_visible is required for set_visible")
			}
			refs[idx].IsVisible = *args.IsVisible
		} else {
			var title *string
			if args.Title != nil && *args.Title != "" {
				title = args.Title
			}
			refs[idx].TitleOverride = title
		}
		if err := writeSectionRefs(sectionOrder, path, refs); err != nil {
			return "", err
		}

	case "move":
		if args.Key == "" {
			return toolProblem("key is required for move")
		}
		if args.FromColumn == "one" || args.ToColumn == "one" {
			return toolProblem("move is only for two-column templates (from_column/to_column must be left/right) — use reorder to reposition within a single-column template's list")
		}
		fromPath, err := sectionOrderPath(args.FromColumn)
		if err != nil {
			return toolProblem("from_column: %v", err)
		}
		toPath, err := sectionOrderPath(args.ToColumn)
		if err != nil {
			return toolProblem("to_column: %v", err)
		}
		fromRefs, err := readSectionRefs(sectionOrder, fromPath)
		if err != nil {
			return "", err
		}
		idx := -1
		for i, ref := range fromRefs {
			if sectionRefKey(ref) == args.Key {
				idx = i
				break
			}
		}
		if idx == -1 {
			return toolProblem("no section %q in column %q — call read_resume to see the current sectionOrder", args.Key, args.FromColumn)
		}
		moved := fromRefs[idx]
		fromRefs = append(fromRefs[:idx], fromRefs[idx+1:]...)

		var toRefs []design.SectionRef
		if args.FromColumn == args.ToColumn {
			toRefs = fromRefs
		} else {
			toRefs, err = readSectionRefs(sectionOrder, toPath)
			if err != nil {
				return "", err
			}
		}
		insertAt := len(toRefs)
		if args.ToIndex != nil {
			insertAt = *args.ToIndex
			if insertAt < 0 {
				insertAt = 0
			}
			if insertAt > len(toRefs) {
				insertAt = len(toRefs)
			}
		}
		toRefs = append(toRefs[:insertAt], append([]design.SectionRef{moved}, toRefs[insertAt:]...)...)

		// toPath already reflects the reinsertion (toRefs, which for a
		// same-column move IS fromRefs post-splice-and-reinsert). Only write
		// fromPath separately when it's a different list.
		if err := writeSectionRefs(sectionOrder, toPath, toRefs); err != nil {
			return "", err
		}
		if args.FromColumn != args.ToColumn {
			if err := writeSectionRefs(sectionOrder, fromPath, fromRefs); err != nil {
				return "", err
			}
		}

	default:
		return toolProblem("unknown action %q", args.Action)
	}

	updates := map[string]any{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &updates); err != nil {
		updates = nil // display-only; fall through with no detail rather than fail the whole call
	}

	sink, err := sinkFromContext(ctx)
	if err != nil {
		return "", err
	}
	sink.add(Proposal{
		Type:          "design_update",
		DesignPatch:   map[string]any{"sectionOrder": sectionOrder},
		DesignUpdates: updates,
		ToolCallID:    compose.GetToolCallID(ctx),
	})
	return fmt.Sprintf("proposed: section %s", args.Action), nil
}
