package design

// defaultSectionTypeOrder mirrors LivePreview.tsx's DEFAULT_ORDER exactly —
// used only by Default() below (for tests and any future in-code fallback);
// the actual seeded default lives in the templates.default_design row
// inserted by migration 000003, which must be kept in sync with this list.
var defaultSectionTypeOrder = []string{
	"summary", "work_experience", "education", "skills", "projects",
	"certifications", "languages", "awards", "publications", "volunteer",
}

// Default returns the "Classic" template's design — the same values as the
// seeded templates.default_design row, expressed as Go so tests can assert
// against it without parsing JSON. templateID should be that row's id.
func Default(templateID string) ResumeDesign {
	sections := make([]SectionRef, 0, len(defaultSectionTypeOrder))
	for _, t := range defaultSectionTypeOrder {
		sections = append(sections, SectionRef{SectionType: t, IsVisible: true})
	}

	d := ResumeDesign{
		TemplateID: templateID,
		Page:       Page{Format: "Letter", MarginTop: 48, MarginBottom: 48, MarginLeft: 56, MarginRight: 56},
		Typography: Typography{
			FontFamily: "Georgia", BaseFontSizePt: 13, LineHeight: 1.55,
			NameFontSizePt: 22, SectionHeadingFontSizePt: 11,
		},
		Colors:  Colors{Text: "#14181d", Accent: "#c9603e", Background: "#ffffff"},
		Heading: Heading{Style: "line", Capitalization: "uppercase"},
		Header:  Header{Photo: HeaderPhoto{Show: false, Size: "m"}, AlignText: "left", JobTitlePosition: "below"},
		EntryLayout: EntryLayout{
			DateDisplayMode: "right", SubtitleStyle: "italic",
		},
		SectionDisplay: SectionDisplay{Skills: "text", Languages: "text", Certifications: "text"},
		Spacing:        Spacing{SectionGap: 18, EntryGap: 10, BulletGap: 4},
		DateFormat:     "Mon YYYY",
	}
	d.Layout.Mode = LayoutOne
	d.Layout.ColumnWidths.Left = 50
	d.Layout.ColumnWidths.Right = 50
	d.SectionOrder.One.Sections = sections
	return d
}
