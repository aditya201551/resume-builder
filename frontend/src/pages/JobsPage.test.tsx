import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router'
import JobsPage from './JobsPage'
import { defaultJobIntent } from '@/lib/jobSearch'

const { discovery, assistant } = vi.hoisted(() => ({ discovery: vi.fn(), assistant: vi.fn() }))
vi.mock('@/hooks/useJobDiscovery', () => ({ useJobDiscovery: discovery }))
vi.mock('@/hooks/useJobAssistant', () => ({ useJobAssistant: assistant }))
vi.mock('@/hooks/useJobs', () => ({
  useJobCountries: () => ({ data: [{ code: 'IN', name: 'India' }], isError: false, refetch: vi.fn() }),
  useSaveJob: () => ({ isPending: false, mutate: vi.fn() }),
}))
vi.mock('@/components/jobs/JobDetails', () => ({ default: () => null }))

function render(path = '/jobs', saved = false) {
  return renderToStaticMarkup(<MemoryRouter initialEntries={[path]}><JobsPage saved={saved} /></MemoryRouter>)
}

beforeEach(() => {
  discovery.mockReturnValue({ data: undefined, isFetching: false, isError: false, hasNextPage: false, refetch: vi.fn() })
  assistant.mockReturnValue({ messages: [], thinking: false, error: null, send: vi.fn(), cancel: vi.fn(), clear: vi.fn(), retry: vi.fn() })
})

describe('dedicated job search surfaces', () => {
  it('opens Discover in the assistant pane with manual controls in a separate hidden pane', () => {
    const html = render()
    expect(html).toContain('aria-label="Message the job search assistant"')
    expect(html).toMatch(/id="jobs-manual-pane" hidden=""/)
    expect(html).not.toMatch(/id="jobs-assistant-pane" hidden=""/)
    expect(html).toContain('aria-label="Job search results"')
  })

  it('opens explicit manual links and Saved in manual mode', () => {
    for (const html of [render('/jobs?mode=manual'), render('/jobs/saved', true)]) {
      expect(html).toMatch(/id="jobs-assistant-pane" hidden=""/)
      expect(html).not.toMatch(/id="jobs-manual-pane" hidden=""/)
    }
  })

  it('keeps the same interpreted criteria visible with either input mode', () => {
    const intent = { ...defaultJobIntent(), query: 'Go', branches: [{ workplace: 'remote', country: 'IN', city: '' }] }
    for (const mode of ['assistant', 'manual']) {
      const params = new URLSearchParams({ mode, intent: JSON.stringify(intent) })
      const html = render(`/jobs?${params}`)
      expect(html).toContain('Remote · work from India')
      expect(html).toContain('jobs-summary-keywords">Go')
      expect(discovery).toHaveBeenLastCalledWith(intent, false, true)
    }
  })

  it('offers retry and manual search when the assistant fails', () => {
    assistant.mockReturnValue({ messages: [{ id: 1, role: 'user', text: 'Find Go jobs' }], thinking: false, error: 'Request failed; criteria unchanged.', send: vi.fn(), cancel: vi.fn(), clear: vi.fn(), retry: vi.fn() })
    const html = render()
    expect(html).toContain('role="alert"')
    expect(html).toContain('Retry request')
    expect(html).toContain('Manual search')
    expect(html).not.toContain('Search criteria updated')
  })

  it('shows automatic pagination status instead of a load-more button', () => {
    discovery.mockReturnValue({ data: { pages: [{ jobs: [], sources: [], nextToken: 'next', retry: false }] }, hasNextPage: true, isFetching: true, isFetchingNextPage: true })
    const html = render()
    expect(html).toContain('Loading more jobs…')
    expect(html).not.toContain('Load more jobs')
  })

  it('keeps a source retry available without claiming the results are exhausted', () => {
    discovery.mockReturnValue({ data: { pages: [{ jobs: [], sources: [], retry: true }] }, hasNextPage: false, isFetching: false })
    const html = render()
    expect(html).toContain('Retry sources')
    expect(html).not.toContain('reached the end')
  })

  it('announces the end when no further source pages are available', () => {
    discovery.mockReturnValue({ data: { pages: [{ jobs: [], sources: [], retry: false }] }, hasNextPage: false, isFetching: false })
    expect(render()).toContain('reached the end of the available results.')
  })
})
