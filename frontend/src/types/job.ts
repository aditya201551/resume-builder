export type Workplace = 'onsite' | 'hybrid' | 'remote' | 'unknown'

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
  sourceUpdatedAt?: string
  expiresAt?: string
  fetchedAt: string
  checkedAt?: string
  checkFailed?: boolean
  closed: boolean
  saved: boolean
}

export interface JobCountry { code: string; name: string; continent: string; aliases: string[] }
