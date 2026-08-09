
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
