export interface ResumeLink {
  label: string
  url: string
}

export interface Resume {
  id: string
  user_id: string
  label: string
  template_id: string | null
  full_name: string
  headline: string | null
  email: string | null
  phone: string | null
  location: string | null
  photo_url: string | null
  summary: string | null
  links: ResumeLink[]
  created_at: string
  updated_at: string
  last_exported_at: string | null
}

export interface Template {
  id: string
  name: string
  slug: string
  regions: string[]
  default_region_map: Record<string, string>
  preview_thumbnail_url: string | null
  is_active: boolean
}

export interface WorkExperience {
  id: string
  resume_id: string
  company: string
  title: string
  location: string | null
  employment_type: string | null
  start_date: string | null
  end_date: string | null
  is_current: boolean
  content: string
  technologies: string[]
  sort_order: number
}

export interface Education {
  id: string
  resume_id: string
  institution: string
  degree: string | null
  field_of_study: string | null
  location: string | null
  start_date: string | null
  end_date: string | null
  gpa: string | null
  honors_description: string | null
  sort_order: number
}

export interface Project {
  id: string
  resume_id: string
  name: string
  content: string
  role: string | null
  technologies: string[]
  url: string | null
  start_date: string | null
  end_date: string | null
  sort_order: number
}

export interface Certification {
  id: string
  resume_id: string
  name: string
  issuer: string | null
  issue_date: string | null
  expiry_date: string | null
  credential_url: string | null
  sort_order: number
}

export interface Language {
  id: string
  resume_id: string
  name: string
  proficiency: string | null
  sort_order: number
}

export interface MiscEntry {
  id: string
  resume_id: string
  kind: 'award' | 'publication' | 'volunteer'
  title: string
  issuer_or_org: string | null
  url: string | null
  entry_date: string | null
  description: string | null
  sort_order: number
}

export interface SkillItem {
  id: string
  skill_group_id: string
  name: string
  proficiency: string | null
  sort_order: number
}

export interface SkillGroup {
  id: string
  resume_id: string
  group_name: string
  sort_order: number
  items: SkillItem[]
}

export interface CustomSectionEntry {
  id: string
  custom_section_id: string
  title: string | null
  description: string | null
  entry_date: string | null
  sort_order: number
}

export interface CustomSection {
  id: string
  resume_id: string
  title: string
  sort_order: number
  entries: CustomSectionEntry[]
}

export interface SectionConfig {
  id: string
  resume_id: string
  section_type: string
  custom_section_id: string | null
  region: string | null
  is_visible: boolean
  display_title_override: string | null
  sort_order: number
}

export interface ContentBlock {
  kind: 'work_experience' | 'project'
  id: string
  label: string
  content: string
  sort_order: number
}

export interface FullResume {
  resume: Resume
  work_experiences: WorkExperience[]
  educations: Education[]
  skill_groups: SkillGroup[]
  projects: Project[]
  certifications: Certification[]
  languages: Language[]
  misc_entries: MiscEntry[]
  custom_sections: CustomSection[]
  section_configs: SectionConfig[]
}
