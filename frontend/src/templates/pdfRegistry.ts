import type { ComponentType } from 'react'
import type { ResumeContent } from './contract'
import type { ResumeDesign } from '@/types/design'
import VerticalPdfTemplate from './vertical/VerticalPdfTemplate'

export interface PdfTemplateProps {
  content: ResumeContent
  design: ResumeDesign
}

export const pdfTemplateRegistry: Readonly<Record<string, ComponentType<PdfTemplateProps>>> = {
  vertical: VerticalPdfTemplate,
}

export function resolvePdfTemplate(rendererKey: string) {
  const component = pdfTemplateRegistry[rendererKey]
  if (!component) throw new Error(`PDF renderer is not registered: ${rendererKey}`)
  return component
}
