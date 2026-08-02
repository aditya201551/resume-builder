import type { FullResume } from '@/types/resume'
import type { SectionRef } from '@/types/design'

// The fixed catalog of non-custom section types and their default display
// labels — used both to back-fill missing refs and to resolve a ref's
// title when no override is set.
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

/**
 * Resolves the single-column ("one" layout mode) section order — the sole
 * source of truth for order/visibility/title-overrides now that
 * resume_section_configs is retired from the frontend. Both
 * resolveResumeContent (rendering) and DesignMode (the reorder UI) call
 * this same function so they can never disagree about what "the current
 * order" is.
 *
 * Default (non-custom) types back-fill onto the end only once they have
 * actual content — an empty "Certifications" a user hasn't touched yet
 * shouldn't clutter the design panel's order list. Custom sections
 * back-fill as soon as they exist (creating one is an explicit action, not
 * an ambient default), and a stored ref whose custom section no longer
 * exists is dropped rather than left dangling.
 */
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

/**
 * Same role as resolveSectionRefs, for two-column templates (layout.mode
 * "two") — reads/backfills against sectionOrder.two.{left,right} instead of
 * .one.sections. The only behavioral difference: an unassigned type (new
 * content, or a ref present in neither list) back-fills into `right` (the
 * main column) rather than a single flat list — sidebar placement should be
 * deliberate (seeded by the template's default_design), not incidental.
 */
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
