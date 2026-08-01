import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'

export type PaginationColumn = 'left' | 'right' | 'full'

export interface PaginationBlock {
  key: string
  column: PaginationColumn
  node: ReactNode
}

export interface PageAssignment {
  key: string
  column: PaginationColumn
}

/**
 * Packs blocks into pages by measured height, independently per column, with
 * "full" blocks forcing a synchronization point (both columns advance to
 * whichever is currently taller before the full block is placed). A page
 * closes for BOTH columns the moment either overflows — pages are always a
 * shared physical unit, even though packing height is tracked per column.
 *
 * Single-column templates (the only ones built so far) tag every block
 * 'left' and leave 'right' empty, which reduces this to the same linear
 * greedy-pack behavior the original single-column engine used — this is a
 * structural generalization, not a behavior change, for that case.
 */
export function useColumnPagination(blocks: PaginationBlock[], contentHeight: number, onReady?: () => void) {
  const measureRefs = useRef(new Map<string, HTMLDivElement>())
  const [pages, setPages] = useState<PageAssignment[][]>(() => [blocks.map((b) => ({ key: b.key, column: b.column }))])

  useLayoutEffect(() => {
    let cancelled = false

    async function recompute() {
      // Web fonts can reflow text after layout has already been measured —
      // waiting here keeps the pagination measurements (and the PDF
      // exporter's data-print-ready gate, which depends on onReady firing
      // only once layout is final) honest once non-system fonts are in use.
      if (typeof document !== 'undefined' && document.fonts && document.fonts.status !== 'loaded') {
        try {
          await document.fonts.ready
        } catch {
          // best-effort — proceed with whatever layout is current
        }
      }
      if (cancelled) return

      const newPages: PageAssignment[][] = []
      let current: PageAssignment[] = []
      let leftY = 0
      let rightY = 0

      for (const b of blocks) {
        const h = measureRefs.current.get(b.key)?.getBoundingClientRect().height ?? 0
        const y = b.column === 'left' ? leftY : b.column === 'right' ? rightY : Math.max(leftY, rightY)

        if (current.length > 0 && y + h > contentHeight) {
          newPages.push(current)
          current = []
          leftY = 0
          rightY = 0
        }

        const placedY = b.column === 'left' ? leftY : b.column === 'right' ? rightY : Math.max(leftY, rightY)
        current.push({ key: b.key, column: b.column })
        if (b.column === 'left') leftY = placedY + h
        else if (b.column === 'right') rightY = placedY + h
        else {
          leftY = placedY + h
          rightY = placedY + h
        }
      }
      if (current.length > 0) newPages.push(current)

      if (cancelled) return
      setPages(newPages.length > 0 ? newPages : [[]])
      onReady?.()
    }

    recompute()
    const observer = new ResizeObserver(() => recompute())
    measureRefs.current.forEach((el) => observer.observe(el))
    return () => {
      cancelled = true
      observer.disconnect()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [blocks, contentHeight])

  function setMeasureRef(key: string) {
    return (el: HTMLDivElement | null) => {
      if (el) measureRefs.current.set(key, el)
      else measureRefs.current.delete(key)
    }
  }

  return { pages, setMeasureRef }
}
