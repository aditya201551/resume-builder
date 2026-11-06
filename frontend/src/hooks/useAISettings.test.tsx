import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { useAISettings, type AISettings } from './useAISettings'

const { query, features } = vi.hoisted(() => ({ query: vi.fn(), features: { platform: true, byok: true, public: false } }))
vi.mock('@tanstack/react-query', () => ({ useQuery: query }))
vi.mock('@/hooks/useAuth', () => ({ useAuth: () => ({ user: { id: 'owner' } }) }))
vi.mock('@/platform/features/useFeatures', () => ({ useFeatures: () => ({ loading: false, enabled: (key: string) => key === 'platform_ai' ? features.platform : key === 'byok_ai' ? features.byok : features.public }) }))
const settings: AISettings = { platform_available: true, byok_available: true, connections: [], source: 'byok', provider: 'anthropic', model: 'model', ready: true, pdf_supported: false, reason: '' }
function Probe() {
  const ai = useAISettings()
  return <p>{JSON.stringify({ available: ai.available, ready: ai.ready, pdf: ai.pdfReady, platform: ai.platformAvailable, byok: ai.byokAvailable })}</p>
}
function state(data = settings, error = false) { query.mockReturnValue({ data, isError: error, isFetchedAfterMount: true, isLoading: false }); return renderToStaticMarkup(<Probe />) }
beforeEach(() => { query.mockReset(); features.platform = true; features.byok = true; features.public = false })
describe('AI mode availability', () => {
  it('refreshes account availability when public platform policy changes', () => {
    state()
    expect(query.mock.calls.at(-1)?.[0].queryKey).toEqual(['ai-settings', 'owner', true, true, false])
    features.public = true
    state()
    expect(query.mock.calls.at(-1)?.[0].queryKey).toEqual(['ai-settings', 'owner', true, true, true])
  })
  it('does not fall back to platform when selected BYOK is disabled', () => {
    features.byok = false
    expect(state()).toContain('&quot;available&quot;:true,&quot;ready&quot;:false')
  })
  it('keeps BYOK independent of platform eligibility', () => {
    features.platform = false
    expect(state({ ...settings, platform_available: false })).toContain('&quot;ready&quot;:true')
  })
  it('blocks cached enabled settings on refresh failure or disabled flags', () => {
    expect(state(settings, true)).toContain('&quot;available&quot;:false,&quot;ready&quot;:false')
    features.platform = false; features.byok = false
    expect(state()).toContain('&quot;available&quot;:false,&quot;ready&quot;:false')
  })
  it('requires native PDF capability and a selected mode', () => {
    expect(state()).toContain('&quot;pdf&quot;:false')
    expect(state({ ...settings, pdf_supported: true })).toContain('&quot;pdf&quot;:true')
    expect(state({ ...settings, source: '' })).toContain('&quot;ready&quot;:false')
  })
})
