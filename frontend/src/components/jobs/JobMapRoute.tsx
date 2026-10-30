import { lazy, Suspense } from 'react'

const Page = lazy(() => import('@/pages/JobMapPage'))

export default function JobMapRoute({ direct = false }: { direct?: boolean }) {
  return <Suspense fallback={<main className="dashboard-page" role="status">Loading {direct ? 'direct jobs' : 'job map'}…</main>}><Page direct={direct} /></Suspense>
}
