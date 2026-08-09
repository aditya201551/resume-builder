import { describe, expect, it } from 'vitest'
import { markdownToHtml } from './markdown'

describe('markdownToHtml', () => {
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
