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

func toolProblem(format string, a ...any) (string, error) {
	return "ERROR: " + fmt.Sprintf(format, a...) +
		". Nothing was changed by this call. Fix the arguments and call the tool again.", nil
}

type readResumeTool struct{}

func newReadResumeTool() tool.InvokableTool { return &readResumeTool{} }

func (t *readResumeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "read_resume",
		Desc:        "Read the user's current resume draft (work experience, education, skills, projects, certifications, languages, misc entries, custom sections, contact info). Takes no arguments.",
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

type TemplatesFetcher func(ctx context.Context) (string, error)

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

type proposeCreateTool struct{}

func newProposeCreateTool() tool.InvokableTool { return &proposeCreateTool{} }

func (t *proposeCreateTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "create_resume_entry",
		Desc: "Add a new entry to a resume section (work experience, education, project, certification, language, or misc entry). " +
			"Use this when the user describes new experience in natural language; turn their description into well-structured, " +
			"achievement-focused fields rather than asking them to fill out a form, but only include numbers or outcomes the user " +
			"actually stated — do not invent metrics. Never include sort_order; the app places new entries.",
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
	return fmt.Sprintf("created %s (tempId %s)", args.Entity, id), nil
}

type proposeUpdateTool struct{}

func newProposeUpdateTool() tool.InvokableTool { return &proposeUpdateTool{} }

func (t *proposeUpdateTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "update_resume_entry",
		Desc: "Change one or more fields on an existing resume entry. Requires the exact id of the entry, as returned by read_resume; " +
			"if you don't already have current ids and values for this entry from earlier in this conversation, call read_resume first " +
			"rather than guessing or reusing an id from a different entry. patch should contain only the fields that should change — " +
			"omitted fields are left as they are. If more than one entry could match what the user described, ask which one before " +
			"calling this. Never include sort_order.",
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
	return fmt.Sprintf("updated %s %s", args.Entity, args.ID), nil
}

type proposeDeleteTool struct{}

func newProposeDeleteTool() tool.InvokableTool { return &proposeDeleteTool{} }

func (t *proposeDeleteTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "delete_resume_entry",
		Desc: "Remove an existing resume entry. Requires the exact id of the entry, as returned by read_resume; call read_resume first " +
			"if you don't already have it. Only use this when the user has clearly asked to remove a specific entry — if it's ambiguous " +
			"which entry they mean, or if they're describing a change rather than a removal, ask or use update_resume_entry instead.",
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
	return fmt.Sprintf("deleted %s %s", args.Entity, args.ID), nil
}

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
		Name: "update_skills",
		Desc: "Change skill groups or the items within them (skills are grouped, e.g. a \"Languages\" group containing \"Go\" and " +
			"\"Python\" as items). create_group must be called before any create_item that belongs to it, in the same turn — pass the " +
			"id create_group returns back as group_id. For update_item/delete_item/update_group/delete_group, you need the item or " +
			"group's exact id from read_resume (or from a create_* call earlier in this conversation) — never guess it. If it's unclear " +
			"which group an item belongs to or which item the user means, ask rather than assume. Never include sort_order.",
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
			return toolProblem("group_id is required for create_item — create the group first and pass back the id it returns")
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
	if p.TempID != "" {
		return fmt.Sprintf("%s (id %s — use this as group_id for items in this group)", args.Action, p.TempID), nil
	}
	return args.Action, nil
}

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
		Name: "update_custom_section",
		Desc: "Change a custom (user-defined) resume section or its entries. create_section must be called before any create_entry " +
			"that belongs to it, in the same turn — pass the id create_section returns back as section_id. For " +
			"update_entry/delete_entry/update_section/delete_section, you need the section or entry's exact id from read_resume (or " +
			"from a create_* call earlier in this conversation) — never guess it. Never include sort_order.",
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
			return toolProblem("section_id is required for create_entry — create the section first and pass back the id it returns")
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
	if p.TempID != "" {
		return fmt.Sprintf("%s (id %s — use this as section_id for entries in this section)", args.Action, p.TempID), nil
	}
	return args.Action, nil
}

type proposeMetaUpdateTool struct{}

func newProposeMetaUpdateTool() tool.InvokableTool { return &proposeMetaUpdateTool{} }

func (t *proposeMetaUpdateTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "update_contact_info",
		Desc: "Change the resume's top-level contact info or summary (full_name, headline, email, phone, location, summary, links). " +
			"patch should contain only the top-level fields that should change; each included field replaces its old value entirely. " +
			"links is an array of {label, url} objects (e.g. [{\"label\": \"GitHub\", \"url\": \"https://github.com/...\"}]) — to change, " +
			"add, or remove one link, call read_resume for the current links array, then send the whole array back in patch.links with " +
			"that one entry changed and the rest unchanged; you don't need a url to change a label, or vice versa. Use read_resume first " +
			"if you need to see current values for anything in this tool (e.g. to edit the existing summary rather than overwrite it " +
			"blind, or to edit one link without dropping the others).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"patch": {
				Type: schema.Object,
				Desc: "Only the top-level fields that should change. For links specifically, this must be the complete array you want " +
					"the resume to end up with — it replaces the existing array rather than merging into it.",
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
	return "updated contact info", nil
}

func designSchemaDescription() string {
	var b strings.Builder
	for _, group := range design.Schema() {
		for _, f := range group.Fields {
			b.WriteString(f.Key)
			b.WriteString(" (")
			b.WriteString(f.Label)
			switch f.Type {
			case design.FieldEnum:
				b.WriteString("; one of: ")
				b.WriteString(strings.Join(f.Options, "/"))
			case design.FieldColor:
				b.WriteString("; hex color, e.g. #14181d")
			case design.FieldBool:
				b.WriteString("; boolean")
			case design.FieldNumber:
				fmt.Fprintf(&b, "; number, %v-%v", *f.Min, *f.Max)
			}
			b.WriteString("); ")
		}
	}
	return b.String()
}

type proposeDesignUpdateTool struct{}

func newProposeDesignUpdateTool() tool.InvokableTool { return &proposeDesignUpdateTool{} }

func (t *proposeDesignUpdateTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "update_design",
		Desc: "Change one or more resume styling/design settings (fonts, colors, spacing, headings, links, footer, etc). Some keys " +
			"only apply to certain templates or layout modes (for example, sidebar-related display isn't meaningful on a single-column " +
			"template) — if you're not sure the current template supports a key you're about to set, check via read_resume first rather " +
			"than setting it speculatively. Template choice itself and section order/visibility/titles are separate concerns — use " +
			"switch_template and update_section_layout for those instead. Only use keys from the list below. " +
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
	return fmt.Sprintf("updated design (%s)", strings.Join(keys, ", ")), nil
}

type proposeTemplateSwitchTool struct{}

func newProposeTemplateSwitchTool() tool.InvokableTool { return &proposeTemplateSwitchTool{} }

func (t *proposeTemplateSwitchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "switch_template",
		Desc: "Switch the resume to a different template. Call list_templates first if you don't already know the target template's id " +
			"and capabilities from earlier in this conversation — don't guess a template id. Note that some previously set design " +
			"settings and section placements may not carry over cleanly to a template with a different layout mode; mention this to " +
			"the user if you know the target template's mode differs from the current one.",
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
	return fmt.Sprintf("switched template to %s (%s)", target.Name, target.ID), nil
}

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

func sectionRefKey(ref design.SectionRef) string {
	if ref.SectionType == "custom" && ref.CustomSectionID != nil {
		return "custom-" + *ref.CustomSectionID
	}
	return ref.SectionType
}

func parseSectionKey(key string) (sectionType string, customSectionID *string) {
	if id, ok := strings.CutPrefix(key, "custom-"); ok {
		return "custom", &id
	}
	return key, nil
}

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
		Name: "update_section_layout",
		Desc: "Reorder, show/hide, rename, or (on two-column templates only) move a resume section between the sidebar and main " +
			"columns. Section keys are either a plain section type (" + strings.Join(design.ValidSectionTypes(), "/") + ") or " +
			"\"custom-<id>\" for a custom section, using the id from read_resume's custom_sections. The move action and left/right " +
			"column values only apply to two-column templates; check the resume's design.layout.mode via read_resume first if you " +
			"don't already know it — single-column templates use column: \"one\" for everything. A custom-<id> section key only works " +
			"once that custom section actually exists — if you created it earlier this same turn you can reference the id its create " +
			"call returned, but don't reference a custom section id from an earlier turn without confirming via read_resume first that " +
			"it's still there.",
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
		updates = nil
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
	return fmt.Sprintf("section %s applied", args.Action), nil
}
