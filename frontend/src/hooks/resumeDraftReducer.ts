import type {
  CustomSection,
  CustomSectionEntry,
  FullResume,
  SkillGroup,
  SkillItem,
} from '@/types/resume'
import type { ResumeDesign } from '@/types/design'

export type FlatKind =
  | 'work_experiences'
  | 'educations'
  | 'projects'
  | 'certifications'
  | 'languages'
  | 'misc_entries'

export type DraftAction =
  | { type: 'replace_all'; data: FullResume }
  | { type: 'hydrate'; data: FullResume; lastSynced: FullResume }
  | { type: 'update_meta'; patch: Record<string, unknown> }
  | { type: 'design_update'; patch: Partial<ResumeDesign> }
  | { type: 'flat_create'; entity: FlatKind; tempId: string; fields: Record<string, unknown> }
  | { type: 'flat_update'; entity: FlatKind; id: string; patch: Record<string, unknown> }
  | { type: 'flat_delete'; entity: FlatKind; id: string }
  | { type: 'flat_reorder'; entity: FlatKind; orderedIds: string[] }
  | { type: 'skill_group_create'; tempId: string; fields: Record<string, unknown> }
  | { type: 'skill_group_update'; id: string; patch: Record<string, unknown> }
  | { type: 'skill_group_delete'; id: string }
  | { type: 'skill_group_reorder'; orderedIds: string[] }
  | { type: 'skill_item_create'; groupId: string; tempId: string; fields: Record<string, unknown> }
  | { type: 'skill_item_update'; groupId: string; id: string; patch: Record<string, unknown> }
  | { type: 'skill_item_delete'; groupId: string; id: string }
  | { type: 'skill_item_reorder'; groupId: string; orderedIds: string[] }
  | { type: 'custom_section_create'; tempId: string; fields: Record<string, unknown> }
  | { type: 'custom_section_update'; id: string; patch: Record<string, unknown> }
  | { type: 'custom_section_delete'; id: string }
  | { type: 'custom_section_reorder'; orderedIds: string[] }
  | { type: 'custom_entry_create'; sectionId: string; tempId: string; fields: Record<string, unknown> }
  | { type: 'custom_entry_update'; sectionId: string; id: string; patch: Record<string, unknown> }
  | { type: 'custom_entry_delete'; sectionId: string; id: string }
  | { type: 'custom_entry_reorder'; sectionId: string; orderedIds: string[] }
  | { type: 'resolve_ids'; idMap: Record<string, string> }
  | { type: 'mark_synced_entity'; slice: keyof FullResume }
  | { type: 'mark_synced_all' }

export interface DraftState {
  data: FullResume
  lastSynced: FullResume
}

function reindex<T extends { id: string; sort_order: number }>(items: T[], orderedIds: string[]): T[] {
  const byId = new Map(items.map((i) => [i.id, i]))
  return orderedIds.map((id, index) => ({ ...byId.get(id)!, sort_order: index }))
}

function updateFlat<T extends { id: string }>(items: T[], id: string, patch: Partial<T>): T[] {
  return items.map((i) => (i.id === id ? { ...i, ...patch } : i))
}

function replaceIdEverywhere(value: unknown, idMap: Record<string, string>): unknown {
  if (Array.isArray(value)) return value.map((v) => replaceIdEverywhere(v, idMap))
  if (value && typeof value === 'object') {
    const out: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(value)) {
      if (typeof v === 'string' && idMap[v] !== undefined) out[k] = idMap[v]
      else out[k] = replaceIdEverywhere(v, idMap)
    }
    return out
  }
  return value
}

export function draftReducer(state: DraftState, action: DraftAction): DraftState {
  if (action.type === 'replace_all') return { data: action.data, lastSynced: action.data }
  if (action.type === 'hydrate') return { data: action.data, lastSynced: action.lastSynced }

  const d = state.data

  switch (action.type) {
    case 'update_meta':
      return { ...state, data: { ...d, resume: { ...d.resume, ...action.patch } } }

    case 'design_update':
      return { ...state, data: { ...d, design: { ...d.design, ...action.patch } } }

    case 'flat_create':
      return {
        ...state,
        data: {
          ...d,
          [action.entity]: [...d[action.entity], { id: action.tempId, resume_id: d.resume.id, ...action.fields }],
        },
      }
    case 'flat_update':
      return { ...state, data: { ...d, [action.entity]: updateFlat(d[action.entity] as { id: string }[], action.id, action.patch) } }
    case 'flat_delete':
      return { ...state, data: { ...d, [action.entity]: (d[action.entity] as { id: string }[]).filter((i) => i.id !== action.id) } }
    case 'flat_reorder':
      return { ...state, data: { ...d, [action.entity]: reindex(d[action.entity] as { id: string; sort_order: number }[], action.orderedIds) } }

    case 'skill_group_create':
      return {
        ...state,
        data: {
          ...d,
          skill_groups: [
            ...d.skill_groups,
            { id: action.tempId, resume_id: d.resume.id, items: [], ...action.fields } as unknown as SkillGroup,
          ],
        },
      }
    case 'skill_group_update':
      return { ...state, data: { ...d, skill_groups: updateFlat(d.skill_groups, action.id, action.patch) } }
    case 'skill_group_delete':
      return { ...state, data: { ...d, skill_groups: d.skill_groups.filter((g) => g.id !== action.id) } }
    case 'skill_group_reorder':
      return { ...state, data: { ...d, skill_groups: reindex(d.skill_groups, action.orderedIds) } }

    case 'skill_item_create':
      return {
        ...state,
        data: {
          ...d,
          skill_groups: d.skill_groups.map((g) =>
            g.id === action.groupId
              ? { ...g, items: [...g.items, { id: action.tempId, skill_group_id: g.id, ...action.fields } as SkillItem] }
              : g,
          ),
        },
      }
    case 'skill_item_update':
      return {
        ...state,
        data: {
          ...d,
          skill_groups: d.skill_groups.map((g) =>
            g.id === action.groupId ? { ...g, items: updateFlat(g.items, action.id, action.patch) } : g,
          ),
        },
      }
    case 'skill_item_delete':
      return {
        ...state,
        data: {
          ...d,
          skill_groups: d.skill_groups.map((g) =>
            g.id === action.groupId ? { ...g, items: g.items.filter((i) => i.id !== action.id) } : g,
          ),
        },
      }
    case 'skill_item_reorder':
      return {
        ...state,
        data: {
          ...d,
          skill_groups: d.skill_groups.map((g) => (g.id === action.groupId ? { ...g, items: reindex(g.items, action.orderedIds) } : g)),
        },
      }

    case 'custom_section_create':
      return {
        ...state,
        data: {
          ...d,
          custom_sections: [
            ...d.custom_sections,
            { id: action.tempId, resume_id: d.resume.id, entries: [], ...action.fields } as unknown as CustomSection,
          ],
        },
      }
    case 'custom_section_update':
      return { ...state, data: { ...d, custom_sections: updateFlat(d.custom_sections, action.id, action.patch) } }
    case 'custom_section_delete':
      return { ...state, data: { ...d, custom_sections: d.custom_sections.filter((s) => s.id !== action.id) } }
    case 'custom_section_reorder':
      return { ...state, data: { ...d, custom_sections: reindex(d.custom_sections, action.orderedIds) } }

    case 'custom_entry_create':
      return {
        ...state,
        data: {
          ...d,
          custom_sections: d.custom_sections.map((s) =>
            s.id === action.sectionId
              ? { ...s, entries: [...s.entries, { id: action.tempId, custom_section_id: s.id, ...action.fields } as CustomSectionEntry] }
              : s,
          ),
        },
      }
    case 'custom_entry_update':
      return {
        ...state,
        data: {
          ...d,
          custom_sections: d.custom_sections.map((s) =>
            s.id === action.sectionId ? { ...s, entries: updateFlat(s.entries, action.id, action.patch) } : s,
          ),
        },
      }
    case 'custom_entry_delete':
      return {
        ...state,
        data: {
          ...d,
          custom_sections: d.custom_sections.map((s) =>
            s.id === action.sectionId ? { ...s, entries: s.entries.filter((e) => e.id !== action.id) } : s,
          ),
        },
      }
    case 'custom_entry_reorder':
      return {
        ...state,
        data: {
          ...d,
          custom_sections: d.custom_sections.map((s) =>
            s.id === action.sectionId ? { ...s, entries: reindex(s.entries, action.orderedIds) } : s,
          ),
        },
      }

    case 'resolve_ids': {
      const resolvedData = replaceIdEverywhere(d, action.idMap) as FullResume
      const resolvedSynced = replaceIdEverywhere(state.lastSynced, action.idMap) as FullResume
      return { data: resolvedData, lastSynced: resolvedSynced }
    }

    case 'mark_synced_entity':
      return { ...state, lastSynced: { ...state.lastSynced, [action.slice]: d[action.slice] } }

    case 'mark_synced_all':
      return { ...state, lastSynced: d }

    default:
      return state
  }
}
