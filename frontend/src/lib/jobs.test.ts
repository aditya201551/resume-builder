import { describe, expect, it } from 'vitest'
import { applySaved, jobFeatures, jobSearchQuery, locationLabel, parseBounds, safeJobUrl, timezoneLabel } from './jobs'
import type { Job, JobSearchResult } from '@/types/job'

const job = (values: Partial<Job> = {}): Job => ({ id: 'job_a', providerId: 'a', title: 'Software Engineer', company: 'Example', source: 'ashby:example', sourceName: 'Example careers', sourceKind: 'company', sourceUrl: 'https://example.com/a', applyUrl: 'https://example.com/a', description: '', workplace: 'hybrid', locations: [{ name: 'Berlin', country: 'Germany', countryCode: 'DE', latitude: 52.5, longitude: 13.4, precision: 'city' }], remote: { scope: 'unspecified', regions: [], timezones: [] }, employmentType: 'full_time', seniority: 'unknown', fetchedAt: '2026-10-31T00:00:00Z', closed: false, saved: false, ...values })

describe('job map data boundaries', () => {
  it('never creates office pins for remote, unmapped or closed jobs', () => {
    expect(jobFeatures([job(), job({ workplace: 'remote' }), job({ locations: [] }), job({ closed: true })]).features).toHaveLength(1)
    expect(jobFeatures([job()], [0, 0, 10, 10]).features).toHaveLength(0)
  })
  it('represents worldwide and unknown remote restrictions differently', () => {
    expect(locationLabel(job({ workplace: 'remote' }))).toContain('unspecified')
    expect(locationLabel(job({ workplace: 'remote', remote: { scope: 'worldwide', regions: [], timezones: [] } }))).toBe('Remote worldwide')
  })
  it('keeps presentation and selection outside the search query, with independent remote country', () => {
    const params = new URLSearchParams('q=backend&country=DE&remoteCountry=IN&job=a&view=map&lat=52&place=Berlin')
    const query = new URLSearchParams(jobSearchQuery(params))
    expect(query.get('country')).toBe('DE')
    expect(query.get('remoteCountry')).toBe('IN')
    expect(query.has('job')).toBe(false)
    expect(query.has('view')).toBe(false)
  })
  it('validates viewport coordinates and supports the dateline', () => {
    expect(parseBounds('170,-10,-170,10')).toEqual([170, -10, -170, 10])
    for (const value of ['NaN,0,1,2', '0,2,1,0', '0,0,181,2', '']) expect(parseBounds(value)).toBeUndefined()
  })
  it('allows only safe outbound links', () => {
    expect(safeJobUrl('javascript:alert(1)')).toBeUndefined()
    expect(safeJobUrl('https://user:pass@example.com')).toBeUndefined()
    expect(safeJobUrl('https://example.com/jobs/a')).toBe('https://example.com/jobs/a')
  })
  it('displays half-hour and quarter-hour timezone restrictions accurately', () => {
    expect(timezoneLabel(5.5)).toBe('UTC+05:30')
    expect(timezoneLabel(-3.5)).toBe('UTC-03:30')
    expect(timezoneLabel(5.75)).toBe('UTC+05:45')
  })
  it('updates only the matching saved state without mutating cached results', () => {
    const before = { jobs: [job(), job({ id: 'job_b' })], total: 2 } as JobSearchResult
    const after = applySaved(before, 'job_a', true)
    expect(after.jobs[0].saved).toBe(true)
    expect(after.jobs[1].saved).toBe(false)
    expect(before.jobs[0].saved).toBe(false)
  })
})
