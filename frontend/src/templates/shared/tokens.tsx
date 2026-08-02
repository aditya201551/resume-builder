// Pure token->CSS mapping shared across templates — like richText.ts, these
// have no JSX/CSS-module coupling (CSS variable names and data-* attribute
// names are a cross-template convention every template's own .module.css
// opts into, not tied to one template's class names), unlike each
// template's block-building helpers (entryBlock, buildSectionBlocks, etc.),
// which stay per-template.
import type { CSSProperties, ReactNode } from 'react'
import type { ResumeDesign } from '@/types/design'

export interface RawBlock {
  key: string
  /** Groups blocks that belong to the same content section, so withSpacing
   * can tell "next entry in the same section" from "next section" apart. */
  sectionKey: string
  /** Groups blocks that belong to the same entry (e.g. one work experience
   * entry split across several rich-text fragments). */
  entryKey: string
  node: ReactNode
}

/** Wraps each block in a margin-bottom sized by whether the next block (in
 * the same column) is part of the same entry, the same section, or a new
 * section — sectionGap/entryGap/bulletGap respectively. The header block
 * never gets a computed gap here; templates give it its own fixed spacing
 * since it's visually distinct (a rule, a border, etc.), not just another
 * section. */
export function withSpacing(raw: RawBlock[], spacing: ResumeDesign['spacing']): { key: string; node: ReactNode }[] {
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

/** Numeric/color design tokens as CSS custom properties, consumed by every
 * template's .module.css via `var(--rd-*, <fallback matching that
 * template's own default look>)`. */
export function pageStyleVars(tokens: ResumeDesign): CSSProperties {
  const accent = tokens.colors.accent
  return {
    '--rd-background': tokens.colors.background,
    '--rd-text': tokens.colors.text,
    '--rd-accent': accent,
    '--rd-font-family': tokens.typography.fontFamily,
    '--rd-name-font-family': tokens.typography.nameFontFamily === 'inherit' ? tokens.typography.fontFamily : tokens.typography.nameFontFamily,
    '--rd-font-size': `${tokens.typography.baseFontSizePt}px`,
    '--rd-line-height': tokens.typography.lineHeight,
    '--rd-name-size': `${tokens.typography.nameFontSizePt}px`,
    '--rd-headline-size': `${tokens.typography.headlineFontSizePt}px`,
    '--rd-section-heading-size': `${tokens.typography.sectionHeadingFontSizePt}px`,
    '--rd-entry-header-size': `${tokens.typography.entryHeaderFontSizePt}px`,
    '--rd-margin-top': `${tokens.page.marginTop}px`,
    '--rd-margin-bottom': `${tokens.page.marginBottom}px`,
    '--rd-margin-left': `${tokens.page.marginLeft}px`,
    '--rd-margin-right': `${tokens.page.marginRight}px`,
    '--rd-heading-transform': tokens.heading.capitalization === 'uppercase' ? 'uppercase' : 'none',
    '--rd-subtitle-style': tokens.entryLayout.subtitleStyle === 'italic' ? 'italic' : 'normal',
    '--rd-name-color': tokens.colors.applyAccent.name ? accent : tokens.colors.text,
    '--rd-heading-color': tokens.colors.applyAccent.headings ? accent : tokens.colors.text,
    '--rd-date-color': tokens.colors.applyAccent.dates ? accent : '#555',
    '--rd-icon-color': tokens.colors.applyAccent.icons ? accent : '#888',
    '--rd-link-underline': tokens.linkStyle.underline ? 'underline' : 'none',
    '--rd-link-color': tokens.linkStyle.useAccentColor ? accent : 'inherit',
  } as CSSProperties
}

/** Enum-driven visual variants that change CSS selectors rather than
 * property values become data-* attributes on the page element — see each
 * template's `.page[data-...]` rules. Keeps templates from having to thread
 * every token into every block-building function. */
export function pageDataAttrs(tokens: ResumeDesign): Record<string, string> {
  return {
    'data-heading-style': tokens.heading.style,
    'data-date-display-mode': tokens.entryLayout.dateDisplayMode,
    'data-header-align': tokens.header.alignText,
    'data-job-title-position': tokens.header.jobTitlePosition,
    'data-link-icon': tokens.linkStyle.showIcon ? 'shown' : 'hidden',
  }
}
