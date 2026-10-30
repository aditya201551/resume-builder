/// <reference lib="webworker" />
import { init, renderDocumentWithLayout } from '@formepdf/core/worker'
import type { FontRegistration } from '@formepdf/react'
import wasmUrl from '@formepdf/core/pkg-web/forme_bg.wasm?url'
import { ResumeDocument } from './ResumeDocument'
import MarkdownIt from 'markdown-it'
import { resolveResumeContentData } from '@/lib/resumeContent'
import { PDF_FONT_FILES, type PdfInput, type PdfSource, type PdfResult, type PdfWorkerRequest, type PdfWorkerResponse } from './types'
import { pdfHitRegions } from './regions'

const markdown = new MarkdownIt({ html: false, linkify: true })

const fonts = new Map<string, Promise<Uint8Array>>()
async function loadFonts(input: PdfInput): Promise<FontRegistration[]> {
  const { fontFamily, nameFontFamily } = input.design.typography
  const families = new Set([fontFamily, nameFontFamily === 'inherit' ? fontFamily : nameFontFamily])
  return Promise.all([...families].flatMap((family) => [400, 700].flatMap((weight) => ['normal', 'italic'].map(async (style) => {
    const file = PDF_FONT_FILES[family]
    if (!file) throw new Error(`Unsupported PDF font: ${family}`)
    // Public assets are rooted at Vite's base URL, independent of the worker asset's path.
    const asset = new URL(`${import.meta.env.BASE_URL}fonts/pdf/${file}-${weight}-${style}.ttf`, self.location.origin).href
    if (!fonts.has(asset)) fonts.set(asset, fetch(asset).then(async (response) => {
      if (!response.ok) throw new Error(`Could not load ${family}. Check your connection and retry.`)
      return new Uint8Array(await response.arrayBuffer())
    }).catch((error) => { fonts.delete(asset); throw error }))
    return { family, fontWeight: weight, fontStyle: style as 'normal' | 'italic', src: await fonts.get(asset)! }
  }))))
}

export async function renderResumePdf(source: PdfSource): Promise<PdfResult> {
  const start = performance.now()
  const input: PdfInput = { content: resolveResumeContentData(source.data, (value) => markdown.render(value)), design: source.data.design, rendererKey: source.rendererKey }
  const [, loadedFonts] = await Promise.all([init(wasmUrl), loadFonts(input)])
  const result = await renderDocumentWithLayout(<ResumeDocument input={input} fonts={loadedFonts} />)
  const regions = pdfHitRegions(result.layout, input.content)
  return { bytes: new Uint8Array(result.pdf), pages: result.layout.pages.map(({ width, height }) => ({ width, height })), regions, warnings: result.warnings, durationMs: performance.now() - start }
}

self.addEventListener('message', async ({ data }: MessageEvent<PdfWorkerRequest>) => {
  try {
    const result = await renderResumePdf(data.input)
    self.postMessage({ id: data.id, result } satisfies PdfWorkerResponse, [result.bytes.buffer])
  } catch (error) {
    self.postMessage({ id: data.id, error: error instanceof Error ? error.message : 'Could not generate this PDF.' } satisfies PdfWorkerResponse)
  }
})
