import { describe, expect, it } from 'vitest'
import { readFile } from 'node:fs/promises'
import { renderDocumentWithLayout, type ElementInfo } from '@formepdf/core'
import { ResumeDocument } from './ResumeDocument'
import { defaultResumeDesign } from '@/lib/defaultResumeDesign'
import { PDF_PAGE_SIZES, safePdfLink, type PdfInput } from './types'
import { pdfHitRegions } from './regions'

function input(): PdfInput {
  return {
    rendererKey: 'vertical', design: structuredClone(defaultResumeDesign),
    content: { header: { fullName: 'Alex Morgan', headline: 'Software Engineer', email: 'alex@example.com', phone: null, location: 'London', links: [{ label: 'Portfolio', url: 'example.com' }] }, sections: [
      { key: 'summary', type: 'summary', title: 'Summary', entries: [{ key: 'summary', description: 'Builds thoughtful, reliable software.' }] },
      { key: 'work_experience', type: 'work_experience', title: 'Experience', entries: [{ key: 'work-1', title: 'Engineer · Acme', subtitle: 'Remote', dateLabel: '2020 – Present', bodyHtml: '<p>Built <strong>reliable</strong> tools with <em>care</em>.</p><ul><li><p>Improved performance by 30%.</p></li></ul>' }] },
    ] },
  }
}
function allText(nodes: ElementInfo[]): string {
  return nodes.map((n) => (n.textContent ?? '') + ' ' + allText(n.children)).join(' ')
}
const font = await readFile(new URL('../../../public/fonts/pdf/gelasio-400-normal.ttf', import.meta.url))
const fonts = [{ family: 'Georgia', src: new Uint8Array(font) }]

describe('client resume PDF', () => {
  it('renders real Letter pages with searchable content and rich text', async () => {
    const value = input()
    value.design.page.format = 'Letter'
    const result = await renderDocumentWithLayout(<ResumeDocument input={value} fonts={fonts} />)
    expect(new TextDecoder().decode(result.pdf.slice(0, 5))).toBe('%PDF-')
    expect(result.layout.pages).toHaveLength(1)
    expect(result.layout.pages[0].width).toBe(PDF_PAGE_SIZES.Letter.width)
    expect(result.layout.pages[0].height).toBe(PDF_PAGE_SIZES.Letter.height)
    const text = allText(result.layout.pages[0].elements)
    expect(text).toContain('Alex Morgan')
    expect(text).toContain('Improved performance by 30%.')
    expect(text).toContain('reliable')
    expect(pdfHitRegions(result.layout, input().content).map((r) => r.section)).toEqual(['contact', 'summary', 'work_experience'])
  })
  it('flows long vertical content across A4 pages without losing the last entry', async () => {
    const value = input()
    value.design.page.format = 'A4'
    value.design.footer.showPageNumbers = true
    const entry = value.content.sections[1].entries[0]
    value.content.sections[1].entries = Array.from({ length: 26 }, (_, i) => ({ ...entry, key: `entry-${i}`, title: `Engineer ${i}`, bodyHtml: '<p>' + 'Built resilient systems with clear documentation. '.repeat(7) + `Final marker ${i}.</p>` }))
    const result = await renderDocumentWithLayout(<ResumeDocument input={value} fonts={fonts} />)
    expect(result.layout.pages.length).toBeGreaterThan(1)
    expect(result.layout.pages[0].height).toBeCloseTo(PDF_PAGE_SIZES.A4.height, 1)
    expect(result.layout.pages.map((p) => allText(p.elements)).join(' ').replace(/\s+/g, ' ')).toContain('Final marker 25.')
    expect(pdfHitRegions(result.layout, value.content).some((r) => r.page === result.layout.pages.length - 1 && r.section === 'work_experience')).toBe(true)
    for (const page of result.layout.pages) {
      const check = (nodes: ElementInfo[]) => nodes.forEach((n) => {
        if (n.nodeType === 'TextLine') {
          expect(n.y).toBeGreaterThanOrEqual(0)
          expect(n.y + n.height).toBeLessThanOrEqual(page.height + 1)
        }
        check(n.children)
      })
      check(page.elements)
    }
  })
  it('rejects unsafe links and preserves supported URI schemes', () => {
    expect(safePdfLink('javascript:alert(1)')).toBeUndefined()
    expect(safePdfLink('data:text/html,test')).toBeUndefined()
    expect(safePdfLink('example.com')).toBe('https://example.com')
    expect(safePdfLink('mailto:alex@example.com')).toBe('mailto:alex@example.com')
  })
  it('keeps duplicate section titles selectable by their own IDs and retains certificate dates', async () => {
    const value = input()
    value.content.sections = [
      { key: 'custom-one', type: 'custom', title: 'Highlights', entries: [{ key: 'one', description: 'First section.' }] },
      { key: 'custom-two', type: 'custom', title: 'Highlights', entries: [{ key: 'two', description: 'Second section.' }] },
      { key: 'certifications', type: 'certifications', title: 'Certificates', entries: [{ key: 'certificate', title: 'Cloud Engineer', dateLabel: 'Oct 2026' }] },
    ]
    const result = await renderDocumentWithLayout(<ResumeDocument input={value} fonts={fonts} />)
    const regions = pdfHitRegions(result.layout, value.content)
    expect(regions.map((r) => r.section)).toContain('custom-one')
    expect(regions.map((r) => r.section)).toContain('custom-two')
    expect(allText(result.layout.pages[0].elements)).toContain('Oct 2026')
  })
})
