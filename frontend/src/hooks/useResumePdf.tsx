import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { createRenderQueue, SupersededRenderError } from '@/lib/pdf/renderQueue'
import { pdfSourceKey, type PdfSource, type PdfResult, type PdfWorkerRequest, type PdfWorkerResponse } from '@/lib/pdf/types'
import { getTemplate, normalizeTemplateDesign } from '@/templates/catalog'
import type { FullResume } from '@/types/resume'

export interface PdfSnapshot {
  key: string
  result: PdfResult
  blob: Blob
}

interface ResumePdfValue {
  snapshot?: PdfSnapshot
  updating: boolean
  error: string | null
  retry: () => void
  getLatest: () => Promise<PdfSnapshot>
}
const ResumePdfContext = createContext<ResumePdfValue | null>(null)

function createSession() {
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
      worker = new Worker(new URL('../lib/pdf/pdf.worker.tsx', import.meta.url), { type: 'module', name: 'resume-pdf' })
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

export function ResumePdfProvider({ data, children }: { data: FullResume; children: ReactNode }) {
  const input = useMemo<PdfSource>(() => {
    const design = normalizeTemplateDesign(data.design)
    const template = getTemplate(design.templateId)
    return { data: { ...data, design }, rendererKey: template.renderer_key, templateRevision: template.revision }
  }, [data])
  const key = pdfSourceKey(input)
  const latest = useRef({ input, key })
  latest.current = { input, key }
  const session = useRef<ReturnType<typeof createSession> | undefined>(undefined)
  const [snapshot, setSnapshot] = useState<PdfSnapshot>()
  const snapshotRef = useRef<PdfSnapshot | undefined>(undefined)
  const [failure, setFailure] = useState<{ key: string; message: string }>()
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    const instance = createSession()
    session.current = instance
    return () => { instance.dispose(); session.current = undefined }
  }, [])

  const publish = useCallback((renderKey: string, result: PdfResult): PdfSnapshot => {
    if (snapshotRef.current?.key === renderKey) return snapshotRef.current
    const value = { key: renderKey, result, blob: new Blob([result.bytes], { type: 'application/pdf' }) }
    if (latest.current.key === renderKey) {
      snapshotRef.current = value
      setSnapshot(value)
      setFailure(undefined)
    }
    return value
  }, [])

  useEffect(() => {
    let cancelled = false
    const instance = session.current!
    void instance.request(key, latest.current.input).then((result) => {
      if (!cancelled) publish(key, result)
    }).catch((error: unknown) => {
      if (!cancelled && !(error instanceof SupersededRenderError)) setFailure({ key, message: error instanceof Error ? error.message : 'Could not generate PDF.' })
    })
    return () => { cancelled = true }
  }, [key, attempt, publish])

  const getLatest = useCallback(async () => {
    const instance = session.current
    if (!instance) throw new Error('PDF workspace is not ready.')
    // Include edits made while Download is waiting, without waiting for server sync.
    for (;;) {
      const target = latest.current
      try {
        const result = await instance.request(target.key, target.input, true)
        if (target.key === latest.current.key) return publish(target.key, result)
      } catch (error) {
        if (!(error instanceof SupersededRenderError)) throw error
        if (session.current !== instance) throw new Error('PDF workspace closed.')
      }
    }
  }, [publish])

  const error = failure?.key === key ? failure.message : null
  const retry = () => { setFailure(undefined); setAttempt((n) => n + 1) }
  return <ResumePdfContext.Provider value={{ snapshot, updating: snapshot?.key !== key, error, retry, getLatest }}>{children}</ResumePdfContext.Provider>
}

export function useResumePdf() {
  const context = useContext(ResumePdfContext)
  if (!context) throw new Error('useResumePdf requires ResumePdfProvider')
  return context
}

