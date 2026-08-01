// Mirrors backend/internal/design/design.go field-for-field (json tags
// match). Keep these two in sync by hand — there is no shared codegen
// between the Go module and the frontend workspace.

export type LayoutMode = 'one' | 'two' | 'mix'

export interface SectionRef {
  sectionType: string
  customSectionId: string | null
  isVisible: boolean
  titleOverride: string | null
}

export interface SectionBand {
  full?: SectionRef[]
  left?: SectionRef[]
  right?: SectionRef[]
}

export interface SectionOrder {
  one: { sections: SectionRef[] }
  two: { left: SectionRef[]; right: SectionRef[] }
  mix: { bands: SectionBand[] }
}

export interface Layout {
  mode: LayoutMode
  columnWidths: { left: number; right: number }
}

export interface Page {
  format: 'A4' | 'Letter'
  marginTop: number
  marginBottom: number
  marginLeft: number
  marginRight: number
}

export interface Typography {
  fontFamily: string
  baseFontSizePt: number
  lineHeight: number
  nameFontSizePt: number
  sectionHeadingFontSizePt: number
}

export interface ApplyAccent {
  name: boolean
  headings: boolean
  dates: boolean
  icons: boolean
}

export interface Colors {
  text: string
  accent: string
  background: string
  applyAccent: ApplyAccent
}

export interface Heading {
  style: 'line' | 'box' | 'underline' | 'thinLine' | 'simple'
  capitalization: 'none' | 'uppercase'
}

export interface HeaderPhoto {
  show: boolean
  size: 's' | 'm' | 'l'
}

export interface Header {
  photo: HeaderPhoto
  alignText: 'left' | 'center'
  jobTitlePosition: 'below' | 'inline'
}

export interface EntryLayout {
  dateDisplayMode: 'right' | 'left' | 'belowTitle'
  subtitleStyle: 'normal' | 'italic'
}

export type SectionDisplayMode = 'grid' | 'text' | 'bullets'

export interface SectionDisplay {
  skills: SectionDisplayMode
  languages: SectionDisplayMode
  certifications: SectionDisplayMode
}

export interface Spacing {
  sectionGap: number
  entryGap: number
  bulletGap: number
}

// Mirrors the enum lists in backend/internal/design/validate.go — keep in
// sync by hand, same as the types above. Used to populate the design
// panel's selects so the UI can never offer a value the backend would
// then reject.
export const FONT_FAMILIES = ['Georgia', 'Times New Roman', 'Arial', 'Helvetica', 'Inter', 'Roboto', 'Lora', 'Merriweather', 'Open Sans']
export const HEADING_STYLES: Heading['style'][] = ['line', 'box', 'underline', 'thinLine', 'simple']
export const CAPITALIZATIONS: Heading['capitalization'][] = ['none', 'uppercase']
export const PAGE_FORMATS: Page['format'][] = ['A4', 'Letter']

export interface ResumeDesign {
  templateId: string
  layout: Layout
  sectionOrder: SectionOrder
  page: Page
  typography: Typography
  colors: Colors
  heading: Heading
  header: Header
  entryLayout: EntryLayout
  sectionDisplay: SectionDisplay
  spacing: Spacing
  dateFormat: 'MM/YYYY' | 'Mon YYYY' | 'YYYY'
}
