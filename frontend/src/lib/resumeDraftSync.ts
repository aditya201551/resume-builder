import { apiDelete, apiPatch, apiPost } from '@/lib/http'
import { draftReducer, type DraftAction, type DraftState, type FlatKind } from '@/hooks/resumeDraftReducer'
import type { FullResume } from '@/types/resume'

const FLAT_ENTITIES: { entity: FlatKind; path: (resumeId: string) => string }[] = [
  { entity: 'work_experiences', path: (id) => `/api/resumes/${id}/work-experiences` },
  { entity: 'educations', path: (id) => `/api/resumes/${id}/educations` },
  { entity: 'projects', path: (id) => `/api/resumes/${id}/projects` },
  { entity: 'certifications', path: (id) => `/api/resumes/${id}/certifications` },
  { entity: 'languages', path: (id) => `/api/resumes/${id}/languages` },
  { entity: 'misc_entries', path: (id) => `/api/resumes/${id}/misc-entries` },
]

function omit(obj: object, keys: string[]): Record<string, unknown> {
  const out: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(obj)) {
    if (!keys.includes(k)) out[k] = v
  }
  return out
}

function isTempId(id: string) {
  return id.startsWith('temp-')
}

/**
 * Diffs `state.data` against `state.lastSynced` and replays only what
 * changed against the backend, in an order where parents (skill groups,
 * custom sections) sync — and get their temp IDs resolved to real ones —
 * before their children, which need the real parent ID in their URL.
 *
 * `apply` both updates the local working copy (so later steps in this same
 * flush see the resolved IDs immediately) and dispatches to React state.
 */
export async function flushDraft(resumeId: string, snapshot: DraftState, dispatch: (action: DraftAction) => void) {
  let state = snapshot
  const apply = (action: DraftAction) => {
    state = draftReducer(state, action)
    dispatch(action)
  }

  for (const { entity, path } of FLAT_ENTITIES) {
    await syncFlatCollection(entity, path(resumeId), apply, () => state)
  }

  await syncSkillGroups(resumeId, apply, () => state)
  await syncCustomSections(resumeId, apply, () => state)
  await syncSectionConfigs(resumeId, apply, () => state)
  await syncResumeMeta(resumeId, apply, () => state)
}

async function syncFlatCollection(
  entity: FlatKind,
  path: string,
  apply: (a: DraftAction) => void,
  getState: () => DraftState,
) {
  const cur = getState().data[entity] as { id: string }[]
  const prev = getState().lastSynced[entity] as { id: string }[]
  const curIds = new Set(cur.map((i) => i.id))
  const prevById = new Map(prev.map((i) => [i.id, i]))

  for (const item of prev) {
    if (!curIds.has(item.id)) await apiDelete(`${path}/${item.id}`)
  }

  const idMap: Record<string, string> = {}
  for (const item of cur) {
    if (!prevById.has(item.id)) {
      const created = await apiPost<{ id: string }>(path, omit(item, ['id', 'resume_id']))
      idMap[item.id] = created.id
    }
  }
  if (Object.keys(idMap).length > 0) apply({ type: 'resolve_ids', idMap })

  for (const item of cur) {
    const prevItem = prevById.get(item.id)
    if (prevItem && !isTempId(item.id) && JSON.stringify(item) !== JSON.stringify(prevItem)) {
      await apiPatch(`${path}/${item.id}`, omit(item, ['id', 'resume_id']))
    }
  }

  apply({ type: 'mark_synced_entity', slice: entity })
}

async function syncSkillGroups(resumeId: string, apply: (a: DraftAction) => void, getState: () => DraftState) {
  const basePath = `/api/resumes/${resumeId}/skill-groups`
  const cur = getState().data.skill_groups
  const prev = getState().lastSynced.skill_groups
  const curIds = new Set(cur.map((g) => g.id))
  const prevById = new Map(prev.map((g) => [g.id, g]))

  for (const g of prev) {
    if (!curIds.has(g.id)) await apiDelete(`${basePath}/${g.id}`)
  }

  const idMap: Record<string, string> = {}
  for (const g of cur) {
    if (!prevById.has(g.id)) {
      const created = await apiPost<{ id: string }>(basePath, omit(g, ['id', 'resume_id', 'items']))
      idMap[g.id] = created.id
    }
  }
  if (Object.keys(idMap).length > 0) apply({ type: 'resolve_ids', idMap })

  for (const g of cur) {
    const prevGroup = prevById.get(g.id)
    if (prevGroup && !isTempId(g.id)) {
      const { items: curItems, ...curFields } = g
      const { items: _prevItems, ...prevFields } = prevGroup
      if (JSON.stringify(curFields) !== JSON.stringify(prevFields)) {
        await apiPatch(`${basePath}/${g.id}`, omit(curFields, ['id', 'resume_id']))
      }
      void curItems
    }
  }
  apply({ type: 'mark_synced_entity', slice: 'skill_groups' })

  // Items, per (now-real-id) group.
  for (const g of getState().data.skill_groups) {
    const itemsPath = `${basePath}/${g.id}/items`
    const curItems = g.items
    const prevGroup = getState().lastSynced.skill_groups.find((p) => p.id === g.id)
    const prevItems = prevGroup?.items ?? []
    const curItemIds = new Set(curItems.map((i) => i.id))
    const prevItemById = new Map(prevItems.map((i) => [i.id, i]))

    for (const item of prevItems) {
      if (!curItemIds.has(item.id)) await apiDelete(`${itemsPath}/${item.id}`)
    }

    const itemIdMap: Record<string, string> = {}
    for (const item of curItems) {
      if (!prevItemById.has(item.id)) {
        const created = await apiPost<{ id: string }>(itemsPath, omit(item, ['id', 'skill_group_id']))
        itemIdMap[item.id] = created.id
      }
    }
    if (Object.keys(itemIdMap).length > 0) apply({ type: 'resolve_ids', idMap: itemIdMap })

    for (const item of curItems) {
      const prevItem = prevItemById.get(item.id)
      if (prevItem && !isTempId(item.id) && JSON.stringify(item) !== JSON.stringify(prevItem)) {
        await apiPatch(`${itemsPath}/${item.id}`, omit(item, ['id', 'skill_group_id']))
      }
    }
  }
  apply({ type: 'mark_synced_entity', slice: 'skill_groups' })
}

async function syncCustomSections(resumeId: string, apply: (a: DraftAction) => void, getState: () => DraftState) {
  const basePath = `/api/resumes/${resumeId}/custom-sections`
  const cur = getState().data.custom_sections
  const prev = getState().lastSynced.custom_sections
  const curIds = new Set(cur.map((s) => s.id))
  const prevById = new Map(prev.map((s) => [s.id, s]))

  for (const s of prev) {
    if (!curIds.has(s.id)) await apiDelete(`${basePath}/${s.id}`)
  }

  const idMap: Record<string, string> = {}
  for (const s of cur) {
    if (!prevById.has(s.id)) {
      const created = await apiPost<{ id: string }>(basePath, omit(s, ['id', 'resume_id', 'entries']))
      idMap[s.id] = created.id
    }
  }
  if (Object.keys(idMap).length > 0) apply({ type: 'resolve_ids', idMap })

  for (const s of cur) {
    const prevSection = prevById.get(s.id)
    if (prevSection && !isTempId(s.id)) {
      const { entries: curEntries, ...curFields } = s
      const { entries: _prevEntries, ...prevFields } = prevSection
      if (JSON.stringify(curFields) !== JSON.stringify(prevFields)) {
        await apiPatch(`${basePath}/${s.id}`, omit(curFields, ['id', 'resume_id']))
      }
      void curEntries
    }
  }
  apply({ type: 'mark_synced_entity', slice: 'custom_sections' })

  for (const s of getState().data.custom_sections) {
    const entriesPath = `${basePath}/${s.id}/entries`
    const curEntries = s.entries
    const prevSection = getState().lastSynced.custom_sections.find((p) => p.id === s.id)
    const prevEntries = prevSection?.entries ?? []
    const curEntryIds = new Set(curEntries.map((e) => e.id))
    const prevEntryById = new Map(prevEntries.map((e) => [e.id, e]))

    for (const entry of prevEntries) {
      if (!curEntryIds.has(entry.id)) await apiDelete(`${entriesPath}/${entry.id}`)
    }

    const entryIdMap: Record<string, string> = {}
    for (const entry of curEntries) {
      if (!prevEntryById.has(entry.id)) {
        const created = await apiPost<{ id: string }>(entriesPath, omit(entry, ['id', 'custom_section_id']))
        entryIdMap[entry.id] = created.id
      }
    }
    if (Object.keys(entryIdMap).length > 0) apply({ type: 'resolve_ids', idMap: entryIdMap })

    for (const entry of curEntries) {
      const prevEntry = prevEntryById.get(entry.id)
      if (prevEntry && !isTempId(entry.id) && JSON.stringify(entry) !== JSON.stringify(prevEntry)) {
        await apiPatch(`${entriesPath}/${entry.id}`, omit(entry, ['id', 'custom_section_id']))
      }
    }
  }
  apply({ type: 'mark_synced_entity', slice: 'custom_sections' })
}

async function syncSectionConfigs(resumeId: string, apply: (a: DraftAction) => void, getState: () => DraftState) {
  const cur = getState().data.section_configs
  const prev = getState().lastSynced.section_configs
  const key = (c: { section_type: string; custom_section_id: string | null }) => `${c.section_type}:${c.custom_section_id ?? ''}`
  const prevByKey = new Map(prev.map((c) => [key(c), c]))

  for (const c of cur) {
    const prevConfig = prevByKey.get(key(c))
    if (prevConfig && JSON.stringify(c) !== JSON.stringify(prevConfig)) {
      const base = `/api/resumes/${resumeId}/section-configs/${c.section_type}`
      const url = c.custom_section_id ? `${base}?custom_section_id=${c.custom_section_id}` : base
      await apiPatch(url, omit(c, ['id', 'resume_id', 'section_type', 'custom_section_id']))
    }
  }
  apply({ type: 'mark_synced_entity', slice: 'section_configs' })
}

async function syncResumeMeta(resumeId: string, apply: (a: DraftAction) => void, getState: () => DraftState) {
  const cur = getState().data.resume
  const prev = getState().lastSynced.resume
  const metaKeys = ['label', 'template_id', 'full_name', 'headline', 'email', 'phone', 'location', 'photo_url', 'summary', 'links']
  const curMeta = Object.fromEntries(metaKeys.map((k) => [k, (cur as unknown as Record<string, unknown>)[k]]))
  const prevMeta = Object.fromEntries(metaKeys.map((k) => [k, (prev as unknown as Record<string, unknown>)[k]]))
  if (JSON.stringify(curMeta) !== JSON.stringify(prevMeta)) {
    await apiPatch(`/api/resumes/${resumeId}`, curMeta)
  }
  apply({ type: 'mark_synced_entity', slice: 'resume' })
}

export function isDraftDirty(state: FullResume, lastSynced: FullResume): boolean {
  return JSON.stringify(state) !== JSON.stringify(lastSynced)
}
