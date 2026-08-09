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

export function useColumnPagination(blocks: PaginationBlock[], contentHeight: number, onReady?: () => void) {
  const measureRefs = useRef(new Map<string, HTMLDivElement>())
  const [pages, setPages] = useState<PageAssignment[][]>(() => [blocks.map((b) => ({ key: b.key, column: b.column }))])

  useLayoutEffect(() => {
    let cancelled = false

    async function recompute() {
      if (typeof document !== 'undefined' && document.fonts && document.fonts.status !== 'loaded') {
        try {
          await document.fonts.ready
        } catch {
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
