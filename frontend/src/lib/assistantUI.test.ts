import { describe, expect, it } from 'vitest'
import { isAssistantUI, uiReviewProposal, type AssistantUI } from './assistantUI'
import { canApplyProposal, groupAssistantParts } from './agentProposal'
import type { FullResume } from '@/types/resume'

const draft = {
  resume: { id: 'r1', summary: 'Original summary', headline: 'Engineer' },
  work_experiences: [{ id: 'real-w1', content: 'Built APIs' }], projects: [],
} as unknown as FullResume
const rewrite: AssistantUI = { id: 'u1', version: 1, kind: 'rewrite_comparison', code: 'root = AssistantPanel("Compare", [choices])', fallback: 'Choose a summary', target: { entity: 'resume', id: 'r1', field: 'summary', original: 'Original summary' } }
const form: AssistantUI = { id: 'u2', version: 1, kind: 'experience_form', code: 'root = AssistantPanel("Experience", [form])', fallback: 'Add experience' }

describe('interactive resume actions', () => {
  it('accepts versioned UI envelopes with scoped rewrite fields', () => {
    expect(isAssistantUI(rewrite)).toBe(true)
    expect(isAssistantUI(form)).toBe(true)
    expect(isAssistantUI({ ...rewrite, version: 99 })).toBe(false)
    expect(isAssistantUI({ ...rewrite, target: { ...rewrite.target, field: 'user_id' } })).toBe(false)
    expect(isAssistantUI({ ...rewrite, target: undefined })).toBe(false)
    expect(isAssistantUI({ ...form, code: 'x'.repeat(16001) })).toBe(false)
  })

  it('creates a scoped proposal without mutating the resume', () => {
    const result = uiReviewProposal(rewrite, { type: 'rewrite', text: 'Clearer summary' }, draft, () => 'unused')
    expect(result).toEqual({ proposal: { type: 'update_meta', patch: { summary: 'Clearer summary' } } })
    expect(draft.resume.summary).toBe('Original summary')
  })

  it('blocks stale and deleted rewrite sources at selection and acceptance', () => {
    const changed = { ...draft, resume: { ...draft.resume, summary: 'A manual edit' } }
    expect(uiReviewProposal(rewrite, { type: 'rewrite', text: 'Clearer summary' }, changed, () => '')).toHaveProperty('error')
    expect(canApplyProposal({ type: 'update_meta', patch: { summary: 'Clearer' }, sourceGuard: rewrite.target }, changed).ok).toBe(false)
    const work = { ...rewrite, target: { entity: 'work_experiences' as const, id: 'temp-w1', field: 'content' as const, original: 'Built APIs' } }
    const result = uiReviewProposal(work, { type: 'rewrite', text: 'Built reliable APIs' }, draft, () => '', () => 'real-w1')
    expect(result).toMatchObject({ proposal: { id: 'real-w1', entity: 'work_experiences' } })
    expect(uiReviewProposal(work, { type: 'rewrite', text: 'Built APIs' }, { ...draft, work_experiences: [] }, () => '')).toHaveProperty('error')
  })

  it('validates experience fields and returns an ordinary create proposal', () => {
    const values = { company: 'Acme', title: 'Engineer', location: 'Remote', startDate: '2024-02', endDate: '', current: true, content: 'Built an editor.' }
    const result = uiReviewProposal(form, { type: 'experience', values }, draft, () => 'temp-new')
    expect(result).toMatchObject({ proposal: { type: 'flat_create', entity: 'work_experiences', tempId: 'temp-new', fields: { company: 'Acme', title: 'Engineer', start_date: '2024-02-01T00:00:00Z', end_date: null, is_current: true, technologies: [], sort_order: 1 } } })
    expect(uiReviewProposal(form, { type: 'experience', values: { ...values, title: '' } }, draft, () => '')).toHaveProperty('error')
    expect(uiReviewProposal(form, { type: 'experience', values: { ...values, startDate: '2024-13' } }, draft, () => '')).toHaveProperty('error')
    expect(uiReviewProposal(form, { type: 'experience', values: { ...values, current: false, endDate: '2023-01' } }, draft, () => '')).toHaveProperty('error')
    expect(draft.work_experiences).toHaveLength(1)
  })

  it('keeps cards distinct from mutations and tool activity', () => {
    const parts = groupAssistantParts([
      { kind: 'text', text: 'Choose an option.' },
      { kind: 'ui', key: 'u1', ui: rewrite, formState: {} },
      { kind: 'proposal', key: 'p1', proposal: { type: 'update_meta', patch: { summary: 'Chosen' } }, status: 'pending' },
    ])
    expect(parts.map((p) => p.kind)).toEqual(['text', 'ui', 'action'])
  })
})
