import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter, useRoutes } from 'react-router'
import { FeatureContext, type FeatureFlags } from './context'
import { FeatureGate } from './FeatureGate'

vi.mock('@/hooks/useAuth', () => ({ useAuth: () => ({ user: null }) }))

function TestRoutes() {
  return useRoutes([{
    element: <FeatureGate feature="test.enabled" />,
    children: [{ path: '/test-feature', element: <p>Feature content mounted</p> }],
  }])
}

function render(flags: FeatureFlags, loading = false, unavailable = false) {
  return renderToStaticMarkup(
    <MemoryRouter initialEntries={['/test-feature']}>
      <FeatureContext value={{ flags, loading, unavailable }}><TestRoutes /></FeatureContext>
    </MemoryRouter>,
  )
}

describe('module availability', () => {
  it('does not mount module content while loading, disabled, or unavailable', () => {
    expect(render({}, true)).toContain('Checking availability')
    expect(render({ 'test.enabled': false })).not.toContain('Feature content mounted')
    expect(render({}, false, true)).not.toContain('Feature content mounted')
    // Malformed truthy values from a server/proxy must not grant availability.
    expect(render({ 'test.enabled': 'false' } as unknown as FeatureFlags)).not.toContain('Feature content mounted')
    expect(render({ 'test.enabled': true })).toContain('Feature content mounted')
  })
})
