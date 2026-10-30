import { Document, type FontRegistration } from '@formepdf/react'
import type { PdfInput } from './types'
import { resolvePdfTemplate } from '@/templates/pdfRegistry'

export function ResumeDocument({ input, fonts }: { input: PdfInput; fonts: FontRegistration[] }) {
  const TemplatePage = resolvePdfTemplate(input.rendererKey)
  return <Document title={input.content.header.fullName + ' - Resume'} author={input.content.header.fullName} lang="en" tagged fonts={fonts}>
    <TemplatePage content={input.content} design={input.design} />
  </Document>
}
