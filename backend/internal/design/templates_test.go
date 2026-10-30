package design

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestTemplateCatalog(t *testing.T) {
	list := Templates()
	if len(list) != 1 || list[0].ID != "vertical" || DefaultTemplateID() != "vertical" {
		t.Fatal("expected only the vertical template")
	}
	for _, template := range list {
		if template.Revision < 1 || template.RendererKey == "" {
			t.Fatal("missing renderer or revision")
		}
		if template.Capabilities.Columns != 1 {
			t.Fatal("vertical must have one column")
		}
		if err := Validate(Default(template.ID)); err != nil {
			t.Fatalf("invalid template defaults: %v", err)
		}
		raw, err := json.Marshal(template)
		if err != nil {
			t.Fatal(err)
		}
		var payload struct {
			Defaults ResumeDesign `json:"defaults"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Defaults.Typography.FontFamily != "Georgia" || payload.Defaults.Typography.BaseFontSizePt != 13 || payload.Defaults.Page.MarginLeft != 56 {
			t.Fatal("published catalog must include Vertical's existing formatting defaults")
		}
		seen := map[string]bool{}
		for _, key := range template.SupportedFields {
			if seen[key] {
				t.Fatalf("duplicate capability: %s", key)
			}
			seen[key] = true
			if fieldByKey(key) == nil {
				t.Fatalf("catalog field %s has no shared schema definition", key)
			}
		}
	}
	list[0].SupportedFields[0] = "changed"
	if Templates()[0].SupportedFields[0] == "changed" {
		t.Fatal("caller modified the application catalog")
	}
}

func TestNormalizeRetiredTemplate(t *testing.T) {
	d := Default("legacy-database-uuid")
	d.Layout.Mode = LayoutTwo
	d.Typography.FontFamily = "Roboto"
	d.Colors.Accent = "#123456"
	title, customID := "Toolkit", "custom-1"
	d.SectionOrder.Two.Right = []SectionRef{{SectionType: "work_experience", IsVisible: true}, {SectionType: "custom", CustomSectionID: &customID, IsVisible: true}}
	d.SectionOrder.Two.Left = []SectionRef{{SectionType: "skills", IsVisible: false, TitleOverride: &title}}
	out := NormalizeTemplate(d)
	if out.TemplateID != "vertical" || out.Layout.Mode != LayoutOne {
		t.Fatal("legacy design was not normalized")
	}
	if !reflect.DeepEqual(out.Typography, d.Typography) || out.Colors != d.Colors {
		t.Fatal("shared formatting changed")
	}
	if out.SectionOrder.One.Sections[0].SectionType != "work_experience" || out.SectionOrder.One.Sections[1].CustomSectionID == nil {
		t.Fatal("active section order or custom section lost")
	}
	if out.SectionOrder.One.Sections[2].IsVisible || *out.SectionOrder.One.Sections[2].TitleOverride != "Toolkit" {
		t.Fatal("hidden section preference lost")
	}
	if !reflect.DeepEqual(out, NormalizeTemplate(out)) {
		t.Fatal("normalization is not idempotent")
	}
	if d.Layout.Mode != LayoutTwo {
		t.Fatal("normalization mutated its input")
	}
}

func TestTemplateCapabilitiesValidation(t *testing.T) {
	d := Default(DefaultTemplateID())
	if err := ValidateTemplate(d); err != nil {
		t.Fatal(err)
	}
	d.Layout.Mode = LayoutTwo
	if err := ValidateTemplate(d); err == nil {
		t.Fatal("split layout accepted for vertical")
	}
	d.TemplateID = "missing"
	if err := ValidateTemplate(d); err == nil {
		t.Fatal("unknown template accepted")
	}
	if err := ValidateTemplateField("vertical", "colors.accent", "#123456"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTemplateField("vertical", "layout.mode", "two"); err == nil {
		t.Fatal("unsupported control accepted")
	}
}
