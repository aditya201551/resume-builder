import { describe, expect, it } from 'vitest'
import { jobListingDate, locationLabel, safeJobUrl, timezoneLabel } from './jobs'
import type { Job } from '@/types/job'

const job = (values: Partial<Job> = {}): Job => ({ id: 'job_a', providerId: 'a', title: 'Software Engineer', company: 'Example', source: 'ashby:example', sourceName: 'Example careers', sourceKind: 'company', sourceUrl: 'https://example.com/a', applyUrl: 'https://example.com/a', description: '', workplace: 'hybrid', locations: [{ name: 'Berlin', country: 'Germany', countryCode: 'DE', latitude: 52.5, longitude: 13.4, precision: 'city' }], remote: { scope: 'unspecified', regions: [], timezones: [] }, employmentType: 'full_time', seniority: 'unknown', fetchedAt: '2026-10-31T00:00:00Z', closed: false, saved: false, ...values })

describe('job presentation boundaries', () => {
  it('labels provider updates separately from original posting dates', () => {
    const updated = '2026-10-30T00:00:00Z'
    const posted = '2026-09-01T00:00:00Z'
    expect(jobListingDate(job({ sourceUpdatedAt: updated }))).toEqual({ date: updated, label: 'Updated' })
    expect(jobListingDate(job({ postedAt: posted, sourceUpdatedAt: updated }))).toEqual({ date: posted, label: 'Posted' })
    expect(jobListingDate(job())).toEqual({ date: undefined, label: 'Listing date' })
  })
  it('represents worldwide and unknown remote restrictions differently', () => {
    expect(locationLabel(job({ workplace: 'remote' }))).toContain('unspecified')
    expect(locationLabel(job({ workplace: 'remote', remote: { scope: 'worldwide', regions: [], timezones: [] } }))).toBe('Remote worldwide')
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
})
