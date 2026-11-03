import { Files, House, type LucideIcon } from 'lucide-react'
import { jobsNavigation } from '@/features/jobs'
import type { FeatureKey } from '@/platform/features/context'

export interface NavigationItem {
  to: string
  label: string
  icon: LucideIcon
  end: boolean
  feature?: FeatureKey
}

export const navigation: readonly NavigationItem[] = [
  { to: '/resumes', label: 'Home', icon: House, end: true },
  { to: '/resumes/library', label: 'My resumes', icon: Files, end: true },
  jobsNavigation,
]

export function visibleNavigation(enabled: (feature: FeatureKey) => boolean) {
  return navigation.filter(item => !item.feature || enabled(item.feature))
}
