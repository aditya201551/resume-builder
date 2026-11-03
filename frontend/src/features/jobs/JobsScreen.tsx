import { lazy, Suspense } from 'react'

const JobsPage = lazy(() => import('./pages/JobsPage'))

// This component is mounted only after the module's availability guard allows it.
export function JobsScreen({ saved = false }: { saved?: boolean }) {
  return (
    <Suspense fallback={<p role="status" className="p-6">Loading Jobs…</p>}>
      <JobsPage saved={saved} />
    </Suspense>
  )
}
