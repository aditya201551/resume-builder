import { useMemo, type CSSProperties, type ReactNode } from 'react'
import { ExternalLink } from 'lucide-react'
import type { ResumeDesign } from '@/types/design'
import type { ContentEntry, ContentSection, ResumeContent } from '@/templates/contract'
import { useColumnPagination, type PaginationBlock } from '@/templates/pagination/useColumnPagination'
import styles from './classic.module.css'

const PAGE_WIDTH = 680
const PAGE_HEIGHT = 880

// Section types whose entries render as one atomic block per section rather
// than one block per entry — matches the original single-column engine's
// granularity for these three (a short skills/languages list, or the
// one-paragraph summary, never needs an internal page break).
const ATOMIC_SECTION_TYPES = new Set(['summary', 'skills', 'languages'])

interface RawBlock {
  key: string
  sectionKey: string
  entryKey: string
  node: ReactNode
}

function entryBlock(sectionKey: string, key: string, heading: string | undefined, body: ReactNode, entryKey: string = key): RawBlock {
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

/** Splits rendered rich-text HTML into one fragment per top-level paragraph
 * or list item, so pagination can flow bullet-by-bullet instead of treating
 * a whole description as one atomic unit. */
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

/** An entry's heading/meta line is glued to its first paragraph or bullet;
 * every subsequent paragraph/bullet is its own block so overflow only
 * carries forward what doesn't fit, not the whole entry. */
function richTextEntryBlocks(sectionKey: string, entryKey: string, heading: string | undefined, head: ReactNode, bodyHtml: string): RawBlock[] {
  const fragments = splitRichTextBlocks(bodyHtml)
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

function EntryLink({ url, label }: { url: string; label?: string }) {
  return (
    <a
      href={/^https?:\/\//i.test(url) ? url : `https://${url}`}
      target="_blank"
      rel="noopener noreferrer"
      className={styles.entryLinkIcon}
      aria-label={label}
    >
      <ExternalLink />
    </a>
  )
}

function entryHead(entry: ContentEntry) {
  return (
    <div className={styles.entryHead}>
      <span className={styles.entryTitle}>
        {entry.title}
        {entry.linkUrl && <EntryLink url={entry.linkUrl} label={entry.linkLabel} />}
      </span>
      {entry.dateLabel && <span className={styles.entryDates}>{entry.dateLabel}</span>}
    </div>
  )
}

function buildSectionBlocks(section: ContentSection): RawBlock[] {
  if (ATOMIC_SECTION_TYPES.has(section.type)) {
    if (section.type === 'summary') {
      return [entryBlock(section.key, section.key, section.title, <p>{section.entries[0]?.description}</p>)]
    }
    if (section.type === 'skills') {
      return [
        entryBlock(
          section.key,
          section.key,
          section.title,
          <div className={styles.skillGroups}>
            {section.entries.map((e) => (
              <div key={e.key}>
                <strong>{e.title}:</strong> {e.meta}
              </div>
            ))}
          </div>,
        ),
      ]
    }
    // languages
    return [entryBlock(section.key, section.key, section.title, <p>{section.entries[0]?.meta}</p>)]
  }

  return section.entries.flatMap((entry, i) => {
    const heading = i === 0 ? section.title : undefined
    if (entry.bodyHtml !== undefined) {
      return richTextEntryBlocks(section.key, entry.key, heading, entryHead(entry), entry.bodyHtml)
    }
    return [
      entryBlock(
        section.key,
        entry.key,
        heading,
        <>
          {entryHead(entry)}
          {entry.subtitle && <div className={styles.entrySub}>{entry.subtitle}</div>}
          {entry.description && <p>{entry.description}</p>}
        </>,
      ),
    ]
  })
}

function buildRawBlocks(content: ResumeContent): RawBlock[] {
  const blocks: RawBlock[] = [
    {
      key: 'header',
      sectionKey: 'header',
      entryKey: 'header',
      node: (
        <div>
          <h1 className={styles.name}>{content.header.fullName}</h1>
          {content.header.headline && <p className={styles.headline}>{content.header.headline}</p>}
          <div className={styles.metaRow}>
            {[content.header.email, content.header.phone, content.header.location].filter(Boolean).map((v) => (
              <span key={v}>{v}</span>
            ))}
            {content.header.links.map((l) => (
              <a
                key={l.url}
                href={/^https?:\/\//i.test(l.url) ? l.url : `https://${l.url}`}
                target="_blank"
                rel="noopener noreferrer"
                className={styles.metaLink}
              >
                {l.label || l.url}
              </a>
            ))}
          </div>
          <hr className={styles.rule} />
        </div>
      ),
    },
  ]

  content.sections.forEach((section) => blocks.push(...buildSectionBlocks(section)))
  return blocks
}

function withSpacing(raw: RawBlock[], spacing: ResumeDesign['spacing']) {
  return raw.map((b, i) => {
    const next = raw[i + 1]
    let marginBottom = 0
    if (b.sectionKey !== 'header' && next) {
      if (next.entryKey === b.entryKey) marginBottom = spacing.bulletGap
      else if (next.sectionKey === b.sectionKey) marginBottom = spacing.entryGap
      else marginBottom = spacing.sectionGap
    }
    return { key: b.key, node: <div style={{ marginBottom }}>{b.node}</div> }
  })
}

function pageStyleVars(tokens: ResumeDesign): CSSProperties {
  return {
    '--rd-background': tokens.colors.background,
    '--rd-text': tokens.colors.text,
    '--rd-accent': tokens.colors.accent,
    '--rd-font-family': tokens.typography.fontFamily,
    '--rd-font-size': `${tokens.typography.baseFontSizePt}px`,
    '--rd-line-height': tokens.typography.lineHeight,
    '--rd-name-size': `${tokens.typography.nameFontSizePt}px`,
    '--rd-section-heading-size': `${tokens.typography.sectionHeadingFontSizePt}px`,
    '--rd-margin-top': `${tokens.page.marginTop}px`,
    '--rd-margin-bottom': `${tokens.page.marginBottom}px`,
    '--rd-margin-left': `${tokens.page.marginLeft}px`,
    '--rd-margin-right': `${tokens.page.marginRight}px`,
  } as CSSProperties
}

export default function ClassicTemplate({
  content,
  tokens,
  onReady,
}: {
  content: ResumeContent
  tokens: ResumeDesign
  onReady?: () => void
}) {
  const contentHeight = PAGE_HEIGHT - tokens.page.marginTop - tokens.page.marginBottom

  // Memoized on content/tokens (not recomputed as fresh array literals every
  // render): useColumnPagination's effect depends on `paginationBlocks` by
  // reference, and its own setPages call triggers a re-render — an
  // unmemoized array here reruns the effect every render, which reruns
  // setPages, forever. This is load-bearing, not an optimization.
  const blocks = useMemo(() => withSpacing(buildRawBlocks(content), tokens.spacing), [content, tokens.spacing])
  const paginationBlocks: PaginationBlock[] = useMemo(
    () => blocks.map((b) => ({ key: b.key, column: 'left' as const, node: b.node })),
    [blocks],
  )
  const { pages, setMeasureRef } = useColumnPagination(paginationBlocks, contentHeight, onReady)

  const byKey = useMemo(() => new Map(blocks.map((b) => [b.key, b.node])), [blocks])
  const vars = pageStyleVars(tokens)

  return (
    <div className={styles.pageStack}>
      {/* Hidden measuring pass — same nodes, under the exact `.page` font
          metrics/width/padding so measured heights match the real render;
          only position/visibility/height/overflow are overridden here. */}
      <div
        className={styles.page}
        style={{ ...vars, position: 'fixed', top: 0, left: -99999, height: 'auto', overflow: 'visible', visibility: 'hidden' }}
        aria-hidden
      >
        {blocks.map((b) => (
          <div key={b.key} ref={setMeasureRef(b.key)} style={{ display: 'flow-root' }}>
            {b.node}
          </div>
        ))}
      </div>

      {pages.map((page, i) => (
        <div key={i} className={styles.page} style={{ width: PAGE_WIDTH, height: PAGE_HEIGHT, ...vars }}>
          {page.map((assignment) => (
            <div key={assignment.key}>{byKey.get(assignment.key)}</div>
          ))}
        </div>
      ))}
    </div>
  )
}
