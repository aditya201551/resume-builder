import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter, useRoutes } from 'react-router'
import { FeatureContext, type FeatureFlags } from './context'
import { FeatureGate } from './FeatureGate'
import { visibleNavigation } from '@/app/navigation'
import { jobsRoutes } from '@/features/jobs'

vi.mock('@/hooks/useAuth', () => ({ useAuth: () => ({ user: null }) }))

function TestRoutes() {
  return useRoutes([{
    element: <FeatureGate feature="jobs.enabled" />,
    children: [{ path: '/jobs', element: <p>Jobs content mounted</p> }],
  }])
}

function render(flags: FeatureFlags, loading = false, unavailable = false) {
  return renderToStaticMarkup(
    <MemoryRouter initialEntries={['/jobs']}>
      <FeatureContext value={{ flags, loading, unavailable }}><TestRoutes /></FeatureContext>
    </MemoryRouter>,
  )
}

describe('module availability', () => {
  it('keeps Jobs out of navigation unless explicitly enabled', () => {
    expect(visibleNavigation(() => false).map(item => item.to)).toEqual(['/resumes', '/resumes/library'])
    expect(visibleNavigation(() => true).map(item => item.to)).toContain('/jobs')
  })

  it('does not mount module content while loading, disabled, or unavailable', () => {
    expect(render({}, true)).toContain('Checking availability')
    expect(render({ 'jobs.enabled': false })).not.toContain('Jobs content mounted')
    expect(render({}, false, true)).not.toContain('Jobs content mounted')
    // Malformed truthy values from a server/proxy must not grant availability.
    expect(render({ 'jobs.enabled': 'false' } as unknown as FeatureFlags)).not.toContain('Jobs content mounted')
    expect(render({ 'jobs.enabled': true })).toContain('Jobs content mounted')
  })

  it('places all current and legacy Jobs entry routes behind one guard', () => {
    expect(jobsRoutes).toHaveLength(1)
    const paths = jobsRoutes[0].children?.map(route => route.path)
    expect(paths).toEqual(['/jobs', '/jobs/saved', '/jobs/external', '/jobs/direct'])
    expect(jobsRoutes[0].element).toEqual(<FeatureGate feature="jobs.enabled" />)
  })
})
