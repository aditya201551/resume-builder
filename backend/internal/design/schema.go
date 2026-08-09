package design

import "fmt"

type FieldType string

const (
	FieldEnum   FieldType = "enum"
	FieldNumber FieldType = "number"
	FieldColor  FieldType = "color"
	FieldBool   FieldType = "bool"
)

type FieldDef struct {
	Key     string    `json:"key"`
	Label   string    `json:"label"`
	Type    FieldType `json:"type"`
	Options []string  `json:"options,omitempty"`
	Min     *float64  `json:"min,omitempty"`
	Max     *float64  `json:"max,omitempty"`
	Step    *float64  `json:"step,omitempty"`
}

type FieldGroup struct {
	Key    string     `json:"key"`
	Label  string     `json:"label"`
	Fields []FieldDef `json:"fields"`
}

func numberField(key, label string, min, max, step float64) FieldDef {
	return FieldDef{Key: key, Label: label, Type: FieldNumber, Min: &min, Max: &max, Step: &step}
}

func boolField(key, label string) FieldDef {
	return FieldDef{Key: key, Label: label, Type: FieldBool}
}

func fieldByKey(key string) *FieldDef {
	for _, group := range Schema() {
		for i := range group.Fields {
			if group.Fields[i].Key == key {
				return &group.Fields[i]
			}
		}
	}
	return nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	}
	return 0, false
}

func ValidateFieldValue(key string, value any) error {
	f := fieldByKey(key)
	if f == nil {
		return fmt.Errorf("%s: not a recognized design field", key)
	}
	switch f.Type {
	case FieldEnum:
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s: expected a string, got %T", key, value)
		}
		if !contains(f.Options, s) {
			return enumError(key, s, f.Options)
		}
	case FieldColor:
		s, ok := value.(string)
		if !ok || !hexColorPattern.MatchString(s) {
			return fmt.Errorf("%s: %v is not a hex color like #14181d", key, value)
		}
	case FieldBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s: expected a boolean, got %T", key, value)
		}
	case FieldNumber:
		n, ok := toFloat(value)
		if !ok {
			return fmt.Errorf("%s: expected a number, got %T", key, value)
		}
		if (f.Min != nil && n < *f.Min) || (f.Max != nil && n > *f.Max) {
			return fmt.Errorf("%s: %v is out of range [%v, %v]", key, n, *f.Min, *f.Max)
		}
	}
	return nil
}

func Schema() []FieldGroup {
	return []FieldGroup{
		{
			Key:   "layout",
			Label: "Layout",
			Fields: []FieldDef{
				{Key: "layout.mode", Label: "Columns", Type: FieldEnum, Options: validLayoutModes},
			},
		},
		{
			Key:   "fontSize",
			Label: "Font Size",
			Fields: []FieldDef{
				numberField("typography.baseFontSizePt", "Base size (px)", 8, 24, 1),
				numberField("typography.nameFontSizePt", "Name size (px)", 8, 40, 1),
				numberField("typography.headlineFontSizePt", "Professional title size (px)", 8, 40, 1),
				numberField("typography.sectionHeadingFontSizePt", "Section heading size (px)", 8, 20, 1),
				numberField("typography.entryHeaderFontSizePt", "Entry header size (px)", 8, 24, 1),
			},
		},
		{
			Key:   "spacing",
			Label: "Spacing",
			Fields: []FieldDef{
				numberField("typography.lineHeight", "Line height", 1, 2, 0.05),
				numberField("spacing.sectionGap", "Section gap (px)", 0, 48, 1),
				numberField("spacing.entryGap", "Entry gap (px)", 0, 32, 1),
				numberField("spacing.bulletGap", "Bullet gap (px)", 0, 16, 1),
				numberField("page.marginTop", "Top margin (px)", 0, 120, 1),
				numberField("page.marginBottom", "Bottom margin (px)", 0, 120, 1),
				numberField("page.marginLeft", "Left margin (px)", 0, 120, 1),
				numberField("page.marginRight", "Right margin (px)", 0, 120, 1),
			},
		},
		{
			Key:   "entries",
			Label: "Entries",
			Fields: []FieldDef{
				{Key: "entryLayout.dateDisplayMode", Label: "Date & location position", Type: FieldEnum, Options: validDateDisplayModes},
				{Key: "entryLayout.subtitleStyle", Label: "Subtitle style", Type: FieldEnum, Options: validSubtitleStyles},
				{Key: "sectionDisplay.skills", Label: "Skills display", Type: FieldEnum, Options: validSectionDisplayModes},
				{Key: "sectionDisplay.languages", Label: "Languages display", Type: FieldEnum, Options: validSectionDisplayModes},
				{Key: "dateFormat", Label: "Date format", Type: FieldEnum, Options: validDateFormats},
			},
		},
		{
			Key:   "headings",
			Label: "Section Headings",
			Fields: []FieldDef{
				{Key: "heading.style", Label: "Style", Type: FieldEnum, Options: validHeadingStyles},
				{Key: "heading.capitalization", Label: "Capitalization", Type: FieldEnum, Options: validCapitalizations},
			},
		},
		{
			Key:   "font",
			Label: "Font",
			Fields: []FieldDef{
				{Key: "typography.fontFamily", Label: "Body font", Type: FieldEnum, Options: validFontFamilies},
				{Key: "typography.nameFontFamily", Label: "Name font", Type: FieldEnum, Options: validNameFontFamilies},
			},
		},
		{
			Key:   "colors",
			Label: "Colors",
			Fields: []FieldDef{
				{Key: "colors.text", Label: "Text", Type: FieldColor},
				{Key: "colors.accent", Label: "Accent", Type: FieldColor},
				{Key: "colors.background", Label: "Background", Type: FieldColor},
				boolField("colors.applyAccent.name", "Apply accent to name"),
				boolField("colors.applyAccent.headings", "Apply accent to headings"),
				boolField("colors.applyAccent.dates", "Apply accent to dates"),
				boolField("colors.applyAccent.icons", "Apply accent to icons"),
			},
		},
		{
			Key:   "header",
			Label: "Header",
			Fields: []FieldDef{
				{Key: "header.alignText", Label: "Text alignment", Type: FieldEnum, Options: validAlignText},
				{Key: "header.jobTitlePosition", Label: "Job title position", Type: FieldEnum, Options: validJobTitlePositions},
			},
		},
		{
			Key:   "linkStyle",
			Label: "Links",
			Fields: []FieldDef{
				boolField("linkStyle.showIcon", "Show link icon"),
				boolField("linkStyle.underline", "Underline links"),
				boolField("linkStyle.useAccentColor", "Use accent color for links"),
			},
		},
		{
			Key:   "footer",
			Label: "Footer",
			Fields: []FieldDef{
				boolField("footer.showPageNumbers", "Show page numbers"),
				boolField("footer.showEmail", "Show email"),
				boolField("footer.showName", "Show name"),
			},
		},
	}
}
