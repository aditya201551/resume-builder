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

/** Original sequential packer — walks `blocks` in the exact order given,
 * synchronizing both columns whenever a 'full' block appears (both columns
 * advance to whichever is currently taller before it's placed). Used only
 * when the input actually contains 'full' blocks — a page-spanning band, not
 * exercised by any template yet. For pure left/right content, packBalanced
 * below is strictly better: this packer's page-break decisions depend on
 * the caller's interleaving order, which has no reason to track actual
 * rendered height (see packBalanced's comment for what goes wrong). */
function packSequential(blocks: PaginationBlock[], contentHeight: number, height: (key: string) => number): PageAssignment[][] {
  const newPages: PageAssignment[][] = []
  let current: PageAssignment[] = []
  let leftY = 0
  let rightY = 0

  for (const b of blocks) {
    const h = height(b.key)
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
  return newPages
}

/**
 * Height-balanced packer for pure left/right content (no 'full' blocks): at
 * each step, draws the next block from whichever column's cumulative height
 * *so far* is currently smaller, instead of following the caller's combined
 * array order.
 *
 * This matters because a page closes for BOTH columns the moment EITHER
 * overflows (pages are a shared physical unit) — so if blocks are fed in an
 * order that doesn't track real height (e.g. round-robin by block count), a
 * column with very FEW but visually TALL blocks (a narrow sidebar where
 * long text wraps across several lines) can trigger the shared break long
 * before the other column's page is actually full, stranding whatever
 * hadn't been "given a turn" yet in that column and leaving a visible gap
 * for the rest of that page. Balancing by actual measured height instead of
 * input order keeps both columns filling the same page evenly.
 */
function packBalanced(blocks: PaginationBlock[], contentHeight: number, height: (key: string) => number): PageAssignment[][] {
  const left = blocks.filter((b) => b.column === 'left')
  const right = blocks.filter((b) => b.column === 'right')

  const newPages: PageAssignment[][] = []
  let current: PageAssignment[] = []
  let leftY = 0
  let rightY = 0
  let li = 0
  let ri = 0

  while (li < left.length || ri < right.length) {
    const takeLeft = ri >= right.length || (li < left.length && leftY <= rightY)
    const b = takeLeft ? left[li] : right[ri]
    const h = height(b.key)
    const y = takeLeft ? leftY : rightY

    if (current.length > 0 && y + h > contentHeight) {
      newPages.push(current)
      current = []
      leftY = 0
      rightY = 0
      continue
    }

    current.push({ key: b.key, column: b.column })
    if (takeLeft) {
      leftY = y + h
      li++
    } else {
      rightY = y + h
      ri++
    }
  }
  if (current.length > 0) newPages.push(current)
  return newPages
}

/**
 * Packs blocks into pages by measured height, independently per column. A
 * page closes for BOTH columns the moment either overflows — pages are
 * always a shared physical unit, even though packing height is tracked per
 * column.
 *
 * Single-column templates tag every block 'left' and leave 'right' empty —
 * packBalanced's while-loop degenerates to the same linear greedy-pack
 * behavior the original single-column engine used for that case (right is
 * always exhausted, so every pick takes from left in order).
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

      const height = (key: string) => measureRefs.current.get(key)?.getBoundingClientRect().height ?? 0
      const hasFull = blocks.some((b) => b.column === 'full')
      const newPages = hasFull ? packSequential(blocks, contentHeight, height) : packBalanced(blocks, contentHeight, height)

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
