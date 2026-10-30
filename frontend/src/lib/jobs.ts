import type { Bounds, Job, JobSearchResult, MapJob } from '@/types/job'

export function safeJobUrl(raw: string | undefined): string | undefined {
  if (!raw) return undefined
  try {
    const url = new URL(raw)
    return ['https:', 'http:'].includes(url.protocol) && !url.username && !url.password ? url.href : undefined
  } catch { return undefined }
}

export function parseBounds(raw: string | null): Bounds | undefined {
  if (!raw) return undefined
  const values = raw.split(',').map(Number)
  if (values.length !== 4 || values.some(n => !Number.isFinite(n))) return undefined
  const [west, south, east, north] = values
  if (west < -180 || west > 180 || east < -180 || east > 180 || south < -90 || north > 90 || south >= north) return undefined
  return values as Bounds
}

export function jobSearchQuery(params: URLSearchParams): string {
  const out = new URLSearchParams()
  for (const key of ['q', 'workplace', 'country', 'remoteCountry', 'includeUnknown', 'employment', 'seniority', 'days', 'source', 'saved', 'page']) {
    const value = params.get(key)
    if (value) out.set(key, value)
  }
  const bounds = parseBounds(params.get('bounds'))
  if (bounds) out.set('bounds', bounds.join(','))
  out.set('pageSize', '50')
  out.sort()
  return out.toString()
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

export function jobFeatures(jobs: (MapJob & { closed?: boolean })[], bounds?: Bounds) {
  const features = jobs.flatMap(job => job.workplace === 'remote' || job.closed ? [] : job.locations.flatMap((location, index) => {
    const { longitude: lng, latitude: lat } = location
    if (lng === undefined || lat === undefined || !Number.isFinite(lat) || !Number.isFinite(lng)) return []
    if (bounds) {
      const [west, south, east, north] = bounds
      if (lat < south || lat > north || !(west <= east ? lng >= west && lng <= east : lng >= west || lng <= east)) return []
    }
    return [{ type: 'Feature' as const, id: `${job.id}-${index}`, geometry: { type: 'Point' as const, coordinates: [lng, lat] }, properties: { jobId: job.id, title: job.title, company: job.company, location: location.name, precision: location.precision } }]
  }))
  return { type: 'FeatureCollection' as const, features }
}

export function applySaved(result: JobSearchResult, id: string, saved: boolean): JobSearchResult {
  return { ...result, jobs: result.jobs.map(job => job.id === id ? { ...job, saved } : job) }
}

export function jobDate(raw?: string): string {
  if (!raw) return 'Date not supplied'
  const date = new Date(raw)
  return Number.isFinite(date.getTime()) ? new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(date) : 'Date not supplied'
}

export function timezoneLabel(offset: number): string {
  const minutes = Math.round(Math.abs(offset) * 60)
  return `UTC${offset >= 0 ? '+' : '-'}${String(Math.floor(minutes / 60)).padStart(2, '0')}:${String(minutes % 60).padStart(2, '0')}`
}
