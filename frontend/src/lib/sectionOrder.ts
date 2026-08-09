import type { FullResume } from '@/types/resume'
import type { SectionRef } from '@/types/design'

export const SECTION_LABELS: Record<string, string> = {
  summary: 'Summary',
  work_experience: 'Experience',
  education: 'Education',
  skills: 'Skills',
  projects: 'Projects',
  certifications: 'Certifications',
  languages: 'Languages',
  awards: 'Awards',
  publications: 'Publications',
  volunteer: 'Volunteer',
}

const DEFAULT_TYPE_ORDER = Object.keys(SECTION_LABELS)

function hasContent(data: FullResume, type: string): boolean {
  switch (type) {
    case 'summary':
      return Boolean(data.resume.summary)
    case 'work_experience':
      return data.work_experiences.length > 0
    case 'education':
      return data.educations.length > 0
    case 'skills':
      return data.skill_groups.length > 0
    case 'projects':
      return data.projects.length > 0
    case 'certifications':
      return data.certifications.length > 0
    case 'languages':
      return data.languages.length > 0
    case 'awards':
      return data.misc_entries.some((e) => e.kind === 'award')
    case 'publications':
      return data.misc_entries.some((e) => e.kind === 'publication')
    case 'volunteer':
      return data.misc_entries.some((e) => e.kind === 'volunteer')
    default:
      return false
  }
}

export function resolveSectionRefs(data: FullResume): SectionRef[] {
  const stored = data.design.sectionOrder.one.sections.filter(
    (r) => r.sectionType !== 'custom' || data.custom_sections.some((s) => s.id === r.customSectionId),
  )

  const refs = [...stored]
  const knownTypes = new Set(refs.filter((r) => r.sectionType !== 'custom').map((r) => r.sectionType))
  const knownCustomIds = new Set(refs.filter((r) => r.sectionType === 'custom').map((r) => r.customSectionId))

  for (const type of DEFAULT_TYPE_ORDER) {
    if (!knownTypes.has(type) && hasContent(data, type)) {
      refs.push({ sectionType: type, customSectionId: null, isVisible: true, titleOverride: null })
    }
  }
  for (const section of data.custom_sections) {
    if (!knownCustomIds.has(section.id)) {
      refs.push({ sectionType: 'custom', customSectionId: section.id, isVisible: true, titleOverride: null })
    }
  }

  return refs
}

export function sectionRefKey(ref: SectionRef): string {
  return ref.sectionType === 'custom' ? `custom-${ref.customSectionId}` : ref.sectionType
}

export function resolveTwoColumnSectionRefs(data: FullResume): { left: SectionRef[]; right: SectionRef[] } {
  const dropOrphanCustom = (r: SectionRef) => r.sectionType !== 'custom' || data.custom_sections.some((s) => s.id === r.customSectionId)
  const left = data.design.sectionOrder.two.left.filter(dropOrphanCustom)
  const right = data.design.sectionOrder.two.right.filter(dropOrphanCustom)

  const knownTypes = new Set([...left, ...right].filter((r) => r.sectionType !== 'custom').map((r) => r.sectionType))
  const knownCustomIds = new Set([...left, ...right].filter((r) => r.sectionType === 'custom').map((r) => r.customSectionId))

  const newRight = [...right]
  for (const type of DEFAULT_TYPE_ORDER) {
    if (!knownTypes.has(type) && hasContent(data, type)) {
      newRight.push({ sectionType: type, customSectionId: null, isVisible: true, titleOverride: null })
    }
  }
  for (const section of data.custom_sections) {
    if (!knownCustomIds.has(section.id)) {
      newRight.push({ sectionType: 'custom', customSectionId: section.id, isVisible: true, titleOverride: null })
    }
  }

  return { left, right: newRight }
}
