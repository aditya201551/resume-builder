import { describe, expect, it } from 'vitest'
import { assistantMessage } from './assistantTask'

describe('assistant task context', () => {
  it('leaves a task without job context as an ordinary user message', () => {
    expect(assistantMessage('  Improve my summary  ', '  ')).toBe(
      'Improve my summary',
    )
  })

  it('includes the supplied role context and asks for grounded edits', () => {
    const message = assistantMessage(
      'Tailor my resume',
      'Build Go APIs.\nWork with PostgreSQL.',
    )
    expect(message).toContain('Tailor my resume')
    expect(message).toContain(
      '<job_description>\nBuild Go APIs.\nWork with PostgreSQL.\n</job_description>',
    )
    expect(message).toContain(
      'do not invent skills, responsibilities, or achievement metrics',
    )
  })
})
