import { useLayoutEffect, useRef, type RefObject } from 'react'
import { usePrefersReducedMotion } from '@/hooks/usePrefersReducedMotion'
import { wordReveal } from '@/lib/textMotion'

type Reveal = { word: string; start: number }

// Markdown can replace its HTML on every token. Preserve animation start times
// across those replacements, and only reveal words beyond the seen text frontier.
export function useWordReveal(rootRef: RefObject<HTMLDivElement | null>, html: string, enabled: boolean) {
  const reducedMotion = usePrefersReducedMotion()
  const seen = useRef({ length: 0, active: new Map<number, Reveal>() })

  useLayoutEffect(() => {
    const root = rootRef.current
    if (!root) return
    root.querySelectorAll('[data-reveal-word]').forEach((span) => span.replaceWith(...span.childNodes))
    const length = root.textContent?.length ?? 0
    if (!enabled || reducedMotion || typeof root.animate !== 'function') {
      seen.current = { length, active: new Map() }
      return
    }

    const timelineTime = document.timeline.currentTime
    const now = typeof timelineTime === 'number' ? timelineTime : performance.now()
    const duration = wordReveal.duration * 1000
    const active = new Map<number, Reveal>()
    const animations: Animation[] = []
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
    const nodes: Text[] = []
    while (walker.nextNode()) nodes.push(walker.currentNode as Text)
    let offset = 0
    let added = 0

    for (const node of nodes) {
      const value = node.data
      const nodeOffset = offset
      offset += value.length
      // Leave code intact: wrapping its tokens can change indentation and layout.
      if (node.parentElement?.closest('pre, code')) continue
      const fragment = document.createDocumentFragment()
      let copied = 0
      for (const match of value.matchAll(/\S+/gu)) {
        const position = nodeOffset + match.index
        const word = match[0]
        let reveal = seen.current.active.get(position)
        if (reveal && !word.startsWith(reveal.word)) reveal = undefined
        if (!reveal && position >= seen.current.length) {
          // Cap the stagger so a large response never waits behind an animation queue.
          reveal = { word, start: now + Math.min(added++ * wordReveal.stagger * 1000, 120) }
        }
        if (!reveal || now - reveal.start >= duration) continue
        active.set(position, reveal)
        fragment.append(value.slice(copied, match.index))
        const span = document.createElement('span')
        span.dataset.revealWord = ''
        span.textContent = word
        fragment.append(span)
        copied = match.index + word.length
        const animation = span.animate([
          { opacity: 0, transform: `translateY(${wordReveal.distance}px)` },
          { opacity: 1, transform: 'translateY(0)' },
        ], {
          duration,
          easing: `cubic-bezier(${wordReveal.ease.join(',')})`,
          fill: 'backwards',
        })
        animation.startTime = reveal.start
        animation.onfinish = () => span.replaceWith(...span.childNodes)
        animations.push(animation)
      }
      if (copied) {
        fragment.append(value.slice(copied))
        node.replaceWith(fragment)
      }
    }
    seen.current = { length: Math.max(length, seen.current.length), active }
    return () => animations.forEach((animation) => animation.cancel())
  }, [rootRef, html, enabled, reducedMotion])
}
