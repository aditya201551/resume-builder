import type { ResumeContent } from '@/templates/contract'
import type { ResumeDesign } from '@/types/design'
import type { FullResume } from '@/types/resume'

export interface PdfSource {
  data: FullResume
  rendererKey: string
  templateRevision: number
}

export function pdfSourceKey({ data, rendererKey, templateRevision }: PdfSource): string {
  const { full_name, headline, email, phone, location, links, summary } = data.resume
  // Labels and server timestamps cannot change the document. Sync acknowledgements
  // should not cause another render of the same PDF.
  return JSON.stringify({ data: { ...data, resume: { full_name, headline, email, phone, location, links, summary } }, rendererKey, templateRevision })
}

export interface PdfInput {
  content: ResumeContent
  design: ResumeDesign
  rendererKey: string
}

export interface PdfHitRegion {
  page: number
  section: string
  label: string
  x: number
  y: number
  width: number
  height: number
}

export interface PdfResult {
  bytes: Uint8Array<ArrayBuffer>
  pages: { width: number; height: number }[]
  regions: PdfHitRegion[]
  warnings: string[]
  durationMs: number
}

export type PdfWorkerRequest = { id: number; input: PdfSource }
export type PdfWorkerResponse = { id: number; result: PdfResult } | { id: number; error: string }

// Existing design values are CSS pixels. PDF coordinates are physical points.
export const pxToPt = (value: number) => value * 72 / 96
export const PDF_PAGE_SIZES = {
  Letter: { width: 612, height: 792 },
  A4: { width: 595.276, height: 841.89 },
} as const

export const PDF_FONT_FILES: Record<string, string> = {
  Georgia: 'gelasio',
  'Times New Roman': 'tinos',
  Arial: 'arimo',
  Helvetica: 'arimo',
  Inter: 'inter',
  Roboto: 'roboto',
  Lora: 'lora',
  Merriweather: 'merriweather',
  'Open Sans': 'open-sans',
}

export function safePdfLink(url: string): string | undefined {
  const candidate = url.trim()
  if (!candidate) return undefined
  if (/^(https?:|mailto:|tel:)/i.test(candidate)) return candidate
  if (/^[\w+.-]+:/.test(candidate) || candidate.startsWith('//')) return undefined
  return `https://${candidate}`
}
