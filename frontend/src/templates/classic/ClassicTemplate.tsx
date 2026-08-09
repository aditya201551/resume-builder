import { useMemo, type ReactNode } from 'react'
import { ExternalLink } from 'lucide-react'
import type { ResumeDesign, SectionDisplay } from '@/types/design'
import type { ContentEntry, ContentSection, ResumeContent } from '@/templates/contract'
import { useColumnPagination, type PaginationBlock } from '@/templates/pagination/useColumnPagination'
import { splitRichTextBlocks } from '@/templates/shared/richText'
import { withSpacing, pageStyleVars, pageDataAttrs, type RawBlock } from '@/templates/shared/tokens'
import { cn } from '@/lib/utils'
import styles from './classic.module.css'

const PAGE_WIDTH = 680
const PAGE_HEIGHT = 880

const ATOMIC_SECTION_TYPES = new Set(['summary', 'skills', 'languages'])

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

function displayEntries(entries: ContentEntry[], mode: SectionDisplay['skills'], line: (e: ContentEntry) => ReactNode) {
  if (mode === 'bullets') {
    return (
      <ul className={cn(styles.skillGroups, styles.skillGroupsBullets)}>
        {entries.map((e) => (
          <li key={e.key}>{line(e)}</li>
        ))}
      </ul>
    )
  }
  return (
    <div className={cn(styles.skillGroups, mode === 'grid' && styles.skillGroupsGrid)}>
      {entries.map((e) => (
        <div key={e.key}>{line(e)}</div>
      ))}
    </div>
  )
}

function buildSectionBlocks(section: ContentSection, sectionDisplay: SectionDisplay): RawBlock[] {
  if (ATOMIC_SECTION_TYPES.has(section.type)) {
    if (section.type === 'summary') {
      return [entryBlock(section.key, section.key, section.title, <p>{section.entries[0]?.description}</p>)]
    }
    if (section.type === 'skills') {
      const body = displayEntries(section.entries, sectionDisplay.skills, (e) => (
        <>
          <strong>{e.title}:</strong> {e.meta}
        </>
      ))
      return [entryBlock(section.key, section.key, section.title, body)]
    }
    if (sectionDisplay.languages === 'text') {
      const joined = section.entries.map((e) => [e.title, e.meta].filter(Boolean).join(' — ')).join(' · ')
      return [entryBlock(section.key, section.key, section.title, <p>{joined}</p>)]
    }
    const body = displayEntries(section.entries, sectionDisplay.languages, (e) => [e.title, e.meta].filter(Boolean).join(' — '))
    return [entryBlock(section.key, section.key, section.title, body)]
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

function buildRawBlocks(content: ResumeContent, sectionDisplay: SectionDisplay): RawBlock[] {
  const blocks: RawBlock[] = [
    {
      key: 'header',
      sectionKey: 'header',
      entryKey: 'header',
      node: (
        <div className={styles.header}>
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

  content.sections.forEach((section) => blocks.push(...buildSectionBlocks(section, sectionDisplay)))
  return blocks
}

function PageFooter({ footer, header, pageIndex, pageCount }: { footer: ResumeDesign['footer']; header: ResumeContent['header']; pageIndex: number; pageCount: number }) {
  if (!footer.showPageNumbers && !footer.showEmail && !footer.showName) return null
  return (
    <div className={styles.footer}>
      {footer.showName && <span>{header.fullName}</span>}
      {footer.showEmail && header.email && <span>{header.email}</span>}
      {footer.showPageNumbers && (
        <span>
          {pageIndex + 1} / {pageCount}
        </span>
      )}
    </div>
  )
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

  const blocks = useMemo(
    () => withSpacing(buildRawBlocks(content, tokens.sectionDisplay), tokens.spacing),
    [content, tokens.sectionDisplay, tokens.spacing],
  )
  const paginationBlocks: PaginationBlock[] = useMemo(
    () => blocks.map((b) => ({ key: b.key, column: 'left' as const, node: b.node })),
    [blocks],
  )
  const { pages, setMeasureRef } = useColumnPagination(paginationBlocks, contentHeight, onReady)

  const byKey = useMemo(() => new Map(blocks.map((b) => [b.key, b.node])), [blocks])
  const vars = pageStyleVars(tokens)
  const dataAttrs = pageDataAttrs(tokens)

  return (
    <div className={styles.pageStack}>
      {/* Hidden measuring pass — same nodes, under the exact `.page` font
          metrics/width/padding so measured heights match the real render;
          only position/visibility/height/overflow are overridden here. */}
      <div
        className={styles.page}
        style={{ ...vars, position: 'fixed', top: 0, left: -99999, height: 'auto', overflow: 'visible', visibility: 'hidden' }}
        aria-hidden
        {...dataAttrs}
      >
        {blocks.map((b) => (
          <div key={b.key} ref={setMeasureRef(b.key)} style={{ display: 'flow-root' }}>
            {b.node}
          </div>
        ))}
      </div>

      {pages.map((page, i) => (
        <div key={i} className={styles.page} style={{ width: PAGE_WIDTH, height: PAGE_HEIGHT, ...vars }} {...dataAttrs}>
          {page.map((assignment) => (
            <div key={assignment.key}>{byKey.get(assignment.key)}</div>
          ))}
          <PageFooter footer={tokens.footer} header={content.header} pageIndex={i} pageCount={pages.length} />
        </div>
      ))}
    </div>
  )
}
