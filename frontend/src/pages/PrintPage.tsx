import { useParams } from 'react-router'
import { useQuery } from '@tanstack/react-query'
import { apiGet } from '@/lib/http'
import type { FullResume } from '@/types/resume'

/**
 * System route only — navigated by the headless export browser (chromedp),
 * never linked from app navigation. Renders chrome-less, print-ready output.
 */
export default function PrintPage() {
  const { id } = useParams<{ id: string }>()

  const resumeQuery = useQuery({
    queryKey: ['resumes', id, 'full'],
    queryFn: () => apiGet<FullResume>(`/api/resumes/${id}/full`),
    enabled: Boolean(id),
  })

  if (resumeQuery.isLoading) return null

  return (
    <div className="bg-white p-0 text-black">
      {resumeQuery.data?.resume.full_name ?? 'Resume not found'}
    </div>
  )
}
