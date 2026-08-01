import { describe, expect, it } from 'vitest'
import { markdownToHtml } from './markdown'

// Only the guard clause is covered here: everything past it constructs a
// tiptap Editor, which needs a DOM this suite doesn't run in (vitest is on
// the default node environment). The guard is the regression that matters.
describe('markdownToHtml', () => {
  // A draft entity can reach the preview without a content key — an agent
  // proposal that omitted it, or a localStorage draft from an older schema.
  // This used to throw and take the whole editor down with it.
  it('returns empty string for missing content instead of throwing', () => {
    expect(markdownToHtml(undefined)).toBe('')
    expect(markdownToHtml(null)).toBe('')
    expect(markdownToHtml('')).toBe('')
    expect(markdownToHtml('   ')).toBe('')
  })

  it('does not throw on a non-string value forced through at runtime', () => {
    expect(markdownToHtml(42 as unknown as string)).toBe('')
  })
})
