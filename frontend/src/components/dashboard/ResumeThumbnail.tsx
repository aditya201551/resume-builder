import { useLayoutEffect, useRef, useState } from 'react'
import { useFullResume } from '@/hooks/useResumeEditor'
import LivePreview from '@/components/editor/LivePreview'

const PREVIEW_WIDTH = 680

export default function ResumeThumbnail({ resumeId }: { resumeId: string }) {
  const { data, isLoading } = useFullResume(resumeId)
  const containerRef = useRef<HTMLDivElement>(null)
  const [scale, setScale] = useState(0)

  useLayoutEffect(() => {
    const el = containerRef.current
    if (!el) return
    const measure = () => setScale(el.clientWidth / PREVIEW_WIDTH)
    measure()
    const observer = new ResizeObserver(measure)
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  return (
    <div
      ref={containerRef}
      className="relative w-full overflow-hidden rounded-md border border-border bg-white"
      style={{ aspectRatio: '0.773' }}
    >
      {isLoading || !data || scale === 0 ? (
        <div className="absolute inset-0 animate-pulse bg-secondary" />
      ) : (
        <div
          className="pointer-events-none absolute left-0 top-0 origin-top-left"
          style={{ width: PREVIEW_WIDTH, transform: `scale(${scale})` }}
        >
          <LivePreview data={data} />
        </div>
      )}
    </div>
  )
}
