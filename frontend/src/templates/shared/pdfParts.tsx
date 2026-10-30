import type { ReactNode } from 'react'
import { View, Text, H1, H2, Strong, Em, Link, Fixed, UnorderedList, OrderedList, ListItem, type Style } from '@formepdf/react'
import { parseDocument } from 'htmlparser2'
import { type ChildNode, Element, Text as HtmlText } from 'domhandler'
import type { ContentEntry, ContentSection, ResumeContent } from '@/templates/contract'
import { PDF_PAGE_SIZES, pxToPt as pt, safePdfLink } from '@/lib/pdf/types'
import type { ResumeDesign } from '@/types/design'

function inline(nodes: ChildNode[], linkStyle: Style): ReactNode[] {
  return nodes.map((node, i) => {
    if (node instanceof HtmlText) return node.data
    if (!(node instanceof Element)) return null
    const children = inline(node.children, linkStyle)
    if (node.name === 'br') return '\n'
    if (node.name === 'strong' || node.name === 'b') return <Strong key={i}>{children}</Strong>
    if (node.name === 'em' || node.name === 'i') return <Em key={i}>{children}</Em>
    if (node.name === 'a') {
      const href = safePdfLink(node.attribs.href ?? '')
      return href ? <Link key={i} href={href} style={linkStyle}>{children}</Link> : children
    }
    if (['script', 'style', 'iframe'].includes(node.name)) return null
    return children
  })
}

function richBlocks(nodes: ChildNode[], gap: number, linkStyle: Style): ReactNode[] {
  return nodes.map((node, i) => {
    if (node instanceof HtmlText) return node.data.trim() ? <Text key={i}>{node.data}</Text> : null
    if (!(node instanceof Element) || ['script', 'style', 'iframe'].includes(node.name)) return null
    if (node.name === 'ul' || node.name === 'ol') {
      const List = node.name === 'ul' ? UnorderedList : OrderedList
      return <List key={i} style={{ marginBottom: gap, paddingLeft: pt(15) }}>
        {node.children.filter((n) => n instanceof Element && n.name === 'li').map((n, j) =>
          <ListItem key={j} style={{ marginBottom: gap }}>{richBlocks((n as Element).children, gap, linkStyle)}</ListItem>)}
      </List>
    }
    if (node.name === 'div') return <View key={i}>{richBlocks(node.children, gap, linkStyle)}</View>
    return <Text key={i} style={{ marginBottom: gap, minOrphanLines: 2, minWidowLines: 2 }}>{inline(node.children, linkStyle)}</Text>
  })
}

function textOf(entry: ContentEntry) {
  return [[entry.title, entry.meta].filter(Boolean).join(entry.meta ? ': ' : ''), entry.dateLabel].filter(Boolean).join(' · ')
}

export function createPdfTemplateParts(content: ResumeContent, d: ResumeDesign) {
  const t = d.typography
  const pageSize = PDF_PAGE_SIZES[d.page.format]
  const accent = d.colors.accent
  const linkStyle: Style = { color: d.linkStyle.useAccentColor ? accent : d.colors.text, textDecoration: d.linkStyle.underline ? 'underline' : 'none' }
  const headingColor = d.colors.applyAccent.headings ? accent : d.colors.text
  const footerEnabled = Object.values(d.footer).some(Boolean)
  const margins = { top: pt(d.page.marginTop), right: pt(d.page.marginRight), bottom: Math.max(pt(d.page.marginBottom), footerEnabled ? 27 : 0), left: pt(d.page.marginLeft) }
  const bodyWidth = pageSize.width - margins.left - margins.right
  if (bodyWidth < 90 || pageSize.height - margins.top - margins.bottom < 90) throw new Error('Reduce the page margins to leave enough room for your resume.')
  const headingStyle: Style = {
    fontFamily: t.fontFamily, fontSize: pt(t.sectionHeadingFontSizePt), fontWeight: 700,
    lineHeight: 1.3, color: headingColor, marginBottom: pt(8),
    textTransform: d.heading.capitalization === 'uppercase' ? 'uppercase' : 'none',
    ...(d.heading.style === 'box' ? { backgroundColor: accent, color: '#ffffff', padding: pt(4) } : {}),
    ...(['line', 'thinLine'].includes(d.heading.style) ? { borderBottom: `${d.heading.style === 'line' ? 1.5 : 0.75}pt solid ${accent}`, paddingBottom: pt(4) } : {}),
    ...(d.heading.style === 'underline' ? { textDecoration: 'underline' as const } : {}),
  }
  const nameFont = t.nameFontFamily === 'inherit' ? t.fontFamily : t.nameFontFamily
  function header() {
    const aligned = d.header.alignText
    const inlineTitle = d.header.jobTitlePosition === 'inline'
    return <View bookmark="contact" style={{ marginBottom: pt(20), borderBottom: `0.75pt solid ${accent}`, paddingBottom: pt(14), textAlign: aligned }}>
      <H1 bookmark="contact" style={{ margin: 0, fontFamily: nameFont, fontSize: pt(t.nameFontSizePt), lineHeight: 1.2, color: d.colors.applyAccent.name ? accent : d.colors.text }}>
        {content.header.fullName}
        {inlineTitle && content.header.headline ? <Text style={{ fontSize: pt(t.headlineFontSizePt), fontWeight: 400 }}>  {content.header.headline}</Text> : null}
      </H1>
      {!inlineTitle && content.header.headline ? <Text style={{ fontSize: pt(t.headlineFontSizePt), marginTop: pt(3) }}>{content.header.headline}</Text> : null}
      <Text style={{ fontSize: pt(11), marginTop: pt(8), lineHeight: 1.5 }}>
        {[content.header.email, content.header.phone, content.header.location].filter(Boolean).join('  ·  ')}
      </Text>
      {content.header.links.length > 0 && <Text style={{ fontSize: pt(11), marginTop: pt(3) }}>
        {content.header.links.map((link, i) => {
          const href = safePdfLink(link.url)
          return <Text key={i}>{i > 0 ? '  ·  ' : ''}{href ? <Link href={href} style={linkStyle}>{link.label || link.url}</Link> : link.label || link.url}</Text>
        })}
      </Text>}
    </View>
  }
  function entryHead(entry: ContentEntry) {
    const date = entry.dateLabel ? <Text style={{ fontSize: pt(t.baseFontSizePt - 1), color: d.colors.applyAccent.dates ? accent : d.colors.text, flexShrink: 0 }}>{entry.dateLabel}</Text> : null
    const href = entry.linkUrl && safePdfLink(entry.linkUrl)
    const title = <Text style={{ fontWeight: 700, fontSize: pt(t.entryHeaderFontSizePt), flexGrow: 1, flexShrink: 1 }}>
      {href ? <Link href={href} style={linkStyle}>{entry.title}{d.linkStyle.showIcon ? ' ↗' : ''}</Link> : entry.title}
    </Text>
    return <View wrap={false} style={{ marginBottom: pt(3) }}>
      <View style={{ flexDirection: 'row', gap: pt(8), alignItems: 'flex-start' }}>
        {d.entryLayout.dateDisplayMode === 'left' && date}
        {title}
        {d.entryLayout.dateDisplayMode === 'right' && date}
      </View>
      {d.entryLayout.dateDisplayMode === 'belowTitle' && date}
      {entry.subtitle && <Text style={{ fontStyle: d.entryLayout.subtitleStyle, marginTop: pt(2) }}>{entry.subtitle}</Text>}
    </View>
  }
  function displayEntries(section: ContentSection) {
    const mode = d.sectionDisplay[section.type as keyof typeof d.sectionDisplay] ?? 'text'
    if (mode === 'text') return <Text>{section.entries.map((e) => textOf(e)).join('  ·  ')}</Text>
    if (mode === 'bullets') return <UnorderedList style={{ paddingLeft: pt(15) }}>{section.entries.map((entry) => <ListItem key={entry.key} style={{ marginBottom: pt(d.spacing.bulletGap) }}><Text>{textOf(entry)}</Text></ListItem>)}</UnorderedList>
    return <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: pt(8) }}>{section.entries.map((entry) => <Text key={entry.key} style={{ width: '46%' }}>{textOf(entry)}</Text>)}</View>
  }
  function section(section: ContentSection) {
    const display = ['skills', 'languages', 'certifications'].includes(section.type)
    const first = section.entries[0]
    return <View key={section.key} bookmark={section.key} style={{ marginBottom: pt(d.spacing.sectionGap) }}>
      {/* Keep a heading with its first entry header, but allow long bodies to flow. */}
      <View wrap={false}>
        <H2 bookmark={section.key} style={headingStyle}>{section.title}</H2>
        {!display && section.type !== 'summary' && first && entryHead(first)}
      </View>
      {display ? displayEntries(section) : section.entries.map((entry, index) => <View key={entry.key} style={{ marginBottom: index < section.entries.length - 1 ? pt(d.spacing.entryGap) : 0 }}>
        {index > 0 && entryHead(entry)}
        {entry.bodyHtml ? richBlocks(parseDocument(entry.bodyHtml).children, pt(d.spacing.bulletGap), linkStyle) : null}
        {entry.description && <Text style={{ minOrphanLines: 2, minWidowLines: 2 }}>{entry.description}</Text>}
      </View>)}
    </View>
  }
  return {
    page: { size: d.page.format, margin: margins, style: { fontFamily: t.fontFamily, fontSize: pt(t.baseFontSizePt), lineHeight: t.lineHeight, color: d.colors.text, backgroundColor: d.colors.background } },
    header: header(),
    section,
    footer: footerEnabled && <Fixed position="footer" style={{ fontSize: pt(9), color: d.colors.text, paddingTop: 6 }}>
        <Text>{[d.footer.showName && content.header.fullName, d.footer.showEmail && content.header.email, d.footer.showPageNumbers && '{{pageNumber}} / {{totalPages}}'].filter(Boolean).join('  ·  ')}</Text>
      </Fixed>,
  }
}
