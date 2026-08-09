import type { AgentProposal, AgentToolEvent } from '@/lib/agentChat'
import type { DraftAction, FlatKind } from '@/hooks/resumeDraftReducer'
import type { AssistantPart } from '@/hooks/useAgentChat'
import type { FullResume } from '@/types/resume'
import type { ResumeDesign } from '@/types/design'

export function proposalToDraftAction(p: AgentProposal): DraftAction | null {
  switch (p.type) {
    case 'flat_create':
      if (!p.entity || !p.tempId) return null
      return { type: 'flat_create', entity: p.entity as FlatKind, tempId: p.tempId, fields: p.fields ?? {} }
    case 'flat_update':
      if (!p.entity || !p.id) return null
      return { type: 'flat_update', entity: p.entity as FlatKind, id: p.id, patch: p.patch ?? {} }
    case 'flat_delete':
      if (!p.entity || !p.id) return null
      return { type: 'flat_delete', entity: p.entity as FlatKind, id: p.id }

    case 'skill_group_create':
      if (!p.tempId) return null
      return { type: 'skill_group_create', tempId: p.tempId, fields: p.fields ?? {} }
    case 'skill_group_update':
      if (!p.id) return null
      return { type: 'skill_group_update', id: p.id, patch: p.patch ?? {} }
    case 'skill_group_delete':
      if (!p.id) return null
      return { type: 'skill_group_delete', id: p.id }

    case 'skill_item_create':
      if (!p.groupId || !p.tempId) return null
      return { type: 'skill_item_create', groupId: p.groupId, tempId: p.tempId, fields: p.fields ?? {} }
    case 'skill_item_update':
      if (!p.groupId || !p.id) return null
      return { type: 'skill_item_update', groupId: p.groupId, id: p.id, patch: p.patch ?? {} }
    case 'skill_item_delete':
      if (!p.groupId || !p.id) return null
      return { type: 'skill_item_delete', groupId: p.groupId, id: p.id }

    case 'custom_section_create':
      if (!p.tempId) return null
      return { type: 'custom_section_create', tempId: p.tempId, fields: p.fields ?? {} }
    case 'custom_section_update':
      if (!p.id) return null
      return { type: 'custom_section_update', id: p.id, patch: p.patch ?? {} }
    case 'custom_section_delete':
      if (!p.id) return null
      return { type: 'custom_section_delete', id: p.id }

    case 'custom_entry_create':
      if (!p.sectionId || !p.tempId) return null
      return { type: 'custom_entry_create', sectionId: p.sectionId, tempId: p.tempId, fields: p.fields ?? {} }
    case 'custom_entry_update':
      if (!p.sectionId || !p.id) return null
      return { type: 'custom_entry_update', sectionId: p.sectionId, id: p.id, patch: p.patch ?? {} }
    case 'custom_entry_delete':
      if (!p.sectionId || !p.id) return null
      return { type: 'custom_entry_delete', sectionId: p.sectionId, id: p.id }

    case 'update_meta':
      return { type: 'update_meta', patch: p.patch ?? {} }

    case 'design_update':
      return { type: 'design_update', patch: (p.designPatch ?? {}) as Partial<ResumeDesign> }

    default:
      return null
  }
}

export type ProposalApplicability = { ok: true } | { ok: false; reason: string }

const APPLICABLE: ProposalApplicability = { ok: true }

function blocked(reason: string): ProposalApplicability {
  return { ok: false, reason }
}

export function canApplyProposal(
  p: AgentProposal,
  draft: FullResume,
  justCreatedParentIds: ReadonlySet<string> = new Set(),
): ProposalApplicability {
  const flatEntity = (): ProposalApplicability => {
    const list = draft[p.entity as FlatKind] as { id: string }[] | undefined
    if (!list) return blocked(`Unknown section "${p.entity}"`)
    if (!list.some((i) => i.id === p.id)) return blocked('The entry this changes no longer exists')
    return APPLICABLE
  }

  const group = () => draft.skill_groups.find((g) => g.id === p.groupId)
  const section = () => draft.custom_sections.find((s) => s.id === p.sectionId)
  const justCreated = (id: string | undefined) => !!id && justCreatedParentIds.has(id)

  switch (p.type) {
    case 'flat_update':
    case 'flat_delete':
      return flatEntity()

    case 'skill_group_update':
    case 'skill_group_delete':
      return draft.skill_groups.some((g) => g.id === p.id) ? APPLICABLE : blocked('That skill group no longer exists')

    case 'skill_item_create':
      return group() || justCreated(p.groupId) ? APPLICABLE : blocked('Accept the skill group this belongs to first')
    case 'skill_item_update':
    case 'skill_item_delete': {
      const g = group()
      if (!g) return blocked('That skill group no longer exists')
      return g.items.some((i) => i.id === p.id) ? APPLICABLE : blocked('That skill no longer exists')
    }

    case 'custom_section_update':
    case 'custom_section_delete':
      return draft.custom_sections.some((s) => s.id === p.id) ? APPLICABLE : blocked('That section no longer exists')

    case 'custom_entry_create':
      return section() || justCreated(p.sectionId) ? APPLICABLE : blocked('Accept the section this belongs to first')
    case 'custom_entry_update':
    case 'custom_entry_delete': {
      const s = section()
      if (!s) return blocked('That section no longer exists')
      return s.entries.some((e) => e.id === p.id) ? APPLICABLE : blocked('That entry no longer exists')
    }

    default:
      return APPLICABLE
  }
}

const ENTITY_LABELS: Record<string, string> = {
  work_experiences: 'work experience',
  educations: 'education',
  projects: 'project',
  certifications: 'certification',
  languages: 'language',
  misc_entries: 'entry',
}

function pickTitle(fields: Record<string, unknown> | undefined): string | null {
  if (!fields) return null
  for (const key of ['title', 'company', 'name', 'institution', 'group_name']) {
    const v = fields[key]
    if (typeof v === 'string' && v.trim()) return v
  }
  return null
}

export function describeProposal(p: AgentProposal): string {
  const entityLabel = p.entity ? (ENTITY_LABELS[p.entity] ?? p.entity) : null
  const title = pickTitle(p.fields) ?? pickTitle(p.patch)

  switch (p.type) {
    case 'flat_create':
      return title ? `Add ${entityLabel}: ${title}` : `Add a new ${entityLabel}`
    case 'flat_update':
      return title ? `Update ${entityLabel}: ${title}` : `Update ${entityLabel}`
    case 'flat_delete':
      return `Remove ${entityLabel}`
    case 'skill_group_create':
      return `Add skill group${title ? `: ${title}` : ''}`
    case 'skill_group_update':
      return 'Update skill group'
    case 'skill_group_delete':
      return 'Remove skill group'
    case 'skill_item_create':
      return `Add skill${title ? `: ${title}` : ''}`
    case 'skill_item_update':
      return 'Update skill'
    case 'skill_item_delete':
      return 'Remove skill'
    case 'custom_section_create':
      return `Add section${title ? `: ${title}` : ''}`
    case 'custom_section_update':
      return 'Update section'
    case 'custom_section_delete':
      return 'Remove section'
    case 'custom_entry_create':
      return `Add entry${title ? `: ${title}` : ''}`
    case 'custom_entry_update':
      return 'Update entry'
    case 'custom_entry_delete':
      return 'Remove entry'
    case 'update_meta':
      return 'Update contact info / summary'
    case 'design_update':
      return 'Update design settings'
    default:
      return p.type
  }
}

export function proposalDetailEntries(p: AgentProposal): [string, unknown][] {
  if (p.type === 'design_update') return Object.entries(p.designUpdates ?? {})
  const data = p.fields ?? p.patch ?? {}
  return Object.entries(data).filter(([k]) => k !== 'id' && k !== 'sort_order')
}

export type DisplayItem =
  | { kind: 'text'; key: string; text: string }
  | { kind: 'tool'; key: string; event: AgentToolEvent }
  | { kind: 'action'; key: string; event: AgentToolEvent; part: Extract<AssistantPart, { kind: 'proposal' }> }

export function groupAssistantParts(parts: AssistantPart[]): DisplayItem[] {
  const proposalByToolCallId = new Map<string, Extract<AssistantPart, { kind: 'proposal' }>>()
  for (const part of parts) {
    if (part.kind === 'proposal' && part.proposal.toolCallId) {
      proposalByToolCallId.set(part.proposal.toolCallId, part)
    }
  }

  const consumed = new Set<string>()
  const items: DisplayItem[] = []

  for (const part of parts) {
    if (part.kind === 'text') {
      items.push({ kind: 'text', key: `text-${items.length}`, text: part.text })
      continue
    }
    if (part.kind === 'tool') {
      const matched = part.event.id ? proposalByToolCallId.get(part.event.id) : undefined
      if (matched) {
        consumed.add(matched.key)
        items.push({ kind: 'action', key: part.key, event: part.event, part: matched })
      } else {
        items.push({ kind: 'tool', key: part.key, event: part.event })
      }
      continue
    }
    if (!consumed.has(part.key)) {
      items.push({ kind: 'action', key: part.key, event: { name: part.proposal.type, status: 'done' }, part })
    }
  }

  return items
}
