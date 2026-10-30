import { describe, expect, it } from 'vitest'
import { assistantLibrary, assistantPromptOptions } from './library'
import prompt from '../../../../../backend/internal/agent/openui_prompt.txt?raw'

describe('OpenUI catalog', () => {
  it('keeps the embedded Go prompt aligned with the component catalog', () => {
    expect(prompt.trim()).toBe(assistantLibrary.prompt(assistantPromptOptions).trim())
  })
})
