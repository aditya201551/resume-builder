import { useLayoutEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { ExternalLink } from 'lucide-react'
import type { FullResume, MiscEntry } from '@/types/resume'
import { markdownToHtml } from '@/lib/markdown'
import styles from './LivePreview.module.css'

const PAGE_WIDTH = 680
const PAGE_ASPECT = 0.773 // US Letter (8.5:11) — matches ResumeThumbnail's card ratio
const PAGE_HEIGHT = Math.round(PAGE_WIDTH / PAGE_ASPECT)
const PAGE_PADDING_Y = 48
const CONTENT_HEIGHT = PAGE_HEIGHT - PAGE_PADDING_Y * 2

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

const DEFAULT_ORDER = [
  'summary',
  'work_experience',
  'education',
  'skills',
  'projects',
  'certifications',
  'languages',
  'awards',
  'publications',
  'volunteer',
] as const

function miscByKind(entries: MiscEntry[], kind: MiscEntry['kind']) {
  return entries.filter((e) => e.kind === kind)
}

interface RawBlock {
  key: string
  sectionKey: string
  entryKey: string
  node: ReactNode
}

function entryBlock(
  sectionKey: string,
  key: string,
  heading: string | undefined,
  body: ReactNode,
  entryKey: string = key,
): RawBlock {
  return {
    key,
    sectionKey,
    entryKey,
    node: (
      <div>
        {heading && <h2 className={styles.sectionTitle}>{heading}</h2>}
        {body}
      </div>
    ),
  }
}

/**
 * Splits rendered rich-text HTML into one fragment per top-level paragraph
 * or list item, so pagination can flow bullet-by-bullet instead of treating
 * a whole description as one atomic unit — a page break can land between
 * two bullets of the same job without pulling the whole entry to the next
 * page.
 */
function splitRichTextBlocks(html: string): string[] {
  if (!html.trim()) return []
  const root = new DOMParser().parseFromString(`<div>${html}</div>`, 'text/html').body.firstElementChild
  if (!root) return []
  const parts: string[] = []
  for (const child of Array.from(root.children)) {
    if (child.tagName === 'UL' || child.tagName === 'OL') {
      const tag = child.tagName.toLowerCase()
      const startAttr = child.hasAttribute('start') ? Number(child.getAttribute('start')) : 1
      Array.from(child.children).forEach((li, i) => {
        const openTag = tag === 'ol' ? `<ol start="${startAttr + i}">` : '<ul>'
        parts.push(`${openTag}${li.outerHTML}</${tag}>`)
      })
    } else {
      parts.push(child.outerHTML)
    }
  }
  return parts
}

/**
 * A resume entry's heading/meta line is glued to its first paragraph or
 * bullet (never left orphaned alone at a page bottom); every subsequent
 * paragraph/bullet is its own independent block so overflow only carries
 * forward the part that doesn't fit, not the whole entry.
 */
function richTextEntryBlocks(
  sectionKey: string,
  entryKey: string,
  heading: string | undefined,
  head: ReactNode,
  contentHtml: string,
): RawBlock[] {
  const fragments = splitRichTextBlocks(contentHtml)
  if (fragments.length === 0) {
    return [entryBlock(sectionKey, entryKey, heading, head, entryKey)]
  }
  const blocks: RawBlock[] = [
    entryBlock(
      sectionKey,
      `${entryKey}-0`,
      heading,
      <>
        {head}
        <div className={styles.richText} dangerouslySetInnerHTML={{ __html: fragments[0] }} />
      </>,
      entryKey,
    ),
  ]
  fragments.slice(1).forEach((frag, i) => {
    blocks.push({
      key: `${entryKey}-${i + 1}`,
      sectionKey,
      entryKey,
      node: <div className={styles.richText} dangerouslySetInnerHTML={{ __html: frag }} />,
    })
  })
  return blocks
}

/**
 * One block per resume entry (per bullet, for rich-text entries) — pagination
 * packs these individually so a page break falls between whole entries (or
 * between bullets within one), never splitting a heading from its content.
 * The first entry of a section carries that section's heading so a heading
 * never ends up orphaned alone at the bottom of a page.
 */
function buildRawBlocks(data: FullResume): RawBlock[] {
  const { resume } = data
  const blocks: RawBlock[] = []

  blocks.push({
    key: 'header',
    sectionKey: 'header',
    entryKey: 'header',
    node: (
      <div>
        <h1 className={styles.name}>{resume.full_name || 'Your name'}</h1>
        {resume.headline && <p className={styles.headline}>{resume.headline}</p>}
        <div className={styles.metaRow}>
          {[resume.email, resume.phone, resume.location].filter(Boolean).map((v) => (
            <span key={v}>{v}</span>
          ))}
          {resume.links.map((l) => (
            <span key={l.url}>{l.label || l.url}</span>
          ))}
        </div>
        <hr className={styles.rule} />
      </div>
    ),
  })

  const visibleTypes = new Set(
    data.section_configs.length > 0
      ? data.section_configs.filter((c) => c.is_visible).map((c) => c.section_type)
      : [...DEFAULT_ORDER, 'custom'],
  )

  const order =
    data.section_configs.length > 0
      ? [...data.section_configs].filter((c) => c.section_type !== 'contact').sort((a, b) => a.sort_order - b.sort_order)
      : DEFAULT_ORDER.map((t) => ({ section_type: t, custom_section_id: null, display_title_override: null }))

  for (const cfg of order) {
    const type = cfg.section_type
    if (!visibleTypes.has(type)) continue

    if (type === 'summary') {
      if (!resume.summary) continue
      blocks.push({
        key: 'summary',
        sectionKey: 'summary',
        entryKey: 'summary',
        node: (
          <div>
            <h2 className={styles.sectionTitle}>Summary</h2>
            <p>{resume.summary}</p>
          </div>
        ),
      })
    } else if (type === 'work_experience') {
      data.work_experiences.forEach((w, i) =>
        blocks.push(
          ...richTextEntryBlocks(
            'work_experience',
            `we-${w.id}`,
            i === 0 ? 'Experience' : undefined,
            <>
              <div className={styles.entryHead}>
                <span className={styles.entryTitle}>
                  {w.title} · {w.company}
                  {w.company_url && (
                    <a
                      href={w.company_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className={styles.companyLink}
                      aria-label={`Open ${w.company} website`}
                    >
                      <ExternalLink />
                    </a>
                  )}
                </span>
                <span className={styles.entryDates}>{dateRange(w.start_date, w.end_date, w.is_current)}</span>
              </div>
              {w.location && <div className={styles.entrySub}>{w.location}</div>}
            </>,
            markdownToHtml(w.content),
          ),
        ),
      )
    } else if (type === 'education') {
      data.educations.forEach((e, i) =>
        blocks.push(
          entryBlock(
            'education',
            `ed-${e.id}`,
            i === 0 ? 'Education' : undefined,
            <>
              <div className={styles.entryHead}>
                <span className={styles.entryTitle}>{e.institution}</span>
                <span className={styles.entryDates}>{dateRange(e.start_date, e.end_date)}</span>
              </div>
              {(e.degree || e.field_of_study) && (
                <div className={styles.entrySub}>{[e.degree, e.field_of_study].filter(Boolean).join(', ')}</div>
              )}
            </>,
          ),
        ),
      )
    } else if (type === 'skills') {
      if (data.skill_groups.length === 0) continue
      blocks.push({
        key: 'skills',
        sectionKey: 'skills',
        entryKey: 'skills',
        node: (
          <div>
            <h2 className={styles.sectionTitle}>Skills</h2>
            <div className={styles.skillGroups}>
              {data.skill_groups.map((g) => (
                <div key={g.id}>
                  <strong>{g.group_name}:</strong> {g.items.map((i) => i.name).filter(Boolean).join(', ')}
                </div>
              ))}
            </div>
          </div>
        ),
      })
    } else if (type === 'projects') {
      data.projects.forEach((p, i) =>
        blocks.push(
          ...richTextEntryBlocks(
            'projects',
            `pr-${p.id}`,
            i === 0 ? 'Projects' : undefined,
            <div className={styles.entryHead}>
              <span className={styles.entryTitle}>{p.name}</span>
              <span className={styles.entryDates}>{dateRange(p.start_date, p.end_date)}</span>
            </div>,
            markdownToHtml(p.content),
          ),
        ),
      )
    } else if (type === 'certifications') {
      data.certifications.forEach((c, i) =>
        blocks.push(
          entryBlock(
            'certifications',
            `ce-${c.id}`,
            i === 0 ? 'Certifications' : undefined,
            <div className={styles.entryHead}>
              <span className={styles.entryTitle}>
                {c.name}
                {c.issuer ? ` · ${c.issuer}` : ''}
              </span>
              <span className={styles.entryDates}>{formatMonthYear(c.issue_date)}</span>
            </div>,
          ),
        ),
      )
    } else if (type === 'languages') {
      if (data.languages.length === 0) continue
      blocks.push({
        key: 'languages',
        sectionKey: 'languages',
        entryKey: 'languages',
        node: (
          <div>
            <h2 className={styles.sectionTitle}>Languages</h2>
            <p>{data.languages.map((l) => [l.name, l.proficiency].filter(Boolean).join(' — ')).join(' · ')}</p>
          </div>
        ),
      })
    } else if (type === 'awards' || type === 'publications' || type === 'volunteer') {
      const kind = type === 'awards' ? 'award' : type === 'publications' ? 'publication' : 'volunteer'
      const label = type === 'awards' ? 'Awards' : type === 'publications' ? 'Publications' : 'Volunteer'
      miscByKind(data.misc_entries, kind).forEach((e, i) =>
        blocks.push(
          entryBlock(
            type,
            `${type}-${e.id}`,
            i === 0 ? label : undefined,
            <>
              <div className={styles.entryHead}>
                <span className={styles.entryTitle}>
                  {e.title}
                  {e.issuer_or_org ? ` · ${e.issuer_or_org}` : ''}
                </span>
                <span className={styles.entryDates}>{formatMonthYear(e.entry_date)}</span>
              </div>
              {e.description && <p>{e.description}</p>}
            </>,
          ),
        ),
      )
    } else if (type === 'custom') {
      const section = data.custom_sections.find((s) => s.id === cfg.custom_section_id)
      if (!section || section.entries.length === 0) continue
      const title = cfg.display_title_override || section.title
      section.entries.forEach((e, i) =>
        blocks.push(
          entryBlock(
            `custom-${section.id}`,
            `cu-${e.id}`,
            i === 0 ? title : undefined,
            <>
              <div className={styles.entryHead}>
                <span className={styles.entryTitle}>{e.title}</span>
                <span className={styles.entryDates}>{formatMonthYear(e.entry_date)}</span>
              </div>
              {e.description && <p>{e.description}</p>}
            </>,
          ),
        ),
      )
    }
  }

  return blocks
}

function withSpacing(raw: RawBlock[]) {
  return raw.map((b, i) => {
    const next = raw[i + 1]
    // header's own trailing <hr> already carries its bottom spacing
    let marginBottom = 0
    if (b.sectionKey !== 'header' && next) {
      if (next.entryKey === b.entryKey) marginBottom = 4 // bullet-to-bullet within one entry
      else if (next.sectionKey === b.sectionKey) marginBottom = 10 // entry-to-entry within one section
      else marginBottom = 18 // section-to-section
    }
    return { key: b.key, node: <div style={{ marginBottom }}>{b.node}</div> }
  })
}

export default function LivePreview({ data }: { data: FullResume }) {
  const blocks = useMemo(() => withSpacing(buildRawBlocks(data)), [data])
  const measureRefs = useRef(new Map<string, HTMLDivElement>())
  const [pageGroups, setPageGroups] = useState<string[][]>(() => [blocks.map((b) => b.key)])

  useLayoutEffect(() => {
    function recompute() {
      const groups: string[][] = []
      let current: string[] = []
      let currentHeight = 0
      for (const b of blocks) {
        const h = measureRefs.current.get(b.key)?.getBoundingClientRect().height ?? 0
        if (current.length > 0 && currentHeight + h > CONTENT_HEIGHT) {
          groups.push(current)
          current = []
          currentHeight = 0
        }
        current.push(b.key)
        currentHeight += h
      }
      if (current.length > 0) groups.push(current)
      setPageGroups(groups.length > 0 ? groups : [[]])
    }
    recompute()
    const observer = new ResizeObserver(recompute)
    measureRefs.current.forEach((el) => observer.observe(el))
    return () => observer.disconnect()
  }, [blocks])

  const byKey = new Map(blocks.map((b) => [b.key, b.node]))

  return (
    <div className={styles.pageStack}>
      {/* Hidden measuring pass — same nodes, under the exact `.page` font
          metrics/width/padding so measured heights match the real render;
          only position/visibility/height/overflow are overridden here.
          flow-root per block so each wrapper's own bottom margin is
          included in its measured height instead of collapsing out. */}
      <div
        className={styles.page}
        style={{ position: 'fixed', top: 0, left: -99999, height: 'auto', overflow: 'visible', visibility: 'hidden' }}
        aria-hidden
      >
        {blocks.map((b) => (
          <div
            key={b.key}
            ref={(el) => {
              if (el) measureRefs.current.set(b.key, el)
              else measureRefs.current.delete(b.key)
            }}
            style={{ display: 'flow-root' }}
          >
            {b.node}
          </div>
        ))}
      </div>

      {pageGroups.map((group, i) => (
        <div key={i} className={styles.page}>
          {group.map((key) => (
            <div key={key}>{byKey.get(key)}</div>
          ))}
        </div>
      ))}
    </div>
  )
}
