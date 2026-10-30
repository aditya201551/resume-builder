import type { ResumeDesign } from '@/types/design'

/** History covers formatting only. Content edits and sync acknowledgements cannot be undone here. */
export function createDesignHistory() {
  const past: ResumeDesign[] = []
  const future: ResumeDesign[] = []
  let lastGroup = ''
  let lastTime = 0
  let restoring = false
  return {
    record(before: ResumeDesign, after: ResumeDesign, now = Date.now()) {
      if (restoring) { restoring = false; return }
      if (JSON.stringify(before) === JSON.stringify(after)) return
      const changed = changedPaths(before, after).join(',')
      if (changed !== lastGroup || now - lastTime > 600 || past.length === 0) past.push(structuredClone(before))
      if (past.length > 80) past.shift()
      future.length = 0
      lastGroup = changed
      lastTime = now
    },
    undo(current: ResumeDesign) {
      const previous = past.pop()
      if (!previous) return undefined
      future.push(structuredClone(current))
      restoring = true
      lastGroup = ''
      return previous
    },
    redo(current: ResumeDesign) {
      const next = future.pop()
      if (!next) return undefined
      past.push(structuredClone(current))
      restoring = true
      lastGroup = ''
      return next
    },
    get canUndo() { return past.length > 0 },
    get canRedo() { return future.length > 0 },
  }
}

function changedPaths(before: unknown, after: unknown, prefix = ''): string[] {
  if (JSON.stringify(before) === JSON.stringify(after)) return []
  if (!before || !after || typeof before !== 'object' || typeof after !== 'object' || Array.isArray(after)) return [prefix]
  return Object.keys(after).flatMap((key) => changedPaths((before as Record<string, unknown>)[key], (after as Record<string, unknown>)[key], `${prefix}.${key}`))
}
