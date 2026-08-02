import type { FullResume, MiscEntry } from '@/types/resume'
import type { ResumeDesign, SectionRef } from '@/types/design'
import type { ContentEntry, ContentSection, ResumeContent } from '@/templates/contract'
import { markdownToHtml } from './markdown'
import { resolveSectionRefs, resolveTwoColumnSectionRefs, SECTION_LABELS } from './sectionOrder'

function formatMonthYear(value: string | null, format: ResumeDesign['dateFormat']) {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  if (format === 'YYYY') return String(d.getFullYear())
  if (format === 'MM/YYYY') return `${String(d.getMonth() + 1).padStart(2, '0')}/${d.getFullYear()}`
  return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' })
}

function dateRange(start: string | null, end: string | null, format: ResumeDesign['dateFormat'], isCurrent?: boolean) {
  const s = formatMonthYear(start, format)
  const e = isCurrent ? 'Present' : formatMonthYear(end, format)
  if (!s && !e) return ''
  return [s, e].filter(Boolean).join(' – ')
}

function miscByKind(entries: MiscEntry[], kind: MiscEntry['kind']) {
  return entries.filter((e) => e.kind === kind)
}

/** Builds one ContentSection from a single section ref, or null if it's
 * hidden or has no content to show — shared by both the single-column and
 * two-column assembly paths below so the per-type field mapping only lives
 * in one place. */
function buildSection(ref: SectionRef, data: FullResume, fmt: ResumeDesign['dateFormat']): ContentSection | null {
  if (!ref.isVisible) return null
  const type = ref.sectionType
  const title = ref.titleOverride || SECTION_LABELS[type] || type

  if (type === 'summary') {
    if (!data.resume.summary) return null
    return { key: 'summary', type, title, entries: [{ key: 'summary', description: data.resume.summary }] }
  }
  if (type === 'work_experience') {
    if (data.work_experiences.length === 0) return null
    const entries: ContentEntry[] = data.work_experiences.map((w) => ({
      key: `we-${w.id}`,
      title: `${w.title} · ${w.company}`,
      subtitle: w.location ?? undefined,
      dateLabel: dateRange(w.start_date, w.end_date, fmt, w.is_current),
      linkUrl: w.company_url ?? undefined,
      linkLabel: w.company_url ? `Open ${w.company} website` : undefined,
      bodyHtml: markdownToHtml(w.content),
    }))
    return { key: 'work_experience', type, title, entries }
  }
  if (type === 'education') {
    if (data.educations.length === 0) return null
    const entries: ContentEntry[] = data.educations.map((e) => ({
      key: `ed-${e.id}`,
      title: e.institution,
      subtitle: [e.degree, e.field_of_study].filter(Boolean).join(', ') || undefined,
      dateLabel: dateRange(e.start_date, e.end_date, fmt),
    }))
    return { key: 'education', type, title, entries }
  }
  if (type === 'skills') {
    if (data.skill_groups.length === 0) return null
    const entries: ContentEntry[] = data.skill_groups.map((g) => ({
      key: `sk-${g.id}`,
      title: g.group_name,
      meta: g.items.map((i) => i.name).filter(Boolean).join(', '),
    }))
    return { key: 'skills', type, title, entries }
  }
  if (type === 'projects') {
    if (data.projects.length === 0) return null
    const entries: ContentEntry[] = data.projects.map((p) => ({
      key: `pr-${p.id}`,
      title: p.name,
      dateLabel: dateRange(p.start_date, p.end_date, fmt),
      bodyHtml: markdownToHtml(p.content),
    }))
    return { key: 'projects', type, title, entries }
  }
  if (type === 'certifications') {
    if (data.certifications.length === 0) return null
    const entries: ContentEntry[] = data.certifications.map((c) => ({
      key: `ce-${c.id}`,
      title: c.issuer ? `${c.name} · ${c.issuer}` : c.name,
      dateLabel: formatMonthYear(c.issue_date, fmt),
    }))
    return { key: 'certifications', type, title, entries }
  }
  if (type === 'languages') {
    if (data.languages.length === 0) return null
    // One entry per language (not one pre-joined string) so
    // sectionDisplay.languages can actually render grid/bullets variants —
    // ClassicTemplate reconstructs the "text" mode's joined sentence from
    // these at render time to keep that variant looking identical to before.
    const entries: ContentEntry[] = data.languages.map((l) => ({
      key: `lang-${l.id}`,
      title: l.name,
      meta: l.proficiency ?? undefined,
    }))
    return { key: 'languages', type, title, entries }
  }
  if (type === 'awards' || type === 'publications' || type === 'volunteer') {
    const kind = type === 'awards' ? 'award' : type === 'publications' ? 'publication' : 'volunteer'
    const items = miscByKind(data.misc_entries, kind)
    if (items.length === 0) return null
    const entries: ContentEntry[] = items.map((e) => ({
      key: `${type}-${e.id}`,
      title: e.issuer_or_org ? `${e.title} · ${e.issuer_or_org}` : e.title,
      dateLabel: formatMonthYear(e.entry_date, fmt),
      linkUrl: e.url ?? undefined,
      linkLabel: e.url ? `Open ${e.title} link` : undefined,
      description: e.description ?? undefined,
    }))
    return { key: type, type, title, entries }
  }
  if (type === 'custom') {
    const section = data.custom_sections.find((s) => s.id === ref.customSectionId)
    if (!section || section.entries.length === 0) return null
    const entries: ContentEntry[] = section.entries.map((e) => ({
      key: `cu-${e.id}`,
      title: e.title ?? undefined,
      dateLabel: formatMonthYear(e.entry_date, fmt),
      description: e.description ?? undefined,
    }))
    return { key: `custom-${section.id}`, type, title: ref.titleOverride || section.title, entries }
  }
  return null
}

/** Resolves the client-side draft/full-resume state into the template-facing
 * content contract — the only translation point between content data and
 * template rendering. Order, visibility, and title overrides come from
 * design.sectionOrder (resolveSectionRefs for single-column templates,
 * resolveTwoColumnSectionRefs for layout.mode "two"), not resume_section_configs —
 * hidden sections and empty collections are dropped here so templates never
 * have to check visibility or emptiness themselves. */
export function resolveResumeContent(data: FullResume): ResumeContent {
  const { resume } = data
  const fmt = data.design.dateFormat
  const sections: ContentSection[] = []

  if (data.design.layout.mode === 'two') {
    const { left, right } = resolveTwoColumnSectionRefs(data)
    for (const ref of left) {
      const s = buildSection(ref, data, fmt)
      if (s) sections.push({ ...s, column: 'left' })
    }
    for (const ref of right) {
      const s = buildSection(ref, data, fmt)
      if (s) sections.push({ ...s, column: 'right' })
    }
  } else {
    // "mix" isn't rendered by any template yet — falls back to the same
    // flat single-column order as "one" until one is.
    for (const ref of resolveSectionRefs(data)) {
      const s = buildSection(ref, data, fmt)
      if (s) sections.push(s)
    }
  }

  return {
    header: {
      fullName: resume.full_name || 'Your name',
      headline: resume.headline,
      email: resume.email,
      phone: resume.phone,
      location: resume.location,
      links: resume.links,
    },
    sections,
  }
}
