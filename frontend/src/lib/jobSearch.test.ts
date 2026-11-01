import { describe, expect, it } from 'vitest'
import { defaultJobIntent, readJobIntent, uniqueDiscoveryJobs, type DiscoveryPage } from './jobSearch'
import type { Job } from '@/types/job'

describe('job discovery state', () => {
  it('roundtrips multiple work-location alternatives without map state', () => {
    const s = { ...defaultJobIntent(), query: 'Go', branches: [{ workplace: 'remote', country: 'IN', city: '' }, { workplace: 'hybrid', country: 'DE', city: 'Berlin' }] }
    expect(readJobIntent(JSON.stringify(s))).toEqual(s)
    expect(readJobIntent('not JSON')).toEqual(defaultJobIntent())
  })
  it('rejects malformed shared searches instead of crashing the controls', () => {
    for (const changes of [{ branches: [] }, { branches: [{ workplace: 'remote', country: 'IN', city: 'Mumbai' }] }, { excludeCompanies: [42] }, { postedDays: -1 }, { version: 2 }, { employmentType: 'invented' }]) {
      expect(readJobIntent(JSON.stringify({ ...defaultJobIntent(), ...changes }))).toEqual(defaultJobIntent())
    }
  })
  it('removes duplicates across pages using application identity, not company and title', () => {
    const jobs = [
      { id: 'one', title: 'Engineer', company: 'Acme', applyUrl: 'https://example.com/jobs/1?utm_source=board' },
      { id: 'two', title: 'Engineer', company: 'Acme', applyUrl: 'https://example.com/jobs/1' },
      { id: 'three', title: 'Engineer', company: 'Acme', applyUrl: 'https://example.com/jobs/2' },
    ] as Job[]
    const pages = jobs.map(job => ({ jobs: [job], sources: [], retry: false })) as DiscoveryPage[]
    expect(uniqueDiscoveryJobs(pages).map(j => j.id)).toEqual(['one', 'three'])
  })
})
