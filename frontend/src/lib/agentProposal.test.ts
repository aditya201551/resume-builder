import { describe, expect, it } from 'vitest'
import { canApplyProposal, groupAssistantParts, proposalToDraftAction } from './agentProposal'
import type { AgentProposal } from './agentChat'
import type { AssistantPart } from '@/hooks/useAgentChat'
import type { FullResume, SkillGroup, CustomSection } from '@/types/resume'

function emptyDraft(overrides: Partial<FullResume> = {}): FullResume {
  return {
    resume: {
      id: 'r1',
      user_id: 'u1',
      label: 'Untitled',
      full_name: 'Aditya Raj',
      headline: null,
      email: null,
      phone: null,
      location: null,
      photo_url: null,
      summary: null,
      links: [],
      created_at: '',
      updated_at: '',
      last_exported_at: null,
    },
    work_experiences: [],
    educations: [],
    skill_groups: [],
    projects: [],
    certifications: [],
    languages: [],
    misc_entries: [],
    custom_sections: [],
    section_configs: [],
    ...overrides,
  }
}

function group(id: string, items: SkillGroup['items'] = []): SkillGroup {
  return { id, resume_id: 'r1', group_name: 'Languages', sort_order: 0, items }
}

function customSection(id: string, entries: CustomSection['entries'] = []): CustomSection {
  return { id, resume_id: 'r1', title: 'Extras', sort_order: 0, entries }
}

describe('canApplyProposal', () => {
  it('allows creates that have no parent', () => {
    const p: AgentProposal = { type: 'flat_create', entity: 'projects', tempId: 'temp-1', fields: { name: 'X' } }
    expect(canApplyProposal(p, emptyDraft()).ok).toBe(true)
  })

  it('allows metadata updates unconditionally', () => {
    expect(canApplyProposal({ type: 'update_meta', patch: { full_name: 'A' } }, emptyDraft()).ok).toBe(true)
  })

  // The case the user hit: accepting a skill without its group would be
  // dropped by the reducer while the UI reported success.
  it('blocks a skill item whose group is not in the draft', () => {
    const p: AgentProposal = { type: 'skill_item_create', groupId: 'temp-group', tempId: 'temp-item', fields: { name: 'Go' } }
    const result = canApplyProposal(p, emptyDraft())

    expect(result.ok).toBe(false)
    expect(result.ok === false && result.reason).toMatch(/skill group/i)
  })

  it('allows a skill item once its group exists in the draft', () => {
    const p: AgentProposal = { type: 'skill_item_create', groupId: 'g1', tempId: 'temp-item', fields: { name: 'Go' } }
    expect(canApplyProposal(p, emptyDraft({ skill_groups: [group('g1')] })).ok).toBe(true)
  })

  // dispatch() doesn't synchronously update the draft snapshot, so a group
  // accepted moments earlier in the same batch is tracked separately.
  it('allows a skill item whose group was just created but is not yet in the draft', () => {
    const p: AgentProposal = { type: 'skill_item_create', groupId: 'temp-g', tempId: 'temp-item', fields: { name: 'Go' } }

    expect(canApplyProposal(p, emptyDraft()).ok).toBe(false)
    expect(canApplyProposal(p, emptyDraft(), new Set(['temp-g'])).ok).toBe(true)
  })

  it('blocks a custom entry whose section is missing, and allows it once present', () => {
    const p: AgentProposal = { type: 'custom_entry_create', sectionId: 's1', tempId: 'temp-e', fields: { title: 'X' } }

    expect(canApplyProposal(p, emptyDraft()).ok).toBe(false)
    expect(canApplyProposal(p, emptyDraft({ custom_sections: [customSection('s1')] })).ok).toBe(true)
  })

  it('blocks updates and deletes naming an entry the draft does not have', () => {
    const draft = emptyDraft({
      projects: [{ id: 'p1', resume_id: 'r1', name: 'A', content: '', role: null, technologies: [], url: null, start_date: null, end_date: null, sort_order: 0 }],
    })

    expect(canApplyProposal({ type: 'flat_update', entity: 'projects', id: 'p1', patch: { name: 'B' } }, draft).ok).toBe(true)
    expect(canApplyProposal({ type: 'flat_update', entity: 'projects', id: 'nope', patch: { name: 'B' } }, draft).ok).toBe(false)
    expect(canApplyProposal({ type: 'flat_delete', entity: 'projects', id: 'nope' }, draft).ok).toBe(false)
  })

  it('blocks a skill item update when the item is gone from an existing group', () => {
    const draft = emptyDraft({
      skill_groups: [group('g1', [{ id: 'i1', skill_group_id: 'g1', name: 'Go', proficiency: null, sort_order: 0 }])],
    })

    expect(canApplyProposal({ type: 'skill_item_update', groupId: 'g1', id: 'i1', patch: { name: 'Rust' } }, draft).ok).toBe(true)
    expect(canApplyProposal({ type: 'skill_item_update', groupId: 'g1', id: 'gone', patch: { name: 'Rust' } }, draft).ok).toBe(false)
  })

  it('blocks an unknown entity rather than throwing', () => {
    const result = canApplyProposal({ type: 'flat_update', entity: 'not_a_section', id: 'x', patch: {} }, emptyDraft())
    expect(result.ok).toBe(false)
  })
})

describe('proposalToDraftAction', () => {
  it('returns null for a proposal missing the ids its action requires', () => {
    expect(proposalToDraftAction({ type: 'flat_create', entity: 'projects' })).toBeNull()
    expect(proposalToDraftAction({ type: 'skill_item_create', tempId: 'temp-1' })).toBeNull()
    expect(proposalToDraftAction({ type: 'who_knows' })).toBeNull()
  })
})

function toolPart(id: string, status: 'running' | 'done' | 'error' = 'done'): Extract<AssistantPart, { kind: 'tool' }> {
  return { kind: 'tool', key: id, event: { id, name: 'propose_meta_update', status } }
}

function proposalPart(key: string, toolCallId?: string, status: 'pending' | 'accepted' | 'rejected' = 'pending'): Extract<AssistantPart, { kind: 'proposal' }> {
  return { kind: 'proposal', key, status, proposal: { type: 'update_meta', patch: { summary: 'x' }, toolCallId } }
}

describe('groupAssistantParts', () => {
  it('merges a tool part with the proposal sharing its id into one action item', () => {
    const parts: AssistantPart[] = [toolPart('call-1'), proposalPart('p1', 'call-1')]
    const items = groupAssistantParts(parts)

    expect(items).toHaveLength(1)
    expect(items[0]).toMatchObject({ kind: 'action', key: 'call-1' })
  })

  it('keeps a tool part with no matching proposal as a plain tool item', () => {
    const items = groupAssistantParts([{ kind: 'tool', key: 'read', event: { id: 'read-1', name: 'read_resume', status: 'done' } }])
    expect(items).toEqual([{ kind: 'tool', key: 'read', event: { id: 'read-1', name: 'read_resume', status: 'done' } }])
  })

  // The scenario that made position-based pairing unsafe: several propose_*
  // calls run concurrently, so their proposal events can arrive in an order
  // that doesn't match which tool part appears first in the array (in real
  // state, upsertToolEvent merges same-id tool events into one part in
  // place — 'running' becomes 'done' rather than adding a second part — so
  // each id appears once here, as it would in the actual streamingParts).
  it('pairs correctly by id even when proposals arrive out of order relative to their tool calls', () => {
    const parts: AssistantPart[] = [
      toolPart('call-A', 'done'),
      toolPart('call-B', 'done'),
      proposalPart('pB', 'call-B'),
      proposalPart('pA', 'call-A'),
    ]
    const items = groupAssistantParts(parts)

    expect(items).toHaveLength(2)
    const byKey = new Map(items.map((i) => [i.key, i]))
    expect(byKey.get('call-A')).toMatchObject({ kind: 'action', part: { key: 'pA' } })
    expect(byKey.get('call-B')).toMatchObject({ kind: 'action', part: { key: 'pB' } })
  })

  it('still renders a proposal with no matching tool event, as a standalone action', () => {
    const items = groupAssistantParts([proposalPart('orphan', undefined)])
    expect(items).toHaveLength(1)
    expect(items[0]).toMatchObject({ kind: 'action', key: 'orphan', event: { status: 'done' } })
  })

  it('preserves text parts and their position relative to actions', () => {
    const parts: AssistantPart[] = [{ kind: 'text', text: 'Hello' }, toolPart('call-1'), proposalPart('p1', 'call-1')]
    const items = groupAssistantParts(parts)

    expect(items.map((i) => i.kind)).toEqual(['text', 'action'])
  })
})
