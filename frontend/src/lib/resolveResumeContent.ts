import type { FullResume, MiscEntry } from '@/types/resume'
import type { ContentEntry, ContentSection, ResumeContent } from '@/templates/contract'
import { markdownToHtml } from './markdown'
import { resolveSectionRefs, SECTION_LABELS } from './sectionOrder'

function formatMonthYear(value: string | null) {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('en-US', { month: 'short', year: 'numeric' })
}

function dateRange(start: string | null, end: string | null, isCurrent?: boolean) {
  const s = formatMonthYear(start)
  const e = isCurrent ? 'Present' : formatMonthYear(end)
  if (!s && !e) return ''
  return [s, e].filter(Boolean).join(' – ')
}

function miscByKind(entries: MiscEntry[], kind: MiscEntry['kind']) {
  return entries.filter((e) => e.kind === kind)
}

/** Resolves the client-side draft/full-resume state into the template-facing
 * content contract — the only translation point between content data and
 * template rendering. Order, visibility, and title overrides come from
 * resolveSectionRefs (design.sectionOrder), not resume_section_configs —
 * hidden sections and empty collections are dropped here so templates never
 * have to check visibility or emptiness themselves. */
export function resolveResumeContent(data: FullResume): ResumeContent {
  const { resume } = data
  const refs = resolveSectionRefs(data)
  const sections: ContentSection[] = []

  for (const ref of refs) {
    if (!ref.isVisible) continue
    const type = ref.sectionType
    const title = ref.titleOverride || SECTION_LABELS[type] || type

    if (type === 'summary') {
      if (!resume.summary) continue
      sections.push({ key: 'summary', type, title, entries: [{ key: 'summary', description: resume.summary }] })
    } else if (type === 'work_experience') {
      if (data.work_experiences.length === 0) continue
      const entries: ContentEntry[] = data.work_experiences.map((w) => ({
        key: `we-${w.id}`,
        title: `${w.title} · ${w.company}`,
        subtitle: w.location ?? undefined,
        dateLabel: dateRange(w.start_date, w.end_date, w.is_current),
        linkUrl: w.company_url ?? undefined,
        linkLabel: w.company_url ? `Open ${w.company} website` : undefined,
        bodyHtml: markdownToHtml(w.content),
      }))
      sections.push({ key: 'work_experience', type, title, entries })
    } else if (type === 'education') {
      if (data.educations.length === 0) continue
      const entries: ContentEntry[] = data.educations.map((e) => ({
        key: `ed-${e.id}`,
        title: e.institution,
        subtitle: [e.degree, e.field_of_study].filter(Boolean).join(', ') || undefined,
        dateLabel: dateRange(e.start_date, e.end_date),
      }))
      sections.push({ key: 'education', type, title, entries })
    } else if (type === 'skills') {
      if (data.skill_groups.length === 0) continue
      const entries: ContentEntry[] = data.skill_groups.map((g) => ({
        key: `sk-${g.id}`,
        title: g.group_name,
        meta: g.items.map((i) => i.name).filter(Boolean).join(', '),
      }))
      sections.push({ key: 'skills', type, title, entries })
    } else if (type === 'projects') {
      if (data.projects.length === 0) continue
      const entries: ContentEntry[] = data.projects.map((p) => ({
        key: `pr-${p.id}`,
        title: p.name,
        dateLabel: dateRange(p.start_date, p.end_date),
        bodyHtml: markdownToHtml(p.content),
      }))
      sections.push({ key: 'projects', type, title, entries })
    } else if (type === 'certifications') {
      if (data.certifications.length === 0) continue
      const entries: ContentEntry[] = data.certifications.map((c) => ({
        key: `ce-${c.id}`,
        title: c.issuer ? `${c.name} · ${c.issuer}` : c.name,
        dateLabel: formatMonthYear(c.issue_date),
      }))
      sections.push({ key: 'certifications', type, title, entries })
    } else if (type === 'languages') {
      if (data.languages.length === 0) continue
      const entries: ContentEntry[] = [{
        key: 'languages',
        meta: data.languages.map((l) => [l.name, l.proficiency].filter(Boolean).join(' — ')).join(' · '),
      }]
      sections.push({ key: 'languages', type, title, entries })
    } else if (type === 'awards' || type === 'publications' || type === 'volunteer') {
      const kind = type === 'awards' ? 'award' : type === 'publications' ? 'publication' : 'volunteer'
      const items = miscByKind(data.misc_entries, kind)
      if (items.length === 0) continue
      const entries: ContentEntry[] = items.map((e) => ({
        key: `${type}-${e.id}`,
        title: e.issuer_or_org ? `${e.title} · ${e.issuer_or_org}` : e.title,
        dateLabel: formatMonthYear(e.entry_date),
        linkUrl: e.url ?? undefined,
        linkLabel: e.url ? `Open ${e.title} link` : undefined,
        description: e.description ?? undefined,
      }))
      sections.push({ key: type, type, title, entries })
    } else if (type === 'custom') {
      const section = data.custom_sections.find((s) => s.id === ref.customSectionId)
      if (!section || section.entries.length === 0) continue
      const entries: ContentEntry[] = section.entries.map((e) => ({
        key: `cu-${e.id}`,
        title: e.title ?? undefined,
        dateLabel: formatMonthYear(e.entry_date),
        description: e.description ?? undefined,
      }))
      sections.push({ key: `custom-${section.id}`, type, title: ref.titleOverride || section.title, entries })
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
