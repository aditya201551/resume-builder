import { createBrowserRouter, Navigate, Outlet } from 'react-router'
import ProtectedRoute from '@/components/layout/ProtectedRoute'
import AppShell from '@/components/layout/AppShell'
import LoginPage from '@/pages/LoginPage'
import DashboardPage from '@/pages/DashboardPage'
import EditorPage from '@/pages/EditorPage'
import PrintPage from '@/pages/PrintPage'
import { jobsRoutes } from '@/features/jobs'
import { FeatureProvider } from '@/platform/features/FeatureProvider'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/resumes/:id/print', element: <PrintPage /> },
  {
    element: <ProtectedRoute />,
    children: [{
      element: <FeatureProvider><Outlet /></FeatureProvider>,
      children: [
        {
          element: <AppShell />,
          children: [
            { path: '/resumes', element: <DashboardPage /> },
            { path: '/resumes/library', element: <DashboardPage libraryOnly /> },
            { path: '/resumes/:id/edit', element: <EditorPage /> },
            ...jobsRoutes,
          ],
        },
      ],
    }],
  },
  { path: '/', element: <Navigate to="/resumes" replace /> },
])
