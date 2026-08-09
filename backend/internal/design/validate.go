package design

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	validLayoutModes         = []string{"one", "two", "mix"}
	validPageFormats         = []string{"A4", "Letter"}
	validHeadingStyles       = []string{"line", "box", "underline", "thinLine", "simple"}
	validCapitalizations     = []string{"none", "uppercase"}
	validHeaderPhotoSizes    = []string{"s", "m", "l"}
	validAlignText           = []string{"left", "center"}
	validJobTitlePositions   = []string{"below", "inline"}
	validDateDisplayModes    = []string{"right", "left", "belowTitle"}
	validSubtitleStyles      = []string{"normal", "italic"}
	validSectionDisplayModes = []string{"grid", "text", "bullets"}
	validDateFormats         = []string{"MM/YYYY", "Mon YYYY", "YYYY"}
	validFontFamilies        = []string{
		"Georgia", "Times New Roman", "Arial", "Helvetica",
		"Inter", "Roboto", "Lora", "Merriweather", "Open Sans",
	}
	validNameFontFamilies = append([]string{"inherit"}, validFontFamilies...)
	validSectionTypes     = []string{
		"summary", "work_experience", "education", "skills", "projects",
		"certifications", "languages", "awards", "publications", "volunteer", "custom",
	}
)

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func enumError(field, got string, allowed []string) error {
	return fmt.Errorf("%s: %q is not one of %s", field, got, strings.Join(allowed, ", "))
}

func Validate(d ResumeDesign) error {
	if !contains(validLayoutModes, string(d.Layout.Mode)) {
		return enumError("layout.mode", string(d.Layout.Mode), validLayoutModes)
	}
	if !contains(validPageFormats, d.Page.Format) {
		return enumError("page.format", d.Page.Format, validPageFormats)
	}
	if !contains(validFontFamilies, d.Typography.FontFamily) {
		return enumError("typography.fontFamily", d.Typography.FontFamily, validFontFamilies)
	}
	if !contains(validNameFontFamilies, d.Typography.NameFontFamily) {
		return enumError("typography.nameFontFamily", d.Typography.NameFontFamily, validNameFontFamilies)
	}
	if d.Typography.BaseFontSizePt <= 0 || d.Typography.BaseFontSizePt > 24 {
		return fmt.Errorf("typography.baseFontSizePt: %.1f is out of range (0, 24]", d.Typography.BaseFontSizePt)
	}
	if d.Typography.HeadlineFontSizePt <= 0 || d.Typography.HeadlineFontSizePt > 40 {
		return fmt.Errorf("typography.headlineFontSizePt: %.1f is out of range (0, 40]", d.Typography.HeadlineFontSizePt)
	}
	if d.Typography.EntryHeaderFontSizePt <= 0 || d.Typography.EntryHeaderFontSizePt > 40 {
		return fmt.Errorf("typography.entryHeaderFontSizePt: %.1f is out of range (0, 40]", d.Typography.EntryHeaderFontSizePt)
	}
	if !hexColorPattern.MatchString(d.Colors.Text) {
		return fmt.Errorf("colors.text: %q is not a hex color like #14181d", d.Colors.Text)
	}
	if !hexColorPattern.MatchString(d.Colors.Accent) {
		return fmt.Errorf("colors.accent: %q is not a hex color like #14181d", d.Colors.Accent)
	}
	if !hexColorPattern.MatchString(d.Colors.Background) {
		return fmt.Errorf("colors.background: %q is not a hex color like #14181d", d.Colors.Background)
	}
	if !contains(validHeadingStyles, d.Heading.Style) {
		return enumError("heading.style", d.Heading.Style, validHeadingStyles)
	}
	if !contains(validCapitalizations, d.Heading.Capitalization) {
		return enumError("heading.capitalization", d.Heading.Capitalization, validCapitalizations)
	}
	if !contains(validHeaderPhotoSizes, d.Header.Photo.Size) {
		return enumError("header.photo.size", d.Header.Photo.Size, validHeaderPhotoSizes)
	}
	if !contains(validAlignText, d.Header.AlignText) {
		return enumError("header.alignText", d.Header.AlignText, validAlignText)
	}
	if !contains(validJobTitlePositions, d.Header.JobTitlePosition) {
		return enumError("header.jobTitlePosition", d.Header.JobTitlePosition, validJobTitlePositions)
	}
	if !contains(validDateDisplayModes, d.EntryLayout.DateDisplayMode) {
		return enumError("entryLayout.dateDisplayMode", d.EntryLayout.DateDisplayMode, validDateDisplayModes)
	}
	if !contains(validSubtitleStyles, d.EntryLayout.SubtitleStyle) {
		return enumError("entryLayout.subtitleStyle", d.EntryLayout.SubtitleStyle, validSubtitleStyles)
	}
	if !contains(validSectionDisplayModes, d.SectionDisplay.Skills) {
		return enumError("sectionDisplay.skills", d.SectionDisplay.Skills, validSectionDisplayModes)
	}
	if !contains(validSectionDisplayModes, d.SectionDisplay.Languages) {
		return enumError("sectionDisplay.languages", d.SectionDisplay.Languages, validSectionDisplayModes)
	}
	if !contains(validSectionDisplayModes, d.SectionDisplay.Certifications) {
		return enumError("sectionDisplay.certifications", d.SectionDisplay.Certifications, validSectionDisplayModes)
	}
	if !contains(validDateFormats, d.DateFormat) {
		return enumError("dateFormat", d.DateFormat, validDateFormats)
	}

	for _, ref := range d.SectionOrder.One.Sections {
		if err := validateSectionRef("sectionOrder.one.sections", ref); err != nil {
			return err
		}
	}
	for _, ref := range d.SectionOrder.Two.Left {
		if err := validateSectionRef("sectionOrder.two.left", ref); err != nil {
			return err
		}
	}
	for _, ref := range d.SectionOrder.Two.Right {
		if err := validateSectionRef("sectionOrder.two.right", ref); err != nil {
			return err
		}
	}
	for _, band := range d.SectionOrder.Mix.Bands {
		for _, ref := range band.Full {
			if err := validateSectionRef("sectionOrder.mix.bands[].full", ref); err != nil {
				return err
			}
		}
		for _, ref := range band.Left {
			if err := validateSectionRef("sectionOrder.mix.bands[].left", ref); err != nil {
				return err
			}
		}
		for _, ref := range band.Right {
			if err := validateSectionRef("sectionOrder.mix.bands[].right", ref); err != nil {
				return err
			}
		}
	}

	return nil
}

func ValidateSectionRef(ref SectionRef) error {
	return validateSectionRef("sectionRef", ref)
}

func ValidSectionTypes() []string {
	out := make([]string, len(validSectionTypes))
	copy(out, validSectionTypes)
	return out
}

func validateSectionRef(path string, ref SectionRef) error {
	if !contains(validSectionTypes, ref.SectionType) {
		return enumError(path+".sectionType", ref.SectionType, validSectionTypes)
	}
	if ref.SectionType != "custom" && ref.CustomSectionID != nil {
		return fmt.Errorf("%s: customSectionId must be null unless sectionType is \"custom\"", path)
	}
	if ref.SectionType == "custom" && ref.CustomSectionID == nil {
		return fmt.Errorf("%s: customSectionId is required when sectionType is \"custom\"", path)
	}
	return nil
}

func DescribeFields() string {
	return fmt.Sprintf(
		"layout.mode: one of %s. page.format: one of %s. typography.fontFamily: one of %s. "+
			"typography.nameFontFamily: one of %s (\"inherit\" falls back to fontFamily). "+
			"colors.{text,accent,background}: hex color strings like #14181d. "+
			"colors.applyAccent.{name,headings,dates,icons}: booleans. "+
			"heading.style: one of %s. heading.capitalization: one of %s. "+
			"header.photo.size: one of %s. header.alignText: one of %s. header.jobTitlePosition: one of %s. "+
			"entryLayout.dateDisplayMode: one of %s. entryLayout.subtitleStyle: one of %s. "+
			"sectionDisplay.{skills,languages,certifications}: one of %s. "+
			"linkStyle.{showIcon,underline,useAccentColor}: booleans. "+
			"footer.{showPageNumbers,showEmail,showName}: booleans. "+
			"dateFormat: one of %s. "+
			"sectionOrder section refs: sectionType must be one of %s; customSectionId is required "+
			"when sectionType is \"custom\" and must be null otherwise.",
		strings.Join(validLayoutModes, "/"),
		strings.Join(validPageFormats, "/"),
		strings.Join(validFontFamilies, "/"),
		strings.Join(validNameFontFamilies, "/"),
		strings.Join(validHeadingStyles, "/"),
		strings.Join(validCapitalizations, "/"),
		strings.Join(validHeaderPhotoSizes, "/"),
		strings.Join(validAlignText, "/"),
		strings.Join(validJobTitlePositions, "/"),
		strings.Join(validDateDisplayModes, "/"),
		strings.Join(validSubtitleStyles, "/"),
		strings.Join(validSectionDisplayModes, "/"),
		strings.Join(validDateFormats, "/"),
		strings.Join(validSectionTypes, "/"),
	)
}
