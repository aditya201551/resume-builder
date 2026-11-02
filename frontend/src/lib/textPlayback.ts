// Pace presentation independently of network chunks. Complete words keep their
// original whitespace and Markdown; partial words wait for the next chunk.
export const CHAT_WORD_INTERVAL_MS = 50

type Step = { text: string } | { run: () => void }

export function createTextPlayback(onText: (text: string) => void, immediate: () => boolean) {
  let queue: Step[] = []
  let pending = ''
  let timer: ReturnType<typeof setTimeout> | undefined
  let closed = false
  let cancelled = false
  let settle: (() => void) | undefined
  let completion: Promise<void> | undefined

  function pump() {
    if (cancelled || timer !== undefined) return
    while (queue.length && ('run' in queue[0] || immediate())) {
      const step = queue.shift()!
      if ('run' in step) step.run()
      else onText(step.text)
    }
    if (!queue.length) {
      if (closed) settle?.()
      return
    }
    timer = setTimeout(() => {
      timer = undefined
      const step = queue.shift()
      if (step && 'text' in step) onText(step.text)
      pump()
    }, CHAT_WORD_INTERVAL_MS)
  }

  function flushPartial() {
    if (pending) queue.push({ text: pending })
    pending = ''
  }

  return {
    write(text: string) {
      if (closed || cancelled || !text) return
      pending += text
      let word: RegExpExecArray | null
      while ((word = /^\s*\S+\s+/u.exec(pending))) {
        queue.push({ text: word[0] })
        pending = pending.slice(word[0].length)
      }
      // Reduced-motion mode must not wait for a word boundary either.
      if (immediate()) flushPartial()
      pump()
    },
    boundary(run: () => void) {
      if (closed || cancelled) return
      flushPartial()
      queue.push({ run })
      pump()
    },
    finish() {
      if (completion) return completion
      completion = new Promise<void>((resolve) => { settle = resolve })
      closed = true
      flushPartial()
      pump()
      if (cancelled) settle?.()
      return completion
    },
    cancel() {
      cancelled = true
      if (timer !== undefined) clearTimeout(timer)
      timer = undefined
      queue = []
      pending = ''
      settle?.()
    },
  }
}
