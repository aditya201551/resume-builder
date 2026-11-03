import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { FeatureProvider } from './FeatureProvider'
import { useFeatures } from './useFeatures'

const { query, auth } = vi.hoisted(() => ({ query: vi.fn(), auth: vi.fn() }))
vi.mock('@tanstack/react-query', () => ({ useQuery: query }))
vi.mock('@/hooks/useAuth', () => ({ useAuth: auth }))

function Probe() {
  const { enabled } = useFeatures()
  return <p>{enabled('jobs.enabled') ? 'Jobs allowed' : 'Jobs blocked'}</p>
}

const render = () => renderToStaticMarkup(<FeatureProvider><Probe /></FeatureProvider>)

beforeEach(() => {
  query.mockClear()
  auth.mockClear()
  auth.mockReturnValue({ user: { id: 'owner' } })
  query.mockReturnValue({ data: { 'jobs.enabled': true }, isFetchedAfterMount: true, isError: false })
})

describe('backend-owned feature decisions', () => {
  it('uses a user-scoped query with active refresh and focus revalidation', () => {
    expect(render()).toContain('Jobs allowed')
    expect(query.mock.calls[0][0]).toMatchObject({
      queryKey: ['features', 'owner'], enabled: true, refetchInterval: 10_000,
      refetchOnWindowFocus: 'always', staleTime: 0, gcTime: 0, retry: false,
    })
  })

  it('rejects cached enabled decisions until this mount has fetched successfully', () => {
    query.mockReturnValue({ data: { 'jobs.enabled': true }, isFetchedAfterMount: false, isError: false })
    expect(render()).toContain('Jobs blocked')
  })

  it('discards enabled cache data when refresh fails and recovers on success', () => {
    query.mockReturnValue({ data: { 'jobs.enabled': true }, isFetchedAfterMount: true, isError: true })
    expect(render()).toContain('Jobs blocked')
    query.mockReturnValue({ data: { 'jobs.enabled': true }, isFetchedAfterMount: true, isError: false })
    expect(render()).toContain('Jobs allowed')
  })

  it('does not fetch or grant feature availability without a user', () => {
    auth.mockReturnValue({ user: null })
    expect(render()).toContain('Jobs blocked')
    expect(query.mock.calls[0][0].enabled).toBe(false)
  })
})
