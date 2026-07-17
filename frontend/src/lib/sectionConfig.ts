import type { SectionConfig } from '@/types/resume'

export const SECTION_LABELS: Record<string, string> = {
  contact: 'Contact',
  summary: 'Summary',
  work_experience: 'Work Experience',
  education: 'Education',
  skills: 'Skills',
  projects: 'Projects',
  certifications: 'Certifications',
  languages: 'Languages',
  awards: 'Awards',
  publications: 'Publications',
  volunteer: 'Volunteer',
  custom: 'Custom section',
}

export function sectionConfigKey(config: Pick<SectionConfig, 'section_type' | 'custom_section_id'>) {
  return `${config.section_type}:${config.custom_section_id ?? ''}`
}

export function sectionConfigUrl(resumeId: string, config: Pick<SectionConfig, 'section_type' | 'custom_section_id'>) {
  const base = `/api/resumes/${resumeId}/section-configs/${config.section_type}`
  return config.custom_section_id ? `${base}?custom_section_id=${config.custom_section_id}` : base
}
