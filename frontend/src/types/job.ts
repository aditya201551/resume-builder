export type Workplace = 'onsite' | 'hybrid' | 'remote' | 'unknown'
export type Bounds = [number, number, number, number]

export interface JobLocation {
  name: string
  country: string
  countryCode: string
  latitude?: number
  longitude?: number
  precision: 'office' | 'city' | 'region' | 'unknown'
  provenance?: string
}

export interface Job {
  id: string
  providerId: string
  source: string
  sourceName: string
  sourceKind: 'company' | 'aggregator' | 'career_page'
  sourceUrl: string
  applyUrl: string
  company: string
  companyUrl?: string
  title: string
  description: string
  workplace: Workplace
  locations: JobLocation[]
  remote: { scope: 'worldwide' | 'restricted' | 'unspecified'; regions: string[]; timezones: number[] }
  employmentType: string
  seniority: string
  salary?: string
  language?: string
  postedAt?: string
  expiresAt?: string
  fetchedAt: string
  closed: boolean
  saved: boolean
}

export interface JobSource {
  id: string
  name: string
  kind: string
  status: 'ready' | 'refreshing' | 'error' | 'needs_key' | 'quota_exhausted'
  message?: string
  jobCount: number
  refreshedAt?: string
}

export interface JobSearchResult {
  jobs: Job[]
  total: number
  page: number
  pageSize: number
  hasMore: boolean
  mapped: number
  remoteCount: number
  sources: JobSource[]
  refreshing: boolean
}

export interface JobCountry { code: string; name: string; continent: string; aliases: string[] }
export interface JobPlace {
  id: string
  name: string
  latitude: number
  longitude: number
  countryCode: string
  country: string
  attribution: string
}
