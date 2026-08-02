// The only shape a template is allowed to depend on. Templates never import
// FullResume or anything from hooks/lib that touches the draft/sync system —
// resolveResumeContent (lib/resolveResumeContent.ts) is the single place
// that translates FullResume into this contract, so content-module changes
// (new entity fields, draft/sync internals) can never leak into template
// code, and template changes can never reach back into content state.
import type { ResumeLink } from '@/types/resume'

export interface ContentHeader {
  fullName: string
  headline: string | null
  email: string | null
  phone: string | null
  location: string | null
  links: ResumeLink[]
}

// One resume entry, flattened to the generic fields a template needs to
// lay out — templates render from these, never from WorkExperience/
// Education/etc. directly, so a new entity field doesn't require touching
// every template.
export interface ContentEntry {
  key: string
  title?: string
  subtitle?: string
  dateLabel?: string
  linkUrl?: string
  linkLabel?: string
  /** Markdown already rendered to HTML (work experience, projects). */
  bodyHtml?: string
  /** Plain description text (misc entries, custom entries). */
  description?: string
  /** Pre-joined summary line (skill group items, language list). */
  meta?: string
}

export interface ContentSection {
  /** Stable key — 'work_experience', or `custom-${customSectionId}`. */
  key: string
  type: string
  title: string
  entries: ContentEntry[]
  /** Only meaningful when design.layout.mode is "two" — which column this
   * section belongs in. Single-column templates never read this. */
  column?: 'left' | 'right'
}

export interface ResumeContent {
  header: ContentHeader
  /** Already filtered to visible sections, in display order. */
  sections: ContentSection[]
}
