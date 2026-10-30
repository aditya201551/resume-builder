import { describe, expect, it } from 'vitest'
import { createDesignHistory } from './designHistory'
import { defaultResumeDesign } from './defaultResumeDesign'
import { restoreTemplateDefaults } from '@/templates/catalog'

describe('formatting history', () => {
  it('groups consecutive slider changes and restores the original value', () => {
    const history = createDesignHistory()
    const original = structuredClone(defaultResumeDesign)
    const first = { ...original, typography: { ...original.typography, lineHeight: 1.3 } }
    const second = { ...first, typography: { ...first.typography, lineHeight: 1.4 } }
    history.record(original, first, 1000)
    history.record(first, second, 1100)
    expect(history.undo(second)).toEqual(original)
    history.record(second, original, 1200)
    expect(history.canUndo).toBe(false)
    expect(history.redo(original)).toEqual(second)
  })
  it('separates different formatting controls and invalidates redo after a new edit', () => {
    const history = createDesignHistory()
    const original = structuredClone(defaultResumeDesign)
    const first = { ...original, typography: { ...original.typography, lineHeight: 1.3 } }
    const second = { ...first, typography: { ...first.typography, baseFontSizePt: 15 } }
    history.record(original, first, 1000)
    history.record(first, second, 1100)
    expect(history.undo(second)).toEqual(first)
    history.record(second, first, 1200)
    const third = { ...first, page: { ...first.page, format: first.page.format === 'A4' ? 'Letter' as const : 'A4' as const } }
    history.record(first, third, 1300)
    expect(history.canRedo).toBe(false)
    expect(history.undo(third)).toEqual(first)
  })
  it('keeps restoring defaults separate from rapid edits so one undo restores the customized design', () => {
    const history = createDesignHistory()
    const original = structuredClone(defaultResumeDesign)
    const customized = { ...original, typography: { ...original.typography, fontFamily: 'Roboto' } }
    const restored = restoreTemplateDefaults(customized)
    history.record(original, customized, 1000)
    history.record(customized, restored, 1100, { separate: true })
    expect(history.undo(restored)).toEqual(customized)
    history.record(restored, customized, 1200)
    expect(history.redo(customized)).toEqual(restored)
    history.record(customized, restored, 1300)
    const editedAgain = { ...restored, typography: { ...restored.typography, fontFamily: 'Inter' } }
    history.record(restored, editedAgain, 1400)
    expect(history.undo(editedAgain)).toEqual(restored)
  })
})
