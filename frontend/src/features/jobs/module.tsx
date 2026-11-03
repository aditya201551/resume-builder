import { Navigate, type RouteObject } from 'react-router'
import { BriefcaseBusiness } from 'lucide-react'
import { FeatureGate } from '@/platform/features/FeatureGate'
import { JobsScreen } from './JobsScreen'

export const jobsNavigation = {
  to: '/jobs',
  label: 'Jobs',
  icon: BriefcaseBusiness,
  end: false,
  feature: 'jobs.enabled',
} as const

export const jobsRoutes: RouteObject[] = [{
  element: <FeatureGate feature="jobs.enabled" />,
  children: [
    { path: '/jobs', element: <JobsScreen /> },
    { path: '/jobs/saved', element: <JobsScreen saved /> },
    { path: '/jobs/external', element: <Navigate to="/jobs" replace /> },
    { path: '/jobs/direct', element: <Navigate to="/jobs" replace /> },
  ],
}]
