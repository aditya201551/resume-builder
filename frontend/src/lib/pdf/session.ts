import { createRenderQueue, SupersededRenderError } from './renderQueue'
import { pdfSourceKey, type PdfSource, type PdfResult, type PdfWorkerRequest, type PdfWorkerResponse } from './types'
import { getTemplate, normalizeTemplateDesign } from '@/templates/catalog'
import type { FullResume } from '@/types/resume'

export function createPdfSession() {
  let worker: Worker | undefined
  let id = 0
  let stopped = false
  const waiting = new Map<number, { resolve: (r: PdfResult) => void; reject: (e: unknown) => void; timeout: ReturnType<typeof setTimeout> }>()
  function stopWorker(error: unknown) {
    worker?.terminate()
    worker = undefined
    waiting.forEach(({ reject, timeout }) => { clearTimeout(timeout); reject(error) })
    waiting.clear()
  }
  const queue = createRenderQueue<PdfSource, PdfResult>((input) => {
    if (stopped) return Promise.reject(new SupersededRenderError())
    if (!worker) {
      worker = new Worker(new URL('./pdf.worker.tsx', import.meta.url), { type: 'module', name: 'resume-pdf' })
      worker.onmessage = ({ data }: MessageEvent<PdfWorkerResponse>) => {
        const job = waiting.get(data.id)
        if (!job) return
        waiting.delete(data.id)
        clearTimeout(job.timeout)
        if ('error' in data) job.reject(new Error(data.error))
        else job.resolve(data.result)
      }
      worker.onerror = () => stopWorker(new Error('The PDF engine could not start. Please retry.'))
      worker.onmessageerror = () => stopWorker(new Error('The PDF engine returned an unreadable response. Please retry.'))
    }
    const requestId = ++id
    return new Promise<PdfResult>((resolve, reject) => {
      const timeout = setTimeout(() => stopWorker(new Error('PDF generation took too long. Please retry.')), 60000)
      waiting.set(requestId, { resolve, reject, timeout })
      worker!.postMessage({ id: requestId, input } satisfies PdfWorkerRequest)
    })
  })
  return {
    request: queue.request,
    dispose() { stopped = true; queue.dispose(); stopWorker(new SupersededRenderError()) },
  }
}

export function createPdfSource(data: FullResume): PdfSource {
  const design = normalizeTemplateDesign(data.design)
  const template = getTemplate(design.templateId)
  return { data: { ...data, design }, rendererKey: template.renderer_key, templateRevision: template.revision }
}

export async function generateResumePdf(data: FullResume): Promise<Blob> {
  const input = createPdfSource(data)
  const session = createPdfSession()
  try {
    const result = await session.request(pdfSourceKey(input), input, true)
    return new Blob([result.bytes], { type: 'application/pdf' })
  } finally {
    session.dispose()
  }
}

