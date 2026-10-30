import { lazy, Suspense, useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { FileText, Maximize2, Minus, Plus, RefreshCw, Scan } from 'lucide-react'
import { useResumePdf } from '@/hooks/useResumePdf'
import PreviewErrorBoundary from './PreviewErrorBoundary'

const PdfPages = lazy(() => import('./PdfPages'))
const SCREEN_SCALE = 96 / 72

export default function ResumePreview({ selected, onSelect }: { selected?: string; onSelect?: (section: string) => void }) {
  const { snapshot, updating, error, retry } = useResumePdf()
  const viewportRef = useRef<HTMLDivElement>(null)
  const [size, setSize] = useState({ width: 680, height: 880 })
  const [zoom, setZoom] = useState(1)
  const [fit, setFit] = useState<'width' | 'page' | null>('width')
  const [viewerError, setViewerError] = useState<string | null>(null)
  const [viewerAttempt, setViewerAttempt] = useState(0)
  const [displayedKey, setDisplayedKey] = useState<string>()
  const preparing = updating || (snapshot && displayedKey !== snapshot.key)
  const page = snapshot?.result.pages[0] ?? { width: 612, height: 792 }
  useLayoutEffect(() => {
    const viewport = viewportRef.current
    if (!viewport) return
    const measure = () => setSize({ width: Math.max(1, viewport.clientWidth - 40), height: Math.max(1, viewport.clientHeight - 80) })
    const observer = new ResizeObserver(measure)
    observer.observe(viewport)
    measure()
    return () => observer.disconnect()
  }, [])
  const scale = fit === 'width' ? Math.min(SCREEN_SCALE, size.width / page.width) : fit === 'page' ? Math.min(SCREEN_SCALE, size.width / page.width, size.height / page.height) : zoom * SCREEN_SCALE
  const adjustZoom = useCallback((delta: number) => { setZoom(Math.min(2.5, Math.max(0.25, scale / SCREEN_SCALE + delta))); setFit(null) }, [scale])
  useEffect(() => {
    const viewport = viewportRef.current
    const onWheel = (event: WheelEvent) => {
      if (!event.ctrlKey && !event.metaKey) return
      event.preventDefault()
      adjustZoom(event.deltaY > 0 ? -0.05 : 0.05)
    }
    viewport?.addEventListener('wheel', onWheel, { passive: false })
    return () => viewport?.removeEventListener('wheel', onWheel)
  }, [adjustZoom])

  return <section className="preview-panel" aria-label="Live PDF preview">
    <div className="preview-panel-heading">
      <span className="inline-flex items-center gap-2"><FileText className="size-3.5" aria-hidden /> PDF preview</span>
      <span role="status" className="inline-flex items-center gap-1.5">
        {preparing && !error && <RefreshCw className="size-3 animate-spin" aria-hidden />}
        {error ? 'Preview needs attention' : preparing ? snapshot ? 'Updating…' : 'Preparing PDF…' : `${snapshot?.result.pages.length ?? 0} ${(snapshot?.result.pages.length ?? 0) === 1 ? 'page' : 'pages'} · Matches download`}
      </span>
    </div>
    {(error || viewerError) && <div className="pdf-error" role="alert"><span>{error || viewerError}</span><button type="button" onClick={() => { setViewerError(null); setViewerAttempt((n) => n + 1); retry() }}>Retry</button></div>}
    {!!snapshot?.result.warnings.length && <div className="pdf-error" role="status"><span>Some content may not fit correctly. Review your pages and adjust spacing or margins.</span></div>}
    <div ref={viewportRef} className="preview-viewport" tabIndex={0} aria-label="PDF pages. Scroll to navigate; Control or Command plus wheel to zoom.">
      {snapshot ? <PreviewErrorBoundary resetKey={snapshot.key}>
        <Suspense fallback={<div className="pdf-loading" role="status">Opening PDF…</div>}>
          <PdfPages key={viewerAttempt} snapshot={snapshot} scale={scale} selected={selected} onSelect={onSelect} onReady={(key) => { setDisplayedKey(key); setViewerError(null) }} onError={setViewerError} />
        </Suspense>
      </PreviewErrorBoundary> : !error && <div className="pdf-loading" role="status"><div className="pdf-paper-skeleton" /><p>Preparing your resume…</p></div>}
    </div>
    <div className="preview-toolbar" role="group" aria-label="Preview zoom">
      <button type="button" onClick={() => adjustZoom(-0.1)} disabled={scale / SCREEN_SCALE <= 0.25} aria-label="Zoom out"><Minus className="size-3.5" /></button>
      <span className="min-w-10 text-center text-[11px] tabular-nums">{Math.round(scale / SCREEN_SCALE * 100)}%</span>
      <button type="button" onClick={() => adjustZoom(0.1)} disabled={scale / SCREEN_SCALE >= 2.5} aria-label="Zoom in"><Plus className="size-3.5" /></button>
      <span className="mx-1 h-4 w-px bg-border" />
      <button type="button" onClick={() => setFit('width')} aria-pressed={fit === 'width'} aria-label="Fit width" title="Fit width"><Maximize2 className="size-3.5" /></button>
      <button type="button" onClick={() => setFit('page')} aria-pressed={fit === 'page'} aria-label="Fit page" title="Fit page"><Scan className="size-3.5" /></button>
    </div>
  </section>
}
