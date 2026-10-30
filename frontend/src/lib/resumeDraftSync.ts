import { apiDelete, apiPatch, apiPost, apiPut, HttpError } from '@/lib/http'
import { draftReducer, type DraftAction, type DraftState, type FlatKind } from '@/hooks/resumeDraftReducer'
import type { FullResume } from '@/types/resume'

const FLAT_ENTITIES: { entity: FlatKind; path: string }[] = [
  { entity: 'work_experiences', path: 'work-experiences' },
  { entity: 'educations', path: 'educations' },
  { entity: 'projects', path: 'projects' },
  { entity: 'certifications', path: 'certifications' },
  { entity: 'languages', path: 'languages' },
  { entity: 'misc_entries', path: 'misc-entries' },
]

function omit(obj: object, keys: string[]): Record<string, unknown> {
  return Object.fromEntries(Object.entries(obj).filter(([key]) => !keys.includes(key)))
}

async function deleteIfExists(url: string) {
  try {
    await apiDelete(url)
  } catch (err) {
    if (err instanceof HttpError && err.status === 404) return
    throw err
  }
}

// Acknowledge every successful request, including progress before a later request fails.
// Parent collections exclude children here; children sync after parent IDs resolve.
async function syncCollection<T extends { id: string }>(
  current: T[],
  previous: T[],
  path: string,
  parentKey: string,
  apply: (action: DraftAction) => void,
  acknowledge: (items: T[]) => void,
  childrenKey?: keyof T,
) {
  let saved = [...previous]
  const resolvedIds = new Map<string, string>()
  const currentIds = new Set(current.map((item) => item.id))
  const omitted = ['id', parentKey, ...(childrenKey ? [String(childrenKey)] : [])]
  const body = (item: T) => omit(item, omitted)
  const savedItem = (item: T, previousItem?: T): T => childrenKey
    ? { ...item, [childrenKey]: previousItem?.[childrenKey] ?? [] }
    : item

  for (const item of previous) {
    if (currentIds.has(item.id)) continue
    await deleteIfExists(`${path}/${item.id}`)
    saved = saved.filter((entry) => entry.id !== item.id)
    acknowledge(saved)
  }

  for (const item of current) {
    const previousItem = saved.find((entry) => entry.id === item.id)
    if (!previousItem) {
      const created = await apiPost<T>(path, { ...body(item), client_id: item.id })
      resolvedIds.set(item.id, created.id)
      apply({ type: 'resolve_ids', idMap: { [item.id]: created.id } })
      // An idempotent retry can return an earlier create. Use its actual fields
      // so edits since that create remain dirty and are PATCHed on the next pass.
      const acknowledged = Object.fromEntries(Object.entries(item).map(([key, value]) => [
        key, Object.hasOwn(created, key) ? (created as Record<string, unknown>)[key] : value,
      ])) as T
      saved = [...saved, savedItem({ ...acknowledged, id: created.id })]
      acknowledge(saved)
    } else if (JSON.stringify(body(item)) !== JSON.stringify(body(previousItem))) {
      await apiPatch(`${path}/${item.id}`, body(item))
      saved = saved.map((entry) => entry.id === item.id ? savedItem(item, entry) : entry)
      acknowledge(saved)
    }
  }
  const ordered = current.map((item) => saved.find((entry) => entry.id === (resolvedIds.get(item.id) ?? item.id))!)
  if (JSON.stringify(ordered) !== JSON.stringify(saved)) acknowledge(ordered)
}

export async function flushDraft(resumeId: string, snapshot: DraftState, dispatch: (action: DraftAction) => void) {
  let state = snapshot
  const apply = (action: DraftAction) => {
    state = draftReducer(state, action)
    dispatch(action)
  }
  const acknowledge = <K extends keyof FullResume>(slice: K, value: FullResume[K]) => {
    apply({ type: 'mark_synced_entity', slice, snapshot: { ...state.lastSynced, [slice]: value } })
  }
  let hadError = false
  async function step(fn: () => Promise<void>) {
    try {
      await fn()
    } catch (err) {
      hadError = true
      console.error('draft sync step failed:', err)
    }
  }
  const base = `/api/resumes/${resumeId}`

  for (const { entity, path } of FLAT_ENTITIES) {
    await step(() => syncCollection<{ id: string }>(
      state.data[entity], state.lastSynced[entity], `${base}/${path}`, 'resume_id', apply,
      (items) => acknowledge(entity, items as FullResume[typeof entity]),
    ))
  }

  await step(async () => {
    await syncCollection(
      state.data.skill_groups, state.lastSynced.skill_groups, `${base}/skill-groups`, 'resume_id', apply,
      (groups) => acknowledge('skill_groups', groups), 'items',
    )
    for (const group of state.data.skill_groups) {
      await step(() => syncCollection(
        group.items, state.lastSynced.skill_groups.find((saved) => saved.id === group.id)?.items ?? [],
        `${base}/skill-groups/${group.id}/items`, 'skill_group_id', apply,
        (items) => acknowledge('skill_groups', state.lastSynced.skill_groups.map((saved) =>
          saved.id === group.id ? { ...saved, items } : saved)),
      ))
    }
  })

  await step(async () => {
    await syncCollection(
      state.data.custom_sections, state.lastSynced.custom_sections, `${base}/custom-sections`, 'resume_id', apply,
      (sections) => acknowledge('custom_sections', sections), 'entries',
    )
    for (const section of state.data.custom_sections) {
      await step(() => syncCollection(
        section.entries, state.lastSynced.custom_sections.find((saved) => saved.id === section.id)?.entries ?? [],
        `${base}/custom-sections/${section.id}/entries`, 'custom_section_id', apply,
        (entries) => acknowledge('custom_sections', state.lastSynced.custom_sections.map((saved) =>
          saved.id === section.id ? { ...saved, entries } : saved)),
      ))
    }
  })

  await step(async () => {
    const current = state.data.resume
    const previous = state.lastSynced.resume
    const omitted = ['id', 'user_id', 'created_at', 'updated_at', 'last_exported_at']
    if (JSON.stringify(omit(current, omitted)) !== JSON.stringify(omit(previous, omitted))) {
      await apiPatch(base, omit(current, omitted))
      acknowledge('resume', current)
    }
  })
  await step(async () => {
    const current = state.data.design
    if (JSON.stringify(current) !== JSON.stringify(state.lastSynced.design)) {
      await apiPut(`${base}/design`, current)
      acknowledge('design', current)
    }
  })

  if (hadError) throw new Error('one or more draft sync steps failed')
}

export function isDraftDirty(state: FullResume, lastSynced: FullResume): boolean {
  return JSON.stringify(state) !== JSON.stringify(lastSynced)
}
