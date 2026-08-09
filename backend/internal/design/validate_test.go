package design

import (
	"strings"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	d := Default("template-id")
	if err := Validate(d); err != nil {
		t.Fatalf("Default() produced an invalid design: %v", err)
	}
}

func TestValidateRejectsBadEnums(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(d *ResumeDesign)
	}{
		{"bad layout mode", func(d *ResumeDesign) { d.Layout.Mode = "three" }},
		{"bad page format", func(d *ResumeDesign) { d.Page.Format = "Legal" }},
		{"bad font family", func(d *ResumeDesign) { d.Typography.FontFamily = "Comic Sans" }},
		{"bad name font family", func(d *ResumeDesign) { d.Typography.NameFontFamily = "Comic Sans" }},
		{"bad heading style", func(d *ResumeDesign) { d.Heading.Style = "sparkles" }},
		{"bad capitalization", func(d *ResumeDesign) { d.Heading.Capitalization = "titlecase" }},
		{"bad photo size", func(d *ResumeDesign) { d.Header.Photo.Size = "xl" }},
		{"bad align text", func(d *ResumeDesign) { d.Header.AlignText = "justify" }},
		{"bad job title position", func(d *ResumeDesign) { d.Header.JobTitlePosition = "above" }},
		{"bad date display mode", func(d *ResumeDesign) { d.EntryLayout.DateDisplayMode = "diagonal" }},
		{"bad subtitle style", func(d *ResumeDesign) { d.EntryLayout.SubtitleStyle = "bold" }},
		{"bad skills display", func(d *ResumeDesign) { d.SectionDisplay.Skills = "chart" }},
		{"bad date format", func(d *ResumeDesign) { d.DateFormat = "DD-MM-YYYY" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Default("template-id")
			tt.mutate(&d)
			if err := Validate(d); err == nil {
				t.Fatalf("expected an error, got nil")
			}
		})
	}
}

func TestValidateRejectsBadColors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(d *ResumeDesign)
	}{
		{"text not hex", func(d *ResumeDesign) { d.Colors.Text = "black" }},
		{"accent short hex", func(d *ResumeDesign) { d.Colors.Accent = "#fff" }},
		{"background missing hash", func(d *ResumeDesign) { d.Colors.Background = "ffffff" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Default("template-id")
			tt.mutate(&d)
			if err := Validate(d); err == nil {
				t.Fatalf("expected an error, got nil")
			}
		})
	}
}

func TestValidateSectionRefCustomSectionIdRules(t *testing.T) {
	customID := "some-id"

	t.Run("non-custom with customSectionId is rejected", func(t *testing.T) {
		d := Default("template-id")
		d.SectionOrder.One.Sections[0].CustomSectionID = &customID
		if err := Validate(d); err == nil {
			t.Fatalf("expected an error, got nil")
		}
	})

	t.Run("custom without customSectionId is rejected", func(t *testing.T) {
		d := Default("template-id")
		d.SectionOrder.One.Sections = append(d.SectionOrder.One.Sections, SectionRef{SectionType: "custom"})
		if err := Validate(d); err == nil {
			t.Fatalf("expected an error, got nil")
		}
	})

	t.Run("custom with customSectionId is valid", func(t *testing.T) {
		d := Default("template-id")
		d.SectionOrder.One.Sections = append(d.SectionOrder.One.Sections, SectionRef{
			SectionType: "custom", CustomSectionID: &customID, IsVisible: true,
		})
		if err := Validate(d); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestValidateRejectsUnknownSectionType(t *testing.T) {
	d := Default("template-id")
	d.SectionOrder.Two.Left = append(d.SectionOrder.Two.Left, SectionRef{SectionType: "hobbies"})
	if err := Validate(d); err == nil {
		t.Fatalf("expected an error, got nil")
	}
}

func TestDescribeFieldsCoversEveryEnum(t *testing.T) {
	desc := DescribeFields()
	groups := [][]string{
		validLayoutModes, validPageFormats, validHeadingStyles, validCapitalizations,
		validHeaderPhotoSizes, validAlignText, validJobTitlePositions, validDateDisplayModes,
		validSubtitleStyles, validSectionDisplayModes, validDateFormats, validSectionTypes,
		validNameFontFamilies,
	}
	for _, group := range groups {
		for _, value := range group {
			if !strings.Contains(desc, value) {
				t.Errorf("DescribeFields() is missing enum value %q", value)
			}
		}
	}
}
