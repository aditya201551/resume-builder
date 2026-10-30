package design

import (
	"encoding/json"
	"reflect"
	"testing"
)

func addTestTemplate(t *testing.T, mode LayoutMode) Template {
	t.Helper()
	template, _ := FindTemplate(DefaultTemplateID())
	template.ID = "test-" + string(mode)
	template.Defaults.Layout.Mode = mode
	template.SupportedModes = []string{string(mode)}
	template.Defaults.Typography.FontFamily = "Arial"
	template.Defaults.Page.MarginLeft = 24
	template.Defaults.Header.Photo.Size = "l"
	template.Defaults.Colors.ApplyAccent.Name = true
	template.Defaults.Spacing.SectionGap = 24
	refs := []SectionRef{{SectionType: "skills", IsVisible: true}, {SectionType: "summary", IsVisible: true}}
	template.Defaults.SectionOrder.One.Sections = refs
	template.Defaults.SectionOrder.Two.Left = refs[:1]
	template.Defaults.SectionOrder.Two.Right = refs[1:]
	template.Defaults.SectionOrder.Mix.Bands = []SectionBand{{Full: refs[1:], Left: refs[:1]}}
	previous := catalog
	catalog.Templates = append(Templates(), template)
	t.Cleanup(func() { catalog = previous })
	return template
}

func TestDefaultsUseSelectedTemplateWithoutSharingMutableData(t *testing.T) {
	template := addTestTemplate(t, LayoutOne)
	d := Default(template.ID)
	if d.Typography.FontFamily != "Arial" || d.Page.MarginLeft != 24 || d.Spacing.SectionGap != 24 {
		t.Fatal("selected template defaults were not used")
	}
	d.SectionOrder.One.Sections[0].SectionType = "changed"
	listed := Templates()
	listed[len(listed)-1].Defaults.SectionOrder.One.Sections[0].SectionType = "also changed"
	if Default(template.ID).SectionOrder.One.Sections[0].SectionType != "skills" {
		t.Fatal("a caller modified the catalog's nested defaults")
	}
	if Default("retired-id").TemplateID != DefaultTemplateID() {
		t.Fatal("unknown template did not use the registered fallback")
	}
}

func TestDecodeUsesSelectedDefaultsAndPreservesExplicitOverrides(t *testing.T) {
	template := addTestTemplate(t, LayoutOne)
	d, err := Decode([]byte(`{"templateId":"` + template.ID + `","page":{"marginLeft":0},"typography":{"baseFontSizePt":16},"colors":{"applyAccent":{"name":false}},"header":{"photo":{"show":false}},"sectionOrder":{"one":{"sections":[]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Typography.FontFamily != "Arial" || d.Typography.BaseFontSizePt != 16 || d.Page.MarginLeft != 0 || d.Colors.ApplyAccent.Name {
		t.Fatal("saved values did not overlay the selected template defaults")
	}
	if d.Header.Photo.Size != "l" || d.Header.Photo.Show || len(d.SectionOrder.One.Sections) != 0 {
		t.Fatal("nested defaults or explicit empty preferences were lost")
	}
	raw, _ := json.Marshal(d)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields["typography"] == nil || fields["templateId"] == nil || fields["DesignDefaults"] != nil {
		t.Fatal("persisted design JSON must retain its existing flat format")
	}
	if _, err := Decode([]byte(`{"templateId":`)); err == nil {
		t.Fatal("invalid saved JSON accepted")
	}
}

func TestSelectTemplateAppliesDefaultsAndRetainsSectionPreferences(t *testing.T) {
	for _, mode := range []LayoutMode{LayoutOne, LayoutTwo, LayoutMix} {
		t.Run(string(mode), func(t *testing.T) {
			template := addTestTemplate(t, mode)
			current := Default(DefaultTemplateID())
			current.Typography.FontFamily = "Roboto"
			title, customID := "Toolkit", "custom-one"
			current.SectionOrder.One.Sections = []SectionRef{
				{SectionType: "skills", IsVisible: false, TitleOverride: &title},
				{SectionType: "custom", CustomSectionID: &customID, IsVisible: true},
			}
			before, _ := json.Marshal(current)
			selected, err := SelectTemplate(current, template.ID)
			if err != nil {
				t.Fatal(err)
			}
			if selected.Layout.Mode != mode || selected.Typography != template.Defaults.Typography || selected.Page != template.Defaults.Page {
				t.Fatal("template formatting and layout were not applied")
			}
			refs := activeSectionRefs(selected)
			if len(refs) != 3 {
				t.Fatalf("sections lost or duplicated: %+v", refs)
			}
			foundCustom := false
			for _, ref := range refs {
				if ref.SectionType == "skills" && (ref.IsVisible || ref.TitleOverride == nil || *ref.TitleOverride != title) {
					t.Fatal("section visibility/title was reset")
				}
				if ref.CustomSectionID != nil && *ref.CustomSectionID == customID {
					foundCustom = true
				}
			}
			if !foundCustom {
				t.Fatal("custom section was lost")
			}
			after, _ := json.Marshal(current)
			if string(before) != string(after) {
				t.Fatal("switch mutated the original design")
			}
			selected.Typography.FontFamily = "Inter"
			reselected, _ := SelectTemplate(selected, template.ID)
			if !reflect.DeepEqual(selected, reselected) {
				t.Fatal("reselecting the same template reset user edits")
			}
		})
	}
	if _, err := SelectTemplate(Default(DefaultTemplateID()), "missing"); err == nil {
		t.Fatal("unknown selection accepted")
	}
}

func TestCatalogRejectsInvalidDefaults(t *testing.T) {
	tests := map[string]func(*templateCatalog){
		"missing defaults":     func(c *templateCatalog) { c.Templates[0].Defaults = DesignDefaults{} },
		"invalid font":         func(c *templateCatalog) { c.Templates[0].Defaults.Typography.FontFamily = "unknown" },
		"unsupported layout":   func(c *templateCatalog) { c.Templates[0].Defaults.Layout.Mode = LayoutTwo },
		"unregistered default": func(c *templateCatalog) { c.DefaultTemplateID = "missing" },
		"duplicate ids":        func(c *templateCatalog) { c.Templates = append(c.Templates, c.Templates[0]) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			c := templateCatalog{DefaultTemplateID: DefaultTemplateID(), Templates: Templates()}
			mutate(&c)
			raw, _ := json.Marshal(c)
			if _, err := decodeTemplateCatalog(raw); err == nil {
				t.Fatal("invalid template catalog accepted")
			}
		})
	}
}
