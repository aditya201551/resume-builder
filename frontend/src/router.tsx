import { createBrowserRouter, Navigate } from 'react-router'
import ProtectedRoute from '@/components/layout/ProtectedRoute'
import AppShell from '@/components/layout/AppShell'
import LoginPage from '@/pages/LoginPage'
import DashboardPage from '@/pages/DashboardPage'
import EditorPage from '@/pages/EditorPage'
import PrintPage from '@/pages/PrintPage'
import JobsPage from '@/pages/JobsPage'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/resumes/:id/print', element: <PrintPage /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppShell />,
        children: [
          { path: '/resumes', element: <DashboardPage /> },
          { path: '/resumes/library', element: <DashboardPage libraryOnly /> },
          { path: '/jobs', element: <JobsPage /> },
          { path: '/jobs/saved', element: <JobsPage saved /> },
          { path: '/jobs/external', element: <Navigate to="/jobs" replace /> },
          { path: '/jobs/direct', element: <Navigate to="/jobs" replace /> },
        ],
      },
      { path: '/resumes/:id/edit', element: <EditorPage /> },
    ],
  },
  { path: '/', element: <Navigate to="/resumes" replace /> },
])
