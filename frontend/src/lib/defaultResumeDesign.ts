import type { ResumeDesign } from '@/types/design'

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
  heading: { style: 'simple', capitalization: 'uppercase' },
  header: { photo: { show: false, size: 'm' }, alignText: 'left', jobTitlePosition: 'below' },
  entryLayout: { dateDisplayMode: 'right', subtitleStyle: 'italic' },
  sectionDisplay: { skills: 'text', languages: 'text', certifications: 'text' },
  spacing: { sectionGap: 18, entryGap: 10, bulletGap: 4 },
  linkStyle: { showIcon: true, underline: false, useAccentColor: false },
  footer: { showPageNumbers: false, showEmail: false, showName: false },
  dateFormat: 'Mon YYYY',
}
