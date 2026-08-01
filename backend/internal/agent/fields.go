package agent

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

// The model is free to invent field names for any propose_* payload, because
// the tool schema can only describe `fields` as a generic object — its real
// shape depends on which entity was named, and JSON Schema has no way to
// express that dependency in a single object parameter. Left unchecked, the
// model falls back on field names it has memorized from other resume formats
// (JSON Resume's school/area/studyType, camelCase startDate, projects.description
// instead of content), which then flow straight through the frontend reducer
// into draft entities missing the keys the UI reads.
//
// So this file is where an untrusted propose_* payload becomes a payload the
// rest of the system can assume is well-formed:
//
//  1. normalizeFields renames what it can recognize (camelCase, plus the
//     known cross-format aliases below),
//  2. it rejects anything left unrecognized with an error naming the valid
//     fields — tool errors go back to the model, which retries correctly,
//  3. it fills required text fields the UI dereferences (content) so a
//     partial payload can't produce an entity with a missing key.
//
// specs is also the source for the tool descriptions the model sees, so the
// documented shape and the enforced shape can't drift apart.

// entitySpec describes one resume entity's accepted create/update payload.
type entitySpec struct {
	// fields are the only keys accepted after alias normalization.
	fields []string
	// required must be present and non-empty on a create.
	required []string
	// defaults are applied on create when the model omits them. Only
	// non-nullable columns the UI dereferences belong here.
	defaults map[string]any
	// aliases map known-wrong names onto the real field name. Mechanical
	// camelCase->snake_case is handled globally and doesn't need entries.
	aliases map[string]string
	// enums constrain specific fields to a fixed set of values.
	enums map[string][]string
	// dateFields names fields that must parse as a date. The system prompt
	// tells the model to write plain strings like "2023-01", matching
	// however a human would type — but that's not what the manual editor
	// actually sends (WorkExperienceSection.tsx etc. always send a full
	// RFC3339 timestamp via toISOString()), and it's not something
	// time.Time's default JSON decoding accepts either. Left unchecked, an
	// agent-authored date reaches the backend's flushDraft POST and fails
	// there as an opaque "invalid request body" with no indication of why —
	// so it's normalized to RFC3339 here instead, before it ever leaves this
	// package.
	dateFields []string
}

// flatEntitySpecs covers the propose_create/update/delete entities. Field
// lists mirror frontend/src/types/resume.ts, minus server-owned columns
// (id, resume_id, sort_order) the agent must never set.
var flatEntitySpecs = map[string]entitySpec{
	"work_experiences": {
		fields:     []string{"company", "company_url", "title", "location", "employment_type", "start_date", "end_date", "is_current", "content", "technologies"},
		required:   []string{"company", "title"},
		dateFields: []string{"start_date", "end_date"},
		defaults:   map[string]any{"content": "", "is_current": false, "technologies": []any{}},
		aliases: map[string]string{
			"name":         "company",
			"organization": "company",
			"employer":     "company",
			"position":     "title",
			"role":         "title",
			"job_title":    "title",
			"description":  "content",
			"summary":      "content",
			"highlights":   "content",
			"bullets":      "content",
			"url":          "company_url",
			"website":      "company_url",
			"current":      "is_current",
			"tech":         "technologies",
			"tech_stack":   "technologies",
			"skills":       "technologies",
		},
	},
	"educations": {
		fields:     []string{"institution", "degree", "field_of_study", "location", "start_date", "end_date", "gpa", "honors_description"},
		required:   []string{"institution"},
		dateFields: []string{"start_date", "end_date"},
		aliases: map[string]string{
			"school":        "institution",
			"university":    "institution",
			"college":       "institution",
			"name":          "institution",
			"study_type":    "degree",
			"qualification": "degree",
			"area":          "field_of_study",
			"major":         "field_of_study",
			"field":         "field_of_study",
			"score":         "gpa",
			"grade":         "gpa",
			"honors":        "honors_description",
			"description":   "honors_description",
			"summary":       "honors_description",
		},
	},
	"projects": {
		fields:     []string{"name", "content", "role", "technologies", "url", "start_date", "end_date"},
		required:   []string{"name"},
		dateFields: []string{"start_date", "end_date"},
		defaults:   map[string]any{"content": "", "technologies": []any{}},
		aliases: map[string]string{
			"title":       "name",
			"project":     "name",
			"description": "content",
			"summary":     "content",
			"highlights":  "content",
			"bullets":     "content",
			"link":        "url",
			"website":     "url",
			"tech":        "technologies",
			"tech_stack":  "technologies",
			"skills":      "technologies",
			"position":    "role",
		},
	},
	"certifications": {
		fields:     []string{"name", "issuer", "issue_date", "expiry_date", "credential_url"},
		required:   []string{"name"},
		dateFields: []string{"issue_date", "expiry_date"},
		aliases: map[string]string{
			"title":           "name",
			"certification":   "name",
			"organization":    "issuer",
			"authority":       "issuer",
			"issued_by":       "issuer",
			"date":            "issue_date",
			"expires":         "expiry_date",
			"expiration":      "expiry_date",
			"expiration_date": "expiry_date",
			"url":             "credential_url",
			"link":            "credential_url",
			"credential_id":   "credential_url",
		},
	},
	"languages": {
		fields:   []string{"name", "proficiency"},
		required: []string{"name"},
		aliases: map[string]string{
			"language": "name",
			"title":    "name",
			"fluency":  "proficiency",
			"level":    "proficiency",
		},
	},
	"misc_entries": {
		fields:     []string{"kind", "title", "issuer_or_org", "url", "entry_date", "description"},
		required:   []string{"kind", "title"},
		dateFields: []string{"entry_date"},
		enums:      map[string][]string{"kind": {"award", "publication", "volunteer"}},
		aliases: map[string]string{
			"type":         "kind",
			"category":     "kind",
			"name":         "title",
			"issuer":       "issuer_or_org",
			"organization": "issuer_or_org",
			"awarder":      "issuer_or_org",
			"publisher":    "issuer_or_org",
			"org":          "issuer_or_org",
			"date":         "entry_date",
			"link":         "url",
			"website":      "url",
			"summary":      "description",
			"content":      "description",
		},
	},
}

// Non-flat entity specs — the skill and custom-section trees, and the
// resume's own contact/summary block.
var (
	skillGroupSpec = entitySpec{
		fields:   []string{"group_name"},
		required: []string{"group_name"},
		aliases:  map[string]string{"name": "group_name", "title": "group_name", "group": "group_name", "category": "group_name"},
	}
	skillItemSpec = entitySpec{
		fields:   []string{"name", "proficiency"},
		required: []string{"name"},
		aliases:  map[string]string{"skill": "name", "title": "name", "level": "proficiency", "fluency": "proficiency"},
	}
	customSectionSpec = entitySpec{
		fields:   []string{"title"},
		required: []string{"title"},
		aliases:  map[string]string{"name": "title", "section_title": "title", "heading": "title"},
	}
	customEntrySpec = entitySpec{
		fields:     []string{"title", "description", "entry_date"},
		dateFields: []string{"entry_date"},
		aliases:    map[string]string{"name": "title", "heading": "title", "summary": "description", "content": "description", "date": "entry_date"},
	}
	resumeMetaSpec = entitySpec{
		fields: []string{"full_name", "headline", "email", "phone", "location", "summary", "links"},
		aliases: map[string]string{
			"name":      "full_name",
			"title":     "headline",
			"tagline":   "headline",
			"label":     "headline",
			"city":      "location",
			"address":   "location",
			"about":     "summary",
			"profile":   "summary",
			"objective": "summary",
			"bio":       "summary",
			"profiles":  "links",
			"urls":      "links",
		},
	}
)

// snakeCase converts camelCase/PascalCase/kebab-case to snake_case so the
// most common class of wrong name (startDate for start_date) normalizes
// mechanically instead of needing an alias entry per field.
func snakeCase(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for i, r := range s {
		switch {
		case r == '-' || r == ' ':
			b.WriteByte('_')
		case r >= 'A' && r <= 'Z':
			if i > 0 && s[i-1] != '_' && s[i-1] != '-' && s[i-1] != ' ' {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s entitySpec) allows(field string) bool {
	return slices.Contains(s.fields, field)
}

// canonical resolves one model-supplied key onto a spec field, or "" if it
// can't be recognized.
func (s entitySpec) canonical(key string) string {
	k := snakeCase(strings.TrimSpace(key))
	if s.allows(k) {
		return k
	}
	if target, ok := s.aliases[k]; ok {
		return target
	}
	return ""
}

func (s entitySpec) checkEnums(out map[string]any) error {
	for field, allowed := range s.enums {
		v, ok := out[field]
		if !ok {
			continue
		}
		got, _ := v.(string)
		got = strings.ToLower(strings.TrimSpace(got))
		if slices.Contains(allowed, got) {
			out[field] = got
		} else {
			return fmt.Errorf("invalid %s %q: must be one of %s", field, v, strings.Join(allowed, ", "))
		}
	}
	return nil
}

// dateLayouts are tried in order. RFC3339 first since that's what a manually
// edited entry already has when it round-trips through read_resume; the
// shorter layouts are what the system prompt actually tells the model to
// write.
var dateLayouts = []string{
	time.RFC3339,
	"2006-01-02",
	"2006-01",
}

// normalizeDateValue accepts any of dateLayouts (or an explicit null/empty
// string, meaning "no date") and returns the canonical RFC3339 string the
// rest of the system already expects — the same format the manual editor's
// toISOString() call produces, so a normalized proposal needs no special
// handling anywhere downstream: not in the frontend draft, not in the
// flushDraft POST, not in the backend's time.Time decoding.
func normalizeDateValue(field string, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("%s must be a date string, got %v", field, v)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format(time.RFC3339), nil
		}
	}
	return nil, fmt.Errorf(`%s %q is not a recognized date — use "2023-01", "2023-01-15", or a full timestamp`, field, s)
}

func (s entitySpec) normalizeDates(out map[string]any) error {
	var bad []string
	for _, field := range s.dateFields {
		v, ok := out[field]
		if !ok {
			continue
		}
		normalized, err := normalizeDateValue(field, v)
		if err != nil {
			bad = append(bad, err.Error())
			continue
		}
		out[field] = normalized
	}
	if len(bad) > 0 {
		return fmt.Errorf("%s", strings.Join(bad, "; "))
	}
	return nil
}

// isBlank reports whether a required field's value is effectively missing.
func isBlank(v any) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) == ""
}

// normalizeFields maps a model-supplied payload onto the spec's real field
// names. Unrecognized keys are an error rather than a silent drop: the error
// text reaches the model as the tool result, so it retries with the right
// names instead of the user silently losing content it meant to add.
//
// create applies required-field checks and defaults; updates skip both, since
// a patch legitimately carries only the fields that change.
func normalizeFields(label string, spec entitySpec, in map[string]any, create bool) (map[string]any, error) {
	out := make(map[string]any, len(in))
	var unknown []string

	for k, v := range in {
		// Server-owned; silently dropped rather than rejected, since the
		// model including them is harmless and rejecting would waste a turn.
		switch snakeCase(k) {
		case "id", "resume_id", "sort_order", "skill_group_id", "custom_section_id", "created_at", "updated_at":
			continue
		}
		field := spec.canonical(k)
		if field == "" {
			unknown = append(unknown, k)
			continue
		}
		out[field] = v
	}

	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf(
			"unknown %s field(s): %s. Valid fields are: %s. Re-send this call using only those names",
			label, strings.Join(unknown, ", "), strings.Join(spec.fields, ", "),
		)
	}

	if err := spec.normalizeDates(out); err != nil {
		return nil, err
	}

	if err := spec.checkEnums(out); err != nil {
		return nil, err
	}

	if !create {
		if len(out) == 0 {
			return nil, fmt.Errorf("no recognized %s fields to change", label)
		}
		return out, nil
	}

	var missing []string
	for _, r := range spec.required {
		if v, ok := out[r]; !ok || isBlank(v) {
			missing = append(missing, r)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required %s field(s): %s", label, strings.Join(missing, ", "))
	}

	// Defaults last so the client never builds an entity missing a key the
	// preview dereferences (a nil content is what crashes the Markdown
	// renderer, not a wrong one).
	for k, v := range spec.defaults {
		if _, ok := out[k]; !ok {
			out[k] = v
		}
	}
	return out, nil
}

// describeFlatEntityFields renders the per-entity field lists into the
// propose_create tool description, so the schema the model reads is generated
// from the same specs that validate its output.
func describeFlatEntityFields() string {
	names := make([]string, 0, len(flatEntitySpecs))
	for name := range flatEntitySpecs {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for i, name := range names {
		if i > 0 {
			b.WriteString("; ")
		}
		spec := flatEntitySpecs[name]
		fmt.Fprintf(&b, "%s: {%s}", name, strings.Join(spec.fields, ", "))
	}
	return b.String()
}

// describeSkillChangeFields and describeCustomSectionChangeFields render
// propose_skill_change/propose_custom_section_change's `data` field
// descriptions from the same specs normalizeFields validates against — the
// same reasoning as describeFlatEntityFields above, so a field added to one
// of these specs can't silently go undocumented to the model.
func describeSkillChangeFields() string {
	return fmt.Sprintf(
		"For create_group: {%s}. For create_item: {%s}. For update_group/update_item: only the fields that should change.",
		strings.Join(skillGroupSpec.fields, ", "),
		strings.Join(skillItemSpec.fields, ", "),
	)
}

func describeCustomSectionChangeFields() string {
	return fmt.Sprintf(
		"For create_section: {%s}. For create_entry: {%s}. For update_section/update_entry: only the fields that should change.",
		strings.Join(customSectionSpec.fields, ", "),
		strings.Join(customEntrySpec.fields, ", "),
	)
}
