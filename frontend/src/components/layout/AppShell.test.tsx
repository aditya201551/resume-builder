import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter, useRoutes } from 'react-router'
import { FeatureContext } from '@/platform/features/context'
import AppShell from './AppShell'
import type { ReactNode } from 'react'
import type { FeatureFlags } from '@/platform/features/context'

// Render menu contents in SSR so tests can verify gated account links.
vi.mock('@/components/ui/dropdown-menu', () => ({
  DropdownMenu: ({ children }: { children: ReactNode }) => <>{children}</>,
  DropdownMenuTrigger: ({ children }: { children: ReactNode }) => <>{children}</>,
  DropdownMenuContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  DropdownMenuItem: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}))

vi.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({ user: { name: 'Test User', email: 'test@example.com' }, logout: vi.fn() }),
}))

function TestRoutes() {
  return useRoutes([{
    element: <AppShell />,
    children: [
      { path: '/resumes', element: <h1>Home content</h1> },
      { path: '/resumes/library', element: <h1>Resume library</h1> },
      { path: '/resumes/:id/edit', element: <h1>Editor content</h1> },
    ],
  }])
}

function render(path = '/resumes', flags: FeatureFlags = {}) {
  return renderToStaticMarkup(
    <MemoryRouter initialEntries={[path]}>
      <FeatureContext value={{ flags, loading: false, unavailable: false }}>
        <TestRoutes />
      </FeatureContext>
    </MemoryRouter>,
  )
}

beforeEach(() => vi.stubGlobal('localStorage', { getItem: () => null }))
afterEach(() => vi.unstubAllGlobals())

describe('application navigation', () => {
  it('only shows the AI connections account link when BYOK is enabled', () => {
    expect(render('/resumes', { platform_ai: true })).not.toContain('href="/settings/ai"')
    expect(render('/resumes', { platform_ai: true, byok_ai: true })).toContain('href="/settings/ai"')
    expect(render('/resumes', { byok_ai: true })).toContain('href="/settings/ai"')
  })
  it('keeps the page and accessible navigation available when storage is blocked', () => {
    vi.stubGlobal('localStorage', { getItem: () => { throw new Error('Storage blocked') } })
    const html = render()
    expect(html).toContain('Home content')
    expect(html).toContain('aria-label="Main navigation"')
    expect(html).toContain('aria-label="Account menu"')
    expect(html).toContain('aria-label="Collapse sidebar"')
    expect(html).toContain('Skip to content')
  })

  it('keeps navigation focused on resumes', () => {
    const html = render()
    expect(html).toContain('aria-label="Home"')
    expect(html).toContain('aria-label="My resumes"')
    expect(html).not.toContain('href="/jobs"')
  })

  it('restores the saved compact preference while keeping link names accessible', () => {
    vi.stubGlobal('localStorage', { getItem: () => 'true' })
    const html = render('/resumes/library')
    expect(html).toContain('aria-label="Expand sidebar"')
    expect(html).toContain('aria-label="My resumes" aria-current="page"')
    expect(html).toContain('Resume library')
  })

  it('starts the editor with a rail even when other pages prefer expanded navigation', () => {
    vi.stubGlobal('localStorage', { getItem: () => 'false' })
    const html = render('/resumes/example/edit')
    expect(html).toContain('aria-label="Expand sidebar"')
    expect(html).toContain('aria-label="My resumes" aria-current="location"')
    expect(html).toContain('Editor content')
  })
})
