package design

type LayoutMode string

const (
	LayoutOne LayoutMode = "one"
	LayoutTwo LayoutMode = "two"
	LayoutMix LayoutMode = "mix"
)

type SectionRef struct {
	SectionType     string  `json:"sectionType"`
	CustomSectionID *string `json:"customSectionId"`
	IsVisible       bool    `json:"isVisible"`
	TitleOverride   *string `json:"titleOverride"`
}

type SectionBand struct {
	Full  []SectionRef `json:"full,omitempty"`
	Left  []SectionRef `json:"left,omitempty"`
	Right []SectionRef `json:"right,omitempty"`
}

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
	FontFamily               string  `json:"fontFamily"`
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
