import type { Job } from '@/types/job'

export function safeJobUrl(raw: string | undefined): string | undefined {
  if (!raw) return undefined
  try {
    const url = new URL(raw)
    return ['https:', 'http:'].includes(url.protocol) && !url.username && !url.password ? url.href : undefined
  } catch { return undefined }
}

export const workplaceLabels = { onsite: 'Onsite', hybrid: 'Hybrid', remote: 'Remote', unknown: 'Workplace unspecified' }
export const employmentLabels: Record<string, string> = { full_time: 'Full time', part_time: 'Part time', contract: 'Contract', internship: 'Internship', unknown: 'Employment unspecified' }

export function locationLabel(job: Job): string {
  if (job.workplace === 'remote') {
    if (job.remote.scope === 'worldwide') return 'Remote worldwide'
    if (job.remote.scope === 'restricted') return `Remote in ${job.remote.regions.join(', ')}`
    return 'Remote; hiring regions unspecified'
  }
  return job.locations.map(l => l.name).filter(Boolean).join(' / ') || 'Location unspecified'
}

export function jobDate(raw?: string): string {
  if (!raw) return 'Date not supplied'
  const date = new Date(raw)
  return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(date) : 'Date not supplied'
}

export function jobListingDate(job: Pick<Job, 'postedAt' | 'sourceUpdatedAt'>) {
  return { date: job.postedAt ?? job.sourceUpdatedAt, label: job.postedAt ? 'Posted' : job.sourceUpdatedAt ? 'Updated' : 'Listing date' }
}

export function jobDateTime(raw?: string): string {
  if (!raw) return 'Not yet checked'
  const date = new Date(raw)
  return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(date) : 'Date not supplied'
}

export function timezoneLabel(offset: number): string {
  const minutes = Math.round(Math.abs(offset) * 60)
  return `UTC${offset >= 0 ? '+' : '-'}${String(Math.floor(minutes / 60)).padStart(2, '0')}:${String(minutes % 60).padStart(2, '0')}`
}
