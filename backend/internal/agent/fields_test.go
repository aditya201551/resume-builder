package agent

import (
	"slices"
	"strings"
	"testing"
)

func TestSnakeCase(t *testing.T) {
	cases := map[string]string{
		"startDate":     "start_date",
		"issueDate":     "issue_date",
		"studyType":     "study_type",
		"isCurrent":     "is_current",
		"content":       "content",
		"full_name":     "full_name",
		"credential-id": "credential_id",
		"Company":       "company",
	}
	for in, want := range cases {
		if got := snakeCase(in); got != want {
			t.Errorf("snakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}

// The exact payload from the crash report: the model used JSON Resume field
// names for projects, which produced a draft project with no `content` key
// and took down the live preview's Markdown renderer.
func TestNormalizeFieldsRecoversJSONResumeProjectNames(t *testing.T) {
	in := map[string]any{
		"description": "Built a thing.",
		"endDate":     "2023-06",
		"name":        "Expense Tracker",
		"startDate":   "2023-01",
		"url":         "https://example.com",
	}

	out, err := normalizeFields("projects", flatEntitySpecs["projects"], in, true)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	if out["content"] != "Built a thing." {
		t.Errorf("description should map to content, got %#v", out["content"])
	}
	if _, ok := out["description"]; ok {
		t.Error("description should not survive normalization")
	}
	if out["start_date"] != "2023-01-01T00:00:00Z" || out["end_date"] != "2023-06-01T00:00:00Z" {
		t.Errorf("camelCase dates not normalized to RFC3339: %#v", out)
	}
}

func TestNormalizeFieldsRecoversJSONResumeEducationNames(t *testing.T) {
	in := map[string]any{
		"area":      "Computer Science",
		"endDate":   "2018-05",
		"school":    "UC Berkeley",
		"score":     "3.8",
		"studyType": "Bachelor of Science",
	}

	out, err := normalizeFields("educations", flatEntitySpecs["educations"], in, true)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	want := map[string]string{
		"institution":    "UC Berkeley",
		"field_of_study": "Computer Science",
		"degree":         "Bachelor of Science",
		"gpa":            "3.8",
		"end_date":       "2018-05-01T00:00:00Z",
	}
	for k, v := range want {
		if out[k] != v {
			t.Errorf("out[%q] = %#v, want %q", k, out[k], v)
		}
	}
}

// content is what the live preview dereferences; a create that omits it must
// still produce an entity that has the key.
func TestNormalizeFieldsAppliesDefaults(t *testing.T) {
	out, err := normalizeFields("work_experiences", flatEntitySpecs["work_experiences"], map[string]any{
		"company": "Acme",
		"title":   "Engineer",
	}, true)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	if out["content"] != "" {
		t.Errorf("content default = %#v, want empty string", out["content"])
	}
	if out["is_current"] != false {
		t.Errorf("is_current default = %#v, want false", out["is_current"])
	}
	if _, ok := out["technologies"].([]any); !ok {
		t.Errorf("technologies default = %#v, want empty slice", out["technologies"])
	}
}

// An unrecognizable field has to come back as an error the model can act on,
// not be dropped — dropping it silently loses content the user asked for.
func TestNormalizeFieldsRejectsUnknownField(t *testing.T) {
	_, err := normalizeFields("languages", flatEntitySpecs["languages"], map[string]any{
		"name":          "English",
		"totallyMadeUp": "x",
	}, true)
	if err == nil {
		t.Fatal("expected an error for an unknown field")
	}
	if !strings.Contains(err.Error(), "totallyMadeUp") {
		t.Errorf("error should name the offending field, got: %v", err)
	}
	if !strings.Contains(err.Error(), "proficiency") {
		t.Errorf("error should list the valid fields, got: %v", err)
	}
}

func TestNormalizeFieldsRequiresRequiredFieldsOnCreate(t *testing.T) {
	_, err := normalizeFields("work_experiences", flatEntitySpecs["work_experiences"], map[string]any{
		"company": "Acme",
	}, true)
	if err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("expected a missing-title error, got: %v", err)
	}

	// Present-but-blank counts as missing.
	_, err = normalizeFields("languages", flatEntitySpecs["languages"], map[string]any{"name": "   "}, true)
	if err == nil {
		t.Fatal("expected blank required field to be rejected")
	}
}

// A patch legitimately carries only what changes, so required/defaults must
// not apply on update.
func TestNormalizeFieldsUpdateSkipsRequiredAndDefaults(t *testing.T) {
	out, err := normalizeFields("work_experiences", flatEntitySpecs["work_experiences"], map[string]any{
		"description": "New bullets.",
	}, false)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	if out["content"] != "New bullets." {
		t.Errorf("out = %#v", out)
	}
	if _, ok := out["is_current"]; ok {
		t.Error("update must not inject defaults")
	}

	if _, err := normalizeFields("languages", flatEntitySpecs["languages"], map[string]any{}, false); err == nil {
		t.Error("an empty patch should be rejected")
	}
}

func TestNormalizeFieldsDropsServerOwnedFields(t *testing.T) {
	out, err := normalizeFields("languages", flatEntitySpecs["languages"], map[string]any{
		"name":       "English",
		"id":         "abc",
		"resume_id":  "def",
		"sort_order": 3,
		"sortOrder":  4,
	}, true)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	for _, k := range []string{"id", "resume_id", "sort_order", "sortOrder"} {
		if _, ok := out[k]; ok {
			t.Errorf("%q should have been dropped: %#v", k, out)
		}
	}
}

func TestNormalizeFieldsValidatesEnum(t *testing.T) {
	out, err := normalizeFields("misc_entries", flatEntitySpecs["misc_entries"], map[string]any{
		"kind":  "Award",
		"title": "Employee of the Year",
	}, true)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	if out["kind"] != "award" {
		t.Errorf("kind should normalize to lowercase, got %#v", out["kind"])
	}

	if _, err := normalizeFields("misc_entries", flatEntitySpecs["misc_entries"], map[string]any{
		"kind":  "patent",
		"title": "A patent",
	}, true); err == nil {
		t.Error("expected an invalid-kind error")
	}
}

func TestNormalizeFieldsMetaPatch(t *testing.T) {
	out, err := normalizeFields("contact info", resumeMetaSpec, map[string]any{
		"name":  "Sarah Chen",
		"title": "Full Stack Engineer",
	}, false)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	if out["full_name"] != "Sarah Chen" || out["headline"] != "Full Stack Engineer" {
		t.Errorf("out = %#v", out)
	}
}

// The exact failure from the bug report: the agent (correctly following the
// system prompt) sent start_date "2020-06", which flowed unmodified through
// the draft into flushDraft's POST and failed there because the backend
// decodes *time.Time with encoding/json's default RFC3339-only unmarshaling.
// Normalizing at propose time means the draft never holds a date the backend
// will reject.
func TestNormalizeFieldsNormalizesShorthandDatesToRFC3339(t *testing.T) {
	out, err := normalizeFields("work_experiences", flatEntitySpecs["work_experiences"], map[string]any{
		"company":    "CloudScale Inc",
		"title":      "Full-Stack Software Engineer",
		"start_date": "2020-06",
		"end_date":   "2022-02-15",
	}, true)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	if out["start_date"] != "2020-06-01T00:00:00Z" {
		t.Errorf("start_date = %#v", out["start_date"])
	}
	if out["end_date"] != "2022-02-15T00:00:00Z" {
		t.Errorf("end_date = %#v", out["end_date"])
	}
}

func TestNormalizeFieldsPassesThroughAlreadyRFC3339Dates(t *testing.T) {
	out, err := normalizeFields("work_experiences", flatEntitySpecs["work_experiences"], map[string]any{
		"company":    "Acme",
		"title":      "Engineer",
		"start_date": "2020-06-01T00:00:00Z",
	}, true)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	if out["start_date"] != "2020-06-01T00:00:00Z" {
		t.Errorf("start_date = %#v", out["start_date"])
	}
}

// A null/empty date (e.g. end_date on a still-current role) must pass
// through as "no date," not be rejected as unparseable.
func TestNormalizeFieldsTreatsEmptyDateAsNull(t *testing.T) {
	out, err := normalizeFields("work_experiences", flatEntitySpecs["work_experiences"], map[string]any{
		"company":    "Acme",
		"title":      "Engineer",
		"is_current": true,
		"end_date":   nil,
	}, true)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	if out["end_date"] != nil {
		t.Errorf("end_date = %#v, want nil", out["end_date"])
	}

	out, err = normalizeFields("work_experiences", flatEntitySpecs["work_experiences"], map[string]any{
		"company":  "Acme",
		"title":    "Engineer",
		"end_date": "",
	}, true)
	if err != nil {
		t.Fatalf("normalizeFields: %v", err)
	}
	if out["end_date"] != nil {
		t.Errorf("end_date = %#v, want nil", out["end_date"])
	}
}

// A garbage date is exactly the case that must close the retry loop: the
// model gets an error back naming accepted formats instead of the change
// silently reaching the frontend broken.
func TestNormalizeFieldsRejectsUnparseableDate(t *testing.T) {
	_, err := normalizeFields("work_experiences", flatEntitySpecs["work_experiences"], map[string]any{
		"company":    "Acme",
		"title":      "Engineer",
		"start_date": "sometime last summer",
	}, true)
	if err == nil {
		t.Fatal("expected an error for an unparseable date")
	}
	if !strings.Contains(err.Error(), "start_date") {
		t.Errorf("error should name the field, got: %v", err)
	}
	if !strings.Contains(err.Error(), "2023-01") {
		t.Errorf("error should suggest an accepted format, got: %v", err)
	}
}

func TestNormalizeFieldsDateFieldsCoverEveryEntityThatHasOne(t *testing.T) {
	want := map[string][]string{
		"work_experiences": {"start_date", "end_date"},
		"educations":       {"start_date", "end_date"},
		"projects":         {"start_date", "end_date"},
		"certifications":   {"issue_date", "expiry_date"},
		"misc_entries":     {"entry_date"},
	}
	for entity, fields := range want {
		spec := flatEntitySpecs[entity]
		for _, f := range fields {
			if !slices.Contains(spec.dateFields, f) {
				t.Errorf("%s.dateFields missing %q", entity, f)
			}
		}
	}
	if !slices.Contains(customEntrySpec.dateFields, "entry_date") {
		t.Error("customEntrySpec.dateFields missing entry_date")
	}
}

func TestSkillSpecFor(t *testing.T) {
	if got := skillSpecFor("create_item"); !got.allows("proficiency") {
		t.Error("create_item should use the skill item spec")
	}
	if got := skillSpecFor("create_group"); !got.allows("group_name") {
		t.Error("create_group should use the skill group spec")
	}
}

// The tool description the model reads is generated from the same specs that
// validate its output, so the two can't drift.
func TestDescribeFlatEntityFields(t *testing.T) {
	desc := describeFlatEntityFields()
	for name, spec := range flatEntitySpecs {
		if !strings.Contains(desc, name) {
			t.Errorf("description omits entity %q", name)
		}
		for _, f := range spec.fields {
			if !strings.Contains(desc, f) {
				t.Errorf("description omits %s.%s", name, f)
			}
		}
	}
}

// Same guarantee as TestDescribeFlatEntityFields for the two tree-shaped
// tools — a field added to a spec must show up in what the model is told,
// not just in what normalizeFields silently accepts.
func TestDescribeSkillChangeFields(t *testing.T) {
	desc := describeSkillChangeFields()
	for _, f := range skillGroupSpec.fields {
		if !strings.Contains(desc, f) {
			t.Errorf("description omits skill group field %q", f)
		}
	}
	for _, f := range skillItemSpec.fields {
		if !strings.Contains(desc, f) {
			t.Errorf("description omits skill item field %q", f)
		}
	}
}

func TestDescribeCustomSectionChangeFields(t *testing.T) {
	desc := describeCustomSectionChangeFields()
	for _, f := range customSectionSpec.fields {
		if !strings.Contains(desc, f) {
			t.Errorf("description omits custom section field %q", f)
		}
	}
	for _, f := range customEntrySpec.fields {
		if !strings.Contains(desc, f) {
			t.Errorf("description omits custom entry field %q", f)
		}
	}
}
