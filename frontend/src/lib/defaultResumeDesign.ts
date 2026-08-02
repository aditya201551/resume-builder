import type { ResumeDesign } from '@/types/design'

// Mirrors the "classic" template's seeded default_design in
// backend/internal/db/migrations/000003_design_system.up.sql. Used as a
// fallback when a resume's draft predates the design field entirely — a
// draft cached in localStorage before this feature shipped has no `design`
// key at all, and won't get one until it's re-hydrated from the server
// (see the schema-version bump planned for the draft/sync phase). Until
// then, this keeps old drafts renderable instead of crashing.
export const defaultResumeDesign: ResumeDesign = {
  templateId: '',
  layout: { mode: 'one', columnWidths: { left: 50, right: 50 } },
  sectionOrder: { one: { sections: [] }, two: { left: [], right: [] }, mix: { bands: [] } },
  page: { format: 'Letter', marginTop: 48, marginBottom: 48, marginLeft: 56, marginRight: 56 },
  typography: {
    fontFamily: 'Georgia',
    nameFontFamily: 'inherit',
    baseFontSizePt: 13,
    lineHeight: 1.55,
    nameFontSizePt: 22,
    headlineFontSizePt: 13,
    sectionHeadingFontSizePt: 11,
    entryHeaderFontSizePt: 13,
  },
  colors: {
    text: '#14181d',
    accent: '#c9603e',
    background: '#ffffff',
    applyAccent: { name: false, headings: false, dates: false, icons: false },
  },
  // "simple" (no decoration) is what every resume already visually looks
  // like — this field was never actually rendered until it was wired into
  // ClassicTemplate, so "simple" is the correct default, not "line".
  heading: { style: 'simple', capitalization: 'uppercase' },
  header: { photo: { show: false, size: 'm' }, alignText: 'left', jobTitlePosition: 'below' },
  entryLayout: { dateDisplayMode: 'right', subtitleStyle: 'italic' },
  sectionDisplay: { skills: 'text', languages: 'text', certifications: 'text' },
  spacing: { sectionGap: 18, entryGap: 10, bulletGap: 4 },
  linkStyle: { showIcon: true, underline: false, useAccentColor: false },
  footer: { showPageNumbers: false, showEmail: false, showName: false },
  dateFormat: 'Mon YYYY',
}
