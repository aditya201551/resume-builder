import { createBrowserRouter, Navigate } from 'react-router'
import ProtectedRoute from '@/components/layout/ProtectedRoute'
import AppShell from '@/components/layout/AppShell'
import LoginPage from '@/pages/LoginPage'
import DashboardPage from '@/pages/DashboardPage'
import EditorPage from '@/pages/EditorPage'
import PrintPage from '@/pages/PrintPage'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/resumes/:id/print', element: <PrintPage /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppShell />,
        children: [{ path: '/resumes', element: <DashboardPage /> }],
      },
      { path: '/resumes/:id/edit', element: <EditorPage /> },
    ],
  },
  { path: '/', element: <Navigate to="/resumes" replace /> },
])
