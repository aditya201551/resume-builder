import { useEffect, useRef, useState } from 'react'
import { GlobalWorkerOptions, PDFWorker, TextLayer, getDocument, type PDFDocumentLoadingTask, type PDFDocumentProxy, type RenderTask } from 'pdfjs-dist'
import pdfWorkerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import './pdfTextLayer.css'
import type { PdfSnapshot } from '@/hooks/useResumePdf'
import type { PdfHitRegion } from '@/lib/pdf/types'

GlobalWorkerOptions.workerSrc = pdfWorkerUrl

function PdfPage({ document, pageNumber, width, height, scale, regions, selected, onSelect, onPaint }: {
  document: PDFDocumentProxy; pageNumber: number; width: number; height: number; scale: number
  regions: PdfHitRegion[]; selected?: string; onSelect?: (section: string) => void
  onPaint: () => void
}) {
  const ref = useRef<HTMLElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const textRef = useRef<HTMLDivElement>(null)
  const linksRef = useRef<HTMLDivElement>(null)
  const [visible, setVisible] = useState(false)
  const [error, setError] = useState(false)
  const onPaintRef = useRef(onPaint)
  onPaintRef.current = onPaint
  useEffect(() => {
    const observer = new IntersectionObserver(([entry]) => setVisible(entry.isIntersecting), { rootMargin: '400px' })
    if (ref.current) observer.observe(ref.current)
    return () => observer.disconnect()
  }, [])
  useEffect(() => {
    if (!visible || !canvasRef.current) return
    let cancelled = false
    let render: RenderTask | undefined
    let text: TextLayer | undefined
    setError(false)
    const canvas = canvasRef.current
    // Paint a detached canvas first so the visible page never blanks during zoom.
    const buffer = window.document.createElement('canvas')
    const run = async () => {
      const page = await document.getPage(pageNumber)
      if (cancelled) return
      const viewport = page.getViewport({ scale })
      const ratio = Math.min(window.devicePixelRatio || 1, 2)
      buffer.width = Math.ceil(viewport.width * ratio)
      buffer.height = Math.ceil(viewport.height * ratio)
      render = page.render({ canvas: buffer, viewport, transform: [ratio, 0, 0, ratio, 0, 0] })
      await render.promise
      if (cancelled) return
      canvas.width = buffer.width
      canvas.height = buffer.height
      canvas.getContext('2d')!.drawImage(buffer, 0, 0)
      canvas.dataset.pdfFingerprint = document.fingerprints[0] ?? ''
      onPaintRef.current()
      const textContainer = textRef.current!
      textContainer.replaceChildren()
      textContainer.style.setProperty('--scale-factor', String(scale))
      textContainer.style.setProperty('--total-scale-factor', String(scale))
      text = new TextLayer({ textContentSource: await page.getTextContent(), container: textContainer, viewport })
      await text.render()
      if (cancelled) return
      const links = linksRef.current!
      links.replaceChildren()
      const annotations = await page.getAnnotations()
      if (cancelled) return
      // Use explicit anchors for URI annotations; no scripting or embedded actions.
      for (const annotation of annotations) {
        if (annotation.subtype !== 'Link' || !annotation.url || !/^(https?:|mailto:|tel:)/i.test(annotation.url)) continue
        const [x1, y1] = viewport.convertToViewportPoint(annotation.rect[0], annotation.rect[1])
        const [x2, y2] = viewport.convertToViewportPoint(annotation.rect[2], annotation.rect[3])
        const a = window.document.createElement('a')
        a.href = annotation.url
        a.target = '_blank'
        a.rel = 'noopener noreferrer'
        a.ariaLabel = `Open ${annotation.url}`
        Object.assign(a.style, { position: 'absolute', left: `${Math.min(x1, x2)}px`, top: `${Math.min(y1, y2)}px`, width: `${Math.abs(x2 - x1)}px`, height: `${Math.abs(y2 - y1)}px` })
        links.append(a)
      }
    }
    void run().catch((e: unknown) => { if (!cancelled && (e as Error).name !== 'RenderingCancelledException') setError(true) })
    return () => { cancelled = true; render?.cancel(); text?.cancel() }
  }, [document, pageNumber, scale, visible])

  return <figure ref={ref} className="pdf-page" style={{ width: width * scale, height: height * scale }} aria-label={`Resume page ${pageNumber}`} onClick={(event) => {
    if ((event.target as HTMLElement).closest('a, button') || window.getSelection()?.toString()) return
    const bounds = event.currentTarget.getBoundingClientRect()
    const x = (event.clientX - bounds.left) / scale
    const y = (event.clientY - bounds.top) / scale
    const region = regions.find((r) => x >= r.x && x <= r.x + r.width && y >= r.y && y <= r.y + r.height)
    if (region) onSelect?.(region.section)
  }}>
    <canvas ref={canvasRef} className="pdf-page-canvas" aria-hidden />
    <div ref={textRef} className="textLayer" />
    <div ref={linksRef} className="pdf-link-layer" />
    {regions.map((region, index) => <button key={`${region.section}-${index}`} type="button" className="pdf-section-target" data-selected={selected === region.section} aria-label={`Edit ${region.label}`} title={`Edit ${region.label}`} style={{ left: region.x * scale, top: region.y * scale, width: region.width * scale, height: region.height * scale }} onClick={() => onSelect?.(region.section)}><span>Edit {region.label}</span></button>)}
    {error && <div className="pdf-page-error" role="alert">Could not display this page. Adjust zoom to retry.</div>}
  </figure>
}

export default function PdfPages({ snapshot, scale, selected, onSelect, onReady, onError }: { snapshot: PdfSnapshot; scale: number; selected?: string; onSelect?: (section: string) => void; onReady: (key: string) => void; onError: (message: string) => void }) {
  const [opened, setOpened] = useState<{ document: PDFDocumentProxy; snapshot: PdfSnapshot }>()
  const tasksRef = useRef(new Set<PDFDocumentLoadingTask>())
  const currentTask = useRef<PDFDocumentLoadingTask | undefined>(undefined)
  const viewerWorker = useRef<PDFWorker | undefined>(undefined)
  const onReadyRef = useRef(onReady)
  const onErrorRef = useRef(onError)
  onReadyRef.current = onReady
  onErrorRef.current = onError
  useEffect(() => {
    const tasks = tasksRef.current
    return () => {
      tasks.forEach((task) => { void task.destroy() })
      tasks.clear()
      viewerWorker.current?.destroy()
      viewerWorker.current = undefined
      currentTask.current = undefined
    }
  }, [])
  useEffect(() => {
    let cancelled = false
    const tasks = tasksRef.current
    // PDF.js transfers its input buffer. Copy so the cached/downloaded PDF stays intact.
    viewerWorker.current ??= new PDFWorker()
    const task = getDocument({ data: snapshot.result.bytes.slice(), worker: viewerWorker.current })
    tasks.add(task)
    void task.promise.then((value) => {
      if (!cancelled) {
        const previous = currentTask.current
        currentTask.current = task
        setOpened({ document: value, snapshot })
        if (previous) { tasks.delete(previous); void previous.destroy() }
      }
    }).catch(() => { if (!cancelled) onErrorRef.current('Could not display this PDF. Please retry.') })
    return () => { cancelled = true; if (currentTask.current !== task) { tasks.delete(task); void task.destroy() } }
  }, [snapshot])
  if (!opened) return <div className="pdf-loading" role="status">Opening PDF…</div>
  return <div className="pdf-page-stack">{opened.snapshot.result.pages.map((page, index) => <PdfPage key={index} document={opened.document} pageNumber={index + 1} {...page} scale={scale} regions={opened.snapshot.result.regions.filter((r) => r.page === index)} selected={selected} onSelect={onSelect} onPaint={() => onReadyRef.current(opened.snapshot.key)} />)}</div>
}
