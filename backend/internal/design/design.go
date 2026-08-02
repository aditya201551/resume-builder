// Package design models a resume's visual presentation — template choice,
// layout, section order/visibility/titles, typography, colors, and spacing —
// completely independently of internal/service's content types. Nothing in
// this package reads or writes resume content; it only validates and
// describes the shape of a ResumeDesign document, so the same Validate/
// Describe pair can back both the REST PUT handler today and a future
// agent propose_design_update tool without rework (mirrors the
// entitySpec/normalizeFields/describeFlatEntityFields pattern in
// internal/agent/fields.go).
package design

type LayoutMode string

const (
	LayoutOne LayoutMode = "one"
	LayoutTwo LayoutMode = "two"
	LayoutMix LayoutMode = "mix"
)

// SectionRef is one entry in a section order list — a pointer to a content
// section (by type, and by custom_section_id for custom sections) plus the
// presentation state that used to live in resume_section_configs.
type SectionRef struct {
	SectionType     string  `json:"sectionType"`
	CustomSectionID *string `json:"customSectionId"`
	IsVisible       bool    `json:"isVisible"`
	TitleOverride   *string `json:"titleOverride"`
}

// SectionBand is one row of a "mix" layout: either a full-width band or a
// two-column split band. Exactly one of Full or {Left,Right} is populated.
type SectionBand struct {
	Full  []SectionRef `json:"full,omitempty"`
	Left  []SectionRef `json:"left,omitempty"`
	Right []SectionRef `json:"right,omitempty"`
}

// SectionOrder holds one ordering per layout mode rather than a single
// resume-level list, since a sidebar layout needs independent left/right
// order while a single-column layout needs one flat order — switching
// layout.mode doesn't lose the other modes' arrangements.
type SectionOrder struct {
	One struct {
		Sections []SectionRef `json:"sections"`
	} `json:"one"`
	Two struct {
		Left  []SectionRef `json:"left"`
		Right []SectionRef `json:"right"`
	} `json:"two"`
	Mix struct {
		Bands []SectionBand `json:"bands"`
	} `json:"mix"`
}

type Layout struct {
	Mode         LayoutMode `json:"mode"`
	ColumnWidths struct {
		Left  int `json:"left"`
		Right int `json:"right"`
	} `json:"columnWidths"`
}

type Page struct {
	Format       string `json:"format"`
	MarginTop    int    `json:"marginTop"`
	MarginBottom int    `json:"marginBottom"`
	MarginLeft   int    `json:"marginLeft"`
	MarginRight  int    `json:"marginRight"`
}

type Typography struct {
	FontFamily string `json:"fontFamily"`
	// NameFontFamily is "inherit" (fall back to FontFamily) or one of
	// validFontFamilies — never empty, so the frontend never has to special-
	// case a missing value.
	NameFontFamily           string  `json:"nameFontFamily"`
	BaseFontSizePt           float64 `json:"baseFontSizePt"`
	LineHeight               float64 `json:"lineHeight"`
	NameFontSizePt           float64 `json:"nameFontSizePt"`
	HeadlineFontSizePt       float64 `json:"headlineFontSizePt"`
	SectionHeadingFontSizePt float64 `json:"sectionHeadingFontSizePt"`
	EntryHeaderFontSizePt    float64 `json:"entryHeaderFontSizePt"`
}

type ApplyAccent struct {
	Name     bool `json:"name"`
	Headings bool `json:"headings"`
	Dates    bool `json:"dates"`
	Icons    bool `json:"icons"`
}

type Colors struct {
	Text        string      `json:"text"`
	Accent      string      `json:"accent"`
	Background  string      `json:"background"`
	ApplyAccent ApplyAccent `json:"applyAccent"`
}

type Heading struct {
	Style          string `json:"style"`
	Capitalization string `json:"capitalization"`
}

type HeaderPhoto struct {
	Show bool   `json:"show"`
	Size string `json:"size"`
}

type Header struct {
	Photo            HeaderPhoto `json:"photo"`
	AlignText        string      `json:"alignText"`
	JobTitlePosition string      `json:"jobTitlePosition"`
}

type EntryLayout struct {
	DateDisplayMode string `json:"dateDisplayMode"`
	SubtitleStyle   string `json:"subtitleStyle"`
}

// SectionDisplay controls presentation for section types whose entries can
// render as more than one shape — a grid, plain text, or a bullet list.
type SectionDisplay struct {
	Skills         string `json:"skills"`
	Languages      string `json:"languages"`
	Certifications string `json:"certifications"`
}

type Spacing struct {
	SectionGap int `json:"sectionGap"`
	EntryGap   int `json:"entryGap"`
	BulletGap  int `json:"bulletGap"`
}

// LinkStyle controls how links render (entry links, header links) — named
// distinctly from "Links" to avoid colliding with the unrelated concept of a
// resume's own header links (LinkedIn/GitHub URLs), which live in content,
// not design.
type LinkStyle struct {
	ShowIcon       bool `json:"showIcon"`
	Underline      bool `json:"underline"`
	UseAccentColor bool `json:"useAccentColor"`
}

type Footer struct {
	ShowPageNumbers bool `json:"showPageNumbers"`
	ShowEmail       bool `json:"showEmail"`
	ShowName        bool `json:"showName"`
}

// ResumeDesign is the full document stored in resume_designs.design and
// returned as FullResume.Design. It never references resume content by id
// except through SectionRef.CustomSectionID, which is an opaque pointer the
// content system owns — design never creates, renames, or deletes a custom
// section, only orders/hides/retitles it.
type ResumeDesign struct {
	TemplateID     string         `json:"templateId"`
	Layout         Layout         `json:"layout"`
	SectionOrder   SectionOrder   `json:"sectionOrder"`
	Page           Page           `json:"page"`
	Typography     Typography     `json:"typography"`
	Colors         Colors         `json:"colors"`
	Heading        Heading        `json:"heading"`
	Header         Header         `json:"header"`
	EntryLayout    EntryLayout    `json:"entryLayout"`
	SectionDisplay SectionDisplay `json:"sectionDisplay"`
	Spacing        Spacing        `json:"spacing"`
	LinkStyle      LinkStyle      `json:"linkStyle"`
	Footer         Footer         `json:"footer"`
	DateFormat     string         `json:"dateFormat"`
}
