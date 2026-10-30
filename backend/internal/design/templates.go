package design

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
)

//go:embed templates.json
var templateCatalogJSON []byte

type TemplateCapabilities struct {
	Columns         int    `json:"columns"`
	HeaderPlacement string `json:"headerPlacement"`
	Photo           bool   `json:"photo"`
}

type Template struct {
	ID              string               `json:"id"`
	Revision        int                  `json:"revision"`
	RendererKey     string               `json:"renderer_key"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	ThumbnailURL    *string              `json:"thumbnail_url"`
	Tags            []string             `json:"tags"`
	SupportedModes  []string             `json:"supported_modes"`
	SupportedFields []string             `json:"supported_fields"`
	Capabilities    TemplateCapabilities `json:"capabilities"`
}

type templateCatalog struct {
	DefaultTemplateID string     `json:"defaultTemplateId"`
	Templates         []Template `json:"templates"`
}

// This exact catalog is also imported by the frontend. Templates are application code,
// while a resume's selected ID and shared formatting remain persisted resume data.
var catalog = func() templateCatalog {
	var c templateCatalog
	if err := json.Unmarshal(templateCatalogJSON, &c); err != nil {
		panic(fmt.Errorf("invalid embedded template catalog: %w", err))
	}
	if len(c.Templates) == 0 {
		panic("template catalog is empty")
	}
	return c
}()

func DefaultTemplateID() string { return catalog.DefaultTemplateID }

func Templates() []Template {
	out := make([]Template, len(catalog.Templates))
	for i, template := range catalog.Templates {
		template.Tags = slices.Clone(template.Tags)
		template.SupportedModes = slices.Clone(template.SupportedModes)
		template.SupportedFields = slices.Clone(template.SupportedFields)
		out[i] = template
	}
	return out
}

func FindTemplate(id string) (Template, bool) {
	for _, template := range Templates() {
		if template.ID == id {
			return template, true
		}
	}
	return Template{}, false
}

func ValidateTemplate(d ResumeDesign) error {
	template, ok := FindTemplate(d.TemplateID)
	if !ok {
		return fmt.Errorf("templateId: %q is not an available template", d.TemplateID)
	}
	if !slices.Contains(template.SupportedModes, string(d.Layout.Mode)) {
		return fmt.Errorf("layout.mode: %q is not supported by %s", d.Layout.Mode, template.Name)
	}
	return nil
}

func ValidateTemplateField(templateID, key string, value any) error {
	if err := ValidateFieldValue(key, value); err != nil {
		return err
	}
	template, ok := FindTemplate(templateID)
	if !ok {
		return fmt.Errorf("templateId: %q is not an available template", templateID)
	}
	if !slices.Contains(template.SupportedFields, key) {
		return fmt.Errorf("%s is not supported by %s", key, template.Name)
	}
	return nil
}

// Normalize old DB UUIDs and retired layouts at read time. Keep the active section
// order first, including hidden sections/custom titles, then any remaining references.
// No content or formatting is reset to a template preset.
func NormalizeTemplate(d ResumeDesign) ResumeDesign {
	template, ok := FindTemplate(d.TemplateID)
	if !ok {
		template, _ = FindTemplate(DefaultTemplateID())
	}
	d.TemplateID = template.ID
	if slices.Contains(template.SupportedModes, string(d.Layout.Mode)) {
		return d
	}
	refs := []SectionRef{}
	if d.Layout.Mode == LayoutTwo {
		refs = append(refs, d.SectionOrder.Two.Right...)
		refs = append(refs, d.SectionOrder.Two.Left...)
	} else if d.Layout.Mode == LayoutMix {
		for _, band := range d.SectionOrder.Mix.Bands {
			refs = append(refs, band.Full...)
			refs = append(refs, band.Left...)
			refs = append(refs, band.Right...)
		}
	}
	refs = append(refs, d.SectionOrder.One.Sections...)
	merged := make([]SectionRef, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		key := ref.SectionType
		if ref.CustomSectionID != nil {
			key += ":" + *ref.CustomSectionID
		}
		if !seen[key] {
			merged = append(merged, ref)
			seen[key] = true
		}
	}
	d.SectionOrder.One.Sections = merged
	d.Layout.Mode = LayoutMode(template.SupportedModes[0])
	return d
}
