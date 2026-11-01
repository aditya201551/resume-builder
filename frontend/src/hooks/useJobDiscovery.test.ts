import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useJobDiscovery } from './useJobDiscovery'
import { defaultJobIntent, type DiscoveryPage } from '@/lib/jobSearch'

const { query } = vi.hoisted(() => ({
  query: vi.fn((_options: {
    getNextPageParam: (page: DiscoveryPage, pages: DiscoveryPage[], cursor: string, cursors: string[]) => string | undefined
  }) => ({})),
}))
vi.mock('@tanstack/react-query', () => ({ useInfiniteQuery: query }))
vi.mock('@/hooks/useAuth', () => ({ useAuth: () => ({ user: { id: 'user' } }) }))

beforeEach(() => query.mockClear())

describe('automatic job pagination', () => {
  it('continues with a new cursor but stops if a source repeats any requested cursor', () => {
    useJobDiscovery(defaultJobIntent(), false, true)
    const { getNextPageParam } = query.mock.calls[0][0]
    const page = { jobs: [], sources: [], retry: false, nextToken: 'next' }
    expect(getNextPageParam(page, [], '', [''])).toBe('next')
    expect(getNextPageParam(page, [], 'next', ['', 'next'])).toBeUndefined()
    expect(getNextPageParam(page, [], 'later', ['', 'next', 'later'])).toBeUndefined()
    expect(getNextPageParam({ ...page, nextToken: undefined }, [], '', [''])).toBeUndefined()
  })
})
