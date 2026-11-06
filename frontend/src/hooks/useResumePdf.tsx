import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { SupersededRenderError } from '@/lib/pdf/renderQueue'
import { createPdfSession, createPdfSource } from '@/lib/pdf/session'
import { pdfSourceKey, type PdfResult } from '@/lib/pdf/types'
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

export function ResumePdfProvider({ data, children }: { data: FullResume; children: ReactNode }) {
  const input = useMemo(() => createPdfSource(data), [data])
  const key = pdfSourceKey(input)
  const latest = useRef({ input, key })
  latest.current = { input, key }
  const session = useRef<ReturnType<typeof createPdfSession> | undefined>(undefined)
  const [snapshot, setSnapshot] = useState<PdfSnapshot>()
  const snapshotRef = useRef<PdfSnapshot | undefined>(undefined)
  const [failure, setFailure] = useState<{ key: string; message: string }>()
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    const instance = createPdfSession()
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

