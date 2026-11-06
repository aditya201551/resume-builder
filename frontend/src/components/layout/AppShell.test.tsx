import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter, useRoutes } from 'react-router'
import { FeatureContext } from '@/platform/features/context'
import AppShell from './AppShell'

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

function render(path = '/resumes', jobs = false) {
  return renderToStaticMarkup(
    <MemoryRouter initialEntries={[path]}>
      <FeatureContext value={{ flags: { 'jobs.enabled': jobs }, loading: false, unavailable: false }}>
        <TestRoutes />
      </FeatureContext>
    </MemoryRouter>,
  )
}

beforeEach(() => vi.stubGlobal('localStorage', { getItem: () => null }))
afterEach(() => vi.unstubAllGlobals())

describe('application navigation', () => {
  it('keeps the page and accessible navigation available when storage is blocked', () => {
    vi.stubGlobal('localStorage', { getItem: () => { throw new Error('Storage blocked') } })
    const html = render()
    expect(html).toContain('Home content')
    expect(html).toContain('aria-label="Main navigation"')
    expect(html).toContain('aria-label="Account menu"')
    expect(html).toContain('aria-label="Collapse sidebar"')
    expect(html).toContain('Skip to content')
  })

  it('shows Jobs only when its runtime feature flag is enabled', () => {
    expect(render()).not.toContain('href="/jobs"')
    expect(render('/resumes', true)).toContain('href="/jobs"')
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
    const html = render('/resumes/example/edit', true)
    expect(html).toContain('aria-label="Expand sidebar"')
    expect(html).toContain('aria-label="My resumes" aria-current="location"')
    expect(html).toContain('href="/jobs"')
    expect(html).toContain('Editor content')
  })
})
