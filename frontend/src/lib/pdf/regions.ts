import type { ElementInfo, LayoutInfo } from '@formepdf/core'
import type { ResumeContent } from '@/templates/contract'
import type { PdfHitRegion } from './types'

/** Stable section bookmarks identify blocks, including sections with duplicate titles. */
export function pdfHitRegions(layout: LayoutInfo, content: ResumeContent): PdfHitRegion[] {
  const labels = new Map(content.sections.map((s) => [s.key, s.title]))
  labels.set('contact', 'Contact & Summary')
  const regions: PdfHitRegion[] = []
  const carried: PdfHitRegion[] = []
  layout.pages.forEach((page, pageIndex) => {
    const current: PdfHitRegion[] = []
    const headings: PdfHitRegion[] = []
    const lines: ElementInfo[] = []
    const visit = (node: ElementInfo) => {
      if ((node.nodeType === 'H1' || node.nodeType === 'H2') && 'bookmark' in node && typeof node.bookmark === 'string' && labels.has(node.bookmark)) {
        headings.push({ page: pageIndex, section: node.bookmark, label: labels.get(node.bookmark)!, x: node.x, y: node.y, width: node.width, height: node.height })
      }
      if (node.nodeType === 'TextLine' && node.y >= page.contentY && node.y < page.contentY + page.contentHeight) lines.push(node)
      if (node.nodeType === 'View') {
        const marker = node.children.find((child) => child.nodeType === 'Bookmark')
        const key = marker && 'bookmark' in marker && typeof marker.bookmark === 'string' ? marker.bookmark : undefined
        if (key && labels.has(key)) {
          const x = Math.max(page.contentX, node.x)
          const y = Math.max(page.contentY, node.y)
          const width = Math.min(node.x + node.width, page.contentX + page.contentWidth) - x
          const height = Math.min(node.y + node.height, page.contentY + page.contentHeight) - y
          if (width > 0 && height > 0) current.push({ page: pageIndex, section: key, label: labels.get(key)!, x, y, width, height })
        }
      }
      node.children.forEach(visit)
    }
    page.elements.forEach(visit)
    // Overflowing Views may be flattened by the engine. Their heading keeps its ID.
    for (const heading of headings) {
      if (current.some((r) => r.section === heading.section)) continue
      const nextY = [...current, ...headings].filter((r) => r.y > heading.y + 1 && Math.abs(r.x - heading.x) < 8).sort((a, b) => a.y - b.y)[0]?.y
      current.push({ ...heading, height: (nextY ?? page.contentY + page.contentHeight) - heading.y })
    }
    // Fragmented containers only carry their bookmark on the first page in 0.29.
    // Extend the previous column's last section over actual continuation text,
    // stopping at the next section. Empty columns get no invisible edit target.
    for (const previous of carried) {
      const next = current.filter((r) => Math.abs(r.x - previous.x) < 8).sort((a, b) => a.y - b.y)[0]
      const continuation = lines.filter((line) => line.x >= previous.x - 1 && line.x + line.width <= previous.x + previous.width + 1 && line.y < (next?.y ?? Infinity))
      if (!continuation.length) continue
      const y = Math.min(...continuation.map((line) => line.y))
      const bottom = Math.min(page.contentY + page.contentHeight, Math.max(...continuation.map((line) => line.y + line.height)))
      current.push({ ...previous, page: pageIndex, y, height: bottom - y })
    }
    for (const region of [...current].sort((a, b) => a.y - b.y)) {
      const index = carried.findIndex((r) => Math.abs(r.x - region.x) < 8)
      if (index < 0) carried.push(region)
      else carried[index] = region
    }
    regions.push(...current)
  })
  return regions
}
