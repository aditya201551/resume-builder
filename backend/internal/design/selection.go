package design

import "fmt"

func sectionRefKey(ref SectionRef) string {
	if ref.SectionType == "custom" && ref.CustomSectionID != nil {
		return "custom:" + *ref.CustomSectionID
	}
	return ref.SectionType
}

func activeSectionRefs(d ResumeDesign) []SectionRef {
	refs := []SectionRef{}
	switch d.Layout.Mode {
	case LayoutTwo:
		refs = append(refs, d.SectionOrder.Two.Right...)
		refs = append(refs, d.SectionOrder.Two.Left...)
	case LayoutMix:
		for _, band := range d.SectionOrder.Mix.Bands {
			refs = append(refs, band.Full...)
			refs = append(refs, band.Left...)
			refs = append(refs, band.Right...)
		}
	default:
		refs = append(refs, d.SectionOrder.One.Sections...)
	}
	return cloneSectionRefs(refs)
}

// SelectTemplate applies a different template's starting design while carrying
// over section visibility, titles, and custom sections. Reselecting is a no-op.
func SelectTemplate(current ResumeDesign, templateID string) (ResumeDesign, error) {
	if _, ok := FindTemplate(templateID); !ok {
		return ResumeDesign{}, fmt.Errorf("unknown template: %s", templateID)
	}
	if current.TemplateID == templateID {
		return NormalizeTemplate(current), nil
	}
	selected := Default(templateID)
	refs := activeSectionRefs(current)
	saved := map[string]SectionRef{}
	for _, ref := range refs {
		if _, exists := saved[sectionRefKey(ref)]; !exists {
			saved[sectionRefKey(ref)] = ref
		}
	}
	seen := map[string]bool{}
	apply := func(defaults []SectionRef) []SectionRef {
		for i, ref := range defaults {
			key := sectionRefKey(ref)
			if preference, exists := saved[key]; exists {
				defaults[i] = preference
			}
			seen[key] = true
		}
		return defaults
	}
	order := &selected.SectionOrder
	switch selected.Layout.Mode {
	case LayoutOne:
		order.One.Sections = apply(order.One.Sections)
	case LayoutTwo:
		order.Two.Left = apply(order.Two.Left)
		order.Two.Right = apply(order.Two.Right)
	case LayoutMix:
		for i := range order.Mix.Bands {
			band := &order.Mix.Bands[i]
			band.Full, band.Left, band.Right = apply(band.Full), apply(band.Left), apply(band.Right)
		}
	}
	remaining := []SectionRef{}
	for _, ref := range refs {
		key := sectionRefKey(ref)
		if !seen[key] {
			remaining = append(remaining, ref)
			seen[key] = true
		}
	}
	switch selected.Layout.Mode {
	case LayoutOne:
		order.One.Sections = append(order.One.Sections, remaining...)
	case LayoutTwo:
		order.Two.Right = append(order.Two.Right, remaining...)
	case LayoutMix:
		if len(remaining) > 0 {
			order.Mix.Bands = append(order.Mix.Bands, SectionBand{Full: remaining})
		}
	}
	return selected, nil
}
