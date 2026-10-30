import { Page, View } from '@formepdf/react'
import type { PdfTemplateProps } from '../pdfRegistry'
import { createPdfTemplateParts } from '../shared/pdfParts'

// The template owns page composition. Content and common formatting are provided
// by shared parts, so a new layout does not need to duplicate entity rendering.
export default function VerticalPdfTemplate({ content, design }: PdfTemplateProps) {
  const parts = createPdfTemplateParts(content, design)
  return <Page {...parts.page}>
    <View>{parts.header}{content.sections.map(parts.section)}</View>
    {parts.footer}
  </Page>
}
