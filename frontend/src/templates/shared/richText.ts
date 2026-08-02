// Pure HTML-splitting logic shared across templates — no JSX/CSS coupling,
// unlike each template's block-building helpers (entryBlock, buildSectionBlocks,
// etc.), which stay per-template since they're tied to that template's own
// CSS module and JSX shape (templates are meant to be independent custom
// code, not funneled through one shared renderer).

/** Splits rendered rich-text HTML into one fragment per top-level paragraph
 * or list item, so pagination can flow bullet-by-bullet instead of treating
 * a whole description as one atomic unit. */
export function splitRichTextBlocks(html: string): string[] {
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
