import { useState } from 'react'
import { useParams, useSearchParams } from 'react-router'
import { useQuery } from '@tanstack/react-query'
import { apiGet } from '@/lib/http'
import LivePreview from '@/components/editor/LivePreview'
import type { FullResume } from '@/types/resume'

export default function PrintPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const exportToken = searchParams.get('export_token')
  const [ready, setReady] = useState(false)

  const resumeQuery = useQuery({
    queryKey: ['resumes', id, 'export', exportToken],
    queryFn: () => apiGet<FullResume>(`/api/resumes/${id}/export/data?export_token=${encodeURIComponent(exportToken!)}`),
    enabled: Boolean(id) && Boolean(exportToken),
  })

  if (!resumeQuery.data) return null

  return (
    <div className="bg-white p-0 text-black">
      <LivePreview data={resumeQuery.data} onReady={() => setReady(true)} />
      {/* chromedp waits on this marker before calling Page.printToPDF — it
          only mounts once LivePreview's pagination has actually settled.
          visibility:hidden (not display:none) so it still has a bounding
          box — chromedp.WaitVisible checks dimensions, not the CSS
          `visibility` property, so display:none would never resolve. */}
      {ready && (
        <div
          data-print-ready="true"
          style={{ position: 'fixed', top: 0, left: 0, width: 1, height: 1, visibility: 'hidden' }}
        />
      )}
    </div>
  )
}
