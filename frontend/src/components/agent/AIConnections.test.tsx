import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import AISettingsPage, { AIConnectionDialog, AIConnectionSelector } from './AIConnections'
import type { AISettings } from '@/hooks/useAISettings'

const { state } = vi.hoisted(() => ({ state: {
  byok: true,
  ready: true,
  source: 'platform' as 'platform' | 'byok',
} }))

vi.mock('@tanstack/react-query', () => ({ useQueryClient: () => ({ invalidateQueries: vi.fn() }) }))
vi.mock('@/platform/features/useFeatures', () => ({ useFeatures: () => ({ enabled: () => state.byok }) }))
vi.mock('@/hooks/useAISettings', async importOriginal => {
  const actual = await importOriginal<typeof import('@/hooks/useAISettings')>()
  return { ...actual, useAISettings: () => ({
    loading: false, error: false, available: true, ready: state.ready,
    platformAvailable: true, byokAvailable: state.byok, byokEnabled: state.byok,
    label: state.source === 'platform' ? 'Platform AI' : 'Anthropic',
    settings: {
      platform_available: true, byok_available: state.byok,
      source: state.source, provider: 'anthropic',
      model: state.source === 'platform' ? 'private-platform-model' : 'personal-model',
      ready: state.ready, pdf_supported: true, reason: '', connections: [],
    } satisfies AISettings,
  }) }
})

beforeEach(() => { state.byok = true; state.ready = true; state.source = 'platform' })

describe('AI connection presentation', () => {
  it('shows only the Platform AI label in the selector', () => {
    const html = renderToStaticMarkup(<AIConnectionSelector />)
    expect(html).toContain('Platform AI')
    expect(html).not.toContain('private-platform-model')
  })

  it('hides the platform model on the connections page', () => {
    const html = renderToStaticMarkup(<AISettingsPage />)
    expect(html).toContain('Selected:')
    expect(html).toContain('Platform AI')
    expect(html).not.toContain('private-platform-model')
  })

  it('preserves model details for personal connections', () => {
    state.source = 'byok'
    expect(renderToStaticMarkup(<AIConnectionSelector />)).toContain('personal-model')
    expect(renderToStaticMarkup(<AISettingsPage />)).toContain('personal-model')
  })

  it('renders no connection selector or open settings dialog in platform-only mode', () => {
    state.byok = false
    expect(renderToStaticMarkup(<AIConnectionSelector />)).toBe('')
    expect(renderToStaticMarkup(<AIConnectionDialog open onOpenChange={() => {}} />)).toBe('')
  })

  it('offers an explicit platform action when a personal selection becomes unavailable', () => {
    state.byok = false; state.source = 'byok'; state.ready = false
    const html = renderToStaticMarkup(<AIConnectionSelector />)
    expect(html).toContain('Use Platform AI')
    expect(html).not.toContain('AI connections')
    expect(html).not.toContain('personal-model')
  })
})
