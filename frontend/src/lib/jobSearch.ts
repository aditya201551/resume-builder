import type { Job } from '@/types/job'

export interface JobSearchBranch { workplace: 'all' | 'remote' | 'onsite' | 'hybrid' | 'unknown'; country: string; city: string }
export interface SearchIntent {
  version: 1
  query: string
  branches: JobSearchBranch[]
  employmentType: string
  seniority: string
  postedDays: number
  excludeCompanies: string[]
  includeUnknown: boolean
}
export interface DiscoverySource { id: string; name: string; status: string; message?: string; fetchedAt?: string }
export interface DiscoveryPage { jobs: Job[]; sources: DiscoverySource[]; nextToken?: string; retry: boolean }
export interface JobRefinement { intent: SearchIntent; message: string; apply: boolean }

export function defaultJobIntent(): SearchIntent {
  return { version: 1, query: '', branches: [{ workplace: 'all', country: '', city: '' }], employmentType: '', seniority: '', postedDays: 0, excludeCompanies: [], includeUnknown: false }
}

export function readJobIntent(raw: string | null): SearchIntent {
  if (!raw || raw.length > 8000) return defaultJobIntent()
  try {
    const s = JSON.parse(raw) as SearchIntent
    if (s.version !== 1 || typeof s.query !== 'string' || s.query.length > 160 || !Array.isArray(s.branches) || s.branches.length < 1 || s.branches.length > 4) return defaultJobIntent()
    if (!s.branches.every(b => b && ['all', 'remote', 'onsite', 'hybrid', 'unknown'].includes(b.workplace) && typeof b.country === 'string' && b.country.length <= 2 && typeof b.city === 'string' && b.city.length <= 100 && (b.workplace !== 'remote' || b.city === ''))) return defaultJobIntent()
    if (!['', 'full_time', 'part_time', 'contract', 'internship', 'unknown'].includes(s.employmentType) || !['', 'junior', 'mid', 'senior', 'lead', 'unknown'].includes(s.seniority) || !Number.isInteger(s.postedDays) || s.postedDays < 0 || s.postedDays > 365 || typeof s.includeUnknown !== 'boolean' || !Array.isArray(s.excludeCompanies) || s.excludeCompanies.length > 10 || !s.excludeCompanies.every(c => typeof c === 'string' && c.length <= 100)) return defaultJobIntent()
    return s
  } catch { return defaultJobIntent() }
}

export function uniqueDiscoveryJobs(pages: DiscoveryPage[]): Job[] {
  const seen = new Set<string>()
  return pages.flatMap(p => p.jobs).filter(job => {
    let key = job.id
    try { const u = new URL(job.applyUrl); u.hash = ''; for (const k of [...u.searchParams.keys()]) if (k.toLowerCase().startsWith('utm_')) u.searchParams.delete(k); u.searchParams.sort(); key = u.href } catch { /* Provider ID remains the fallback. */ }
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}
