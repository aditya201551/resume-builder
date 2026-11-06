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
  }, {
    element: <FeatureGate feature="byok_ai" />,
    children: [{ path: '/settings/ai', element: <p>AI connections page mounted</p> }],
  }])
}

function render(flags: FeatureFlags, loading = false, unavailable = false, path = '/test-feature') {
  return renderToStaticMarkup(
    <MemoryRouter initialEntries={[path]}>
      <FeatureContext value={{ flags, loading, unavailable }}><TestRoutes /></FeatureContext>
    </MemoryRouter>,
  )
}

describe('module availability', () => {
  it('makes direct navigation to AI settings depend on BYOK, independently of Platform AI', () => {
    expect(render({ platform_ai: true, byok_ai: false }, false, false, '/settings/ai')).not.toContain('AI connections page mounted')
    expect(render({ platform_ai: false, byok_ai: true }, false, false, '/settings/ai')).toContain('AI connections page mounted')
    expect(render({ platform_ai: true, byok_ai: true }, false, false, '/settings/ai')).toContain('AI connections page mounted')
    expect(render({ byok_ai: true }, false, true, '/settings/ai')).not.toContain('AI connections page mounted')
  })
  it('does not mount module content while loading, disabled, or unavailable', () => {
    expect(render({}, true)).toContain('Checking availability')
    expect(render({ 'test.enabled': false })).not.toContain('Feature content mounted')
    expect(render({}, false, true)).not.toContain('Feature content mounted')
    // Malformed truthy values from a server/proxy must not grant availability.
    expect(render({ 'test.enabled': 'false' } as unknown as FeatureFlags)).not.toContain('Feature content mounted')
    expect(render({ 'test.enabled': true })).toContain('Feature content mounted')
  })
})
