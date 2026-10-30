import type { AgentProposal } from './agentChat'
import type { FullResume } from '@/types/resume'

export interface RewriteTarget {
  entity: 'resume' | 'work_experiences' | 'projects'
  id: string
  field: 'summary' | 'headline' | 'content'
  original: string
}

export interface AssistantUI {
  id: string
  version: 1
  kind: 'experience_form' | 'rewrite_comparison'
  code: string
  fallback: string
  target?: RewriteTarget
}

export type UIFormState = Record<string, string | boolean>
export type UIReviewAction =
  | { type: 'experience'; values: UIFormState }
  | { type: 'rewrite'; text: string }

export function isAssistantUI(value: unknown): value is AssistantUI {
  if (!value || typeof value !== 'object') return false
  const ui = value as AssistantUI
  if (typeof ui.id !== 'string' || !ui.id || ui.id.length > 200 || ui.version !== 1) return false
  if (typeof ui.code !== 'string' || !ui.code || ui.code.length > 16000 || typeof ui.fallback !== 'string' || ui.fallback.length > 1000) return false
  if (ui.kind === 'experience_form') return !ui.target
  const target = ui.target
  return ui.kind === 'rewrite_comparison' && Boolean(target && typeof target.id === 'string' && target.id && typeof target.original === 'string' && target.original.length <= 16000 && (
    target.entity === 'resume' ? ['summary', 'headline'].includes(target.field) : ['work_experiences', 'projects'].includes(target.entity) && target.field === 'content'
  ))
}

export function rewriteSource(target: RewriteTarget, draft: FullResume, resolveId: (id: string) => string = (id) => id): string | undefined {
  if (target.entity === 'resume') return resolveId(target.id) === draft.resume.id ? (draft.resume[target.field as 'summary' | 'headline'] ?? '') : undefined
  return draft[target.entity].find((entry) => entry.id === resolveId(target.id))?.content
}

function monthDate(value: string): string | null | undefined {
  if (!value) return null
  if (!/^\d{4}-(0[1-9]|1[0-2])$/.test(value)) return undefined
  return `${value}-01T00:00:00Z`
}

// UI components never dispatch arbitrary patches. Each supported action becomes
// an ordinary pending proposal, with the same ID resolution and review path.
export function uiReviewProposal(ui: AssistantUI, action: UIReviewAction, draft: FullResume, newId: () => string, resolveId: (id: string) => string = (id) => id): { proposal: AgentProposal } | { error: string } {
  if (ui.kind === 'rewrite_comparison' && action.type === 'rewrite' && ui.target) {
    if (!action.text.trim() || action.text.length > 16000) return { error: 'Enter a rewrite of up to 16,000 characters.' }
    if (rewriteSource(ui.target, draft, resolveId) !== ui.target.original) return { error: 'This text has changed since the comparison was created. Ask for a fresh comparison.' }
    const patch = { [ui.target.field]: action.text }
    return { proposal: ui.target.entity === 'resume' ? { type: 'update_meta', patch } : { type: 'flat_update', entity: ui.target.entity, id: resolveId(ui.target.id), patch } }
  }
  if (ui.kind !== 'experience_form' || action.type !== 'experience') return { error: 'This action is not available for this card.' }
  const text = (key: string) => typeof action.values[key] === 'string' ? (action.values[key] as string).trim() : ''
  const company = text('company'), title = text('title'), location = text('location'), content = text('content')
  if (!company || !title) return { error: 'Add a company and role before reviewing this experience.' }
  if (company.length > 200 || title.length > 200 || location.length > 200 || content.length > 12000) return { error: 'Shorten the company, role or experience details before continuing.' }
  const start = monthDate(text('startDate')), end = monthDate(text('endDate'))
  const current = action.values.current === true
  if (start === undefined || (!current && end === undefined)) return { error: 'Use a valid month and year, or leave unknown dates blank.' }
  if (start && end && !current && end < start) return { error: 'The end date must be after the start date.' }
  return { proposal: { type: 'flat_create', entity: 'work_experiences', tempId: newId(), fields: {
    company, title, location: location || null, content, start_date: start, end_date: current ? null : end,
    is_current: current, company_url: null, employment_type: null, technologies: [], sort_order: draft.work_experiences.length,
  } } }
}
