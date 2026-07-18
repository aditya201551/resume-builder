import type { FullResume } from '@/types/resume'

function mergeEntity<T extends { id: string }>(items: T[], overrides: Record<string, Record<string, unknown>>): T[] {
  return items.map((item) => (overrides[item.id] ? ({ ...item, ...overrides[item.id] } as T) : item))
}

/** Overlays in-progress (uncommitted) editor field values onto the draft data, for preview only. */
export function applyPreviewOverrides(data: FullResume, overrides: Record<string, Record<string, unknown>>): FullResume {
  if (Object.keys(overrides).length === 0) return data

  return {
    ...data,
    resume: overrides[data.resume.id] ? { ...data.resume, ...overrides[data.resume.id] } : data.resume,
    work_experiences: mergeEntity(data.work_experiences, overrides),
    educations: mergeEntity(data.educations, overrides),
    projects: mergeEntity(data.projects, overrides),
    certifications: mergeEntity(data.certifications, overrides),
    languages: mergeEntity(data.languages, overrides),
    misc_entries: mergeEntity(data.misc_entries, overrides),
    skill_groups: data.skill_groups.map((g) => {
      const merged = overrides[g.id] ? { ...g, ...overrides[g.id] } : g
      return { ...merged, items: mergeEntity(g.items, overrides) }
    }),
    custom_sections: data.custom_sections.map((s) => {
      const merged = overrides[s.id] ? { ...s, ...overrides[s.id] } : s
      return { ...merged, entries: mergeEntity(s.entries, overrides) }
    }),
  }
}
