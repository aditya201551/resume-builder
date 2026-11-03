import { Navigate, Outlet } from 'react-router'
import { useFeatures } from './useFeatures'
import type { FeatureKey } from './context'

export function FeatureGate({ feature }: { feature: FeatureKey }) {
  const { enabled, loading } = useFeatures()
  if (loading) {
    return <p role="status" className="p-6 text-sm text-muted-foreground">Checking availability…</p>
  }
  return enabled(feature) ? <Outlet /> : <Navigate to="/resumes" replace />
}
