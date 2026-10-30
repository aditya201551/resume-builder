import { useLayoutEffect, useRef, useState } from 'react'
import { FileText, Maximize2, Minus, Plus } from 'lucide-react'
import LivePreview from '@/components/editor/LivePreview'
import PreviewErrorBoundary from '@/components/editor/PreviewErrorBoundary'
import type { FullResume } from '@/types/resume'

const PAGE_WIDTH = 680

export default function ResumePreview({ data }: { data: FullResume }) {
  const viewportRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const [size, setSize] = useState({ width: PAGE_WIDTH, height: 880 })
  const [zoom, setZoom] = useState(1)
  useLayoutEffect(() => {
    const viewport = viewportRef.current
    const content = contentRef.current
    if (!viewport || !content) return
    const measure = () =>
      setSize({
        width: Math.max(1, viewport.clientWidth - 40),
        height: content.offsetHeight,
      })
    const observer = new ResizeObserver(measure)
    observer.observe(viewport)
    observer.observe(content)
    measure()
    return () => observer.disconnect()
  }, [])
  const scale = Math.min(1, size.width / PAGE_WIDTH) * zoom

  return (
    <section className="preview-panel" aria-label="Live resume preview">
      <div className="preview-panel-heading">
        <span className="inline-flex items-center gap-2">
          <FileText className="size-3.5" aria-hidden /> Live preview
        </span>
        <span className="text-[11px]">Updates as you work</span>
      </div>
      <div ref={viewportRef} className="preview-viewport">
        <div
          className="preview-size"
          style={{ width: PAGE_WIDTH * scale, height: size.height * scale }}
        >
          <div
            ref={contentRef}
            className="preview-scaled"
            style={{ width: PAGE_WIDTH, transform: `scale(${scale})` }}
          >
            <PreviewErrorBoundary resetKey={data}>
              <LivePreview data={data} />
            </PreviewErrorBoundary>
          </div>
        </div>
      </div>
      <div className="preview-toolbar" role="group" aria-label="Preview zoom">
        <button
          type="button"
          onClick={() => setZoom((value) => Math.max(0.5, value - 0.1))}
          disabled={zoom <= 0.5}
          aria-label="Zoom out"
        >
          <Minus className="size-3.5" />
        </button>
        <span className="min-w-10 text-center text-[11px] tabular-nums">
          {Math.round(scale * 100)}%
        </span>
        <button
          type="button"
          onClick={() => setZoom((value) => Math.min(1.5, value + 0.1))}
          disabled={zoom >= 1.5}
          aria-label="Zoom in"
        >
          <Plus className="size-3.5" />
        </button>
        <span className="mx-1 h-4 w-px bg-border" />
        <button
          type="button"
          onClick={() => setZoom(1)}
          aria-label="Fit preview to available space"
        >
          <Maximize2 className="size-3.5" />
        </button>
      </div>
    </section>
  )
}
