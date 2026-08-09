import type { ResumeLink } from '@/types/resume'

export interface ContentHeader {
  fullName: string
  headline: string | null
  email: string | null
  phone: string | null
  location: string | null
  links: ResumeLink[]
}

export interface ContentEntry {
  key: string
  title?: string
  subtitle?: string
  dateLabel?: string
  linkUrl?: string
  linkLabel?: string
  bodyHtml?: string
  description?: string
  meta?: string
}

export interface ContentSection {
  key: string
  type: string
  title: string
  entries: ContentEntry[]
  column?: 'left' | 'right'
}

export interface ResumeContent {
  header: ContentHeader
  sections: ContentSection[]
}
